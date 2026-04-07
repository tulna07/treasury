package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/kienlongbank/treasury-api/internal/logger"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/httputil"
	"go.uber.org/zap"
)

// Recovery returns a middleware that recovers from panics, logs the error,
// and records it on the otel span.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := string(debug.Stack())
				log := logger.FromContext(r.Context())
				log.Error("panic recovered",
					zap.Any("panic", rec),
					zap.String("stack", stack),
				)

				// Record on otel span
				span := trace.SpanFromContext(r.Context())
				span.SetStatus(codes.Error, fmt.Sprintf("panic: %v", rec))
				span.SetAttributes(attribute.String("panic.stack", stack))

				httputil.Error(w, r, apperror.New(apperror.ErrInternal, "internal server error"))
			}
		}()

		next.ServeHTTP(w, r)
	})
}
