// Package gtcg xử lý nghiệp vụ Giấy tờ có giá.
package gtcg

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kienlongbank/treasury-api/pkg/httputil"
)

// Handler xử lý các request liên quan đến GTCG.
type Handler struct {
	pool *pgxpool.Pool
}

// NewHandler tạo handler mới cho module GTCG.
func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

// List trả về danh sách giấy tờ có giá.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data":  []interface{}{},
		"total": 0,
	})
}

// Create thêm giấy tờ có giá mới.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSON(w, http.StatusCreated, map[string]string{
		"message": "Thêm GTCG thành công",
	})
}

// Get trả về chi tiết một GTCG.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	httputil.WriteJSON(w, http.StatusOK, map[string]string{
		"id":      id,
		"message": "Chi tiết GTCG",
	})
}

// Update cập nhật GTCG.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	httputil.WriteJSON(w, http.StatusOK, map[string]string{
		"id":      id,
		"message": "Cập nhật GTCG thành công",
	})
}
