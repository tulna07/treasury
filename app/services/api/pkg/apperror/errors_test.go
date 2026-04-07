package apperror

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(ErrNotFound, "deal not found")
	if err.Code != ErrNotFound {
		t.Errorf("expected code %s, got %s", ErrNotFound, err.Code)
	}
	if err.Message != "deal not found" {
		t.Errorf("expected message 'deal not found', got %q", err.Message)
	}
	if err.HTTPStatus != http.StatusNotFound {
		t.Errorf("expected HTTP status %d, got %d", http.StatusNotFound, err.HTTPStatus)
	}
}

func TestNewWithDetail(t *testing.T) {
	err := NewWithDetail(ErrValidation, "invalid request", "field 'amount' must be positive")
	if err.Detail != "field 'amount' must be positive" {
		t.Errorf("expected detail, got %q", err.Detail)
	}
}

func TestWrap(t *testing.T) {
	original := fmt.Errorf("database connection failed")
	wrapped := Wrap(original, ErrInternal, "failed to create deal")

	if wrapped.Err != original {
		t.Error("expected wrapped error to contain original")
	}
	if !errors.Is(wrapped, original) {
		t.Error("errors.Is should find original error")
	}
	if wrapped.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("expected HTTP 500, got %d", wrapped.HTTPStatus)
	}
}

func TestError_String(t *testing.T) {
	err := New(ErrDealLocked, "deal is locked")
	expected := "[DEAL_LOCKED] deal is locked"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}

	original := fmt.Errorf("db error")
	wrapped := Wrap(original, ErrInternal, "operation failed")
	if wrapped.Error() != "[INTERNAL_ERROR] operation failed: db error" {
		t.Errorf("unexpected error string: %q", wrapped.Error())
	}
}

func TestIs(t *testing.T) {
	err := New(ErrLimitExceeded, "credit limit exceeded")
	if !Is(err, ErrLimitExceeded) {
		t.Error("Is should return true for matching code")
	}
	if Is(err, ErrNotFound) {
		t.Error("Is should return false for non-matching code")
	}
	if Is(fmt.Errorf("plain error"), ErrNotFound) {
		t.Error("Is should return false for non-AppError")
	}
}

func TestAsAppError(t *testing.T) {
	err := New(ErrSelfApproval, "cannot approve own deal")
	wrapped := fmt.Errorf("wrapper: %w", err)

	appErr, ok := AsAppError(wrapped)
	if !ok {
		t.Fatal("AsAppError should find AppError in chain")
	}
	if appErr.Code != ErrSelfApproval {
		t.Errorf("expected code %s, got %s", ErrSelfApproval, appErr.Code)
	}
}

func TestHTTPStatusForCode_Unknown(t *testing.T) {
	status := HTTPStatusForCode(ErrorCode("UNKNOWN"))
	if status != http.StatusInternalServerError {
		t.Errorf("expected 500 for unknown code, got %d", status)
	}
}

func TestAllCodes_HaveHTTPStatus(t *testing.T) {
	codes := []ErrorCode{
		ErrNotFound, ErrUnauthorized, ErrForbidden, ErrValidation,
		ErrConflict, ErrInternal, ErrDealLocked, ErrInsufficientInventory,
		ErrLimitExceeded, ErrInvalidTransition, ErrSelfApproval,
	}
	for _, code := range codes {
		status := HTTPStatusForCode(code)
		if status == 0 {
			t.Errorf("code %s has no HTTP status mapping", code)
		}
	}
}
