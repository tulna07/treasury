package dto

// ApprovalRequest is the payload for approving or rejecting a deal.
// @Description Request body for deal approval or rejection.
type ApprovalRequest struct {
	Action  string  `json:"action" validate:"required,oneof=APPROVE REJECT"`
	Comment *string `json:"comment" validate:"omitempty,max=2000"`
	Version int     `json:"version" validate:"required,min=1"`
}

// APIResponse is the standard envelope for ALL API responses.
// @Description Standard API response envelope.
type APIResponse struct {
	Success bool          `json:"success"`
	Data    interface{}   `json:"data,omitempty"`
	Error   *APIError     `json:"error,omitempty"`
	Meta    *ResponseMeta `json:"meta,omitempty"`
}

// ResponseMeta contains request metadata.
// @Description Metadata about the API response including request tracing and pagination.
type ResponseMeta struct {
	RequestID  string          `json:"request_id,omitempty"`
	Timestamp  string          `json:"timestamp"`
	Pagination *PaginationMeta `json:"pagination,omitempty"`
}

// PaginationMeta provides pagination details for list endpoints.
// @Description Pagination metadata for paginated responses.
type PaginationMeta struct {
	// Offset mode
	Page       int `json:"page,omitempty"`
	PageSize   int `json:"page_size,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`

	// Shared
	Total   int64 `json:"total"`
	HasMore bool  `json:"has_more"`

	// Cursor mode
	NextCursor string `json:"next_cursor,omitempty"`
	PrevCursor string `json:"prev_cursor,omitempty"`
}

// APIError is the standard error body in API responses.
// @Description Standard error body with optional field-level validation errors.
type APIError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Detail  string            `json:"detail,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"`
}
