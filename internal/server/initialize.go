package server

import (
	"momoko/api/gen/v1"
	"momoko/internal/conf"
	"momoko/internal/service"
	"momoko/pkg/response"
	"momoko/pkg/validate"

	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
)

func NewInitializeHTTPServer(c *conf.Server, initializeApi *service.InitializeService) *http.Server {
	var opts = []http.ServerOption{
		http.Filter(
			corsMiddleware(),
			distMiddleware(),
		),
		http.Middleware(
			recovery.Recovery(),
			validate.Middleware(),
		),
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
	v1.RegisterInitializeHTTPServer(srv, initializeApi)
	return srv
}
