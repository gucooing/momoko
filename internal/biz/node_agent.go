package biz

import (
	"context"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	ghost "github.com/shirou/gopsutil/v4/host"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/user"
	"momoko/pkg/auth"
	"momoko/pkg/constant"
	"momoko/pkg/nodeauth"
	"momoko/pkg/version"
)

const nodeHostInfoTTL = 10 * time.Minute

// NodePrincipal 是经 API Key 时间戳签名鉴权的远程主控调用主体，以 API Key 所属用户的身份执行。
type NodePrincipal struct {
	APIKeyID   string
	APIKeyName string
	KeyID      string
	UserID     string
	Username   string
	ExpiresAt  *time.Time
	// Operator 主控侧的操作者，仅用于审计。
	Operator string
	scopes   map[constant.Permissions]struct{}
}

// Auth 返回以 API Key 所属用户身份执行的鉴权上下文。
func (p *NodePrincipal) Auth() *auth.Auth {
	return &auth.Auth{UserID: p.UserID, Type: auth.AuthTypeApiKey}
}

// Granted 判断授予范围是否包含该权限。
func (p *NodePrincipal) Granted(perm constant.Permissions) bool {
	_, ok := p.scopes[perm]
	return ok
}

type nodePrincipalKey struct{}

func NewNodePrincipalContext(ctx context.Context, p *NodePrincipal) context.Context {
	return context.WithValue(ctx, nodePrincipalKey{}, p)
}

func NodePrincipalFromContext(ctx context.Context) (*NodePrincipal, bool) {
	p, ok := ctx.Value(nodePrincipalKey{}).(*NodePrincipal)
	return p, ok
}

// NodeAgentUsecase 是节点侧对远程主控调用的鉴权与授权。
type NodeAgentUsecase struct {
	keys   APIKeyRepo
	users  UserRepo
	system *SystemUsecase
	replay *nodeauth.ReplayGuard
}

func NewNodeAgentUsecase(keys APIKeyRepo, users UserRepo, system *SystemUsecase) *NodeAgentUsecase {
	return &NodeAgentUsecase{
		keys:   keys,
		users:  users,
		system: system,
		replay: nodeauth.NewReplayGuard(),
	}
}

// Authenticate 校验签名 metadata：时间窗口 → 按 key-id 取种子验签 → 过期 → 防重放 → 所属用户状态。
// method 必须取自服务端实际路由到的 gRPC 方法，签名与之绑定。
func (u *NodeAgentUsecase) Authenticate(ctx context.Context, md metadata.MD, method string) (*NodePrincipal, error) {
	now := time.Now()
	signed, err := nodeauth.ParseMetadata(md)
	if err != nil {
		return nil, ErrNodeAuthFailed
	}
	if err := signed.CheckTime(now); err != nil {
		return nil, ErrNodeClockSkew(now)
	}
	key, err := u.keys.GetAPIKeyByKeyID(ctx, signed.KeyID)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrNodeAuthFailed
		}
		return nil, ErrSystem(err)
	}
	if err := signed.Verify(key.APIKey, method, now); err != nil {
		return nil, ErrNodeAuthFailed
	}
	if key.ExpiresAt != nil && !now.Before(*key.ExpiresAt) {
		return nil, ErrNodeAPIKeyExpired
	}
	if err := u.replay.Use(signed.KeyID, signed.Nonce, now); err != nil {
		return nil, ErrNodeRequestReplay
	}
	owner, err := u.users.FindByID(ctx, key.UserID)
	if err != nil || owner.Status != user.StatusActive {
		return nil, ErrNodeAPIKeyOwnerInactive
	}
	u.touchAPIKey(ctx, key, now)

	scopes := make(map[constant.Permissions]struct{}, len(key.Permissions))
	for _, perm := range key.Permissions {
		scopes[constant.Permissions(perm)] = struct{}{}
	}
	return &NodePrincipal{
		APIKeyID:   key.ID,
		APIKeyName: key.Name,
		KeyID:      key.KeyID,
		UserID:     owner.ID,
		Username:   owner.Username,
		ExpiresAt:  key.ExpiresAt,
		Operator:   signed.Operator(),
		scopes:     scopes,
	}, nil
}

// apiKeyTouchInterval 最近使用时间写库的最小间隔（来源 IP 变化时立即写）。
const apiKeyTouchInterval = time.Minute

// touchAPIKey 异步记录 API Key 最近一次被调用的时间与来源 IP（节流）。
func (u *NodeAgentUsecase) touchAPIKey(ctx context.Context, key *gen.UserAPIKey, now time.Time) {
	ip := ""
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		ip = p.Addr.String()
		if host, _, err := net.SplitHostPort(ip); err == nil {
			ip = host
		}
	}
	if key.LastUsedAt != nil && now.Sub(*key.LastUsedAt) < apiKeyTouchInterval && key.LastUsedIP == ip {
		return
	}
	id := key.ID
	go func() {
		tctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = u.keys.TouchAPIKeyUsed(tctx, id, now, ip)
	}()
}

// Authorize 校验主体能否执行需要 perm 的操作：授予范围必须包含 perm；
// RBAC 类权限还要求 API Key 所属用户在本节点拥有它，API Key 不会超出所属用户的权限。
func (u *NodeAgentUsecase) Authorize(ctx context.Context, p *NodePrincipal, perm constant.Permissions) error {
	if !p.Granted(perm) {
		return ErrNodePermissionDenied(string(perm))
	}
	if IsRBACNodePermission(perm) {
		if err := u.system.Check(auth.NewContext(ctx, p.Auth()), perm); err != nil {
			return ErrNodePermissionDenied(string(perm))
		}
	}
	return nil
}

// EffectivePermissions 返回主体在本节点的生效权限。
func (u *NodeAgentUsecase) EffectivePermissions(ctx context.Context, p *NodePrincipal) []string {
	permissions := make([]string, 0, len(p.scopes))
	for _, spec := range nodePermissionSpecs {
		if u.Authorize(ctx, p, spec.code) == nil {
			permissions = append(permissions, string(spec.code))
		}
	}
	return permissions
}

// AgentInfo 汇总节点信息与生效权限，主控据此展示节点并过滤页面与操作。
func (u *NodeAgentUsecase) AgentInfo(ctx context.Context, p *NodePrincipal) *v1.NodeRemoteInfo {
	info := LocalNodeInfo(ctx)
	info.ApiKeyName = p.APIKeyName
	info.UserId = p.UserID
	info.Username = p.Username
	info.Permissions = u.EffectivePermissions(ctx, p)
	if p.ExpiresAt != nil {
		info.ExpiresAt = timestamppb.New(*p.ExpiresAt)
	}
	return info
}

// LocalNodeInfo 返回本机的主机名 / 系统 / 架构 / 版本（节点自报信息与主控节点列表的本机卡片共用）。
func LocalNodeInfo(ctx context.Context) *v1.NodeRemoteInfo {
	info := &v1.NodeRemoteInfo{
		Os:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		Version:    version.Current(),
		ServerTime: timestamppb.Now(),
	}
	if host := localHost.get(ctx); host != nil {
		info.Hostname = host.Hostname
		info.Platform = strings.TrimSpace(host.Platform + " " + host.PlatformVersion)
		if host.KernelVersion != "" {
			info.Platform = strings.TrimSpace(info.Platform + " · " + host.KernelVersion)
		}
	} else {
		info.Hostname, _ = os.Hostname()
	}
	return info
}

// localHost 缓存本机主机信息（gopsutil 读取较慢）。
var localHost = &hostInfoCache{}

type hostInfoCache struct {
	mu   sync.Mutex
	info *ghost.InfoStat
	at   time.Time
}

func (c *hostInfoCache) get(ctx context.Context) *ghost.InfoStat {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.info != nil && time.Since(c.at) < nodeHostInfoTTL {
		return c.info
	}
	host, err := ghost.InfoWithContext(ctx)
	if err != nil {
		return c.info
	}
	c.info = host
	c.at = time.Now()
	return host
}

// NodeTrace 记录一次主控请求被转发到的节点，供操作审计写入。
type NodeTrace struct {
	NodeID   string
	NodeName string
}

type nodeTraceKey struct{}

// WithNodeTrace 在请求上下文中预留节点转发记录，由转发中间件填写。
func WithNodeTrace(ctx context.Context) (context.Context, *NodeTrace) {
	trace := &NodeTrace{}
	return context.WithValue(ctx, nodeTraceKey{}, trace), trace
}

// MarkNodeTrace 标记当前请求已转发到节点。
func MarkNodeTrace(ctx context.Context, cfg *NodeConfig) {
	if trace, ok := ctx.Value(nodeTraceKey{}).(*NodeTrace); ok && cfg != nil {
		trace.NodeID = cfg.ID
		trace.NodeName = cfg.Name
	}
}

// NodeTraceFromContext 返回请求被转发到的节点（未转发时 NodeID 为空）。
func NodeTraceFromContext(ctx context.Context) (*NodeTrace, bool) {
	trace, ok := ctx.Value(nodeTraceKey{}).(*NodeTrace)
	return trace, ok
}
