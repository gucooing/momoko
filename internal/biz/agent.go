package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/agentapi"
	"momoko/pkg/agentapi/anthropic"
	"momoko/pkg/agentapi/deepseek"
	"momoko/pkg/agentapi/openai"
	"momoko/pkg/agentdriver"
	"momoko/pkg/agentdriver/claude"
	"momoko/pkg/agentdriver/codex"
	"momoko/pkg/agentfs"
	"momoko/pkg/agentgit"
	"momoko/pkg/agenthub"
	"momoko/pkg/agentrun"
	"momoko/pkg/auth"
	"momoko/pkg/common"
	"momoko/pkg/constant"
	"momoko/pkg/response"
	"momoko/pkg/secretbox"
)

// Agent（Code 与 Agent 管理）的业务层：谁能用哪个会话和供应商、配置和密钥怎么存、会话的状态规则、
// 什么时候发通知。启动进程、ACP 协议、各家 Agent 的专门处理（pkg/agentdriver 下的驱动）、Git、文件、
// 登录命令这些底层都在 pkg 里，这里只做编排。
//
//   - agent.go          仓储接口、用例、公共工具、名称缓存、选择器和概览
//   - agent_manage.go   运行时、模型供应商、工作区、MCP、技能
//   - agent_notify.go   通知设置、批准令牌
//   - agent_session.go  会话：新建、发消息、切换选项、复刻、归档、删除、改动和文件、页面的长连接
//   - agent_live.go     活着的会话：进程池、引擎回调、按顺序落库和推送

// AgentWsPath 会话事件的 WebSocket 路径。
const AgentWsPath = "/api/v1/agent/ws"

// agentDataDir Agent 相关文件的根目录：下载的适配器、每个用户自己的技能（users/）、会话目录。
const agentDataDir = "data/agents"

var (
	ErrAgentSessionNotFound   = response.BadRequest("会话不存在")
	ErrAgentWorkspaceNotFound = response.BadRequest("工作区不存在")
	ErrAgentProviderNotFound  = response.BadRequest("供应商不存在")
	ErrAgentMcpNotFound       = response.BadRequest("MCP 服务器不存在")
	ErrAgentSkillNotFound     = response.BadRequest("技能不存在")
	ErrAgentRuntimeNotFound   = response.BadRequest("Agent 不存在")
	ErrAgentRuntimeNotReady   = response.BadRequest("这个 Agent 还没就绪，先到「运行时」里检测")
	ErrAgentSessionBusy       = response.BadRequest("这个会话正在运行，先停下来")
	ErrAgentSessionArchived   = response.BadRequest("这个会话已经归档，先取消归档")
)

// AgentSessionFilter 会话列表的筛选条件。UserID 为空表示所有人的会话。
type AgentSessionFilter struct {
	UserID   string
	Status   string
	Keyword  string
	Page     int
	PageSize int
}

// AgentSessionCounts 各状态的会话数。
type AgentSessionCounts struct {
	All, Running, Waiting, Idle, Error, Archived int64
}

// AgentWorkspaceStat 一个工作区的会话数和最近活动时间。
type AgentWorkspaceStat struct {
	Sessions int
	ActiveAt time.Time
}

// AgentCounts 管理页各标签上的数量。
type AgentCounts struct {
	Sessions, SessionsRunning, SessionsWaiting int
	Workspaces                                 int
	Providers                                  int
	McpServers, McpOK                          int
	Skills, SkillsEnabled                      int
	Runtimes, RuntimesReady                    int
}

// AgentUser 会话主人的名称和用户名。
type AgentUser struct {
	ID, Name, Username string
}

// AgentModelLimit 用户给服务器上的登录里一个模型设的最长上下文和最长输出（token 数，0 = 用 Agent 自己的）。
type AgentModelLimit struct {
	Model         string
	ContextWindow int64
	MaxOutput     int64
}

// agentRuntimeCatalog 运行时上记的 Agent 可选项：权限模式、默认权限模式、默认模型。模型在 agent_runtime_models 表里。
type agentRuntimeCatalog struct {
	Modes        []agentrun.Choice `json:"modes,omitempty"`
	DefaultMode  string            `json:"defaultMode,omitempty"`
	DefaultModel string            `json:"defaultModel,omitempty"`
}

type AgentRepo interface {
	ListAgentRuntimes(ctx context.Context) ([]*gen.AgentRuntime, error)
	GetAgentRuntime(ctx context.Context, id string) (*gen.AgentRuntime, error)
	CountAgentRuntimes(ctx context.Context) (int, error)
	SaveAgentRuntime(ctx context.Context, rt *gen.AgentRuntime) error
	DeleteAgentRuntime(ctx context.Context, id string) error

	SyncAgentRuntimeModels(ctx context.Context, runtimeID string, models []*gen.AgentRuntimeModel) error
	ListAgentRuntimeModels(ctx context.Context, runtimeIDs ...string) ([]*gen.AgentRuntimeModel, error)
	GetAgentRuntimeModel(ctx context.Context, runtimeID, model string) (*gen.AgentRuntimeModel, error)
	SetAgentRuntimeModelLimits(ctx context.Context, runtimeID string, limits []AgentModelLimit) error

	ListAgentProviders(ctx context.Context) ([]*gen.AgentProvider, error)
	GetAgentProvider(ctx context.Context, id string) (*gen.AgentProvider, error)
	SaveAgentProvider(ctx context.Context, p *gen.AgentProvider) error
	DeleteAgentProvider(ctx context.Context, id string) error

	// ListAgentWorkspaces userID 为空时是所有人的。
	ListAgentWorkspaces(ctx context.Context, userID string) ([]*gen.AgentWorkspace, error)
	GetAgentWorkspace(ctx context.Context, id string) (*gen.AgentWorkspace, error)
	FindAgentWorkspaceByPath(ctx context.Context, userID, path string) (*gen.AgentWorkspace, error)
	SaveAgentWorkspace(ctx context.Context, w *gen.AgentWorkspace) error
	DeleteAgentWorkspace(ctx context.Context, id string) error
	AgentWorkspaceStats(ctx context.Context, ids []string) (map[string]AgentWorkspaceStat, error)

	// ListAgentMcpServers userID 为空时是所有人的。
	ListAgentMcpServers(ctx context.Context, userID string) ([]*gen.AgentMcpServer, error)
	// ListAgentMcpServersByIDs 这些 id 里属于 userID 的。
	ListAgentMcpServersByIDs(ctx context.Context, userID string, ids []string) ([]*gen.AgentMcpServer, error)
	AgentMcpServerNames(ctx context.Context, userID string) ([]string, error)
	GetAgentMcpServer(ctx context.Context, id string) (*gen.AgentMcpServer, error)
	FindAgentMcpServerByName(ctx context.Context, userID, name string) (*gen.AgentMcpServer, error)
	SaveAgentMcpServer(ctx context.Context, m *gen.AgentMcpServer) error
	DeleteAgentMcpServer(ctx context.Context, id string) error

	// ListAgentSkills userID 为空时是所有人的。
	ListAgentSkills(ctx context.Context, userID string) ([]*gen.AgentSkill, error)
	ListEnabledAgentSkills(ctx context.Context, userID string) ([]*gen.AgentSkill, error)
	GetAgentSkill(ctx context.Context, id string) (*gen.AgentSkill, error)
	FindAgentSkillByName(ctx context.Context, userID, name string) (*gen.AgentSkill, error)
	SaveAgentSkill(ctx context.Context, s *gen.AgentSkill) error
	DeleteAgentSkill(ctx context.Context, id string) error
	// ClaimLegacyAgentConfigs 没记主人的 MCP 服务器和技能（以前是大家共用的）都归给 ownerID。
	ClaimLegacyAgentConfigs(ctx context.Context, ownerID string) error

	CreateAgentSession(ctx context.Context, s *gen.AgentSession) (*gen.AgentSession, error)
	ForkAgentSession(ctx context.Context, fork *gen.AgentSession, from string) (*gen.AgentSession, error)
	GetAgentSession(ctx context.Context, id string) (*gen.AgentSession, error)
	// UpdateAgentSession 改一个会话并返回改后的样子；fn 里加了条件而条件不满足时返回 NotFound。
	UpdateAgentSession(ctx context.Context, id string, fn func(u *gen.AgentSessionUpdateOne)) (*gen.AgentSession, error)
	// ListAgentSessions 返回这一页、符合筛选的总数、各状态的数量（不受状态筛选影响）。
	ListAgentSessions(ctx context.Context, f AgentSessionFilter) ([]*gen.AgentSession, int64, AgentSessionCounts, error)
	ListAgentSidebarSessions(ctx context.Context, userID, keyword string, limit int) ([]*gen.AgentSession, error)
	// ListAgentSessionsByIDs userID 不为空时只要这个人的。
	ListAgentSessionsByIDs(ctx context.Context, ids []string, userID string) ([]*gen.AgentSession, error)
	ListArchivedAgentSessions(ctx context.Context, userID string) ([]*gen.AgentSession, error)
	ListAgentForks(ctx context.Context, parentIDs []string) ([]*gen.AgentSession, error)
	CountAgentForks(ctx context.Context, parentIDs []string) (map[string]int, error)
	DetachAgentForks(ctx context.Context, parentIDs []string) error
	// ArchiveAgentSessions 返回状态真的改了的会话。
	ArchiveAgentSessions(ctx context.Context, ids []string, archived bool) ([]*gen.AgentSession, error)
	DeleteAgentSessions(ctx context.Context, ids []string) error
	ResetLiveAgentSessions(ctx context.Context) error

	SaveAgentEvents(ctx context.Context, events []*gen.AgentEvent) error
	ListAgentEventsAfter(ctx context.Context, sessionID string, afterRev int64) ([]*gen.AgentEvent, error)
	ListAgentEventsBefore(ctx context.Context, sessionID string, beforeSeq int64, limit int) ([]*gen.AgentEvent, bool, error)
	MaxAgentEvent(ctx context.Context, sessionID string) (seq, rev int64, err error)

	// AgentCounts userID 为空时会话、工作区、MCP、技能数所有人的。
	AgentCounts(ctx context.Context, userID string) (*AgentCounts, error)
	AgentUsers(ctx context.Context, ids []string) ([]AgentUser, error)
}

type AgentUsecase struct {
	repo     AgentRepo
	config   ConfigRepo
	messages MessageRepo
	system   *SystemUsecase
	box      *secretbox.Box
	log      *log.Helper

	// root Agent 文件根目录的绝对路径。
	root string
	// drivers 认得出的各系 Agent；认不出的用 agentdriver.Generic。
	drivers []agentdriver.Driver
	// vendors 能添加的供应商类型，按添加时列出的顺序。
	vendors []agentapi.Vendor

	hub   *agenthub.Hub
	pool  *agentrun.Pool[*agentLive]
	names agentNameCache
	// files 「@ 引用文件」按名字搜的文件列表缓存。
	files *agentfs.Index

	installMu  sync.Mutex
	installing map[string]bool

	approvals agentApprovals
	telegram  agentTelegram
}

// NewAgentUsecase 建好就能用；进程池的回收、Telegram 的轮询都是有事可做时才起，不在这里开协程。
func NewAgentUsecase(repo AgentRepo, config ConfigRepo, authRepo AuthRepo, system *SystemUsecase, logger log.Logger) (*AgentUsecase, func()) {
	root, err := filepath.Abs(agentDataDir)
	if err != nil {
		root = agentDataDir
	}
	uc := &AgentUsecase{
		repo:       repo,
		config:     config,
		messages:   authRepo,
		system:     system,
		box:        secretbox.New(auth.AuthSecretKey),
		log:        log.NewHelper(log.With(logger, "module", "agent")),
		root:       root,
		drivers:    []agentdriver.Driver{claude.Driver{}, codex.Driver{}},
		vendors:    []agentapi.Vendor{anthropic.Vendor{}, openai.Vendor{}, deepseek.Vendor{}},
		hub:        agenthub.New(),
		installing: make(map[string]bool),
		approvals:  newAgentApprovals(),
	}
	ctx := context.Background()
	limit, idle := uc.runLimits(ctx)
	uc.pool = agentrun.NewPool[*agentLive](limit, idle)
	uc.files = agentfs.NewIndex(15*time.Second, func(ctx context.Context, dir string) []string {
		if files, ok := agentgit.Files(ctx, dir); ok {
			return files
		}
		return agentfs.Walk(dir, 20000)
	})
	// 上次运行留下的「运行中」都已经不成立了：进程随 Momoko 一起没了。
	if err := repo.ResetLiveAgentSessions(ctx); err != nil {
		uc.log.Warnf("reset live agent sessions: %v", err)
	}
	uc.seedRuntimes(ctx)
	uc.claimLegacy(ctx)
	return uc, uc.close
}

// close 停服时结束所有 Agent 进程和 Telegram 轮询。
func (uc *AgentUsecase) close() {
	uc.pool.Close()
	uc.telegram.stop()
}

// —— 公共工具 ——

func agentUserID(ctx context.Context) (string, error) {
	a, ok := auth.FromContext(ctx)
	if !ok || a.UserID == "" {
		return "", ErrTokenInvalid
	}
	return a.UserID, nil
}

// 会话、工作区、MCP 服务器、技能都属于创建它们的人：Code 页和 Agent 管理页里每个人（超级管理员也一样）
// 只看得到、只用得上自己的。模型供应商、运行时、通知是大家共用的。
// 「全员 Agent」页（要有 agent:admin）能看所有人的，并能清理（归档、删除、导出会话，删工作区、MCP、技能），
// 但不能打开别人的会话、也不能改别人的配置。

// isAdmin 当前用户能不能看和清理所有人的：要有 agent:admin，而且是直接登录的（从远程节点用 API Key 过来的不算）。
func (uc *AgentUsecase) isAdmin(ctx context.Context) bool {
	a, ok := auth.FromContext(ctx)
	if !ok || a.Type != auth.AuthTypeJWT {
		return false
	}
	return uc.system.Check(ctx, constant.AgentAdmin) == nil
}

// listScope 列表要谁的：all 为 true 时是所有人的（返回空；没有 agent:admin 时报没有权限），否则是自己的。
func (uc *AgentUsecase) listScope(ctx context.Context, all bool) (string, error) {
	userID, err := agentUserID(ctx)
	if err != nil {
		return "", err
	}
	if !all {
		return userID, nil
	}
	if !uc.isAdmin(ctx) {
		return "", ErrNoPermission
	}
	return "", nil
}

// cleanScope 清理（归档、删除、导出、看进程）能动谁的：有 agent:admin 时为空（所有人），否则是自己。
func (uc *AgentUsecase) cleanScope(ctx context.Context) (string, error) {
	userID, err := agentUserID(ctx)
	if err != nil {
		return "", err
	}
	if uc.isAdmin(ctx) {
		return "", nil
	}
	return userID, nil
}

// ownedBy scope 能不能动 owner 的东西：scope 为空是谁的都行。
func ownedBy(scope, owner string) bool {
	return scope == "" || scope == owner
}

// agentUsersPB 这些用户的名称和用户名（列表上显示主人用）。
func (uc *AgentUsecase) agentUsersPB(ctx context.Context, ids []string) ([]*v1.AgentUser, error) {
	seen := map[string]bool{}
	uniq := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != "" && !seen[id] {
			seen[id] = true
			uniq = append(uniq, id)
		}
	}
	users, err := uc.repo.AgentUsers(ctx, uniq)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := make([]*v1.AgentUser, 0, len(users))
	for _, u := range users {
		out = append(out, &v1.AgentUser{Id: u.ID, Name: u.Name, Username: u.Username})
	}
	return out, nil
}

// userDir 一个用户自己的文件（技能）放在 Agent 根目录的 users/<键>/ 下。用户 id 里可能有冒号（Windows 上不能做目录名），键用它的摘要。
func (uc *AgentUsecase) userDir(userID string, sub ...string) string {
	sum := sha256.Sum256([]byte(userID))
	return filepath.Join(append([]string{uc.root, "users", hex.EncodeToString(sum[:8])}, sub...)...)
}

// claimLegacy 以前 MCP 服务器和技能是大家共用的：升级后没记主人的都归给内置超级管理员，
// 技能文件从原来的公共技能目录（skills/<名称>）挪进他自己的技能目录。挪不动的下次启动再挪。
func (uc *AgentUsecase) claimLegacy(ctx context.Context) {
	if err := uc.repo.ClaimLegacyAgentConfigs(ctx, BuiltinAdminUserID); err != nil {
		uc.log.Errorf("把以前共用的 MCP 服务器和技能归给内置超级管理员失败：%v", err)
		return
	}
	legacy := filepath.Join(uc.root, "skills")
	entries, err := os.ReadDir(legacy)
	if err != nil {
		return
	}
	skills, err := uc.repo.ListAgentSkills(ctx, BuiltinAdminUserID)
	if err != nil {
		return
	}
	owned := map[string]bool{}
	for _, s := range skills {
		owned[s.Name] = true
	}
	store := uc.skillStore(BuiltinAdminUserID)
	for _, e := range entries {
		if !e.IsDir() || !owned[e.Name()] {
			continue
		}
		to := store.Dir(e.Name())
		if _, err := os.Stat(to); err == nil {
			continue
		}
		if err := os.MkdirAll(store.Root, 0o755); err != nil {
			return
		}
		if err := os.Rename(filepath.Join(legacy, e.Name()), to); err != nil {
			uc.log.Warnf("挪技能 %s 失败：%v", e.Name(), err)
		}
	}
	// 挪空了就把旧目录删掉（还有别的东西时删不掉，留着）
	_ = os.Remove(legacy)
}

// dir 返回 Agent 根目录下的一个子目录，不存在就建出来。只在要往里写东西时调用。
func (uc *AgentUsecase) dir(parts ...string) string {
	return agentfs.EnsureDir(filepath.Join(append([]string{uc.root}, parts...)...))
}

// sessionDir 会话目录：这个会话在 Momoko 这边的东西都放在里面，删会话时整个删掉。
//
//	work/      没选工作区时的工作目录
//	worktree/  在独立工作树里运行时的工作树
//	uploads/   对话里的附件和图片：上传的附件，以及 Agent 工具调用产出的图片
//	tmp/       Agent 进程的临时目录
func (uc *AgentUsecase) sessionDir(id string, sub ...string) string {
	return filepath.Join(append([]string{uc.root, "sessions", id}, sub...)...)
}

// agentDirs 交给 Agent 的附加目录：会话自己选的附加文件夹，加上会话目录里的附件和临时目录（让它读写这两处不用批准）。
func (uc *AgentUsecase) agentDirs(id string, extra []string) []string {
	return append(slices.Clone(extra), agentfs.EnsureDir(uc.sessionDir(id, "uploads")), agentfs.EnsureDir(uc.sessionDir(id, "tmp")))
}

// driverOf 某一系的驱动；认不出的（custom）用 Generic。
func (uc *AgentUsecase) driverOf(kind string) agentdriver.Driver {
	for _, d := range uc.drivers {
		if d.Kind() == kind {
			return d
		}
	}
	return agentdriver.Generic{}
}

// vendorOf 某种供应商类型；认不出的 ok 为 false。
func (uc *AgentUsecase) vendorOf(id string) (agentapi.Vendor, bool) {
	for _, v := range uc.vendors {
		if v.ID() == id {
			return v, true
		}
	}
	return nil, false
}

// serves 这个供应商能不能给这一系 Agent 用：它的类型给不给这一系的协议。
func (uc *AgentUsecase) serves(p *gen.AgentProvider, d agentdriver.Driver) bool {
	v, ok := uc.vendorOf(p.Vendor)
	return ok && agentapi.Serves(v, d.Protocol())
}

// agentErr 把 pkg 返回的错误变成给用户看的业务错误。
func agentErr(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(response.Error); ok {
		return err
	}
	return response.BadRequest(err.Error())
}

func agentJSON(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(raw)
}

func agentParse[T any](raw string) T {
	var out T
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	return out
}

// agentTail 密钥脱敏后留下的后四位。
func agentTail(secret string) string {
	r := []rune(secret)
	if len(r) <= 4 {
		return string(r)
	}
	return string(r[len(r)-4:])
}

func (uc *AgentUsecase) encrypt(plain string) string {
	enc, err := uc.box.Encrypt(plain)
	if err != nil {
		return ""
	}
	return enc
}

func (uc *AgentUsecase) decrypt(cipher string) string {
	plain, err := uc.box.Decrypt(cipher)
	if err != nil {
		return ""
	}
	return plain
}

// runLimits 进程数上限和空闲时长（默认值在 common 的配置默认表里）。
func (uc *AgentUsecase) runLimits(ctx context.Context) (int, time.Duration) {
	read := func(key common.ConfigKey) int {
		raw, _ := uc.config.Get(ctx, key)
		n, _ := strconv.Atoi(strings.TrimSpace(raw))
		return max(n, 0)
	}
	return read(common.ConfigAgentMaxProcesses), time.Duration(read(common.ConfigAgentIdleMinutes)) * time.Minute
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// —— 名称缓存 ——

// agentNames 会话、工作区上要带的运行时、供应商、工作区的名称。这三张表很小、只经这个用例改，
// 整份留在内存里，改了就作废，下次用时重读：推送会话状态时不用每次查库。
type agentNames struct {
	runtimes   map[string]*gen.AgentRuntime
	providers  map[string]*gen.AgentProvider
	workspaces map[string]*gen.AgentWorkspace
	modes      map[string]map[string]string // 运行时 id → 权限模式 id → Agent 给的名称
}

type agentNameCache struct {
	mu    sync.Mutex
	names *agentNames
}

func (uc *AgentUsecase) loadNames(ctx context.Context) *agentNames {
	uc.names.mu.Lock()
	defer uc.names.mu.Unlock()
	if uc.names.names != nil {
		return uc.names.names
	}
	names := &agentNames{
		runtimes:   map[string]*gen.AgentRuntime{},
		providers:  map[string]*gen.AgentProvider{},
		workspaces: map[string]*gen.AgentWorkspace{},
		modes:      map[string]map[string]string{},
	}
	rts, err1 := uc.repo.ListAgentRuntimes(ctx)
	ps, err2 := uc.repo.ListAgentProviders(ctx)
	ws, err3 := uc.repo.ListAgentWorkspaces(ctx, "")
	for _, rt := range rts {
		names.runtimes[rt.ID] = rt
		modes := map[string]string{}
		for _, m := range agentParse[agentRuntimeCatalog](rt.Catalog).Modes {
			modes[m.ID] = m.Name
		}
		names.modes[rt.ID] = modes
	}
	for _, p := range ps {
		names.providers[p.ID] = p
	}
	for _, w := range ws {
		names.workspaces[w.ID] = w
	}
	// 读失败了这次照样用，但不缓存，下次再读。
	if err1 == nil && err2 == nil && err3 == nil {
		uc.names.names = names
	}
	return names
}

// configChanged Agent 配置（运行时、供应商、工作区、MCP、技能）有改动之后调用：名称缓存作废，
// 并告诉开着的页面配置变了（Code 页据此重读选项和列表，不用刷新整个网页）。
func (uc *AgentUsecase) configChanged() {
	uc.names.mu.Lock()
	uc.names.names = nil
	uc.names.mu.Unlock()
	uc.hub.ToAll([]byte(`{"t":"config"}`))
}

func (n *agentNames) runtime(id string) (name, kind string) {
	if rt := n.runtimes[id]; rt != nil {
		return rt.Name, string(rt.Kind)
	}
	return "", "custom"
}

// provider 供应商的名称、种类（api / system）和供应商类型；没有供应商（服务器上的登录）时名称是检查时 Agent 报的账号。
func (n *agentNames) provider(runtimeID, providerID string) (name, kind, vendor string) {
	if p := n.providers[providerID]; p != nil {
		return p.Name, "api", p.Vendor
	}
	if rt := n.runtimes[runtimeID]; rt != nil && providerID == "" {
		return rt.AuthLabel, "system", ""
	}
	return "", "", ""
}

func (n *agentNames) mode(runtimeID, mode string) string {
	if name := n.modes[runtimeID][mode]; name != "" {
		return name
	}
	return mode
}

// —— 选择器和概览 ——

func agentChoicesPB(in []agentrun.Choice) []*v1.AgentChoice {
	out := make([]*v1.AgentChoice, 0, len(in))
	for _, c := range in {
		out = append(out, &v1.AgentChoice{Id: c.ID, Name: c.Name, Description: c.Description})
	}
	return out
}

// agentModelsPB 服务器上的登录能用的模型。
func agentModelsPB(in []*gen.AgentRuntimeModel) []*v1.AgentModelOption {
	out := make([]*v1.AgentModelOption, 0, len(in))
	for _, m := range in {
		opt := &v1.AgentModelOption{
			Id: m.Model, Name: m.Name, DefaultEffort: m.DefaultEffort,
			ContextWindow: m.ContextWindow, MaxOutput: m.MaxOutput,
		}
		for _, e := range m.Efforts {
			opt.Efforts = append(opt.Efforts, &v1.AgentChoice{Id: e.ID, Name: e.Name, Description: e.Description})
		}
		out = append(out, opt)
	}
	return out
}

// agentModelsByRuntime 这些运行时的模型，按运行时分好。
func (uc *AgentUsecase) agentModelsByRuntime(ctx context.Context, runtimeIDs ...string) (map[string][]*gen.AgentRuntimeModel, error) {
	rows, err := uc.repo.ListAgentRuntimeModels(ctx, runtimeIDs...)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := make(map[string][]*gen.AgentRuntimeModel, len(runtimeIDs))
	for _, m := range rows {
		out[m.RuntimeID] = append(out[m.RuntimeID], m)
	}
	return out, nil
}

func agentCapsPB(c agentrun.Caps) *v1.AgentCapabilities {
	return &v1.AgentCapabilities{
		Resume: c.Resume, Fork: c.Fork, AdditionalDirs: c.AdditionalDirs,
		Image: c.Image, EmbeddedContext: c.EmbeddedContext, McpHttp: c.McpHTTP, McpSse: c.McpSSE, Close: c.Close,
		Subagents: c.Subagents, Steer: c.Steer,
	}
}

// GetOptions 每个 Agent 下能选的供应商、模型、推理强度和权限模式：
//   - 服务器上的登录：用户自己在服务器上登录命令行，最近一次检查时登录着才有；模型和推理强度是检查时 Agent 给的；
//   - 第三方接口：供应商类型给这一系协议的，模型和推理等级是模型目录里声明的。
func (uc *AgentUsecase) GetOptions(ctx context.Context) (*v1.GetAgentOptionsResponse, error) {
	if _, err := agentUserID(ctx); err != nil {
		return nil, err
	}
	runtimes, err := uc.repo.ListAgentRuntimes(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	providers, err := uc.repo.ListAgentProviders(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	ids := make([]string, 0, len(runtimes))
	for _, rt := range runtimes {
		ids = append(ids, rt.ID)
	}
	models, err := uc.agentModelsByRuntime(ctx, ids...)
	if err != nil {
		return nil, err
	}
	resp := &v1.GetAgentOptionsResponse{}
	for _, rt := range runtimes {
		catalog := agentParse[agentRuntimeCatalog](rt.Catalog)
		item := &v1.AgentRuntimeOptions{
			RuntimeId:   rt.ID,
			RuntimeName: rt.Name,
			Kind:        string(rt.Kind),
			Ready:       rt.Ready,
			Modes:       agentChoicesPB(catalog.Modes),
			DefaultMode: catalog.DefaultMode,
			Caps:        agentCapsPB(agentParse[agentrun.Caps](rt.Caps)),
		}
		if rt.LoggedIn {
			item.Providers = append(item.Providers, &v1.AgentProviderOptions{
				Name:         rt.AuthLabel,
				Kind:         "system",
				Models:       agentModelsPB(models[rt.ID]),
				DefaultModel: catalog.DefaultModel,
			})
		}
		d := uc.driverOf(string(rt.Kind))
		for _, p := range providers {
			if uc.serves(p, d) {
				item.Providers = append(item.Providers, agentProviderOptions(p))
			}
		}
		resp.Runtimes = append(resp.Runtimes, item)
	}
	return resp, nil
}

// agentProviderOptions 一个第三方接口下能选的模型：模型目录里声明的，推理强度是声明过的等级。
func agentProviderOptions(p *gen.AgentProvider) *v1.AgentProviderOptions {
	out := &v1.AgentProviderOptions{ProviderId: p.ID, Name: p.Name, Kind: "api", Status: p.Status, Vendor: p.Vendor, DefaultModel: p.MainModel}
	for _, m := range p.Models {
		levels, def := agentLevels(p, m.ID)
		opt := &v1.AgentModelOption{Id: m.ID, Name: m.ID, DefaultEffort: def}
		for _, lv := range levels {
			opt.Efforts = append(opt.Efforts, &v1.AgentChoice{Id: lv, Name: lv})
		}
		out.Models = append(out.Models, opt)
	}
	return out
}

// GetOverview 管理页各标签上的数量：会话、工作区、MCP、技能是自己的（all 时是所有人的），供应商、运行时、通知是共用的。
func (uc *AgentUsecase) GetOverview(ctx context.Context, all bool) (*v1.GetAgentOverviewResponse, error) {
	userID, err := uc.listScope(ctx, all)
	if err != nil {
		return nil, err
	}
	c, err := uc.repo.AgentCounts(ctx, userID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	notify := uc.loadNotify(ctx)
	enabled := 0
	for _, kind := range agentNotifyKinds {
		if ch := notify.Channels[kind]; ch != nil && ch.Enabled {
			enabled++
		}
	}
	return &v1.GetAgentOverviewResponse{
		Sessions:        int32(c.Sessions),
		SessionsRunning: int32(c.SessionsRunning),
		SessionsWaiting: int32(c.SessionsWaiting),
		Workspaces:      int32(c.Workspaces),
		Providers:       int32(c.Providers),
		McpServers:      int32(c.McpServers),
		McpOk:           int32(c.McpOK),
		Skills:          int32(c.Skills),
		SkillsEnabled:   int32(c.SkillsEnabled),
		Runtimes:        int32(c.Runtimes),
		RuntimesReady:   int32(c.RuntimesReady),
		NotifyChannels:  int32(len(agentNotifyKinds)),
		NotifyEnabled:   int32(enabled),
	}, nil
}
