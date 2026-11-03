package frp_tunnel

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/fatedier/frp/pkg/util/version"
)

// ClientInfo 是最近一次以某隧道凭证登录的 frpc 客户端信息。
type ClientInfo struct {
	Hostname  string
	Os        string
	Arch      string
	Version   string
	Address   string
	LoginTime time.Time
}

// Status 是 frps 实例的运行状态。
type Status struct {
	Running   bool
	StartTime time.Time
	Version   string
	Error     string // 最近一次启动失败的原因
}

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
	TLSForce       bool
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
	startTime time.Time
	lastErr   string
	lastOpts  Options

	clientsMu sync.RWMutex
	clients   map[string]ClientInfo // 隧道名 → 最近登录的客户端
	runs      map[string]ClientInfo // frpc run id → 客户端（同一 frpc 承载多条隧道时据此归属）
}

// maxTrackedRuns 限制记忆的 frpc run id 数量，超出时整体清空重新累积。
const maxTrackedRuns = 4096

// NewManager 创建一个 frps 管理器。此时尚未启动 frps，需调用 Apply。
func NewManager(cfg ManagerConfig) *Manager {
	return &Manager{
		lookup:     cfg.Lookup,
		pluginPath: "/plugin/" + randomToken(),
		clients:    make(map[string]ClientInfo),
		runs:       make(map[string]ClientInfo),
	}
}

// Status 返回 frps 的运行状态。
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := Status{Running: m.svc != nil, Version: version.Full(), Error: m.lastErr}
	if st.Running {
		st.StartTime = m.startTime
	}
	return st
}

// Restart 按最近一次 Apply 的设置重启 frps。
func (m *Manager) Restart() error {
	m.mu.Lock()
	opts := m.lastOpts
	m.mu.Unlock()
	return m.Apply(opts)
}

// Client 返回最近一次以该隧道凭证登录的 frpc 客户端。
func (m *Manager) Client(name string) (ClientInfo, bool) {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()
	c, ok := m.clients[name]
	return c, ok
}

// Apply 按 opts（重新）装配 frps 实例及其专用鉴权回环服务。
// opts.Enable=false 时仅关闭已有实例。
func (m *Manager) Apply(opts Options) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stopLocked()
	m.lastOpts = opts
	m.lastErr = ""

	if !opts.Enable {
		return nil
	}
	err := m.startLocked(opts)
	if err != nil {
		m.lastErr = err.Error()
		return err
	}
	m.startTime = time.Now()
	return nil
}

// startLocked 装配并启动 frps，调用方须持有 m.mu。
func (m *Manager) startLocked(opts Options) error {
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
		TLSForce:       opts.TLSForce,
		PluginAddr:     "http://" + ln.Addr().String(),
		PluginPath:     m.pluginPath,
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
			if err := authLogin(ctx, m.lookup, info); err != nil {
				return err
			}
			client := ClientInfo{
				Hostname:  info.Hostname,
				Os:        info.Os,
				Arch:      info.Arch,
				Version:   info.Version,
				Address:   info.ClientAddress,
				LoginTime: time.Now(),
			}
			m.clientsMu.Lock()
			m.clients[info.Metas["name"]] = client
			if info.RunID != "" {
				if len(m.runs) >= maxTrackedRuns {
					m.runs = make(map[string]ClientInfo)
				}
				m.runs[info.RunID] = client
			}
			m.clientsMu.Unlock()
			return nil
		},
		OnNewProxy: func(ctx context.Context, info NewProxyInfo) error {
			if err := authNewProxy(ctx, m.lookup, info); err != nil {
				return err
			}
			// 一个 frpc 可以承载多条隧道（合并配置）：按 run id 把客户端归属到每条隧道。
			m.clientsMu.Lock()
			if c, ok := m.runs[info.RunID]; ok && info.RunID != "" {
				m.clients[info.ProxyName] = c
			}
			m.clientsMu.Unlock()
			return nil
		},
		OnNewUserConn: func(ctx context.Context, info NewUserConnInfo) error {
			// 当前活跃连接数取自实时统计；新连接此刻尚未计入，故用 >= 判定。
			var active int64
			if stat, ok := Stat(info.ProxyName); ok {
				active = stat.ActiveConns
			}
			return authNewUserConn(ctx, m.lookup, active, info)
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
