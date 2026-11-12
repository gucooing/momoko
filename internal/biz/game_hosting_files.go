package biz

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	entgameserver "momoko/internal/data/ent/gen/gameserver"
	"momoko/pkg/file"
	"momoko/pkg/jsondoc"
	"momoko/pkg/logtail"
	"momoko/pkg/pre"
	"momoko/pkg/response"
	"momoko/pkg/task"
	"momoko/pkg/utils"
)

// ---- 专用设置 ----

func configRevision(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])[:16]
}

// protoPlain 把文档值转成 structpb 能接受的结构（json.Number → float64）。
func protoPlain(v any) any {
	switch t := v.(type) {
	case json.Number:
		if f, err := t.Float64(); err == nil {
			return f
		}
		return t.String()
	case map[string]any:
		for k, x := range t {
			t[k] = protoPlain(x)
		}
		return t
	case []any:
		for i, x := range t {
			t[i] = protoPlain(x)
		}
		return t
	default:
		return v
	}
}

func findField(sections []*v1.GameSettingsSection, key string) *v1.GameSettingsField {
	for _, sec := range sections {
		for _, f := range sec.Fields {
			if f.Key == key {
				return f
			}
		}
	}
	return nil
}

// GetSettings 读取专用设置：适配器表单、配置内容、原文与版本。
func (h *GameHostingUsecase) GetSettings(ctx context.Context, id string) (*v1.GameServerSettings, error) {
	s, ha, _, opts, err := h.hosted(ctx, id)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(s.InstallDir, opts.ConfigFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, ErrSystem(err)
	}
	doc, err := jsondoc.Parse(raw)
	if err != nil {
		return nil, response.BadRequest(400, "服务端配置不是有效的 JSON："+err.Error())
	}
	out := &v1.GameServerSettings{
		Sections:   ha.settingsSchema(),
		Values:     anyValue(protoPlain(jsondoc.Plain(doc))),
		Revision:   configRevision(raw),
		Raw:        string(raw),
		ConfigFile: opts.ConfigFile,
		Running:    h.processState(s, "main") == GameStateRunning,
	}
	if st, err := os.Stat(path); err == nil {
		out.ModifiedAt = timestamppb.New(st.ModTime())
	}
	known := ha.topKeys()
	for _, k := range doc.Keys() {
		if !known[k] {
			out.UnknownKeys = append(out.UnknownKeys, k)
		}
	}
	return out, nil
}

// coerceSetting 把表单值转成配置里的类型。
func coerceSetting(f *v1.GameSettingsField, v any) (any, error) {
	switch f.Type {
	case "int":
		n, ok := v.(float64)
		if !ok || n != math.Trunc(n) {
			return nil, errors.New("integer expected")
		}
		return json.Number(strconv.FormatInt(int64(n), 10)), nil
	case "bool":
		b, ok := v.(bool)
		if !ok {
			return nil, errors.New("boolean expected")
		}
		return b, nil
	case "list":
		arr, ok := v.([]any)
		if !ok {
			return nil, errors.New("list expected")
		}
		out := make([]any, 0, len(arr))
		for _, x := range arr {
			str, ok := x.(string)
			if !ok {
				return nil, errors.New("list of strings expected")
			}
			if str = strings.TrimSpace(str); str != "" {
				out = append(out, str)
			}
		}
		return out, nil
	default:
		str, ok := v.(string)
		if !ok {
			return nil, errors.New("string expected")
		}
		return strings.TrimSpace(str), nil
	}
}

// validateSetting 校验一个字段（取值范围与服务端 config.Load / Validate 一致；必填项不能是空串）。
func validateSetting(f *v1.GameSettingsField, v any, present bool) error {
	if !present {
		// 缺省的字段由服务端取默认值（config.Default）
		return nil
	}
	switch f.Type {
	case "int":
		n, ok := v.(json.Number)
		if !ok {
			return errors.New("integer expected")
		}
		i, err := n.Int64()
		if err != nil {
			return errors.New("integer expected")
		}
		if (f.HasMin && i < f.Min) || (f.HasMax && i > f.Max) {
			return errors.New("out of range")
		}
	case "bool":
		if _, ok := v.(bool); !ok {
			return errors.New("boolean expected")
		}
	case "list":
		arr, ok := v.([]any)
		if !ok {
			return errors.New("list expected")
		}
		for _, x := range arr {
			if _, ok := x.(string); !ok {
				return errors.New("list of strings expected")
			}
		}
	default:
		str, ok := v.(string)
		if !ok {
			return errors.New("string expected")
		}
		if str == "" {
			if f.Required {
				return errors.New("required")
			}
			return nil
		}
		switch f.Type {
		case "enum":
			found := false
			for _, o := range f.Options {
				if o == str {
					found = true
				}
			}
			if !found {
				return errors.New("unknown option")
			}
		case "listen":
			if !validListen(str) {
				return errors.New("host:port expected")
			}
		case "url":
			if !validOrigin(str) {
				return errors.New("absolute http(s) URL expected")
			}
		}
	}
	return nil
}

func settingPreview(f *v1.GameSettingsField, v any, present bool) string {
	if !present {
		return ""
	}
	switch t := v.(type) {
	case string:
		if len([]rune(t)) > 60 {
			return string([]rune(t)[:60]) + "…"
		}
		return t
	case []any:
		parts := make([]string, 0, len(t))
		for _, x := range t {
			parts = append(parts, fmt.Sprint(x))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprint(v)
	}
}

// SaveSettings 保存专用设置：版本不一致拒绝；表单模式只改动给出的字段，原文模式整体替换；未识别的字段原样保留。
// MGP 监听或密钥变化时同步 Momoko 保存的接入信息。
func (h *GameHostingUsecase) SaveSettings(ctx context.Context, req *v1.SaveGameServerSettingsRequest) (*v1.SaveGameServerSettingsResponse, error) {
	s, ha, _, opts, err := h.hosted(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if h.isBusy(s.ID) {
		return nil, ErrGameHostingBusy
	}
	path := filepath.Join(s.InstallDir, opts.ConfigFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, ErrSystem(err)
	}
	rev := configRevision(raw)
	if req.Revision != rev {
		return nil, ErrGameSettingsChanged
	}
	old, err := jsondoc.Parse(raw)
	if err != nil {
		return nil, response.BadRequest(400, "服务端配置不是有效的 JSON："+err.Error())
	}
	schema := ha.settingsSchema()
	var next *jsondoc.Object
	var out []byte
	rawMode := strings.TrimSpace(req.Raw) != ""
	if rawMode {
		next, err = jsondoc.Parse([]byte(req.Raw))
		if err != nil {
			return nil, response.BadRequest(400, "JSON 无效："+err.Error())
		}
		out = []byte(req.Raw)
		if !bytes.HasSuffix(out, []byte("\n")) {
			out = append(out, '\n')
		}
	} else {
		next, _ = jsondoc.Parse(raw)
		if req.Values != nil {
			for k, v := range req.Values.AsMap() {
				f := findField(schema, k)
				if f == nil || f.Managed {
					return nil, response.BadRequest(400, "不能修改配置项 "+k)
				}
				val, err := coerceSetting(f, v)
				if err != nil {
					return nil, response.BadRequest(400, "配置项 "+k+" 无效："+err.Error())
				}
				if err := next.Assign(k, val); err != nil {
					return nil, response.BadRequest(400, "配置项 "+k+" 无效："+err.Error())
				}
			}
		}
		if out, err = jsondoc.Marshal(next); err != nil {
			return nil, ErrSystem(err)
		}
	}
	// 校验与差异
	params := map[string]string{}
	var changed []string
	for _, sec := range schema {
		for _, f := range sec.Fields {
			nv, nok := next.Lookup(f.Key)
			if err := validateSetting(f, nv, nok); err != nil {
				return nil, response.BadRequest(400, "配置项 "+f.Key+" 无效："+err.Error())
			}
			ov, ook := old.Lookup(f.Key)
			if nok == ook && (!nok || jsondoc.Equal(ov, nv)) {
				continue
			}
			changed = append(changed, f.Key)
			if f.Type != "secret" && !f.Managed && len(params) < 24 {
				params["old:"+f.Key] = settingPreview(f, ov, ook)
				params["new:"+f.Key] = settingPreview(f, nv, nok)
			}
		}
	}
	known := ha.topKeys()
	seen := map[string]bool{}
	for _, k := range append(old.Keys(), next.Keys()...) {
		if known[k] || seen[k] {
			continue
		}
		seen[k] = true
		ov, ook := old.Get(k)
		nv, nok := next.Get(k)
		if nok != ook || (nok && !jsondoc.Equal(ov, nv)) {
			changed = append(changed, k)
		}
	}
	res := &v1.SaveGameServerSettingsResponse{Revision: rev, ChangedKeys: changed}
	if len(changed) == 0 {
		return res, nil
	}
	if err := writeConfigRaw(s.InstallDir, path, out); err != nil {
		return nil, ErrSystem(err)
	}
	res.Revision = configRevision(out)
	h.forgetLayout(s.ID)
	// MGP 监听 / 密钥变化：同步接入信息
	if nl, err := ha.loadLayout(s.InstallDir, opts.ConfigFile); err == nil {
		keyChanged := nl.MGPKeyID != docString(old, "mgp.key_id", "") || nl.MGPSecret != docString(old, "mgp.secret", "")
		if keyChanged || nl.mgpEndpoint() != s.MgpEndpoint {
			if err := h.applyLayoutMGP(s, nl, keyChanged); err == nil {
				if _, err := h.servers.repo.Update(ctx, s); err != nil {
					h.log.Warnf("sync mgp of %s: %v", s.ID, err)
				}
				h.servers.forget(s.ID)
			}
		}
	}
	sort.Strings(changed)
	keys := changed
	if len(keys) > 40 {
		keys = keys[:40]
	}
	params["keys"] = strings.Join(keys, ",")
	params["count"] = itoa(len(changed))
	action := "settings"
	if rawMode {
		action = "settingsRaw"
	}
	running := h.processState(s, "main") == GameStateRunning
	if running && req.Restart {
		h.restartAsync(ctx, s, "main")
		res.Restarted = true
	} else if running {
		res.PendingRestart = true
	}
	params["restarted"] = boolStr(res.Restarted)
	h.history(ctx, s.ID, "config", action, params, "success", "")
	return res, nil
}

// ---- 文件（服务端目录；与应用实例的文件接口同构）----

func (h *GameHostingUsecase) store(ctx context.Context, id string) (file.Store, string, string, error) {
	s, err := h.servers.Get(ctx, id)
	if err != nil {
		return nil, "", "", err
	}
	if s.Mode != entgameserver.ModeHosted || s.InstallDir == "" {
		return nil, "", "", ErrGameNotHosted
	}
	store, err := file.NewLocalStore(s.InstallDir)
	if err != nil {
		return nil, "", "", ErrSystem(err)
	}
	uid, _ := h.actor(ctx)
	return store, s.InstallDir, uid, nil
}

func (h *GameHostingUsecase) FileList(ctx context.Context, req *v1.GetInstanceFileListRequest) (*v1.GetInstanceFileListResponse, error) {
	store, _, _, err := h.store(ctx, req.Id)
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
	start, end := (req.Page-1)*req.PageSize, req.Page*req.PageSize
	var pages []*v1.FileEntryInfo
	if start >= 0 && start < total {
		pages = result[start:min(end, total)]
	}
	file.FillChildCounts(ctx, store, pages)
	directory, err := store.DirInfo(ctx, req.Path)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.GetInstanceFileListResponse{Directory: directory, Items: pages, Page: req.Page, PageSize: req.PageSize, Total: total}, nil
}

func (h *GameHostingUsecase) FileTree(ctx context.Context, req *v1.GetInstanceFileTreeRequest) (*v1.GetInstanceFileTreeResponse, error) {
	store, _, _, err := h.store(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	entries, err := store.List(ctx, req.Path, v1.FileSortField_FILE_SORT_FIELD_NAME, false)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.GetInstanceFileTreeResponse{Path: req.Path, Nodes: toTreeNodes(entries)}, nil
}

func (h *GameHostingUsecase) CreateFile(ctx context.Context, req *v1.CreateInstanceFileRequest) (*v1.CreateInstanceFileResponse, error) {
	store, _, _, err := h.store(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if req.Info == nil {
		return nil, ErrGameServerInvalid
	}
	if err := store.Create(ctx, req.Info); err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.CreateInstanceFileResponse{}, nil
}

func (h *GameHostingUsecase) RenameFile(ctx context.Context, req *v1.RenameInstanceFileRequest) (*v1.RenameInstanceFileResponse, error) {
	store, _, _, err := h.store(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	path, err := store.Rename(ctx, req.Path, req.NewName)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.RenameInstanceFileResponse{Path: path}, nil
}

func (h *GameHostingUsecase) CopyFile(ctx context.Context, req *v1.CopyInstanceFileRequest) (*v1.CopyInstanceFileResponse, error) {
	store, base, uid, err := h.store(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	copier, ok := store.(file.Copier)
	if !ok {
		return nil, ErrFileSourceUnsupported
	}
	t := file.NewCopyTask(task.Meta{UserID: uid, Title: transferTitle("复制", req.Paths)}, copier, file.TransferPayload{BasePath: base, Paths: req.Paths, Target: req.TargetPath})
	info, err := submitTransfer(ctx, h.tasks, t)
	if err != nil {
		return nil, err
	}
	return &v1.CopyInstanceFileResponse{Task: info}, nil
}

func (h *GameHostingUsecase) CutFile(ctx context.Context, req *v1.CutInstanceFileRequest) (*v1.CutInstanceFileResponse, error) {
	store, base, uid, err := h.store(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	mover, ok := store.(file.Mover)
	if !ok {
		return nil, ErrFileSourceUnsupported
	}
	t := file.NewMoveTask(task.Meta{UserID: uid, Title: transferTitle("移动", req.Paths)}, mover, file.TransferPayload{BasePath: base, Paths: req.Paths, Target: req.TargetPath})
	info, err := submitTransfer(ctx, h.tasks, t)
	if err != nil {
		return nil, err
	}
	return &v1.CutInstanceFileResponse{Task: info}, nil
}

func (h *GameHostingUsecase) DeleteFiles(ctx context.Context, req *v1.BatchDeleteInstanceFileRequest) (*v1.BatchDeleteInstanceFileResponse, error) {
	store, _, _, err := h.store(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.BatchDeleteInstanceFileResponse{Items: store.Delete(ctx, req.Paths)}, nil
}

func (h *GameHostingUsecase) CompressFiles(ctx context.Context, req *v1.BatchCompressInstanceFileRequest) (*v1.BatchCompressInstanceFileResponse, error) {
	store, _, _, err := h.store(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	archiver, ok := store.(file.Archiver)
	if !ok {
		return nil, ErrFileSourceUnsupported
	}
	output, err := archiver.Compress(ctx, req.Paths, req.GetTargetPath())
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.BatchCompressInstanceFileResponse{OutputPath: output}, nil
}

func (h *GameHostingUsecase) UnzipFile(ctx context.Context, req *v1.UnzipInstanceFileRequest) (*v1.UnzipInstanceFileResponse, error) {
	store, _, _, err := h.store(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	archiver, ok := store.(file.Archiver)
	if !ok {
		return nil, ErrFileSourceUnsupported
	}
	output, err := archiver.Unzip(ctx, req.Path, req.GetTargetPath())
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.UnzipInstanceFileResponse{OutputPath: output}, nil
}

func (h *GameHostingUsecase) EditFile(ctx context.Context, req *v1.EditInstanceFileRequest) (*v1.EditInstanceFileResponse, error) {
	store, _, _, err := h.store(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if len(req.Content) > file.MaxLoadFileSize {
		return nil, ErrSystem(fmt.Errorf("文件太大"))
	}
	if err := store.Write(ctx, req.Path, bytes.NewReader(req.Content)); err != nil {
		return nil, ErrSystem(err)
	}
	h.forgetLayout(req.Id)
	return &v1.EditInstanceFileResponse{}, nil
}

func (h *GameHostingUsecase) FilePreSign(ctx context.Context, req *v1.InstanceFilePreSignRequest) (*v1.InstanceFilePreSignResponse, error) {
	_, base, uid, err := h.store(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	realPath, err := file.ResolveLocalPath(base, req.Path)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if st, err := os.Stat(realPath); err != nil || st.IsDir() {
		return nil, ErrFileNotExist
	}
	sign, err := pre.NewFileSignInfo(utils.GenerateRandomString(10), realPath, uid, 24*time.Hour).Sign()
	if err != nil {
		return nil, ErrSign
	}
	return &v1.InstanceFilePreSignResponse{DownloadUrlPath: fmt.Sprintf("%s?sign=%s", PreFileDownload, sign)}, nil
}

func (h *GameHostingUsecase) FilePreSignUpload(ctx context.Context, req *v1.InstanceFilePreSignUploadRequest) (*v1.UploadInfo, error) {
	if req.FileSize > math.MaxInt64 {
		return nil, ErrUploadRequestInvalid
	}
	store, base, uid, err := h.store(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if uid == "" {
		return nil, ErrTokenInvalid
	}
	realPath, err := file.ResolveLocalPath(base, req.Path)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if err := os.MkdirAll(realPath, 0o755); err != nil {
		return nil, ErrSystem(err)
	}
	seed := file.NewChunkedUpload(req.Hash, realPath, req.FileName, req.FileSize, true)
	row, err := h.instances.fileRepo.GetOrCreate(ctx, uid, seed)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if row.Completed {
		return toUploadInfo(row, &file.Upload{}), nil
	}
	u, err := buildUpload(h.instances.fileRepo, uid, row)
	if err != nil {
		return nil, err
	}
	if err := store.PrepareUpload(ctx, u); err != nil {
		return nil, ErrSystem(err)
	}
	uploadCache.Set(row.ID, &file.ChunkedUpload{FileUpload: row})
	return toUploadInfo(row, u), nil
}

// ---- 日志 ----

// ListLogs 服务端日志文件（<log.directory>/<模块>.log）。
func (h *GameHostingUsecase) ListLogs(ctx context.Context, id string) (*v1.ListGameServerLogsResponse, error) {
	s, ha, l, _, err := h.hosted(ctx, id)
	if err != nil {
		return nil, err
	}
	out := &v1.ListGameServerLogsResponse{FileDisabled: !l.LogFile}
	if rel, ok := relInside(s.InstallDir, l.LogDir); ok {
		out.Dir = rel
	} else {
		out.Dir = l.LogDir
	}
	for _, m := range ha.Spec().LogModules {
		lm := &v1.GameServerLogModule{Key: m, File: m + ".log"}
		if st, err := os.Stat(filepath.Join(l.LogDir, m+".log")); err == nil {
			lm.Exists, lm.Size, lm.ModifiedAt = true, st.Size(), timestamppb.New(st.ModTime())
		}
		out.Modules = append(out.Modules, lm)
	}
	return out, nil
}

var logLevelRank = map[string]int{"debug": 0, "info": 1, "warn": 2, "error": 3}

func logLevelOf(level string) string {
	l := strings.ToLower(level)
	for _, k := range []string{"debug", "info", "warn", "error"} {
		if strings.HasPrefix(l, k) {
			return k
		}
	}
	return l
}

// TailLog 读取一个日志模块的最近记录（JSONL，从末尾向前，按级别与关键字筛选）。
func (h *GameHostingUsecase) TailLog(ctx context.Context, req *v1.TailGameServerLogRequest) (*v1.TailGameServerLogResponse, error) {
	_, ha, l, _, err := h.hosted(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	found := false
	for _, m := range ha.Spec().LogModules {
		if m == req.Module {
			found = true
		}
	}
	if !found {
		return nil, ErrGameServerInvalid
	}
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}
	minRank, filterLevel := logLevelRank[strings.ToLower(req.Level)], req.Level != ""
	q := strings.ToLower(strings.TrimSpace(req.Q))
	match := func(line []byte) bool {
		if q != "" && !bytes.Contains(bytes.ToLower(line), []byte(q)) {
			return false
		}
		if filterLevel {
			var head struct {
				Level string `json:"level"`
			}
			if json.Unmarshal(line, &head) != nil {
				return false
			}
			if r, ok := logLevelRank[logLevelOf(head.Level)]; !ok || r < minRank {
				return false
			}
		}
		return true
	}
	path := filepath.Join(l.LogDir, req.Module+".log")
	res, err := logtail.Tail(path, req.Before, limit, match)
	if errors.Is(err, os.ErrNotExist) {
		return &v1.TailGameServerLogResponse{}, nil
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := &v1.TailGameServerLogResponse{NextBefore: res.Next, Size: res.Size}
	for _, line := range res.Lines {
		e := &v1.GameServerLogEntry{}
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) != nil {
			e.Raw = line
			out.Items = append(out.Items, e)
			continue
		}
		if t, ok := m["time"].(string); ok {
			if pt, err := time.Parse(time.RFC3339Nano, t); err == nil {
				e.Time = timestamppb.New(pt)
			}
		}
		if lv, ok := m["level"].(string); ok {
			e.Level = logLevelOf(lv)
		}
		if msg, ok := m["msg"].(string); ok {
			e.Message = msg
		}
		for _, k := range []string{"time", "level", "msg", "module"} {
			delete(m, k)
		}
		if len(m) > 0 {
			if st, err := structpb.NewStruct(m); err == nil {
				e.Attrs = st
			}
		}
		out.Items = append(out.Items, e)
	}
	return out, nil
}
