package middleware

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/kienlongbank/treasury-api/internal/ctxutil"
)

// RequestID returns a middleware that ensures every request has an X-Request-ID.
// If the client provides one, it is used; otherwise a new UUID is generated.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}

		w.Header().Set("X-Request-ID", reqID)
		ctx := ctxutil.WithRequestID(r.Context(), reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
