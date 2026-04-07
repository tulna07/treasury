package apperror

import (
	"errors"
	"fmt"
)

// ErrorCode represents a machine-readable error code.
type ErrorCode string

// AppError is the standard application error used across the Treasury API.
type AppError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	Detail     string    `json:"detail,omitempty"`
	HTTPStatus int       `json:"-"`
	Err        error     `json:"-"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error for errors.Is/As support.
func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError with the given code and message.
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: HTTPStatusForCode(code),
	}
}

// NewWithDetail creates a new AppError with code, message, and detail.
func NewWithDetail(code ErrorCode, message, detail string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Detail:     detail,
		HTTPStatus: HTTPStatusForCode(code),
	}
}

// Wrap wraps an existing error into an AppError.
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: HTTPStatusForCode(code),
		Err:        err,
	}
}

// Is checks whether the given error is an AppError with the specified code.
func Is(err error, code ErrorCode) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

// AsAppError extracts AppError from an error chain, if present.
func AsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
