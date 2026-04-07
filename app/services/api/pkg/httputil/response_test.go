package httputil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/dto"
)

func newRequest() *http.Request {
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("X-Request-ID", "test-req-id")
	return r
}

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	r := newRequest()
	Success(w, r, map[string]string{"id": "123"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp dto.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.Error != nil {
		t.Error("expected no error")
	}
	if resp.Meta == nil {
		t.Fatal("expected meta to be present")
	}
	if resp.Meta.RequestID != "test-req-id" {
		t.Errorf("expected request_id=test-req-id, got %s", resp.Meta.RequestID)
	}
	if resp.Meta.Timestamp == "" {
		t.Error("expected timestamp to be set")
	}
}

func TestCreated(t *testing.T) {
	w := httptest.NewRecorder()
	r := newRequest()
	Created(w, r, map[string]string{"id": "456"})

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}

	var resp dto.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Meta == nil {
		t.Fatal("expected meta to be present")
	}
	if resp.Meta.RequestID != "test-req-id" {
		t.Errorf("expected request_id=test-req-id, got %s", resp.Meta.RequestID)
	}
}

func TestNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	NoContent(w)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestError_AppError(t *testing.T) {
	w := httptest.NewRecorder()
	r := newRequest()
	appErr := apperror.New(apperror.ErrNotFound, "deal not found")
	Error(w, r, appErr)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	var resp dto.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Success {
		t.Error("expected success=false")
	}
	if resp.Error == nil {
		t.Fatal("expected error object")
	}
	if resp.Error.Code != "NOT_FOUND" {
		t.Errorf("expected NOT_FOUND, got %s", resp.Error.Code)
	}
	if resp.Meta == nil {
		t.Fatal("expected meta to be present")
	}
	if resp.Meta.RequestID != "test-req-id" {
		t.Errorf("expected request_id=test-req-id, got %s", resp.Meta.RequestID)
	}
}

func TestError_GenericError(t *testing.T) {
	w := httptest.NewRecorder()
	r := newRequest()
	Error(w, r, fmt.Errorf("something broke"))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}

	var resp dto.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Meta == nil {
		t.Fatal("expected meta to be present")
	}
}

func TestValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	r := newRequest()
	ValidationError(w, r, []string{"field1 is required", "field2 must be positive"})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp dto.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Error.Detail == "" {
		t.Error("expected validation detail")
	}
	if resp.Meta == nil {
		t.Fatal("expected meta to be present")
	}
}

func TestPaginated(t *testing.T) {
	w := httptest.NewRecorder()
	r := newRequest()
	data := []string{"item1", "item2"}
	Paginated(w, r, data, 25, 1, 10)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp dto.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.Meta == nil {
		t.Fatal("expected meta to be present")
	}
	if resp.Meta.Pagination == nil {
		t.Fatal("expected pagination meta")
	}
	if resp.Meta.Pagination.Total != 25 {
		t.Errorf("expected total=25, got %d", resp.Meta.Pagination.Total)
	}
	if resp.Meta.Pagination.TotalPages != 3 {
		t.Errorf("expected total_pages=3, got %d", resp.Meta.Pagination.TotalPages)
	}
	if resp.Meta.Pagination.Page != 1 {
		t.Errorf("expected page=1, got %d", resp.Meta.Pagination.Page)
	}
	if resp.Meta.Pagination.PageSize != 10 {
		t.Errorf("expected page_size=10, got %d", resp.Meta.Pagination.PageSize)
	}
	if !resp.Meta.Pagination.HasMore {
		t.Error("expected has_more=true")
	}
}

func TestPaginatedCursor(t *testing.T) {
	w := httptest.NewRecorder()
	r := newRequest()
	data := []string{"item1", "item2"}
	PaginatedCursor(w, r, data, 100, "next-abc", "prev-xyz", true)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp dto.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.Meta == nil {
		t.Fatal("expected meta to be present")
	}
	if resp.Meta.Pagination == nil {
		t.Fatal("expected pagination meta")
	}
	if resp.Meta.Pagination.Total != 100 {
		t.Errorf("expected total=100, got %d", resp.Meta.Pagination.Total)
	}
	if resp.Meta.Pagination.NextCursor != "next-abc" {
		t.Errorf("expected next_cursor=next-abc, got %s", resp.Meta.Pagination.NextCursor)
	}
	if resp.Meta.Pagination.PrevCursor != "prev-xyz" {
		t.Errorf("expected prev_cursor=prev-xyz, got %s", resp.Meta.Pagination.PrevCursor)
	}
	if !resp.Meta.Pagination.HasMore {
		t.Error("expected has_more=true")
	}
}

func TestErrorWithStatus(t *testing.T) {
	w := httptest.NewRecorder()
	r := newRequest()
	ErrorWithStatus(w, r, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}

	var resp dto.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Meta == nil {
		t.Fatal("expected meta to be present")
	}
}

func TestBuildMeta_GeneratesRequestID(t *testing.T) {
	r := httptest.NewRequest("GET", "/test", nil)
	// No X-Request-ID header and no context value
	meta := buildMeta(r)
	if meta.RequestID == "" {
		t.Error("expected generated request_id")
	}
	if meta.Timestamp == "" {
		t.Error("expected timestamp")
	}
}

func TestBuildMeta_NilRequest(t *testing.T) {
	meta := buildMeta(nil)
	if meta.RequestID == "" {
		t.Error("expected generated request_id even with nil request")
	}
	if meta.Timestamp == "" {
		t.Error("expected timestamp")
	}
}
