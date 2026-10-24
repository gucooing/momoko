package server

import (
	"context"
	"errors"
	"strconv"
	"strings"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"momoko/internal/biz"
	"momoko/pkg/auth"
	"momoko/pkg/response"
)

const (
	grpcHealthPrefix    = "/grpc.health.v1.Health/"
	nodeTunnelOperation = "/v1.NodeAgent/Tunnel"
)

// toNodeGRPCError 把业务错误转为 Kratos 错误：gRPC 状态里携带原始业务码与提示，主控据此还原。
func toNodeGRPCError(err error) error {
	if err == nil {
		return nil
	}
	var re response.Error
	if errors.As(err, &re) {
		return kerrors.New(re.Code, biz.NodeErrorReason, re.Message).
			WithMetadata(map[string]string{"code": strconv.Itoa(re.Code)})
	}
	return err
}

// NodeGRPCErrorMiddleware 统一转换节点 gRPC 接口返回的业务错误。
func NodeGRPCErrorMiddleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			reply, err := handler(ctx, req)
			if err != nil {
				return nil, toNodeGRPCError(err)
			}
			return reply, nil
		}
	}
}

// authenticateNode 校验签名并确认操作对远程开放、授予范围覆盖其所需权限。
func (a *Authorization) authenticateNode(ctx context.Context, operation string) (context.Context, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	principal, err := a.agent.Authenticate(ctx, md, operation)
	if err != nil {
		return nil, err
	}
	perm, allowed := nodeOperationPermission(operation)
	if !allowed {
		return nil, biz.ErrNodeOperationForbidden
	}
	if perm != "" {
		if err := a.agent.Authorize(ctx, principal, perm); err != nil {
			return nil, err
		}
	}
	return biz.NewNodePrincipalContext(auth.NewContext(ctx, principal.Auth()), principal), nil
}

// NodeGRPCMiddleware 是节点 gRPC 端口的一元调用鉴权：除健康检查外都必须携带 API Key 时间戳签名，
// 并以 API Key 所属用户身份执行。
func (a *Authorization) NodeGRPCMiddleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, biz.ErrNodeAuthFailed
			}
			if strings.HasPrefix(tr.Operation(), grpcHealthPrefix) {
				return handler(ctx, req)
			}
			ctx, err := a.authenticateNode(ctx, tr.Operation())
			if err != nil {
				return nil, err
			}
			return handler(ctx, req)
		}
	}
}

type principalStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *principalStream) Context() context.Context {
	return s.ctx
}

// NodeGRPCStreamInterceptor 是流式调用鉴权：只开放 HTTP 隧道（及健康检查），其余流（反射等）一律拒绝。
func (a *Authorization) NodeGRPCStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if strings.HasPrefix(info.FullMethod, grpcHealthPrefix) {
			return handler(srv, ss)
		}
		if info.FullMethod != nodeTunnelOperation {
			return toNodeGRPCError(biz.ErrNodeOperationForbidden)
		}
		ctx, err := a.authenticateNode(ss.Context(), info.FullMethod)
		if err != nil {
			return toNodeGRPCError(err)
		}
		return handler(srv, &principalStream{ServerStream: ss, ctx: ctx})
	}
}
