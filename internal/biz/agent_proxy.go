package biz

import (
	"context"
	"errors"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/net/websocket"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/agentapi"
	"momoko/pkg/agentdriver"
	"momoko/pkg/agentinstall"
	"momoko/pkg/agentproc"
	"momoko/pkg/agentproxy"
	"momoko/pkg/agentsandbox"
	"momoko/pkg/constant"
	"momoko/pkg/response"
	"momoko/pkg/shellcore"
	"momoko/pkg/sysenv"
)

// ———————————————————————————— 代理 ————————————————————————————
//
// 官方账户（运行时）和第三方接口（供应商）各自能配一个代理（只有 Linux）。配了之后，用它的一切都只经这个代理联网：
// Agent 进程连同它启动的 CLI、命令、MCP 服务器关在只有这一个出口的网络里（pkg/agentsandbox），检查、登录终端、
// 会话里的 Git 推拉也一样；Momoko 替它发的请求（测试、拉模型、查余额、下载和更新）经这个代理发出去。
// 代理读不出来（密钥换了之类）时一律当作出错，不会悄悄改成直连。

// AgentLoginWsPath 官方账户的登录终端：在服务器上跑这一系 Agent 的登录命令，接到页面的终端里（?runtime=运行时 id）。
const AgentLoginWsPath = "/api/v1/agent/login/ws"

// errProxyUnreadable 存着的代理解不开。
var errProxyUnreadable = response.BadRequest("存着的代理读不出来，重新设一次代理")

// proxyOf 存着的代理（密文）；没配为 nil。
func (uc *AgentUsecase) proxyOf(stored string) (*agentproxy.Proxy, error) {
	if stored == "" {
		return nil, nil
	}
	plain := uc.decrypt(stored)
	if plain == "" {
		return nil, errProxyUnreadable
	}
	p, err := agentproxy.Parse(plain)
	if err != nil || p == nil {
		return nil, errProxyUnreadable
	}
	return p, nil
}

// proxyPB 给能改的人看的代理：不带密码，只说有没有。
func (uc *AgentUsecase) proxyPB(stored string) *v1.AgentProxy {
	p, err := uc.proxyOf(stored)
	if err != nil || p == nil {
		return nil
	}
	return &v1.AgentProxy{Type: p.Type, Host: p.Host, Port: int32(p.Port), Username: p.Username, HasPassword: p.Password != ""}
}

// proxyViewPB 只能用的人看的代理：类型和地址。
func (uc *AgentUsecase) proxyViewPB(stored string) *v1.AgentProxy {
	out := uc.proxyPB(stored)
	if out != nil {
		out.Username, out.HasPassword = "", false
	}
	return out
}

// formProxy 表单里的代理：密码留空、用户名没变时用 stored 里的密码。type 为空返回 nil。
func (uc *AgentUsecase) formProxy(stored string, in *v1.AgentProxy) (*agentproxy.Proxy, error) {
	kind := strings.ToLower(strings.TrimSpace(in.GetType()))
	if kind == "" {
		return nil, nil
	}
	p := &agentproxy.Proxy{
		Type:     kind,
		Host:     strings.TrimSpace(in.GetHost()),
		Port:     int(in.GetPort()),
		Username: strings.TrimSpace(in.GetUsername()),
		Password: in.GetPassword(),
	}
	if p.Password == "" && p.Username != "" {
		if old, _ := uc.proxyOf(stored); old != nil && old.Username == p.Username {
			p.Password = old.Password
		}
	}
	if err := p.Validate(); err != nil {
		return nil, response.BadRequest(err.Error())
	}
	return p, nil
}

// mergeProxy 表单里的代理换成要存的密文：in 为 nil 时不改；type 为空是不走代理。只有 Linux 能配。
func (uc *AgentUsecase) mergeProxy(stored string, in *v1.AgentProxy) (string, error) {
	if in == nil {
		return stored, nil
	}
	p, err := uc.formProxy(stored, in)
	if err != nil {
		return "", err
	}
	if p == nil {
		return "", nil
	}
	if !agentsandbox.Supported() {
		return "", response.BadRequest(agentsandbox.ErrUnsupported.Error())
	}
	return uc.encrypt(p.URL()), nil
}

// draftProxy 测试、拉模型用的代理：表单里带了就用表单里的（密码留空用存着的），没带用存着的。
func (uc *AgentUsecase) draftProxy(stored string, in *v1.AgentProxy) (*agentproxy.Proxy, error) {
	if in == nil {
		return uc.proxyOf(stored)
	}
	return uc.formProxy(stored, in)
}

// proxySupport 列表里带的：能不能配代理，Linux 上现在用不了的原因。
func proxySupport() (supported bool, unavailable string) {
	if !agentsandbox.Supported() {
		return false, ""
	}
	if err := agentsandbox.Check(); err != nil {
		return true, err.Error()
	}
	return true, ""
}

// withProxyClient Momoko 替它发的请求（测试、拉模型、查余额）经 p 发出去；p 为 nil 时原样返回。
func withProxyClient(ctx context.Context, p *agentproxy.Proxy) context.Context {
	if p == nil {
		return ctx
	}
	return agentapi.WithClient(ctx, p.Client(3*time.Minute))
}

// proxied 用服务器上的登录起这个运行时（检查）：配了代理就带上，并关掉用不着的联网。
func (uc *AgentUsecase) proxied(rt *gen.AgentRuntime, launch agentdriver.Launch) (agentdriver.Launch, error) {
	p, err := uc.proxyOf(rt.Proxy)
	if err != nil || p == nil {
		return launch, err
	}
	env := map[string]string{}
	for k, v := range launch.Env {
		env[k] = v
	}
	for k, v := range uc.driverOf(string(rt.Kind)).ProxyEnv() {
		env[k] = v
	}
	launch.Env, launch.Proxy = env, p
	return launch, nil
}

// installCtx 下载、更新这个运行时用的 ctx（不受请求超时影响）：配了代理就经代理下载。
func (uc *AgentUsecase) installCtx(ctx context.Context, rt *gen.AgentRuntime) (context.Context, error) {
	bg := context.WithoutCancel(ctx)
	p, err := uc.proxyOf(rt.Proxy)
	if err != nil || p == nil {
		return bg, err
	}
	return agentinstall.WithClient(bg, p.Client(0)), nil
}

// sameProxy 存着的代理和 p 是不是同一个（都没有也算）。
func (uc *AgentUsecase) sameProxy(stored string, p *agentproxy.Proxy) bool {
	cur, err := uc.proxyOf(stored)
	if err != nil {
		return false
	}
	if cur == nil || p == nil {
		return cur == nil && p == nil
	}
	return cur.URL() == p.URL()
}

// sessionProxy 会话用的代理：用第三方接口的是接口配的，用服务器上的登录的是运行时配的。
func (uc *AgentUsecase) sessionProxy(ctx context.Context, s *gen.AgentSession) (*agentproxy.Proxy, error) {
	if s.ProviderID != "" {
		p, err := uc.getProvider(ctx, s.ProviderID)
		if err != nil {
			return nil, err
		}
		return uc.proxyOf(p.Proxy)
	}
	rt, err := uc.getRuntime(ctx, s.RuntimeID)
	if err != nil {
		return nil, err
	}
	return uc.proxyOf(rt.Proxy)
}

// restartUsing 代理改了：正在跑的、用它的会话进程按旧的网络起的，结束掉，下次发消息时按新的代理重起（接着原来的上下文）。
func (uc *AgentUsecase) restartUsing(runtimeID, providerID string) {
	for _, l := range uc.pool.Snapshot() {
		if (providerID != "" && l.providerID == providerID) || (runtimeID != "" && l.runtimeID == runtimeID && l.providerID == "") {
			go l.Shutdown()
		}
	}
}

// requireManage 官方账户（运行时）的事要 agent:manage。
func (uc *AgentUsecase) requireManage(ctx context.Context) error {
	if uc.system.Check(ctx, constant.AgentManage) != nil {
		return ErrNoPermission
	}
	return nil
}

// SetRuntimeProxy 服务器上的登录走的代理。改了就结束用它的会话进程，下次按新的起。
func (uc *AgentUsecase) SetRuntimeProxy(ctx context.Context, id string, in *v1.AgentProxy) (*v1.AgentRuntime, error) {
	rt, err := uc.getRuntime(ctx, id)
	if err != nil {
		return nil, err
	}
	if in == nil {
		in = &v1.AgentProxy{}
	}
	stored, err := uc.mergeProxy(rt.Proxy, in)
	if err != nil {
		return nil, err
	}
	if stored != rt.Proxy {
		before := uc.proxyKey(rt.Proxy)
		rt.Proxy = stored
		if err := uc.repo.SaveAgentRuntime(ctx, rt); err != nil {
			return nil, ErrSystem(err)
		}
		if before != uc.proxyKey(stored) {
			uc.restartUsing(rt.ID, "")
		}
		uc.configChanged()
	}
	return uc.runtimeOut(ctx, rt, uc.activeByRuntime())
}

// proxyKey 比较两份密文是不是同一个代理（同一个代理每次加密出来的密文不一样）。
func (uc *AgentUsecase) proxyKey(stored string) string {
	if stored == "" {
		return ""
	}
	return uc.decrypt(stored)
}

// 检查代理时连去哪里：这一系 Agent 的官方地址。
var proxyProbeTargets = map[string]string{
	"claude": "api.anthropic.com:443",
	"codex":  "chatgpt.com:443",
}

// TestProxy 经代理连到要用的那一家，做一次 TLS 握手。连不上是代理那边的状况，放在 error 里返回，不算请求出错。
func (uc *AgentUsecase) TestProxy(ctx context.Context, req *v1.TestAgentProxyRequest) (*v1.TestAgentProxyResponse, error) {
	var stored, target string
	switch {
	case req.GetRuntimeId() != "":
		if err := uc.requireManage(ctx); err != nil {
			return nil, err
		}
		rt, err := uc.getRuntime(ctx, req.GetRuntimeId())
		if err != nil {
			return nil, err
		}
		stored = rt.Proxy
		if target = proxyProbeTargets[string(rt.Kind)]; target == "" {
			target = "www.google.com:443"
		}
	default:
		vendor, baseURL := req.GetVendor(), strings.TrimSpace(req.GetBaseUrl())
		if req.GetProviderId() != "" {
			p, err := uc.editableProvider(ctx, req.GetProviderId())
			if err != nil {
				return nil, err
			}
			stored, vendor = p.Proxy, p.Vendor
			if baseURL == "" {
				baseURL = p.BaseURL
			}
		}
		v, ok := uc.vendorOf(vendor)
		if !ok {
			return nil, errUnknownVendor
		}
		if v.BaseURL() != "" {
			baseURL = v.BaseURL()
		}
		var err error
		if target, err = hostPort(baseURL); err != nil {
			return nil, err
		}
	}
	p, err := uc.draftProxy(stored, req.GetProxy())
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, response.BadRequest("先填代理")
	}
	bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), 40*time.Second)
	defer cancel()
	resp := &v1.TestAgentProxyResponse{Target: target}
	took, err := p.Check(bg, target)
	if err != nil {
		resp.Error = err.Error()
		return resp, nil
	}
	resp.Ok, resp.LatencyMs = true, took.Milliseconds()
	return resp, nil
}

// hostPort 接口地址的「主机:端口」（没写端口按 https 的 443）。
func hostPort(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" {
		return "", response.BadRequest("先填接口地址")
	}
	port := u.Port()
	if port == "" {
		port = "443"
	}
	return net.JoinHostPort(u.Hostname(), port), nil
}

// ———————————————————————————— 登录终端 ————————————————————————————

// loginLaunch 这个运行时在终端里登录用的命令；登录不了时 ok 为 false。
func (uc *AgentUsecase) loginLaunch(rt *gen.AgentRuntime) (agentdriver.Launch, agentdriver.Driver, bool) {
	d := uc.driverOf(string(rt.Kind))
	if builtin, ok := uc.builtinDriver(rt.ID); ok {
		d = builtin
	}
	launch, err := uc.runtimeLaunch(rt)
	if err != nil {
		return agentdriver.Launch{}, d, false
	}
	login, ok := d.Login(launch.Command)
	return login, d, ok
}

// ServeLogin 在服务器上跑这个运行时的登录命令，接到页面的终端里；配了代理就在只走代理的网络里跑。
// 命令结束后自动检查一次登录。
func (uc *AgentUsecase) ServeLogin(conn *websocket.Conn, runtimeID string) error {
	ctx := context.WithoutCancel(conn.Request().Context())
	say := func(text string) { _ = websocket.Message.Send(conn, "\x1b[31m[system] "+text+"\x1b[0m\r\n") }
	rt, err := uc.getRuntime(ctx, runtimeID)
	if err != nil {
		say(err.Error())
		return err
	}
	login, d, ok := uc.loginLaunch(rt)
	if !ok {
		err := errors.New(rt.Name + " 没有能在终端里跑的登录命令")
		if _, lerr := uc.runtimeLaunch(rt); lerr != nil {
			err = lerr
		}
		say(err.Error())
		return err
	}
	proxy, err := uc.proxyOf(rt.Proxy)
	if err != nil {
		say(err.Error())
		return err
	}
	env := map[string]string{}
	for k, v := range login.Env {
		env[k] = v
	}
	if proxy != nil {
		for k, v := range d.ProxyEnv() {
			env[k] = v
		}
		env = agentproc.WithProxyEnv(env)
	}
	// 用注册表现在的环境变量（Momoko 启动后装的工具也找得到）
	base := append(sysenv.Environ(), shellcore.TermEnv()...)
	dir, _ := os.UserHomeDir()
	spec := shellcore.Spec{Path: login.Command, Args: login.Args, Dir: dir, Env: agentproc.MergeEnv(base, env), Proxy: proxy}
	if proxy != nil {
		_ = websocket.Message.Send(conn, "\x1b[2m[system] 经代理 "+proxy.String()+" 联网，别的路都断开了\x1b[0m\r\n")
	}
	farewell := "\r\n\x1b[2m[system] 登录命令结束了，正在检查 " + rt.Name + " 的登录…\x1b[0m\r\n"
	serveErr := shellcore.ServeCommand(conn, spec, farewell)
	// 不管登录成没成，都按现在的状态检查一次（页面收到配置变了会刷新）
	if latest, err := uc.getRuntime(ctx, runtimeID); err == nil {
		go func() { _, _ = uc.detectRuntime(ctx, latest) }()
	}
	return serveErr
}
