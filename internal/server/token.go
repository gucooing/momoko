package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	httpm "github.com/go-kratos/kratos/v2/transport/http"
	"go.einride.tech/aip/fieldbehavior"
	"google.golang.org/protobuf/proto"

	"momoko/internal/biz"
	auth2 "momoko/internal/data/ent/gen/auth"
	"momoko/pkg/auth"
	"momoko/pkg/response"
)

// Middleware is a middleware that validates the request message with [FieldBehavior](https://google.aip.dev/203)
func Middleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			if msg, ok := req.(proto.Message); ok {
				if err := fieldbehavior.ValidateRequiredFields(msg); err != nil {
					return nil, errors.BadRequest("VALIDATOR", err.Error()).WithCause(err)
				}
			}
			return handler(ctx, req)
		}
	}
}

type publicRoute struct {
	method string
	path   string
}

type Authorization struct {
	ar biz.AuthRepo
}

func NewAuthorization(ar biz.AuthRepo) *Authorization {
	return &Authorization{
		ar: ar,
	}
}

var (
	noAuthRoutes = map[publicRoute]struct{}{
		{method: http.MethodPost, path: "/api/v1/auth/login"}:                      {},
		{method: http.MethodPost, path: "/api/v1/auth/register"}:                   {},
		{method: http.MethodPost, path: "/api/v1/auth/refresh"}:                    {},
		{method: http.MethodPost, path: "/api/v1/auth/register/email-code"}:        {},
		{method: http.MethodPost, path: "/api/v1/auth/login/email-code"}:           {},
		{method: http.MethodGet, path: "/api/v1/system/login-config"}:              {},
		{method: http.MethodGet, path: "/api/v1/public/sub2api/home"}:              {},
		{method: http.MethodGet, path: "/api/v1/public/sub2api/stats"}:             {},
		{method: http.MethodGet, path: "/api/v1/system/initialize/status"}:         {},
		{method: http.MethodPost, path: "/api/v1/system/initialize/confirm"}:       {},
		{method: http.MethodPost, path: "/api/v1/system/initialize/database/test"}: {},
		{method: http.MethodGet, path: biz.PreFileDownload}:                        {},
		{method: http.MethodPut, path: biz.PreFileUpload}:                          {},
	}
	ErrTokenInvalid = response.BadRequest(401, "token invalid")
)

// Middleware is an authentication middleware for HTTP servers.
func (a *Authorization) Middleware() httpm.FilterFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := noAuthRoutes[publicRoute{method: r.Method, path: r.URL.Path}]; ok {
				next.ServeHTTP(w, r)
				return
			}
			authorization := r.Header.Get("Authorization")
			if authorization == "" && r.Method == http.MethodGet {
				authorization = r.URL.Query().Get("accessToken")
			}
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

func (a *Authorization) GRPCMiddleware() middleware.Middleware {
	//ctx := metadata.AppendToOutgoingContext(ctx, "token", "114514")

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, ErrTokenInvalid
			}

			token := tr.RequestHeader().Get("token")
			if token == "" {
				return nil, ErrTokenInvalid
			}

			authInfo, err := auth.ParseToken(token)
			if err != nil {
				return nil, ErrTokenInvalid
			}

			ctx = auth.NewContext(ctx, authInfo)
			return handler(ctx, req)
		}
	}
}
