package httputil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/kienlongbank/treasury-api/pkg/dto"
)

// ParseUUID extracts and parses a UUID from a URL parameter.
func ParseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UUID: %s", s)
	}
	return id, nil
}

// ParsePagination extracts pagination parameters from query string.
func ParsePagination(r *http.Request) dto.PaginationRequest {
	p := dto.DefaultPagination()

	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			p.Page = n
		}
	}
	if v := r.URL.Query().Get("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 100 {
			p.PageSize = n
		}
	}
	if v := r.URL.Query().Get("sort_by"); v != "" {
		p.SortBy = v
	}
	if v := r.URL.Query().Get("sort_dir"); v != "" {
		if v == "asc" || v == "desc" || v == "ASC" || v == "DESC" {
			p.SortDir = v
		}
	}

	return p
}

// ParseBody decodes a JSON request body into the target struct.
func ParseBody(r *http.Request, target interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}
	return nil
}
