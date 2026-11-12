package biz

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	gcpu "github.com/shirou/gopsutil/v4/cpu"
	gmem "github.com/shirou/gopsutil/v4/mem"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	entgameserver "momoko/internal/data/ent/gen/gameserver"
	entschema "momoko/internal/data/ent/schema"
	"momoko/pkg/auth"
	"momoko/pkg/jsondoc"
	"momoko/pkg/netprobe"
	"momoko/pkg/response"
	"momoko/pkg/servercore"
	"momoko/pkg/task"
)

// gameHostingDir 托管时 Momoko 在服务端目录里使用的子目录（备份、上传的构建、配置历史）。
const gameHostingDir = ".momoko"

const (
	// GameServerConsoleWsPath 托管进程的控制台（WebSocket，?id=<服务器>&role=main|resource），权限 game:server。
	GameServerConsoleWsPath = "/api/v1/game/server-console/ws"
	// GameServerEventsPrefix 服务器实时动态（SSE，转发 MGP /events）：/api/v1/game/server-events/{id}，权限 game:server。
	GameServerEventsPrefix = "/api/v1/game/server-events/"
)

var (
	ErrGameNotHosted       = response.BadRequest(400, "这台服务器不是托管模式")
	ErrGameProcessMissing  = response.BadRequest(400, "托管进程（应用实例）已被删除，请在托管选项里重新创建")
	ErrGameHostingBusy     = response.BadRequest(409, "这台服务器正在执行其他托管任务")
	ErrGameSettingsChanged = response.BadRequest(409, "配置已被其他人修改，请刷新后重试")
	ErrGameHostedDirFailed = response.BadRequest(400, "服务端目录未通过预检")
)

// gameHostingOptions 托管选项（存于 game_server.options）。
type gameHostingOptions struct {
	ConfigFile     string           `json:"configFile"`
	AutoStart      bool             `json:"autoStart"`
	AutoRestart    bool             `json:"autoRestart"`
	ResourceServer bool             `json:"resourceServer"`
	ResourceListen string           `json:"resourceListen"`
	ResourceRoot   string           `json:"resourceRoot"`
	Backup         gameBackupPolicy `json:"backup"`
}

// gameBackupPolicy 定时备份。
type gameBackupPolicy struct {
	Enabled       bool       `json:"enabled"`
	IntervalHours int        `json:"intervalHours"`
	Keep          int        `json:"keep"`
	LastRun       *time.Time `json:"lastRun,omitempty"`
}

func hostingOptionsOf(s *gen.GameServer) *gameHostingOptions {
	o := &gameHostingOptions{}
	if s.Options != "" {
		_ = json.Unmarshal([]byte(s.Options), o)
	}
	if o.Backup.IntervalHours <= 0 {
		o.Backup.IntervalHours = 24
	}
	if o.Backup.Keep <= 0 {
		o.Backup.Keep = 7
	}
	return o
}

func (o *gameHostingOptions) encode() string {
	raw, _ := json.Marshal(o)
	return string(raw)
}

// gameHostAdapter 托管用例需要的适配器能力（学マス：gakumasHosting）。
type gameHostAdapter interface {
	GameHostingDriver
	detect(ctx context.Context, dir, configFile string, listens map[string]string, selfID string) (*v1.DetectGameServerDirResponse, error)
	configure(dir string, spec *v1.GameServerHostedSpec, keyID, secret string) (*gameHostLayout, error)
	loadLayout(dir, configFile string) (*gameHostLayout, error)
	fillLayout(l *gameHostLayout, doc *jsondoc.Object)
	processes(l *gameHostLayout, opts *gameHostingOptions) []gameHostProcess
	settingsSchema() []*v1.GameSettingsSection
	checkBuild(path string) *v1.GameServerBuildInfo
	resolveBuildPath(dir, path string) string
}

type cachedLayout struct {
	path   string
	mod    time.Time
	size   int64
	layout *gameHostLayout
}

// GameHostingUsecase 托管：服务端进程是本机的应用实例（复用控制台、自启动、崩溃重启），
// 在其上提供专用设置、更新、资源、备份回档、文件与日志。只支持本机（node_id 为空）。
type GameHostingUsecase struct {
	servers   *GameServerUsecase
	instances *InstanceUsecase
	tasks     *task.Manager
	users     UserRepo
	log       *log.Helper

	mu      sync.Mutex
	busy    map[string]string // 服务器 → 进行中的独占任务（更新 / 回档 / 备份）
	resBusy map[string]string // 服务器 → 进行中的资源任务
	pending map[string]string // 服务器/角色 → stopping / restarting（异步停止中）
	layouts map[string]*cachedLayout
}

// 托管任务类型（任务管理器里的 type）。
const (
	gameTaskBuild    = "game.server.build"
	gameTaskResource = "game.server.resource"
	gameTaskBackup   = "game.server.backup"
	gameTaskRestore  = "game.server.restore"
)

func NewGameHostingUsecase(servers *GameServerUsecase, instances *InstanceUsecase, tasks *task.Manager, users UserRepo, logger log.Logger) *GameHostingUsecase {
	h := &GameHostingUsecase{
		servers:   servers,
		instances: instances,
		tasks:     tasks,
		users:     users,
		log:       log.NewHelper(log.With(logger, "module", "biz/game-hosting")),
		busy:      map[string]string{},
		resBusy:   map[string]string{},
		pending:   map[string]string{},
		layouts:   map[string]*cachedLayout{},
	}
	// 学マス 的托管驱动注册到适配器上，适配器的接入方式与能力随之变为可托管。
	if a, ok := GameAdapterByID("gakumas"); ok {
		next := *a
		next.Hosting = &gakumasHosting{h: h}
		RegisterGameAdapter(&next)
	}
	ctx := context.Background()
	_ = tasks.Resume(ctx, gameTaskBuild, gameTaskResource, gameTaskBackup, gameTaskRestore)
	if err := servers.repo.FailInterrupted(ctx, "interrupted"); err != nil {
		h.log.Warnf("mark interrupted hosting records: %v", err)
	}
	go h.scheduleLoop()
	servers.SetHostedSync(h.syncFromConfig)
	return h
}

func hostAdapterOf(game string) (gameHostAdapter, bool) {
	a, ok := GameAdapterByID(game)
	if !ok || a.Hosting == nil {
		return nil, false
	}
	ha, ok := a.Hosting.(gameHostAdapter)
	return ha, ok
}

// actor 当前操作的管理员。
func (h *GameHostingUsecase) actor(ctx context.Context) (string, string) {
	a, ok := auth.FromContext(ctx)
	if !ok {
		return "", ""
	}
	if u, err := h.users.FindByID(ctx, a.UserID); err == nil {
		return a.UserID, displayUserName(u)
	}
	return a.UserID, ""
}

// history 写一条变更记录（失败只记日志）。
func (h *GameHostingUsecase) history(ctx context.Context, serverID, kind, action string, params map[string]string, result, message string) *gen.GameServerHistory {
	uid, name := h.actor(ctx)
	rec := &gen.GameServerHistory{
		ServerID:     serverID,
		Kind:         kind,
		Action:       action,
		Params:       params,
		Message:      message,
		OperatorID:   uid,
		OperatorName: name,
	}
	rec.Result = gameHistoryResult(result)
	if err := h.servers.repo.AddHistory(context.WithoutCancel(ctx), rec); err != nil {
		h.log.Warnf("add history of %s: %v", serverID, err)
	}
	return rec
}

func (h *GameHostingUsecase) finishHistory(rec *gen.GameServerHistory, result, message string, params map[string]string) {
	if rec == nil || rec.ID == "" {
		return
	}
	if err := h.servers.repo.FinishHistory(context.Background(), rec.ID, result, message, params); err != nil {
		h.log.Warnf("finish history %s: %v", rec.ID, err)
	}
}

// hosted 读取托管服务器、适配器与布局。
func (h *GameHostingUsecase) hosted(ctx context.Context, id string) (*gen.GameServer, gameHostAdapter, *gameHostLayout, *gameHostingOptions, error) {
	s, err := h.servers.Get(ctx, id)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if s.Mode != entgameserver.ModeHosted || s.InstallDir == "" {
		return nil, nil, nil, nil, ErrGameNotHosted
	}
	ha, ok := hostAdapterOf(s.Game)
	if !ok {
		return nil, nil, nil, nil, ErrGameAdapterModeLimit
	}
	opts := hostingOptionsOf(s)
	l, err := h.layoutOf(s, ha, opts)
	if err != nil {
		return nil, nil, nil, nil, response.BadRequest(400, "读取服务端配置失败："+err.Error())
	}
	return s, ha, l, opts, nil
}

// layoutOf 解析服务端配置（按文件修改时间缓存）。返回的布局只读，修改配置时另行解析。
func (h *GameHostingUsecase) layoutOf(s *gen.GameServer, ha gameHostAdapter, opts *gameHostingOptions) (*gameHostLayout, error) {
	path := filepath.Join(s.InstallDir, opts.ConfigFile)
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	h.mu.Lock()
	c, ok := h.layouts[s.ID]
	h.mu.Unlock()
	if ok && c.path == path && c.mod.Equal(st.ModTime()) && c.size == st.Size() {
		return c.layout, nil
	}
	l, err := ha.loadLayout(s.InstallDir, opts.ConfigFile)
	if err != nil {
		return nil, err
	}
	h.mu.Lock()
	h.layouts[s.ID] = &cachedLayout{path: path, mod: st.ModTime(), size: st.Size(), layout: l}
	h.mu.Unlock()
	return l, nil
}

func (h *GameHostingUsecase) forgetLayout(id string) {
	h.mu.Lock()
	delete(h.layouts, id)
	h.mu.Unlock()
}

// processState 某个角色的进程状态（只看内存里的进程管理器，不查库）。
func (h *GameHostingUsecase) processState(s *gen.GameServer, role string) string {
	for _, p := range s.Processes {
		if p.Role != role {
			continue
		}
		h.mu.Lock()
		pending := h.pending[s.ID+"/"+role]
		h.mu.Unlock()
		if pending != "" {
			return pending
		}
		if core, ok := h.instances.instance.Get(p.InstanceID); ok && core.Running() {
			return GameStateRunning
		}
		return GameStateStopped
	}
	return ""
}

func (h *GameHostingUsecase) serverRunning(ctx context.Context, id string) bool {
	s, err := h.servers.Get(ctx, id)
	if err != nil || s.Mode != entgameserver.ModeHosted {
		return false
	}
	return h.processState(s, "main") == GameStateRunning
}

// instanceCore 读取应用实例并确保进程管理器里有它。
func (h *GameHostingUsecase) instanceCore(ctx context.Context, id string) (*gen.Instance, *servercore.Server, error) {
	item, err := h.servers.repo.GetInstance(ctx, id)
	if gen.IsNotFound(err) {
		return nil, nil, ErrGameProcessMissing
	}
	if err != nil {
		return nil, nil, ErrSystem(err)
	}
	core, err := h.instances.ensureInstance(item)
	if err != nil {
		return nil, nil, err
	}
	return item, core, nil
}

func processInstanceID(s *gen.GameServer, role string) string {
	for _, p := range s.Processes {
		if p.Role == role {
			return p.InstanceID
		}
	}
	return ""
}

func roleExecutable(ha gameHostAdapter, role string) string {
	for _, r := range ha.Spec().Roles {
		if r.Key == role {
			return r.Executable
		}
	}
	return role
}

// ensureProcesses 按布局创建或更新托管进程的应用实例；不再需要的角色停止并删除。
func (h *GameHostingUsecase) ensureProcesses(ctx context.Context, s *gen.GameServer, ha gameHostAdapter, l *gameHostLayout, opts *gameHostingOptions, ownerID string) error {
	defs := ha.processes(l, opts)
	existing := map[string]string{}
	for _, p := range s.Processes {
		existing[p.Role] = p.InstanceID
	}
	// 同一服务端目录下遗留的托管进程（以前切换接入方式后留在应用列表里的）：按可执行文件认领，
	// 而不是再建一个新的；认领不了的在最后删除，避免应用列表里越堆越多。
	tracked := map[string]bool{}
	for _, id := range existing {
		tracked[id] = true
	}
	var strays []*gen.Instance
	if list, err := h.servers.repo.ListHostedInstances(ctx, s.InstallDir); err == nil {
		for _, it := range list {
			if !tracked[it.ID] && !h.instanceOfOtherServer(ctx, s.ID, it.ID) {
				strays = append(strays, it)
			}
		}
	}
	for _, d := range defs {
		if _, ok := existing[d.Role]; ok {
			continue
		}
		exe := roleExecutable(ha, d.Role)
		for i, it := range strays {
			if it != nil && strings.HasPrefix(strings.TrimSpace(it.StartCommand), exe) {
				existing[d.Role] = it.ID
				strays[i] = nil
				break
			}
		}
	}
	var out []entschema.GameServerProcess
	for _, d := range defs {
		name := s.Name + " · " + roleExecutable(ha, d.Role)
		stop := d.StopCommand
		timeout := int32(d.StopTimeout)
		dir := s.InstallDir
		if id, ok := existing[d.Role]; ok {
			delete(existing, d.Role)
			item, err := h.servers.repo.GetInstance(ctx, id)
			if err == nil {
				owner := ownerID
				if item.Edges.User != nil {
					owner = item.Edges.User.ID
				}
				cmd := d.Command
				req := &v1.UpdateInstanceRequest{
					Id:           id,
					Name:         &name,
					StartCommand: &cmd,
					StopCommand:  &stop,
					InstancePath: &dir,
					AutoStart:    &opts.AutoStart,
					AutoRestart:  &opts.AutoRestart,
					StopTimeout:  &timeout,
				}
				updated, err := h.instances.repo.UpdateInstance(ctx, item, req, owner, nil)
				if err != nil {
					return ErrSystem(err)
				}
				if _, err := h.instances.ensureInstance(updated); err != nil {
					return err
				}
				out = append(out, entschema.GameServerProcess{Role: d.Role, InstanceID: id})
				continue
			}
			if !gen.IsNotFound(err) {
				return ErrSystem(err)
			}
		}
		if ownerID == "" {
			return ErrTokenInvalid
		}
		info, err := h.instances.CreateInstance(ctx, &v1.CreateInstanceRequest{
			Name:         name,
			Tags:         "game",
			StartCommand: d.Command,
			InstancePath: dir,
			StopCommand:  stop,
			AutoStart:    opts.AutoStart,
			AutoRestart:  opts.AutoRestart,
			StopTimeout:  timeout,
		}, ownerID)
		if err != nil {
			return err
		}
		out = append(out, entschema.GameServerProcess{Role: d.Role, InstanceID: info.Id})
	}
	for _, id := range existing {
		h.removeInstance(ctx, id)
	}
	for _, it := range strays {
		if it != nil {
			h.removeInstance(ctx, it.ID)
		}
	}
	s.Processes = out
	return nil
}

// removeInstance 停止并删除一个托管进程的应用实例。
func (h *GameHostingUsecase) removeInstance(ctx context.Context, id string) {
	item, err := h.servers.repo.GetInstance(ctx, id)
	if err != nil {
		return
	}
	if core, ok := h.instances.instance.Get(id); ok && core.Running() {
		_ = core.Stop()
	}
	owner := ""
	if item.Edges.User != nil {
		owner = item.Edges.User.ID
	}
	if err := h.instances.DeleteInstance(ctx, owner, id); err != nil {
		h.log.Warnf("delete hosted instance %s: %v", id, err)
	}
}

// removeHostedInstances 停止并删除这台服务器的全部托管进程：记录在案的，以及同一服务端目录下遗留的托管进程
// （以前切换接入方式时没有删除、只解除了关联的那些）。
func (h *GameHostingUsecase) removeHostedInstances(ctx context.Context, s *gen.GameServer) {
	seen := map[string]bool{}
	for _, p := range s.Processes {
		seen[p.InstanceID] = true
		h.removeInstance(ctx, p.InstanceID)
	}
	if s.InstallDir == "" {
		return
	}
	stray, err := h.servers.repo.ListHostedInstances(ctx, s.InstallDir)
	if err != nil {
		h.log.Warnf("list hosted instances of %s: %v", s.ID, err)
		return
	}
	for _, it := range stray {
		if !seen[it.ID] && !h.instanceOfOtherServer(ctx, s.ID, it.ID) {
			h.removeInstance(ctx, it.ID)
		}
	}
}

// instanceOfOtherServer 这个应用实例是否是另一台服务器正在用的托管进程（同一目录被两台服务器托管时不误删）。
func (h *GameHostingUsecase) instanceOfOtherServer(ctx context.Context, selfID, instanceID string) bool {
	list, err := h.servers.repo.List(ctx)
	if err != nil {
		return true
	}
	for _, o := range list {
		if o.ID == selfID {
			continue
		}
		for _, p := range o.Processes {
			if p.InstanceID == instanceID {
				return true
			}
		}
	}
	return false
}

// Detect 识别服务端目录（接入前的预检）。
func (h *GameHostingUsecase) Detect(ctx context.Context, req *v1.DetectGameServerDirRequest) (*v1.DetectGameServerDirResponse, error) {
	ha, ok := hostAdapterOf(req.Game)
	if !ok {
		return nil, ErrGameAdapterModeLimit
	}
	out, err := ha.detect(ctx, req.InstallDir, req.ConfigFile, req.Listens, req.ServerId)
	if err != nil {
		return nil, err
	}
	out.HostName = LocalNodeInfo(ctx).Hostname
	return out, nil
}

// setupHosted 预检、写配置（监听、MGP 密钥）、创建进程，并把托管信息写到 s 上（不落库）。
func (h *GameHostingUsecase) setupHosted(ctx context.Context, s *gen.GameServer, ha gameHostAdapter, spec *v1.GameServerHostedSpec, ownerID string) error {
	if spec == nil {
		return ErrGameServerInvalid
	}
	det, err := ha.detect(ctx, spec.InstallDir, spec.ConfigFile, mergeListens(spec), s.ID)
	if err != nil {
		return err
	}
	if !det.Ok {
		for _, c := range det.Checks {
			if c.State == "fail" {
				return response.BadRequest(400, "服务端目录未通过预检："+c.Key)
			}
		}
		return ErrGameHostedDirFailed
	}
	if spec.ResourceServer && !det.HasResourceServer {
		return response.BadRequest(400, "服务端目录里没有资源服务器可执行文件")
	}
	opts := hostingOptionsOf(s)
	opts.ConfigFile = det.ConfigFile
	opts.AutoStart, opts.AutoRestart = spec.AutoStart, spec.AutoRestart
	opts.ResourceServer = spec.ResourceServer
	if opts.ResourceServer {
		opts.ResourceListen = firstNonEmpty(spec.ResourceListen, det.Listens["resource"])
		opts.ResourceRoot = firstNonEmpty(spec.ResourceRoot, det.ResourceRoot)
		if !validListen(opts.ResourceListen) {
			return response.BadRequest(400, "资源服务器监听地址无效")
		}
		if opts.ResourceRoot != "" && !filepath.IsAbs(opts.ResourceRoot) {
			return response.BadRequest(400, "资源数据目录必须是绝对路径")
		}
	}
	keyID, secret := "", ""
	if spec.GenerateKey {
		// 密钥由前端生成（与服务端 mgp keygen 同格式），这里只校验后写入配置。
		if keyID, secret, err = ValidateMGPKey(spec.KeyId, spec.Secret); err != nil {
			return err
		}
	}
	// 资源服务器的对外地址写进服务端配置的 resource_server_url（以配置为准，Momoko 不另存）
	origin := strings.TrimRight(firstNonEmpty(spec.ResourceOrigin, det.ResourceOrigin), "/")
	if opts.ResourceServer && origin != "" && !validOrigin(origin) {
		return response.BadRequest(400, "资源服务器 origin 无效")
	}
	cfg := &v1.GameServerHostedSpec{
		ConfigFile:     det.ConfigFile,
		ConfigTemplate: det.ConfigTemplate,
		Listens:        spec.Listens,
		ResourceServer: opts.ResourceServer,
		ResourceOrigin: origin,
	}
	l, err := ha.configure(det.InstallDir, cfg, keyID, secret)
	if err != nil {
		return response.BadRequest(400, "写入服务端配置失败："+err.Error())
	}
	s.Mode = entgameserver.ModeHosted
	s.InstallDir = det.InstallDir
	s.NodeID = ""
	if err := h.applyLayoutMGP(s, l, keyID != ""); err != nil {
		return err
	}
	if err := h.ensureProcesses(ctx, s, ha, l, opts, ownerID); err != nil {
		return err
	}
	s.Options = opts.encode()
	h.forgetLayout(s.ID)
	return nil
}

func mergeListens(spec *v1.GameServerHostedSpec) map[string]string {
	out := map[string]string{}
	for k, v := range spec.Listens {
		out[k] = v
	}
	if spec.ResourceListen != "" {
		out["resource"] = spec.ResourceListen
	}
	return out
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// syncFromConfig 托管服务器的接入信息以服务端配置文件为准：不管配置是在专用设置、文件管理还是直接在磁盘上改的，
// 读取服务器时都按当前配置刷新 MGP 地址与密钥；有变化才写库并清掉客户端缓存。配置读不了时保持原样。
func (h *GameHostingUsecase) syncFromConfig(ctx context.Context, s *gen.GameServer) {
	if s.InstallDir == "" {
		return
	}
	ha, ok := hostAdapterOf(s.Game)
	if !ok {
		return
	}
	l, err := h.layoutOf(s, ha, hostingOptionsOf(s))
	if err != nil {
		return
	}
	endpoint, keyID, secret := "", "", ""
	if l.MGPKeyID != "" && l.MGPSecret != "" {
		endpoint, keyID, secret = l.mgpEndpoint(), l.MGPKeyID, l.MGPSecret
	}
	current := ""
	if s.Secret != "" {
		current, _ = gameBox().Decrypt(s.Secret)
	}
	if endpoint == s.MgpEndpoint && keyID == s.KeyID && secret == current {
		return
	}
	keyChanged := keyID != "" && (keyID != s.KeyID || secret != current)
	if err := h.applyLayoutMGP(s, l, keyChanged); err != nil {
		return
	}
	if _, err := h.servers.repo.Update(ctx, s); err != nil {
		h.log.Warnf("sync %s from config: %v", s.ID, err)
	}
	h.servers.forget(s.ID)
}

// applyLayoutMGP 用服务端配置里的 MGP 监听与密钥更新接入信息（托管服务器的 MGP 以配置为准）。
func (h *GameHostingUsecase) applyLayoutMGP(s *gen.GameServer, l *gameHostLayout, rotated bool) error {
	if l.MGPKeyID == "" || l.MGPSecret == "" {
		s.MgpEndpoint, s.KeyID, s.Secret = "", "", ""
		return nil
	}
	enc, err := EncryptGameSecret(l.MGPSecret)
	if err != nil {
		return ErrSystem(err)
	}
	s.MgpEndpoint, s.KeyID, s.Secret = l.mgpEndpoint(), l.MGPKeyID, enc
	if rotated {
		now := time.Now()
		s.KeyRotatedAt = &now
	}
	return nil
}

// CreateHosted 托管接入：预检 → 写配置 → 创建进程 → 落库 → 按需启动。
func (h *GameHostingUsecase) CreateHosted(ctx context.Context, req *v1.CreateGameServerRequest) (*gen.GameServer, error) {
	adapter, ok := GameAdapterByID(req.Game)
	if !ok {
		return nil, ErrGameAdapterNotFound
	}
	if !adapter.SupportsMode("hosted") {
		return nil, ErrGameAdapterModeLimit
	}
	ha, _ := hostAdapterOf(req.Game)
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, response.BadRequest(400, "名称不能为空")
	}
	ownerID, _ := h.actor(ctx)
	if ownerID == "" {
		return nil, ErrTokenInvalid
	}
	s := &gen.GameServer{
		ID:     uuid.NewString(),
		Game:   req.Game,
		Name:   name,
		Tags:   normalizeTags(req.Tags),
		Mode:   entgameserver.ModeHosted,
		Cover:  req.Cover,
		Remark: req.Remark,
	}
	if err := h.setupHosted(ctx, s, ha, req.Hosted, ownerID); err != nil {
		return nil, err
	}
	created, err := h.servers.repo.Create(ctx, s)
	if err != nil {
		for _, p := range s.Processes {
			h.removeInstance(ctx, p.InstanceID)
		}
		return nil, ErrSystem(err)
	}
	h.history(ctx, created.ID, "mode", "create", map[string]string{"mode": "hosted", "dir": created.InstallDir}, "success", "")
	if req.Hosted.Start {
		if err := h.startRoles(ctx, created, ""); err != nil {
			h.history(ctx, created.ID, "process", "start", nil, "failed", err.Error())
		}
	}
	return created, nil
}

// SwitchMode 切换接入方式。
func (h *GameHostingUsecase) SwitchMode(ctx context.Context, req *v1.SwitchGameServerModeRequest) (*gen.GameServer, error) {
	s, err := h.servers.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	adapter, ok := GameAdapterByID(s.Game)
	if !ok || !adapter.SupportsMode(req.Mode) {
		return nil, ErrGameAdapterModeLimit
	}
	from := string(s.Mode)
	switch req.Mode {
	case "remote":
		if s.Mode != entgameserver.ModeHosted {
			return nil, ErrGameServerInvalid
		}
		if h.isBusy(s.ID) {
			return nil, ErrGameHostingBusy
		}
		if req.MgpEndpoint != "" {
			if err := h.servers.applyMGP(s, req.MgpEndpoint, req.KeyId, req.Secret); err != nil {
				return nil, err
			}
		}
		if !MGPConfigured(s) {
			return nil, response.BadRequest(400, "仅 GM 需要 MGP 地址与密钥")
		}
		// 托管进程属于这台服务器：切走时一并停止并删除（包括以前遗留在应用列表里的同目录托管进程）
		h.removeHostedInstances(ctx, s)
		s.Mode = entgameserver.ModeRemote
		s.Processes = nil
		s.InstallDir = ""
		s.Options = ""
	case "hosted":
		if s.Mode != entgameserver.ModeRemote {
			return nil, ErrGameServerInvalid
		}
		ha, _ := hostAdapterOf(s.Game)
		ownerID, _ := h.actor(ctx)
		if ownerID == "" {
			return nil, ErrTokenInvalid
		}
		if err := h.setupHosted(ctx, s, ha, req.Hosted, ownerID); err != nil {
			return nil, err
		}
	default:
		return nil, ErrGameServerInvalid
	}
	updated, err := h.servers.repo.Update(ctx, s)
	if err != nil {
		return nil, ErrSystem(err)
	}
	h.servers.forget(s.ID)
	h.forgetLayout(s.ID)
	h.history(ctx, s.ID, "mode", "switch", map[string]string{"from": from, "to": req.Mode}, "success", "")
	if req.Mode == "hosted" && req.Hosted != nil && req.Hosted.Start {
		if err := h.startRoles(ctx, updated, ""); err != nil {
			h.history(ctx, s.ID, "process", "start", nil, "failed", err.Error())
		}
	}
	return updated, nil
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// RotateKey 轮换托管服务器的 MGP 密钥：写入服务端配置，Momoko 同步保存；运行中按需重启使其生效。
func (h *GameHostingUsecase) RotateKey(ctx context.Context, req *v1.RotateGameServerKeyRequest) (*v1.RotateGameServerKeyResponse, *gen.GameServer, error) {
	s, ha, _, opts, err := h.hosted(ctx, req.Id)
	if err != nil {
		return nil, nil, err
	}
	if h.isBusy(s.ID) {
		return nil, nil, ErrGameHostingBusy
	}
	// 新密钥由前端生成，这里只校验后写入配置。
	keyID, secret, err := ValidateMGPKey(req.KeyId, req.Secret)
	if err != nil {
		return nil, nil, err
	}
	l, err := ha.loadLayout(s.InstallDir, opts.ConfigFile)
	if err != nil {
		return nil, nil, response.BadRequest(400, "读取服务端配置失败："+err.Error())
	}
	if err := l.Doc.Assign("mgp.key_id", keyID); err != nil {
		return nil, nil, ErrSystem(err)
	}
	if err := l.Doc.Assign("mgp.secret", secret); err != nil {
		return nil, nil, ErrSystem(err)
	}
	if err := writeConfigDoc(s.InstallDir, l.ConfigPath, l.Doc); err != nil {
		return nil, nil, ErrSystem(err)
	}
	ha.fillLayout(l, l.Doc)
	if err := h.applyLayoutMGP(s, l, true); err != nil {
		return nil, nil, err
	}
	updated, err := h.servers.repo.Update(ctx, s)
	if err != nil {
		return nil, nil, ErrSystem(err)
	}
	h.servers.forget(s.ID)
	h.forgetLayout(s.ID)
	out := &v1.RotateGameServerKeyResponse{KeyId: keyID}
	if h.processState(updated, "main") == GameStateRunning {
		if req.Restart {
			h.restartAsync(ctx, updated, "main")
			out.Restarted = true
		} else {
			out.PendingRestart = true
		}
	}
	h.history(ctx, s.ID, "key", "rotate", map[string]string{"keyId": keyID, "restarted": boolStr(out.Restarted)}, "success", "")
	return out, updated, nil
}

// startRoles 启动进程：资源服务器先于游戏服务端（服务端的状态里会探测资源服务器）。
func (h *GameHostingUsecase) startRoles(ctx context.Context, s *gen.GameServer, role string) error {
	roles := []string{"resource", "main"}
	if role != "" {
		roles = []string{role}
	}
	for _, r := range roles {
		id := processInstanceID(s, r)
		if id == "" {
			if role != "" {
				return ErrGameServerInvalid
			}
			continue
		}
		item, core, err := h.instanceCore(ctx, id)
		if err != nil {
			// 启动全部进程时跳过已被删除的应用实例（详情页会提示重新创建）
			if role == "" && errors.Is(err, ErrGameProcessMissing) {
				continue
			}
			return err
		}
		if item.Maintenance {
			return ErrInstanceMaintenance
		}
		if core.Running() {
			continue
		}
		if err := ensureWorkDir(core.Dir()); err != nil {
			return ErrSystem(err)
		}
		if err := core.Start(); err != nil {
			return response.BadRequest(500, err.Error())
		}
	}
	return nil
}

// stopRoles 停止进程（游戏服务端先停，停服时它要保存全部玩家）。阻塞直到停止。
func (h *GameHostingUsecase) stopRoles(ctx context.Context, s *gen.GameServer, role string, force bool) error {
	roles := []string{"main", "resource"}
	if role != "" {
		roles = []string{role}
	}
	for _, r := range roles {
		id := processInstanceID(s, r)
		if id == "" {
			continue
		}
		_, core, err := h.instanceCore(ctx, id)
		if err != nil {
			if errors.Is(err, ErrGameProcessMissing) {
				continue
			}
			return err
		}
		if !core.Running() {
			continue
		}
		if force {
			err = core.ForceStop()
		} else {
			err = core.Stop()
		}
		if err != nil {
			return response.BadRequest(500, err.Error())
		}
	}
	return nil
}

func (h *GameHostingUsecase) setPending(serverID, role, state string) {
	h.mu.Lock()
	if state == "" {
		delete(h.pending, serverID+"/"+role)
	} else {
		h.pending[serverID+"/"+role] = state
	}
	h.mu.Unlock()
}

// stopAsync / restartAsync 停止可能要等服务端保存全部玩家（超过请求超时），放到后台执行。
func (h *GameHostingUsecase) stopAsync(ctx context.Context, s *gen.GameServer, role string, force bool) {
	roles := []string{role}
	if role == "" {
		roles = []string{"main", "resource"}
	}
	for _, r := range roles {
		if processInstanceID(s, r) != "" {
			h.setPending(s.ID, r, "stopping")
		}
	}
	bg := context.WithoutCancel(ctx)
	go func() {
		defer func() {
			for _, r := range roles {
				h.setPending(s.ID, r, "")
			}
		}()
		if err := h.stopRoles(bg, s, role, force); err != nil {
			h.history(bg, s.ID, "process", "stop", map[string]string{"role": role}, "failed", err.Error())
		}
	}()
}

func (h *GameHostingUsecase) restartAsync(ctx context.Context, s *gen.GameServer, role string) {
	roles := []string{role}
	if role == "" {
		roles = []string{"main", "resource"}
	}
	for _, r := range roles {
		if processInstanceID(s, r) != "" {
			h.setPending(s.ID, r, "restarting")
		}
	}
	bg := context.WithoutCancel(ctx)
	go func() {
		defer func() {
			for _, r := range roles {
				h.setPending(s.ID, r, "")
			}
		}()
		if err := h.stopRoles(bg, s, role, false); err != nil {
			h.history(bg, s.ID, "process", "restart", map[string]string{"role": role}, "failed", err.Error())
			return
		}
		for _, r := range roles {
			h.setPending(s.ID, r, "starting")
		}
		if err := h.startRoles(bg, s, role); err != nil {
			h.history(bg, s.ID, "process", "restart", map[string]string{"role": role}, "failed", err.Error())
		}
	}()
}

// Control 启动 / 停止 / 重启托管进程。
func (h *GameHostingUsecase) Control(ctx context.Context, req *v1.ControlGameServerRequest) (*gen.GameServer, error) {
	s, _, _, _, err := h.hosted(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if req.Role != "" && processInstanceID(s, req.Role) == "" {
		return nil, ErrGameServerInvalid
	}
	if h.isBusy(s.ID) {
		return nil, ErrGameHostingBusy
	}
	switch req.Action {
	case "start":
		if err := h.startRoles(ctx, s, req.Role); err != nil {
			return nil, err
		}
	case "stop":
		h.stopAsync(ctx, s, req.Role, req.Force)
	case "restart":
		h.restartAsync(ctx, s, req.Role)
	default:
		return nil, ErrGameServerInvalid
	}
	return s, nil
}

// UpdateHosting 修改托管选项，并据此更新（或重新创建缺失的）进程。
func (h *GameHostingUsecase) UpdateHosting(ctx context.Context, req *v1.UpdateGameServerHostingRequest) (*gen.GameServer, error) {
	s, ha, l, opts, err := h.hosted(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if h.isBusy(s.ID) {
		return nil, ErrGameHostingBusy
	}
	opts.AutoStart, opts.AutoRestart = req.AutoStart, req.AutoRestart
	opts.ResourceServer = req.ResourceServer
	if opts.ResourceServer {
		if _, err := os.Stat(l.ResExe); err != nil {
			return nil, response.BadRequest(400, "服务端目录里没有资源服务器可执行文件")
		}
		opts.ResourceListen = firstNonEmpty(req.ResourceListen, opts.ResourceListen, gakumasDefaultResourceListen)
		opts.ResourceRoot = firstNonEmpty(req.ResourceRoot, opts.ResourceRoot, gakumasResourceRoot(s.InstallDir))
		if !validListen(opts.ResourceListen) {
			return nil, ErrGameServerInvalid
		}
		if opts.ResourceRoot != "" && !filepath.IsAbs(opts.ResourceRoot) {
			return nil, response.BadRequest(400, "资源数据目录必须是绝对路径")
		}
		// 对外地址在服务端配置里（resource_server_url）：改了就写回配置
		if origin := strings.TrimRight(strings.TrimSpace(req.ResourceOrigin), "/"); origin != "" && origin != l.ResOrigin {
			if !validOrigin(origin) {
				return nil, response.BadRequest(400, "资源服务器 origin 无效")
			}
			nl, err := ha.loadLayout(s.InstallDir, opts.ConfigFile)
			if err != nil {
				return nil, response.BadRequest(400, "读取服务端配置失败："+err.Error())
			}
			if err := nl.Doc.Assign("resource_server_url", origin); err != nil {
				return nil, ErrSystem(err)
			}
			if err := writeConfigDoc(s.InstallDir, nl.ConfigPath, nl.Doc); err != nil {
				return nil, ErrSystem(err)
			}
			ha.fillLayout(nl, nl.Doc)
			h.forgetLayout(s.ID)
			l = nl
		}
	}
	ownerID, _ := h.actor(ctx)
	if err := h.ensureProcesses(ctx, s, ha, l, opts, ownerID); err != nil {
		return nil, err
	}
	s.Options = opts.encode()
	updated, err := h.servers.repo.Update(ctx, s)
	if err != nil {
		return nil, ErrSystem(err)
	}
	h.history(ctx, s.ID, "process", "options", map[string]string{
		"autoStart": boolStr(opts.AutoStart), "autoRestart": boolStr(opts.AutoRestart), "resourceServer": boolStr(opts.ResourceServer),
	}, "success", "")
	return updated, nil
}

// Delete 删除服务器；托管时同时停止并删除它的托管进程（应用实例）。
func (h *GameHostingUsecase) Delete(ctx context.Context, req *v1.DeleteGameServerRequest) error {
	s, err := h.servers.Get(ctx, req.Id)
	if err != nil {
		return err
	}
	if h.isBusy(s.ID) {
		return ErrGameHostingBusy
	}
	if s.Mode == entgameserver.ModeHosted {
		h.removeHostedInstances(ctx, s)
	}
	if err := h.servers.DeleteServer(ctx, s.ID); err != nil {
		return err
	}
	h.forgetLayout(s.ID)
	return nil
}

// describe 补充托管服务器详情。
func (h *GameHostingUsecase) describe(ctx context.Context, s *gen.GameServer, info *v1.GameServerInfo) {
	opts := hostingOptionsOf(s)
	info.Hosting = &v1.GameServerHostingOptions{
		ConfigFile:     opts.ConfigFile,
		AutoStart:      opts.AutoStart,
		AutoRestart:    opts.AutoRestart,
		ResourceServer: opts.ResourceServer,
		ResourceListen: opts.ResourceListen,
		ResourceRoot:   opts.ResourceRoot,
	}
	info.HostName = LocalNodeInfo(ctx).Hostname
	ha, ok := hostAdapterOf(s.Game)
	var l *gameHostLayout
	if ok {
		l, _ = h.layoutOf(s, ha, opts)
	}
	if l != nil {
		info.Hosting.StopTimeout = int32(l.stopTimeout())
		info.Hosting.ResourceOrigin = l.ResOrigin
	}
	info.Processes = nil
	for _, p := range s.Processes {
		pi := &v1.GameServerProcessInfo{Role: p.Role, InstanceId: p.InstanceID, State: "missing", Listens: map[string]string{}}
		if l != nil {
			for _, d := range ha.processes(l, opts) {
				if d.Role == p.Role {
					pi.Listens = d.Listens
				}
			}
		}
		if item, err := h.servers.repo.GetInstance(ctx, p.InstanceID); err == nil {
			pi.InstanceName = item.Name
			pi.Command = item.StartCommand
			pi.AutoStart, pi.AutoRestart, pi.StopTimeout = item.AutoStart, item.AutoRestart, item.StopTimeout
			if item.Edges.User != nil {
				pi.OwnerName = displayUserName(item.Edges.User)
			}
			if item.LastStopTime != nil {
				pi.StopTime = timestamppb.New(*item.LastStopTime)
			}
			pi.State = GameStateStopped
			if item.Maintenance {
				pi.State = GameStateMaintenance
			}
			if core, ok := h.instances.instance.Get(p.InstanceID); ok && core.Running() {
				pi.State = GameStateRunning
				pi.Pid = int32(core.PID())
				if t, ok := core.StartTime(); ok {
					pi.StartTime = timestamppb.New(t)
				}
			}
			if pending := h.pendingState(s.ID, p.Role); pending != "" {
				pi.State = pending
			}
		}
		info.Processes = append(info.Processes, pi)
	}
	for _, id := range h.runningTaskIDs(s.ID) {
		if t := h.taskInfo(id); t != nil {
			info.RunningTasks = append(info.RunningTasks, t)
		}
	}
}

func (h *GameHostingUsecase) pendingState(serverID, role string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.pending[serverID+"/"+role]
}

// ConsoleCore 返回托管进程的控制台（应用实例的伪终端）。
func (h *GameHostingUsecase) ConsoleCore(ctx context.Context, serverID, role string) (*servercore.Server, error) {
	s, err := h.servers.Get(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if s.Mode != entgameserver.ModeHosted {
		return nil, ErrGameNotHosted
	}
	if role == "" {
		role = "main"
	}
	id := processInstanceID(s, role)
	if id == "" {
		return nil, ErrGameServerInvalid
	}
	_, core, err := h.instanceCore(ctx, id)
	return core, err
}

// ClearConsole 清空托管进程控制台的输出记录（与应用实例的「清屏」相同：重新连接时不再回放旧输出）。
func (h *GameHostingUsecase) ClearConsole(ctx context.Context, serverID, role string) error {
	core, err := h.ConsoleCore(ctx, serverID, role)
	if err != nil {
		return err
	}
	core.ClearLogs()
	return nil
}

// Instances 托管进程所复用的应用实例用例（控制台 WebSocket）。
func (h *GameHostingUsecase) Instances() *InstanceUsecase { return h.instances }

// ---- 概览 ----

// gameStatusDetail 是 MGP status 里概览要用的部分。
type gameStatusDetail struct {
	Health []struct {
		Name      string  `json:"name"`
		OK        bool    `json:"ok"`
		Detail    string  `json:"detail"`
		LatencyMs float64 `json:"latencyMs"`
	} `json:"health"`
	Saves struct {
		Waiting  int `json:"waiting"`
		Capacity int `json:"capacity"`
	} `json:"saves"`
	Database struct {
		Migration int `json:"migration"`
	} `json:"database"`
}

// Overview 进程占用、实时 status 与状态探测。
func (h *GameHostingUsecase) Overview(ctx context.Context, id string) (*v1.GameServerOverview, error) {
	s, err := h.servers.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	out := &v1.GameServerOverview{}
	if cores, err := gcpu.CountsWithContext(ctx, true); err == nil {
		out.CpuCores = int32(cores)
	}
	if vm, err := gmem.VirtualMemoryWithContext(ctx); err == nil {
		out.MemoryTotal = vm.Total
	}
	var l *gameHostLayout
	var opts *gameHostingOptions
	if s.Mode == entgameserver.ModeHosted {
		if ha, ok := hostAdapterOf(s.Game); ok {
			opts = hostingOptionsOf(s)
			l, _ = h.layoutOf(s, ha, opts)
		}
		for _, p := range s.Processes {
			ps := &v1.GameServerProcessStats{Role: p.Role, State: GameStateStopped}
			core, ok := h.instances.instance.Get(p.InstanceID)
			if ok && core.Running() && core.PID() > 0 {
				ps.State = GameStateRunning
				ps.Pid = int32(core.PID())
				if t, ok := core.StartTime(); ok {
					ps.StartTime = timestamppb.New(t)
				}
				procs := instanceStats.processTree(ctx, p.InstanceID, int32(core.PID()))
				ps.ProcessCount = int32(len(procs))
				for _, pr := range procs {
					if pct, err := pr.PercentWithContext(ctx, 0); err == nil {
						ps.CpuPercent += pct
					}
					if mi, err := pr.MemoryInfoWithContext(ctx); err == nil && mi != nil {
						ps.MemoryBytes += mi.RSS
					}
				}
			} else if !ok {
				if _, err := h.servers.repo.GetInstance(ctx, p.InstanceID); gen.IsNotFound(err) {
					ps.State = "missing"
				}
			}
			if pending := h.pendingState(s.ID, p.Role); pending != "" {
				ps.State = pending
			}
			out.Processes = append(out.Processes, ps)
		}
	}
	// 托管：直接探测监听端口
	if l != nil {
		for _, k := range []string{"grpc", "http"} {
			probe := &v1.GameServerProbe{Key: k, Params: map[string]string{"listen": l.Listens[k]}}
			if d, err := netprobe.TCP(ctx, l.Listens[k], 2*time.Second); err == nil {
				probe.State, probe.LatencyMs = "ok", float64(d.Microseconds())/1000
			} else {
				probe.State, probe.Error = "fail", err.Error()
			}
			out.Probes = append(out.Probes, probe)
		}
	}
	// MGP status：在线、注册、数据库、资源服务器、存档队列
	if MGPConfigured(s) {
		start := time.Now()
		st, err := h.servers.Status(ctx, s)
		probe := &v1.GameServerProbe{Key: "mgp", Params: map[string]string{"endpoint": s.MgpEndpoint}}
		if err != nil {
			out.StatusError = err.Error()
			probe.State, probe.Error = "fail", err.Error()
			out.Probes = append(out.Probes, probe)
		} else {
			probe.State = "ok"
			probe.LatencyMs = float64(s.ProbeLatencyMs)
			if probe.LatencyMs == 0 {
				probe.LatencyMs = float64(time.Since(start).Microseconds()) / 1000
			}
			out.Probes = append(out.Probes, probe)
			out.Status = JSONValue(st.Raw)
			var detail gameStatusDetail
			_ = json.Unmarshal(st.Raw, &detail)
			for _, hc := range detail.Health {
				p := &v1.GameServerProbe{Key: hc.Name, LatencyMs: hc.LatencyMs, Params: map[string]string{"detail": hc.Detail}}
				if hc.Name == "database" {
					p.Params["migration"] = itoa(detail.Database.Migration)
				}
				if hc.OK {
					p.State = "ok"
				} else {
					p.State, p.Error = "fail", hc.Detail
				}
				out.Probes = append(out.Probes, p)
			}
			saves := &v1.GameServerProbe{Key: "saves", State: "ok", Params: map[string]string{"waiting": itoa(detail.Saves.Waiting), "capacity": itoa(detail.Saves.Capacity)}}
			if detail.Saves.Capacity > 0 && detail.Saves.Waiting*2 >= detail.Saves.Capacity {
				saves.State = "warn"
			}
			out.Probes = append(out.Probes, saves)
		}
	} else {
		out.Probes = append(out.Probes, &v1.GameServerProbe{Key: "mgp", State: "pending"})
	}
	// 托管的资源服务器：直接读 /health
	if opts != nil && opts.ResourceServer && opts.ResourceListen != "" {
		if v, _, err := netprobe.GetJSON(ctx, "http://"+netprobe.DialAddr(opts.ResourceListen)+"/health", 3*time.Second); err == nil {
			if pv := anyValue(v); pv != nil {
				out.ResourceHealth = pv
			}
		}
	}
	return out, nil
}

// ---- 任务 ----

func (h *GameHostingUsecase) isBusy(serverID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.busy[serverID] != ""
}

// acquire 占用服务器的独占任务位；返回释放函数。
func (h *GameHostingUsecase) acquire(serverID, taskID string, resource bool) (func(), bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	m := h.busy
	if resource {
		m = h.resBusy
	}
	if m[serverID] != "" {
		return nil, false
	}
	m[serverID] = taskID
	return func() {
		h.mu.Lock()
		if m[serverID] == taskID {
			delete(m, serverID)
		}
		h.mu.Unlock()
	}, true
}

func (h *GameHostingUsecase) runningTaskIDs(serverID string) []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []string
	if id := h.busy[serverID]; id != "" {
		out = append(out, id)
	}
	if id := h.resBusy[serverID]; id != "" {
		out = append(out, id)
	}
	return out
}
