package biz

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/constant"
	"momoko/pkg/nodeauth"
	"momoko/pkg/response"
	"momoko/pkg/utils"
)

const (
	nodeProbeTimeout  = 5 * time.Second
	nodeProbeInterval = 30 * time.Second
)

type NodeRepo interface {
	ListNodes(ctx context.Context, keywords *string) ([]*NodeConfig, error)
	GetNode(ctx context.Context, id string) (*NodeConfig, error)
	CreateNode(ctx context.Context, node *NodeConfig) (*NodeConfig, error)
	UpdateNode(ctx context.Context, node *NodeConfig) (*NodeConfig, error)
	DeleteNode(ctx context.Context, id string) error
	// TouchNodeOnline 记录节点最近一次在线时间（不改动 update_time）
	TouchNodeOnline(ctx context.Context, id string, at time.Time) error
}

func nodeError(code int, format string, args ...any) error {
	return response.BadRequest(code, fmt.Sprintf(format, args...))
}

// validateNodeAddress 要求 host:port，端口合法。
func validateNodeAddress(address string) (string, error) {
	address = strings.TrimSpace(address)
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" {
		return "", ErrNodeAddressInvalid
	}
	if p, err := strconv.Atoi(port); err != nil || p <= 0 || p > 65535 {
		return "", ErrNodeAddressInvalid
	}
	return address, nil
}

func (n *NodeUsecase) ListNodes(ctx context.Context, req *v1.ListNodesRequest) ([]*v1.NodeInfo, error) {
	nodes, err := n.nodes.ListNodes(ctx, req.Keywords)
	if err != nil {
		return nil, ErrSystem(err)
	}
	infos := make([]*v1.NodeInfo, 0, len(nodes))
	for _, node := range nodes {
		infos = append(infos, n.toNodeInfo(node))
	}
	return infos, nil
}

func (n *NodeUsecase) CreateNode(ctx context.Context, req *v1.CreateNodeRequest) (*v1.NodeInfo, error) {
	cfg := &NodeConfig{
		Name:          strings.TrimSpace(req.Name),
		APIKey:        strings.TrimSpace(req.ApiKey),
		TLS:           req.Tls,
		TLSSkipVerify: req.TlsSkipVerify,
		Remark:        req.Remark,
	}
	if cfg.Name == "" {
		return nil, ErrNodeNameEmpty
	}
	if cfg.APIKey == "" {
		return nil, ErrNodeAPIKeyEmpty
	}
	address, err := validateNodeAddress(req.Address)
	if err != nil {
		return nil, err
	}
	cfg.Address = address
	cfg.KeyID = nodeauth.KeyID(cfg.APIKey)

	saved, err := n.nodes.CreateNode(ctx, cfg)
	if err != nil {
		return nil, ErrSystem(err)
	}
	n.probeAsync(saved)
	return n.toNodeInfo(saved), nil
}

func (n *NodeUsecase) UpdateNode(ctx context.Context, req *v1.UpdateNodeRequest) (*v1.NodeInfo, error) {
	cfg, err := n.loadNode(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return nil, ErrNodeNameEmpty
		}
		cfg.Name = strings.TrimSpace(*req.Name)
	}
	if req.Address != nil {
		address, err := validateNodeAddress(*req.Address)
		if err != nil {
			return nil, err
		}
		cfg.Address = address
	}
	if req.ApiKey != nil && strings.TrimSpace(*req.ApiKey) != "" {
		cfg.APIKey = strings.TrimSpace(*req.ApiKey)
		cfg.KeyID = nodeauth.KeyID(cfg.APIKey)
	}
	if cfg.APIKey == "" {
		return nil, ErrNodeKeyBroken
	}
	if req.Tls != nil {
		cfg.TLS = *req.Tls
	}
	if req.TlsSkipVerify != nil {
		cfg.TLSSkipVerify = *req.TlsSkipVerify
	}
	if req.Remark != nil {
		cfg.Remark = *req.Remark
	}

	saved, err := n.nodes.UpdateNode(ctx, cfg)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrNodeNotFound
		}
		return nil, ErrSystem(err)
	}
	n.clients.remove(saved.ID)
	n.status.remove(saved.ID)
	n.probeAsync(saved)
	return n.toNodeInfo(saved), nil
}

func (n *NodeUsecase) DeleteNode(ctx context.Context, req *v1.DeleteNodeRequest) error {
	if err := n.nodes.DeleteNode(ctx, req.Id); err != nil {
		if gen.IsNotFound(err) {
			return ErrNodeNotFound
		}
		return ErrSystem(err)
	}
	n.clients.remove(req.Id)
	n.status.remove(req.Id)
	return nil
}

// TestNode 测试连接；传 id 时以已保存配置为底、用请求中给出的字段覆盖（便于编辑时先测后存）。
func (n *NodeUsecase) TestNode(ctx context.Context, req *v1.TestNodeRequest) (*v1.NodeStatus, error) {
	cfg := &NodeConfig{Name: "test"}
	if req.Id != nil && *req.Id != "" {
		saved, err := n.loadNode(ctx, *req.Id)
		if err != nil {
			return nil, err
		}
		cfg = saved
	}
	// 只有原样测试已保存配置时，结果才回写状态缓存。
	overridden := false
	if req.Address != nil && strings.TrimSpace(*req.Address) != "" {
		cfg.Address = strings.TrimSpace(*req.Address)
		overridden = true
	}
	if req.ApiKey != nil && strings.TrimSpace(*req.ApiKey) != "" {
		cfg.APIKey = strings.TrimSpace(*req.ApiKey)
		overridden = true
	}
	if req.Tls != nil {
		cfg.TLS = *req.Tls
		overridden = true
	}
	if req.TlsSkipVerify != nil {
		cfg.TLSSkipVerify = *req.TlsSkipVerify
		overridden = true
	}
	address, err := validateNodeAddress(cfg.Address)
	if err != nil {
		return nil, err
	}
	cfg.Address = address
	if cfg.APIKey == "" {
		return nil, ErrNodeAPIKeyEmpty
	}

	// 测试用独立连接，不污染缓存中的正式连接。
	client, err := newNodeClient(cfg)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	result := n.probe(ctx, client)
	if cfg.ID != "" && !overridden {
		n.setStatus(cfg.ID, result)
	}
	return result, nil
}

// GetNodeContext 实时检测节点，返回当前用户在该节点上的生效权限：
// 节点授予给 API Key 的权限中，RBAC 类权限还需当前用户在主控同样拥有（主控转发前也会按自身 RBAC 校验）。
func (n *NodeUsecase) GetNodeContext(ctx context.Context, req *v1.GetNodeContextRequest) (*v1.NodeInfo, []string, error) {
	client, err := n.NodeClient(ctx, req.Id)
	if err != nil {
		return nil, nil, err
	}
	result := n.probe(ctx, client)
	n.setStatus(client.Config.ID, result)
	info := n.toNodeInfo(client.Config)

	permissions := make([]string, 0)
	if result.State == v1.NodeState_NodeState_Online && result.Remote != nil {
		for _, perm := range result.Remote.Permissions {
			code := constant.Permissions(perm)
			if _, known := nodePermissionIndex[code]; !known {
				continue
			}
			if IsRBACNodePermission(code) {
				if err := n.system.Check(ctx, code); err != nil {
					continue
				}
			}
			permissions = append(permissions, perm)
		}
	}
	return info, permissions, nil
}

// NodeClient 返回节点的正式连接（转发业务请求用）。
func (n *NodeUsecase) NodeClient(ctx context.Context, id string) (*NodeClient, error) {
	if client, ok := n.clients.cached(id); ok {
		return client, nil
	}
	cfg, err := n.loadNode(ctx, id)
	if err != nil {
		return nil, err
	}
	return n.clients.get(cfg)
}

func (n *NodeUsecase) loadNode(ctx context.Context, id string) (*NodeConfig, error) {
	cfg, err := n.nodes.GetNode(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrNodeNotFound
		}
		return nil, ErrSystem(err)
	}
	return cfg, nil
}

func (n *NodeUsecase) probe(ctx context.Context, client *NodeClient) *v1.NodeStatus {
	ctx, cancel := context.WithTimeout(ctx, nodeProbeTimeout)
	defer cancel()
	start := time.Now()
	info, err := client.AgentInfo(ctx)
	result := &v1.NodeStatus{
		CheckTime: timestamppb.Now(),
		LatencyMs: time.Since(start).Milliseconds(),
	}
	if err != nil {
		result.State, result.Message = classifyNodeStatus(client.Config.Name, err)
		return result
	}
	result.State = v1.NodeState_NodeState_Online
	result.Remote = info
	return result
}

func (n *NodeUsecase) probeAsync(cfg *NodeConfig) {
	go func() {
		client, err := n.clients.get(cfg)
		if err != nil {
			n.setStatus(cfg.ID, &v1.NodeStatus{
				State:     v1.NodeState_NodeState_Offline,
				Message:   err.Error(),
				CheckTime: timestamppb.Now(),
			})
			return
		}
		n.setStatus(cfg.ID, n.probe(context.Background(), client))
	}()
}

// startProber 定时检测全部节点，刷新列表与切换器里的在线状态。
func (n *NodeUsecase) startProber() func() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		timer := time.NewTimer(3 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}
			nodes, err := n.nodes.ListNodes(ctx, nil)
			if err == nil {
				for _, cfg := range nodes {
					client, err := n.clients.get(cfg)
					if err != nil {
						n.setStatus(cfg.ID, &v1.NodeStatus{
							State:     v1.NodeState_NodeState_Offline,
							Message:   err.Error(),
							CheckTime: timestamppb.Now(),
						})
						continue
					}
					n.seedLastOnline(cfg)
					n.setStatus(cfg.ID, n.probe(ctx, client))
				}
			}
			timer.Reset(nodeProbeInterval)
		}
	}()
	return cancel
}

// setStatus 写入检测结果；节点在线时按节流间隔把在线时间写库，供重启后显示“最后在线”。
func (n *NodeUsecase) setStatus(id string, s *v1.NodeStatus) {
	if at := n.status.set(id, s); !at.IsZero() {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = n.nodes.TouchNodeOnline(ctx, id, at)
		}()
	}
}

func (n *NodeUsecase) seedLastOnline(cfg *NodeConfig) {
	if cfg.LastOnlineAt != nil {
		n.status.seedLastOnline(cfg.ID, *cfg.LastOnlineAt)
	}
}

func (n *NodeUsecase) toNodeInfo(cfg *NodeConfig) *v1.NodeInfo {
	n.seedLastOnline(cfg)
	status := n.status.get(cfg.ID)
	if status.LastOnlineTime == nil && cfg.LastOnlineAt != nil {
		status = proto.Clone(status).(*v1.NodeStatus)
		status.LastOnlineTime = timestamppb.New(*cfg.LastOnlineAt)
	}
	apiKey := "******"
	if cfg.APIKey != "" {
		apiKey = utils.PrivacyString(cfg.APIKey)
	}
	return &v1.NodeInfo{
		Id:            cfg.ID,
		Name:          cfg.Name,
		Address:       cfg.Address,
		Tls:           cfg.TLS,
		TlsSkipVerify: cfg.TLSSkipVerify,
		Remark:        cfg.Remark,
		ApiKey:        apiKey,
		KeyId:         cfg.KeyID,
		Status:        status,
		CreateTime:    timestamppb.New(cfg.CreateTime),
		UpdateTime:    timestamppb.New(cfg.UpdateTime),
	}
}
