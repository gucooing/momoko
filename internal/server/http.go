package server

import (
	nethttp "net/http"

	adminV1 "momoko/api/gen/admin/v1"
	authV1 "momoko/api/gen/auth/v1"

	"momoko/internal/conf"
	"momoko/internal/service"
	"momoko/pkg/auth"
	"momoko/pkg/response"
	"momoko/pkg/validate"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server,
	admin *service.AdminService,
	authApi *service.AuthService,
) *http.Server {
	var opts = []http.ServerOption{
		http.Filter(
			corsMiddleware(),
			auth.Middleware(),
		),
		http.Middleware(
			recovery.Recovery(),
			validate.Middleware(),
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
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	adminV1.RegisterAdminServiceHTTPServer(srv, admin)
	authV1.RegisterAuthServiceHTTPServer(srv, authApi)

	return srv
}
