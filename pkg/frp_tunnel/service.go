// Package frp_tunnel 在 momoko 进程内嵌入 frp 服务端（frps），用于「内网穿透」工具。
//
// frps 以被动模式运行：隧道监听由远端官方 frpc 声明 NewProxy 后才开启。momoko 仅接管
// 认证（通过 frp 服务端 HTTP 插件）与统计（通过 pkg/metrics/mem 进程内读取），不开启
// frp 自带的 Web 面板。每条隧道使用独立凭证，frps 自身 Auth 中性化（空 token）。
package frp_tunnel

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	metrics "github.com/fatedier/frp/pkg/metrics"
	pserver "github.com/fatedier/frp/pkg/plugin/server"
	"github.com/fatedier/frp/server"
)

// PluginOps 是 momoko 插件关注的 frp 服务端操作集合。
var PluginOps = []string{pserver.OpLogin, pserver.OpNewProxy, pserver.OpNewUserConn}

// Config 是 momoko 装配一个 frps 实例所需的配置。
type Config struct {
	BindAddr       string // frps 监听地址，默认 0.0.0.0
	BindPort       int    // frpc 连接端口（frps 主端口），默认 7000
	VhostHTTPPort  int    // http 代理 vhost 端口，0=不监听
	VhostHTTPSPort int    // https 代理 vhost 端口，0=不监听
	KCPBindPort    int    // kcp 传输端口，0=不监听
	QUICBindPort   int    // quic 传输端口，0=不监听
	SubdomainHost  string // http/https subdomain 根域
	TLSForce       bool   // 只接受 TLS 加密的 frpc 连接

	// PluginAddr 是 momoko 自身 HTTP 服务的基址（如 http://127.0.0.1:22633），
	// frps 在 Login/NewProxy 时回调该地址下的插件端点。
	PluginAddr string
	// PluginPath 是插件端点的完整路径（含防外部调用的 path token）。
	PluginPath string
}

// Service 封装一个进程内 frps 实例的生命周期。
type Service struct {
	svr    *server.Service
	cancel context.CancelFunc
}

// enableMemOnce 确保进程内只向 frp 的聚合统计注册一次内存后端，避免热重启时重复计数。
var enableMemOnce sync.Once

// NewService 按 cfg 装配并启动一个 frps 实例。返回的 Service 需在退出时调用 Close。
func NewService(cfg Config) (*Service, error) {
	// frp 的 server.NewService 在中途失败时不会释放已经打开的监听，端口会一直被占到进程退出。
	// 先逐个试探端口是否可用，冲突时直接报错，避免留下半开的监听。
	if err := checkPortsFree(cfg); err != nil {
		return nil, err
	}
	serverCfg := buildServerConfig(cfg)
	if err := serverCfg.Complete(); err != nil {
		return nil, fmt.Errorf("complete frps config: %w", err)
	}

	// frps 仅在开启 WebServer 时才自动启用内存统计；本实现不开面板，需显式启用，
	// 否则 mem.StatsCollector 读到的全是 0。
	enableMemOnce.Do(metrics.EnableMem)
	// frp 日志改写到进程内环形缓冲（供「frps 日志」查看），警告及以上同步打到 momoko 日志。
	installLogCapture()

	svr, err := server.NewService(serverCfg)
	if err != nil {
		return nil, fmt.Errorf("new frps service: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go svr.Run(ctx)

	return &Service{svr: svr, cancel: cancel}, nil
}

// Close 停止 frps 实例并释放其监听端口。可安全地对 nil 调用。
func (s *Service) Close() {
	if s == nil {
		return
	}
	if s.cancel != nil {
		s.cancel()
	}
	if s.svr != nil {
		_ = s.svr.Close()
	}
}

// checkPortsFree 试探 frps 将要监听的 TCP/UDP 端口是否空闲（试探后立即关闭）。
func checkPortsFree(cfg Config) error {
	host := cfg.BindAddr
	tcpPorts := []int{cfg.BindPort}
	for _, p := range []int{cfg.VhostHTTPPort, cfg.VhostHTTPSPort} {
		if p > 0 && p != cfg.BindPort {
			tcpPorts = append(tcpPorts, p)
		}
	}
	for _, p := range tcpPorts {
		if p <= 0 {
			continue
		}
		ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(p)))
		if err != nil {
			return fmt.Errorf("TCP 端口 %d 不可用: %w", p, err)
		}
		_ = ln.Close()
	}
	for _, p := range []int{cfg.KCPBindPort, cfg.QUICBindPort} {
		if p <= 0 {
			continue
		}
		pc, err := net.ListenPacket("udp", net.JoinHostPort(host, strconv.Itoa(p)))
		if err != nil {
			return fmt.Errorf("UDP 端口 %d 不可用: %w", p, err)
		}
		_ = pc.Close()
	}
	return nil
}

func buildServerConfig(cfg Config) *v1.ServerConfig {
	serverCfg := &v1.ServerConfig{
		BindAddr:       cfg.BindAddr,
		BindPort:       cfg.BindPort,
		VhostHTTPPort:  cfg.VhostHTTPPort,
		VhostHTTPSPort: cfg.VhostHTTPSPort,
		KCPBindPort:    cfg.KCPBindPort,
		QUICBindPort:   cfg.QUICBindPort,
		SubDomainHost:  cfg.SubdomainHost,
	}
	serverCfg.Transport.TLS.Force = cfg.TLSForce

	// Auth 中性化：method=token + 空 token，frps 协议握手通过但不做统一认证，
	// 真正的认证权威是 momoko 插件（按每条隧道独立 credential 校验）。
	serverCfg.Auth.Method = "token"
	serverCfg.Auth.Token = ""

	// 降低 frps 自身日志噪音（momoko 通过插件与统计观测）。
	serverCfg.Log.To = "console"
	serverCfg.Log.Level = "warn"

	if cfg.PluginAddr != "" && cfg.PluginPath != "" {
		serverCfg.HTTPPlugins = []v1.HTTPPluginOptions{
			{
				Name: "momoko-tunnel-auth",
				Addr: cfg.PluginAddr,
				Path: cfg.PluginPath,
				Ops:  PluginOps,
			},
		}
	}

	return serverCfg
}
