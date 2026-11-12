package biz

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	entgameserver "momoko/internal/data/ent/gen/gameserver"
	entbackup "momoko/internal/data/ent/gen/gameserverbackup"
	"momoko/pkg/backupzip"
	"momoko/pkg/pre"
	"momoko/pkg/response"
	"momoko/pkg/sqlitesnap"
	"momoko/pkg/task"
	"momoko/pkg/utils"
)

const gameBackupDir = gameHostingDir + "/backups"

// 备份文件里放在服务端目录之外的数据库的位置。
const gameBackupExternalDB = "_external/database/"

// backupPlan 备份范围（适配器的布局决定）：配置文件、数据目录、SQLite 数据库（一致性快照）。
type backupPlan struct {
	// Paths 相对服务端目录打包的文件或目录（目录以 / 结尾）
	Paths []string
	// DBPath SQLite 数据库的绝对路径（空表示不备份数据库文件）
	DBPath string
	// DBName 数据库在备份里的位置
	DBName string
	// Notes 说明码：database_external（PostgreSQL 数据库不在备份范围内）
	Notes []string
}

func relInside(base, path string) (string, bool) {
	rel, err := filepath.Rel(base, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

func planBackup(l *gameHostLayout) backupPlan {
	p := backupPlan{}
	if rel, ok := relInside(l.Dir, l.ConfigPath); ok {
		p.Paths = append(p.Paths, rel)
	}
	if rel, ok := relInside(l.Dir, l.DataDir); ok {
		p.Paths = append(p.Paths, rel+"/")
	}
	switch {
	case l.DBDriver == "sqlite" && l.DBPath != "":
		p.DBPath = l.DBPath
		if rel, ok := relInside(l.Dir, l.DBPath); ok {
			p.DBName = rel
		} else {
			p.DBName = gameBackupExternalDB + filepath.Base(l.DBPath)
		}
	case l.DBDriver != "sqlite":
		p.Notes = append(p.Notes, "database_external")
	}
	return p
}

// scope 记录在备份上的范围：打包的路径 + 数据库位置（外部数据库为 external:<绝对路径>）。
func (p backupPlan) scope() []string {
	out := append([]string(nil), p.Paths...)
	if p.DBPath != "" {
		if strings.HasPrefix(p.DBName, gameBackupExternalDB) {
			out = append(out, "external:"+p.DBPath)
		} else {
			out = append(out, p.DBName)
		}
	}
	return out
}

func toBackupInfo(b *gen.GameServerBackup) *v1.GameServerBackupInfo {
	out := &v1.GameServerBackupInfo{
		Id:           b.ID,
		Trigger:      string(b.Trigger),
		Scope:        b.Scope,
		Size:         b.Size,
		State:        string(b.State),
		Note:         b.Note,
		Error:        b.Error,
		OperatorName: b.OperatorName,
		CreateTime:   timestamppb.New(b.CreateTime),
		FileName:     filepath.Base(b.Path),
	}
	if b.Path != "" {
		if st, err := os.Stat(b.Path); err == nil && !st.IsDir() {
			out.Exists = true
		}
	}
	return out
}

// runBackup 做一次备份：SQLite 用 VACUUM INTO 做一致性快照（运行中也不阻塞写者），再与配置、数据目录一起打包。
// 自动备份的保留份数由调用方在流程结束后清理（回档时不能在解压前删掉要恢复的那份）。
func (h *GameHostingUsecase) runBackup(ctx context.Context, s *gen.GameServer, l *gameHostLayout, opts *gameHostingOptions, trigger, note string, r task.Reporter) (*gen.GameServerBackup, error) {
	plan := planBackup(l)
	uid, name := h.actor(ctx)
	b := &gen.GameServerBackup{
		ServerID:     s.ID,
		Trigger:      entbackup.Trigger(trigger),
		Scope:        plan.scope(),
		State:        entbackup.StateRunning,
		Note:         note,
		OperatorID:   uid,
		OperatorName: name,
	}
	b, err := h.servers.repo.CreateBackup(ctx, b)
	if err != nil {
		return nil, ErrSystem(err)
	}
	now := time.Now()
	dir := filepath.Join(s.InstallDir, filepath.FromSlash(gameBackupDir))
	b.Path = filepath.Join(dir, fmt.Sprintf("%s-%s-%s.zip", now.Format("20060102-150405"), trigger, b.ID[:8]))
	fail := func(err error) (*gen.GameServerBackup, error) {
		b.State, b.Error = entbackup.StateFailed, err.Error()
		_ = h.servers.repo.UpdateBackup(context.WithoutCancel(ctx), b)
		h.history(ctx, s.ID, "backup", "create", map[string]string{"backupId": b.ID, "trigger": trigger}, "failed", err.Error())
		return nil, err
	}
	zopt := backupzip.Options{}
	if plan.DBPath != "" {
		if r != nil {
			r.Step(stepTitle("snapshot"), "")
		}
		snap := filepath.Join(s.InstallDir, gameHostingDir, "tmp", b.ID+".sqlite")
		if err := sqlitesnap.Snapshot(ctx, plan.DBPath, snap); err != nil {
			return fail(fmt.Errorf("snapshot: %w", err))
		}
		defer os.Remove(snap)
		skip := map[string]bool{}
		if rel, ok := relInside(l.Dir, plan.DBPath); ok {
			skip[rel] = true
			for _, c := range sqlitesnap.Companions(plan.DBPath) {
				if crel, ok := relInside(l.Dir, c); ok {
					skip[crel] = true
				}
			}
		}
		zopt.Skip = func(rel string) bool { return skip[rel] }
		zopt.Extras = []backupzip.Extra{{Name: plan.DBName, Path: snap}}
	}
	if r != nil {
		r.Step(stepTitle("archive"), "")
		zopt.Progress = func(files int, bytes int64) { r.SetProgress(int64(files), 0, "") }
	}
	paths := make([]string, 0, len(plan.Paths))
	for _, p := range plan.Paths {
		paths = append(paths, strings.TrimSuffix(p, "/"))
	}
	files, size, err := backupzip.Create(ctx, s.InstallDir, paths, b.Path, zopt)
	if err != nil {
		return fail(err)
	}
	b.State, b.Size = entbackup.StateReady, size
	if err := h.servers.repo.UpdateBackup(context.WithoutCancel(ctx), b); err != nil {
		return nil, ErrSystem(err)
	}
	if r != nil {
		r.Log(task.LogSuccess, fmt.Sprintf("backup %s: %d files, %d bytes", filepath.Base(b.Path), files, size))
	}
	h.history(ctx, s.ID, "backup", "create", map[string]string{"backupId": b.ID, "trigger": trigger, "size": fmt.Sprint(size), "files": fmt.Sprint(files)}, "success", "")
	return b, nil
}

// pruneBackups 自动备份（定时、更新前、回档前）只保留最新 keep 份；手动备份不自动删除。
func (h *GameHostingUsecase) pruneBackups(ctx context.Context, s *gen.GameServer, keep int) {
	if keep <= 0 {
		return
	}
	items, _, err := h.servers.repo.ListBackups(ctx, s.ID, 0, 0)
	if err != nil {
		return
	}
	n := 0
	for _, b := range items {
		if b.Trigger == entbackup.TriggerManual || b.State != entbackup.StateReady {
			continue
		}
		n++
		if n > keep {
			h.removeBackup(ctx, s, b)
		}
	}
}

func (h *GameHostingUsecase) removeBackup(ctx context.Context, s *gen.GameServer, b *gen.GameServerBackup) {
	if b.Path != "" && h.backupPathAllowed(s, b.Path) {
		_ = os.Remove(b.Path)
	}
	if err := h.servers.repo.DeleteBackup(ctx, b.ID); err != nil {
		h.log.Warnf("delete backup %s: %v", b.ID, err)
	}
}

// backupPathAllowed 只删除 / 读取服务端目录 .momoko/backups 下的文件。
func (h *GameHostingUsecase) backupPathAllowed(s *gen.GameServer, path string) bool {
	dir := filepath.Join(s.InstallDir, filepath.FromSlash(gameBackupDir))
	_, ok := relInside(dir, path)
	return ok
}

// extractBackup 把备份解回服务端目录：数据目录先整体移开（解压成功后删除，失败时移回），
// SQLite 的 -wal / -shm 先删除，避免旧日志被重放到恢复的数据库上。
func (h *GameHostingUsecase) extractBackup(ctx context.Context, s *gen.GameServer, l *gameHostLayout, b *gen.GameServerBackup) error {
	if b.State != entbackup.StateReady || !h.backupPathAllowed(s, b.Path) {
		return response.BadRequest(400, "备份不可用")
	}
	if _, err := backupzip.Entries(b.Path); err != nil {
		return fmt.Errorf("backup archive: %w", err)
	}
	trash := filepath.Join(s.InstallDir, gameHostingDir, "tmp", "restore-"+time.Now().Format("20060102-150405"))
	type moved struct{ from, to string }
	var moves []moved
	undo := func() {
		for _, m := range moves {
			_ = os.RemoveAll(m.from)
			_ = os.Rename(m.to, m.from)
		}
	}
	var externalDB string
	for _, sc := range b.Scope {
		switch {
		case strings.HasPrefix(sc, "external:"):
			externalDB = strings.TrimPrefix(sc, "external:")
			for _, c := range sqlitesnap.Companions(externalDB) {
				_ = os.Remove(c)
			}
		case strings.HasSuffix(sc, "/"):
			src := filepath.Join(s.InstallDir, filepath.FromSlash(strings.TrimSuffix(sc, "/")))
			if _, err := os.Stat(src); err != nil {
				continue
			}
			dst := filepath.Join(trash, filepath.FromSlash(strings.TrimSuffix(sc, "/")))
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				undo()
				return err
			}
			if err := os.Rename(src, dst); err != nil {
				undo()
				return err
			}
			moves = append(moves, moved{from: src, to: dst})
		default:
			path := filepath.Join(s.InstallDir, filepath.FromSlash(sc))
			for _, c := range sqlitesnap.Companions(path) {
				_ = os.Remove(c)
			}
		}
	}
	if _, err := backupzip.Extract(ctx, b.Path, s.InstallDir); err != nil {
		undo()
		return err
	}
	if externalDB != "" {
		src := filepath.Join(s.InstallDir, filepath.FromSlash(gameBackupExternalDB+filepath.Base(externalDB)))
		if err := copyFile(src, externalDB); err != nil {
			undo()
			return err
		}
		_ = os.RemoveAll(filepath.Join(s.InstallDir, "_external"))
	}
	_ = os.RemoveAll(trash)
	h.forgetLayout(s.ID)
	return nil
}

// ListBackups 备份列表、范围与定时策略。
func (h *GameHostingUsecase) ListBackups(ctx context.Context, req *v1.ListGameServerBackupsRequest) (*v1.ListGameServerBackupsResponse, error) {
	s, _, l, opts, err := h.hosted(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	page, size := int(req.Page), int(req.PageSize)
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	items, total, err := h.servers.repo.ListBackups(ctx, s.ID, (page-1)*size, size)
	if err != nil {
		return nil, ErrSystem(err)
	}
	plan := planBackup(l)
	out := &v1.ListGameServerBackupsResponse{
		Total:  int32(total),
		Policy: h.policyInfo(ctx, s, opts),
		Scope:  plan.scope(),
		Notes:  plan.Notes,
		Dir:    gameBackupDir,
	}
	for _, b := range items {
		out.Items = append(out.Items, toBackupInfo(b))
	}
	if all, _, err := h.servers.repo.ListBackups(ctx, s.ID, 0, 0); err == nil {
		for _, b := range all {
			out.TotalSize += b.Size
		}
	}
	return out, nil
}

func (h *GameHostingUsecase) policyInfo(ctx context.Context, s *gen.GameServer, opts *gameHostingOptions) *v1.GameServerBackupPolicy {
	p := opts.Backup
	out := &v1.GameServerBackupPolicy{Enabled: p.Enabled, IntervalHours: int32(p.IntervalHours), Keep: int32(p.Keep)}
	if p.LastRun != nil {
		out.LastRunTime = timestamppb.New(*p.LastRun)
	}
	if p.Enabled {
		base := time.Now()
		if p.LastRun != nil {
			base = *p.LastRun
		}
		out.NextRunTime = timestamppb.New(base.Add(time.Duration(p.IntervalHours) * time.Hour))
	}
	return out
}

// CreateBackup 立即备份（后台任务）。
func (h *GameHostingUsecase) CreateBackup(ctx context.Context, req *v1.CreateGameServerBackupRequest) (string, error) {
	s, _, l, opts, err := h.hosted(ctx, req.Id)
	if err != nil {
		return "", err
	}
	note := strings.TrimSpace(req.Note)
	if len([]rune(note)) > 200 {
		note = string([]rune(note)[:200])
	}
	return h.submitTask(ctx, s, gameTaskBackup, "备份", []string{"snapshot", "archive"}, false, 0, func(ctx context.Context, r task.Reporter) error {
		_, err := h.runBackup(ctx, s, l, opts, "manual", note, r)
		return err
	})
}

// DeleteBackup 删除备份（文件与记录）。
func (h *GameHostingUsecase) DeleteBackup(ctx context.Context, req *v1.DeleteGameServerBackupRequest) error {
	s, err := h.servers.Get(ctx, req.Id)
	if err != nil {
		return err
	}
	b, err := h.servers.repo.GetBackup(ctx, req.BackupId)
	if err != nil || b.ServerID != s.ID {
		return response.BadRequest(404, "备份不存在")
	}
	if b.State == entbackup.StateRunning {
		return ErrGameHostingBusy
	}
	h.removeBackup(ctx, s, b)
	h.history(ctx, s.ID, "backup", "delete", map[string]string{"backupId": b.ID, "file": filepath.Base(b.Path)}, "success", "")
	return nil
}

// RestoreBackup 回档（后台任务）：停止 → 备份当前 → 解压 → 按原状态启动。
func (h *GameHostingUsecase) RestoreBackup(ctx context.Context, req *v1.RestoreGameServerBackupRequest) (string, error) {
	s, _, l, opts, err := h.hosted(ctx, req.Id)
	if err != nil {
		return "", err
	}
	b, err := h.servers.repo.GetBackup(ctx, req.BackupId)
	if err != nil || b.ServerID != s.ID {
		return "", response.BadRequest(404, "备份不存在")
	}
	if b.State != entbackup.StateReady {
		return "", response.BadRequest(400, "备份不可用")
	}
	hist := h.history(ctx, s.ID, "backup", "restore", map[string]string{"backupId": b.ID, "file": filepath.Base(b.Path)}, "running", "")
	return h.submitTask(ctx, s, gameTaskRestore, "回档", []string{"stop", "backup", "extract", "start"}, false, 0, func(ctx context.Context, r task.Reporter) error {
		params := map[string]string{"backupId": b.ID, "file": filepath.Base(b.Path)}
		wasRunning := h.processState(s, "main") == GameStateRunning
		r.Step(stepTitle("stop"), "")
		h.setPending(s.ID, "main", "stopping")
		err := h.stopRoles(ctx, s, "main", false)
		h.setPending(s.ID, "main", "")
		if err != nil {
			h.finishHistory(hist, "failed", err.Error(), params)
			return err
		}
		r.Step(stepTitle("backup"), "")
		cur, err := h.runBackup(ctx, s, l, opts, "restore", "", r)
		if err != nil {
			if wasRunning {
				_ = h.startRoles(ctx, s, "main")
			}
			h.finishHistory(hist, "failed", err.Error(), params)
			return err
		}
		params["previousBackupId"] = cur.ID
		r.Step(stepTitle("extract"), filepath.Base(b.Path))
		if err := h.extractBackup(ctx, s, l, b); err != nil {
			if wasRunning {
				_ = h.startRoles(context.WithoutCancel(ctx), s, "main")
			}
			h.finishHistory(hist, "failed", err.Error(), params)
			return err
		}
		if wasRunning {
			r.Step(stepTitle("start"), "")
			if err := h.startRoles(ctx, s, "main"); err != nil {
				h.finishHistory(hist, "failed", err.Error(), params)
				return err
			}
		}
		h.finishHistory(hist, "success", "", params)
		h.pruneBackups(context.WithoutCancel(ctx), s, opts.Backup.Keep)
		return nil
	})
}

// DownloadBackup 备份的预签名下载地址。
func (h *GameHostingUsecase) DownloadBackup(ctx context.Context, req *v1.DownloadGameServerBackupRequest) (string, error) {
	s, err := h.servers.Get(ctx, req.Id)
	if err != nil {
		return "", err
	}
	b, err := h.servers.repo.GetBackup(ctx, req.BackupId)
	if err != nil || b.ServerID != s.ID || !h.backupPathAllowed(s, b.Path) {
		return "", response.BadRequest(404, "备份不存在")
	}
	if st, err := os.Stat(b.Path); err != nil || st.IsDir() {
		return "", ErrFileNotExist
	}
	uid, _ := h.actor(ctx)
	sign, err := pre.NewFileSignInfo(utils.GenerateRandomString(10), b.Path, uid, time.Hour).Sign()
	if err != nil {
		return "", ErrSign
	}
	return fmt.Sprintf("%s?sign=%s", PreFileDownload, sign), nil
}

// UpdateBackupPolicy 定时备份与保留份数。
func (h *GameHostingUsecase) UpdateBackupPolicy(ctx context.Context, req *v1.UpdateGameServerBackupPolicyRequest) (*v1.GameServerBackupPolicy, error) {
	s, _, _, opts, err := h.hosted(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if req.IntervalHours < 1 || req.IntervalHours > 24*30 || req.Keep < 1 || req.Keep > 200 {
		return nil, ErrGameServerInvalid
	}
	opts.Backup.Enabled = req.Enabled
	opts.Backup.IntervalHours = int(req.IntervalHours)
	opts.Backup.Keep = int(req.Keep)
	s.Options = opts.encode()
	if _, err := h.servers.repo.Update(ctx, s); err != nil {
		return nil, ErrSystem(err)
	}
	h.history(ctx, s.ID, "backup", "policy", map[string]string{
		"enabled": boolStr(req.Enabled), "intervalHours": itoa(int(req.IntervalHours)), "keep": itoa(int(req.Keep)),
	}, "success", "")
	return h.policyInfo(ctx, s, opts), nil
}

// scheduleLoop 定时备份：每 5 分钟检查一次到期的托管服务器。
func (h *GameHostingUsecase) scheduleLoop() {
	time.Sleep(time.Minute)
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		h.runSchedules()
		<-ticker.C
	}
}

func (h *GameHostingUsecase) runSchedules() {
	ctx := context.Background()
	list, err := h.servers.repo.List(ctx)
	if err != nil {
		return
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	now := time.Now()
	for _, s := range list {
		if s.Mode != entgameserver.ModeHosted || s.InstallDir == "" {
			continue
		}
		opts := hostingOptionsOf(s)
		p := opts.Backup
		if !p.Enabled {
			continue
		}
		if p.LastRun != nil && now.Sub(*p.LastRun) < time.Duration(p.IntervalHours)*time.Hour {
			continue
		}
		if h.isBusy(s.ID) {
			continue
		}
		ha, ok := hostAdapterOf(s.Game)
		if !ok {
			continue
		}
		l, err := h.layoutOf(s, ha, opts)
		if err != nil {
			continue
		}
		opts.Backup.LastRun = &now
		s.Options = opts.encode()
		if _, err := h.servers.repo.Update(ctx, s); err != nil {
			continue
		}
		server, layout, o := s, l, opts
		if _, err := h.submitTask(ctx, server, gameTaskBackup, "定时备份", []string{"snapshot", "archive"}, false, 0, func(ctx context.Context, r task.Reporter) error {
			if _, err := h.runBackup(ctx, server, layout, o, "schedule", "", r); err != nil {
				return err
			}
			h.pruneBackups(context.WithoutCancel(ctx), server, o.Backup.Keep)
			return nil
		}); err != nil {
			h.log.Warnf("schedule backup of %s: %v", s.ID, err)
		}
	}
}
