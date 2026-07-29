package biz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"momoko/pkg/file"
	"momoko/pkg/pre"
	"momoko/pkg/utils"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/servercore"
	"momoko/pkg/task"
)

const (
	InstanceWsPath              = "/api/v1/instance/cmd/ws"
	defaultInstanceRestartTimes = 3
	defaultShutdownTimeout      = 3 * time.Second
	defaultWsPingInterval       = 20 * time.Second
)

type InstanceUsecase struct {
	repo     InstanceRepo
	fileRepo FileRepo
	tasks    *task.Manager
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

func NewInstanceUsecase(repo InstanceRepo, fileRepo FileRepo, tasks *task.Manager) (*InstanceUsecase, func(), error) {
	usecase := &InstanceUsecase{
		repo:     repo,
		instance: servercore.NewServerManager(),
		fileRepo: fileRepo,
		tasks:    tasks,
	}
	go usecase.start()
	return usecase, usecase.Close, nil
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

// Close 关闭实例管理器，超时后自动强制停止剩余进程。
func (i *InstanceUsecase) Close() {
	if i == nil {
		return
	}

	_ = i.instance.Shutdown(defaultShutdownTimeout)
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
	// 原样转发为二进制帧：日志/输出是原始字节流(见 servercore.readPipe)，用二进制帧发送，
	// 避免文本帧在多字节字符(如中文)跨帧边界处被 UTF-8 校验损坏；前端用流式解码还原。
	sendText := func(text string) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		return websocket.Message.Send(conn, []byte(text))
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

	// 创建连接后发送历史日志（原始字节，前端 xterm 直接渲染）
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
		for { // 循环接收ws消息：resize 控制帧调整 PTY 尺寸，其余按原始键盘流写入 PTY（与 SSH 终端同构）
			var input string
			if err := websocket.Message.Receive(conn, &input); err != nil {
				closeWsDone()
				return
			}
			if cols, rows, ok := parseWsResizeControl(input); ok {
				_ = server.Resize(cols, rows)
				continue
			}
			// 未运行时静默丢弃击键（前端以状态标签提示，不逐键回错误）
			_ = server.Write([]byte(input))
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
		case entry := <-logCh: // 将实例日志转发到ws（原始字节）
			if err := sendText(entry.Text); err != nil {
				closeWsDone()
				return
			}
		}
	}
}

// wsResizeControl 是终端 ws 上的窗口尺寸控制帧（与 SSH 终端同一协议）。
type wsResizeControl struct {
	Type string `json:"type"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

// parseWsResizeControl 识别 {"type":"resize","cols":N,"rows":N} 控制帧。
func parseWsResizeControl(input string) (cols, rows int, ok bool) {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "{") {
		return 0, 0, false
	}
	var control wsResizeControl
	if err := json.Unmarshal([]byte(trimmed), &control); err != nil {
		return 0, 0, false
	}
	if control.Type != "resize" || control.Cols <= 0 || control.Rows <= 0 {
		return 0, 0, false
	}
	return control.Cols, control.Rows, true
}

func toInstanceTypeInfo(data *gen.InstanceType) *v1.InstanceTypeInfo {
	return &v1.InstanceTypeInfo{
		Id:       data.ID,
		Name:     data.Name,
		IsSystem: data.IsSystem,
		IsEnable: data.IsEnable,
	}
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

// instanceStore 返回实例工作目录对应的本地 Store 及其根路径（实例文件统一走 Store 接口）。
func (i *InstanceUsecase) instanceStore(ctx context.Context, userID, instanceID string) (file.Store, string, error) {
	item, err := i.repo.GetInstanceByUserID(ctx, userID, instanceID)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, "", ErrInstanceAccess
		}
		return nil, "", ErrSystem(err)
	}
	store, err := file.NewLocalStore(item.Path)
	if err != nil {
		return nil, "", ErrSystem(err)
	}
	return store, item.Path, nil
}

func (i *InstanceUsecase) GetFileList(ctx context.Context, userID string, req *v1.GetInstanceFileListRequest) (*v1.GetInstanceFileListResponse, error) {
	store, _, err := i.instanceStore(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	var result []*v1.FileEntryInfo
	if req.Keywords != nil {
		result, err = store.Search(ctx, req.Path, req.GetKeywords(), req.GetIncludeSubDir())
	} else {
		result, err = store.List(ctx, req.Path, req.SortField, req.IsDesc)
	}
	if err != nil {
		return nil, ErrSystem(err)
	}

	total := int64(len(result))
	start := (req.Page - 1) * req.PageSize
	end := req.Page * req.PageSize
	var pages []*v1.FileEntryInfo
	if start >= 0 && start < total {
		if end > total {
			end = total
		}
		pages = result[start:end]
	}

	directory, err := store.DirInfo(ctx, req.Path)
	if err != nil {
		return nil, ErrSystem(err)
	}

	return &v1.GetInstanceFileListResponse{
		Directory: directory,
		Items:     pages,
		Page:      req.Page,
		PageSize:  req.PageSize,
		Total:     total,
	}, nil
}

// GetFileTree 列出实例目录的直接子项（懒加载，供编辑器文件树使用）。
func (i *InstanceUsecase) GetFileTree(ctx context.Context, userID string, req *v1.GetInstanceFileTreeRequest) (*v1.GetInstanceFileTreeResponse, error) {
	store, _, err := i.instanceStore(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	entries, err := store.List(ctx, req.Path, v1.FileSortField_FILE_SORT_FIELD_NAME, false)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.GetInstanceFileTreeResponse{
		Path:  req.Path,
		Nodes: toTreeNodes(entries),
	}, nil
}

func (i *InstanceUsecase) CreateFile(ctx context.Context, userID string, req *v1.CreateInstanceFileRequest) (*v1.CreateInstanceFileResponse, error) {
	store, _, err := i.instanceStore(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	if req.Info == nil {
		return nil, ErrSystem(errors.New("创建信息为空"))
	}
	if err = store.Create(ctx, req.Info); err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.CreateInstanceFileResponse{}, nil
}

func (i *InstanceUsecase) RenameFile(ctx context.Context, userID string, req *v1.RenameInstanceFileRequest) (*v1.RenameInstanceFileResponse, error) {
	store, _, err := i.instanceStore(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	path, err := store.Rename(ctx, req.Path, req.NewName)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.RenameInstanceFileResponse{Path: path}, nil
}

func (i *InstanceUsecase) CopyFile(ctx context.Context, userID string, req *v1.CopyInstanceFileRequest) (*v1.CopyInstanceFileResponse, error) {
	store, basePath, err := i.instanceStore(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	copier, ok := store.(file.Copier)
	if !ok {
		return nil, ErrFileSourceUnsupported
	}
	payload := file.TransferPayload{BasePath: basePath, Paths: req.Paths, Target: req.TargetPath}
	t := file.NewCopyTask(task.Meta{UserID: userID, Title: transferTitle("复制", req.Paths)}, copier, payload)
	info, err := submitTransfer(ctx, i.tasks, t)
	if err != nil {
		return nil, err
	}
	return &v1.CopyInstanceFileResponse{Task: info}, nil
}

func (i *InstanceUsecase) CutFile(ctx context.Context, userID string, req *v1.CutInstanceFileRequest) (*v1.CutInstanceFileResponse, error) {
	store, basePath, err := i.instanceStore(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	mover, ok := store.(file.Mover)
	if !ok {
		return nil, ErrFileSourceUnsupported
	}
	payload := file.TransferPayload{BasePath: basePath, Paths: req.Paths, Target: req.TargetPath}
	t := file.NewMoveTask(task.Meta{UserID: userID, Title: transferTitle("移动", req.Paths)}, mover, payload)
	info, err := submitTransfer(ctx, i.tasks, t)
	if err != nil {
		return nil, err
	}
	return &v1.CutInstanceFileResponse{Task: info}, nil
}

func (i *InstanceUsecase) BatchDeleteFile(ctx context.Context, userID string, req *v1.BatchDeleteInstanceFileRequest) (*v1.BatchDeleteInstanceFileResponse, error) {
	store, _, err := i.instanceStore(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	items := store.Delete(ctx, req.Paths)
	return &v1.BatchDeleteInstanceFileResponse{Items: items}, nil
}

func (i *InstanceUsecase) BatchCompressFile(ctx context.Context, userID string, req *v1.BatchCompressInstanceFileRequest) (*v1.BatchCompressInstanceFileResponse, error) {
	store, _, err := i.instanceStore(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	archiver, ok := store.(file.Archiver)
	if !ok {
		return nil, ErrFileSourceUnsupported
	}
	outputPath, err := archiver.Compress(ctx, req.Paths, req.GetTargetPath())
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.BatchCompressInstanceFileResponse{OutputPath: outputPath}, nil
}

func (i *InstanceUsecase) UnzipFile(ctx context.Context, userID string, req *v1.UnzipInstanceFileRequest) (*v1.UnzipInstanceFileResponse, error) {
	store, _, err := i.instanceStore(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	archiver, ok := store.(file.Archiver)
	if !ok {
		return nil, ErrFileSourceUnsupported
	}
	outputPath, err := archiver.Unzip(ctx, req.Path, req.GetTargetPath())
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.UnzipInstanceFileResponse{OutputPath: outputPath}, nil
}

func (i *InstanceUsecase) EditFile(ctx context.Context, userID string, req *v1.EditInstanceFileRequest) (*v1.EditInstanceFileResponse, error) {
	store, _, err := i.instanceStore(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	if len(req.Content) > file.MaxLoadFileSize {
		return nil, ErrSystem(fmt.Errorf("文件太大"))
	}
	if err = store.Write(ctx, req.Path, bytes.NewReader(req.Content)); err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.EditInstanceFileResponse{}, nil
}

func (i *InstanceUsecase) FilePreSign(ctx context.Context, userID string, req *v1.InstanceFilePreSignRequest) (*v1.InstanceFilePreSignResponse, error) {
	_, basePath, err := i.instanceStore(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	realPath, err := file.ResolveLocalPath(basePath, req.Path)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if info, err := os.Stat(realPath); err != nil || info.IsDir() {
		return nil, ErrFileNotExist
	}
	preInfo := pre.NewFileSignInfo(utils.GenerateRandomString(10), realPath, userID, 24*time.Hour)
	sign, err := preInfo.Sign()
	if err != nil {
		return nil, ErrSign
	}
	return &v1.InstanceFilePreSignResponse{DownloadUrlPath: fmt.Sprintf("%s?sign=%s", PreFileDownload, sign)}, nil
}

func (i *InstanceUsecase) FilePreSignUpload(ctx context.Context, userID string, req *v1.InstanceFilePreSignUploadRequest) (*v1.UploadInfo, error) {
	if req.FileSize > math.MaxInt64 {
		return nil, ErrUploadRequestInvalid
	}
	store, basePath, err := i.instanceStore(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	realPath, err := file.ResolveLocalPath(basePath, req.Path)
	if err != nil {
		return nil, ErrSystem(err)
	}
	seed := file.NewChunkedUpload(req.Hash, realPath, req.FileName, req.FileSize)
	row, err := i.fileRepo.GetOrCreate(ctx, userID, seed)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if row.Completed {
		return toUploadInfo(row, &file.Upload{}), nil
	}
	u, err := buildUpload(i.fileRepo, userID, row)
	if err != nil {
		return nil, err
	}
	// 盲调用：实例文件为本地来源，store 在接口内填充 momoko 签名分片地址。
	if err := store.PrepareUpload(ctx, u); err != nil {
		return nil, ErrSystem(err)
	}
	uploadCache.Set(row.ID, &file.ChunkedUpload{FileUpload: row})
	return toUploadInfo(row, u), nil
}
