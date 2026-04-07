// Package ctxutil provides helpers for storing and retrieving values from context.
package ctxutil

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const (
	keyUserID   contextKey = "user_id"
	keyRoles    contextKey = "user_roles"
	keyBranchID contextKey = "branch_id"
	keyReqID    contextKey = "request_id"
)

// WithUserID stores the user ID in context.
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, keyUserID, userID)
}

// GetUserID retrieves the user ID from context as a string.
func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(keyUserID).(uuid.UUID); ok {
		return v.String()
	}
	return ""
}

// GetUserUUID retrieves the user ID from context as a UUID.
func GetUserUUID(ctx context.Context) uuid.UUID {
	if v, ok := ctx.Value(keyUserID).(uuid.UUID); ok {
		return v
	}
	return uuid.Nil
}

// WithRoles stores the user roles in context.
func WithRoles(ctx context.Context, roles []string) context.Context {
	return context.WithValue(ctx, keyRoles, roles)
}

// GetRoles retrieves the user roles from context.
func GetRoles(ctx context.Context) []string {
	if v, ok := ctx.Value(keyRoles).([]string); ok {
		return v
	}
	return nil
}

// WithBranchID stores the branch ID in context.
func WithBranchID(ctx context.Context, branchID string) context.Context {
	return context.WithValue(ctx, keyBranchID, branchID)
}

// GetBranchID retrieves the branch ID from context.
func GetBranchID(ctx context.Context) string {
	if v, ok := ctx.Value(keyBranchID).(string); ok {
		return v
	}
	return ""
}

// WithRequestID stores the request ID in context.
func WithRequestID(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, keyReqID, reqID)
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(keyReqID).(string); ok {
		return v
	}
	return ""
}

// GetTraceID extracts the trace ID from the otel span in context.
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID extracts the span ID from the otel span in context.
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}
