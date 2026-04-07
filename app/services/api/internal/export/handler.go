// Package export provides the export history HTTP handler.
package export

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/export"
	"github.com/kienlongbank/treasury-api/pkg/httputil"
)

// Handler handles export history HTTP requests.
type Handler struct {
	engine *export.Engine
	logger *zap.Logger
}

// NewHandler creates a new export history handler.
func NewHandler(engine *export.Engine, logger *zap.Logger) *Handler {
	return &Handler{engine: engine, logger: logger}
}

// List handles GET /exports — list export audit logs.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	logs, total, err := h.engine.ListExports(r.Context(), nil, limit, offset)
	if err != nil {
		httputil.Error(w, r, apperror.Wrap(err, apperror.ErrInternal, "failed to list exports"))
		return
	}

	httputil.Success(w, r, map[string]interface{}{
		"data":  logs,
		"total": total,
	})
}

// Get handles GET /exports/{code} — get single export detail.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "export code is required"))
		return
	}

	_, auditLog, err := h.engine.GetExportFile(r.Context(), code)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, auditLog)
}

// Download handles POST /exports/{code}/download — re-download from MinIO.
func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "export code is required"))
		return
	}

	stream, stat, auditLog, err := h.engine.StreamExportFile(r.Context(), code)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}
	defer stream.Close()

	// Content-Type from MinIO metadata (preserves original format)
	contentType := stat.ContentType
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.xlsx"`, auditLog.ExportCode))
	// Prevent gzip middleware from re-encoding binary file
	w.Header().Set("Content-Encoding", "identity")
	w.WriteHeader(http.StatusOK)

	// Stream byte-for-byte from MinIO → client (no buffering)
	io.Copy(w, stream)
}
