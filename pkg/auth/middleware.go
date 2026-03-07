package auth

import (
	"net/http"
	"strings"
	"time"

	"momoko/pkg/response"

	"github.com/go-kratos/kratos/v2/errors"
	httpm "github.com/go-kratos/kratos/v2/transport/http"
)

var (
	noAuthPaths = map[string]struct{}{
		"/api/v1/auth/login":   {},
		"/api/v1/auth/refresh": {},
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
			authorization := r.Header.Get("Authorization")
			tokens := strings.Split(authorization, " ")
			if len(tokens) != 2 || tokens[0] != "Bearer" {
				response.WriteError(w, r, ErrUnauthorized)
				return
			}
			auth, err := ParseToken(tokens[1])
			if err != nil {
				response.WriteError(w, r, ErrUnauthorized)
				return
			}
			if auth.ExpiresAt.Before(time.Now()) {
				response.WriteError(w, r, response.BadRequest(401, "token过期"))
				return
			}
			ctx := NewContext(r.Context(), auth)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
