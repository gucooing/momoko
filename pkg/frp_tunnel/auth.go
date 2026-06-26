package frp_tunnel

import (
	"context"
	"fmt"

	"github.com/fatedier/frp/pkg/config/types"
)

// Tunnel 是鉴权所需的隧道规则视图，由业务层通过 TunnelLookup 按需提供。
// pkg 不感知 momoko 的 ent/proto 类型，仅依赖这一最小视图。
type Tunnel struct {
	Name       string
	Credential string
	ProxyType  string   // tcp/udp/http/https/stcp/xtcp/tcpmux
	RemotePort int      // tcp/udp/tcpmux 公共端口；0 表示不校验
	AllowUsers []string // 允许的 frpc user；空=不限制
	Enabled    bool

	// 资源类限制（以 momoko 为准，客户端超出即拒绝）。加密/压缩属客户端自选优化，不在此约束。
	MaxBandwidth   string // 带宽上限/限速（如 "1MB"，空=不限）
	MaxActiveConns int    // 活跃连接数上限，0=不限
}

// TunnelLookup 由业务层实现：按 frp proxy 名称返回隧道规则；未找到返回 ok=false。
type TunnelLookup interface {
	LookupTunnel(ctx context.Context, name string) (*Tunnel, bool)
}

// authLogin 实现 frpc 登录鉴权规则：凭 common metadatas 中的 name+credential 匹配隧道规则。
func authLogin(ctx context.Context, lookup TunnelLookup, info LoginInfo) error {
	name := info.Metas["name"]
	credential := info.Metas["credential"]
	if name == "" || credential == "" {
		return fmt.Errorf("缺少隧道标识或凭证")
	}
	tn, ok := lookup.LookupTunnel(ctx, name)
	if !ok {
		return fmt.Errorf("隧道不存在或未在 momoko 创建")
	}
	if !tn.Enabled {
		return fmt.Errorf("隧道已禁用")
	}
	if tn.Credential != credential {
		return fmt.Errorf("凭证不匹配")
	}
	if !userAllowed(tn.AllowUsers, info.User) {
		return fmt.Errorf("用户不被允许")
	}
	return nil
}

// authNewProxy 实现 frpc 声明 proxy 的鉴权规则：proxy 是否与隧道规则一致。
func authNewProxy(ctx context.Context, lookup TunnelLookup, info NewProxyInfo) error {
	tn, ok := lookup.LookupTunnel(ctx, info.ProxyName)
	if !ok {
		return fmt.Errorf("隧道不存在：%s", info.ProxyName)
	}
	if !tn.Enabled {
		return fmt.Errorf("隧道已禁用")
	}
	if info.LoginMetas["credential"] != tn.Credential {
		return fmt.Errorf("凭证不匹配")
	}
	if info.ProxyType != tn.ProxyType {
		return fmt.Errorf("代理类型不匹配：声明 %s，规则 %s", info.ProxyType, tn.ProxyType)
	}
	switch tn.ProxyType {
	case "tcp", "udp", "tcpmux":
		if tn.RemotePort != 0 && info.RemotePort != tn.RemotePort {
			return fmt.Errorf("公共端口不匹配：声明 %d，规则 %d", info.RemotePort, tn.RemotePort)
		}
	}
	if err := enforceClientPolicy(tn, info); err != nil {
		return err
	}
	return nil
}

// enforceClientPolicy 以 momoko 隧道规则为准校验 frpc 声明的 proxy 配置：
// 带宽（限速）超过上限时拒绝（在 frps 插件机制内完成，不改 frps 源码）。
// 加密/压缩属客户端自选优化，不做约束。
func enforceClientPolicy(tn *Tunnel, info NewProxyInfo) error {
	if tn.MaxBandwidth != "" {
		limit, err := types.NewBandwidthQuantity(tn.MaxBandwidth)
		if err != nil {
			// 规则侧带宽上限配置非法时不放行，避免上限形同虚设。
			return fmt.Errorf("隧道带宽上限配置非法：%q", tn.MaxBandwidth)
		}
		if info.BandwidthLimit == "" {
			return fmt.Errorf("该隧道限制带宽不超过 %s，请在 frpc 配置中声明 transport.bandwidthLimit", tn.MaxBandwidth)
		}
		got, err := types.NewBandwidthQuantity(info.BandwidthLimit)
		if err != nil {
			return fmt.Errorf("客户端带宽限制声明非法：%q", info.BandwidthLimit)
		}
		if got.Bytes() > limit.Bytes() {
			return fmt.Errorf("客户端带宽 %s 超过隧道上限 %s", info.BandwidthLimit, tn.MaxBandwidth)
		}
	}
	return nil
}

// authNewUserConn 在每条用户连接建立时校验隧道的活跃连接数上限。
// activeConns 为该隧道当前活跃连接数（由 Manager 通过实时统计提供）。
// 注：frp 服务端配置无此项，只能在 momoko 的 NewUserConn 插件钩子里逐隧道限制。
func authNewUserConn(ctx context.Context, lookup TunnelLookup, activeConns int64, info NewUserConnInfo) error {
	tn, ok := lookup.LookupTunnel(ctx, info.ProxyName)
	if !ok {
		// NewProxy 阶段已校验隧道存在；此处查不到则放行，避免误伤。
		return nil
	}
	if tn.MaxActiveConns > 0 && activeConns >= int64(tn.MaxActiveConns) {
		return fmt.Errorf("隧道活跃连接数已达上限 %d", tn.MaxActiveConns)
	}
	return nil
}

func userAllowed(allowUsers []string, user string) bool {
	if len(allowUsers) == 0 {
		return true
	}
	for _, u := range allowUsers {
		if u == user {
			return true
		}
	}
	return false
}
