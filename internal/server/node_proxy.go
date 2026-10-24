package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"momoko/internal/biz"
	"momoko/pkg/auth"
	"momoko/pkg/cache"
	"momoko/pkg/constant"
	"momoko/pkg/nodeauth"
	"momoko/pkg/response"
	"momoko/pkg/utils"
)

const (
	// NodeHeader 前端在远程节点上下文中为每个请求携带的目标节点 id。
	NodeHeader = "X-Momoko-Node"
	// NodeQueryKey 浏览器无法设置请求头的场景（WebSocket、预签名链接）改用该查询参数。
	NodeQueryKey = "momokoNode"
	// LocalNodeID 表示主控本机。
	LocalNodeID = "local"

	nodeTunnelHost = "momoko-node"
)

// NodeProxy 把带节点标识的请求转发到远程节点：
//   - 一元接口：在 Kratos 中间件里按 gRPC 完整方法名直接 Invoke，请求/响应即生成的 proto 消息，无需逐接口适配；
//   - WebSocket / 预签名上传下载：经 NodeAgent.Tunnel 双向流反向代理到节点自身的 HTTP 处理逻辑。
//
// 主控先按本机 RBAC 校验当前用户（PermissionMiddleware 在本中间件之前执行），节点再按 API Key 授予范围校验，
// 两者取交集。
type NodeProxy struct {
	nodes  *biz.NodeUsecase
	system *biz.SystemUsecase
	users  biz.UserRepo

	replies sync.Map // operation → protoreflect.MessageType
	proxies sync.Map // node id → *nodeReverseProxy
	names   *cache.Cache[string, string]
}

type nodeReverseProxy struct {
	client *biz.NodeClient
	proxy  *httputil.ReverseProxy
}

func NewNodeProxy(nodes *biz.NodeUsecase, system *biz.SystemUsecase, users biz.UserRepo) *NodeProxy {
	return &NodeProxy{
		nodes:  nodes,
		system: system,
		users:  users,
		names:  cache.New[string, string](5 * time.Minute),
	}
}

func isLocalNode(nodeID string) bool {
	return nodeID == "" || nodeID == LocalNodeID
}

// Middleware 转发一元接口。未开放远程的“按节点区分”操作直接拒绝，其余（系统、用户等全局功能）仍在主控本机执行。
func (p *NodeProxy) Middleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			httpReq, ok := khttp.RequestFromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}
			nodeID := strings.TrimSpace(httpReq.Header.Get(NodeHeader))
			if isLocalNode(nodeID) {
				return handler(ctx, req)
			}
			// 未登录的公开接口（公开分享页等）始终在本机处理，即使浏览器标签页仍处于远程节点上下文。
			if _, ok := auth.FromContext(ctx); !ok {
				return handler(ctx, req)
			}
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}
			operation := tr.Operation()
			if _, remote := nodeOperationPermission(operation); !remote {
				if isNodeScopedOperation(operation) {
					return nil, biz.ErrNodeOperationUnsupported
				}
				return handler(ctx, req)
			}
			msg, ok := req.(proto.Message)
			if !ok {
				return nil, biz.ErrNodeOperationUnsupported
			}
			if err := p.system.Check(ctx, constant.NodeView); err != nil {
				return nil, err
			}
			client, err := p.nodes.NodeClient(ctx, nodeID)
			if err != nil {
				return nil, err
			}
			reply, err := p.newReply(operation)
			if err != nil {
				return nil, biz.ErrNodeOperationUnsupported
			}
			biz.MarkNodeTrace(ctx, client.Config)
			ctx = nodeauth.WithOperator(ctx, p.operator(ctx))
			if err := client.Invoke(ctx, operation, msg, reply); err != nil {
				return nil, biz.NodeCallError(client.Config.Name, err)
			}
			return reply, nil
		}
	}
}

// newReply 按方法描述符构造响应消息；生成的 HTTP handler 会把结果断言为具体类型，必须返回同一类型。
func (p *NodeProxy) newReply(operation string) (proto.Message, error) {
	if mt, ok := p.replies.Load(operation); ok {
		return mt.(protoreflect.MessageType).New().Interface(), nil
	}
	name := strings.TrimPrefix(operation, "/")
	i := strings.LastIndex(name, "/")
	if i <= 0 {
		return nil, fmt.Errorf("invalid operation %q", operation)
	}
	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(name[:i]))
	if err != nil {
		return nil, err
	}
	service, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("%s is not a service", name[:i])
	}
	method := service.Methods().ByName(protoreflect.Name(name[i+1:]))
	if method == nil {
		return nil, fmt.Errorf("method %s not found", operation)
	}
	mt, err := protoregistry.GlobalTypes.FindMessageByName(method.Output().FullName())
	if err != nil {
		return nil, err
	}
	p.replies.Store(operation, mt)
	return mt.New().Interface(), nil
}

// operator 返回当前主控用户名，随签名写入节点审计。
func (p *NodeProxy) operator(ctx context.Context) string {
	a, ok := auth.FromContext(ctx)
	if !ok {
		return ""
	}
	name, ok := p.names.GetByAdd(a.UserID, func() (string, error) {
		u, err := p.users.FindByID(ctx, a.UserID)
		if err != nil {
			return "", err
		}
		return u.Username, nil
	})
	if !ok || name == "" {
		return a.UserID
	}
	return name
}

// Filter 经隧道转发原始 HTTP 路径（WebSocket 终端、预签名上传/下载）。
// 需放在身份验证 filter 之后：WebSocket 已完成主控登录态与本机 RBAC 校验；预签名路径由节点验签自授权。
func (p *NodeProxy) Filter() khttp.FilterFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nodeID := strings.TrimSpace(r.URL.Query().Get(NodeQueryKey))
			if nodeID == "" {
				nodeID = strings.TrimSpace(r.Header.Get(NodeHeader))
			}
			if isLocalNode(nodeID) {
				next.ServeHTTP(w, r)
				return
			}
			if _, ok := nodeTunnelPaths[r.URL.Path]; !ok {
				next.ServeHTTP(w, r)
				return
			}
			if !isPresignedPath(r.URL.Path) {
				if _, ok := auth.FromContext(r.Context()); !ok {
					response.WriteError(w, r, ErrTokenInvalid)
					return
				}
				if err := p.system.Check(r.Context(), constant.NodeView); err != nil {
					response.WriteError(w, r, err)
					return
				}
			}
			client, err := p.nodes.NodeClient(r.Context(), nodeID)
			if err != nil {
				response.WriteError(w, r, err)
				return
			}

			out := r.Clone(nodeauth.WithOperator(r.Context(), p.operator(r.Context())))
			// 主控凭据不进入节点：去掉节点标识、登录 token 与 Cookie。
			query := out.URL.Query()
			query.Del(NodeQueryKey)
			query.Del("accessToken")
			out.URL.RawQuery = query.Encode()
			out.Header.Del("Authorization")
			out.Header.Del("Cookie")
			out.Header.Del(NodeHeader)
			p.reverseProxy(client).ServeHTTP(w, out)
		})
	}
}

func (p *NodeProxy) reverseProxy(client *biz.NodeClient) *httputil.ReverseProxy {
	if v, ok := p.proxies.Load(client.Config.ID); ok {
		if cached := v.(*nodeReverseProxy); cached.client == client {
			return cached.proxy
		}
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return client.DialTunnel(ctx)
		},
		// 一条隧道流对应一次请求，避免空闲连接长期占用节点上的流。
		DisableKeepAlives:     true,
		DisableCompression:    true,
		ResponseHeaderTimeout: time.Minute,
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL.Scheme = "http"
			pr.Out.URL.Host = nodeTunnelHost
			pr.Out.Host = nodeTunnelHost
			pr.Out.Header.Set("X-Forwarded-For", utils.ClientIP(pr.In))
		},
		Transport:     transport,
		FlushInterval: -1,
		// 主控 CORS filter 已写过跨域头，丢弃节点返回的同名头，避免重复导致浏览器拒绝。
		ModifyResponse: func(res *http.Response) error {
			for key := range res.Header {
				if strings.HasPrefix(key, "Access-Control-") {
					res.Header.Del(key)
				}
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			response.WriteError(w, r, biz.NodeCallError(client.Config.Name, err))
		},
	}
	p.proxies.Store(client.Config.ID, &nodeReverseProxy{client: client, proxy: proxy})
	return proxy
}
