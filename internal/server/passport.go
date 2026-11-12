package server

import (
	"context"
	nethttp "net/http"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/conf"
	"momoko/internal/service"
	"momoko/pkg/passporttoken"
	"momoko/pkg/response"
)

// PassportAPIPrefix 用户通行证接口的路径前缀。通行证是独立端口上的独立服务，这个端口上只有通行证自己的接口，
// 所以路由就是普通的 /api/v1/...（管理后台的接口在这里根本不存在）。
const PassportAPIPrefix = "/api/v1/"

// passportPublicRoutes 通行证上不需要登录的接口；其余 /api/v1/ 接口都要通行证访问令牌。
var passportPublicRoutes = map[publicRoute]struct{}{
	{method: nethttp.MethodGet, path: "/api/v1/config"}:                          {},
	{method: nethttp.MethodPost, path: "/api/v1/auth/register/email-code"}:       {},
	{method: nethttp.MethodPost, path: "/api/v1/auth/register"}:                  {},
	{method: nethttp.MethodPost, path: "/api/v1/auth/login/email-code"}:          {},
	{method: nethttp.MethodPost, path: "/api/v1/auth/login"}:                     {},
	{method: nethttp.MethodPost, path: "/api/v1/auth/login/mfa"}:                 {},
	{method: nethttp.MethodPost, path: "/api/v1/auth/refresh"}:                   {},
	{method: nethttp.MethodPost, path: "/api/v1/auth/password/reset/email-code"}: {},
	{method: nethttp.MethodPost, path: "/api/v1/auth/password/reset"}:            {},
	{method: nethttp.MethodGet, path: "/api/v1/oidc/jwks"}:                       {},
	{method: nethttp.MethodPost, path: "/api/v1/oidc/token"}:                     {},
	{method: nethttp.MethodGet, path: "/api/v1/oidc/userinfo"}:                   {},
	{method: nethttp.MethodPost, path: "/api/v1/oidc/userinfo"}:                  {},
	{method: nethttp.MethodGet, path: "/.well-known/openid-configuration"}:       {},
}

// passportAuthFilter 通行证的鉴权：只认通行证访问令牌（管理后台的令牌签名域不同，在这里一律无效）。
func passportAuthFilter(uc *biz.PassportUsecase) http.FilterFunc {
	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			if !strings.HasPrefix(r.URL.Path, PassportAPIPrefix) || r.Method == nethttp.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			if _, ok := passportPublicRoutes[publicRoute{method: r.Method, path: r.URL.Path}]; ok {
				next.ServeHTTP(w, r)
				return
			}
			parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				response.WriteError(w, r, biz.ErrPassportUnauthorized)
				return
			}
			claims, err := uc.Authenticate(r.Context(), strings.TrimSpace(parts[1]))
			if err != nil {
				response.WriteError(w, r, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(passporttoken.NewContext(r.Context(), claims)))
		})
	}
}

// PassportServer 用户通行证（玩家端）独立 HTTP server：只注册玩家路由与它自己的前端产物，
// 管理接口在这个端口上根本不存在。
type PassportServer struct {
	*http.Server
	enabled bool
}

// Start 未启用时什么也不做。
func (p *PassportServer) Start(ctx context.Context) error {
	if !p.enabled {
		return nil
	}
	return p.Server.Start(ctx)
}

// Stop 未启用时什么也不做。
func (p *PassportServer) Stop(ctx context.Context) error {
	if !p.enabled {
		return nil
	}
	return p.Server.Stop(ctx)
}

// PassportEnabled 用户通行证是否启用（不写时默认启用）。
func PassportEnabled(c *conf.Passport) bool {
	return c == nil || c.Enabled == nil || *c.Enabled
}

// NewPassportServer 创建用户通行证 server（默认 0.0.0.0:22634）。
func NewPassportServer(c *conf.Passport, passportApi *service.PassportService) *PassportServer {
	if !PassportEnabled(c) {
		return &PassportServer{enabled: false}
	}
	addr := "0.0.0.0:22634"
	if c != nil && c.Addr != "" {
		addr = c.Addr
	}
	opts := []http.ServerOption{
		http.Address(addr),
		http.Filter(
			corsMiddleware(),
			passportDistMiddleware(), // 通行证前端与游戏素材
			passportAuthFilter(passportApi.Usecase()),
		),
		http.Middleware(
			recovery.Recovery(),
			Middleware(),
		),
		http.NotFoundHandler(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			response.WriteError(w, r, errors.NotFound("NOT_FOUND", "Not Found"))
		})),
		http.MethodNotAllowedHandler(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			response.WriteError(w, r, errors.New(nethttp.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method Not Allowed"))
		})),
		http.ResponseEncoder(response.ResponseEncoder),
		http.ErrorEncoder(response.ErrorEncoder),
	}
	if c != nil && c.Timeout != nil {
		opts = append(opts, http.Timeout(c.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	v1.RegisterPassportServiceHTTPServer(srv, passportApi)
	passportApi.RegisterOIDCServer(srv)
	return &PassportServer{Server: srv, enabled: true}
}
