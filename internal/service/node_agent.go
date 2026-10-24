package service

import (
	"context"
	"net"

	"google.golang.org/grpc/peer"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/nodetunnel"
)

// NodeAgentService 是节点侧暴露给远程主控的 gRPC 服务（仅注册在 gRPC 服务器上）。
type NodeAgentService struct {
	v1.UnimplementedNodeAgentServer

	uc       *biz.NodeAgentUsecase
	listener *nodetunnel.Listener
}

func NewNodeAgentService(uc *biz.NodeAgentUsecase) *NodeAgentService {
	return &NodeAgentService{
		uc:       uc,
		listener: nodetunnel.NewListener(),
	}
}

// TunnelListener 返回隧道连接的监听器，由进程内 HTTP 服务消费。
func (s *NodeAgentService) TunnelListener() net.Listener {
	return s.listener
}

func (s *NodeAgentService) AgentInfo(ctx context.Context, _ *v1.AgentInfoRequest) (*v1.AgentInfoResponse, error) {
	principal, ok := biz.NodePrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrNodeAuthFailed
	}
	return &v1.AgentInfoResponse{Info: s.uc.AgentInfo(ctx, principal)}, nil
}

// Tunnel 把一条已鉴权的流作为 HTTP 连接交给进程内 HTTP 服务，直到连接关闭或主控断开。
func (s *NodeAgentService) Tunnel(stream v1.NodeAgent_TunnelServer) error {
	ctx := stream.Context()
	principal, ok := biz.NodePrincipalFromContext(ctx)
	if !ok {
		return biz.ErrNodeAuthFailed
	}
	var remote net.Addr
	if p, ok := peer.FromContext(ctx); ok {
		remote = p.Addr
	}
	conn := nodetunnel.NewServerConn(stream, remote, principal)
	if err := s.listener.Push(ctx, conn); err != nil {
		return err
	}
	select {
	case <-conn.Done():
	case <-ctx.Done():
		_ = conn.Close()
	}
	return nil
}
