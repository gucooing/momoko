package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	httpm "github.com/go-kratos/kratos/v2/transport/http"
	"go.einride.tech/aip/fieldbehavior"
	"google.golang.org/protobuf/proto"

	"momoko/internal/biz"
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
	ar     biz.AuthRepo
	system *biz.SystemUsecase
}

func NewAuthorization(ar biz.AuthRepo, system *biz.SystemUsecase) *Authorization {
	return &Authorization{
		ar:     ar,
		system: system,
	}
}

// isWebSocketUpgrade 判断请求是否为 WebSocket 握手（Upgrade: websocket）。
// 仅此类请求允许用 query 参数携带 token（浏览器 WS API 无法设置请求头）。
func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

var (
	noAuthRoutes = map[publicRoute]struct{}{
		{method: http.MethodPost, path: "/api/v1/auth/login"}:                           {},
		{method: http.MethodPost, path: "/api/v1/auth/register"}:                        {},
		{method: http.MethodPost, path: "/api/v1/auth/refresh"}:                         {},
		{method: http.MethodPost, path: "/api/v1/auth/register/email-code"}:             {},
		{method: http.MethodPost, path: "/api/v1/auth/login/email-code"}:                {},
		{method: http.MethodGet, path: "/api/v1/system/login-config"}:                   {},
		{method: http.MethodGet, path: "/api/v1/public/sub2api/home"}:                   {},
		{method: http.MethodGet, path: "/api/v1/public/sub2api/stats"}:                  {},
		{method: http.MethodGet, path: "/api/v1/system/initialize/status"}:              {},
		{method: http.MethodPost, path: "/api/v1/system/initialize/confirm"}:            {},
		{method: http.MethodPost, path: "/api/v1/system/initialize/database/test"}:      {},
		{method: http.MethodGet, path: "/.well-known/openid-configuration"}:             {},
		{method: http.MethodGet, path: "/api/v1/oidc/.well-known/openid-configuration"}: {},
		{method: http.MethodGet, path: "/api/v1/oidc/jwks"}:                             {},
		{method: http.MethodPost, path: "/api/v1/oidc/token"}:                           {},
		{method: http.MethodGet, path: "/api/v1/oidc/userinfo"}:                         {},
		{method: http.MethodPost, path: "/api/v1/oidc/userinfo"}:                        {},
		{method: http.MethodGet, path: biz.PreFileDownload}:                             {},
		{method: http.MethodPut, path: biz.PreFileUpload}:                               {},
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
			// 公开路由前缀免 JWT：业务层（如 Imagine）自行用 X-Sub2API-Token 校验 sub2api token。
			if strings.HasPrefix(r.URL.Path, "/api/v1/public/") {
				next.ServeHTTP(w, r)
				return
			}
			authorization := r.Header.Get("Authorization")
			// 仅 WebSocket 握手允许用 query 参数携带 token，避免普通请求把 token 暴露到
			// 访问日志 / 浏览器历史 / Referer。
			if authorization == "" && isWebSocketUpgrade(r) {
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
			// access JWT 必须：kind=access，且会话噪声匹配。
			if authInfo.Kind != auth.TokenKindAccess {
				response.WriteError(w, r, ErrTokenInvalid)
				return
			}
			authData, err := a.ar.GetAuth(r.Context(), authInfo.SessionID)
			if err != nil {
				response.WriteError(w, r, ErrTokenInvalid)
				return
			}
			if authData.DeviceID != authInfo.DeviceId ||
				authData.UserID != authInfo.UserID ||
				authData.AccessNoise == "" ||
				authData.AccessNoise != authInfo.Noise {
				response.WriteError(w, r, ErrTokenInvalid)
				return
			}
			ctx := auth.NewContext(r.Context(), authInfo)
			// 原始 WS 路由（终端/容器 exec 等）不经 operation 鉴权中间件，这里按路径补做权限校验。
			if err := a.checkWSPermission(ctx, r.URL.Path); err != nil {
				response.WriteError(w, r, err)
				return
			}
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
