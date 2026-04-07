package server

import (
	nethttp "net/http"

	"momoko/api/gen/v1"

	"momoko/internal/conf"
	"momoko/internal/service"
	"momoko/pkg/avatar"
	"momoko/pkg/response"
	"momoko/pkg/validate"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server,
	avatarManager *avatar.Manager,
	authorization *Authorization,
	authApi *service.AuthService,
	fileApi *service.FileService,
	userApi *service.UserService,
	systemApi *service.SystemService,
	instanceApi *service.InstanceService,
) *http.Server {
	var opts = []http.ServerOption{
		http.Filter(
			corsMiddleware(),
			avatarManager.Filter(),     // 头像服务
			distMiddleware(),           // 前端资源
			authorization.Middleware(), // 身份验证
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
	v1.RegisterAuthServiceHTTPServer(srv, authApi)
	v1.RegisterFileSystemManagerHTTPServer(srv, fileApi)
	v1.RegisterUserServiceHTTPServer(srv, userApi)
	v1.RegisterSystemHTTPServer(srv, systemApi)
	v1.RegisterInstanceManagerHTTPServer(srv, instanceApi)
	instanceApi.RegisterWsServer(srv)

	return srv
}
