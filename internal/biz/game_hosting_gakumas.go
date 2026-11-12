package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/gobuild"
	"momoko/pkg/jsondoc"
	"momoko/pkg/netprobe"
)

// 学マス 服务端（gakumas/server）的托管驱动。以下约定全部来自服务端源码：
//   - cmd/gakumas：`gakumas-server -config <file>` 启动，`gakumas-server mgp keygen` 生成密钥；收到 Ctrl+C / SIGTERM 时保存全部玩家后退出；
//   - config/config.go：配置结构、默认值与取值范围（解析时拒绝未知字段，相对路径相对配置文件所在目录）；
//   - app.Version：构建时用 -ldflags "-X gucooing/gakumas/server/app.Version=..." 注入的版本；
//   - pkg/log：日志按模块写 <log.directory>/<app|game|gateway|sdk|packet>.log（JSONL）；
//   - tools/resource-server：`resource-server serve -listen -origin -root`，GET /health，显式任务 verify / update，
//     数据目录默认是「含 server/go.mod 的项目根」下的 resource（manifest.json、active.json、progress.json）。

const (
	gakumasModule      = "gucooing/gakumas/server"
	gakumasMainPath    = "gucooing/gakumas/server/cmd/gakumas"
	gakumasVersionVar  = "gucooing/gakumas/server/app.Version"
	gakumasResourceMod = "gucooing/gakumas/server/tools/resource-server"
	gakumasConfigFile  = "config.json"
	gakumasTemplate    = "config.example.json"
)

// 配置里各监听的默认值（config.Default）。
var gakumasDefaultListens = map[string]string{
	"grpc": "127.0.0.1:50051",
	"http": "127.0.0.1:8080",
	"mgp":  "127.0.0.1:8081",
}

const gakumasDefaultResourceListen = "127.0.0.1:8090"

var gakumasLogModules = []string{"app", "game", "gateway", "sdk", "packet"}

func exeName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

type gakumasHosting struct {
	h *GameHostingUsecase
}

func (g *gakumasHosting) Capabilities() []string {
	return []string{"console", "files", "logs", "settings", "update", "resource", "backup"}
}

func (g *gakumasHosting) Spec() GameHostingSpec {
	return GameHostingSpec{
		Roles: []*v1.GameAdapterRole{
			{Key: "main", Executable: "gakumas-server"},
			{Key: "resource", Executable: "resource-server", Optional: true},
		},
		ConfigFile:  gakumasConfigFile,
		LogModules:  gakumasLogModules,
		BackupScope: []string{gakumasConfigFile, "data/", "database"},
	}
}

func (g *gakumasHosting) State(ctx context.Context, s *gen.GameServer) string {
	return g.h.processState(s, "main")
}

func (g *gakumasHosting) Describe(ctx context.Context, s *gen.GameServer, info *v1.GameServerInfo) {
	g.h.describe(ctx, s, info)
}

// gameHostLayout 托管服务器在磁盘上的布局（从服务端配置读出）。
type gameHostLayout struct {
	Dir        string
	ConfigFile string
	ConfigPath string
	Doc        *jsondoc.Object
	Listens    map[string]string // grpc / http / mgp
	MGPKeyID   string
	MGPSecret  string
	DataDir    string
	ResDir     string
	LogDir     string
	LogFile    bool
	DBDriver   string
	DBPath     string // sqlite 绝对路径
	SaveTimout int
	Executable string
	ResExe     string
	ResOrigin  string // 配置里的 resource_server_url
}

// relPath 相对配置所在目录解析（与 config.Load 一致）。
func (l *gameHostLayout) relPath(p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(filepath.Dir(l.ConfigPath), p)
}

func docString(doc *jsondoc.Object, path, def string) string {
	v, ok := doc.Lookup(path)
	if !ok {
		return def
	}
	if s, ok := v.(string); ok {
		return s
	}
	return def
}

func docInt(doc *jsondoc.Object, path string, def int) int {
	v, ok := doc.Lookup(path)
	if !ok {
		return def
	}
	if n, ok := v.(json.Number); ok {
		if i, err := n.Int64(); err == nil {
			return int(i)
		}
	}
	return def
}

func docBool(doc *jsondoc.Object, path string, def bool) bool {
	v, ok := doc.Lookup(path)
	if !ok {
		return def
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return def
}

// loadLayout 解析服务端目录与配置。
func (g *gakumasHosting) loadLayout(dir, configFile string) (*gameHostLayout, error) {
	if configFile == "" {
		configFile = gakumasConfigFile
	}
	l := &gameHostLayout{Dir: dir, ConfigFile: configFile, ConfigPath: filepath.Join(dir, configFile)}
	raw, err := os.ReadFile(l.ConfigPath)
	if err != nil {
		return nil, err
	}
	doc, err := jsondoc.Parse(raw)
	if err != nil {
		return nil, err
	}
	g.fillLayout(l, doc)
	return l, nil
}

func (g *gakumasHosting) fillLayout(l *gameHostLayout, doc *jsondoc.Object) {
	l.Doc = doc
	l.Listens = map[string]string{
		"grpc": docString(doc, "grpc_listen", gakumasDefaultListens["grpc"]),
		"http": docString(doc, "http_listen", gakumasDefaultListens["http"]),
		"mgp":  docString(doc, "mgp.listen", gakumasDefaultListens["mgp"]),
	}
	l.MGPKeyID = docString(doc, "mgp.key_id", "")
	l.MGPSecret = docString(doc, "mgp.secret", "")
	l.DataDir = l.relPath(docString(doc, "data_path", "data"))
	l.ResDir = l.relPath(docString(doc, "resource_path", "res"))
	l.LogDir = l.relPath(docString(doc, "log.directory", "log"))
	l.LogFile = docBool(doc, "log.file", true)
	l.DBDriver = docString(doc, "database.driver", "sqlite")
	if l.DBDriver == "sqlite" {
		l.DBPath = l.relPath(docString(doc, "database.path", "./gakumas.sqlite"))
	}
	l.SaveTimout = docInt(doc, "game.save_timeout_seconds", 15)
	l.Executable = filepath.Join(l.Dir, exeName("gakumas-server"))
	l.ResExe = filepath.Join(l.Dir, exeName("resource-server"))
	l.ResOrigin = docString(doc, "resource_server_url", "")
}

// mgpEndpoint 托管服务器的 MGP 地址（监听 0.0.0.0 时经本机回环访问）。
func (l *gameHostLayout) mgpEndpoint() string {
	if l.MGPKeyID == "" {
		return ""
	}
	return "http://" + netprobe.DialAddr(l.Listens["mgp"])
}

// stopTimeout 停止超时：服务端停服时保存全部玩家，单次落库期限是 game.save_timeout_seconds。
func (l *gameHostLayout) stopTimeout() int {
	return l.SaveTimout + 15
}

// resourceRoot 资源服务器的默认数据目录：从服务端目录向上找到含 server/go.mod 的项目根，取其中的 resource。
func gakumasResourceRoot(dir string) string {
	for cur := dir; ; cur = filepath.Dir(cur) {
		if _, err := os.Stat(filepath.Join(cur, "server", "go.mod")); err == nil {
			return filepath.Join(cur, "resource")
		}
		if filepath.Dir(cur) == cur {
			return ""
		}
	}
}

// checkBuild 读取一个服务端可执行文件的构建信息并确认它是学マス 服务端。
func (g *gakumasHosting) checkBuild(path string) *v1.GameServerBuildInfo {
	out := &v1.GameServerBuildInfo{Path: path}
	st, err := os.Stat(path)
	if err != nil {
		out.Error = "not_found"
		return out
	}
	if st.IsDir() {
		out.Error = "is_directory"
		return out
	}
	out.Exists = true
	out.Size = st.Size()
	out.ModifiedAt = timestamppb.New(st.ModTime())
	info, err := gobuild.Read(path)
	if err != nil {
		out.Error = "not_go_binary"
		return out
	}
	out.GoVersion = info.GoVersion
	out.Module = info.Module
	out.VcsRevision = info.VCSRevision
	if !info.VCSTime.IsZero() {
		out.VcsTime = timestamppb.New(info.VCSTime)
	}
	out.Version = info.Var(gakumasVersionVar)
	if info.Module != gakumasModule || info.MainPath != gakumasMainPath {
		out.Error = "wrong_module"
		return out
	}
	out.Valid = true
	return out
}

// resolveBuildPath 候选构建：目录时取其中的服务端可执行文件；相对路径相对服务端目录。
func (g *gakumasHosting) resolveBuildPath(dir, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, filepath.FromSlash(path))
	}
	if st, err := os.Stat(path); err == nil && st.IsDir() {
		return filepath.Join(path, exeName("gakumas-server"))
	}
	return filepath.Clean(path)
}

func countFiles(dir, ext string) (int, error) {
	n := 0
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && (ext == "" || strings.EqualFold(filepath.Ext(d.Name()), ext)) {
			n++
		}
		return nil
	})
	return n, err
}

func validListen(addr string) bool {
	host, port, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return false
	}
	_ = host
	p, err := strconv.Atoi(port)
	return err == nil && p > 0 && p < 65536
}

func validOrigin(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Host != "" && (u.Scheme == "http" || u.Scheme == "https") && u.User == nil && u.RawQuery == "" && u.Fragment == ""
}

func hostCheck(key, state string, params map[string]string, err error) *v1.GameServerCheck {
	c := &v1.GameServerCheck{Key: key, State: state, Params: params}
	if err != nil {
		c.ErrorMessage = err.Error()
	}
	return c
}

// detect 识别服务端目录：可执行文件、配置、数据库、资源表、数据、密钥、资源服务器与镜像、端口占用。
func (g *gakumasHosting) detect(ctx context.Context, dir, configFile string, listens map[string]string, selfID string) (*v1.DetectGameServerDirResponse, error) {
	out := &v1.DetectGameServerDirResponse{Listens: map[string]string{}}
	dir = strings.TrimSpace(dir)
	if dir == "" || !filepath.IsAbs(dir) {
		out.Checks = append(out.Checks, hostCheck("dir", "fail", map[string]string{"dir": dir}, errors.New("absolute path required")))
		return out, nil
	}
	dir = filepath.Clean(dir)
	out.InstallDir = dir
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		out.Checks = append(out.Checks, hostCheck("dir", "fail", map[string]string{"dir": dir}, err))
		return out, nil
	}
	// 可执行文件与版本
	build := g.checkBuild(filepath.Join(dir, exeName("gakumas-server")))
	out.Build = build
	if build.Valid {
		out.Checks = append(out.Checks, hostCheck("executable", "ok", map[string]string{"file": filepath.Base(build.Path), "version": build.Version, "size": strconv.FormatInt(build.Size, 10)}, nil))
	} else {
		out.Checks = append(out.Checks, &v1.GameServerCheck{Key: "executable", State: "fail", Params: map[string]string{"file": filepath.Base(build.Path)}, ErrorCode: build.Error})
	}
	// 配置：优先指定的，其次 config.json，没有时从 config.example.json 复制
	if configFile == "" {
		configFile = gakumasConfigFile
	}
	out.ConfigFile = configFile
	l := &gameHostLayout{Dir: dir, ConfigFile: configFile, ConfigPath: filepath.Join(dir, configFile)}
	raw, err := os.ReadFile(l.ConfigPath)
	if err == nil {
		out.ConfigExists = true
	} else if errors.Is(err, os.ErrNotExist) {
		if tpl, terr := os.ReadFile(filepath.Join(dir, gakumasTemplate)); terr == nil {
			raw, err = tpl, nil
			out.ConfigTemplate = gakumasTemplate
		}
	}
	if err != nil {
		out.Checks = append(out.Checks, hostCheck("config", "fail", map[string]string{"file": configFile}, err))
		return out, nil
	}
	doc, err := jsondoc.Parse(raw)
	if err != nil {
		out.Checks = append(out.Checks, hostCheck("config", "fail", map[string]string{"file": configFile}, err))
		return out, nil
	}
	g.fillLayout(l, doc)
	out.Checks = append(out.Checks, hostCheck("config", "ok", map[string]string{"file": configFile, "template": out.ConfigTemplate}, nil))
	// 数据库
	switch l.DBDriver {
	case "sqlite":
		params := map[string]string{"driver": "sqlite", "path": l.DBPath}
		if st, err := os.Stat(l.DBPath); err == nil {
			params["size"] = strconv.FormatInt(st.Size(), 10)
			out.Checks = append(out.Checks, hostCheck("database", "ok", params, nil))
		} else {
			out.Checks = append(out.Checks, hostCheck("database", "warn", params, nil))
		}
	default:
		params := map[string]string{"driver": l.DBDriver}
		if u, err := url.Parse(docString(doc, "database.dsn", "")); err == nil && u.Host != "" {
			params["host"] = u.Host
		}
		out.Checks = append(out.Checks, hostCheck("database", "ok", params, nil))
	}
	// 资源表与手写数据
	if n, err := countFiles(l.ResDir, ".json"); err == nil && n > 0 {
		out.Checks = append(out.Checks, hostCheck("resources", "ok", map[string]string{"path": l.ResDir, "files": strconv.Itoa(n)}, nil))
	} else {
		out.Checks = append(out.Checks, hostCheck("resources", "fail", map[string]string{"path": l.ResDir}, err))
	}
	if n, err := countFiles(l.DataDir, ".json"); err == nil && n > 0 {
		out.Checks = append(out.Checks, hostCheck("data", "ok", map[string]string{"path": l.DataDir, "files": strconv.Itoa(n)}, nil))
	} else {
		out.Checks = append(out.Checks, hostCheck("data", "fail", map[string]string{"path": l.DataDir}, err))
	}
	// MGP 密钥
	out.MgpKeyPresent = l.MGPKeyID != "" && l.MGPSecret != ""
	out.MgpKeyId = l.MGPKeyID
	if out.MgpKeyPresent {
		out.Checks = append(out.Checks, hostCheck("mgpKey", "ok", map[string]string{"keyId": l.MGPKeyID}, nil))
	} else {
		out.Checks = append(out.Checks, hostCheck("mgpKey", "pending", nil, nil))
	}
	// 配置里的 token_secret 仍是示例里的公开值
	if out.ConfigExists {
		if tpl, err := os.ReadFile(filepath.Join(dir, gakumasTemplate)); err == nil {
			if tdoc, err := jsondoc.Parse(tpl); err == nil {
				if a := docString(doc, "game.token_secret", ""); a != "" && a == docString(tdoc, "game.token_secret", "") {
					out.Checks = append(out.Checks, hostCheck("tokenSecret", "warn", map[string]string{"template": gakumasTemplate}, nil))
				}
			}
		}
	} else {
		out.Checks = append(out.Checks, hostCheck("tokenSecret", "warn", map[string]string{"template": gakumasTemplate}, nil))
	}
	// 监听地址（请求里给出的优先）
	for _, k := range []string{"grpc", "http", "mgp"} {
		out.Listens[k] = l.Listens[k]
		if v := strings.TrimSpace(listens[k]); v != "" {
			out.Listens[k] = v
		}
	}
	// 资源服务器与镜像
	if st, err := os.Stat(l.ResExe); err == nil && !st.IsDir() {
		out.HasResourceServer = true
	}
	out.ResourceOrigin = l.ResOrigin
	out.ResourceRoot = gakumasResourceRoot(dir)
	resListen := gakumasDefaultResourceListen
	if u, err := url.Parse(l.ResOrigin); err == nil && u.Port() != "" {
		resListen = net.JoinHostPort("127.0.0.1", u.Port())
	}
	if v := strings.TrimSpace(listens["resource"]); v != "" {
		resListen = v
	}
	if out.HasResourceServer {
		out.Listens["resource"] = resListen
		out.Checks = append(out.Checks, hostCheck("resourceServer", "ok", map[string]string{"file": filepath.Base(l.ResExe)}, nil))
		if out.ResourceRoot != "" {
			if m, err := readResourceManifest(out.ResourceRoot); err == nil {
				out.Checks = append(out.Checks, hostCheck("mirror", "ok", map[string]string{
					"root": out.ResourceRoot, "version": strconv.Itoa(m.Version), "revision": strconv.Itoa(m.Revision),
					"objects": strconv.Itoa(m.Objects), "bytes": strconv.FormatInt(m.Bytes, 10), "published": strconv.FormatBool(m.Published),
				}, nil))
			} else {
				out.Checks = append(out.Checks, hostCheck("mirror", "warn", map[string]string{"root": out.ResourceRoot}, err))
			}
		} else {
			out.Checks = append(out.Checks, hostCheck("mirror", "warn", nil, nil))
		}
	}
	// 端口占用（同一台服务器自己的进程在运行时不算）
	selfRunning := selfID != "" && g.h.serverRunning(ctx, selfID)
	for _, k := range []string{"grpc", "http", "mgp", "resource"} {
		addr, ok := out.Listens[k]
		if !ok {
			continue
		}
		params := map[string]string{"role": k, "listen": addr}
		switch {
		case !validListen(addr):
			out.Checks = append(out.Checks, hostCheck("port", "fail", params, errors.New("invalid listen address")))
		case selfRunning:
			out.Checks = append(out.Checks, hostCheck("port", "pending", params, nil))
		default:
			if err := netprobe.Free(addr); err != nil {
				out.Checks = append(out.Checks, hostCheck("port", "warn", params, err))
			} else {
				out.Checks = append(out.Checks, hostCheck("port", "ok", params, nil))
			}
		}
	}
	out.Ok = true
	for _, c := range out.Checks {
		if c.State == "fail" {
			out.Ok = false
		}
	}
	return out, nil
}

// configure 写服务端配置：不存在时从模板复制，写入监听地址与 MGP 密钥（其余字段原样保留）。
func (g *gakumasHosting) configure(dir string, spec *v1.GameServerHostedSpec, keyID, secret string) (*gameHostLayout, error) {
	configFile := spec.ConfigFile
	if configFile == "" {
		configFile = gakumasConfigFile
	}
	path := filepath.Join(dir, configFile)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		tpl := spec.ConfigTemplate
		if tpl == "" {
			tpl = gakumasTemplate
		}
		raw, err = os.ReadFile(filepath.Join(dir, filepath.Base(tpl)))
	}
	if err != nil {
		return nil, err
	}
	doc, err := jsondoc.Parse(raw)
	if err != nil {
		return nil, err
	}
	keys := map[string]string{"grpc": "grpc_listen", "http": "http_listen", "mgp": "mgp.listen"}
	for role, key := range keys {
		v := strings.TrimSpace(spec.Listens[role])
		if v == "" {
			continue
		}
		if !validListen(v) {
			return nil, fmt.Errorf("invalid listen address %q", v)
		}
		if err := doc.Assign(key, v); err != nil {
			return nil, err
		}
	}
	if spec.ResourceServer && strings.TrimSpace(spec.ResourceOrigin) != "" {
		origin := strings.TrimRight(strings.TrimSpace(spec.ResourceOrigin), "/")
		if !validOrigin(origin) {
			return nil, fmt.Errorf("invalid resource origin %q", origin)
		}
		if err := doc.Assign("resource_server_url", origin); err != nil {
			return nil, err
		}
	}
	if keyID != "" {
		if err := doc.Assign("mgp.key_id", keyID); err != nil {
			return nil, err
		}
		if err := doc.Assign("mgp.secret", secret); err != nil {
			return nil, err
		}
	}
	if err := writeConfigDoc(dir, path, doc); err != nil {
		return nil, err
	}
	l := &gameHostLayout{Dir: dir, ConfigFile: configFile, ConfigPath: path}
	g.fillLayout(l, doc)
	return l, nil
}

// writeConfigDoc 写回配置：原文件先复制到 .momoko/config-history/，再原子替换。
func writeConfigDoc(dir, path string, doc *jsondoc.Object) error {
	out, err := jsondoc.Marshal(doc)
	if err != nil {
		return err
	}
	return writeConfigRaw(dir, path, out)
}

func writeConfigRaw(dir, path string, out []byte) error {
	if old, err := os.ReadFile(path); err == nil {
		hist := filepath.Join(dir, gameHostingDir, "config-history")
		if err := os.MkdirAll(hist, 0o755); err == nil {
			name := filepath.Base(path) + "." + time.Now().Format("20060102-150405") + ".bak"
			_ = os.WriteFile(filepath.Join(hist, name), old, 0o600)
			pruneOldFiles(hist, filepath.Base(path)+".", 20)
		}
	}
	tmp := path + ".momoko-tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// pruneOldFiles 只保留目录里以 prefix 开头的最新 keep 个文件。
func pruneOldFiles(dir, prefix string, keep int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), prefix) {
			names = append(names, e.Name())
		}
	}
	if len(names) <= keep {
		return
	}
	// 文件名带时间戳，字典序即时间序
	sortHostNames(names)
	for _, n := range names[:len(names)-keep] {
		_ = os.Remove(filepath.Join(dir, n))
	}
}

func sortHostNames(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// gakumasProcess 一个进程的启动定义。
type gameHostProcess struct {
	Role        string
	Command     string
	StopCommand string
	StopTimeout int
	Listens     map[string]string
}

func quoteArg(s string) string {
	if strings.ContainsAny(s, " \t\"") {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return s
}

// ctrlC 写入 PTY 的 Ctrl+C：Unix 伪终端转为 SIGINT，Windows ConPTY 转为 CTRL_C_EVENT；
// 服务端（signal.NotifyContext(os.Interrupt, SIGTERM)）据此保存全部玩家后退出。
const ctrlC = "\x03"

// processes 托管的进程：游戏服务端，以及可选的资源服务器。
func (g *gakumasHosting) processes(l *gameHostLayout, opts *gameHostingOptions) []gameHostProcess {
	out := []gameHostProcess{{
		Role:        "main",
		Command:     exeName("gakumas-server") + " -config " + quoteArg(l.ConfigFile),
		StopCommand: ctrlC,
		StopTimeout: l.stopTimeout(),
		Listens:     map[string]string{"grpc": l.Listens["grpc"], "http": l.Listens["http"], "mgp": l.Listens["mgp"]},
	}}
	if opts.ResourceServer {
		cmd := exeName("resource-server") + " serve -listen " + quoteArg(opts.ResourceListen)
		if l.ResOrigin != "" {
			cmd += " -origin " + quoteArg(strings.TrimRight(l.ResOrigin, "/"))
		}
		if opts.ResourceRoot != "" {
			cmd += " -root " + quoteArg(opts.ResourceRoot)
		}
		out = append(out, gameHostProcess{
			Role:        "resource",
			Command:     cmd,
			StopCommand: ctrlC,
			StopTimeout: 10,
			Listens:     map[string]string{"resource": opts.ResourceListen},
		})
	}
	return out
}

// ---- 专用设置表单（config.Config 的结构与 Validate 的取值范围）----

func gsField(key, typ string, opts ...func(*v1.GameSettingsField)) *v1.GameSettingsField {
	f := &v1.GameSettingsField{Key: key, Type: typ, RequiresRestart: true}
	for _, o := range opts {
		o(f)
	}
	return f
}

func gsRange(min, max int64) func(*v1.GameSettingsField) {
	return func(f *v1.GameSettingsField) {
		f.HasMin, f.Min = true, min
		if max > 0 {
			f.HasMax, f.Max = true, max
		}
	}
}

func gsOptions(o ...string) func(*v1.GameSettingsField) {
	return func(f *v1.GameSettingsField) { f.Options = o }
}

func gsRequired(f *v1.GameSettingsField) { f.Required = true }

// settingsSchema 配置字段的提示（标签分组、类型、选项与取值范围）：专用设置展示的是整个配置文件，
// 这里只给认识的字段补充类型与校验；不在这里的字段照样显示、照样能改。服务端只在启动时读取配置，全部修改都要重启生效。
func (g *gakumasHosting) settingsSchema() []*v1.GameSettingsSection {
	return []*v1.GameSettingsSection{
		{Key: "net", Fields: []*v1.GameSettingsField{
			gsField("grpc_listen", "listen", gsRequired),
			gsField("http_listen", "listen", gsRequired),
			gsField("http_url", "url", gsRequired),
			gsField("resource_server_url", "url"),
			gsField("tls_cert", "path"),
			gsField("tls_key", "path"),
		}},
		{Key: "database", Fields: []*v1.GameSettingsField{
			gsField("database.driver", "enum", gsOptions("sqlite", "postgres")),
			gsField("database.path", "path"),
			gsField("database.dsn", "secret"),
			gsField("database.max_open_connections", "int", gsRange(2, 1024)),
			gsField("database.max_idle_connections", "int", gsRange(1, 1024)),
			gsField("database.busy_timeout_seconds", "int", gsRange(1, 3600)),
		}},
		{Key: "paths", Fields: []*v1.GameSettingsField{
			gsField("resource_path", "path", gsRequired),
			gsField("data_path", "path", gsRequired),
		}},
		{Key: "account", Fields: []*v1.GameSettingsField{
			gsField("auto_register", "bool"),
			gsField("session_hours", "int", gsRange(1, 8760)),
			gsField("packet_secret", "secret", gsRequired),
		}},
		{Key: "game", Fields: []*v1.GameSettingsField{
			gsField("game.save_interval_seconds", "int", gsRange(1, 86400)),
			gsField("game.disconnect_retention_seconds", "int", gsRange(1, 86400)),
			gsField("game.session_retention_hours", "int", gsRange(1, 8760)),
			gsField("game.save_workers", "int", gsRange(1, 128)),
			gsField("game.save_queue_per_worker", "int", gsRange(1, 4096)),
			gsField("game.save_timeout_seconds", "int", gsRange(1, 3600)),
			gsField("game.response_cache_entries", "int", gsRange(1, 4096)),
			gsField("game.response_cache_seconds", "int", gsRange(1, 86400)),
			gsField("game.backup_keep", "int", gsRange(1, 100)),
			gsField("game.login_log_days", "int", gsRange(1, 3650)),
			gsField("game.gm_request_days", "int", gsRange(1, 3650)),
			gsField("game.token_secret", "secret", gsRequired),
		}},
		{Key: "log", Fields: []*v1.GameSettingsField{
			gsField("log.level", "enum", gsOptions("debug", "info", "warn", "error")),
			gsField("log.directory", "path"),
			gsField("log.console", "bool"),
			gsField("log.file", "bool"),
			gsField("log.max_size_mb", "int", gsRange(1, 0)),
			gsField("log.max_backups", "int", gsRange(1, 0)),
			gsField("log.packet_bodies", "bool"),
			gsField("log.packet_hex", "bool"),
			gsField("log.packet_blacklist", "list"),
		}},
		{Key: "mgp", Fields: []*v1.GameSettingsField{
			gsField("mgp.listen", "listen", gsRequired),
			gsField("mgp.key_id", "string"),
			gsField("mgp.secret", "secret"),
		}},
	}
}
