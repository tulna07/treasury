package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Middleware returns a chi middleware that logs each request with duration, status, and trace context.
func Middleware(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &responseLogger{ResponseWriter: w, status: http.StatusOK}

			// Store logger in context
			ctx := WithContext(r.Context(), log)
			next.ServeHTTP(ww, r.WithContext(ctx))

			duration := time.Since(start)
			contextLog := FromContext(r.Context())

			contextLog.Info("HTTP request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("query", r.URL.RawQuery),
				zap.Int("status", ww.status),
				zap.Duration("duration", duration),
				zap.Int("bytes", ww.bytes),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()),
			)
		})
	}
}

type responseLogger struct {
	http.ResponseWriter
	status  int
	bytes   int
	written bool
}

func (w *responseLogger) WriteHeader(status int) {
	if !w.written {
		w.status = status
		w.written = true
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseLogger) Write(b []byte) (int, error) {
	if !w.written {
		w.written = true
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// Flush implements http.Flusher for SSE support.
func (w *responseLogger) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap returns the underlying ResponseWriter so http.NewResponseController can find Flusher.
func (w *responseLogger) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// StreamBypass returns the raw underlying http.ResponseWriter, bypassing the logger wrapper.
// Use for SSE/streaming endpoints where the wrapper interferes with chunked encoding.
func (w *responseLogger) StreamBypass() http.ResponseWriter {
	return w.ResponseWriter
}
