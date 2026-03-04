package auth

import (
	"net/http"

	"momoko/pkg/response"

	"github.com/go-kratos/kratos/v2/errors"
	httpm "github.com/go-kratos/kratos/v2/transport/http"
)

var (
	noAuthPaths = map[string]struct{}{
		"/v1/auth/login":   {},
		"/v1/auth/refresh": {},
	}
	AuthSecretKey   = "123456"
	ErrUnauthorized = errors.Unauthorized("UNAUTHORIZED", "Token is invalid")
)

// Middleware is an authentication middleware for HTTP servers.
func Middleware() httpm.FilterFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := noAuthPaths[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}
			token := r.Header.Get("Authorization")
			auth, err := ParseToken(token, AuthSecretKey)
			if err != nil {
				response.WriteError(w, r, ErrUnauthorized)
				return
			}
			ctx := NewContext(r.Context(), auth)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
