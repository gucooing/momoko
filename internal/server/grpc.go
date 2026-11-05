package server

import (
	"crypto/tls"
	"fmt"

	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	ggrpc "google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/conf"
	"momoko/internal/service"
)

// NewGRPCServer 是节点 gRPC 端口：远程主控以 API Key 时间戳签名调用本机的机器状态、应用、文件、容器与工具接口。
// 业务服务直接复用 HTTP 侧的实现；可调用的操作与所需权限见 node_policy.go，其余方法（含反射）一律拒绝。
func NewGRPCServer(c *conf.Server,
	authorization *Authorization,
	operationLog *service.OperationLogMiddleware,
	hs *http.Server,
	nodeAgent *biz.NodeAgentUsecase,
	nodeAgentApi *service.NodeAgentService,
	systemApi *service.SystemService,
	fileApi *service.FileService,
	instanceApi *service.InstanceService,
	dockerApi *service.DockerService,
	networkApi *service.NetworkService,
	tunnelApi *service.TunnelService,
	taskApi *service.TaskService,
	openSSHApi *service.OpenSSHService,
	monitorApi *service.MonitorService,
) (*grpc.Server, error) {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			NodeGRPCErrorMiddleware(),
			authorization.NodeGRPCMiddleware(),
			operationLog.Middleware(),
			Middleware(),
		),
		grpc.StreamInterceptor(authorization.NodeGRPCStreamInterceptor()),
		grpc.DisableReflection(),
		grpc.Options(
			ggrpc.MaxRecvMsgSize(biz.NodeMaxMessageSize),
			ggrpc.MaxSendMsgSize(biz.NodeMaxMessageSize),
			// 主控在有活跃流（WebSocket 隧道）时按 NodeKeepaliveTime 保活，强制策略需放宽到不小于该频率。
			ggrpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
				MinTime:             biz.NodeKeepaliveTime / 2,
				PermitWithoutStream: true,
			}),
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	if c.Grpc.TlsCertFile != "" || c.Grpc.TlsKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.Grpc.TlsCertFile, c.Grpc.TlsKeyFile)
		if err != nil {
			return nil, fmt.Errorf("load grpc tls certificate: %w", err)
		}
		opts = append(opts, grpc.TLSConfig(&tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterNodeAgentServer(srv, nodeAgentApi)
	v1.RegisterSystemServer(srv, systemApi)
	v1.RegisterFileManagerServer(srv, fileApi)
	v1.RegisterInstanceManagerServer(srv, instanceApi)
	v1.RegisterDockerManagerServer(srv, dockerApi)
	v1.RegisterNetworkManagerServer(srv, networkApi)
	v1.RegisterTunnelManagerServer(srv, tunnelApi)
	v1.RegisterTaskManagerServer(srv, taskApi)
	v1.RegisterOpenSSHManagerServer(srv, openSSHApi)
	v1.RegisterMonitorManagerServer(srv, monitorApi)

	startNodeTunnelServer(hs, nodeAgentApi, nodeAgent)
	return srv, nil
}
