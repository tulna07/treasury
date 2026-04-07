// Package limits xử lý nghiệp vụ Quản lý Hạn mức.
package limits

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kienlongbank/treasury-api/pkg/httputil"
)

// Handler xử lý các request liên quan đến hạn mức.
type Handler struct {
	pool *pgxpool.Pool
}

// NewHandler tạo handler mới cho module Hạn mức.
func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

// List trả về danh sách hạn mức.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data":  []interface{}{},
		"total": 0,
	})
}

// Create cấp hạn mức mới.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSON(w, http.StatusCreated, map[string]string{
		"message": "Cấp hạn mức thành công",
	})
}

// Get trả về chi tiết một hạn mức.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	httputil.WriteJSON(w, http.StatusOK, map[string]string{
		"id":      id,
		"message": "Chi tiết hạn mức",
	})
}

// Update cập nhật hạn mức.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	httputil.WriteJSON(w, http.StatusOK, map[string]string{
		"id":      id,
		"message": "Cập nhật hạn mức thành công",
	})
}
