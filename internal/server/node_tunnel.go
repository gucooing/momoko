package server

import (
	"context"
	"net"
	"net/http"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"

	"momoko/internal/biz"
	"momoko/internal/service"
	"momoko/pkg/nodetunnel"
	"momoko/pkg/response"
)

// startNodeTunnelServer 在节点上启动进程内 HTTP 服务，承接经 NodeAgent.Tunnel 进入的连接，
// 请求最终交给本机 HTTP 服务处理（复用 WebSocket 终端、预签名上传/下载等原始 handler）。
func startNodeTunnelServer(hs *khttp.Server, agentApi *service.NodeAgentService, agent *biz.NodeAgentUsecase) {
	srv := &http.Server{
		Handler: nodeTunnelGate(hs, agent),
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			if conn, ok := c.(*nodetunnel.ServerConn); ok {
				if principal, ok := conn.Principal().(*biz.NodePrincipal); ok {
					return biz.NewNodePrincipalContext(ctx, principal)
				}
			}
			return ctx
		},
		ReadHeaderTimeout: 30 * time.Second,
	}
	go func() {
		_ = srv.Serve(agentApi.TunnelListener())
	}()
}

// nodeTunnelGate 只放行登记过的原始路径，并按路径校验 API Key 授予范围；其余路径一律拒绝，
// 避免隧道成为访问节点全部 HTTP 接口的旁路。
func nodeTunnelGate(next http.Handler, agent *biz.NodeAgentUsecase) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := biz.NodePrincipalFromContext(r.Context())
		if !ok {
			response.WriteError(w, r, biz.ErrNodeAuthFailed)
			return
		}
		perm, allowed := nodeTunnelPaths[r.URL.Path]
		if !allowed {
			response.WriteError(w, r, biz.ErrNodeOperationForbidden)
			return
		}
		if perm != "" {
			if err := agent.Authorize(r.Context(), principal, perm); err != nil {
				response.WriteError(w, r, err)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
