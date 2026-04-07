package telemetry

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/kienlongbank/treasury-api/internal/ctxutil"
)

const instrumentationName = "github.com/kienlongbank/treasury-api/internal/telemetry"

// Middleware returns a chi middleware that creates otel spans and records metrics.
func Middleware(serviceName string) func(http.Handler) http.Handler {
	tracer := otel.Tracer(instrumentationName)
	meter := otel.Meter(instrumentationName)

	requestDuration, _ := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	requestCount, _ := meter.Int64Counter(
		"http.server.request.count",
		metric.WithDescription("Total HTTP requests"),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status
			ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}

			// Create span
			ctx, span := tracer.Start(r.Context(), fmt.Sprintf("%s %s", r.Method, r.URL.Path),
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(r.Method),
					semconv.URLPath(r.URL.Path),
				),
			)
			defer span.End()

			// Serve
			next.ServeHTTP(ww, r.WithContext(ctx))

			// Post-request attributes
			duration := time.Since(start).Seconds()

			// Get chi route pattern if available
			routePattern := chi.RouteContext(r.Context())
			route := r.URL.Path
			if routePattern != nil && routePattern.RoutePattern() != "" {
				route = routePattern.RoutePattern()
			}

			attrs := []attribute.KeyValue{
				semconv.HTTPRequestMethodKey.String(r.Method),
				semconv.HTTPResponseStatusCode(ww.status),
				semconv.HTTPRoute(route),
			}

			// Add user.id if available
			if userID := ctxutil.GetUserID(ctx); userID != "" {
				attrs = append(attrs, attribute.String("user.id", userID))
			}

			span.SetAttributes(attrs...)
			requestDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
			requestCount.Add(ctx, 1, metric.WithAttributes(attrs...))
		})
	}
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
	written bool
}

func (w *statusWriter) WriteHeader(status int) {
	if !w.written {
		w.status = status
		w.written = true
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}

// Flush implements http.Flusher for SSE/streaming support.
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap returns the underlying ResponseWriter for http.NewResponseController.
func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
