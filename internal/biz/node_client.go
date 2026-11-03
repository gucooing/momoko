package biz

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strconv"
	"sync"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/nodeauth"
	"momoko/pkg/nodetunnel"
)

const (
	// NodeMaxMessageSize 主控与节点之间单条 gRPC 消息上限（文件编辑等请求体可能超过默认 4MB）。
	NodeMaxMessageSize = 64 << 20
	// NodeErrorReason 节点把业务错误转成 gRPC 状态时使用的 reason，metadata["code"] 携带原始业务码。
	NodeErrorReason = "MOMOKO"
	// NodeKeepaliveTime 主控在有活跃流时的保活间隔；节点侧强制策略需不大于该值。
	NodeKeepaliveTime = 30 * time.Second
)

// NodeConfig 是主控保存的远程节点配置（APIKey 为解密后的签名种子）。
type NodeConfig struct {
	ID            string
	Name          string
	Address       string
	APIKey        string
	KeyID         string
	TLS           bool
	TLSSkipVerify bool
	Remark        string
	CreateTime    time.Time
	UpdateTime    time.Time
	// LastOnlineAt 最近一次探测在线的时间（持久化，节流写入）
	LastOnlineAt *time.Time
}

func (c *NodeConfig) sameConnection(other *NodeConfig) bool {
	return c.Address == other.Address &&
		c.APIKey == other.APIKey &&
		c.TLS == other.TLS &&
		c.TLSSkipVerify == other.TLSSkipVerify
}

// NodeClient 是到单个远程节点的 gRPC 连接，所有调用都附带 API Key 时间戳签名。
type NodeClient struct {
	Config *NodeConfig
	conn   *grpc.ClientConn
	agent  v1.NodeAgentClient
}

func newNodeClient(cfg *NodeConfig) (*NodeClient, error) {
	if cfg.APIKey == "" {
		return nil, ErrNodeKeyBroken
	}
	credential := nodeauth.NewCredential(cfg.APIKey)
	transportCreds := insecure.NewCredentials()
	if cfg.TLS {
		transportCreds = credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
			// 自签证书场景由用户在节点配置中显式开启。
			InsecureSkipVerify: cfg.TLSSkipVerify, //nolint:gosec
		})
	}
	conn, err := grpc.NewClient(cfg.Address,
		grpc.WithTransportCredentials(transportCreds),
		grpc.WithChainUnaryInterceptor(credential.UnaryClientInterceptor()),
		grpc.WithChainStreamInterceptor(credential.StreamClientInterceptor()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(NodeMaxMessageSize),
			grpc.MaxCallSendMsgSize(NodeMaxMessageSize),
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:    NodeKeepaliveTime,
			Timeout: 10 * time.Second,
		}),
	)
	if err != nil {
		return nil, ErrNodeAddressInvalid
	}
	return &NodeClient{
		Config: cfg,
		conn:   conn,
		agent:  v1.NewNodeAgentClient(conn),
	}, nil
}

// Invoke 按 gRPC 完整方法名调用节点上的一元接口。
func (c *NodeClient) Invoke(ctx context.Context, method string, req, reply any) error {
	return c.conn.Invoke(ctx, method, req, reply)
}

// AgentInfo 获取节点自报信息与 API Key 生效权限。
func (c *NodeClient) AgentInfo(ctx context.Context) (*v1.NodeRemoteInfo, error) {
	resp, err := c.agent.AgentInfo(ctx, &v1.AgentInfoRequest{})
	if err != nil {
		return nil, err
	}
	return resp.GetInfo(), nil
}

// DialTunnel 建立一条 HTTP 隧道连接。连接生命周期独立于 ctx（由 Close 结束），ctx 只用于携带操作者。
func (c *NodeClient) DialTunnel(ctx context.Context) (net.Conn, error) {
	streamCtx, cancel := context.WithCancel(nodeauth.WithOperator(context.Background(), nodeauth.OperatorFromContext(ctx)))
	stream, err := c.agent.Tunnel(streamCtx)
	if err != nil {
		cancel()
		return nil, err
	}
	return nodetunnel.NewClientConn(stream, cancel, c.Config.Address), nil
}

func (c *NodeClient) Close() error {
	return c.conn.Close()
}

// nodeClients 按节点 id 缓存连接；配置变化时重建，节点被修改或删除时由用例主动失效。
type nodeClients struct {
	mu      sync.Mutex
	clients map[string]*NodeClient
}

func newNodeClients() *nodeClients {
	return &nodeClients{clients: make(map[string]*NodeClient)}
}

func (m *nodeClients) cached(id string) (*NodeClient, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	client, ok := m.clients[id]
	return client, ok
}

func (m *nodeClients) get(cfg *NodeConfig) (*NodeClient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if client, ok := m.clients[cfg.ID]; ok {
		if client.Config.sameConnection(cfg) {
			client.Config = cfg
			return client, nil
		}
		_ = client.Close()
		delete(m.clients, cfg.ID)
	}
	client, err := newNodeClient(cfg)
	if err != nil {
		return nil, err
	}
	m.clients[cfg.ID] = client
	return client, nil
}

func (m *nodeClients) remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if client, ok := m.clients[id]; ok {
		_ = client.Close()
		delete(m.clients, id)
	}
}

func (m *nodeClients) closeAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, client := range m.clients {
		_ = client.Close()
		delete(m.clients, id)
	}
}

// nodeStatusCache 保存最近一次检测结果，供列表与切换器展示；
// 同时记录每个节点最近一次在线的时间（离线后显示“最后在线”），并节流持久化。
type nodeStatusCache struct {
	mu          sync.RWMutex
	status      map[string]*v1.NodeStatus
	lastOnline  map[string]time.Time
	lastPersist map[string]time.Time
	// lastRemote 最近一次在线时的节点自报信息（离线后仍可显示版本 / 系统）
	lastRemote map[string]*v1.NodeRemoteInfo
}

// nodeOnlinePersistInterval 在线时间写库的最小间隔，避免每轮探测都写一次。
const nodeOnlinePersistInterval = 5 * time.Minute

func newNodeStatusCache() *nodeStatusCache {
	return &nodeStatusCache{
		status:      make(map[string]*v1.NodeStatus),
		lastOnline:  make(map[string]time.Time),
		lastPersist: make(map[string]time.Time),
		lastRemote:  make(map[string]*v1.NodeRemoteInfo),
	}
}

func (c *nodeStatusCache) get(id string) *v1.NodeStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if s, ok := c.status[id]; ok {
		return s
	}
	return &v1.NodeStatus{State: v1.NodeState_NodeState_Unknown}
}

// set 写入检测结果并补上最后在线时间；返回需要持久化的在线时间（无需持久化时为零值）。
func (c *nodeStatusCache) set(id string, s *v1.NodeStatus) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	var persist time.Time
	if s.State == v1.NodeState_NodeState_Online {
		now := time.Now()
		c.lastOnline[id] = now
		if now.Sub(c.lastPersist[id]) >= nodeOnlinePersistInterval {
			c.lastPersist[id] = now
			persist = now
		}
	}
	if t, ok := c.lastOnline[id]; ok && s.LastOnlineTime == nil {
		s.LastOnlineTime = timestamppb.New(t)
	}
	if s.State == v1.NodeState_NodeState_Online && s.Remote != nil {
		c.lastRemote[id] = s.Remote
	} else if r, ok := c.lastRemote[id]; ok && s.LastRemote == nil {
		s.LastRemote = r
	}
	c.status[id] = s
	return persist
}

// seedLastOnline 用库里保存的在线时间补齐内存记录（重启后首次探测前）。
func (c *nodeStatusCache) seedLastOnline(id string, t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if prev, ok := c.lastOnline[id]; !ok || prev.Before(t) {
		c.lastOnline[id] = t
	}
	if _, ok := c.lastPersist[id]; !ok {
		c.lastPersist[id] = t
	}
}

func (c *nodeStatusCache) remove(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.status, id)
	delete(c.lastRemote, id)
}

// NodeCallError 把节点调用错误转换为主控返回给浏览器的业务错误：
// 节点业务错误保留原始业务码与提示；节点鉴权失败（401）改写为 500，避免浏览器误判主控登录失效。
func NodeCallError(nodeName string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return nodeError(500, "节点 %s 响应超时", nodeName)
	}
	st, ok := status.FromError(err)
	if !ok {
		return nodeError(500, "节点 %s 调用失败：%v", nodeName, err)
	}
	switch st.Code() {
	case codes.Unavailable:
		return nodeError(500, "节点 %s 不可用：%s", nodeName, st.Message())
	case codes.DeadlineExceeded:
		return nodeError(500, "节点 %s 响应超时", nodeName)
	case codes.Canceled:
		return nodeError(500, "节点 %s 请求已取消", nodeName)
	case codes.Unimplemented:
		return nodeError(500, "节点 %s 不支持该接口（请升级节点 momoko 版本）", nodeName)
	}
	code, message := nodeErrorCode(err)
	if code == 401 {
		return nodeError(500, "节点 %s 鉴权失败：%s", nodeName, message)
	}
	return nodeError(code, "%s", message)
}

// nodeErrorCode 还原节点返回的业务码与提示。
func nodeErrorCode(err error) (int, string) {
	se := kerrors.FromError(err)
	code := int(se.Code)
	if se.Reason == NodeErrorReason {
		if raw, ok := se.Metadata["code"]; ok {
			if parsed, perr := strconv.Atoi(raw); perr == nil {
				code = parsed
			}
		}
	}
	if code <= 0 {
		code = 500
	}
	return code, se.Message
}

// classifyNodeStatus 把检测错误归类为离线或鉴权失败。
func classifyNodeStatus(nodeName string, err error) (v1.NodeState, string) {
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
			return v1.NodeState_NodeState_Offline, NodeCallError(nodeName, err).Error()
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return v1.NodeState_NodeState_Offline, NodeCallError(nodeName, err).Error()
	}
	if code, message := nodeErrorCode(err); code == 401 {
		return v1.NodeState_NodeState_Unauthorized, message
	}
	return v1.NodeState_NodeState_Offline, NodeCallError(nodeName, err).Error()
}
