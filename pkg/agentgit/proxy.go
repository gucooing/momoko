package agentgit

import (
	"context"
	"os/exec"

	"momoko/pkg/agentproxy"
	"momoko/pkg/agentsandbox"
)

type proxyKey struct{}

// WithProxy 之后用这个 ctx 跑的 git 命令只能经 p 联网：和配了代理的 Agent 一样关在隔离的网络里（见 pkg/agentsandbox），
// HTTPS 远程走代理，SSH 远程连不出去。p 为 nil 时原样返回。
func WithProxy(ctx context.Context, p *agentproxy.Proxy) context.Context {
	if p == nil {
		return ctx
	}
	return context.WithValue(ctx, proxyKey{}, p)
}

// execute 跑 cmd 直到结束：ctx 带着代理时在隔离网络里跑。
func execute(ctx context.Context, cmd *exec.Cmd) error {
	p, _ := ctx.Value(proxyKey{}).(*agentproxy.Proxy)
	if p == nil {
		return cmd.Run()
	}
	for k, v := range agentproxy.Env(agentsandbox.ProxyAddr) {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	sb, err := agentsandbox.Wrap(cmd)
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		sb.Close()
		return err
	}
	if err := sb.Attach(*p); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		sb.Close()
		return err
	}
	err = cmd.Wait()
	sb.Close()
	return err
}
