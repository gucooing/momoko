package frp_tunnel

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"sync"
)

// Options 是一个 frps 实例的网络监听设置（不含插件回调，由 Manager 注入）。
type Options struct {
	Enable         bool
	BindAddr       string
	BindPort       int
	VhostHTTPPort  int
	VhostHTTPSPort int
	KCPBindPort    int
	QUICBindPort   int
	SubdomainHost  string

	// 优化/调优项（透传至 frps，零值沿用 frps 默认）。
	TLSForce          bool
	MaxPoolCount      int
	MaxPortsPerClient int
	HeartbeatTimeout  int
	TCPMux            bool
	UDPPacketSize     int
}

// ManagerConfig 装配 Manager 所需信息。
type ManagerConfig struct {
	// Lookup 由业务层实现，供鉴权按名称查隧道规则。
	Lookup TunnelLookup
}

// Manager 封装进程内 frps 实例的生命周期与鉴权接入，供业务层调用。
//
// 鉴权回调走一个专用的本地回环（127.0.0.1）HTTP 服务，仅同机 frps 可达，
// 不经 momoko 主 HTTP 服务，也不暴露到任何对外路由。frp 相关机制全部内聚于此包。
type Manager struct {
	lookup     TunnelLookup
	pluginPath string // 随机 token 路径，进一步防止同机其它进程调用

	mu        sync.Mutex
	svc       *Service
	pluginSrv *http.Server
}

// NewManager 创建一个 frps 管理器。此时尚未启动 frps，需调用 Apply。
func NewManager(cfg ManagerConfig) *Manager {
	return &Manager{
		lookup:     cfg.Lookup,
		pluginPath: "/plugin/" + randomToken(),
	}
}

// Apply 按 opts（重新）装配 frps 实例及其专用鉴权回环服务。
// opts.Enable=false 时仅关闭已有实例。
func (m *Manager) Apply(opts Options) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stopLocked()

	if !opts.Enable {
		return nil
	}

	// 专用本地回环鉴权服务：端口随机、仅监听 127.0.0.1，仅同机 frps 可达。
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc(m.pluginPath, PluginHandler(m.pluginHooks()))
	pluginSrv := &http.Server{Handler: mux}
	go func() { _ = pluginSrv.Serve(ln) }()

	svc, err := NewService(Config{
		BindAddr:       orDefault(opts.BindAddr, "0.0.0.0"),
		BindPort:       opts.BindPort,
		VhostHTTPPort:  opts.VhostHTTPPort,
		VhostHTTPSPort: opts.VhostHTTPSPort,
		KCPBindPort:    opts.KCPBindPort,
		QUICBindPort:   opts.QUICBindPort,
		SubdomainHost:  opts.SubdomainHost,

		TLSForce:          opts.TLSForce,
		MaxPoolCount:      opts.MaxPoolCount,
		MaxPortsPerClient: opts.MaxPortsPerClient,
		HeartbeatTimeout:  opts.HeartbeatTimeout,
		TCPMux:            opts.TCPMux,
		UDPPacketSize:     opts.UDPPacketSize,

		PluginAddr: "http://" + ln.Addr().String(),
		PluginPath: m.pluginPath,
	})
	if err != nil {
		_ = pluginSrv.Close()
		return err
	}
	m.svc = svc
	m.pluginSrv = pluginSrv
	return nil
}

// Close 停止 frps 实例与鉴权回环服务。
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopLocked()
}

// Stat 返回指定 proxy 名称的实时统计快照。
func (m *Manager) Stat(name string) (ProxyStat, bool) { return Stat(name) }

func (m *Manager) pluginHooks() PluginHooks {
	return PluginHooks{
		OnLogin: func(ctx context.Context, info LoginInfo) error {
			return authLogin(ctx, m.lookup, info)
		},
		OnNewProxy: func(ctx context.Context, info NewProxyInfo) error {
			return authNewProxy(ctx, m.lookup, info)
		},
	}
}

// stopLocked 关闭 frps 与鉴权服务，调用方须持有 m.mu。
func (m *Manager) stopLocked() {
	if m.svc != nil {
		m.svc.Close()
		m.svc = nil
	}
	if m.pluginSrv != nil {
		_ = m.pluginSrv.Close()
		m.pluginSrv = nil
	}
}

func randomToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "momoko-tunnel-plugin"
	}
	return hex.EncodeToString(b)
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
