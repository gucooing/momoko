package mcpprobe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"momoko/pkg/agentproc"
	"momoko/pkg/jsonrpc"
)

const (
	// 一个请求发出去后，命令这么久不回话就算没响应。
	replyTimeout = 10 * time.Second
	// 结束进程后最多等这么久收尾，免得一个杀不掉的进程把测试挂住。
	exitWait = 3 * time.Second
)

// stdioConn 把 MCP 服务器当子进程启动，在它的 stdin / stdout 上跑 JSON-RPC。
type stdioConn struct {
	proc *agentproc.Process
	rpc  *jsonrpc.Conn
}

func dialStdio(s Server) (conn, error) {
	if strings.TrimSpace(s.Command) == "" {
		return nil, errors.New("命令还没填")
	}
	if s.Dir != "" {
		if info, err := os.Stat(s.Dir); err != nil || !info.IsDir() {
			return nil, fmt.Errorf("工作目录不存在：%s", s.Dir)
		}
	}
	proc, err := agentproc.Start(agentproc.Config{Command: s.Command, Args: s.Args, Dir: s.Dir, Env: s.Env})
	if err != nil {
		return nil, err
	}
	return &stdioConn{proc: proc, rpc: jsonrpc.NewConn(proc.Stdout(), proc.Stdin(), pingHandler{})}, nil
}

// pingHandler 回服务器发来的请求：ping 回空对象，别的一律说不支持（握手时没声明任何能力）。
type pingHandler struct{}

func (pingHandler) Notification(string, json.RawMessage) {}

func (pingHandler) Request(method string, _ json.RawMessage, reply func(any, *jsonrpc.Error)) {
	if method == "ping" {
		reply(struct{}{}, nil)
		return
	}
	reply(nil, jsonrpc.NewError(jsonrpc.CodeMethodNotFound, "Method not found"))
}

func (c *stdioConn) call(ctx context.Context, method string, params, result any) error {
	callCtx, cancel := context.WithTimeout(ctx, replyTimeout)
	defer cancel()
	err := c.rpc.Call(callCtx, method, params, result)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, jsonrpc.ErrClosed) || c.proc.Exited():
		return c.exitError()
	case ctx.Err() != nil:
		return timeoutError(ctx, "命令")
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("命令 %d 秒内没有响应", int(replyTimeout/time.Second))
	}
	return err
}

func (c *stdioConn) notify(_ context.Context, method string) error {
	if err := c.rpc.Notify(method, nil); err != nil {
		return c.exitError()
	}
	return nil
}

func (c *stdioConn) negotiated(string) {}

// exitError 进程提前退出时的原因：取 stderr 的最后一行。
func (c *stdioConn) exitError() error {
	// 等进程真的结束，stderr 才是全的。
	c.proc.Wait(exitWait)
	if line := c.proc.LastError(); line != "" {
		return fmt.Errorf("命令退出了：%s", clip(line))
	}
	return errors.New("命令退出了")
}

func (c *stdioConn) close() {
	c.rpc.Close()
	c.proc.Kill()
	c.proc.Wait(exitWait)
}
