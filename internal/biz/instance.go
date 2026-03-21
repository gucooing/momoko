package biz

import (
	"context"
	"errors"
	"io"
	"os"

	"golang.org/x/net/websocket"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent"
	"momoko/pkg/servercore"
)

const (
	// terminalType 表示用户独立终端实例类型。
	terminalType = "terminal"
	// terminalName 表示终端实例名称。
	terminalName = "终端"
	// TerminalWSPath 表示当前终端控制台 ws 路径。
	TerminalWSPath = "/api/v1/instance/terminal/ws"
	// InstanceWsPath 应用实例控制台路径
	InstanceWsPath = "/api/v1/instance/cmd/ws"
)

type InstanceUsecase struct {
	repo     InstanceRepo
	terminal *servercore.ServerManager // 用户终端实例
	instance *servercore.ServerManager // 普通应用实例
}

type InstanceRepo interface {
	GetTypes(ctx context.Context) ([]*ent.InstanceType, error)
	CreateType(ctx context.Context, name string) (*ent.InstanceType, error)
	UpdateType(ctx context.Context, id string, name *string, isEnable *bool) (*ent.InstanceType, error)
	DeleteType(ctx context.Context, id string) error
	GetInstances(ctx context.Context, page, pageSize int64, userId string, keywords, types *string) ([]*ent.Instance, int64, error)
	GetInstanceByUserID(ctx context.Context, userId, instanceId string) (*ent.Instance, error)
	CreateInstance(ctx context.Context, req *v1.CreateInstanceRequest, userId string) (*ent.Instance, error)
}

func NewInstanceUsecase(repo InstanceRepo) *InstanceUsecase {
	return &InstanceUsecase{
		repo:     repo,
		terminal: servercore.NewServerManager(),
		instance: servercore.NewServerManager(),
	}
}

func (i *InstanceUsecase) GetTypes(ctx context.Context) ([]*v1.InstanceTypeInfo, error) {
	typeInfos, err := i.repo.GetTypes(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}

	types := make([]*v1.InstanceTypeInfo, 0, len(typeInfos))
	for _, typeInfo := range typeInfos {
		types = append(types, toInstanceTypeInfo(typeInfo))
	}
	return types, nil
}

func (i *InstanceUsecase) CreateType(ctx context.Context, req *v1.CreateInstanceTypeRequest) (*v1.InstanceTypeInfo, error) {
	if req.Name == "" {
		return nil, ErrInstanceTypeName
	}

	typeInfo, err := i.repo.CreateType(ctx, req.Name)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toInstanceTypeInfo(typeInfo), nil
}

func (i *InstanceUsecase) UpdateType(ctx context.Context, req *v1.UpdateInstanceTypeRequest) (*v1.InstanceTypeInfo, error) {
	if req.Id == "" {
		return nil, ErrInstanceTypeID
	}

	typeInfo, err := i.repo.UpdateType(ctx, req.Id, req.Name, req.IsEnable)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toInstanceTypeInfo(typeInfo), nil
}

func (i *InstanceUsecase) DeleteType(ctx context.Context, id string) error {
	if err := i.repo.DeleteType(ctx, id); err != nil {
		return ErrSystem(err)
	}
	return nil
}

// GetTerminalInfo 获取当前用户终端实例信息。
func (i *InstanceUsecase) GetTerminalInfo(ctx context.Context, userID string) (*v1.InstanceInfo, error) {
	terminal, err := i.ensureTerminal(userID)
	if err != nil {
		return nil, err
	}

	return i.terminalToInstanceInfo(terminal), nil
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
	terminal, err := i.ensureTerminal(userID)
	if err != nil {
		return err
	}
	return terminal.Start()
}

// StopTerminal 停止当前用户终端。
func (i *InstanceUsecase) StopTerminal(ctx context.Context, userID string) error {
	terminal, err := i.ensureTerminal(userID)
	if err != nil {
		return err
	}
	return terminal.Stop()
}

// RestartTerminal 重启当前用户终端。
func (i *InstanceUsecase) RestartTerminal(ctx context.Context, userID string) error {
	terminal, err := i.ensureTerminal(userID)
	if err != nil {
		return err
	}
	return terminal.Restart()
}

func defaultTerminalDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}

func (i *InstanceUsecase) GetInstances(ctx context.Context, req *v1.GetInstancesRequest, userId string) ([]*v1.InstanceInfo, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 50 {
		req.PageSize = 50
	}
	items, total, err := i.repo.GetInstances(ctx, req.Page, req.PageSize, userId, req.Keywords, req.Type)
	if err != nil {
		return nil, 0, ErrSystem(err)
	}
	infos := make([]*v1.InstanceInfo, 0, len(items))
	for _, item := range items {
		core, err := i.ensureInstance(item)
		if err != nil {
			return nil, 0, err
		}
		infos = append(infos, i.toInstanceInfo(core, item))
	}
	return infos, total, nil
}

// CreateInstance 将实例创建在目标用户下面
func (i *InstanceUsecase) CreateInstance(ctx context.Context, req *v1.CreateInstanceRequest, userId string) (*v1.InstanceInfo, error) {
	item, err := i.repo.CreateInstance(ctx, req, userId)
	if err != nil {
		return nil, ErrSystem(err)
	}
	core, err := i.ensureInstance(item)
	if err != nil {
		return nil, err
	}
	return i.toInstanceInfo(core, item), nil
}

func (i *InstanceUsecase) GetInstanceByUserID(ctx context.Context, userId, instanceId string) (*v1.InstanceInfo, error) {
	item, err := i.repo.GetInstanceByUserID(ctx, userId, instanceId)
	if err != nil {
		return nil, ErrSystem(err)
	}
	core, err := i.ensureInstance(item)
	if err != nil {
		return nil, err
	}
	return i.toInstanceInfo(core, item), nil
}

// GetInstanceCore 用户id限制拉取实例信息
func (i *InstanceUsecase) GetInstanceCore(ctx context.Context, userId, instanceId string) (*servercore.Server, error) {
	item, err := i.repo.GetInstanceByUserID(ctx, userId, instanceId)
	if err != nil {
		return nil, ErrSystem(err)
	}
	core, err := i.ensureInstance(item)
	if err != nil {
		return nil, err
	}
	return core, nil
}

// StartInstance 启动实例
func (i *InstanceUsecase) StartInstance(ctx context.Context, userId, instanceId string) (*v1.InstanceInfo, error) {
	item, err := i.repo.GetInstanceByUserID(ctx, userId, instanceId)
	if err != nil {
		return nil, ErrSystem(err)
	}
	core, err := i.ensureInstance(item)
	if err != nil {
		return nil, err
	}
	err = core.Start()
	if err != nil {
		return nil, err
	}
	return i.toInstanceInfo(core, item), nil
}

// StopInstance 停止实例
func (i *InstanceUsecase) StopInstance(ctx context.Context, userId, instanceId string) error {
	item, err := i.repo.GetInstanceByUserID(ctx, userId, instanceId)
	if err != nil {
		return ErrSystem(err)
	}
	core, err := i.ensureInstance(item)
	if err != nil {
		return err
	}
	return core.Stop()
}

// RestartInstance 重启实例
func (i *InstanceUsecase) RestartInstance(ctx context.Context, userId, instanceId string) error {
	item, err := i.repo.GetInstanceByUserID(ctx, userId, instanceId)
	if err != nil {
		return ErrSystem(err)
	}
	core, err := i.ensureInstance(item)
	if err != nil {
		return err
	}
	return core.Restart()
}

// 删除实例

// 获取/创建终端实例
func (i *InstanceUsecase) ensureTerminal(userID string) (*servercore.Server, error) {
	if terminal, ok := i.terminal.Get(userID); ok {
		return terminal, nil
	}
	cfg := servercore.NewTerminalConfig(userID, defaultTerminalDir())
	terminal, err := i.terminal.Create(cfg)
	if err == nil {
		return terminal, nil
	}
	if terminal, ok := i.terminal.Get(userID); ok {
		return terminal, nil
	}
	return nil, ErrInstanceNotFound
}

// 获取/创建应用实例
func (i *InstanceUsecase) ensureInstance(item *ent.Instance) (*servercore.Server, error) {
	if terminal, ok := i.instance.Get(item.ID); ok {
		return terminal, nil
	}
	instance, err := i.instance.Create(servercore.ServerConfig{
		ID:               item.ID,
		Command:          item.StartCommand,
		Args:             nil,
		Dir:              item.Path,
		Env:              item.Env,
		LogLimit:         0,
		SubscriberBuffer: 0,
		Terminal:         false,
	})
	if err == nil {
		return instance, nil
	}
	if instance, ok := i.instance.Get(item.ID); ok {
		return instance, nil
	}
	return nil, ErrInstanceNotFound
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

func toInstanceTypeInfo(data *ent.InstanceType) *v1.InstanceTypeInfo {
	return &v1.InstanceTypeInfo{
		Id:       data.ID,
		Name:     data.Name,
		IsSystem: data.IsSystem,
		IsEnable: data.IsEnable,
	}
}

func (i *InstanceUsecase) terminalToInstanceInfo(terminal *servercore.Server) *v1.InstanceInfo {
	info := &v1.InstanceInfo{
		Id:           terminal.ID(),
		Name:         terminalName,
		Status:       v1.InstanceStatus_INSTANCE_STATUS_STOPPED,
		CreateTime:   timestamppb.New(terminal.CreateTime()),
		Remark:       "",
		Tags:         "",
		Type:         terminalType,
		UserId:       terminal.ID(),
		StartCommand: terminal.CommandLine(),
		InstancePath: terminal.Dir(),
		StartTime:    nil,
		WsPath:       "",
		StopCommand:  "",
		AutoStart:    false,
	}
	if terminal.Running() {
		info.Status = v1.InstanceStatus_INSTANCE_STATUS_RUNNING
		if startTime, ok := terminal.StartTime(); ok {
			info.StartTime = timestamppb.New(startTime)
		}
		info.WsPath = TerminalWSPath
	}

	return info
}

func (i *InstanceUsecase) toInstanceInfo(server *servercore.Server, item *ent.Instance) *v1.InstanceInfo {
	info := &v1.InstanceInfo{
		Id:           item.ID,
		Name:         item.Name,
		Status:       v1.InstanceStatus_INSTANCE_STATUS_STOPPED,
		CreateTime:   timestamppb.New(item.CreateTime),
		Remark:       item.Remark,
		Tags:         item.Tags,
		Type:         "",
		UserId:       item.UserID,
		StartCommand: item.StartCommand,
		InstancePath: item.Path,
		StartTime:    nil,
		WsPath:       "",
		StopCommand:  item.StopCommand,
		AutoStart:    item.AutoStart,
		Env:          item.Env,
	}
	if server.Running() {
		info.Status = v1.InstanceStatus_INSTANCE_STATUS_RUNNING
		if startTime, ok := server.StartTime(); ok {
			info.StartTime = timestamppb.New(startTime)
		}
		info.WsPath = InstanceWsPath
	}
	if t := item.Edges.Type; t != nil {
		info.Type = t.Name
	}

	return info
}
