package middleware

import (
	"net/http"

	"github.com/kienlongbank/treasury-api/internal/ctxutil"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/httputil"
	"github.com/kienlongbank/treasury-api/pkg/security"
)

// RequirePermission returns a middleware that checks if the authenticated user
// has the required permission. Must be used after Auth middleware.
func RequirePermission(rbac *security.RBACChecker, permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles := ctxutil.GetRoles(r.Context())
			if len(roles) == 0 {
				httputil.Error(w, r, apperror.New(apperror.ErrForbidden, "no roles assigned"))
				return
			}

			if !rbac.HasAnyPermission(roles, permission) {
				httputil.Error(w, r, apperror.New(apperror.ErrForbidden, "insufficient permissions"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission returns a middleware that checks if the authenticated user
// has at least one of the required permissions. Must be used after Auth middleware.
func RequireAnyPermission(rbac *security.RBACChecker, permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles := ctxutil.GetRoles(r.Context())
			if len(roles) == 0 {
				httputil.Error(w, r, apperror.New(apperror.ErrForbidden, "no roles assigned"))
				return
			}

			if !rbac.HasAnyOfPermissions(roles, permissions...) {
				httputil.Error(w, r, apperror.New(apperror.ErrForbidden, "insufficient permissions"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

