// Package dto defines data transfer objects for the Treasury API.
package dto

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PaginationRequest holds pagination parameters from the client.
type PaginationRequest struct {
	// OFFSET-based (backward compat)
	Page     int    `json:"page" validate:"omitempty,min=1"`
	PageSize int    `json:"page_size" validate:"omitempty,min=1,max=100"`
	SortBy   string `json:"sort_by,omitempty" validate:"omitempty,max=50"`
	SortDir  string `json:"sort_dir,omitempty" validate:"omitempty,oneof=asc desc ASC DESC"`

	// Keyset/Cursor-based (preferred for large datasets)
	Cursor    string `json:"cursor,omitempty"`    // opaque cursor (base64 encoded)
	Limit     int    `json:"limit" validate:"omitempty,min=1,max=100"`
	Direction string `json:"direction,omitempty"` // "next" or "prev"
}

// DefaultPagination returns a PaginationRequest with sensible defaults.
func DefaultPagination() PaginationRequest {
	return PaginationRequest{
		Page:     1,
		PageSize: 20,
		SortBy:   "created_at",
		SortDir:  "desc",
	}
}

// Offset calculates the SQL offset from page and page_size.
func (p PaginationRequest) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// IsCursorMode returns true if cursor-based pagination is being used.
func (p PaginationRequest) IsCursorMode() bool {
	return p.Cursor != "" || p.Limit > 0
}

// EffectiveLimit returns the limit for cursor mode, falling back to PageSize.
func (p PaginationRequest) EffectiveLimit() int {
	if p.Limit > 0 {
		return p.Limit
	}
	if p.PageSize > 0 {
		return p.PageSize
	}
	return 20
}

// PaginationResponse wraps paginated data.
type PaginationResponse[T any] struct {
	Data       []T    `json:"data"`
	Total      int64  `json:"total"`
	Page       int    `json:"page,omitempty"`
	PageSize   int    `json:"page_size,omitempty"`
	TotalPages int    `json:"total_pages,omitempty"`

	// Keyset
	NextCursor string `json:"next_cursor,omitempty"`
	PrevCursor string `json:"prev_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// NewPaginationResponse creates a typed pagination response (offset mode).
func NewPaginationResponse[T any](data []T, total int64, page, pageSize int) PaginationResponse[T] {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	if data == nil {
		data = []T{}
	}
	return PaginationResponse[T]{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		HasMore:    page < totalPages,
	}
}

// NewCursorPaginationResponse creates a typed pagination response (cursor mode).
func NewCursorPaginationResponse[T any](data []T, total int64, nextCursor, prevCursor string, hasMore bool) PaginationResponse[T] {
	if data == nil {
		data = []T{}
	}
	return PaginationResponse[T]{
		Data:       data,
		Total:      total,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
		HasMore:    hasMore,
	}
}

// CursorData is base64(json({id, created_at}))
type CursorData struct {
	ID        string `json:"id"`
	CreatedAt string `json:"ts"`
}

// EncodeCursor encodes a cursor from id and createdAt.
func EncodeCursor(id uuid.UUID, createdAt time.Time) string {
	data := CursorData{
		ID:        id.String(),
		CreatedAt: createdAt.Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(data)
	return base64.URLEncoding.EncodeToString(b)
}

// DecodeCursor decodes a cursor string back to CursorData.
func DecodeCursor(cursor string) (*CursorData, error) {
	b, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor encoding: %w", err)
	}
	var data CursorData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("invalid cursor data: %w", err)
	}
	if data.ID == "" || data.CreatedAt == "" {
		return nil, fmt.Errorf("cursor missing required fields")
	}
	return &data, nil
}
