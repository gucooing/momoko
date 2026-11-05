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
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/response"
	"momoko/pkg/servercore"
	"momoko/pkg/task"
)

const (
	InstanceWsPath              = "/api/v1/instance/cmd/ws"
	defaultInstanceRestartTimes = 3
	defaultShutdownTimeout      = 3 * time.Second
	defaultWsPingInterval       = 20 * time.Second
)

var (
	ErrInstanceMaintenance = response.BadRequest(500, "应用正在维护，结束维护后才能启动")
)

// maxRuntimeEvents 每个实例在内存里保留的进程事件(退出/自动重启)条数。
const maxRuntimeEvents = 20

// secretMask 控制台输出里替换密钥变量值的占位符。
const secretMask = "******"

type InstanceUsecase struct {
	repo     InstanceRepo
	fileRepo FileRepo
	tasks    *task.Manager
	instance *servercore.ServerManager

	mu      sync.RWMutex
	secrets map[string][]string            // 实例id -> 需要在输出里打码的密钥值
	events  map[string][]*v1.InstanceEvent // 实例id -> 最近的进程事件(新的在后)
}

// InstanceEnv 环境变量的存储形态：KEY=VALUE 行 + 密钥变量名。
type InstanceEnv struct {
	Lines      []string
	SecretKeys []string
}

type InstanceRepo interface {
	GetTypes(ctx context.Context) ([]*gen.InstanceType, error)
	CreateType(ctx context.Context, req *v1.CreateInstanceTypeRequest) (*gen.InstanceType, error)
	UpdateType(ctx context.Context, req *v1.UpdateInstanceTypeRequest) (*gen.InstanceType, error)
	DeleteType(ctx context.Context, id string) (int, error)
	GetVisibleInstancesWithType(ctx context.Context, userId string) ([]*gen.Instance, error)
	GetInstances(ctx context.Context, userId string, keywords, types *string) ([]*gen.Instance, error)
	GetInstanceByUserID(ctx context.Context, userId, instanceId string) (*gen.Instance, error)
	CreateInstance(ctx context.Context, req *v1.CreateInstanceRequest, userId string, env InstanceEnv) (*gen.Instance, error)
	UpdateInstance(ctx context.Context, current *gen.Instance, req *v1.UpdateInstanceRequest, userId string, env *InstanceEnv) (*gen.Instance, error)
	DeleteInstance(ctx context.Context, id, userId string) error
	GetAllAutoStartInstances(ctx context.Context) ([]*gen.Instance, error)
	GetInstanceNames(ctx context.Context, ids []string) (map[string]string, error)
	SetLastStopTime(ctx context.Context, id string, t time.Time) error
	ListInstanceOperationLogs(ctx context.Context, instanceID string, limit int) ([]*gen.OperationLog, error)
}

func NewInstanceUsecase(repo InstanceRepo, fileRepo FileRepo, tasks *task.Manager) (*InstanceUsecase, func(), error) {
	usecase := &InstanceUsecase{
		repo:     repo,
		instance: servercore.NewServerManager(),
		fileRepo: fileRepo,
		tasks:    tasks,
		secrets:  make(map[string][]string),
		events:   make(map[string][]*v1.InstanceEvent),
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

// RunningProcesses 返回正在运行的实例 id → 主进程 pid（性能监控按应用统计用）。
func (i *InstanceUsecase) RunningProcesses() map[string]int {
	out := map[string]int{}
	for _, core := range i.instance.List() {
		if pid := core.PID(); pid > 0 {
			out[core.ID()] = pid
		}
	}
	return out
}

// InstanceNames 按 id 查实例名称（已删除的实例不在结果里）。
func (i *InstanceUsecase) InstanceNames(ctx context.Context, ids []string) map[string]string {
	if len(ids) == 0 {
		return map[string]string{}
	}
	names, err := i.repo.GetInstanceNames(ctx, ids)
	if err != nil {
		return map[string]string{}
	}
	return names
}

// Close 关闭实例管理器，超时后自动强制停止剩余进程。
func (i *InstanceUsecase) Close() {
	if i == nil {
		return
	}

	_ = i.instance.Shutdown(defaultShutdownTimeout)
}

// maxInstanceTypeApps 类型卡片上展示的应用摘要上限（总数见 app_count）。
const maxInstanceTypeApps = 8

func (i *InstanceUsecase) GetTypes(ctx context.Context, userID string) ([]*v1.InstanceTypeInfo, error) {
	typeInfos, err := i.repo.GetTypes(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}

	// 按类型归集当前用户可见的应用（数量 + 摘要），用于类型卡片的「在用的应用」
	usage := make(map[string][]*v1.InstanceTypeApp)
	counts := make(map[string]int64)
	if userID != "" {
		items, err := i.repo.GetVisibleInstancesWithType(ctx, userID)
		if err != nil {
			return nil, ErrSystem(err)
		}
		for _, item := range items {
			t := item.Edges.Type
			if t == nil {
				continue
			}
			counts[t.ID]++
			if len(usage[t.ID]) >= maxInstanceTypeApps {
				continue
			}
			status := v1.InstanceStatus_INSTANCE_STATUS_STOPPED
			if server, ok := i.instance.Get(item.ID); ok && server.Running() {
				status = v1.InstanceStatus_INSTANCE_STATUS_RUNNING
			} else if item.Maintenance {
				status = v1.InstanceStatus_INSTANCE_STATUS_MAINTENANCE
			}
			usage[t.ID] = append(usage[t.ID], &v1.InstanceTypeApp{Id: item.ID, Name: item.Name, Status: status})
		}
	}

	types := make([]*v1.InstanceTypeInfo, 0, len(typeInfos))
	for _, typeInfo := range typeInfos {
		info := toInstanceTypeInfo(typeInfo)
		info.AppCount = counts[typeInfo.ID]
		info.Apps = usage[typeInfo.ID]
		types = append(types, info)
	}
	return types, nil
}

func (i *InstanceUsecase) CreateType(ctx context.Context, req *v1.CreateInstanceTypeRequest) (*v1.InstanceTypeInfo, error) {
	if req.Name == "" {
		return nil, ErrInstanceTypeName
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, ErrInstanceTypeName
	}
	req.QuickCommands = normalizeQuickCommands(req.QuickCommands)
	typeInfo, err := i.repo.CreateType(ctx, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toInstanceTypeInfo(typeInfo), nil
}

func (i *InstanceUsecase) UpdateType(ctx context.Context, req *v1.UpdateInstanceTypeRequest) (*v1.InstanceTypeInfo, error) {
	if req.Id == "" {
		return nil, ErrInstanceTypeID
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, ErrInstanceTypeName
		}
		req.Name = &name
	}
	if req.QuickCommands != nil {
		req.QuickCommands.Items = normalizeQuickCommands(req.QuickCommands.Items)
	}
	typeInfo, err := i.repo.UpdateType(ctx, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toInstanceTypeInfo(typeInfo), nil
}

const (
	maxQuickCommands      = 20
	maxQuickCommandLength = 200
)

// normalizeQuickCommands 去掉首尾空白、空行与重复项，限制条数与单条长度。
func normalizeQuickCommands(items []string) []string {
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		cmd := strings.TrimSpace(item)
		if cmd == "" {
			continue
		}
		if r := []rune(cmd); len(r) > maxQuickCommandLength {
			cmd = string(r[:maxQuickCommandLength])
		}
		if _, ok := seen[cmd]; ok {
			continue
		}
		seen[cmd] = struct{}{}
		out = append(out, cmd)
		if len(out) == maxQuickCommands {
			break
		}
	}
	return out
}

// DeleteType 删除类型；仍在用的应用变为未分类，返回受影响的应用数。
func (i *InstanceUsecase) DeleteType(ctx context.Context, id string) (int64, error) {
	n, err := i.repo.DeleteType(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return 0, ErrInstanceTypeID
		}
		return 0, ErrSystem(err)
	}
	return int64(n), nil
}

// GetInstances 按关键字/类型取出可见的全部实例，计算运行状态与各状态数量后按状态筛选并分页。
func (i *InstanceUsecase) GetInstances(ctx context.Context, req *v1.GetInstancesRequest, userID string) (*v1.GetInstancesResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 50 {
		req.PageSize = 50
	}
	if req.Keywords != nil && strings.TrimSpace(*req.Keywords) == "" {
		req.Keywords = nil
	}
	if req.Type != nil && strings.TrimSpace(*req.Type) == "" {
		req.Type = nil
	}
	filterStatus := req.Status != nil && *req.Status != v1.InstanceStatus_INSTANCE_STATUS_UNSPECIFIED

	items, err := i.repo.GetInstances(ctx, userID, req.Keywords, req.Type)
	if err != nil {
		return nil, ErrSystem(err)
	}

	counts := &v1.InstanceStatusCount{}
	infos := make([]*v1.InstanceInfo, 0, len(items))
	for _, item := range items {
		core, err := i.ensureInstance(item)
		if err != nil {
			return nil, err
		}
		info := i.toInstanceInfo(core, item)
		counts.All++
		switch info.Status {
		case v1.InstanceStatus_INSTANCE_STATUS_RUNNING:
			counts.Running++
		case v1.InstanceStatus_INSTANCE_STATUS_MAINTENANCE:
			counts.Maintenance++
		default:
			counts.Stopped++
		}
		if filterStatus && info.Status != *req.Status {
			continue
		}
		infos = append(infos, info)
	}

	total := int64(len(infos))
	start := (req.Page - 1) * req.PageSize
	end := min(start+req.PageSize, total)
	page := []*v1.InstanceInfo{}
	if start < total {
		page = infos[start:end]
	}
	return &v1.GetInstancesResponse{
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    total,
		Infos:    page,
		Counts:   counts,
	}, nil
}

// GetInstanceTags 汇总当前用户可见应用用过的标签，按使用次数倒序。
func (i *InstanceUsecase) GetInstanceTags(ctx context.Context, userID string) ([]*v1.InstanceTag, error) {
	items, err := i.repo.GetVisibleInstancesWithType(ctx, userID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	counts := make(map[string]int64)
	for _, item := range items {
		for _, tag := range splitInstanceTags(item.Tags) {
			counts[tag]++
		}
	}
	tags := make([]*v1.InstanceTag, 0, len(counts))
	for name, n := range counts {
		tags = append(tags, &v1.InstanceTag{Name: name, Count: n})
	}
	sort.Slice(tags, func(a, b int) bool {
		if tags[a].Count != tags[b].Count {
			return tags[a].Count > tags[b].Count
		}
		return tags[a].Name < tags[b].Name
	})
	return tags, nil
}

// splitInstanceTags 拆分逗号分隔的标签（兼容中文逗号与 # 前缀），去重保序。
func splitInstanceTags(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '，' })
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		tag := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(part), "#"))
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	return out
}

// buildInstanceEnv 把前端提交的变量列表转成存储形态；密钥变量留空时沿用 previous 里的旧值。
func buildInstanceEnv(items []*v1.InstanceEnvVar, previous *gen.Instance) InstanceEnv {
	oldValues := make(map[string]string)
	oldSecret := make(map[string]struct{})
	if previous != nil {
		for _, line := range previous.Env {
			key, value, _ := strings.Cut(line, "=")
			oldValues[strings.TrimSpace(key)] = value
		}
		for _, key := range previous.SecretEnvKeys {
			oldSecret[key] = struct{}{}
		}
	}
	env := InstanceEnv{Lines: make([]string, 0, len(items)), SecretKeys: []string{}}
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		key := strings.TrimSpace(item.Key)
		if key == "" || strings.Contains(key, "=") {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		value := item.Value
		if item.IsSecret {
			value = item.Secret
			if _, wasSecret := oldSecret[key]; wasSecret && value == "" {
				value = oldValues[key]
			}
			env.SecretKeys = append(env.SecretKeys, key)
		}
		env.Lines = append(env.Lines, key+"="+value)
	}
	return env
}

// ensureWorkDir 工作目录不存在时自动创建。
func ensureWorkDir(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if info, err := os.Stat(abs); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("工作目录 %s 不是文件夹", abs)
		}
		return nil
	}
	return os.MkdirAll(abs, 0o755)
}

func (i *InstanceUsecase) CreateInstance(ctx context.Context, req *v1.CreateInstanceRequest, userID string) (*v1.InstanceInfo, error) {
	if err := ensureWorkDir(req.InstancePath); err != nil {
		return nil, ErrSystem(err)
	}
	item, err := i.repo.CreateInstance(ctx, req, userID, buildInstanceEnv(req.Env, nil))
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
	current, err := i.repo.GetInstanceByUserID(ctx, userID, req.Id)
	if err != nil {
		return nil, i.wrapInstanceRepoErr(err)
	}
	if req.InstancePath != nil && strings.TrimSpace(*req.InstancePath) != current.Path {
		if err := ensureWorkDir(*req.InstancePath); err != nil {
			return nil, ErrSystem(err)
		}
	}
	var env *InstanceEnv
	if req.Env != nil {
		built := buildInstanceEnv(req.Env.Items, current)
		env = &built
	}
	item, err := i.repo.UpdateInstance(ctx, current, req, userID, env)
	if err != nil {
		return nil, i.wrapInstanceRepoErr(err)
	}
	core, err := i.ensureInstance(item)
	if err != nil {
		return nil, err
	}
	// 进入维护：先停止应用
	if item.Maintenance && !current.Maintenance && core.Running() {
		if err := core.Stop(); err != nil {
			return nil, err
		}
	}

	return i.toInstanceInfo(core, item), nil
}

func (i *InstanceUsecase) DeleteInstance(ctx context.Context, userID, instanceID string) error {
	if err := i.repo.DeleteInstance(ctx, instanceID, userID); err != nil {
		return i.wrapInstanceRepoErr(err)
	}

	i.instance.Remove(instanceID)
	i.mu.Lock()
	delete(i.secrets, instanceID)
	delete(i.events, instanceID)
	i.mu.Unlock()
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
	if item.Maintenance {
		return nil, ErrInstanceMaintenance
	}
	core, err := i.ensureInstance(item)
	if err != nil {
		return nil, err
	}
	if err := ensureWorkDir(core.Dir()); err != nil {
		return nil, ErrSystem(err)
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
	if item.Maintenance {
		return ErrInstanceMaintenance
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
	cfg := i.serverConfig(item)
	i.rememberSecrets(item)

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

func (i *InstanceUsecase) serverConfig(item *gen.Instance) servercore.ServerConfig {
	id := item.ID
	return servercore.ServerConfig{
		ID:                 item.ID,
		Command:            item.StartCommand,
		CommandLine:        true,
		Dir:                item.Path,
		Env:                item.Env,
		StopCommand:        item.StopCommand,
		AutoRestart:        item.AutoRestart && !item.Maintenance,
		MaxRestartAttempts: defaultInstanceRestartTimes,
		StopTimeout:        time.Duration(item.StopTimeout) * time.Second,
		OnExit:             func(info servercore.ExitInfo) { i.onProcessExit(id, info) },
	}
}

// onProcessExit 记录进程退出：最终停止时持久化停止时间；非主动退出记一条进程事件。
func (i *InstanceUsecase) onProcessExit(id string, info servercore.ExitInfo) {
	if !info.Restarted {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = i.repo.SetLastStopTime(ctx, id, info.Time)
		cancel()
	}
	if info.Manual {
		return
	}
	kind := v1.InstanceEventKind_INSTANCE_EVENT_KIND_EXIT
	if info.Restarted {
		kind = v1.InstanceEventKind_INSTANCE_EVENT_KIND_AUTO_RESTART
	}
	event := &v1.InstanceEvent{
		Kind:     kind,
		Time:     timestamppb.New(info.Time),
		Success:  info.Code == 0 && info.Err == "",
		ExitCode: int32(info.Code),
		Error:    info.Err,
	}
	i.mu.Lock()
	list := append(i.events[id], event)
	if len(list) > maxRuntimeEvents {
		list = list[len(list)-maxRuntimeEvents:]
	}
	i.events[id] = list
	i.mu.Unlock()
}

// rememberSecrets 缓存实例的密钥变量值，用于控制台输出打码（只记长度 >= 3 的值，避免误伤）。
func (i *InstanceUsecase) rememberSecrets(item *gen.Instance) {
	values := make([]string, 0, len(item.SecretEnvKeys))
	if len(item.SecretEnvKeys) > 0 {
		secret := make(map[string]struct{}, len(item.SecretEnvKeys))
		for _, key := range item.SecretEnvKeys {
			secret[key] = struct{}{}
		}
		for _, line := range item.Env {
			key, value, _ := strings.Cut(line, "=")
			if _, ok := secret[strings.TrimSpace(key)]; ok && len(value) >= 3 {
				values = append(values, value)
			}
		}
		// 长的先替换，避免短值是长值的一部分时替换不完整
		sort.Slice(values, func(a, b int) bool { return len(values[a]) > len(values[b]) })
	}
	i.mu.Lock()
	if len(values) == 0 {
		delete(i.secrets, item.ID)
	} else {
		i.secrets[item.ID] = values
	}
	i.mu.Unlock()
}

// secretMasker 返回把密钥值替换成占位符的函数；没有密钥变量时原样返回。
func (i *InstanceUsecase) secretMasker(id string) func(string) string {
	i.mu.RLock()
	values := i.secrets[id]
	i.mu.RUnlock()
	if len(values) == 0 {
		return func(text string) string { return text }
	}
	pairs := make([]string, 0, len(values)*2)
	for _, v := range values {
		pairs = append(pairs, v, secretMask)
	}
	return strings.NewReplacer(pairs...).Replace
}

// maxInstanceEvents GetInstanceEvents 单次最多返回的条数。
const maxInstanceEvents = 50

// GetInstanceEvents 合并实例的操作记录、进程事件与创建记录，按时间倒序返回最近 limit 条。
func (i *InstanceUsecase) GetInstanceEvents(ctx context.Context, userID string, req *v1.GetInstanceEventsRequest) ([]*v1.InstanceEvent, error) {
	item, err := i.repo.GetInstanceByUserID(ctx, userID, req.Id)
	if err != nil {
		return nil, i.wrapInstanceRepoErr(err)
	}
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 10
	}
	limit = min(limit, maxInstanceEvents)

	logs, err := i.repo.ListInstanceOperationLogs(ctx, item.ID, limit)
	if err != nil {
		return nil, ErrSystem(err)
	}
	events := make([]*v1.InstanceEvent, 0, len(logs)+maxRuntimeEvents+1)
	for _, log := range logs {
		if event := toInstanceEvent(log); event != nil {
			events = append(events, event)
		}
	}
	i.mu.RLock()
	events = append(events, i.events[item.ID]...)
	i.mu.RUnlock()

	created := &v1.InstanceEvent{
		Kind:    v1.InstanceEventKind_INSTANCE_EVENT_KIND_CREATE,
		Time:    timestamppb.New(item.CreateTime),
		Success: true,
	}
	if u := item.Edges.User; u != nil {
		created.UserName = displayUserName(u)
	}
	events = append(events, created)

	sort.SliceStable(events, func(a, b int) bool {
		return events[a].Time.AsTime().After(events[b].Time.AsTime())
	})
	if len(events) > limit {
		events = events[:limit]
	}
	return events, nil
}

// toInstanceEvent 把一条应用操作记录转成事件；与单个应用无关的操作返回 nil。
func toInstanceEvent(log *gen.OperationLog) *v1.InstanceEvent {
	var detail struct {
		Error   string         `json:"error"`
		Request map[string]any `json:"request"`
	}
	_ = json.Unmarshal([]byte(log.Detail), &detail)
	force, _ := detail.Request["force"].(bool)

	var kind v1.InstanceEventKind
	switch operationTypeFromString(log.OperationType) {
	case v1.OperationType_OperationTypeInstanceCreate:
		kind = v1.InstanceEventKind_INSTANCE_EVENT_KIND_CREATE
	case v1.OperationType_OperationTypeInstanceStart:
		kind = v1.InstanceEventKind_INSTANCE_EVENT_KIND_START
	case v1.OperationType_OperationTypeInstanceStop:
		kind = v1.InstanceEventKind_INSTANCE_EVENT_KIND_STOP
		if force {
			kind = v1.InstanceEventKind_INSTANCE_EVENT_KIND_FORCE_STOP
		}
	case v1.OperationType_OperationTypeInstanceRestart:
		kind = v1.InstanceEventKind_INSTANCE_EVENT_KIND_RESTART
		if force {
			kind = v1.InstanceEventKind_INSTANCE_EVENT_KIND_FORCE_RESTART
		}
	case v1.OperationType_OperationTypeInstanceUpdate:
		kind = v1.InstanceEventKind_INSTANCE_EVENT_KIND_UPDATE
		// 只切换维护状态的请求单独标出
		if on, ok := detail.Request["maintenance"].(bool); ok && detail.Request["name"] == nil {
			kind = v1.InstanceEventKind_INSTANCE_EVENT_KIND_MAINTENANCE_OFF
			if on {
				kind = v1.InstanceEventKind_INSTANCE_EVENT_KIND_MAINTENANCE_ON
			}
		}
	case v1.OperationType_OperationTypeInstanceLogDelete:
		kind = v1.InstanceEventKind_INSTANCE_EVENT_KIND_LOG_CLEAR
	case v1.OperationType_OperationTypeInstanceFileCreate,
		v1.OperationType_OperationTypeInstanceFileRename,
		v1.OperationType_OperationTypeInstanceFileCopy,
		v1.OperationType_OperationTypeInstanceFileMove,
		v1.OperationType_OperationTypeInstanceFileDelete,
		v1.OperationType_OperationTypeInstanceFileCompress,
		v1.OperationType_OperationTypeInstanceFileDecompress,
		v1.OperationType_OperationTypeInstanceFileUploadPreSign,
		v1.OperationType_OperationTypeInstanceFileEdit:
		kind = v1.InstanceEventKind_INSTANCE_EVENT_KIND_FILE
	default:
		return nil
	}
	event := &v1.InstanceEvent{
		Kind:    kind,
		Time:    timestamppb.New(log.OperationTime),
		Success: log.Success,
		Error:   detail.Error,
	}
	if u := log.Edges.User; u != nil {
		event.UserName = displayUserName(u)
	}
	return event
}

// displayUserName 用户展示名：优先昵称，否则用户名。
func displayUserName(u *gen.User) string {
	if u.Name != "" {
		return u.Name
	}
	return u.Username
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
	mask := i.secretMasker(server.ID())
	// 原样转发为二进制帧：日志/输出是原始字节流(见 servercore.readPipe)，用二进制帧发送，
	// 避免文本帧在多字节字符(如中文)跨帧边界处被 UTF-8 校验损坏；前端用流式解码还原。
	// 密钥变量的值在发送前替换为占位符。
	sendText := func(text string) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		return websocket.Message.Send(conn, []byte(mask(text)))
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
		Id:            data.ID,
		Name:          data.Name,
		IsSystem:      data.IsSystem,
		IsEnable:      data.IsEnable,
		Description:   data.Description,
		StartCommand:  data.StartCommand,
		StopCommand:   data.StopCommand,
		QuickCommands: data.QuickCommands,
		CreateTime:    timestamppb.New(data.CreateTime),
		UpdateTime:    timestamppb.New(data.UpdateTime),
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
		AutoRestart:  item.AutoRestart,
		StopTimeout:  item.StopTimeout,
		Env:          toInstanceEnvVars(item),
		WsPath:       InstanceWsPath,
		UpdateTime:   timestamppb.New(item.UpdateTime),
		AbsolutePath: server.Dir(),
		Maintenance:  item.Maintenance,
	}
	if item.LastStopTime != nil {
		info.StopTime = timestamppb.New(*item.LastStopTime)
	}
	if item.MaintenanceTime != nil {
		info.MaintenanceTime = timestamppb.New(*item.MaintenanceTime)
	}
	if server.Running() {
		info.Status = v1.InstanceStatus_INSTANCE_STATUS_RUNNING
		info.Pid = int32(server.PID())
		if startTime, ok := server.StartTime(); ok {
			info.StartTime = timestamppb.New(startTime)
		}
	} else if item.Maintenance {
		info.Status = v1.InstanceStatus_INSTANCE_STATUS_MAINTENANCE
	}
	if t := item.Edges.Type; t != nil {
		info.Type = t.Name
		info.TypeId = t.ID
		info.QuickCommands = t.QuickCommands
	}
	if u := item.Edges.User; u != nil {
		info.UserId = u.ID
		info.UserName = displayUserName(u)
	}
	return info
}

// toInstanceEnvVars 输出环境变量；密钥变量只给出变量名，不返回值。
func toInstanceEnvVars(item *gen.Instance) []*v1.InstanceEnvVar {
	secret := make(map[string]struct{}, len(item.SecretEnvKeys))
	for _, key := range item.SecretEnvKeys {
		secret[key] = struct{}{}
	}
	out := make([]*v1.InstanceEnvVar, 0, len(item.Env))
	for _, line := range item.Env {
		key, value, _ := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := secret[key]; ok {
			out = append(out, &v1.InstanceEnvVar{Key: key, IsSecret: true})
			continue
		}
		out = append(out, &v1.InstanceEnvVar{Key: key, Value: value})
	}
	return out
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
	// 只为当前页的目录统计直接子项数。
	file.FillChildCounts(ctx, store, pages)

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
	seed := file.NewChunkedUpload(req.Hash, realPath, req.FileName, req.FileSize, true)
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
