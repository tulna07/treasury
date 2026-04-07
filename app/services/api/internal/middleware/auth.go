// Package middleware provides HTTP middleware for the Treasury API.
package middleware

import (
	"net/http"
	"strings"

	"github.com/kienlongbank/treasury-api/internal/ctxutil"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/httputil"
	"github.com/kienlongbank/treasury-api/pkg/security"
)

// Auth returns a middleware that validates JWT tokens.
// Token source priority: 1) treasury_access_token cookie (browser), 2) Authorization header (API/Swagger).
func Auth(jwtMgr *security.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""

			// 1. Try cookie first (browser clients)
			if cookie, err := r.Cookie("treasury_access_token"); err == nil && cookie.Value != "" {
				token = cookie.Value
			}

			// 2. Fall back to Authorization header (API clients, Swagger)
			if token == "" {
				authHeader := r.Header.Get("Authorization")
				if authHeader != "" {
					parts := strings.SplitN(authHeader, " ", 2)
					if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
						token = parts[1]
					}
				}
			}

			if token == "" {
				httputil.Error(w, r, apperror.New(apperror.ErrUnauthorized, "authentication required"))
				return
			}

			claims, err := jwtMgr.ValidateToken(token)
			if err != nil {
				httputil.Error(w, r, apperror.Wrap(err, apperror.ErrUnauthorized, "invalid or expired token"))
				return
			}

			// Store claims in context
			ctx := r.Context()
			ctx = ctxutil.WithUserID(ctx, claims.UserID)
			ctx = ctxutil.WithRoles(ctx, claims.Roles)
			ctx = ctxutil.WithBranchID(ctx, claims.BranchID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
