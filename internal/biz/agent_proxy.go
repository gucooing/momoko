package biz

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/websocket"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
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
// 官方账户（运行时）和第三方接口（供应商）各自能配一个代理（只有 Linux），连同沙箱里的时区和语言。
// 配了代理，跟 AI 打交道的事都在沙箱里做（pkg/agentsandbox：sing-box 接管沙箱的网络，只经这个代理联网，环境里看不到代理）：
//   - Agent 的进程：会话（连同它执行的命令、MCP 服务器）、检查登录、登录终端，代理按 effectiveProxy 定；
//   - Momoko 自己的 AI 请求：测试接口、拉模型、查余额、测试代理，经 agentsandbox.Transport 在沙箱里发。
// 跟 AI 无关的（Git、下载和更新、工作区、技能、MCP 测试、通知）不进沙箱。
// 存着的代理读不出来、sing-box 没下载，一律算出错，不会改成直连。

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
	if err != nil {
		return nil, errProxyUnreadable
	}
	return &p, nil
}

// storeProxy 要存的密文：nil 是不走代理；只有 Linux 能配。
func (uc *AgentUsecase) storeProxy(p *agentproxy.Proxy) (string, error) {
	if p == nil {
		return "", nil
	}
	if !agentsandbox.Supported() {
		return "", response.BadRequest(agentsandbox.ErrUnsupported.Error())
	}
	return uc.encrypt(p.Encode()), nil
}

// proxyKey 比较两份存着的代理是不是同一个（同一个代理每次加密出来的密文不一样）。
func (uc *AgentUsecase) proxyKey(stored string) string {
	if stored == "" {
		return ""
	}
	return uc.decrypt(stored)
}

// encodedProxy 和 proxyKey 比较用：p 存起来的样子；nil 为空。
func encodedProxy(p *agentproxy.Proxy) string {
	if p == nil {
		return ""
	}
	return p.Encode()
}

// proxyPB 页面上的代理：能改的人看到整个节点；只能用的人（view）只看到协议、地址、时区、语言。
func (uc *AgentUsecase) proxyPB(stored string, view bool) *v1.AgentProxy {
	p, err := uc.proxyOf(stored)
	if err != nil || p == nil {
		return nil
	}
	out := &v1.AgentProxy{Protocol: p.Protocol(), Server: p.Server(), Timezone: p.Timezone, Locale: p.Locale}
	if !view {
		out.Node = p.Node
	}
	return out
}

// proxyFromPB 页面上填的代理：不传或节点为空是不走代理。节点认不出、时区服务器上没有、sing-box 用不了这个节点（下载了的话）都报错。
func (uc *AgentUsecase) proxyFromPB(in *v1.AgentProxy) (*agentproxy.Proxy, error) {
	if strings.TrimSpace(in.GetNode()) == "" {
		return nil, nil
	}
	p, err := agentproxy.New(in.GetNode(), in.GetTimezone(), in.GetLocale())
	if err != nil {
		return nil, response.BadRequest(err.Error())
	}
	if p.Timezone != "" {
		if _, err := time.LoadLocation(p.Timezone); err != nil {
			return nil, response.BadRequest("服务器上没有时区 " + p.Timezone + " 的数据")
		}
	}
	if launch, ok := agentinstall.Installed(uc.singBoxDir()); ok {
		if err := agentsandbox.CheckProxy(launch.Command, p); err != nil {
			return nil, response.BadRequest(err.Error())
		}
	}
	return &p, nil
}

// effectiveProxy 一个 Agent 进程该走的代理：用第三方接口、接口配了代理时用接口的；否则用这个 Agent（运行时）配的。
// 两边都没配才直接联网——只要有一边配了，进程就进沙箱。
func (uc *AgentUsecase) effectiveProxy(rt *gen.AgentRuntime, provider *gen.AgentProvider) (*agentproxy.Proxy, error) {
	if provider != nil && provider.Proxy != "" {
		return uc.proxyOf(provider.Proxy)
	}
	return uc.proxyOf(rt.Proxy)
}

// sandboxOf 有代理时的沙箱：sing-box 要先下载好。没有代理为 nil。
func (uc *AgentUsecase) sandboxOf(p *agentproxy.Proxy) (*agentsandbox.Config, error) {
	if p == nil {
		return nil, nil
	}
	launch, ok := agentinstall.Installed(uc.singBoxDir())
	if !ok {
		return nil, response.BadRequest("沙箱要用的 sing-box 还没下载：到「运行时」里下载")
	}
	return &agentsandbox.Config{Proxy: *p, SingBox: launch.Command}, nil
}

// runtimeSandbox 用服务器上的登录起这个 Agent（检查登录、登录终端）时的沙箱：按运行时的代理。
func (uc *AgentUsecase) runtimeSandbox(rt *gen.AgentRuntime) (*agentsandbox.Config, error) {
	p, err := uc.proxyOf(rt.Proxy)
	if err != nil {
		return nil, err
	}
	return uc.sandboxOf(p)
}

// sandboxed 起 Agent 进程的 launch 放进沙箱 sb（为 nil 时原样返回），并带上这一系在沙箱里时的环境（关掉用不着的联网）。
func sandboxed(d agentdriver.Driver, launch agentdriver.Launch, env map[string]string, sb *agentsandbox.Config) (agentdriver.Launch, map[string]string) {
	if sb == nil {
		return launch, env
	}
	out := make(map[string]string, len(env)+4)
	for k, v := range env {
		out[k] = v
	}
	for k, v := range d.SandboxEnv() {
		out[k] = v
	}
	launch.Sandbox = sb
	return launch, out
}

// clientVia Momoko 自己的 AI 请求用的客户端：有代理时每个请求都在沙箱里发；没有代理为 nil（直接联网）。
func (uc *AgentUsecase) clientVia(p *agentproxy.Proxy, timeout time.Duration) (*http.Client, error) {
	sb, err := uc.sandboxOf(p)
	if err != nil || sb == nil {
		return nil, err
	}
	return &http.Client{Transport: agentsandbox.Transport{Config: *sb}, Timeout: timeout}, nil
}

// restartUsing 代理改了：用它的会话进程是按旧的网络起的，结束掉，下次发消息时按新的代理重起（接着原来的上下文）。
// 运行时的代理也管着这个 Agent 用没配代理的第三方接口的会话（见 effectiveProxy），所以这个 Agent 的进程都重起。
func (uc *AgentUsecase) restartUsing(runtimeID, providerID string) {
	for _, l := range uc.pool.Snapshot() {
		if (providerID != "" && l.providerID == providerID) || (runtimeID != "" && l.runtimeID == runtimeID) {
			go l.Shutdown()
		}
	}
}

// requireManage 官方账户（运行时）和沙箱的事要 agent:manage。
func (uc *AgentUsecase) requireManage(ctx context.Context) error {
	if uc.system.Check(ctx, constant.AgentManage) != nil {
		return ErrNoPermission
	}
	return nil
}

// SetRuntimeProxy 官方账户的代理。改了就结束这个 Agent 的会话进程，下次按新的起。
func (uc *AgentUsecase) SetRuntimeProxy(ctx context.Context, id string, in *v1.AgentProxy) (*v1.AgentRuntime, error) {
	rt, err := uc.getRuntime(ctx, id)
	if err != nil {
		return nil, err
	}
	p, err := uc.proxyFromPB(in)
	if err != nil {
		return nil, err
	}
	stored, err := uc.storeProxy(p)
	if err != nil {
		return nil, err
	}
	changed := uc.proxyKey(rt.Proxy) != uc.proxyKey(stored)
	rt.Proxy = stored
	if err := uc.repo.SaveAgentRuntime(ctx, rt); err != nil {
		return nil, ErrSystem(err)
	}
	if changed {
		uc.restartUsing(rt.ID, "")
	}
	uc.configChanged()
	return uc.runtimeOut(ctx, rt, uc.activeByRuntime())
}

// 测试代理时请求哪里：这一系 Agent 的官方地址（Antigravity 的是 agy acp --help 里给的默认接口地址）。
var proxyProbeTargets = map[string]string{
	"claude":      "https://api.anthropic.com/",
	"codex":       "https://chatgpt.com/",
	"antigravity": "https://daily-cloudcode-pa.googleapis.com/",
}

// TestProxy 在沙箱里经代理请求一次要用的那一家：有应答（不管状态码）就算通。不通是代理那边的状况，放在 error 里返回。
func (uc *AgentUsecase) TestProxy(ctx context.Context, req *v1.TestAgentProxyRequest) (*v1.TestAgentProxyResponse, error) {
	var target string
	if req.GetRuntimeId() != "" {
		if err := uc.requireManage(ctx); err != nil {
			return nil, err
		}
		rt, err := uc.getRuntime(ctx, req.GetRuntimeId())
		if err != nil {
			return nil, err
		}
		if target = proxyProbeTargets[string(rt.Kind)]; target == "" {
			target = "https://www.google.com/"
		}
	} else {
		vendor, baseURL := req.GetVendor(), strings.TrimSpace(req.GetBaseUrl())
		if req.GetProviderId() != "" {
			p, err := uc.editableProvider(ctx, req.GetProviderId())
			if err != nil {
				return nil, err
			}
			vendor = p.Vendor
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
		if baseURL == "" {
			return nil, response.BadRequest("先填接口地址")
		}
		target = baseURL
	}
	p, err := uc.proxyFromPB(req.GetProxy())
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, response.BadRequest("先填节点")
	}
	client, err := uc.clientVia(p, 60*time.Second)
	if err != nil {
		return nil, err
	}
	bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), 60*time.Second)
	defer cancel()
	resp := &v1.TestAgentProxyResponse{Target: target}
	hr, err := http.NewRequestWithContext(bg, http.MethodGet, target, nil)
	if err != nil {
		return nil, response.BadRequest("接口地址不对")
	}
	started := time.Now()
	r, err := client.Do(hr)
	if err != nil {
		resp.Error = err.Error()
		return resp, nil
	}
	_ = r.Body.Close()
	resp.Ok, resp.LatencyMs = true, time.Since(started).Milliseconds()
	return resp, nil
}

// ———————————————————————————— 登录终端 ————————————————————————————

// loginLaunch 这个运行时在终端里登录用的命令和它这一系的驱动；登录不了时 ok 为 false。
func (uc *AgentUsecase) loginLaunch(rt *gen.AgentRuntime) (agentdriver.Launch, agentdriver.Driver, bool) {
	d, builtin := uc.builtinDriver(rt.ID)
	if !builtin {
		d = uc.driverOf(string(rt.Kind))
	}
	launch, err := uc.runtimeLaunch(rt)
	if err != nil {
		return agentdriver.Launch{}, d, false
	}
	login, ok := d.Login(launch.Command)
	return login, d, ok
}

// ServeLogin 在服务器上跑这个运行时的登录命令，接到页面的终端里；配了代理就在沙箱里跑。命令结束后检查一次登录。
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
	sb, err := uc.runtimeSandbox(rt)
	if err != nil {
		say(err.Error())
		return err
	}
	login, env := sandboxed(d, login, login.Env, sb)
	dir, _ := os.UserHomeDir()
	spec := shellcore.Spec{
		Path:    login.Command,
		Args:    login.Args,
		Dir:     dir,
		Env:     agentproc.MergeEnv(append(sysenv.Environ(), shellcore.TermEnv()...), env),
		Sandbox: login.Sandbox,
	}
	if sb != nil {
		_ = websocket.Message.Send(conn, "\x1b[2m[system] 在沙箱里运行：只经代理 "+sb.Proxy.String()+" 联网\x1b[0m\r\n")
	}
	serveErr := shellcore.ServeCommand(conn, spec, "\r\n\x1b[2m[system] 登录命令结束了，正在检查 "+rt.Name+" 的登录…\x1b[0m\r\n")
	// 不管登录成没成，都按现在的状态检查一次（页面收到配置变了会刷新）
	if latest, err := uc.getRuntime(ctx, runtimeID); err == nil {
		go func() { _, _ = uc.detectRuntime(ctx, latest) }()
	}
	return serveErr
}
