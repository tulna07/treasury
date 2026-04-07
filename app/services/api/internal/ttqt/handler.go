// Package ttqt xử lý nghiệp vụ Thanh toán Quốc tế.
package ttqt

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kienlongbank/treasury-api/pkg/httputil"
)

// Handler xử lý các request liên quan đến giao dịch TTQT.
type Handler struct {
	pool *pgxpool.Pool
}

// NewHandler tạo handler mới cho module TTQT.
func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

// List trả về danh sách giao dịch TTQT.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data":  []interface{}{},
		"total": 0,
	})
}

// Create tạo giao dịch TTQT mới.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSON(w, http.StatusCreated, map[string]string{
		"message": "Tạo giao dịch TTQT thành công",
	})
}

// Get trả về chi tiết một giao dịch TTQT.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	httputil.WriteJSON(w, http.StatusOK, map[string]string{
		"id":      id,
		"message": "Chi tiết giao dịch TTQT",
	})
}

// Update cập nhật giao dịch TTQT.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	httputil.WriteJSON(w, http.StatusOK, map[string]string{
		"id":      id,
		"message": "Cập nhật giao dịch TTQT thành công",
	})
}
