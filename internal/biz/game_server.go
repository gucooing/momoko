package biz

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	entgameserver "momoko/internal/data/ent/gen/gameserver"
	authpkg "momoko/pkg/auth"
	"momoko/pkg/mgp"
	"momoko/pkg/mgpauth"
	"momoko/pkg/response"
	"momoko/pkg/secretbox"
)

var (
	ErrGameServerNotFound   = response.BadRequest(404, "游戏服务器不存在")
	ErrGameServerNoMGP      = response.BadRequest(400, "这台服务器没有接入 MGP")
	ErrGameServerInvalid    = response.BadRequest(400, "游戏服务器参数无效")
	ErrGameAdapterNotFound  = response.BadRequest(400, "不支持的游戏")
	ErrGameAdapterModeLimit = response.BadRequest(400, "这款游戏不支持该接入方式")
)

// GameMGPError 把 MGP 错误映射为接口错误：连接不上为 502，其余沿用 MGP 的 HTTP 状态。
func GameMGPError(err error) error {
	var e *mgp.Error
	if !errors.As(err, &e) {
		return ErrSystem(err)
	}
	status := e.Status
	switch {
	case status == 0:
		status = 502
	case status == 401:
		// MGP 签名失败不能让前端当成 Momoko 登录失效。
		status = 502
	case status < 400:
		status = 502
	}
	return response.BadRequest(status, e.Code+": "+e.Message)
}

// 游戏服务器运行状态。
const (
	GameStateRunning     = "running"
	GameStateStopped     = "stopped"
	GameStateUnreachable = "unreachable"
	GameStateMaintenance = "maintenance"
	GameStateUnknown     = "unknown"
)

// 最近查看所在端。
const (
	GameRecentScopeAdmin    = "admin"
	GameRecentScopePassport = "passport"
)

// GameProbe 一次 MGP 探测结果。
type GameProbe struct {
	OK        bool
	LatencyMs int
	Error     string
	Status    string
	At        time.Time
}

// GameServerRepo 游戏服务器的持久化。
type GameServerRepo interface {
	Get(ctx context.Context, id string) (*gen.GameServer, error)
	List(ctx context.Context) ([]*gen.GameServer, error)
	Create(ctx context.Context, s *gen.GameServer) (*gen.GameServer, error)
	// Update 写回可修改字段（名称、标签、封面、备注、接入方式、MGP 地址与密钥、托管信息、通行证可见性、时区）。
	Update(ctx context.Context, s *gen.GameServer) (*gen.GameServer, error)
	// Delete 删除服务器及 Momoko 侧的关联数据（绑定、自助设置、配置草稿、邮件、兑换码、最近查看）；GM 记录保留。
	Delete(ctx context.Context, id string) error
	SaveManifest(ctx context.Context, id, manifest string, at time.Time) error
	SaveProbe(ctx context.Context, id string, p GameProbe) error
	TouchRecent(ctx context.Context, userID, serverID, scope string) error
	ListRecent(ctx context.Context, userID, scope string, limit int) ([]*gen.GameRecentServer, error)
	AddHistory(ctx context.Context, h *gen.GameServerHistory) error
	// FinishHistory 写回一条记录的结果（进行中 → 成功 / 失败 / 已回滚）。
	FinishHistory(ctx context.Context, id, result, message string, params map[string]string) error
	// ListHistory 按时间倒序分页列出记录（kind 为空表示全部）。
	ListHistory(ctx context.Context, serverID, kind string, offset, limit int) ([]*gen.GameServerHistory, int, error)
	// LastHistory 最近一条匹配的记录（result 为空表示不限）。
	LastHistory(ctx context.Context, serverID, kind, action, result string) (*gen.GameServerHistory, error)
	// FailInterrupted 把重启前仍在进行中的记录与备份标记为失败。
	FailInterrupted(ctx context.Context, message string) error
	// GetInstance 按 id 读取应用实例（不按用户过滤；托管进程由游戏服务器管理权限控制）。
	GetInstance(ctx context.Context, id string) (*gen.Instance, error)
	CreateBackup(ctx context.Context, b *gen.GameServerBackup) (*gen.GameServerBackup, error)
	UpdateBackup(ctx context.Context, b *gen.GameServerBackup) error
	GetBackup(ctx context.Context, id string) (*gen.GameServerBackup, error)
	// ListBackups 按时间倒序分页列出备份（limit <= 0 表示全部）。
	ListBackups(ctx context.Context, serverID string, offset, limit int) ([]*gen.GameServerBackup, int, error)
	DeleteBackup(ctx context.Context, id string) error
}

// GameServerUsecase 游戏服务器：接入信息、MGP 客户端、manifest / status 缓存与定时探测。
// GM 运营、通行证、邮件与兑换码都经它拿到服务器与 MGP 客户端。
type GameServerUsecase struct {
	repo GameServerRepo
	log  *log.Helper

	mu        sync.Mutex
	clients   map[string]*gameClient
	manifests map[string]*cachedManifest
	statuses  map[string]*cachedStatus
}

type gameClient struct {
	fingerprint string
	client      *mgp.Client
}

type cachedManifest struct {
	at       time.Time
	manifest *mgp.Manifest
}

type cachedStatus struct {
	at     time.Time
	status *mgp.Status
	err    error
}

const (
	gameManifestTTL = 10 * time.Minute
	gameStatusTTL   = 5 * time.Second
	gameProbeEvery  = 30 * time.Second
)

func NewGameServerUsecase(repo GameServerRepo, logger log.Logger) *GameServerUsecase {
	uc := &GameServerUsecase{
		repo:      repo,
		log:       log.NewHelper(log.With(logger, "module", "biz/game")),
		clients:   map[string]*gameClient{},
		manifests: map[string]*cachedManifest{},
		statuses:  map[string]*cachedStatus{},
	}
	go uc.probeLoop()
	return uc
}

func gameBox() *secretbox.Box { return secretbox.New(authpkg.AuthSecretKey) }

// EncryptGameSecret 加密 MGP 密钥后再落库。
func EncryptGameSecret(secret string) (string, error) { return gameBox().Encrypt(secret) }

// Get 读取服务器。
func (uc *GameServerUsecase) Get(ctx context.Context, id string) (*gen.GameServer, error) {
	s, err := uc.repo.Get(ctx, id)
	if gen.IsNotFound(err) {
		return nil, ErrGameServerNotFound
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	return s, nil
}

// List 读取全部服务器（几百台以内，内存筛选）。
func (uc *GameServerUsecase) List(ctx context.Context) ([]*gen.GameServer, error) {
	list, err := uc.repo.List(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return list, nil
}

// Repo 暴露持久化（托管、更新与备份等同属游戏服务器管理的用例使用）。
func (uc *GameServerUsecase) Repo() GameServerRepo { return uc.repo }

// MGPConfigured 判断服务器是否接入了 MGP。
func MGPConfigured(s *gen.GameServer) bool {
	return s.MgpEndpoint != "" && s.KeyID != "" && s.Secret != ""
}

// Client 返回服务器的 MGP 客户端（按接入信息缓存）。
func (uc *GameServerUsecase) Client(s *gen.GameServer) (*mgp.Client, error) {
	if !MGPConfigured(s) {
		return nil, ErrGameServerNoMGP
	}
	fp := s.MgpEndpoint + "\x00" + s.KeyID + "\x00" + s.Secret
	uc.mu.Lock()
	defer uc.mu.Unlock()
	if c, ok := uc.clients[s.ID]; ok && c.fingerprint == fp {
		return c.client, nil
	}
	secret, err := gameBox().Decrypt(s.Secret)
	if err != nil {
		return nil, response.BadRequest(400, "MGP 密钥无法解密（主密钥变更或数据损坏），请重新填写密钥")
	}
	client, err := mgp.NewClient(mgp.Credentials{Endpoint: s.MgpEndpoint, KeyID: s.KeyID, Secret: secret})
	if err != nil {
		return nil, response.BadRequest(400, err.Error())
	}
	uc.clients[s.ID] = &gameClient{fingerprint: fp, client: client}
	return client, nil
}

// ClientOf 读取服务器并返回其 MGP 客户端。
func (uc *GameServerUsecase) ClientOf(ctx context.Context, id string) (*gen.GameServer, *mgp.Client, error) {
	s, err := uc.Get(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	c, err := uc.Client(s)
	if err != nil {
		return nil, nil, err
	}
	return s, c, nil
}

func (uc *GameServerUsecase) forget(id string) {
	uc.mu.Lock()
	delete(uc.clients, id)
	delete(uc.manifests, id)
	delete(uc.statuses, id)
	uc.mu.Unlock()
}

// Manifest 返回服务器的 manifest：内存缓存 10 分钟，其次读库里的缓存，都没有时现拉。
func (uc *GameServerUsecase) Manifest(ctx context.Context, s *gen.GameServer) (*mgp.Manifest, error) {
	uc.mu.Lock()
	if c, ok := uc.manifests[s.ID]; ok && time.Since(c.at) < gameManifestTTL {
		uc.mu.Unlock()
		return c.manifest, nil
	}
	uc.mu.Unlock()
	if m, err := uc.FetchManifest(ctx, s); err == nil {
		return m, nil
	} else if s.Manifest == "" {
		return nil, err
	}
	// 服务端暂时不可达：用库里最近一次的缓存。
	m, err := mgp.ParseManifest(json.RawMessage(s.Manifest))
	if err != nil {
		return nil, ErrSystem(err)
	}
	return m, nil
}

// FetchManifest 现拉 manifest 并写入缓存与库。
func (uc *GameServerUsecase) FetchManifest(ctx context.Context, s *gen.GameServer) (*mgp.Manifest, error) {
	c, err := uc.Client(s)
	if err != nil {
		return nil, err
	}
	raw, err := c.Get(ctx, "/manifest", nil)
	if err != nil {
		return nil, GameMGPError(err)
	}
	m, err := mgp.ParseManifest(raw)
	if err != nil {
		return nil, response.BadRequest(502, "manifest 无法解析："+err.Error())
	}
	now := time.Now()
	uc.mu.Lock()
	uc.manifests[s.ID] = &cachedManifest{at: now, manifest: m}
	uc.mu.Unlock()
	if string(raw) != s.Manifest {
		if err := uc.repo.SaveManifest(ctx, s.ID, string(raw), now); err != nil {
			uc.log.Warnf("save manifest of %s: %v", s.ID, err)
		}
		s.Manifest = string(raw)
	}
	return m, nil
}

// Status 返回实时 status（5 秒内复用），并记录探测结果。
func (uc *GameServerUsecase) Status(ctx context.Context, s *gen.GameServer) (*mgp.Status, error) {
	uc.mu.Lock()
	if c, ok := uc.statuses[s.ID]; ok && time.Since(c.at) < gameStatusTTL {
		uc.mu.Unlock()
		return c.status, c.err
	}
	uc.mu.Unlock()
	st, err := uc.probe(ctx, s)
	return st, err
}

func (uc *GameServerUsecase) probe(ctx context.Context, s *gen.GameServer) (*mgp.Status, error) {
	c, err := uc.Client(s)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	raw, err := c.Get(ctx, "/status", nil)
	p := GameProbe{At: time.Now(), LatencyMs: int(time.Since(start).Milliseconds())}
	var st *mgp.Status
	if err == nil {
		st, err = mgp.ParseStatus(raw)
		if err == nil {
			p.OK = true
			p.Status = string(raw)
		}
	}
	if err != nil {
		p.Error = err.Error()
		if code := mgp.CodeOf(err); code != "" {
			p.Error = code
		}
		err = GameMGPError(err)
	}
	uc.mu.Lock()
	uc.statuses[s.ID] = &cachedStatus{at: time.Now(), status: st, err: err}
	uc.mu.Unlock()
	if saveErr := uc.repo.SaveProbe(context.WithoutCancel(ctx), s.ID, p); saveErr != nil {
		uc.log.Warnf("save probe of %s: %v", s.ID, saveErr)
	}
	s.ProbeOk, s.ProbeLatencyMs, s.ProbeError, s.ProbeAt = p.OK, p.LatencyMs, p.Error, &p.At
	if p.OK {
		s.Status, s.StatusAt, s.LastOnlineAt = p.Status, &p.At, &p.At
	}
	return st, err
}

// probeLoop 定时探测全部接入了 MGP 的服务器（status 与延迟；manifest 过期时顺带刷新）。
func (uc *GameServerUsecase) probeLoop() {
	time.Sleep(5 * time.Second)
	ticker := time.NewTicker(gameProbeEvery)
	defer ticker.Stop()
	for {
		uc.probeAll()
		<-ticker.C
	}
}

func (uc *GameServerUsecase) probeAll() {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	list, err := uc.repo.List(ctx)
	if err != nil {
		return
	}
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for _, s := range list {
		if !MGPConfigured(s) {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(s *gen.GameServer) {
			defer wg.Done()
			defer func() { <-sem }()
			if _, err := uc.probe(ctx, s); err != nil {
				return
			}
			if s.ManifestAt == nil || time.Since(*s.ManifestAt) > gameManifestTTL {
				_, _ = uc.FetchManifest(ctx, s)
			}
		}(s)
	}
	wg.Wait()
}

// Location 返回服务器的时区（「今天」与每日次数按它的自然日计算）。
func (uc *GameServerUsecase) Location(s *gen.GameServer) *time.Location {
	if s.Timezone != "" {
		if loc, err := time.LoadLocation(s.Timezone); err == nil {
			return loc
		}
	}
	return time.Local
}

// DayStart 返回服务器时区下 t 所在自然日的开始。
func (uc *GameServerUsecase) DayStart(s *gen.GameServer, t time.Time) time.Time {
	t = t.In(uc.Location(s))
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// cachedStatusOf 解析库里的 status 缓存。
func cachedStatusOf(s *gen.GameServer) *mgp.Status {
	if s.Status == "" {
		return nil
	}
	st, err := mgp.ParseStatus(json.RawMessage(s.Status))
	if err != nil {
		return nil
	}
	return st
}

// cachedManifestOf 解析库里的 manifest 缓存。
func cachedManifestOf(s *gen.GameServer) *mgp.Manifest {
	if s.Manifest == "" {
		return nil
	}
	m, err := mgp.ParseManifest(json.RawMessage(s.Manifest))
	if err != nil {
		return nil
	}
	return m
}

// gameProbeFresh 探测结果在 2 个探测周期内视为有效。
func gameProbeFresh(s *gen.GameServer) bool {
	return s.ProbeAt != nil && time.Since(*s.ProbeAt) < 3*gameProbeEvery
}

// StateOf 推断运行状态：仅 GM 以 MGP 可达为准；托管服务器的进程状态由托管用例补充（见 HostedState）。
func (uc *GameServerUsecase) StateOf(s *gen.GameServer) string {
	if !MGPConfigured(s) {
		return GameStateUnknown
	}
	if !gameProbeFresh(s) {
		return GameStateUnknown
	}
	if !s.ProbeOk {
		return GameStateUnreachable
	}
	if st := cachedStatusOf(s); st != nil && st.Maintenance.Enabled {
		return GameStateMaintenance
	}
	return GameStateRunning
}

// Brief 生成服务器摘要。
func (uc *GameServerUsecase) Brief(ctx context.Context, s *gen.GameServer) *v1.GameServerBrief {
	b := &v1.GameServerBrief{
		Id:            s.ID,
		Game:          s.Game,
		Name:          s.Name,
		Mode:          string(s.Mode),
		Tags:          s.Tags,
		State:         uc.StateOf(s),
		Online:        -1,
		Registered:    -1,
		MgpOk:         s.ProbeOk && gameProbeFresh(s),
		MgpLatencyMs:  int32(s.ProbeLatencyMs),
		MgpConfigured: MGPConfigured(s),
		Cover:         s.Cover,
		NodeId:        s.NodeID,
	}
	if b.Tags == nil {
		b.Tags = []string{}
	}
	if a, ok := GameAdapterByID(s.Game); ok && s.Mode == "hosted" && a.Hosting != nil {
		if state := a.Hosting.State(ctx, s); state != "" && !(state == GameStateRunning && b.State == GameStateMaintenance) {
			b.State = state
		}
	}
	if m := cachedManifestOf(s); m != nil {
		b.GameName, b.Icon, b.Theme = m.Game.Name, m.Game.Icon, m.Game.Theme
	}
	if st := cachedStatusOf(s); st != nil && b.MgpOk {
		b.Online, b.Registered = int32(st.Online), int32(st.Registered)
	}
	return b
}

// Search 搜索服务器（名称、标签、游戏），最多 limit 条；q 为空时同时返回最近查看。
func (uc *GameServerUsecase) Search(ctx context.Context, userID, scope, q, game string, limit int, filter func(*gen.GameServer) bool) (*v1.SearchGameServersResponse, error) {
	if limit <= 0 || limit > 20 {
		limit = 20
	}
	list, err := uc.List(ctx)
	if err != nil {
		return nil, err
	}
	q = strings.ToLower(strings.TrimSpace(q))
	byID := make(map[string]*gen.GameServer, len(list))
	out := &v1.SearchGameServersResponse{}
	var matched []*gen.GameServer
	for _, s := range list {
		if filter != nil && !filter(s) {
			continue
		}
		byID[s.ID] = s
		out.Total++
		if game != "" && s.Game != game {
			continue
		}
		if q != "" && !gameServerMatches(s, q) {
			continue
		}
		matched = append(matched, s)
	}
	sort.SliceStable(matched, func(i, j int) bool { return matched[i].Name < matched[j].Name })
	for i, s := range matched {
		if i >= limit {
			break
		}
		out.Items = append(out.Items, uc.Brief(ctx, s))
	}
	if q == "" && userID != "" {
		recents, err := uc.repo.ListRecent(ctx, userID, scope, 5)
		if err == nil {
			for _, r := range recents {
				if s, ok := byID[r.ServerID]; ok {
					b := uc.Brief(ctx, s)
					b.ViewedAt = timestamppb.New(r.ViewedAt)
					out.Recent = append(out.Recent, b)
				}
			}
		}
	}
	return out, nil
}

func gameServerMatches(s *gen.GameServer, q string) bool {
	if strings.Contains(strings.ToLower(s.Name), q) || strings.Contains(strings.ToLower(s.Game), q) ||
		strings.Contains(strings.ToLower(s.MgpEndpoint), q) {
		return true
	}
	for _, t := range s.Tags {
		if strings.Contains(strings.ToLower(t), q) {
			return true
		}
	}
	if m := cachedManifestOf(s); m != nil && strings.Contains(strings.ToLower(m.Game.Name), q) {
		return true
	}
	return false
}

// TouchRecent 记入「最近查看」（失败不影响请求）。
func (uc *GameServerUsecase) TouchRecent(ctx context.Context, userID, serverID, scope string) {
	if userID == "" {
		return
	}
	if err := uc.repo.TouchRecent(ctx, userID, serverID, scope); err != nil {
		uc.log.Warnf("touch recent server: %v", err)
	}
}

// JSONValue 把原始 JSON 转为 protobuf Value（透传给前端）。
func JSONValue(raw json.RawMessage) *structpb.Value {
	if len(raw) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	pv, err := structpb.NewValue(v)
	if err != nil {
		return nil
	}
	return pv
}

// ValueJSON 把 protobuf Value 转回原始 JSON（nil 为 null）。
func ValueJSON(v *structpb.Value) json.RawMessage {
	if v == nil {
		return json.RawMessage("null")
	}
	raw, err := v.MarshalJSON()
	if err != nil {
		return json.RawMessage("null")
	}
	return raw
}

// Info 生成服务器详情。
func (uc *GameServerUsecase) Info(ctx context.Context, s *gen.GameServer) *v1.GameServerInfo {
	info := &v1.GameServerInfo{
		Brief:              uc.Brief(ctx, s),
		MgpEndpoint:        s.MgpEndpoint,
		KeyId:              s.KeyID,
		ProbeError:         s.ProbeError,
		InstallDir:         s.InstallDir,
		PassportVisibility: string(s.PassportVisibility),
		PassportInviteCode: s.PassportInviteCode,
		Timezone:           s.Timezone,
		Remark:             s.Remark,
		Status:             JSONValue(json.RawMessage(s.Status)),
		CreateTime:         timestamppb.New(s.CreateTime),
		UpdateTime:         timestamppb.New(s.UpdateTime),
	}
	if s.KeyRotatedAt != nil {
		info.KeyRotatedAt = timestamppb.New(*s.KeyRotatedAt)
	}
	if s.ProbeAt != nil {
		info.ProbeAt = timestamppb.New(*s.ProbeAt)
	}
	if s.LastOnlineAt != nil {
		info.LastOnlineAt = timestamppb.New(*s.LastOnlineAt)
	}
	if s.StatusAt != nil {
		info.StatusAt = timestamppb.New(*s.StatusAt)
	}
	if m := cachedManifestOf(s); m != nil {
		info.Protocol = int32(m.Protocol)
		info.ServerVersion = m.Server.Version
		info.ViewCount = int32(len(m.Views))
		info.CommandCount = int32(len(m.Commands))
		info.ConfigCount = int32(len(m.Configs))
	}
	if a, ok := GameAdapterByID(s.Game); ok {
		info.Capabilities = a.ServerCapabilities(s)
		if s.Mode == entgameserver.ModeHosted && a.Hosting != nil {
			a.Hosting.Describe(ctx, s, info)
		}
	}
	if len(info.Processes) == 0 {
		for _, p := range s.Processes {
			info.Processes = append(info.Processes, &v1.GameServerProcessInfo{Role: p.Role, InstanceId: p.InstanceID, State: GameStateUnknown})
		}
	}
	return info
}

// ListServers 服务器列表：筛选、计数、标签与顶部统计。
func (uc *GameServerUsecase) ListServers(ctx context.Context, req *v1.ListGameServersRequest) (*v1.ListGameServersResponse, error) {
	list, err := uc.List(ctx)
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(req.Q))
	type row struct {
		s     *gen.GameServer
		brief *v1.GameServerBrief
	}
	rows := make([]row, 0, len(list))
	out := &v1.ListGameServersResponse{Counts: map[string]int32{}, Summary: &v1.GameServersSummary{}}
	games := map[string]struct{}{}
	tags := map[string]int32{}
	for _, s := range list {
		b := uc.Brief(ctx, s)
		rows = append(rows, row{s: s, brief: b})
		out.Summary.Total++
		games[s.Game] = struct{}{}
		if b.State == GameStateRunning || b.State == GameStateMaintenance {
			out.Summary.Running++
		}
		if b.Online > 0 {
			out.Summary.Online += b.Online
		}
	}
	out.Summary.Games = int32(len(games))
	match := func(r row, skip string) bool {
		if q != "" && !gameServerMatches(r.s, q) && !strings.Contains(strings.ToLower(r.s.NodeID), q) {
			return false
		}
		if skip != "game" && req.Game != "" && r.s.Game != req.Game {
			return false
		}
		if skip != "mode" && req.Mode != "" && string(r.s.Mode) != req.Mode {
			return false
		}
		if skip != "state" && req.State != "" && r.brief.State != req.State {
			return false
		}
		if skip != "tag" && req.Tag != "" {
			found := false
			for _, t := range r.s.Tags {
				if t == req.Tag {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	var filtered []row
	for _, r := range rows {
		if match(r, "mode") {
			out.Counts["mode:all"]++
			out.Counts["mode:"+string(r.s.Mode)]++
		}
		if match(r, "game") {
			out.Counts["game:all"]++
			out.Counts["game:"+r.s.Game]++
		}
		if match(r, "state") {
			out.Counts["state:all"]++
			out.Counts["state:"+r.brief.State]++
		}
		if match(r, "tag") {
			for _, t := range r.s.Tags {
				tags[t]++
			}
		}
		if match(r, "") {
			filtered = append(filtered, r)
		}
	}
	for t, n := range tags {
		out.Tags = append(out.Tags, &v1.GameServerTagCount{Tag: t, Count: n})
	}
	sort.Slice(out.Tags, func(i, j int) bool {
		if out.Tags[i].Count != out.Tags[j].Count {
			return out.Tags[i].Count > out.Tags[j].Count
		}
		return out.Tags[i].Tag < out.Tags[j].Tag
	})
	sort.SliceStable(filtered, func(i, j int) bool { return filtered[i].s.CreateTime.Before(filtered[j].s.CreateTime) })
	out.Total = int32(len(filtered))
	page, size := int(req.Page), int(req.PageSize)
	if size <= 0 || size > 200 {
		size = 24
	}
	if page <= 0 {
		page = 1
	}
	for i := (page - 1) * size; i < len(filtered) && i < page*size; i++ {
		out.Items = append(out.Items, uc.Info(ctx, filtered[i].s))
	}
	return out, nil
}

// normalizeTags 去空白、去重，最多 20 个。
func normalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" || len([]rune(t)) > 32 {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
		if len(out) == 20 {
			break
		}
	}
	return out
}

// CreateServer 接入服务器（仅 GM 方式在这里完成；托管方式由托管用例在此基础上安装与创建进程）。
func (uc *GameServerUsecase) CreateServer(ctx context.Context, req *v1.CreateGameServerRequest) (*gen.GameServer, error) {
	adapter, ok := GameAdapterByID(req.Game)
	if !ok || len(adapter.Modes()) == 0 {
		return nil, ErrGameAdapterNotFound
	}
	mode := req.Mode
	if mode == "" {
		mode = "remote"
	}
	// 托管接入由托管用例完成（安装、写配置、创建进程），这里只处理仅 GM。
	if mode != "remote" || !adapter.SupportsMode(mode) {
		return nil, ErrGameAdapterModeLimit
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, response.BadRequest(400, "名称不能为空")
	}
	s := &gen.GameServer{
		ID:     uuid.NewString(),
		Game:   req.Game,
		Name:   name,
		Tags:   normalizeTags(req.Tags),
		Mode:   gameServerMode(mode),
		Cover:  req.Cover,
		Remark: req.Remark,
	}
	if req.MgpEndpoint != "" || req.KeyId != "" || req.Secret != "" {
		if err := uc.applyMGP(s, req.MgpEndpoint, req.KeyId, req.Secret); err != nil {
			return nil, err
		}
	} else if mode == "remote" {
		return nil, response.BadRequest(400, "请填写 MGP 地址、Key ID 与密钥")
	}
	created, err := uc.repo.Create(ctx, s)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if MGPConfigured(created) {
		_, _ = uc.FetchManifest(ctx, created)
		_, _ = uc.probe(ctx, created)
	}
	return created, nil
}

func (uc *GameServerUsecase) applyMGP(s *gen.GameServer, endpoint, keyID, secret string) error {
	if strings.TrimSpace(endpoint) == "" {
		s.MgpEndpoint, s.KeyID, s.Secret = "", "", ""
		return nil
	}
	normalized, err := mgp.NormalizeEndpoint(endpoint)
	if err != nil {
		return response.BadRequest(400, err.Error())
	}
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return response.BadRequest(400, "Key ID 不能为空")
	}
	if secret != "" {
		// 新密钥由前端生成：校验格式后加密保存。
		if keyID, secret, err = ValidateMGPKey(keyID, secret); err != nil {
			return err
		}
	}
	s.MgpEndpoint, s.KeyID = normalized, keyID
	if secret != "" {
		enc, err := EncryptGameSecret(secret)
		if err != nil {
			return ErrSystem(err)
		}
		s.Secret = enc
	}
	if s.Secret == "" {
		return response.BadRequest(400, "密钥不能为空")
	}
	return nil
}

// UpdateServer 修改服务器的可编辑字段。
func (uc *GameServerUsecase) UpdateServer(ctx context.Context, req *v1.UpdateGameServerRequest) (*gen.GameServer, error) {
	s, err := uc.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		s.Name = name
	}
	s.Tags = normalizeTags(req.Tags)
	s.Cover, s.Remark = req.Cover, req.Remark
	// 托管服务器的 MGP 地址与密钥由服务端配置决定（轮换密钥 / 专用设置），这里不改。
	if s.Mode != entgameserver.ModeHosted {
		if err := uc.applyMGP(s, req.MgpEndpoint, req.KeyId, req.Secret); err != nil {
			return nil, err
		}
	}
	switch req.PassportVisibility {
	case "public", "invite", "hidden":
		s.PassportVisibility = gameVisibility(req.PassportVisibility)
	case "":
	default:
		return nil, ErrGameServerInvalid
	}
	s.PassportInviteCode = strings.TrimSpace(req.PassportInviteCode)
	if s.PassportVisibility == "invite" && s.PassportInviteCode == "" {
		s.PassportInviteCode = strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", "")[:8])
	}
	if req.Timezone != "" {
		if _, err := time.LoadLocation(req.Timezone); err != nil {
			return nil, response.BadRequest(400, "时区无效")
		}
	}
	s.Timezone = req.Timezone
	updated, err := uc.repo.Update(ctx, s)
	if err != nil {
		return nil, ErrSystem(err)
	}
	uc.forget(s.ID)
	if MGPConfigured(updated) {
		_, _ = uc.FetchManifest(ctx, updated)
		_, _ = uc.probe(ctx, updated)
	}
	return updated, nil
}

// DeleteServer 删除服务器的接入记录与 Momoko 侧数据。
func (uc *GameServerUsecase) DeleteServer(ctx context.Context, id string) error {
	if _, err := uc.Get(ctx, id); err != nil {
		return err
	}
	if err := uc.repo.Delete(ctx, id); err != nil {
		return ErrSystem(err)
	}
	uc.forget(id)
	return nil
}

// TestServer 测试 MGP 连接：连接、签名、manifest、status。
func (uc *GameServerUsecase) TestServer(ctx context.Context, req *v1.TestGameServerRequest) (*v1.TestGameServerResponse, error) {
	endpoint, keyID, secret := req.MgpEndpoint, req.KeyId, req.Secret
	if req.Id != "" {
		s, err := uc.Get(ctx, req.Id)
		if err != nil {
			return nil, err
		}
		if endpoint == "" {
			endpoint = s.MgpEndpoint
		}
		if keyID == "" {
			keyID = s.KeyID
		}
		if secret == "" && s.Secret != "" {
			secret, _ = gameBox().Decrypt(s.Secret)
		}
	}
	out := &v1.TestGameServerResponse{}
	add := func(key, state string, params map[string]string, err error) {
		c := &v1.GameServerCheck{Key: key, State: state, Params: params}
		if err != nil {
			c.ErrorCode = mgp.CodeOf(err)
			c.ErrorMessage = err.Error()
		}
		out.Checks = append(out.Checks, c)
	}
	client, err := mgp.NewClient(mgp.Credentials{Endpoint: endpoint, KeyID: keyID, Secret: secret})
	if err != nil {
		add("connect", "fail", map[string]string{"endpoint": endpoint}, err)
		add("signature", "pending", nil, nil)
		add("manifest", "pending", nil, nil)
		add("status", "pending", nil, nil)
		return out, nil
	}
	normalized, _ := mgp.NormalizeEndpoint(endpoint)
	start := time.Now()
	raw, err := client.Get(ctx, "/status", nil)
	latency := time.Since(start)
	switch {
	case err != nil && mgp.IsUnreachable(err):
		add("connect", "fail", map[string]string{"endpoint": normalized}, err)
		add("signature", "pending", nil, nil)
		add("manifest", "pending", nil, nil)
		add("status", "pending", nil, nil)
		return out, nil
	case err != nil && (mgp.CodeOf(err) == "SIGNATURE_INVALID" || mgp.CodeOf(err) == "SIGNATURE_MISSING" || mgp.CodeOf(err) == "CLOCK_SKEW" || mgp.CodeOf(err) == "REPLAY"):
		add("connect", "ok", map[string]string{"endpoint": normalized, "latencyMs": itoa(int(latency.Milliseconds()))}, nil)
		add("signature", "fail", map[string]string{"keyId": keyID}, err)
		add("manifest", "pending", nil, nil)
		add("status", "pending", nil, nil)
		return out, nil
	case err != nil:
		add("connect", "fail", map[string]string{"endpoint": normalized}, err)
		return out, nil
	}
	add("connect", "ok", map[string]string{"endpoint": normalized, "latencyMs": itoa(int(latency.Milliseconds()))}, nil)
	st, _ := mgp.ParseStatus(raw)
	skew := ""
	if st != nil && st.Now > 0 {
		d := time.Since(time.UnixMilli(st.Now)) - latency/2
		if d < 0 {
			d = -d
		}
		skew = strconvFormatSeconds(d)
	}
	add("signature", "ok", map[string]string{"keyId": keyID, "skewSeconds": skew}, nil)
	mraw, err := client.Get(ctx, "/manifest", nil)
	if err != nil {
		add("manifest", "fail", nil, err)
	} else if m, perr := mgp.ParseManifest(mraw); perr != nil {
		add("manifest", "fail", nil, perr)
	} else {
		add("manifest", "ok", map[string]string{
			"game":     m.Game.ID,
			"gameName": m.Game.Name,
			"protocol": itoa(m.Protocol),
			"commands": itoa(len(m.Commands)),
			"views":    itoa(len(m.Views)),
			"configs":  itoa(len(m.Configs)),
		}, nil)
	}
	if st != nil {
		add("status", "ok", map[string]string{
			"online":     itoa(st.Online),
			"registered": itoa(int(st.Registered)),
			"version":    st.Version,
		}, nil)
	}
	out.Ok = true
	for _, c := range out.Checks {
		if c.State == "fail" {
			out.Ok = false
		}
	}
	return out, nil
}

// RefreshServer 重新拉取 manifest 与 status。
func (uc *GameServerUsecase) RefreshServer(ctx context.Context, id string) (*gen.GameServer, error) {
	s, err := uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !MGPConfigured(s) {
		return s, nil
	}
	uc.forget(id)
	if _, err := uc.FetchManifest(ctx, s); err != nil {
		uc.log.Warnf("refresh manifest of %s: %v", id, err)
	}
	_, _ = uc.probe(ctx, s)
	return uc.Get(ctx, id)
}

// MGPQuery 解析前端透传的查询串（只保留合法的键值）。
func MGPQuery(raw string) url.Values {
	if raw == "" {
		return nil
	}
	v, err := url.ParseQuery(strings.TrimPrefix(raw, "?"))
	if err != nil {
		return nil
	}
	return v
}

// ValidateMGPKey 校验（由前端生成的）MGP Key ID 与密钥，返回去掉首尾空白后的值。
func ValidateMGPKey(keyID, secret string) (string, string, error) {
	keyID, secret = strings.TrimSpace(keyID), strings.TrimSpace(secret)
	if err := mgpauth.ValidateKey(keyID, secret); err != nil {
		return "", "", response.BadRequest(400, err.Error())
	}
	return keyID, secret, nil
}
