// Package httputil provides HTTP request/response helpers for the Treasury API.
package httputil

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kienlongbank/treasury-api/internal/ctxutil"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/dto"
)

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// buildMeta creates a ResponseMeta with request_id and timestamp.
func buildMeta(r *http.Request) *dto.ResponseMeta {
	reqID := ""
	if r != nil {
		reqID = ctxutil.GetRequestID(r.Context())
		if reqID == "" {
			reqID = r.Header.Get("X-Request-ID")
		}
	}
	if reqID == "" {
		reqID = uuid.New().String()
	}
	return &dto.ResponseMeta{
		RequestID: reqID,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// Success writes a successful API response with metadata.
func Success(w http.ResponseWriter, r *http.Request, data interface{}) {
	WriteJSON(w, http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    data,
		Meta:    buildMeta(r),
	})
}

// Created writes a 201 Created response with metadata.
func Created(w http.ResponseWriter, r *http.Request, data interface{}) {
	WriteJSON(w, http.StatusCreated, dto.APIResponse{
		Success: true,
		Data:    data,
		Meta:    buildMeta(r),
	})
}

// NoContent writes a 204 No Content response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Error writes an error API response using an AppError, with metadata.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	appErr, ok := apperror.AsAppError(err)
	if !ok {
		appErr = apperror.New(apperror.ErrInternal, err.Error())
	}

	WriteJSON(w, appErr.HTTPStatus, dto.APIResponse{
		Success: false,
		Error: &dto.APIError{
			Code:    string(appErr.Code),
			Message: appErr.Message,
			Detail:  appErr.Detail,
		},
		Meta: buildMeta(r),
	})
}

// ErrorWithStatus writes an error with a specific HTTP status, with metadata.
func ErrorWithStatus(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	WriteJSON(w, status, dto.APIResponse{
		Success: false,
		Error: &dto.APIError{
			Code:    code,
			Message: message,
		},
		Meta: buildMeta(r),
	})
}

// ValidationError writes a 400 validation error response with metadata.
func ValidationError(w http.ResponseWriter, r *http.Request, errors []string) {
	WriteJSON(w, http.StatusBadRequest, dto.APIResponse{
		Success: false,
		Error: &dto.APIError{
			Code:    string(apperror.ErrValidation),
			Message: "Validation failed",
			Detail:  joinErrors(errors),
		},
		Meta: buildMeta(r),
	})
}

// Paginated writes a paginated response with metadata including pagination info.
func Paginated(w http.ResponseWriter, r *http.Request, data interface{}, total int64, page, pageSize int) {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	meta := buildMeta(r)
	meta.Pagination = &dto.PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		HasMore:    page < totalPages,
	}
	WriteJSON(w, http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// PaginatedCursor writes a cursor-paginated response with metadata.
func PaginatedCursor(w http.ResponseWriter, r *http.Request, data interface{}, total int64, nextCursor, prevCursor string, hasMore bool) {
	meta := buildMeta(r)
	meta.Pagination = &dto.PaginationMeta{
		Total:      total,
		HasMore:    hasMore,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
	}
	WriteJSON(w, http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

func joinErrors(errs []string) string {
	return strings.Join(errs, "; ")
}
