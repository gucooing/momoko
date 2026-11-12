package biz

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	enthistory "momoko/internal/data/ent/gen/gameserverhistory"
	"momoko/pkg/gobuild"
	"momoko/pkg/netprobe"
	"momoko/pkg/response"
	"momoko/pkg/task"
)

// 托管任务的类型（任务管理器 type，前缀 game.server.）。
const (
	gameTaskResourceVerify = gameTaskResource + ".verify"
	gameTaskResourceUpdate = gameTaskResource + ".update"
)

// 任务步骤：key 给本页面本地化，标题给「任务与日志」面板。
var gameTaskSteps = []struct{ key, title string }{
	{"check", "检查新构建"},
	{"stop", "停止服务端"},
	{"backup", "备份"},
	{"replace", "替换可执行文件"},
	{"start", "启动服务端"},
	{"health", "健康检查"},
	{"rollback", "回滚"},
	{"snapshot", "数据库快照"},
	{"archive", "打包"},
	{"extract", "解压"},
	{"run", "运行资源服务器任务"},
	{"restart", "重启资源服务器"},
}

func stepTitle(key string) string {
	for _, s := range gameTaskSteps {
		if s.key == key {
			return s.title
		}
	}
	return key
}

func stepKey(title string) string {
	for _, s := range gameTaskSteps {
		if s.title == title {
			return s.key
		}
	}
	return title
}

func stepTitles(keys ...string) []string {
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = stepTitle(k)
	}
	return out
}

func gameHistoryResult(result string) enthistory.Result {
	switch result {
	case "failed":
		return enthistory.ResultFailed
	case "rolled_back":
		return enthistory.ResultRolledBack
	case "running":
		return enthistory.ResultRunning
	default:
		return enthistory.ResultSuccess
	}
}

func anyValue(v any) *structpb.Value {
	pv, err := structpb.NewValue(v)
	if err != nil {
		return nil
	}
	return pv
}

// hostTask 托管任务（一次性，不可续跑：重启后标记为中断）。
type hostTask struct {
	meta task.Meta
	run  func(ctx context.Context, r task.Reporter) error
}

func (t *hostTask) Meta() task.Meta                                { return t.meta }
func (t *hostTask) Run(ctx context.Context, r task.Reporter) error { return t.run(ctx, r) }
func (t *hostTask) Payload() any                                   { return nil }

// submitTask 提交一个托管任务；同一台服务器同一时间只有一个独占任务（资源任务单独计）。
func (h *GameHostingUsecase) submitTask(ctx context.Context, s *gen.GameServer, typ, title string, steps []string, resource bool, timeout time.Duration, fn func(ctx context.Context, r task.Reporter) error) (string, error) {
	id := uuid.NewString()
	release, ok := h.acquire(s.ID, id, resource)
	if !ok {
		return "", ErrGameHostingBusy
	}
	t := &hostTask{
		meta: task.Meta{
			ID:      id,
			Type:    typ,
			Kind:    task.KindOneShot,
			Title:   s.Name + " · " + title,
			Target:  s.ID,
			Source:  "game.server",
			Steps:   stepTitles(steps...),
			Timeout: timeout,
			Resume:  task.ResumeNone,
		},
		run: func(ctx context.Context, r task.Reporter) error {
			defer release()
			return fn(ctx, r)
		},
	}
	if _, err := h.tasks.Submit(ctx, t); err != nil {
		release()
		return "", ErrSystem(err)
	}
	return id, nil
}

// taskInfo 转为本页面的任务结构。
func (h *GameHostingUsecase) taskInfo(id string) *v1.GameServerTask {
	info, ok := h.tasks.Get(id)
	if !ok {
		return nil
	}
	out := &v1.GameServerTask{
		Id:         info.ID,
		Kind:       strings.TrimPrefix(info.Type, "game.server."),
		Status:     string(info.Status),
		Error:      info.Error,
		Total:      info.Total,
		Finished:   info.Finished,
		Message:    info.Message,
		CreateTime: timestamppb.New(info.CreateTime),
	}
	if info.StartTime != nil {
		out.StartTime = timestamppb.New(*info.StartTime)
	}
	if info.EndTime != nil {
		out.EndTime = timestamppb.New(*info.EndTime)
	}
	for _, st := range info.Steps {
		out.Steps = append(out.Steps, &v1.GameServerTaskStep{Key: stepKey(st.Title), State: string(st.State), Detail: st.Detail})
	}
	for _, lg := range info.Logs {
		out.Logs = append(out.Logs, &v1.GameServerTaskLog{Time: timestamppb.New(lg.Time), Level: string(lg.Level), Message: lg.Message})
	}
	if info.UserID != "" {
		if u, err := h.users.FindByID(context.Background(), info.UserID); err == nil {
			out.UserName = displayUserName(u)
		}
	}
	return out
}

// latestTask 某台服务器最近一次某类任务。
func (h *GameHostingUsecase) latestTask(ctx context.Context, serverID, typePrefix string) *v1.GameServerTask {
	infos, _, err := h.tasks.List(ctx, task.Filter{TypePrefix: typePrefix, Keywords: serverID}, 1, 1)
	if err != nil || len(infos) == 0 {
		return nil
	}
	return h.taskInfo(infos[0].ID)
}

// GetTask 查看托管任务。
func (h *GameHostingUsecase) GetTask(ctx context.Context, serverID, taskID string) (*v1.GameServerTask, error) {
	info, ok := h.tasks.Get(taskID)
	if !ok || info.Target != serverID || !strings.HasPrefix(info.Type, "game.server.") {
		return nil, response.BadRequest(404, "任务不存在")
	}
	return h.taskInfo(taskID), nil
}

// CancelTask 取消托管任务。
func (h *GameHostingUsecase) CancelTask(ctx context.Context, serverID, taskID string) (*v1.GameServerTask, error) {
	if _, err := h.GetTask(ctx, serverID, taskID); err != nil {
		return nil, err
	}
	_, name := h.actor(ctx)
	h.tasks.Cancel(taskID, name)
	return h.taskInfo(taskID), nil
}

// ListHistory 变更记录。
func (h *GameHostingUsecase) ListHistory(ctx context.Context, req *v1.ListGameServerHistoryRequest) (*v1.ListGameServerHistoryResponse, error) {
	if _, err := h.servers.Get(ctx, req.Id); err != nil {
		return nil, err
	}
	page, size := int(req.Page), int(req.PageSize)
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	items, total, err := h.servers.repo.ListHistory(ctx, req.Id, req.Kind, (page-1)*size, size)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := &v1.ListGameServerHistoryResponse{Total: int32(total)}
	for _, it := range items {
		out.Items = append(out.Items, &v1.GameServerHistoryEntry{
			Id:           it.ID,
			Kind:         it.Kind,
			Action:       it.Action,
			Params:       it.Params,
			Result:       string(it.Result),
			Message:      it.Message,
			OperatorName: it.OperatorName,
			CreateTime:   timestamppb.New(it.CreateTime),
		})
	}
	return out, nil
}

// ---- 服务端程序更新 ----

const gameUploadDir = gameHostingDir + "/uploads"

// GetBuild 当前服务端程序、运行中的版本、已上传的构建与最近的更新任务。
func (h *GameHostingUsecase) GetBuild(ctx context.Context, id string) (*v1.GameServerBuild, error) {
	s, ha, l, _, err := h.hosted(ctx, id)
	if err != nil {
		return nil, err
	}
	out := &v1.GameServerBuild{
		Current:   ha.checkBuild(l.Executable),
		UploadDir: gameUploadDir,
		Task:      h.latestTask(ctx, s.ID, gameTaskBuild),
	}
	if out.Current.Exists {
		if sum, err := gobuild.SHA256(l.Executable); err == nil {
			out.Current.Sha256 = sum
		}
	}
	if st := cachedStatusOf(s); st != nil && h.processState(s, "main") == GameStateRunning {
		out.RunningVersion = st.Version
		var detail gameStatusDetail
		if json.Unmarshal([]byte(s.Status), &detail) == nil {
			out.Migration = int32(detail.Database.Migration)
		}
	}
	if rec, err := h.servers.repo.LastHistory(ctx, s.ID, "server", "update", "success"); err == nil {
		out.LastUpdateTime = timestamppb.New(rec.CreateTime)
	}
	dir := filepath.Join(s.InstallDir, filepath.FromSlash(gameUploadDir))
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			if e.IsDir() || strings.HasSuffix(e.Name(), ".part") {
				continue
			}
			b := ha.checkBuild(filepath.Join(dir, e.Name()))
			b.Path = gameUploadDir + "/" + e.Name()
			out.Uploads = append(out.Uploads, b)
		}
	}
	return out, nil
}

// CheckBuild 检查候选构建。
func (h *GameHostingUsecase) CheckBuild(ctx context.Context, req *v1.CheckGameServerBuildRequest) (*v1.GameServerBuildInfo, error) {
	s, ha, _, _, err := h.hosted(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	path := ha.resolveBuildPath(s.InstallDir, req.Path)
	if path == "" {
		return nil, ErrGameServerInvalid
	}
	info := ha.checkBuild(path)
	info.Path = req.Path
	if info.Exists {
		if sum, err := gobuild.SHA256(path); err == nil {
			info.Sha256 = sum
		}
	}
	return info, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".momoko-tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

// waitHealthy 等待服务端启动后 MGP status 可用（没有接入 MGP 时等游戏端口可连）；进程退出即失败。
func (h *GameHostingUsecase) waitHealthy(ctx context.Context, s *gen.GameServer, l *gameHostLayout, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if h.processState(s, "main") != GameStateRunning {
			return "", errors.New("process_exited")
		}
		if MGPConfigured(s) {
			h.servers.forget(s.ID)
			st, err := h.servers.probe(ctx, s)
			if err == nil && st != nil {
				return st.Version, nil
			}
			last = err
		} else if _, err := netprobe.TCP(ctx, l.Listens["grpc"], 2*time.Second); err == nil {
			return "", nil
		} else {
			last = err
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	if last == nil {
		last = errors.New("timeout")
	}
	return "", fmt.Errorf("health_timeout: %v", last)
}

// UpdateBuild 更新服务端程序：检查 → 停止 → 备份（数据库、data/ 与配置）→ 替换 → 启动 → 健康检查；
// 启动或健康检查失败时停止新版本、换回旧程序并从刚才的备份恢复数据，再按原状态启动。
func (h *GameHostingUsecase) UpdateBuild(ctx context.Context, req *v1.UpdateGameServerBuildRequest) (string, error) {
	s, ha, l, opts, err := h.hosted(ctx, req.Id)
	if err != nil {
		return "", err
	}
	candidate := ha.resolveBuildPath(s.InstallDir, req.Path)
	next := ha.checkBuild(candidate)
	if !next.Valid {
		return "", response.BadRequest(400, "新构建无效："+next.Error)
	}
	if abs, _ := filepath.Abs(candidate); strings.EqualFold(abs, l.Executable) {
		return "", response.BadRequest(400, "新构建就是当前的服务端程序")
	}
	cur := ha.checkBuild(l.Executable)
	timeout := time.Duration(req.HealthTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	hist := h.history(ctx, s.ID, "server", "update", map[string]string{"from": cur.Version, "to": next.Version, "source": req.Path}, "running", "")
	steps := []string{"check", "stop", "backup", "replace", "start", "health"}
	return h.submitTask(ctx, s, gameTaskBuild, "更新服务端", steps, false, 0, func(ctx context.Context, r task.Reporter) error {
		params := map[string]string{"from": cur.Version, "to": next.Version, "source": req.Path}
		fail := func(code string, err error) error {
			h.finishHistory(hist, "failed", code+": "+err.Error(), params)
			return err
		}
		r.Step(stepTitle("check"), "")
		check := ha.checkBuild(candidate)
		if !check.Valid {
			return fail("check", errors.New(check.Error))
		}
		sum, _ := gobuild.SHA256(candidate)
		r.Log(task.LogInfo, "new build "+filepath.Base(candidate)+" version="+check.Version+" sha256="+sum)
		wasRunning := h.processState(s, "main") == GameStateRunning
		r.Step(stepTitle("stop"), "")
		h.setPending(s.ID, "main", "stopping")
		err := h.stopRoles(ctx, s, "main", false)
		h.setPending(s.ID, "main", "")
		if err != nil {
			return fail("stop", err)
		}
		r.Step(stepTitle("backup"), "")
		b, err := h.runBackup(ctx, s, l, opts, "update", "", r)
		if err != nil {
			if wasRunning {
				_ = h.startRoles(ctx, s, "main")
			}
			return fail("backup", err)
		}
		params["backupId"] = b.ID
		r.Step(stepTitle("replace"), "")
		keep := filepath.Join(s.InstallDir, gameHostingDir, "builds", time.Now().Format("20060102-150405")+"-"+filepath.Base(l.Executable))
		if err := copyFile(l.Executable, keep); err != nil {
			return fail("replace", err)
		}
		pruneOldFiles(filepath.Dir(keep), "", 3)
		if err := copyFile(candidate, l.Executable); err != nil {
			_ = copyFile(keep, l.Executable)
			if wasRunning {
				_ = h.startRoles(ctx, s, "main")
			}
			return fail("replace", err)
		}
		r.Log(task.LogInfo, "previous build kept at "+keep)
		rollback := func(cause error) error {
			r.Step(stepTitle("rollback"), cause.Error())
			r.Log(task.LogError, "rolling back: "+cause.Error())
			_ = h.stopRoles(context.WithoutCancel(ctx), s, "main", true)
			if err := copyFile(keep, l.Executable); err != nil {
				r.Log(task.LogError, "restore previous build: "+err.Error())
			}
			if err := h.extractBackup(context.WithoutCancel(ctx), s, l, b); err != nil {
				r.Log(task.LogError, "restore data: "+err.Error())
			}
			if wasRunning {
				if err := h.startRoles(context.WithoutCancel(ctx), s, "main"); err != nil {
					r.Log(task.LogError, "start previous build: "+err.Error())
				}
			}
			h.finishHistory(hist, "rolled_back", cause.Error(), params)
			h.pruneBackups(context.WithoutCancel(ctx), s, opts.Backup.Keep)
			return cause
		}
		r.Step(stepTitle("start"), "")
		if err := h.startRoles(ctx, s, "main"); err != nil {
			return rollback(err)
		}
		r.Step(stepTitle("health"), "")
		version, err := h.waitHealthy(ctx, s, l, timeout)
		if err != nil {
			return rollback(err)
		}
		if version != "" {
			params["to"] = version
		}
		r.Log(task.LogSuccess, "healthy, version="+version)
		if !wasRunning {
			r.Log(task.LogInfo, "server was stopped before the update and is running now")
		}
		uploads := filepath.Join(s.InstallDir, filepath.FromSlash(gameUploadDir))
		if abs, _ := filepath.Abs(candidate); strings.HasPrefix(abs, uploads+string(os.PathSeparator)) {
			_ = os.Remove(abs)
		}
		h.finishHistory(hist, "success", "", params)
		h.pruneBackups(context.WithoutCancel(ctx), s, opts.Backup.Keep)
		return nil
	})
}

// ---- 资源镜像（tools/resource-server）----

type resourceManifestInfo struct {
	Version   int
	Revision  int
	Objects   int
	Bytes     int64
	Masters   int
	Published bool
	PubVer    int
	PubRev    int
}

type resourceManifestCache struct {
	mod       time.Time
	activeMod time.Time
	info      resourceManifestInfo
}

var resourceManifests = struct {
	sync.Mutex
	m map[string]*resourceManifestCache
}{m: map[string]*resourceManifestCache{}}

// readResourceManifest 读取资源数据目录的候选清单（manifest.json）与已发布版本（active.json → versions/<sha>.json）。
// 清单有十几 MB，按两个文件的修改时间缓存。
func readResourceManifest(root string) (*resourceManifestInfo, error) {
	path := filepath.Join(root, "manifest.json")
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	var activeMod time.Time
	if ast, err := os.Stat(filepath.Join(root, "active.json")); err == nil {
		activeMod = ast.ModTime()
	}
	resourceManifests.Lock()
	c, ok := resourceManifests.m[root]
	resourceManifests.Unlock()
	if ok && c.mod.Equal(st.ModTime()) && c.activeMod.Equal(activeMod) {
		info := c.info
		return &info, nil
	}
	type manifestDoc struct {
		Version  int `json:"version"`
		Revision int `json:"revision"`
		Objects  []struct {
			Size int64 `json:"size"`
		} `json:"objects"`
		Masters []json.RawMessage `json:"masters"`
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m manifestDoc
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	info := resourceManifestInfo{Version: m.Version, Revision: m.Revision, Objects: len(m.Objects), Masters: len(m.Masters)}
	for _, o := range m.Objects {
		info.Bytes += o.Size
	}
	var pub struct {
		Manifest string `json:"manifest"`
	}
	if raw, err := os.ReadFile(filepath.Join(root, "active.json")); err == nil && json.Unmarshal(raw, &pub) == nil && pub.Manifest != "" && filepath.Base(pub.Manifest) == pub.Manifest {
		if f, err := os.Open(filepath.Join(root, "versions", pub.Manifest)); err == nil {
			var head struct {
				Version  int `json:"version"`
				Revision int `json:"revision"`
			}
			if json.NewDecoder(bufio.NewReader(f)).Decode(&head) == nil {
				info.Published = true
				info.PubVer, info.PubRev = head.Version, head.Revision
			}
			_ = f.Close()
		}
	}
	resourceManifests.Lock()
	resourceManifests.m[root] = &resourceManifestCache{mod: st.ModTime(), activeMod: activeMod, info: info}
	resourceManifests.Unlock()
	return &info, nil
}

// GetResource 资源镜像：清单、发布状态、下载进度、/health 与服务端加载的资源版本。
func (h *GameHostingUsecase) GetResource(ctx context.Context, id string) (*v1.GameServerResource, error) {
	s, _, l, opts, err := h.hosted(ctx, id)
	if err != nil {
		return nil, err
	}
	out := &v1.GameServerResource{
		Hosted: opts.ResourceServer,
		Root:   firstNonEmpty(opts.ResourceRoot, gakumasResourceRoot(s.InstallDir)),
		Task:   h.latestTask(ctx, s.ID, gameTaskResource),
	}
	if st, err := os.Stat(l.ResExe); err == nil && !st.IsDir() {
		out.Executable = true
	}
	if st := cachedStatusOf(s); st != nil {
		var detail struct {
			Resource json.RawMessage `json:"resource"`
		}
		if json.Unmarshal([]byte(s.Status), &detail) == nil && len(detail.Resource) > 0 {
			out.ServerResource = JSONValue(detail.Resource)
		}
	}
	if out.Root != "" {
		if m, err := readResourceManifest(out.Root); err == nil {
			out.Version, out.Revision, out.Objects, out.Bytes, out.Masters = int32(m.Version), int32(m.Revision), int32(m.Objects), m.Bytes, int32(m.Masters)
			out.Published, out.PublishedVersion, out.PublishedRevision = m.Published, int32(m.PubVer), int32(m.PubRev)
		} else {
			out.Error = err.Error()
		}
		if raw, err := os.ReadFile(filepath.Join(out.Root, "progress.json")); err == nil {
			out.Progress = JSONValue(raw)
		}
	}
	if opts.ResourceServer && opts.ResourceListen != "" {
		if v, _, err := netprobe.GetJSON(ctx, "http://"+netprobe.DialAddr(opts.ResourceListen)+"/health", 3*time.Second); err == nil {
			out.Health = anyValue(v)
		} else {
			out.HealthError = err.Error()
		}
	}
	return out, nil
}

// RunResourceTask 运行资源服务器的显式任务：verify 完整重校验本地对象；update 访问官方目录、下载全部对象并发布新版本
// （成功后重启托管的资源服务器，让它分发新版本）。
func (h *GameHostingUsecase) RunResourceTask(ctx context.Context, req *v1.RunGameServerResourceTaskRequest) (string, error) {
	s, _, l, opts, err := h.hosted(ctx, req.Id)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(l.ResExe); err != nil {
		return "", response.BadRequest(400, "服务端目录里没有资源服务器可执行文件")
	}
	root := firstNonEmpty(opts.ResourceRoot, gakumasResourceRoot(s.InstallDir))
	if root == "" {
		return "", response.BadRequest(400, "没有资源数据目录")
	}
	args := []string{}
	typ, title := "", ""
	switch req.Action {
	case "verify":
		args = append(args, "verify", "-root", root)
		typ, title = gameTaskResourceVerify, "校验资源"
	case "update":
		args = append(args, "update", "-root", root)
		if p := strings.TrimSpace(req.MasterMetadata); p != "" {
			if !filepath.IsAbs(p) {
				return "", response.BadRequest(400, "主数据元信息文件必须是绝对路径")
			}
			if _, err := os.Stat(p); err != nil {
				return "", response.BadRequest(400, "主数据元信息文件不存在")
			}
			args = append(args, "-master-metadata", p)
		}
		typ, title = gameTaskResourceUpdate, "更新资源"
	default:
		return "", ErrGameServerInvalid
	}
	hist := h.history(ctx, s.ID, "resource", req.Action, map[string]string{"root": root}, "running", "")
	steps := []string{"run"}
	if req.Action == "update" && opts.ResourceServer {
		steps = append(steps, "restart")
	}
	return h.submitTask(ctx, s, typ, title, steps, true, 0, func(ctx context.Context, r task.Reporter) error {
		r.Step(stepTitle("run"), strings.Join(args, " "))
		cmd := exec.CommandContext(ctx, l.ResExe, args...)
		cmd.Dir = s.InstallDir
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			h.finishHistory(hist, "failed", err.Error(), nil)
			return err
		}
		cmd.Stderr = cmd.Stdout
		if err := cmd.Start(); err != nil {
			h.finishHistory(hist, "failed", err.Error(), nil)
			return err
		}
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 64<<10), 1<<20)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			level := task.LogInfo
			if strings.Contains(line, "level=ERROR") || strings.Contains(line, "level=WARN") {
				level = task.LogWarn
			}
			r.Log(level, line)
		}
		if err := cmd.Wait(); err != nil {
			h.finishHistory(hist, "failed", err.Error(), nil)
			return err
		}
		params := map[string]string{"root": root}
		if m, err := readResourceManifest(root); err == nil {
			params["version"] = itoa(m.Version)
			params["revision"] = itoa(m.Revision)
			params["objects"] = itoa(m.Objects)
		}
		if req.Action == "update" && opts.ResourceServer && h.processState(s, "resource") == GameStateRunning {
			r.Step(stepTitle("restart"), "")
			if err := h.stopRoles(ctx, s, "resource", false); err == nil {
				if err := h.startRoles(ctx, s, "resource"); err != nil {
					r.Log(task.LogError, "start resource server: "+err.Error())
				}
			}
		}
		h.finishHistory(hist, "success", "", params)
		return nil
	})
}
