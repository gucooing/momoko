package server

import (
	"net/http"
	"strings"
	"time"

	httpm "github.com/go-kratos/kratos/v2/transport/http"

	"momoko/internal/biz"
	auth2 "momoko/internal/data/ent/auth"
	"momoko/pkg/auth"
	"momoko/pkg/response"
)

type Authorization struct {
	ar biz.AuthRepo
}

func NewAuthorization(ar biz.AuthRepo) *Authorization {
	return &Authorization{
		ar: ar,
	}
}

var (
	noAuthPaths = map[string]struct{}{
		"/api/v1/auth/login":   {},
		"/api/v1/auth/refresh": {},
	}
	ErrTokenInvalid = response.BadRequest(401, "token invalid")
)

// Middleware is an authentication middleware for HTTP servers.
func (a *Authorization) Middleware() httpm.FilterFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := noAuthPaths[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}
			authorization := r.Header.Get("Authorization")
			tokens := strings.Split(authorization, " ")
			if len(tokens) != 2 || tokens[0] != "Bearer" {
				response.WriteError(w, r, ErrTokenInvalid)
				return
			}
			authInfo, err := auth.ParseToken(tokens[1])
			if err != nil {
				response.WriteError(w, r, ErrTokenInvalid)
				return
			}
			if authInfo.ExpiresAt.Before(time.Now()) {
				response.WriteError(w, r, ErrTokenInvalid)
				return
			}
			authData, err := a.ar.GetAuthByDeviceID(r.Context(), authInfo.DeviceId, auth2.TypeToken)
			if err != nil {
				response.WriteError(w, r, ErrTokenInvalid)
				return
			}
			if authData.SessionID != authInfo.SessionID ||
				authData.UserID != authInfo.UserID {
				response.WriteError(w, r, ErrTokenInvalid)
				return
			}
			ctx := auth.NewContext(r.Context(), authInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
