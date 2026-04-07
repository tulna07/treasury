package attachment

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/ctxutil"
	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/internal/repository"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/httputil"
)

const (
	maxFileSize      = 10 << 20 // 10 MB per file
	maxTotalSize     = 50 << 20 // 50 MB total upload
	maxFilesPerReq   = 5
	maxFilesPerDeal  = 20
	attachmentBucket = "treasury-attachments"
)

var allowedContentTypes = map[string]bool{
	"application/pdf":          true,
	"image/jpeg":               true,
	"image/png":                true,
	"image/gif":                true,
	"text/plain":               true,
	"application/msword":       true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":    true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
}

var allowedModules = map[string]bool{
	"FX": true, "MM": true, "BOND": true, "GTCG": true,
}

// sanitizeRe strips path separators and special chars from filenames.
var sanitizeRe = regexp.MustCompile(`[^a-zA-Z0-9._\-\p{L}\p{N} ]`)

// Handler handles attachment HTTP requests.
type Handler struct {
	repo   repository.AttachmentRepository
	minio  *minio.Client
	logger *zap.Logger
}

// NewHandler creates a new attachment handler.
func NewHandler(repo repository.AttachmentRepository, minioClient *minio.Client, logger *zap.Logger) *Handler {
	return &Handler{repo: repo, minio: minioClient, logger: logger}
}

// Upload handles POST /api/v1/attachments/upload — multipart upload (up to 5 files).
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	userID := ctxutil.GetUserUUID(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, apperror.New(apperror.ErrUnauthorized, "user not authenticated"))
		return
	}

	if err := r.ParseMultipartForm(maxTotalSize); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "failed to parse multipart form"))
		return
	}

	dealModule := strings.TrimSpace(r.FormValue("deal_module"))
	dealIDStr := strings.TrimSpace(r.FormValue("deal_id"))

	if !allowedModules[dealModule] {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid deal_module"))
		return
	}
	dealID, err := uuid.Parse(dealIDStr)
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid deal_id"))
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "no files provided"))
		return
	}
	if len(files) > maxFilesPerReq {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, fmt.Sprintf("max %d files per upload", maxFilesPerReq)))
		return
	}

	// Check deal attachment count limit
	count, err := h.repo.CountByDeal(r.Context(), dealModule, dealID)
	if err != nil {
		httputil.Error(w, r, apperror.Wrap(err, apperror.ErrInternal, "failed to count attachments"))
		return
	}
	if count+len(files) > maxFilesPerDeal {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, fmt.Sprintf("max %d files per deal (current: %d)", maxFilesPerDeal, count)))
		return
	}

	// Ensure bucket exists
	if err := h.ensureBucket(r.Context()); err != nil {
		httputil.Error(w, r, apperror.Wrap(err, apperror.ErrInternal, "storage unavailable"))
		return
	}

	var results []dto.AttachmentResponse
	for _, fh := range files {
		if fh.Size > maxFileSize {
			httputil.Error(w, r, apperror.New(apperror.ErrValidation, fmt.Sprintf("file %q exceeds 10MB limit", fh.Filename)))
			return
		}

		f, err := fh.Open()
		if err != nil {
			httputil.Error(w, r, apperror.Wrap(err, apperror.ErrInternal, "failed to open uploaded file"))
			return
		}
		defer f.Close()

		// Detect content type from first 512 bytes
		buf := make([]byte, 512)
		n, err := f.Read(buf)
		if err != nil && err != io.EOF {
			httputil.Error(w, r, apperror.Wrap(err, apperror.ErrInternal, "failed to read file"))
			return
		}
		contentType := http.DetectContentType(buf[:n])
		// Strip parameters (e.g. "text/plain; charset=utf-8" → "text/plain")
		if idx := strings.Index(contentType, ";"); idx != -1 {
			contentType = strings.TrimSpace(contentType[:idx])
		}

		// For office documents, DetectContentType returns application/zip — trust extension
		if contentType == "application/zip" || contentType == "application/octet-stream" {
			ext := strings.ToLower(filepath.Ext(fh.Filename))
			switch ext {
			case ".xlsx":
				contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
			case ".docx":
				contentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
			case ".doc":
				contentType = "application/msword"
			case ".xls":
				contentType = "application/vnd.ms-excel"
			case ".pdf":
				contentType = "application/pdf"
			case ".txt":
				contentType = "text/plain"
			}
		}

		if !allowedContentTypes[contentType] {
			httputil.Error(w, r, apperror.New(apperror.ErrValidation, fmt.Sprintf("file type %q not allowed for %q", contentType, fh.Filename)))
			return
		}

		// Seek back to start after reading header
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			httputil.Error(w, r, apperror.Wrap(err, apperror.ErrInternal, "failed to seek file"))
			return
		}

		// Sanitize filename and build MinIO key
		safeName := sanitizeFilename(fh.Filename)
		fileUUID := uuid.New()
		minioKey := fmt.Sprintf("%s/%s/%s_%s", dealModule, dealID.String(), fileUUID.String(), safeName)

		// Upload to MinIO
		_, err = h.minio.PutObject(r.Context(), attachmentBucket, minioKey, f, fh.Size, minio.PutObjectOptions{
			ContentType: contentType,
		})
		if err != nil {
			httputil.Error(w, r, apperror.Wrap(err, apperror.ErrInternal, "failed to upload file to storage"))
			return
		}

		// Insert DB record
		att := &model.DealAttachment{
			ID:          fileUUID,
			DealModule:  dealModule,
			DealID:      dealID,
			FileName:    fh.Filename,
			FileSize:    fh.Size,
			ContentType: contentType,
			MinioBucket: attachmentBucket,
			MinioKey:    minioKey,
			UploadedBy:  userID,
		}
		if err := h.repo.Create(r.Context(), att); err != nil {
			httputil.Error(w, r, apperror.Wrap(err, apperror.ErrInternal, "failed to save attachment record"))
			return
		}

		results = append(results, toResponse(att))
	}

	httputil.Created(w, r, results)
}

// Download handles GET /api/v1/attachments/{id}/download — stream from MinIO.
func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid attachment id"))
		return
	}

	att, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	obj, err := h.minio.GetObject(r.Context(), att.MinioBucket, att.MinioKey, minio.GetObjectOptions{})
	if err != nil {
		httputil.Error(w, r, apperror.Wrap(err, apperror.ErrInternal, "failed to get file from storage"))
		return
	}
	defer obj.Close()

	stat, err := obj.Stat()
	if err != nil {
		httputil.Error(w, r, apperror.Wrap(err, apperror.ErrInternal, "failed to stat file"))
		return
	}

	ct := stat.ContentType
	if ct == "" || ct == "application/octet-stream" {
		ct = att.ContentType
	}

	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, att.FileName))
	w.Header().Set("Content-Encoding", "identity")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, obj)
}

// ListByDeal handles GET /api/v1/attachments/deal/{module}/{dealId}.
func (h *Handler) ListByDeal(w http.ResponseWriter, r *http.Request) {
	module := chi.URLParam(r, "module")
	dealID, err := uuid.Parse(chi.URLParam(r, "dealId"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid deal_id"))
		return
	}

	atts, err := h.repo.ListByDeal(r.Context(), module, dealID)
	if err != nil {
		httputil.Error(w, r, apperror.Wrap(err, apperror.ErrInternal, "failed to list attachments"))
		return
	}

	var results []dto.AttachmentResponse
	for _, a := range atts {
		results = append(results, toResponse(&a))
	}
	if results == nil {
		results = []dto.AttachmentResponse{}
	}
	httputil.Success(w, r, results)
}

// Delete handles DELETE /api/v1/attachments/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := ctxutil.GetUserUUID(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, apperror.New(apperror.ErrUnauthorized, "user not authenticated"))
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid attachment id"))
		return
	}

	att, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	// Only owner or admin can delete
	roles := ctxutil.GetRoles(r.Context())
	isAdmin := false
	for _, role := range roles {
		if role == "ADMIN" {
			isAdmin = true
			break
		}
	}
	if att.UploadedBy != userID && !isAdmin {
		httputil.Error(w, r, apperror.New(apperror.ErrForbidden, "only owner or admin can delete attachments"))
		return
	}

	// Remove from MinIO
	_ = h.minio.RemoveObject(r.Context(), att.MinioBucket, att.MinioKey, minio.RemoveObjectOptions{})

	// Remove from DB
	if err := h.repo.Delete(r.Context(), id); err != nil {
		httputil.Error(w, r, apperror.Wrap(err, apperror.ErrInternal, "failed to delete attachment"))
		return
	}

	httputil.NoContent(w)
}

func (h *Handler) ensureBucket(ctx context.Context) error {
	exists, err := h.minio.BucketExists(ctx, attachmentBucket)
	if err != nil {
		return err
	}
	if !exists {
		return h.minio.MakeBucket(ctx, attachmentBucket, minio.MakeBucketOptions{})
	}
	return nil
}

func toResponse(a *model.DealAttachment) dto.AttachmentResponse {
	return dto.AttachmentResponse{
		ID:          a.ID,
		DealModule:  a.DealModule,
		DealID:      a.DealID,
		FileName:    a.FileName,
		FileSize:    a.FileSize,
		ContentType: a.ContentType,
		UploadedBy:  a.UploadedBy,
		CreatedAt:   a.CreatedAt,
		DownloadURL: fmt.Sprintf("/api/v1/attachments/%s/download", a.ID.String()),
	}
}

func sanitizeFilename(name string) string {
	// Take only the base name (strip directory path)
	name = filepath.Base(name)
	// Replace special chars
	name = sanitizeRe.ReplaceAllString(name, "_")
	// Limit length to 200 chars
	if len(name) > 200 {
		ext := filepath.Ext(name)
		name = name[:200-len(ext)] + ext
	}
	return name
}
