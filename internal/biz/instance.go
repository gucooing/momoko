package biz

import (
	"context"
	"os"
	"sync"
	"time"

	"golang.org/x/net/websocket"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/servercore"
)

const (
	terminalType                = "terminal"
	terminalName                = "终端"
	TerminalWSPath              = "/api/v1/instance/terminal/ws"
	InstanceWsPath              = "/api/v1/instance/cmd/ws"
	defaultInstanceRestartTimes = 3
	defaultShutdownTimeout      = 3 * time.Second
	defaultWsPingInterval       = 20 * time.Second
)

type InstanceUsecase struct {
	repo     InstanceRepo
	terminal *servercore.ServerManager
	instance *servercore.ServerManager
}

type InstanceRepo interface {
	GetTypes(ctx context.Context) ([]*gen.InstanceType, error)
	CreateType(ctx context.Context, name string) (*gen.InstanceType, error)
	UpdateType(ctx context.Context, id string, name *string, isEnable *bool) (*gen.InstanceType, error)
	DeleteType(ctx context.Context, id string) error
	GetInstances(ctx context.Context, page, pageSize int64, userId string, keywords, types *string) ([]*gen.Instance, int64, error)
	GetInstanceByUserID(ctx context.Context, userId, instanceId string) (*gen.Instance, error)
	CreateInstance(ctx context.Context, req *v1.CreateInstanceRequest, userId string) (*gen.Instance, error)
	UpdateInstance(ctx context.Context, req *v1.UpdateInstanceRequest, userId string) (*gen.Instance, error)
	DeleteInstance(ctx context.Context, id, userId string) error
	GetAllAutoStartInstances(ctx context.Context) ([]*gen.Instance, error)
}

func NewInstanceUsecase(repo InstanceRepo) (*InstanceUsecase, func(), error) {
	usecase := &InstanceUsecase{
		repo:     repo,
		terminal: servercore.NewServerManager(),
		instance: servercore.NewServerManager(),
	}
	go usecase.start()
	return usecase, func() {
		usecase.Close()
	}, nil
}

func (i *InstanceUsecase) start() {
	items, err := i.repo.GetAllAutoStartInstances(context.Background())
	if err != nil {
		return
	}
	for _, item := range items {
		core, err := i.ensureInstance(item)
		if err != nil {
			continue
		}
		core.Start()
	}
}

// Close 并发关闭终端和实例管理器，超时后自动强制停止剩余进程。
func (i *InstanceUsecase) Close() {
	if i == nil {
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_ = i.terminal.Shutdown(defaultShutdownTimeout)
	}()

	go func() {
		defer wg.Done()
		_ = i.instance.Shutdown(defaultShutdownTimeout)
	}()

	wg.Wait()
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

func (i *InstanceUsecase) StartTerminal(ctx context.Context, userID string) error {
	terminal, err := i.ensureTerminal(userID)
	if err != nil {
		return err
	}
	return terminal.Start()
}

func (i *InstanceUsecase) StopTerminal(ctx context.Context, userID string) error {
	terminal, err := i.ensureTerminal(userID)
	if err != nil {
		return err
	}
	return terminal.Stop()
}

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

func (i *InstanceUsecase) GetInstances(ctx context.Context, req *v1.GetInstancesRequest, userID string) ([]*v1.InstanceInfo, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 50 {
		req.PageSize = 50
	}

	items, total, err := i.repo.GetInstances(ctx, req.Page, req.PageSize, userID, req.Keywords, req.Type)
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

func (i *InstanceUsecase) CreateInstance(ctx context.Context, req *v1.CreateInstanceRequest, userID string) (*v1.InstanceInfo, error) {
	item, err := i.repo.CreateInstance(ctx, req, userID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	core, err := i.ensureInstance(item)
	if err != nil {
		return nil, err
	}
	return i.toInstanceInfo(core, item), nil
}

func (i *InstanceUsecase) UpdateInstance(ctx context.Context, req *v1.UpdateInstanceRequest, userID string) (*v1.InstanceInfo, error) {
	item, err := i.repo.UpdateInstance(ctx, req, userID)
	if err != nil {
		return nil, i.wrapInstanceRepoErr(err)
	}
	core, err := i.ensureInstance(item)
	if err != nil {
		return nil, err
	}
	if err := core.UpdateConfig(toServerConfig(item)); err != nil {
		return nil, err
	}

	return i.toInstanceInfo(core, item), nil
}

func (i *InstanceUsecase) DeleteInstance(ctx context.Context, userID, instanceID string) error {
	if err := i.repo.DeleteInstance(ctx, instanceID, userID); err != nil {
		return i.wrapInstanceRepoErr(err)
	}

	i.instance.Remove(instanceID)
	return nil
}

func (i *InstanceUsecase) GetInstanceByUserID(ctx context.Context, userID, instanceID string) (*v1.InstanceInfo, error) {
	item, err := i.repo.GetInstanceByUserID(ctx, userID, instanceID)
	if err != nil {
		return nil, i.wrapInstanceRepoErr(err)
	}
	core, err := i.ensureInstance(item)
	if err != nil {
		return nil, err
	}
	return i.toInstanceInfo(core, item), nil
}

func (i *InstanceUsecase) GetInstanceCore(ctx context.Context, userID, instanceID string) (*servercore.Server, error) {
	item, err := i.repo.GetInstanceByUserID(ctx, userID, instanceID)
	if err != nil {
		return nil, i.wrapInstanceRepoErr(err)
	}
	core, err := i.ensureInstance(item)
	if err != nil {
		return nil, err
	}
	return core, nil
}

func (i *InstanceUsecase) StartInstance(ctx context.Context, userID, instanceID string) (*v1.InstanceInfo, error) {
	item, err := i.repo.GetInstanceByUserID(ctx, userID, instanceID)
	if err != nil {
		return nil, i.wrapInstanceRepoErr(err)
	}
	core, err := i.ensureInstance(item)
	if err != nil {
		return nil, err
	}
	if err := core.Start(); err != nil {
		return nil, err
	}
	return i.toInstanceInfo(core, item), nil
}

func (i *InstanceUsecase) StopInstance(ctx context.Context, userID, instanceID string, force bool) error {
	item, err := i.repo.GetInstanceByUserID(ctx, userID, instanceID)
	if err != nil {
		return i.wrapInstanceRepoErr(err)
	}
	core, err := i.ensureInstance(item)
	if err != nil {
		return err
	}
	if force {
		return core.ForceStop()
	}
	return core.Stop()
}

func (i *InstanceUsecase) RestartInstance(ctx context.Context, userID, instanceID string, force bool) error {
	item, err := i.repo.GetInstanceByUserID(ctx, userID, instanceID)
	if err != nil {
		return i.wrapInstanceRepoErr(err)
	}
	core, err := i.ensureInstance(item)
	if err != nil {
		return err
	}
	if force {
		return core.ForceRestart()
	}
	return core.Restart()
}

func (i *InstanceUsecase) DelInstanceLog(ctx context.Context, userID, instanceID string) error {
	item, err := i.repo.GetInstanceByUserID(ctx, userID, instanceID)
	if err != nil {
		return i.wrapInstanceRepoErr(err)
	}
	core, err := i.ensureInstance(item)
	if err != nil {
		return err
	}
	core.ClearLogs()
	return nil
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
	if terminal, ok := i.terminal.Get(userID); ok {
		return terminal, nil
	}
	return nil, ErrInstanceNotFound
}

func (i *InstanceUsecase) ensureInstance(item *gen.Instance) (*servercore.Server, error) {
	cfg := toServerConfig(item)

	if server, ok := i.instance.Get(item.ID); ok {
		if err := server.UpdateConfig(cfg); err != nil {
			return nil, err
		}
		return server, nil
	}

	server, err := i.instance.Create(cfg)
	if err == nil {
		return server, nil
	}
	if server, ok := i.instance.Get(item.ID); ok {
		if err := server.UpdateConfig(cfg); err != nil {
			return nil, err
		}
		return server, nil
	}
	return nil, ErrInstanceNotFound
}

func toServerConfig(item *gen.Instance) servercore.ServerConfig {
	return servercore.ServerConfig{
		ID:                 item.ID,
		Command:            item.StartCommand,
		CommandLine:        true,
		Dir:                item.Path,
		Env:                item.Env,
		StopCommand:        item.StopCommand,
		AutoRestart:        item.AutoStart,
		MaxRestartAttempts: defaultInstanceRestartTimes,
	}
}

func (i *InstanceUsecase) wrapInstanceRepoErr(err error) error {
	if gen.IsNotFound(err) {
		return ErrInstanceAccess
	}
	return ErrSystem(err)
}

func (i *InstanceUsecase) StartInstanceWsConn(conn *websocket.Conn, server *servercore.Server) {
	if server == nil {
		_ = websocket.Message.Send(conn, "实例不存在")
		return
	}

	var sendMu sync.Mutex
	sendText := func(text string) error { // 发送文本方法
		sendMu.Lock()
		defer sendMu.Unlock()
		return websocket.Message.Send(conn, text)
	}
	sendPing := func() error { // 发送ping方法
		sendMu.Lock()
		defer sendMu.Unlock()

		payloadType := conn.PayloadType
		conn.PayloadType = websocket.PingFrame
		defer func() {
			conn.PayloadType = payloadType
		}()

		_, err := conn.Write(nil)
		return err
	}

	// 创建连接后发送历史日志
	for _, entry := range server.RecentLogs() {
		if err := sendText(entry.Text); err != nil {
			return
		}
	}

	logCh, subDone, cancel := server.SubscribeWithDone() // 订阅实例控制台
	defer cancel()

	pingTicker := time.NewTicker(defaultWsPingInterval)
	defer pingTicker.Stop()
	wsDone := make(chan struct{})
	var wsDoneOnce sync.Once
	closeWsDone := func() {
		wsDoneOnce.Do(func() {
			close(wsDone)
		})
	}
	defer closeWsDone()

	go func() {
		for { // 循环接收ws消息并转发到实例中
			var input string
			if err := websocket.Message.Receive(conn, &input); err != nil {
				closeWsDone()
				return
			}
			if err := server.Send(input); err != nil {
				if sendErr := sendText(err.Error()); sendErr != nil {
					closeWsDone()
					return
				}
			}
		}
	}()

	for {
		select {
		case <-subDone: // 订阅被取消
			return
		case <-wsDone: // ws断开
			return
		case <-pingTicker.C: // ping到了
			if err := sendPing(); err != nil {
				closeWsDone()
				return
			}
		case entry := <-logCh: // 将实例日志转发到ws
			if err := sendText(entry.Text); err != nil {
				closeWsDone()
				return
			}
		}
	}
}

func toInstanceTypeInfo(data *gen.InstanceType) *v1.InstanceTypeInfo {
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
		Type:         terminalType,
		UserId:       terminal.ID(),
		StartCommand: terminal.CommandLine(),
		InstancePath: terminal.Dir(),
		StopCommand:  "",
		AutoStart:    false,
		WsPath:       TerminalWSPath,
	}
	if terminal.Running() {
		info.Status = v1.InstanceStatus_INSTANCE_STATUS_RUNNING
		if startTime, ok := terminal.StartTime(); ok {
			info.StartTime = timestamppb.New(startTime)
		}
	}
	return info
}

func (i *InstanceUsecase) toInstanceInfo(server *servercore.Server, item *gen.Instance) *v1.InstanceInfo {
	info := &v1.InstanceInfo{
		Id:           item.ID,
		Name:         item.Name,
		Status:       v1.InstanceStatus_INSTANCE_STATUS_STOPPED,
		CreateTime:   timestamppb.New(item.CreateTime),
		Remark:       item.Remark,
		Tags:         item.Tags,
		StartCommand: item.StartCommand,
		InstancePath: item.Path,
		StopCommand:  item.StopCommand,
		AutoStart:    item.AutoStart,
		Env:          item.Env,
		WsPath:       InstanceWsPath,
	}
	if server.Running() {
		info.Status = v1.InstanceStatus_INSTANCE_STATUS_RUNNING
		if startTime, ok := server.StartTime(); ok {
			info.StartTime = timestamppb.New(startTime)
		}
	}
	if t := item.Edges.Type; t != nil {
		info.Type = t.Name
	}
	if u := item.Edges.User; u != nil {
		info.UserId = u.ID
	}
	return info
}
