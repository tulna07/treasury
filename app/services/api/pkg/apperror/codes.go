// Package apperror defines application-level error types for the Treasury API.
package apperror

import "net/http"

// Error code constants for banking operations.
const (
	ErrNotFound              ErrorCode = "NOT_FOUND"
	ErrUnauthorized          ErrorCode = "UNAUTHORIZED"
	ErrForbidden             ErrorCode = "FORBIDDEN"
	ErrValidation            ErrorCode = "VALIDATION_ERROR"
	ErrConflict              ErrorCode = "CONFLICT"
	ErrInternal              ErrorCode = "INTERNAL_ERROR"
	ErrDealLocked            ErrorCode = "DEAL_LOCKED"
	ErrInsufficientInventory ErrorCode = "INSUFFICIENT_INVENTORY"
	ErrLimitExceeded         ErrorCode = "LIMIT_EXCEEDED"
	ErrInvalidTransition     ErrorCode = "INVALID_TRANSITION"
	ErrSelfApproval          ErrorCode = "SELF_APPROVAL"
)

// httpStatusMap maps error codes to HTTP status codes.
var httpStatusMap = map[ErrorCode]int{
	ErrNotFound:              http.StatusNotFound,
	ErrUnauthorized:          http.StatusUnauthorized,
	ErrForbidden:             http.StatusForbidden,
	ErrValidation:            http.StatusBadRequest,
	ErrConflict:              http.StatusConflict,
	ErrInternal:              http.StatusInternalServerError,
	ErrDealLocked:            http.StatusConflict,
	ErrInsufficientInventory: http.StatusUnprocessableEntity,
	ErrLimitExceeded:         http.StatusUnprocessableEntity,
	ErrInvalidTransition:     http.StatusUnprocessableEntity,
	ErrSelfApproval:          http.StatusForbidden,
}

// HTTPStatusForCode returns the HTTP status code for the given error code.
func HTTPStatusForCode(code ErrorCode) int {
	if status, ok := httpStatusMap[code]; ok {
		return status
	}
	return http.StatusInternalServerError
}
