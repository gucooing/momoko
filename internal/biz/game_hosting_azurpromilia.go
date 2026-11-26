package biz

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/jsondoc"
	"momoko/pkg/netprobe"
)

// 蓝色星原服务端（AzurPromilia server，Go 模块 blue_loli）的托管驱动。以下约定全部来自服务端源码：
//   - cmd/azurpromilia：`azurpromilia-server -config <file>` 启动，`azurpromilia-server mgp keygen` 生成 MGP 密钥；
//     收到 Ctrl+C / SIGTERM（signal.NotifyContext）后关闭 SDK 与 MGP 的 HTTP（app.shutdownTimeout 5 秒）和网关，
//     各玩家 actor 保存改动（单次落库 game.dbTimeout 10 秒，各 actor 同时进行），最后关闭数据库；
//   - config/config.go：配置结构与缺省值（Default），未写的字段取缺省、未知字段忽略；log.directory、resource_path、
//     data_path、database.path 的相对路径相对配置文件所在目录；mgp.key_id 或 mgp.secret 非空才开 MGP；
//     gateway.tcp_listen 为空则不开 TCP；
//   - app.Version：构建时用 -ldflags "-X blue_loli/app.Version=..." 注入的版本；
//   - pkg/log：日志按模块写 <log.directory>/<app|gateway|game|sdk|packet>.log（slog JSONL）；
//   - db：只有 SQLite（WAL）；运行中写入的只有数据库与日志，res（客户端导出资源）与 data（手写规则；
//     data/hotupdate 是本地 CDN 的官方热更新文件，只整套替换）随服务端版本更换，不在备份范围内。

const (
	azurModule     = "blue_loli"
	azurMainPath   = "blue_loli/cmd/azurpromilia"
	azurVersionVar = "blue_loli/app.Version"
	azurExecutable = "azurpromilia-server"
	azurConfigFile = "config.json"
	azurTemplate   = "config.example.json"
	// azurStopTimeout HTTP 关闭 5 秒 + 玩家存档 10 秒，另留 15 秒余量。
	azurStopTimeout = 5 + 10 + 15
)

// azurListens 配置里的监听：角色键、配置路径、网络与缺省值（config.Default）。
var azurListens = []struct {
	role, path, network, def string
}{
	{"kcp", "gateway.kcp_listen", "udp", "127.0.0.1:20002"},
	{"tcp", "gateway.tcp_listen", "tcp", "127.0.0.1:20002"},
	{"sdk", "sdk.listen", "tcp", "127.0.0.1:20001"},
	{"mgp", "mgp.listen", "tcp", "127.0.0.1:20003"},
}

var azurLogModules = []string{"app", "gateway", "game", "sdk", "packet"}

// azurResourceDirs gdconf.Init 要求资源目录下存在的子目录；表在 Excel 下。
var azurResourceDirs = []string{"Excel", "Config", "Localization"}

type azurpromiliaHosting struct {
	h *GameHostingUsecase
}

func (a *azurpromiliaHosting) Capabilities() []string {
	return []string{"console", "files", "logs", "settings", "update", "backup"}
}

func (a *azurpromiliaHosting) Spec() GameHostingSpec {
	listens := make([]string, 0, len(azurListens))
	for _, k := range azurListens {
		listens = append(listens, k.role)
	}
	return GameHostingSpec{
		Roles: []*v1.GameAdapterRole{
			{Key: "main", Executable: azurExecutable, Listens: listens},
		},
		ConfigFile:  azurConfigFile,
		LogModules:  azurLogModules,
		BackupScope: []string{azurConfigFile, "database"},
	}
}

func (a *azurpromiliaHosting) State(ctx context.Context, s *gen.GameServer) string {
	return a.h.processState(s, "main")
}

func (a *azurpromiliaHosting) Describe(ctx context.Context, s *gen.GameServer, info *v1.GameServerInfo) {
	a.h.describe(ctx, s, info)
}

// configKeys 服务端配置没有通行证地址：启动票据只用 momoko.app_id / app_secret 离线校验。
func (a *azurpromiliaHosting) configKeys() gameHostConfigKeys {
	return gameHostConfigKeys{
		MGPKeyID:        "mgp.key_id",
		MGPSecret:       "mgp.secret",
		MomokoAppID:     "momoko.app_id",
		MomokoAppSecret: "momoko.app_secret",
	}
}

func (a *azurpromiliaHosting) loadLayout(dir, configFile string) (*gameHostLayout, error) {
	if configFile == "" {
		configFile = azurConfigFile
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
	a.fillLayout(l, doc)
	return l, nil
}

func (a *azurpromiliaHosting) fillLayout(l *gameHostLayout, doc *jsondoc.Object) {
	l.Doc = doc
	l.Listens = map[string]string{}
	for _, k := range azurListens {
		l.Listens[k.role] = docString(doc, k.path, k.def)
	}
	l.MGPKeyID = docString(doc, "mgp.key_id", "")
	l.MGPSecret = docString(doc, "mgp.secret", "")
	l.DataDir = l.relPath(docString(doc, "data_path", "data"))
	l.ResDir = l.relPath(docString(doc, "resource_path", "res"))
	l.LogDir = l.relPath(docString(doc, "log.directory", "log"))
	l.LogFile = docBool(doc, "log.file", true)
	l.DBDriver = "sqlite"
	l.DBPath = l.relPath(docString(doc, "database.path", "azur.sqlite"))
	l.Probes = nil
	for _, k := range azurListens {
		if k.role != "mgp" && l.Listens[k.role] != "" {
			l.Probes = append(l.Probes, gameHostProbe{Key: k.role, Network: k.network, Listen: l.Listens[k.role]})
		}
	}
	l.BackupDirs = nil
	l.StopTimeout = azurStopTimeout
	l.Executable = filepath.Join(l.Dir, exeName(azurExecutable))
}

func (a *azurpromiliaHosting) checkBuild(path string) *v1.GameServerBuildInfo {
	return checkGoBuild(path, azurModule, azurMainPath, azurVersionVar)
}

func (a *azurpromiliaHosting) resolveBuildPath(dir, path string) string {
	return resolveExecutablePath(dir, path, azurExecutable)
}

// countTopFiles 目录下（不含子目录）某个扩展名的文件数。
func countTopFiles(dir, ext string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ext) {
			n++
		}
	}
	return n, nil
}

// azurResourcesCheck 客户端导出资源：三个子目录都在，表数为 Excel 下的 JSON 文件数。
func azurResourcesCheck(root string) *v1.GameServerCheck {
	for _, d := range azurResourceDirs {
		st, err := os.Stat(filepath.Join(root, d))
		if err == nil && !st.IsDir() {
			err = fmt.Errorf("%s is not a directory", d)
		}
		if err != nil {
			return hostCheck("resources", "fail", map[string]string{"path": root}, err)
		}
	}
	n, err := countTopFiles(filepath.Join(root, "Excel"), ".json")
	if err != nil || n == 0 {
		return hostCheck("resources", "fail", map[string]string{"path": root}, err)
	}
	return hostCheck("resources", "ok", map[string]string{"path": root, "files": strconv.Itoa(n)}, nil)
}

// detect 识别服务端目录：可执行文件、配置、数据库、资源与手写规则、密钥、客户端应用凭据、端口占用。
func (a *azurpromiliaHosting) detect(ctx context.Context, dir, configFile string, listens map[string]string, selfID string) (*v1.DetectGameServerDirResponse, error) {
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
	out.Build = a.checkBuild(filepath.Join(dir, exeName(azurExecutable)))
	out.Checks = append(out.Checks, executableCheck(out.Build))
	l, doc, ok := detectConfig(out, dir, configFile, azurConfigFile, azurTemplate)
	if !ok {
		return out, nil
	}
	a.fillLayout(l, doc)
	db := map[string]string{"driver": "sqlite", "path": l.DBPath}
	if st, err := os.Stat(l.DBPath); err == nil {
		db["size"] = strconv.FormatInt(st.Size(), 10)
		out.Checks = append(out.Checks, hostCheck("database", "ok", db, nil))
	} else {
		out.Checks = append(out.Checks, hostCheck("database", "warn", db, nil))
	}
	out.Checks = append(out.Checks, azurResourcesCheck(l.ResDir))
	if n, err := countTopFiles(l.DataDir, ".json"); err == nil && n > 0 {
		out.Checks = append(out.Checks, hostCheck("data", "ok", map[string]string{"path": l.DataDir, "files": strconv.Itoa(n)}, nil))
	} else {
		out.Checks = append(out.Checks, hostCheck("data", "fail", map[string]string{"path": l.DataDir}, err))
	}
	out.MgpKeyPresent = l.MGPKeyID != "" && l.MGPSecret != ""
	out.MgpKeyId = l.MGPKeyID
	if out.MgpKeyPresent {
		out.Checks = append(out.Checks, hostCheck("mgpKey", "ok", map[string]string{"keyId": l.MGPKeyID}, nil))
	} else {
		out.Checks = append(out.Checks, hostCheck("mgpKey", "pending", nil, nil))
	}
	keys := a.configKeys()
	if appID := docString(doc, keys.MomokoAppID, ""); appID != "" && docString(doc, keys.MomokoAppSecret, "") != "" {
		out.Checks = append(out.Checks, hostCheck("momokoApp", "ok", map[string]string{"appId": appID}, nil))
	} else {
		out.Checks = append(out.Checks, hostCheck("momokoApp", "pending", nil, nil))
	}
	for _, k := range azurListens {
		out.Listens[k.role] = l.Listens[k.role]
		if v := strings.TrimSpace(listens[k.role]); v != "" {
			out.Listens[k.role] = v
		}
	}
	selfRunning := selfID != "" && a.h.serverRunning(ctx, selfID)
	for _, k := range azurListens {
		addr := out.Listens[k.role]
		if addr == "" {
			continue
		}
		params := map[string]string{"role": k.role, "listen": addr}
		free := netprobe.Free
		if k.network == "udp" {
			free = netprobe.FreeUDP
		}
		switch {
		case !validListen(addr):
			out.Checks = append(out.Checks, hostCheck("port", "fail", params, errors.New("invalid listen address")))
		case selfRunning:
			out.Checks = append(out.Checks, hostCheck("port", "pending", params, nil))
		default:
			if err := free(addr); err != nil {
				out.Checks = append(out.Checks, hostCheck("port", "warn", params, err))
			} else {
				out.Checks = append(out.Checks, hostCheck("port", "ok", params, nil))
			}
		}
	}
	out.Ok = checksOK(out.Checks)
	return out, nil
}

// configure 写服务端配置：不存在时从模板复制，写入监听地址与 MGP 密钥（其余字段原样保留）。
func (a *azurpromiliaHosting) configure(dir string, spec *v1.GameServerHostedSpec, keyID, secret string) (*gameHostLayout, error) {
	configFile := firstNonEmpty(spec.ConfigFile, azurConfigFile)
	path := filepath.Join(dir, configFile)
	doc, err := readConfigOrTemplate(dir, path, firstNonEmpty(spec.ConfigTemplate, azurTemplate))
	if err != nil {
		return nil, err
	}
	for _, k := range azurListens {
		v := strings.TrimSpace(spec.Listens[k.role])
		if v == "" {
			continue
		}
		if !validListen(v) {
			return nil, fmt.Errorf("invalid listen address %q", v)
		}
		if err := doc.Assign(k.path, v); err != nil {
			return nil, err
		}
	}
	if keyID != "" {
		keys := a.configKeys()
		if err := doc.Assign(keys.MGPKeyID, keyID); err != nil {
			return nil, err
		}
		if err := doc.Assign(keys.MGPSecret, secret); err != nil {
			return nil, err
		}
	}
	if err := writeConfigDoc(dir, path, doc); err != nil {
		return nil, err
	}
	l := &gameHostLayout{Dir: dir, ConfigFile: configFile, ConfigPath: path}
	a.fillLayout(l, doc)
	return l, nil
}

// processes 只有游戏服务端一个进程；Ctrl+C 停服时它保存全部玩家后退出。
func (a *azurpromiliaHosting) processes(l *gameHostLayout, opts *gameHostingOptions) []gameHostProcess {
	listens := map[string]string{}
	for _, k := range azurListens {
		if l.Listens[k.role] == "" || (k.role == "mgp" && l.MGPKeyID == "" && l.MGPSecret == "") {
			continue
		}
		listens[k.role] = l.Listens[k.role]
	}
	return []gameHostProcess{{
		Role:        "main",
		Command:     exeName(azurExecutable) + " -config " + quoteArg(l.ConfigFile),
		StopCommand: ctrlC,
		StopTimeout: l.StopTimeout,
		Listens:     listens,
	}}
}

// settingsSchema 配置字段的提示（config.Config 的结构；取值范围取自各包启动时的校验：pkg/log、db.Open、
// transport.ListenKCP、gateway.New、game.New、account.New、sdk.New、mgp.New）。服务端只在启动时读取配置，全部修改都要重启生效。
func (a *azurpromiliaHosting) settingsSchema() []*v1.GameSettingsSection {
	return []*v1.GameSettingsSection{
		{Key: "gateway", Fields: []*v1.GameSettingsField{
			gsField("gateway.kcp_listen", "listen", gsRequired),
			gsField("gateway.tcp_listen", "listen"),
			gsField("gateway.login_timeout_seconds", "int", gsRange(1, 0)),
			gsField("gateway.idle_timeout_seconds", "int", gsRange(1, 0)),
			gsField("gateway.send_queue", "int", gsRange(1, 0)),
			gsField("gateway.compress", "bool"),
			gsField("gateway.crc_rand_index", "int", gsRange(0, 4294967295)),
			gsField("gateway.kcp.interval", "int", gsRange(10, 5000)),
			gsField("gateway.kcp.mtu", "int", gsRange(50, 1500)),
			gsField("gateway.kcp.send_window", "int", gsRange(1, 0)),
			gsField("gateway.kcp.recv_window", "int", gsRange(1, 0)),
			gsField("gateway.kcp.close_flush_ms", "int", gsRange(1, 0)),
			gsField("gateway.kcp.frame_queue", "int", gsRange(1, 0)),
		}},
		{Key: "sdk", Fields: []*v1.GameSettingsField{
			gsField("sdk.listen", "listen", gsRequired),
			gsField("sdk.server.addr", "string", gsRequired),
			gsField("sdk.server.port", "int", gsRange(1, 65535)),
			gsField("sdk.server.bak_gate", "string"),
			gsField("sdk.server.id", "int"),
			gsField("sdk.server.tag", "string"),
			gsField("sdk.cdn", "url", gsRequired),
			gsField("sdk.cdn_bak", "list"),
			gsField("sdk.release_hot", "string", gsRequired),
			gsField("sdk.latest_version", "string"),
			gsField("sdk.launch_key", "secret", gsRequired),
			gsField("sdk.app_id", "string", gsRequired),
			gsField("sdk.app_key", "secret", gsRequired),
			gsField("sdk.password_key", "secret", gsRequired),
			gsField("sdk.token_ttl_hours", "int", gsRange(1, 0)),
			gsField("sdk.mail_code_ttl_seconds", "int", gsRange(1, 0)),
		}},
		{Key: "zone", Fields: []*v1.GameSettingsField{
			gsField("game.zone_id", "int", gsRange(0, 4294967295)),
			gsField("game.time_zone", "int", gsRange(-12, 14)),
		}},
		{Key: "game", Fields: []*v1.GameSettingsField{
			gsField("game.save_interval_seconds", "int", gsRange(1, 0)),
			gsField("game.offline_retention_seconds", "int", gsRange(1, 0)),
			gsField("game.mailbox", "int", gsRange(1, 0)),
		}},
		{Key: "momoko", Fields: []*v1.GameSettingsField{
			gsField("momoko.app_id", "string"),
			gsField("momoko.app_secret", "secret"),
		}},
		{Key: "mgp", Fields: []*v1.GameSettingsField{
			gsField("mgp.listen", "listen"),
			gsField("mgp.key_id", "string"),
			gsField("mgp.secret", "secret"),
			gsField("mgp.request_days", "int", gsRange(1, 0)),
		}},
		{Key: "database", Fields: []*v1.GameSettingsField{
			gsField("database.path", "path", gsRequired),
			gsField("database.max_open_connections", "int", gsRange(2, 1024)),
			gsField("database.max_idle_connections", "int", gsRange(1, 1024)),
			gsField("database.busy_timeout_seconds", "int", gsRange(1, 3600)),
		}},
		{Key: "paths", Fields: []*v1.GameSettingsField{
			gsField("resource_path", "path", gsRequired),
			gsField("data_path", "path", gsRequired),
		}},
		{Key: "log", Fields: []*v1.GameSettingsField{
			gsField("log.level", "enum", gsOptions("debug", "info", "warn", "error")),
			gsField("log.directory", "path"),
			gsField("log.console", "bool"),
			gsField("log.file", "bool"),
			gsField("log.max_size_mb", "int", gsRange(1, 0)),
			gsField("log.max_backups", "int", gsRange(1, 0)),
			gsField("log.packet_bodies", "bool"),
			gsField("log.packet_blacklist", "list"),
		}},
	}
}
