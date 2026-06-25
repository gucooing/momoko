package frp_tunnel

import (
	"context"
	"fmt"
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
