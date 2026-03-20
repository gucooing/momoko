package biz

import (
	"context"
	"errors"
	"golang.org/x/net/websocket"
	"io"
	"os"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/servercore"
)

const (
	// terminalType 表示用户独立终端实例类型。
	terminalType = "terminal"
	// terminalName 表示终端实例名称。
	terminalName = "终端"
	// TerminalWSPath 表示当前终端控制台 ws 路径。
	TerminalWSPath = "/api/v1/instance/terminal/ws"
)

type InstanceUsecase struct {
	terminal *servercore.ServerManager // 用户终端实例
	instance *servercore.ServerManager // 普通应用实例
}

func NewInstanceUsecase() *InstanceUsecase {
	return &InstanceUsecase{
		terminal: servercore.NewServerManager(),
		instance: servercore.NewServerManager(),
	}
}

// GetTerminalInfo 获取当前用户终端实例信息。
func (i *InstanceUsecase) GetTerminalInfo(ctx context.Context, userID string) (*v1.InstanceInfo, error) {
	_ = ctx

	terminal, err := i.ensureTerminal(userID)
	if err != nil {
		return nil, err
	}

	info := &v1.InstanceInfo{
		Id:           terminal.ID(),
		Name:         terminalName,
		Status:       v1.InstanceStatus_INSTANCE_STATUS_STOPPED,
		CreateTime:   timestamppb.New(terminal.CreateTime()),
		Remark:       "",
		Tags:         nil,
		Type:         terminalType,
		UserId:       userID,
		StartCommand: terminal.CommandLine(),
		InstancePath: terminal.Dir(),
	}

	if terminal.Running() {
		info.Status = v1.InstanceStatus_INSTANCE_STATUS_RUNNING
		if startTime, ok := terminal.StartTime(); ok {
			info.StartTime = timestamppb.New(startTime)
		}
		info.WsPath = TerminalWSPath
	}

	return info, nil
}

func (i *InstanceUsecase) GetTerminalServer(ctx context.Context, userID string) (*servercore.Server, error) {
	terminal, err := i.ensureTerminal(userID)
	if err != nil {
		return nil, err
	}

	return terminal, nil
}

// StartTerminal 启动当前用户终端。
func (i *InstanceUsecase) StartTerminal(ctx context.Context, userID string) error {
	_ = ctx

	terminal, err := i.ensureTerminal(userID)
	if err != nil {
		return err
	}
	return terminal.Start()
}

// StopTerminal 停止当前用户终端。
func (i *InstanceUsecase) StopTerminal(ctx context.Context, userID string) error {
	_ = ctx

	terminal, err := i.ensureTerminal(userID)
	if err != nil {
		return err
	}
	return terminal.Stop()
}

// RestartTerminal 重启当前用户终端。
func (i *InstanceUsecase) RestartTerminal(ctx context.Context, userID string) error {
	_ = ctx

	terminal, err := i.ensureTerminal(userID)
	if err != nil {
		return err
	}
	return terminal.Restart()
}

func (i *InstanceUsecase) ensureTerminal(userID string) (*servercore.Server, error) {
	if terminal, ok := i.terminal.Get(userID); ok {
		return terminal, nil
	}

	cfg := servercore.NewTerminalConfig(userID, defaultTerminalDir())
	terminal, err := i.terminal.Create(cfg)
	if err == nil {
		return terminal, nil
	}

	// 并发创建时，优先复用已经注册成功的实例。
	if terminal, ok := i.terminal.Get(userID); ok {
		return terminal, nil
	}

	return nil, ErrInstanceNotFound
}

func defaultTerminalDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}

// StartInstanceWsConn 启动指定实例的 ws 控制台连接。
func (i *InstanceUsecase) StartInstanceWsConn(conn *websocket.Conn, server *servercore.Server) {
	if server == nil || !server.Running() {
		websocket.Message.Send(conn, "实例未启动")
		return
	}
	// 新连接建立后，先补发最近日志，再进入实时转发。
	for _, entry := range server.RecentLogs() {
		if err := websocket.Message.Send(conn, entry.Text); err != nil {
			return
		}
	}
	logCh, cancel := server.Subscribe()
	defer cancel()

	done := make(chan struct{})
	defer close(done)

	go func() {
		for {
			select {
			case <-done:
				return
			case entry := <-logCh:
				if err := websocket.Message.Send(conn, entry.Text); err != nil {
					return
				}
			}
		}
	}()

	for {
		var input string
		if err := websocket.Message.Receive(conn, &input); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			return
		}
		_ = server.Send(input)
	}
}
