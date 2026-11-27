package biz

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/agentmcpserver"
	"momoko/internal/data/ent/gen/agentruntime"
	"momoko/internal/data/ent/gen/agentskill"
	"momoko/internal/data/ent/schema"
	"momoko/pkg/agentapi"
	"momoko/pkg/agentdriver"
	"momoko/pkg/agentfs"
	"momoko/pkg/agentgit"
	"momoko/pkg/agentinstall"
	"momoko/pkg/agentproc"
	"momoko/pkg/agentproxy"
	"momoko/pkg/agentrun"
	"momoko/pkg/agentsandbox"
	agentskillpkg "momoko/pkg/agentskill"
	"momoko/pkg/common"
	"momoko/pkg/mcpprobe"
	"momoko/pkg/response"
)

// ———————————————————————————— 运行时 ————————————————————————————

// builtinDriver 内置的两条运行时（id 就是驱动的 kind）：不能移除，命令由驱动决定。
func (uc *AgentUsecase) builtinDriver(id string) (agentdriver.Driver, bool) {
	for _, d := range uc.drivers {
		if d.Kind() == id {
			return d, true
		}
	}
	return nil, false
}

// release 这条运行时能「Momoko 下载并更新」时的发行包。
func (uc *AgentUsecase) release(id string) (agentinstall.Release, bool) {
	if d, ok := uc.builtinDriver(id); ok {
		return d.Release()
	}
	return agentinstall.Release{}, false
}

// seedRuntimes 保证内置运行时存在。检测要启动进程，只在用户点「重新检测」或切换安装方式时做。
func (uc *AgentUsecase) seedRuntimes(ctx context.Context) {
	existing, err := uc.repo.ListAgentRuntimes(ctx)
	if err != nil {
		uc.log.Warnf("list agent runtimes: %v", err)
		return
	}
	have := map[string]bool{}
	for _, rt := range existing {
		have[rt.ID] = true
	}
	titles := map[string]string{"claude": "Claude Code", "codex": "Codex"}
	for i, d := range uc.drivers {
		if have[d.Kind()] {
			continue
		}
		seed := &gen.AgentRuntime{ID: d.Kind(), Name: titles[d.Kind()], Kind: agentruntime.Kind(d.Kind()), Sort: i}
		if err := uc.repo.SaveAgentRuntime(ctx, seed); err != nil {
			uc.log.Warnf("seed agent runtime %s: %v", seed.ID, err)
		}
	}
}

func (uc *AgentUsecase) managedDir(id string) string {
	return filepath.Join(uc.root, "runtimes", id)
}

// runtimeLaunch 这个运行时现在该怎么启动：自己添加的用填的命令；内置的要么是 Momoko 下载的那一份，
// 要么是服务器 PATH 里已有的适配器。
func (uc *AgentUsecase) runtimeLaunch(rt *gen.AgentRuntime) (agentdriver.Launch, error) {
	d, builtin := uc.builtinDriver(rt.ID)
	if !builtin {
		cmd, args := agentproc.SplitCommand(rt.Command)
		if cmd == "" {
			return agentdriver.Launch{}, response.BadRequest("没有配置启动命令")
		}
		return agentdriver.Launch{Command: cmd, Args: args}, nil
	}
	if _, ok := d.Release(); ok && rt.Managed {
		launch, ok := agentinstall.Installed(uc.managedDir(rt.ID))
		if !ok {
			return agentdriver.Launch{}, response.BadRequest(rt.Name + " 还没下载好")
		}
		return agentdriver.Launch{Command: launch.Command}, nil
	}
	path, ok := agentproc.Lookup(d.Adapter())
	if !ok {
		if _, can := d.Release(); can {
			return agentdriver.Launch{}, response.BadRequest("服务器上没有找到 " + d.Adapter() + "；装上它，或者把安装方式改成「Momoko 下载并更新」")
		}
		return agentdriver.Launch{}, response.BadRequest("服务器上没有找到 " + d.Adapter())
	}
	return agentdriver.Launch{Command: path}, nil
}

// detectRuntime 检测一个运行时并把结果存下来。检测不通过不算错误：原因写在 runtime.Error 里。
func (uc *AgentUsecase) detectRuntime(ctx context.Context, rt *gen.AgentRuntime) (*gen.AgentRuntime, error) {
	bg := context.WithoutCancel(ctx)
	now := time.Now()
	rt.DetectedAt = &now
	launch, err := uc.runtimeLaunch(rt)
	if err == nil {
		var sb *agentsandbox.Config
		if sb, err = uc.runtimeSandbox(rt); err == nil {
			launch, launch.Env = sandboxed(uc.driverOf(string(rt.Kind)), launch, launch.Env, sb)
		}
	}
	var models []*gen.AgentRuntimeModel
	synced := false
	if err == nil {
		tmp := filepath.Join(uc.root, "tmp", "probe-"+uuid.NewString()[:8])
		var result *agentrun.ProbeResult
		result, err = agentrun.Probe(bg, launch, tmp)
		agentfs.RemoveAll(tmp)
		if err == nil {
			rt.Ready, rt.Error = true, ""
			rt.AgentName, rt.AgentVersion = result.AgentName, result.AgentVersion
			// 是哪一系看 Agent 自己报的名字，不看它是哪一条：自己添加的 codex-acp 也能用 OpenAI 兼容的供应商。
			rt.Kind = agentruntime.KindCustom
			for _, d := range uc.drivers {
				if d.Detect(result.AgentName) {
					rt.Kind = agentruntime.Kind(d.Kind())
				}
			}
			rt.Caps = agentJSON(result.Caps)
			// 能开会话就是登录着；没登录时 Agent 可能报了登录状态，也可能没报。
			rt.LoggedIn = !result.NeedAuth
			rt.AuthLabel, rt.AuthDetail = "", ""
			if result.AuthOK {
				rt.AuthLabel, rt.AuthDetail = result.Auth.Label, result.Auth.Description()
			}
			// 检查登录就是一次同步：权限模式和默认模型记在运行时上，模型写进模型表
			if !result.NeedAuth {
				rt.Catalog = agentJSON(agentRuntimeCatalog{
					Modes: result.Catalog.Modes, DefaultMode: result.Catalog.DefaultMode, DefaultModel: result.Catalog.DefaultModel,
				})
				models, synced = runtimeModels(result.Catalog.Models), true
			}
		}
	}
	if err != nil {
		rt.Ready, rt.LoggedIn, rt.Error = false, false, err.Error()
	}
	if err := uc.repo.SaveAgentRuntime(bg, rt); err != nil {
		return nil, ErrSystem(err)
	}
	if synced {
		if err := uc.repo.SyncAgentRuntimeModels(bg, rt.ID, models); err != nil {
			return nil, ErrSystem(err)
		}
	}
	uc.configChanged()
	return rt, nil
}

// runtimeModels 检查时 Agent 给的模型，按它列出的顺序。
func runtimeModels(in []agentrun.CatalogModel) []*gen.AgentRuntimeModel {
	out := make([]*gen.AgentRuntimeModel, 0, len(in))
	for i, m := range in {
		efforts := make([]schema.AgentChoice, 0, len(m.Efforts))
		for _, e := range m.Efforts {
			efforts = append(efforts, schema.AgentChoice{ID: e.ID, Name: e.Name, Description: e.Description})
		}
		out = append(out, &gen.AgentRuntimeModel{Model: m.ID, Name: m.Name, Efforts: efforts, DefaultEffort: m.DefaultEffort, Sort: i})
	}
	return out
}

// activeByRuntime 每个运行时现在有多少个会话的进程在跑。
func (uc *AgentUsecase) activeByRuntime() map[string]int {
	out := map[string]int{}
	for _, l := range uc.pool.Snapshot() {
		out[l.runtimeID]++
	}
	return out
}

// runtimeOut 一个运行时连同它的模型。
func (uc *AgentUsecase) runtimeOut(ctx context.Context, rt *gen.AgentRuntime, active map[string]int) (*v1.AgentRuntime, error) {
	models, err := uc.agentModelsByRuntime(ctx, rt.ID)
	if err != nil {
		return nil, err
	}
	return uc.runtimePB(rt, models[rt.ID], active), nil
}

func (uc *AgentUsecase) runtimePB(rt *gen.AgentRuntime, models []*gen.AgentRuntimeModel, active map[string]int) *v1.AgentRuntime {
	_, builtin := uc.builtinDriver(rt.ID)
	_, canInstall := uc.release(rt.ID)
	out := &v1.AgentRuntime{
		Id:             rt.ID,
		Name:           rt.Name,
		Kind:           string(rt.Kind),
		Builtin:        builtin,
		CanInstall:     canInstall,
		Managed:        rt.Managed && canInstall,
		Ready:          rt.Ready,
		AgentName:      rt.AgentName,
		AgentVersion:   rt.AgentVersion,
		Error:          rt.Error,
		ActiveSessions: int32(active[rt.ID]),
		Caps:           agentCapsPB(agentParse[agentrun.Caps](rt.Caps)),
		AuthLabel:      rt.AuthLabel,
		AuthDetail:     rt.AuthDetail,
		LoggedIn:       rt.LoggedIn,
		Models:         agentModelsPB(models),
		Proxy:          uc.proxyPB(rt.Proxy, false),
	}
	_, _, out.CanLogin = uc.loginLaunch(rt)
	if launch, err := uc.runtimeLaunch(rt); err == nil {
		out.Command, out.Args = launch.Command, launch.Args
	}
	if rt.DetectedAt != nil {
		out.DetectedAt = timestamppb.New(*rt.DetectedAt)
	}
	uc.installMu.Lock()
	out.Installing = uc.installing[rt.ID]
	uc.installMu.Unlock()
	return out
}

func (uc *AgentUsecase) runSettings(ctx context.Context) *v1.AgentRunSettings {
	limit, idle := uc.runLimits(ctx)
	return &v1.AgentRunSettings{MaxProcesses: int32(limit), IdleMinutes: int32(idle / time.Minute)}
}

func (uc *AgentUsecase) ListRuntimes(ctx context.Context) (*v1.ListAgentRuntimesResponse, error) {
	rts, err := uc.repo.ListAgentRuntimes(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	ids := make([]string, 0, len(rts))
	for _, rt := range rts {
		ids = append(ids, rt.ID)
	}
	models, err := uc.agentModelsByRuntime(ctx, ids...)
	if err != nil {
		return nil, err
	}
	active := uc.activeByRuntime()
	resp := &v1.ListAgentRuntimesResponse{Settings: uc.runSettings(ctx), Sandbox: uc.sandboxPB()}
	for _, rt := range rts {
		resp.Runtimes = append(resp.Runtimes, uc.runtimePB(rt, models[rt.ID], active))
	}
	return resp, nil
}

func (uc *AgentUsecase) getRuntime(ctx context.Context, id string) (*gen.AgentRuntime, error) {
	rt, err := uc.repo.GetAgentRuntime(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrAgentRuntimeNotFound
		}
		return nil, ErrSystem(err)
	}
	return rt, nil
}

func (uc *AgentUsecase) DetectRuntime(ctx context.Context, id string) (*v1.AgentRuntime, error) {
	rt, err := uc.getRuntime(ctx, id)
	if err != nil {
		return nil, err
	}
	if rt, err = uc.detectRuntime(ctx, rt); err != nil {
		return nil, err
	}
	return uc.runtimeOut(ctx, rt, uc.activeByRuntime())
}

// install 下载最新的适配器；同一个运行时同时只装一次，有会话在用时不换。
func (uc *AgentUsecase) install(ctx context.Context, rt *gen.AgentRuntime, rel agentinstall.Release) error {
	uc.installMu.Lock()
	if uc.installing[rt.ID] {
		uc.installMu.Unlock()
		return response.BadRequest(rt.Name + " 正在下载，稍等一下")
	}
	uc.installing[rt.ID] = true
	uc.installMu.Unlock()
	defer func() {
		uc.installMu.Lock()
		delete(uc.installing, rt.ID)
		uc.installMu.Unlock()
	}()
	launch, err := agentinstall.Install(context.WithoutCancel(ctx), rel, uc.managedDir(rt.ID), nil)
	if err != nil {
		return agentErr(err)
	}
	rt.Version = launch.Version
	return nil
}

// SetRuntimeMode 切换安装方式。切到「Momoko 下载并更新」时没装过就先下载（可能要几分钟，不受请求超时影响）。
func (uc *AgentUsecase) SetRuntimeMode(ctx context.Context, id string, managed bool) (*v1.AgentRuntime, error) {
	rt, err := uc.getRuntime(ctx, id)
	if err != nil {
		return nil, err
	}
	rel, ok := uc.release(rt.ID)
	if !ok {
		return nil, response.BadRequest(rt.Name + " 不能由 Momoko 下载，在服务器上装好后点「重新检测」")
	}
	if managed {
		if _, ok := agentinstall.Installed(uc.managedDir(rt.ID)); !ok {
			if err := uc.install(ctx, rt, rel); err != nil {
				return nil, err
			}
		}
	}
	rt.Managed = managed
	if rt, err = uc.detectRuntime(ctx, rt); err != nil {
		return nil, err
	}
	return uc.runtimeOut(ctx, rt, uc.activeByRuntime())
}

// UpdateRuntime 检查更新（只有能由 Momoko 下载的才有）。由 Momoko 下载的直接更新到最新版本；
// 用服务器上已有命令的只告诉有没有新版本。
func (uc *AgentUsecase) UpdateRuntime(ctx context.Context, id string) (*v1.UpdateAgentRuntimeResponse, error) {
	rt, err := uc.getRuntime(ctx, id)
	if err != nil {
		return nil, err
	}
	rel, ok := uc.release(rt.ID)
	if !ok {
		return nil, response.BadRequest(rt.Name + " 由你在服务器上更新")
	}
	latest, err := agentinstall.Latest(context.WithoutCancel(ctx), rel)
	if err != nil {
		return nil, agentErr(err)
	}
	resp := &v1.UpdateAgentRuntimeResponse{Latest: latest, Current: rt.AgentVersion}
	if rt.Managed {
		resp.Current = ""
		if launch, ok := agentinstall.Installed(uc.managedDir(rt.ID)); ok {
			resp.Current = launch.Version
		}
		if resp.Current == "" || agentinstall.Newer(latest, resp.Current) {
			if uc.activeByRuntime()[rt.ID] > 0 {
				return nil, response.BadRequest("有会话正在使用 " + rt.Name + "，等它们空闲后再更新")
			}
			if err := uc.install(ctx, rt, rel); err != nil {
				return nil, err
			}
			resp.Updated = true
			if rt, err = uc.detectRuntime(ctx, rt); err != nil {
				return nil, err
			}
		}
	}
	if resp.Runtime, err = uc.runtimeOut(ctx, rt, uc.activeByRuntime()); err != nil {
		return nil, err
	}
	return resp, nil
}

// CreateRuntime 添加其它 ACP Agent：名称加一条启动命令，添加后立刻检测一次。
func (uc *AgentUsecase) CreateRuntime(ctx context.Context, name, command string) (*v1.AgentRuntime, error) {
	name, command = strings.TrimSpace(name), strings.TrimSpace(command)
	if name == "" {
		return nil, response.BadRequest("先填一个名称")
	}
	if command == "" {
		return nil, response.BadRequest("先填启动命令")
	}
	count, err := uc.repo.CountAgentRuntimes(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	rt := &gen.AgentRuntime{ID: "rt_" + uuid.NewString(), Name: name, Kind: agentruntime.KindCustom, Command: command, Sort: count}
	if rt, err = uc.detectRuntime(ctx, rt); err != nil {
		return nil, err
	}
	return uc.runtimeOut(ctx, rt, nil)
}

func (uc *AgentUsecase) DeleteRuntime(ctx context.Context, id string) error {
	rt, err := uc.getRuntime(ctx, id)
	if err != nil {
		return err
	}
	if _, builtin := uc.builtinDriver(rt.ID); builtin {
		return response.BadRequest("内置的 Agent 不能移除")
	}
	if uc.activeByRuntime()[id] > 0 {
		return response.BadRequest("有会话正在使用 " + rt.Name + "，等它们结束后再移除")
	}
	if err := uc.repo.DeleteAgentRuntime(ctx, id); err != nil {
		return ErrSystem(err)
	}
	uc.configChanged()
	return nil
}

func (uc *AgentUsecase) UpdateRunSettings(ctx context.Context, s *v1.AgentRunSettings) (*v1.AgentRunSettings, error) {
	if s.GetMaxProcesses() < 1 {
		return nil, response.BadRequest("同时运行的进程上限至少是 1")
	}
	if s.GetIdleMinutes() < 0 {
		return nil, response.BadRequest("空闲时长不能是负数")
	}
	err := uc.config.BatchUpdate(ctx, map[common.ConfigKey]string{
		common.ConfigAgentMaxProcesses: strconv.Itoa(int(s.MaxProcesses)),
		common.ConfigAgentIdleMinutes:  strconv.Itoa(int(s.IdleMinutes)),
	})
	if err != nil {
		return nil, ErrSystem(err)
	}
	uc.pool.SetLimits(int(s.MaxProcesses), time.Duration(s.IdleMinutes)*time.Minute)
	return uc.runSettings(ctx), nil
}

// ———————————————————————————— 模型供应商 ————————————————————————————

// agentLevels 模型目录里某个模型声明的推理等级，和默认用哪个：有 medium 用 medium，没有就 high，再没有用第一个。
func agentLevels(p *gen.AgentProvider, model string) (levels []string, def string) {
	for _, m := range p.Models {
		if m.ID != model {
			continue
		}
		for _, lv := range m.Levels {
			levels = append(levels, lv.Name)
		}
	}
	for _, want := range []string{"medium", "high"} {
		for _, lv := range levels {
			if strings.EqualFold(lv, want) {
				return levels, lv
			}
		}
	}
	if len(levels) > 0 {
		def = levels[0]
	}
	return levels, def
}

// agentLevelValue 某个推理等级对应的 API 参数值；ok 为 false 表示这个模型没声明这个等级。
func agentLevelValue(p *gen.AgentProvider, model, level string) (value string, ok bool) {
	for _, m := range p.Models {
		if m.ID != model {
			continue
		}
		for _, lv := range m.Levels {
			if lv.Name == level {
				return lv.Value, true
			}
		}
	}
	return "", false
}

// 新模型默认的最长上下文和最长输出。
const (
	agentDefaultContextWindow = 1_000_000
	agentDefaultMaxOutput     = 128_000
)

func (uc *AgentUsecase) providerPB(p *gen.AgentProvider) *v1.AgentProvider {
	out := &v1.AgentProvider{
		Id:            p.ID,
		Vendor:        p.Vendor,
		Name:          p.Name,
		Status:        p.Status,
		StatusText:    p.StatusText,
		BaseUrl:       p.BaseURL,
		ApiKeyTail:    agentTail(uc.decrypt(p.APIKey)),
		MainModel:     p.MainModel,
		FastModel:     p.FastModel,
		TestLatencyMs: p.TestLatencyMs,
		UserId:        p.UserID,
		Share:         p.Share,
		ShareUserIds:  p.ShareUsers,
		Proxy:         uc.proxyPB(p.Proxy, false),
	}
	for _, m := range p.Models {
		pm := &v1.AgentModel{Id: m.ID, ContextWindow: m.ContextWindow, MaxOutput: m.MaxOutput}
		for _, lv := range m.Levels {
			pm.Levels = append(pm.Levels, &v1.AgentModelLevel{Name: lv.Name, Value: lv.Value})
		}
		out.Models = append(out.Models, pm)
	}
	for _, e := range p.Env {
		out.Env = append(out.Env, &v1.AgentKV{Key: e.Key, Value: e.Value})
	}
	if p.TestedAt != nil {
		out.TestedAt = timestamppb.New(*p.TestedAt)
	}
	return out
}

// vendorPB 一种供应商类型：给哪些协议、地址固不固定、能不能查余额。
func vendorPB(v agentapi.Vendor) *v1.AgentVendor {
	_, balance := v.(agentapi.Balancer)
	return &v1.AgentVendor{Id: v.ID(), Protocols: v.Protocols(), BaseUrl: v.BaseURL(), Balance: balance}
}

// providerViewPB 改不了的那一份只给看名称、类型、地址和模型：密钥后几位、环境变量（可能也是密钥）、共享给谁都不给。
// 被共享到的人看共享的、管理员看别人自己的，都是这样。
func (uc *AgentUsecase) providerViewPB(p *gen.AgentProvider) *v1.AgentProvider {
	out := uc.providerPB(p)
	out.ApiKeyTail, out.Env, out.ShareUserIds = "", nil, nil
	out.Proxy = uc.proxyPB(p.Proxy, true)
	return out
}

// sharedProvidersFor 共享给 userID 的第三方接口。
func (uc *AgentUsecase) sharedProvidersFor(ctx context.Context, userID string) ([]*gen.AgentProvider, error) {
	rows, err := uc.repo.ListAgentProviders(ctx, agentSharedOwner)
	if err != nil {
		return nil, err
	}
	out := rows[:0]
	for _, p := range rows {
		if sharedTo(p.UserID, p.Share, p.ShareUsers, userID) {
			out = append(out, p)
		}
	}
	return out, nil
}

// usableProviders userID 用得上的第三方接口：自己的，后面跟着共享给他的。
func (uc *AgentUsecase) usableProviders(ctx context.Context, userID string) ([]*gen.AgentProvider, error) {
	own, err := uc.repo.ListAgentProviders(ctx, userID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	shared, err := uc.sharedProvidersFor(ctx, userID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return append(own, shared...), nil
}

// ListProviders 自己的第三方接口，后面跟着共享给自己的（只能用）；all 时（Agent 管理页）是所有人的和共享的，
// 连同它们的主人和共享给的人。另外带上能添加的供应商类型。
func (uc *AgentUsecase) ListProviders(ctx context.Context, all bool) (*v1.ListAgentProvidersResponse, error) {
	userID, err := uc.listScope(ctx, all)
	if err != nil {
		return nil, err
	}
	var rows []*gen.AgentProvider
	if all {
		if rows, err = uc.repo.ListAgentProviders(ctx, ""); err != nil {
			return nil, ErrSystem(err)
		}
	} else if rows, err = uc.usableProviders(ctx, userID); err != nil {
		return nil, err
	}
	resp := &v1.ListAgentProvidersResponse{
		Providers: make([]*v1.AgentProvider, 0, len(rows)),
		Vendors:   make([]*v1.AgentVendor, 0, len(uc.vendors)),
	}
	resp.Sandbox = uc.sandboxPB()
	people := make([]string, 0, len(rows))
	for _, p := range rows {
		// 能改的才给全：自己的；Agent 管理页里共享的
		if (!all && p.UserID == userID) || (all && p.UserID == agentSharedOwner) {
			resp.Providers = append(resp.Providers, uc.providerPB(p))
		} else {
			resp.Providers = append(resp.Providers, uc.providerViewPB(p))
		}
		people = append(append(people, p.UserID), p.ShareUsers...)
	}
	for _, v := range uc.vendors {
		resp.Vendors = append(resp.Vendors, vendorPB(v))
	}
	if all {
		if resp.Users, err = uc.agentUsersPB(ctx, people); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

// getProvider 取一个第三方接口，不管是谁的：能不能用、能不能改由调用的地方判断。
func (uc *AgentUsecase) getProvider(ctx context.Context, id string) (*gen.AgentProvider, error) {
	p, err := uc.repo.GetAgentProvider(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrAgentProviderNotFound
		}
		return nil, ErrSystem(err)
	}
	return p, nil
}

// providerIn 取一个属于 scope 的第三方接口（scope 为空是谁的都行）；不属于时当作不存在。
func (uc *AgentUsecase) providerIn(ctx context.Context, scope, id string) (*gen.AgentProvider, error) {
	p, err := uc.getProvider(ctx, id)
	if err != nil {
		return nil, err
	}
	if !ownedBy(scope, p.UserID) {
		return nil, ErrAgentProviderNotFound
	}
	return p, nil
}

// editableProvider 取一个能改的第三方接口：自己的；共享的要管理员。别人的当作不存在。
// 测试、拉模型、查余额也走这里：要用到密钥，被共享到的人不行。
func (uc *AgentUsecase) editableProvider(ctx context.Context, id string) (*gen.AgentProvider, error) {
	userID, err := agentUserID(ctx)
	if err != nil {
		return nil, err
	}
	p, err := uc.getProvider(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.UserID == agentSharedOwner {
		if err := uc.requireAdmin(ctx); err != nil {
			return nil, err
		}
		return p, nil
	}
	if p.UserID != userID {
		return nil, ErrAgentProviderNotFound
	}
	return p, nil
}

// usableProvider 取一个 userID 用得上的第三方接口（自己的，或者共享给他的）；用不上时当作不存在。
func (uc *AgentUsecase) usableProvider(ctx context.Context, userID, id string) (*gen.AgentProvider, error) {
	p, err := uc.getProvider(ctx, id)
	if err != nil {
		return nil, err
	}
	if !usableBy(p.UserID, p.Share, p.ShareUsers, userID) {
		return nil, ErrAgentProviderNotFound
	}
	return p, nil
}

func (uc *AgentUsecase) saveProvider(ctx context.Context, p *gen.AgentProvider) error {
	if err := uc.repo.SaveAgentProvider(ctx, p); err != nil {
		return ErrSystem(err)
	}
	uc.configChanged()
	return nil
}

// agentModel 表单里的一个模型：推理等级去重，最长上下文和最长输出没填（或填了非正数）用默认值。
func agentModel(in *v1.AgentModel) schema.AgentModel {
	model := schema.AgentModel{
		ID:            strings.TrimSpace(in.GetId()),
		Levels:        []schema.AgentModelLevel{},
		ContextWindow: in.GetContextWindow(),
		MaxOutput:     in.GetMaxOutput(),
	}
	if model.ContextWindow <= 0 {
		model.ContextWindow = agentDefaultContextWindow
	}
	if model.MaxOutput <= 0 {
		model.MaxOutput = agentDefaultMaxOutput
	}
	seen := map[string]bool{}
	for _, lv := range in.GetLevels() {
		if name := strings.TrimSpace(lv.GetName()); name != "" && !seen[name] {
			seen[name] = true
			model.Levels = append(model.Levels, schema.AgentModelLevel{Name: name, Value: strings.TrimSpace(lv.GetValue())})
		}
	}
	return model
}

// errUnknownVendor 不认识的供应商类型。
var errUnknownVendor = response.BadRequest("不认识这种供应商类型")

// applyAPIProvider 把表单里的配置写到记录上：类型只在创建时定，地址固定的类型不记地址，API Key 留空表示保留原值。
func (uc *AgentUsecase) applyAPIProvider(p *gen.AgentProvider, in *v1.AgentProvider, creating bool) error {
	name := strings.TrimSpace(in.GetName())
	if name == "" {
		return response.BadRequest("先填一个名称")
	}
	if creating {
		p.Vendor = in.GetVendor()
	}
	v, ok := uc.vendorOf(p.Vendor)
	if !ok {
		return errUnknownVendor
	}
	baseURL := ""
	if v.BaseURL() == "" {
		baseURL = strings.TrimRight(strings.TrimSpace(in.GetBaseUrl()), "/")
		if u, err := url.Parse(baseURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return response.BadRequest("接口地址要以 http:// 或 https:// 开头")
		}
	}
	if key := strings.TrimSpace(in.GetApiKey()); key != "" {
		p.APIKey = uc.encrypt(key)
	} else if creating {
		return response.BadRequest("先填 API Key")
	}
	p.Name, p.BaseURL, p.Models = name, baseURL, nil
	seen := map[string]bool{}
	for _, m := range in.Models {
		model := agentModel(m)
		if model.ID == "" || seen[model.ID] {
			continue
		}
		seen[model.ID] = true
		p.Models = append(p.Models, model)
	}
	pick := func(want string) string {
		if want = strings.TrimSpace(want); seen[want] {
			return want
		}
		if len(p.Models) > 0 {
			return p.Models[0].ID
		}
		return ""
	}
	// 快速模型（起标题、做摘要）只有 Claude 系用
	p.MainModel, p.FastModel = pick(in.MainModel), ""
	if agentapi.Serves(v, agentapi.ProtocolAnthropic) {
		p.FastModel = p.MainModel
		if strings.TrimSpace(in.FastModel) != "" {
			p.FastModel = pick(in.FastModel)
		}
	}
	p.Env = nil
	for _, kv := range in.Env {
		if key := strings.TrimSpace(kv.GetKey()); key != "" {
			p.Env = append(p.Env, schema.AgentEnv{Key: key, Value: kv.GetValue()})
		}
	}
	proxy, err := uc.proxyFromPB(p.Proxy, in.GetProxy())
	if err != nil {
		return err
	}
	if p.Proxy, err = uc.storeProxy(proxy); err != nil {
		return err
	}
	return nil
}

// CreateProvider 添加一个自己的第三方接口；带共享范围时（Agent 管理页，要管理员）添加成共享的。
func (uc *AgentUsecase) CreateProvider(ctx context.Context, in *v1.AgentProvider) (*v1.AgentProvider, error) {
	userID, err := agentUserID(ctx)
	if err != nil {
		return nil, err
	}
	p := &gen.AgentProvider{ID: "prov_" + uuid.NewString(), UserID: userID, Status: "untested"}
	if in.GetShare() != "" {
		if err := uc.requireAdmin(ctx); err != nil {
			return nil, err
		}
		p.UserID = agentSharedOwner
		if p.Share, p.ShareUsers, err = uc.shareOf(ctx, in.GetShare(), in.GetShareUserIds()); err != nil {
			return nil, err
		}
	}
	if err := uc.applyAPIProvider(p, in, true); err != nil {
		return nil, err
	}
	if err := uc.saveProvider(ctx, p); err != nil {
		return nil, err
	}
	return uc.providerPB(p), nil
}

// UpdateProvider 改自己的第三方接口；共享的要管理员，带了共享范围就一起改。
func (uc *AgentUsecase) UpdateProvider(ctx context.Context, id string, in *v1.AgentProvider) (*v1.AgentProvider, error) {
	p, err := uc.editableProvider(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.UserID == agentSharedOwner && in.GetShare() != "" {
		if p.Share, p.ShareUsers, err = uc.shareOf(ctx, in.GetShare(), in.GetShareUserIds()); err != nil {
			return nil, err
		}
	}
	before := p.BaseURL + "\x00" + p.APIKey
	beforeProxy := uc.proxyKey(p.Proxy)
	if err := uc.applyAPIProvider(p, in, false); err != nil {
		return nil, err
	}
	proxyChanged := beforeProxy != uc.proxyKey(p.Proxy)
	// 地址、密钥或代理变了，之前的测试结果就不作数了。
	if before != p.BaseURL+"\x00"+p.APIKey || proxyChanged {
		p.Status, p.StatusText, p.TestLatencyMs, p.TestedAt = "untested", "", 0, nil
	}
	if err := uc.saveProvider(ctx, p); err != nil {
		return nil, err
	}
	// 用它的会话进程是按旧的网络起的
	if proxyChanged {
		uc.restartUsing("", p.ID)
	}
	return uc.providerPB(p), nil
}

// DeleteProvider 删除第三方接口；有 agent:admin 时也能删别人的和共享的（Agent 管理页）。
func (uc *AgentUsecase) DeleteProvider(ctx context.Context, id string) error {
	scope, err := uc.cleanScope(ctx)
	if err != nil {
		return err
	}
	if _, err := uc.providerIn(ctx, scope, id); err != nil {
		return err
	}
	for _, l := range uc.pool.Snapshot() {
		if l.providerID == id {
			return response.BadRequest("有会话正在用这个供应商，等它们空闲后再删除")
		}
	}
	if err := uc.repo.DeleteAgentProvider(ctx, id); err != nil {
		return ErrSystem(err)
	}
	uc.configChanged()
	return nil
}

// draftProvider 测试、获取可用模型用的配置：表单里的草稿，留空的地址和密钥用已保存的（要是自己能改的）；已保存的类型不能改。
// 代理：useForm 时用表单里的 form（不传是不走代理，密码留空用已保存的），否则用已保存的；连接都经它（conn.HTTP）。
func (uc *AgentUsecase) draftProvider(ctx context.Context, id, vendor, baseURL, apiKey string, form *v1.AgentProxy, useForm bool, timeout time.Duration) (agentapi.Vendor, agentapi.Conn, *gen.AgentProvider, *agentproxy.Proxy, error) {
	var stored *gen.AgentProvider
	storedProxy := ""
	if id != "" {
		p, err := uc.editableProvider(ctx, id)
		if err != nil {
			return nil, agentapi.Conn{}, nil, nil, err
		}
		stored, storedProxy = p, p.Proxy
	}
	var px *agentproxy.Proxy
	var err error
	if useForm {
		px, err = uc.proxyFromPB(storedProxy, form)
	} else {
		px, err = uc.proxyOf(storedProxy)
	}
	if err != nil {
		return nil, agentapi.Conn{}, nil, nil, err
	}
	baseURL, apiKey = strings.TrimRight(strings.TrimSpace(baseURL), "/"), strings.TrimSpace(apiKey)
	if stored != nil {
		vendor = stored.Vendor
		if baseURL == "" {
			baseURL = stored.BaseURL
		}
		if apiKey == "" {
			apiKey = uc.decrypt(stored.APIKey)
		}
	}
	v, ok := uc.vendorOf(vendor)
	if !ok {
		return nil, agentapi.Conn{}, nil, nil, errUnknownVendor
	}
	if v.BaseURL() != "" {
		baseURL = ""
	} else if baseURL == "" {
		return nil, agentapi.Conn{}, nil, nil, response.BadRequest("先填接口地址")
	}
	if apiKey == "" {
		return nil, agentapi.Conn{}, nil, nil, response.BadRequest("先填 API Key")
	}
	client, err := uc.clientVia(px, timeout)
	if err != nil {
		return nil, agentapi.Conn{}, nil, nil, err
	}
	return v, agentapi.Conn{BaseURL: baseURL, APIKey: apiKey, HTTP: client}, stored, px, nil
}

// TestProvider 用主模型发一条消息（空 = hello），供应商给的每种协议同时各发一次。
// 测的是已保存的那一份时，把结果记到记录上：都通才算正常，耗时记最慢的一种。
func (uc *AgentUsecase) TestProvider(ctx context.Context, req *v1.TestAgentProviderRequest) (*v1.TestAgentProviderResponse, error) {
	draft := req.GetDraft()
	v, conn, stored, px, err := uc.draftProvider(ctx, req.GetId(), draft.GetVendor(), draft.GetBaseUrl(), draft.GetApiKey(), draft.GetProxy(), draft != nil, 3*time.Minute)
	if err != nil {
		return nil, err
	}
	model := strings.TrimSpace(draft.GetMainModel())
	if model == "" && len(draft.GetModels()) > 0 {
		model = draft.GetModels()[0].GetId()
	}
	if model == "" && stored != nil {
		model = stored.MainModel
	}
	if model == "" {
		return nil, response.BadRequest("模型目录里还没有模型，先添加一个")
	}
	message := strings.TrimSpace(req.GetMessage())
	if message == "" {
		message = "hello"
	}
	// 页面上的请求两分钟超时，这里要先结束
	bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), 110*time.Second)
	defer cancel()
	protocols := v.Protocols()
	results := make([]*v1.AgentProviderTestResult, len(protocols))
	var wg sync.WaitGroup
	for i, protocol := range protocols {
		wg.Add(1)
		go func() {
			defer wg.Done()
			started := time.Now()
			reply, err := v.Test(bg, protocol, conn, model, message)
			r := &v1.AgentProviderTestResult{
				Protocol: protocol, Ok: err == nil, LatencyMs: time.Since(started).Milliseconds(),
				Reply: reply.Text, InputTokens: reply.InputTokens, OutputTokens: reply.OutputTokens,
			}
			if err != nil {
				r.Error = err.Error()
			}
			results[i] = r
		}()
	}
	wg.Wait()
	if stored != nil && conn.BaseURL == stored.BaseURL && conn.APIKey == uc.decrypt(stored.APIKey) && uc.proxyKey(stored.Proxy) == encodedProxy(px) {
		now := time.Now()
		stored.TestedAt, stored.TestLatencyMs = &now, 0
		stored.Status, stored.StatusText = "ok", ""
		for _, r := range results {
			stored.TestLatencyMs = max(stored.TestLatencyMs, r.LatencyMs)
			if !r.Ok && stored.Status == "ok" {
				stored.Status, stored.StatusText = "bad", r.Error
			}
		}
		_ = uc.saveProvider(context.WithoutCancel(ctx), stored)
	}
	return &v1.TestAgentProviderResponse{Model: model, Results: results}, nil
}

// FetchProviderModels 从接口拉可用模型；接口给了推理强度、最长上下文和最长输出的一并带上（推理等级的参数值就是等级名）。
func (uc *AgentUsecase) FetchProviderModels(ctx context.Context, req *v1.FetchAgentProviderModelsRequest) ([]*v1.AgentModel, error) {
	v, conn, _, _, err := uc.draftProvider(ctx, req.GetId(), req.GetVendor(), req.GetBaseUrl(), req.GetApiKey(), req.GetProxy(), true, time.Minute)
	if err != nil {
		return nil, err
	}
	bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	models, err := v.Models(bg, conn)
	if err != nil {
		return nil, agentErr(err)
	}
	out := make([]*v1.AgentModel, 0, len(models))
	for _, m := range models {
		pm := &v1.AgentModel{Id: m.ID, ContextWindow: m.ContextWindow, MaxOutput: m.MaxOutput}
		for _, lv := range m.Levels {
			pm.Levels = append(pm.Levels, &v1.AgentModelLevel{Name: lv, Value: lv})
		}
		out = append(out, pm)
	}
	return out, nil
}

// ProviderBalance 账户余额。查不到是接口那边的状况，原因放在 error 里返回，不算请求出错。
func (uc *AgentUsecase) ProviderBalance(ctx context.Context, id string) (*v1.GetAgentProviderBalanceResponse, error) {
	p, err := uc.editableProvider(ctx, id)
	if err != nil {
		return nil, err
	}
	v, _ := uc.vendorOf(p.Vendor)
	b, ok := v.(agentapi.Balancer)
	if !ok {
		return nil, response.BadRequest("这种供应商查不了余额")
	}
	px, err := uc.proxyOf(p.Proxy)
	if err != nil {
		return nil, err
	}
	client, err := uc.clientVia(px, time.Minute)
	if err != nil {
		return nil, err
	}
	bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), 40*time.Second)
	defer cancel()
	bal, err := b.Balance(bg, agentapi.Conn{BaseURL: p.BaseURL, APIKey: uc.decrypt(p.APIKey), HTTP: client})
	if err != nil {
		return &v1.GetAgentProviderBalanceResponse{Error: err.Error()}, nil
	}
	resp := &v1.GetAgentProviderBalanceResponse{Available: bal.Available}
	for _, it := range bal.Items {
		resp.Balances = append(resp.Balances, &v1.AgentBalance{Currency: it.Currency, Total: it.Total, Granted: it.Granted, ToppedUp: it.ToppedUp})
	}
	return resp, nil
}

// providerEnv 用某个供应商启动 Agent 进程时的环境变量。配置目录一直是 Agent 默认的，换供应商只是换这一组参数：
//   - 没有供应商：服务器上已有的登录，什么都不传；
//   - 第三方接口：接口地址、密钥、模型、最长上下文和输出靠环境注入（byEnv），换模型要重启进程。
func (uc *AgentUsecase) providerEnv(d agentdriver.Driver, p *gen.AgentProvider, model, effort string) (env map[string]string, byEnv bool, err error) {
	if p == nil {
		return nil, false, nil
	}
	if !uc.serves(p, d) {
		return nil, false, response.BadRequest("这个供应商的接口和这个 Agent 不兼容")
	}
	key := uc.decrypt(p.APIKey)
	if key == "" {
		return nil, false, response.BadRequest("供应商「" + p.Name + "」没有可用的 API Key")
	}
	if model == "" {
		model = p.MainModel
	}
	// 同一个供应商给不同协议的地址可以不同（DeepSeek 的 Anthropic 兼容在 /anthropic 下）；serves 已经确认认得这个类型
	v, _ := uc.vendorOf(p.Vendor)
	baseURL := v.Endpoint(d.Protocol(), agentapi.Conn{BaseURL: p.BaseURL})
	api := agentdriver.API{Name: p.Name, BaseURL: baseURL, Key: key, Model: model, FastModel: p.FastModel, Extra: map[string]string{}}
	api.Level, api.HasLevel = agentLevelValue(p, model, effort)
	for _, m := range p.Models {
		if m.ID == model {
			api.ContextWindow, api.MaxOutput = m.ContextWindow, m.MaxOutput
		}
	}
	for _, kv := range p.Env {
		api.Extra[kv.Key] = kv.Value
	}
	return d.APIEnv(api), true, nil
}

// loginModel 服务器上的登录里这个模型在模型表里的那一行；model 为空时看 Agent 默认的模型。模型表里没有时为 nil。
func (uc *AgentUsecase) loginModel(ctx context.Context, rt *gen.AgentRuntime, model string) (*gen.AgentRuntimeModel, error) {
	if model == "" {
		model = agentParse[agentRuntimeCatalog](rt.Catalog).DefaultModel
	}
	m, err := uc.repo.GetAgentRuntimeModel(ctx, rt.ID, model)
	if gen.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	return m, nil
}

// loginLimits 用户给这个模型设的最长上下文和最长输出（没设、模型表里没有为 0）。
func loginLimits(m *gen.AgentRuntimeModel) (contextWindow, maxOutput int64) {
	if m == nil {
		return 0, 0
	}
	return m.ContextWindow, m.MaxOutput
}

// SetRuntimeModels 服务器上的登录：给检查时同步进来的模型设最长上下文和最长输出（0 = 用 Agent 自己的），直接写模型表，下次起进程时交给 Agent。
func (uc *AgentUsecase) SetRuntimeModels(ctx context.Context, id string, in []*v1.AgentModelLimit) (*v1.AgentRuntime, error) {
	rt, err := uc.getRuntime(ctx, id)
	if err != nil {
		return nil, err
	}
	limits := make([]AgentModelLimit, 0, len(in))
	for _, m := range in {
		if m.GetContextWindow() < 0 || m.GetMaxOutput() < 0 {
			return nil, response.BadRequest("最长上下文和最长输出不能是负数")
		}
		limits = append(limits, AgentModelLimit{Model: m.GetModel(), ContextWindow: m.GetContextWindow(), MaxOutput: m.GetMaxOutput()})
	}
	if err := uc.repo.SetAgentRuntimeModelLimits(ctx, rt.ID, limits); err != nil {
		return nil, ErrSystem(err)
	}
	uc.configChanged()
	return uc.runtimeOut(ctx, rt, uc.activeByRuntime())
}

// markLoggedOut 会话里发现服务器上的登录失效了：记下来，选择器里就不再给「服务器上的登录」，等用户登录后再检查。
func (uc *AgentUsecase) markLoggedOut(ctx context.Context, runtimeID string) {
	rt, err := uc.repo.GetAgentRuntime(ctx, runtimeID)
	if err != nil || !rt.LoggedIn {
		return
	}
	rt.LoggedIn = false
	if uc.repo.SaveAgentRuntime(ctx, rt) == nil {
		uc.configChanged()
	}
}

// ———————————————————————————— 工作区 ————————————————————————————

// agentDocNames 项目指令文件：Codex 读 AGENTS.md，Claude Code 读 CLAUDE.md。
var agentDocNames = []string{"AGENTS.md", "CLAUDE.md"}

// agentDocMax 项目指令文件的大小上限。
const agentDocMax = 512 * 1024

func sortWorkspaces(ws []*gen.AgentWorkspace) {
	sort.Slice(ws, func(i, j int) bool { return ws[i].CreateTime.Before(ws[j].CreateTime) })
}

func agentEntriesPB(entries []agentfs.Entry) []*v1.AgentFileEntry {
	out := make([]*v1.AgentFileEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, &v1.AgentFileEntry{Name: e.Name, Path: e.Path, IsDir: e.IsDir, Size: e.Size, ModTime: timestamppb.New(e.ModTime)})
	}
	return out
}

func (uc *AgentUsecase) workspacePB(w *gen.AgentWorkspace, names *agentNames, stats map[string]AgentWorkspaceStat) *v1.AgentWorkspace {
	out := &v1.AgentWorkspace{
		Id:          w.ID,
		Name:        w.Name,
		Path:        w.Path,
		CloneStatus: w.CloneStatus,
		CloneError:  w.CloneError,
		RuntimeId:   w.RuntimeID,
		ProviderId:  w.ProviderID,
		Model:       w.Model,
		Effort:      w.Effort,
		Mode:        w.Mode,
		ModeName:    names.mode(w.RuntimeID, w.Mode),
		McpIds:      w.McpIds,
		UserId:      w.UserID,
	}
	out.RuntimeName, _ = names.runtime(w.RuntimeID)
	if w.RuntimeID != "" {
		out.ProviderName, out.ProviderKind, out.ProviderVendor = names.provider(w.RuntimeID, w.ProviderID)
	}
	if agentfs.IsDir(w.Path) {
		out.IsGit, out.Branch = agentgit.Info(w.Path)
	} else {
		out.Missing = w.CloneStatus == ""
	}
	if stat, ok := stats[w.ID]; ok {
		out.SessionCount = int32(stat.Sessions)
		if !stat.ActiveAt.IsZero() {
			out.ActiveAt = timestamppb.New(stat.ActiveAt)
		}
	}
	return out
}

// ListWorkspaces 自己的工作区；all 时（Agent 管理页）是所有人的，连同它们的主人。
func (uc *AgentUsecase) ListWorkspaces(ctx context.Context, all bool) ([]*v1.AgentWorkspace, []*v1.AgentUser, error) {
	userID, err := uc.listScope(ctx, all)
	if err != nil {
		return nil, nil, err
	}
	rows, err := uc.repo.ListAgentWorkspaces(ctx, userID)
	if err != nil {
		return nil, nil, ErrSystem(err)
	}
	ids := make([]string, 0, len(rows))
	owners := make([]string, 0, len(rows))
	for _, w := range rows {
		ids = append(ids, w.ID)
		owners = append(owners, w.UserID)
	}
	stats, err := uc.repo.AgentWorkspaceStats(ctx, ids)
	if err != nil {
		return nil, nil, ErrSystem(err)
	}
	names := uc.loadNames(ctx)
	out := make([]*v1.AgentWorkspace, 0, len(rows))
	for _, w := range rows {
		out = append(out, uc.workspacePB(w, names, stats))
	}
	if !all {
		return out, nil, nil
	}
	users, err := uc.agentUsersPB(ctx, owners)
	if err != nil {
		return nil, nil, err
	}
	return out, users, nil
}

// getWorkspace 取当前用户自己的一个工作区。
func (uc *AgentUsecase) getWorkspace(ctx context.Context, id string) (*gen.AgentWorkspace, error) {
	userID, err := agentUserID(ctx)
	if err != nil {
		return nil, err
	}
	return uc.workspaceIn(ctx, userID, id)
}

// workspaceIn 取一个属于 scope 的工作区（scope 为空是谁的都行）；不属于时当作不存在。
func (uc *AgentUsecase) workspaceIn(ctx context.Context, scope, id string) (*gen.AgentWorkspace, error) {
	w, err := uc.repo.GetAgentWorkspace(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrAgentWorkspaceNotFound
		}
		return nil, ErrSystem(err)
	}
	if !ownedBy(scope, w.UserID) {
		return nil, ErrAgentWorkspaceNotFound
	}
	return w, nil
}

func (uc *AgentUsecase) saveWorkspace(ctx context.Context, w *gen.AgentWorkspace) error {
	if err := uc.repo.SaveAgentWorkspace(ctx, w); err != nil {
		return ErrSystem(err)
	}
	uc.configChanged()
	return nil
}

// checkDefaults 校验工作区的默认模型：模型定了，Agent 和供应商也就定了，两者要对得上；供应商要是 userID 用得上的。
func (uc *AgentUsecase) checkDefaults(ctx context.Context, userID, runtimeID, providerID string) error {
	if runtimeID == "" {
		if providerID != "" {
			return response.BadRequest("默认模型缺少对应的 Agent")
		}
		return nil
	}
	rt, err := uc.getRuntime(ctx, runtimeID)
	if err != nil || providerID == "" {
		return err
	}
	p, err := uc.usableProvider(ctx, userID, providerID)
	if err != nil {
		return err
	}
	if !uc.serves(p, uc.driverOf(string(rt.Kind))) {
		return response.BadRequest("这个供应商的接口和这个 Agent 不兼容")
	}
	return nil
}

// pathFree 同一个人的工作区不能是同一个目录（不同的人可以各建一个）。
func (uc *AgentUsecase) pathFree(ctx context.Context, userID, path, exceptID string) error {
	existing, err := uc.repo.FindAgentWorkspaceByPath(ctx, userID, path)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil
		}
		return ErrSystem(err)
	}
	if existing.ID != exceptID {
		return response.BadRequest("这个目录已经是工作区「" + existing.Name + "」了")
	}
	return nil
}

// checkAgentDocs 项目指令文件只能是 AGENTS.md 或 CLAUDE.md，有大小上限。
func checkAgentDocs(docs []*v1.AgentDoc) error {
	for _, doc := range docs {
		if doc.GetName() != agentDocNames[0] && doc.GetName() != agentDocNames[1] {
			return response.BadRequest("项目指令文件只能是 AGENTS.md 或 CLAUDE.md")
		}
		if len(doc.GetContent()) > agentDocMax {
			return response.BadRequest(doc.GetName() + " 太大了")
		}
	}
	return nil
}

// writeAgentDocs 把表单里创建或改过的项目指令文件写进工作区目录。
func writeAgentDocs(dir string, docs []*v1.AgentDoc) error {
	for _, doc := range docs {
		if err := agentfs.WriteText(dir, doc.GetName(), doc.GetContent()); err != nil {
			return agentErr(err)
		}
	}
	return nil
}

func (uc *AgentUsecase) CreateWorkspace(ctx context.Context, req *v1.CreateAgentWorkspaceRequest) (*v1.AgentWorkspace, error) {
	userID, err := agentUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := uc.checkDefaults(ctx, userID, req.RuntimeId, req.ProviderId); err != nil {
		return nil, err
	}
	if err := checkAgentDocs(req.Docs); err != nil {
		return nil, err
	}
	mcpIDs, err := uc.ownMcpIDs(ctx, userID, req.McpIds)
	if err != nil {
		return nil, err
	}
	w := &gen.AgentWorkspace{
		ID: "ws_" + uuid.NewString(), UserID: userID, Name: strings.TrimSpace(req.Name),
		RuntimeID: req.RuntimeId, ProviderID: req.ProviderId, Model: req.Model, Effort: req.Effort, Mode: req.Mode, McpIds: mcpIDs,
		CreateTime: time.Now(),
	}
	if req.Source == "git" {
		gitURL := strings.TrimSpace(req.GitUrl)
		if gitURL == "" {
			return nil, response.BadRequest("先填仓库地址")
		}
		if w.Name == "" {
			w.Name = agentgit.RepoName(gitURL)
		}
		if w.Name == "" {
			return nil, response.BadRequest("先填一个名称")
		}
		path := strings.TrimSpace(req.Path)
		if path == "" {
			path = filepath.Join(uc.root, "workspaces", w.Name)
		}
		if w.Path, err = agentfs.CleanDir(path); err != nil {
			return nil, agentErr(err)
		}
		if !agentfs.Empty(w.Path) {
			return nil, response.BadRequest("克隆的目标目录已经有东西了，换一个目录")
		}
		if err := uc.pathFree(ctx, userID, w.Path, ""); err != nil {
			return nil, err
		}
		w.GitURL, w.CloneStatus = gitURL, "cloning"
		if err := uc.saveWorkspace(ctx, w); err != nil {
			return nil, err
		}
		// 保存后在后台克隆，完成之前这个工作区显示为「正在克隆」。
		go uc.cloneWorkspace(w.ID, gitURL, strings.TrimSpace(req.GitBranch), w.Path, req.Docs)
	} else {
		if w.Path, err = agentfs.CleanDir(req.Path); err != nil {
			return nil, agentErr(err)
		}
		if !agentfs.IsDir(w.Path) {
			return nil, response.BadRequest("服务器上没有这个目录")
		}
		if err := uc.pathFree(ctx, userID, w.Path, ""); err != nil {
			return nil, err
		}
		if w.Name == "" {
			w.Name = filepath.Base(w.Path)
		}
		if err := writeAgentDocs(w.Path, req.Docs); err != nil {
			return nil, err
		}
		if err := uc.saveWorkspace(ctx, w); err != nil {
			return nil, err
		}
	}
	return uc.workspacePB(w, uc.loadNames(ctx), nil), nil
}

// cloneWorkspace 后台克隆；克隆好、或者失败了，把结果写回工作区并推给页面刷新。
func (uc *AgentUsecase) cloneWorkspace(id, gitURL, branch, path string, docs []*v1.AgentDoc) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	err := agentgit.Clone(ctx, gitURL, branch, path, 0)
	if err == nil {
		err = writeAgentDocs(path, docs)
	}
	w, getErr := uc.repo.GetAgentWorkspace(ctx, id)
	if getErr != nil {
		// 克隆期间工作区被移除了：记录没了，目录留着。
		return
	}
	w.CloneStatus, w.CloneError = "", ""
	if err != nil {
		w.CloneStatus, w.CloneError = "failed", err.Error()
	}
	_ = uc.saveWorkspace(ctx, w)
}

func (uc *AgentUsecase) UpdateWorkspace(ctx context.Context, req *v1.UpdateAgentWorkspaceRequest) (*v1.AgentWorkspace, error) {
	w, err := uc.getWorkspace(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if err := uc.checkDefaults(ctx, w.UserID, req.RuntimeId, req.ProviderId); err != nil {
		return nil, err
	}
	if err := checkAgentDocs(req.Docs); err != nil {
		return nil, err
	}
	if len(req.Docs) > 0 && w.CloneStatus != "" {
		return nil, response.BadRequest("仓库还没克隆好，之后再改项目指令")
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		w.Name = name
	}
	if path := strings.TrimSpace(req.Path); path != "" && w.CloneStatus == "" {
		clean, err := agentfs.CleanDir(path)
		if err != nil {
			return nil, agentErr(err)
		}
		if clean != w.Path {
			if !agentfs.IsDir(clean) {
				return nil, response.BadRequest("服务器上没有这个目录")
			}
			if err := uc.pathFree(ctx, w.UserID, clean, w.ID); err != nil {
				return nil, err
			}
			w.Path = clean
		}
	}
	mcpIDs, err := uc.ownMcpIDs(ctx, w.UserID, req.McpIds)
	if err != nil {
		return nil, err
	}
	w.RuntimeID, w.ProviderID, w.Model, w.Effort, w.Mode, w.McpIds = req.RuntimeId, req.ProviderId, req.Model, req.Effort, req.Mode, mcpIDs
	if err := writeAgentDocs(w.Path, req.Docs); err != nil {
		return nil, err
	}
	if err := uc.saveWorkspace(ctx, w); err != nil {
		return nil, err
	}
	stats, err := uc.repo.AgentWorkspaceStats(ctx, []string{w.ID})
	if err != nil {
		return nil, ErrSystem(err)
	}
	return uc.workspacePB(w, uc.loadNames(ctx), stats), nil
}

// DeleteWorkspace 移除工作区：只删记录，目录里的文件不动；它的会话归到「无工作区」。
// 有 agent:admin 时也能移除别人的（Agent 管理页）。
func (uc *AgentUsecase) DeleteWorkspace(ctx context.Context, id string) error {
	scope, err := uc.cleanScope(ctx)
	if err != nil {
		return err
	}
	if _, err := uc.workspaceIn(ctx, scope, id); err != nil {
		return err
	}
	if err := uc.repo.DeleteAgentWorkspace(ctx, id); err != nil {
		return ErrSystem(err)
	}
	uc.configChanged()
	return nil
}

// ListDirs 「浏览」：服务器上某个目录的子目录。
func (uc *AgentUsecase) ListDirs(ctx context.Context, path string) (*v1.ListAgentDirsResponse, error) {
	current, parent, dirs := agentfs.Subdirs(path)
	resp := &v1.ListAgentDirsResponse{Path: current, Parent: parent, CloneBase: filepath.Join(uc.root, "workspaces")}
	for _, d := range dirs {
		item := &v1.AgentDir{Name: d.Name, Path: d.Path}
		item.IsGit, item.Branch = agentgit.Info(d.Path)
		resp.Dirs = append(resp.Dirs, item)
	}
	return resp, nil
}

func (uc *AgentUsecase) BrowseWorkspace(ctx context.Context, id, dir string) ([]*v1.AgentFileEntry, error) {
	w, err := uc.getWorkspace(ctx, id)
	if err != nil {
		return nil, err
	}
	entries, err := agentfs.List(w.Path, dir)
	if err != nil {
		return nil, agentErr(err)
	}
	// 和会话里的「文件」一样带上未提交改动的标记（工作区下面放着几个仓库时按各自的仓库看）
	out := agentEntriesPB(entries)
	marks := gitMarks(ctx, w.Path, dir)
	for _, e := range out {
		e.Change = marks[e.Path]
	}
	return out, nil
}

// ReadWorkspaceFile 只读地打开工作区里的一个文件（新会话页还没有会话目录，按工作区读）。
func (uc *AgentUsecase) ReadWorkspaceFile(ctx context.Context, id, path string) (*v1.ReadAgentSessionFileResponse, error) {
	w, err := uc.getWorkspace(ctx, id)
	if err != nil {
		return nil, err
	}
	text, err := agentfs.ReadText(w.Path, path, agentFileMax)
	if err != nil {
		return nil, agentErr(err)
	}
	return &v1.ReadAgentSessionFileResponse{Path: path, Content: text.Content, Binary: text.Binary, TooLarge: text.TooLarge, Size: text.Size, ModTime: text.ModTime}, nil
}

// GetDocs 读目录里的项目指令文件。目录不存在或还没填完时，两个文件都算没有。
func (uc *AgentUsecase) GetDocs(ctx context.Context, path string) ([]*v1.AgentDoc, error) {
	out := make([]*v1.AgentDoc, 0, len(agentDocNames))
	clean, cleanErr := agentfs.CleanDir(path)
	for _, name := range agentDocNames {
		doc := &v1.AgentDoc{Name: name}
		if cleanErr == nil {
			if text, err := agentfs.ReadText(clean, name, agentDocMax); err == nil && !text.Binary && !text.TooLarge {
				doc.Exists, doc.Content = true, text.Content
			}
		}
		out = append(out, doc)
	}
	return out, nil
}

// ———————————————————————————— MCP ————————————————————————————

// agentSecretKV 存库的一条键值：Value 是 secretbox 加密后的密文。
type agentSecretKV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// secrets 解密成明文的键值表，交给 Agent 或测试连接时用。
func (uc *AgentUsecase) secrets(raw string) map[string]string {
	out := map[string]string{}
	for _, kv := range agentParse[[]agentSecretKV](raw) {
		out[kv.Key] = uc.decrypt(kv.Value)
	}
	return out
}

// mergeSecrets 表单里的键值和已保存的合起来：值留空表示保留同名键原来的值，没出现的键就是删掉了。
func (uc *AgentUsecase) mergeSecrets(stored string, incoming []*v1.AgentSecretKV) string {
	old := map[string]string{}
	for _, kv := range agentParse[[]agentSecretKV](stored) {
		old[kv.Key] = kv.Value
	}
	out := make([]agentSecretKV, 0, len(incoming))
	seen := map[string]bool{}
	for _, kv := range incoming {
		key := strings.TrimSpace(kv.GetKey())
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		value := old[key]
		if kv.GetValue() != "" {
			value = uc.encrypt(kv.GetValue())
		}
		out = append(out, agentSecretKV{Key: key, Value: value})
	}
	return agentJSON(out)
}

func (uc *AgentUsecase) secretKVsPB(raw string) []*v1.AgentSecretKV {
	var out []*v1.AgentSecretKV
	for _, kv := range agentParse[[]agentSecretKV](raw) {
		out = append(out, &v1.AgentSecretKV{Key: kv.Key, Tail: agentTail(uc.decrypt(kv.Value))})
	}
	return out
}

func (uc *AgentUsecase) mcpPB(m *gen.AgentMcpServer) *v1.AgentMcpServer {
	out := &v1.AgentMcpServer{
		Id: m.ID, Name: m.Name, Description: m.Description, Transport: string(m.Transport),
		Command: m.Command, Args: m.Args, Url: m.URL, DefaultOn: m.DefaultOn,
		Status: m.Status, StatusText: m.StatusText, ToolCount: int32(m.ToolCount),
		Env: uc.secretKVsPB(m.Env), Headers: uc.secretKVsPB(m.Headers),
		UserId: m.UserID, Share: m.Share, ShareUserIds: m.ShareUsers,
	}
	if m.TestedAt != nil {
		out.TestedAt = timestamppb.New(*m.TestedAt)
	}
	return out
}

// sharedMcpFor 共享给 userID 的 MCP 服务器。
func (uc *AgentUsecase) sharedMcpFor(ctx context.Context, userID string) ([]*gen.AgentMcpServer, error) {
	rows, err := uc.repo.ListAgentMcpServers(ctx, agentSharedOwner)
	if err != nil {
		return nil, err
	}
	out := rows[:0]
	for _, m := range rows {
		if sharedTo(m.UserID, m.Share, m.ShareUsers, userID) {
			out = append(out, m)
		}
	}
	return out, nil
}

// ListMcpServers 自己的 MCP 服务器，后面跟着共享给自己的；all 时（Agent 管理页）是所有人的和共享的，
// 连同它们的主人和共享给的人。
func (uc *AgentUsecase) ListMcpServers(ctx context.Context, all bool) ([]*v1.AgentMcpServer, []*v1.AgentUser, error) {
	userID, err := uc.listScope(ctx, all)
	if err != nil {
		return nil, nil, err
	}
	rows, err := uc.repo.ListAgentMcpServers(ctx, userID)
	if err != nil {
		return nil, nil, ErrSystem(err)
	}
	if !all {
		shared, err := uc.sharedMcpFor(ctx, userID)
		if err != nil {
			return nil, nil, ErrSystem(err)
		}
		rows = append(rows, shared...)
	}
	out := make([]*v1.AgentMcpServer, 0, len(rows))
	people := make([]string, 0, len(rows))
	for _, m := range rows {
		out = append(out, uc.mcpPB(m))
		people = append(append(people, m.UserID), m.ShareUsers...)
	}
	if !all {
		return out, nil, nil
	}
	users, err := uc.agentUsersPB(ctx, people)
	if err != nil {
		return nil, nil, err
	}
	return out, users, nil
}

// editableMcp 取一个能改的 MCP 服务器：自己的；共享的要管理员。别人的当作不存在。
func (uc *AgentUsecase) editableMcp(ctx context.Context, id string) (*gen.AgentMcpServer, error) {
	userID, err := agentUserID(ctx)
	if err != nil {
		return nil, err
	}
	m, err := uc.mcpIn(ctx, "", id)
	if err != nil {
		return nil, err
	}
	if m.UserID == agentSharedOwner {
		if err := uc.requireAdmin(ctx); err != nil {
			return nil, err
		}
		return m, nil
	}
	if m.UserID != userID {
		return nil, ErrAgentMcpNotFound
	}
	return m, nil
}

// mcpIn 取一个属于 scope 的 MCP 服务器（scope 为空是谁的都行）；不属于时当作不存在。
func (uc *AgentUsecase) mcpIn(ctx context.Context, scope, id string) (*gen.AgentMcpServer, error) {
	m, err := uc.repo.GetAgentMcpServer(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrAgentMcpNotFound
		}
		return nil, ErrSystem(err)
	}
	if !ownedBy(scope, m.UserID) {
		return nil, ErrAgentMcpNotFound
	}
	return m, nil
}

// ownMcpIDs 只留下 userID 用得上的 MCP 服务器 id（会话、工作区只能开自己的和共享给自己的），顺序不变。
func (uc *AgentUsecase) ownMcpIDs(ctx context.Context, userID string, ids []string) ([]string, error) {
	if len(ids) == 0 {
		return ids, nil
	}
	rows, err := uc.repo.ListAgentMcpServersByIDs(ctx, ids)
	if err != nil {
		return nil, ErrSystem(err)
	}
	owned := make(map[string]bool, len(rows))
	for _, m := range rows {
		if usableBy(m.UserID, m.Share, m.ShareUsers, userID) {
			owned[m.ID] = true
		}
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if owned[id] {
			out = append(out, id)
		}
	}
	return out, nil
}

// applyMcp 把表单内容写到记录上。连接方式变了的话，之前的测试结果作废。
func (uc *AgentUsecase) applyMcp(ctx context.Context, m *gen.AgentMcpServer, in *v1.AgentMcpServer) error {
	name := strings.TrimSpace(in.GetName())
	if name == "" {
		return response.BadRequest("先填一个名称")
	}
	if strings.ContainsAny(name, " \t/\\") {
		return response.BadRequest("名称里不能有空格和斜杠")
	}
	if other, err := uc.repo.FindAgentMcpServerByName(ctx, m.UserID, name); err == nil && other.ID != m.ID {
		return response.BadRequest("已经有一个叫 " + name + " 的 MCP 服务器了")
	}
	before := agentJSON([]any{m.Transport, m.Command, m.Args, m.Env, m.URL, m.Headers})
	m.Name, m.Description, m.DefaultOn = name, strings.TrimSpace(in.Description), in.DefaultOn
	if in.Transport == "http" {
		u, err := url.Parse(strings.TrimSpace(in.Url))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return response.BadRequest("地址要以 http:// 或 https:// 开头")
		}
		m.Transport, m.URL, m.Headers = agentmcpserver.TransportHTTP, u.String(), uc.mergeSecrets(m.Headers, in.Headers)
		m.Command, m.Args, m.Env = "", nil, ""
	} else {
		command := strings.TrimSpace(in.Command)
		if command == "" {
			return response.BadRequest("先填命令")
		}
		m.Transport, m.Command, m.Args, m.Env = agentmcpserver.TransportStdio, command, in.Args, uc.mergeSecrets(m.Env, in.Env)
		m.URL, m.Headers = "", ""
	}
	if before != agentJSON([]any{m.Transport, m.Command, m.Args, m.Env, m.URL, m.Headers}) {
		m.Status, m.StatusText, m.ToolCount, m.TestedAt = "untested", "", 0, nil
	}
	return nil
}

// CreateMcpServer 新建一个自己的 MCP 服务器；带共享范围时（Agent 管理页，要管理员）新建成共享的。
func (uc *AgentUsecase) CreateMcpServer(ctx context.Context, in *v1.AgentMcpServer) (*v1.AgentMcpServer, error) {
	userID, err := agentUserID(ctx)
	if err != nil {
		return nil, err
	}
	m := &gen.AgentMcpServer{ID: "mcp_" + uuid.NewString(), UserID: userID, Status: "untested"}
	if in.GetShare() != "" {
		if err := uc.requireAdmin(ctx); err != nil {
			return nil, err
		}
		m.UserID = agentSharedOwner
		if m.Share, m.ShareUsers, err = uc.shareOf(ctx, in.GetShare(), in.GetShareUserIds()); err != nil {
			return nil, err
		}
	}
	if err := uc.applyMcp(ctx, m, in); err != nil {
		return nil, err
	}
	if err := uc.repo.SaveAgentMcpServer(ctx, m); err != nil {
		return nil, ErrSystem(err)
	}
	uc.configChanged()
	return uc.mcpPB(m), nil
}

// UpdateMcpServer 改自己的 MCP 服务器；共享的要管理员，带了共享范围就一起改。
func (uc *AgentUsecase) UpdateMcpServer(ctx context.Context, id string, in *v1.AgentMcpServer) (*v1.AgentMcpServer, error) {
	m, err := uc.editableMcp(ctx, id)
	if err != nil {
		return nil, err
	}
	if m.UserID == agentSharedOwner && in.GetShare() != "" {
		if m.Share, m.ShareUsers, err = uc.shareOf(ctx, in.GetShare(), in.GetShareUserIds()); err != nil {
			return nil, err
		}
	}
	if err := uc.applyMcp(ctx, m, in); err != nil {
		return nil, err
	}
	if err := uc.repo.SaveAgentMcpServer(ctx, m); err != nil {
		return nil, ErrSystem(err)
	}
	uc.configChanged()
	return uc.mcpPB(m), nil
}

// DeleteMcpServer 删除 MCP 服务器；有 agent:admin 时也能删别人的（Agent 管理页）。
func (uc *AgentUsecase) DeleteMcpServer(ctx context.Context, id string) error {
	scope, err := uc.cleanScope(ctx)
	if err != nil {
		return err
	}
	if _, err := uc.mcpIn(ctx, scope, id); err != nil {
		return err
	}
	if err := uc.repo.DeleteAgentMcpServer(ctx, id); err != nil {
		return ErrSystem(err)
	}
	uc.configChanged()
	return nil
}

// SetMcpDefault 新会话默认开不开；共享的是管理员替被共享到的人定。
func (uc *AgentUsecase) SetMcpDefault(ctx context.Context, id string, on bool) error {
	m, err := uc.editableMcp(ctx, id)
	if err != nil {
		return err
	}
	m.DefaultOn = on
	if err := uc.repo.SaveAgentMcpServer(ctx, m); err != nil {
		return ErrSystem(err)
	}
	uc.configChanged()
	return nil
}

// TestMcpServer 测试连接。只给 id 时测已保存的并把结果记下来；带草稿时测草稿，草稿里留空的密钥用已保存的同名值。
func (uc *AgentUsecase) TestMcpServer(ctx context.Context, req *v1.TestAgentMcpServerRequest) (*v1.TestAgentMcpServerResponse, error) {
	var stored *gen.AgentMcpServer
	if req.GetId() != "" {
		m, err := uc.editableMcp(ctx, req.GetId())
		if err != nil {
			return nil, err
		}
		stored = m
	}
	target := stored
	if draft := req.GetDraft(); draft != nil {
		target = &gen.AgentMcpServer{}
		if stored != nil {
			copied := *stored
			target = &copied
		}
		if draft.Transport == "http" {
			target.Transport, target.URL = agentmcpserver.TransportHTTP, strings.TrimSpace(draft.Url)
			target.Headers = uc.mergeSecrets(target.Headers, draft.Headers)
		} else {
			target.Transport, target.Command, target.Args = agentmcpserver.TransportStdio, strings.TrimSpace(draft.Command), draft.Args
			target.Env = uc.mergeSecrets(target.Env, draft.Env)
		}
	}
	if target == nil {
		return nil, response.BadRequest("缺少 MCP 服务器配置")
	}
	server := mcpprobe.Server{Transport: string(target.Transport)}
	if target.Transport == agentmcpserver.TransportHTTP {
		if target.URL == "" {
			return nil, response.BadRequest("先填地址")
		}
		server.URL, server.Headers = target.URL, uc.secrets(target.Headers)
	} else {
		if target.Command == "" {
			return nil, response.BadRequest("先填命令")
		}
		server.Command, server.Args, server.Env, server.Dir = target.Command, target.Args, uc.secrets(target.Env), uc.dir("tmp")
	}
	probeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	result, probeErr := mcpprobe.Probe(probeCtx, server)
	resp := &v1.TestAgentMcpServerResponse{Ok: probeErr == nil}
	if probeErr != nil {
		resp.Message = probeErr.Error()
	} else {
		resp.Tools = result.Tools
	}
	if stored != nil && req.GetDraft() == nil {
		now := time.Now()
		stored.TestedAt = &now
		stored.Status, stored.StatusText, stored.ToolCount = "ok", "", len(resp.Tools)
		if !resp.Ok {
			stored.Status, stored.StatusText, stored.ToolCount = "fail", resp.Message, 0
		}
		_ = uc.repo.SaveAgentMcpServer(context.WithoutCancel(ctx), stored)
	}
	return resp, nil
}

// ImportMcpServers 粘贴 {"mcpServers": {...}} 导入。重名的自动加序号。带共享范围时（Agent 管理页，要管理员）导入成共享的。
func (uc *AgentUsecase) ImportMcpServers(ctx context.Context, req *v1.ImportAgentMcpServersRequest) ([]*v1.AgentMcpServer, error) {
	owner, err := agentUserID(ctx)
	if err != nil {
		return nil, err
	}
	share, shareUsers := req.GetShare(), req.GetShareUserIds()
	if share != "" {
		if err := uc.requireAdmin(ctx); err != nil {
			return nil, err
		}
		if share, shareUsers, err = uc.shareOf(ctx, share, shareUsers); err != nil {
			return nil, err
		}
		owner = agentSharedOwner
	}
	names, servers, err := mcpprobe.ParseConfig(req.GetJson())
	if err != nil {
		return nil, agentErr(err)
	}
	existing, err := uc.repo.AgentMcpServerNames(ctx, owner)
	if err != nil {
		return nil, ErrSystem(err)
	}
	taken := map[string]bool{}
	for _, name := range existing {
		taken[name] = true
	}
	kvs := func(m map[string]string) []*v1.AgentSecretKV {
		out := make([]*v1.AgentSecretKV, 0, len(m))
		for _, key := range sortedKeys(m) {
			out = append(out, &v1.AgentSecretKV{Key: key, Value: m[key]})
		}
		return out
	}
	var out []*v1.AgentMcpServer
	for _, name := range names {
		clean := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", "\t", "-").Replace(strings.TrimSpace(name))
		if clean == "" {
			continue
		}
		unique := clean
		for n := 2; taken[unique]; n++ {
			unique = fmt.Sprintf("%s-%d", clean, n)
		}
		taken[unique] = true
		s := servers[name]
		in := &v1.AgentMcpServer{
			Name: unique, Transport: s.Transport, Command: s.Command, Args: s.Args, Env: kvs(s.Env), Url: s.URL, Headers: kvs(s.Headers),
			Share: share, ShareUserIds: shareUsers,
		}
		created, err := uc.CreateMcpServer(ctx, in)
		if err != nil {
			return nil, err
		}
		out = append(out, created)
	}
	if len(out) == 0 {
		return nil, response.BadRequest("这段 JSON 里没有能导入的 MCP 服务器")
	}
	return out, nil
}

// sessionMcpServers 会话开启的 MCP 服务器，交给引擎（Agent 不支持的传输方式由引擎去掉）。只用得上会话主人自己的
// 和共享给他的；名字撞了用他自己的（Agent 按名字区分 MCP 服务器）。
func (uc *AgentUsecase) sessionMcpServers(ctx context.Context, userID string, ids []string) ([]agentrun.McpServer, error) {
	rows, err := uc.repo.ListAgentMcpServersByIDs(ctx, ids)
	if err != nil {
		return nil, ErrSystem(err)
	}
	own := map[string]bool{}
	for _, m := range rows {
		if m.UserID == userID {
			own[m.Name] = true
		}
	}
	servers := make([]agentrun.McpServer, 0, len(rows))
	for _, m := range rows {
		if !usableBy(m.UserID, m.Share, m.ShareUsers, userID) || (m.UserID != userID && own[m.Name]) {
			continue
		}
		if m.Transport == agentmcpserver.TransportHTTP {
			servers = append(servers, agentrun.McpServer{Name: m.Name, HTTP: true, URL: m.URL, Headers: uc.secrets(m.Headers)})
			continue
		}
		servers = append(servers, agentrun.McpServer{Name: m.Name, Command: m.Command, Args: m.Args, Env: uc.secrets(m.Env)})
	}
	return servers, nil
}

// ———————————————————————————— 技能 ————————————————————————————

// skillStore 一个用户的技能库：每个人的技能各放各的，不同的人可以有同名的技能。共享的单独放一处。
func (uc *AgentUsecase) skillStore(userID string) agentskillpkg.Store {
	if userID == agentSharedOwner {
		return agentskillpkg.Store{Root: uc.dir("shared", "skills")}
	}
	return agentskillpkg.Store{Root: uc.userDir(userID, "skills")}
}

func skillPB(s *gen.AgentSkill) *v1.AgentSkill {
	return &v1.AgentSkill{
		Id: s.ID, Name: s.Name, Description: s.Description, Source: string(s.Source), Enabled: s.Enabled,
		FileCount: int32(s.FileCount), UserId: s.UserID, Share: s.Share, ShareUserIds: s.ShareUsers,
	}
}

// sharedSkillsFor 共享给 userID 的技能；enabledOnly 时只要启用着的。
func (uc *AgentUsecase) sharedSkillsFor(ctx context.Context, userID string, enabledOnly bool) ([]*gen.AgentSkill, error) {
	rows, err := uc.repo.ListAgentSkills(ctx, agentSharedOwner)
	if err != nil {
		return nil, err
	}
	out := rows[:0]
	for _, s := range rows {
		if (!enabledOnly || s.Enabled) && sharedTo(s.UserID, s.Share, s.ShareUsers, userID) {
			out = append(out, s)
		}
	}
	return out, nil
}

// ListSkills 自己的技能，后面跟着共享给自己的；enabledOnly 为 true 时只要启用着的。
// all 时（Agent 管理页）是所有人的和共享的，连同它们的主人和共享给的人。
func (uc *AgentUsecase) ListSkills(ctx context.Context, enabledOnly, all bool) ([]*v1.AgentSkill, []*v1.AgentUser, error) {
	userID, err := uc.listScope(ctx, all)
	if err != nil {
		return nil, nil, err
	}
	var rows []*gen.AgentSkill
	if enabledOnly && userID != "" {
		rows, err = uc.repo.ListEnabledAgentSkills(ctx, userID)
	} else {
		rows, err = uc.repo.ListAgentSkills(ctx, userID)
	}
	if err != nil {
		return nil, nil, ErrSystem(err)
	}
	if !all {
		shared, err := uc.sharedSkillsFor(ctx, userID, enabledOnly)
		if err != nil {
			return nil, nil, ErrSystem(err)
		}
		rows = append(rows, shared...)
	}
	out := make([]*v1.AgentSkill, 0, len(rows))
	people := make([]string, 0, len(rows))
	for _, s := range rows {
		if enabledOnly && !s.Enabled {
			continue
		}
		out = append(out, skillPB(s))
		people = append(append(people, s.UserID), s.ShareUsers...)
	}
	if !all {
		return out, nil, nil
	}
	users, err := uc.agentUsersPB(ctx, people)
	if err != nil {
		return nil, nil, err
	}
	return out, users, nil
}

// editableSkill 取一个能改的技能：自己的；共享的要管理员。别人的当作不存在。
func (uc *AgentUsecase) editableSkill(ctx context.Context, id string) (*gen.AgentSkill, error) {
	userID, err := agentUserID(ctx)
	if err != nil {
		return nil, err
	}
	s, err := uc.skillIn(ctx, "", id)
	if err != nil {
		return nil, err
	}
	if s.UserID == agentSharedOwner {
		if err := uc.requireAdmin(ctx); err != nil {
			return nil, err
		}
		return s, nil
	}
	if s.UserID != userID {
		return nil, ErrAgentSkillNotFound
	}
	return s, nil
}

// skillIn 取一个属于 scope 的技能（scope 为空是谁的都行）；不属于时当作不存在。
func (uc *AgentUsecase) skillIn(ctx context.Context, scope, id string) (*gen.AgentSkill, error) {
	s, err := uc.repo.GetAgentSkill(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrAgentSkillNotFound
		}
		return nil, ErrSystem(err)
	}
	if !ownedBy(scope, s.UserID) {
		return nil, ErrAgentSkillNotFound
	}
	return s, nil
}

// GetSkill 看一个技能的内容：自己的；共享的被共享到的人能看（改不了），管理员都能看。
func (uc *AgentUsecase) GetSkill(ctx context.Context, id string) (*v1.GetAgentSkillResponse, error) {
	userID, err := agentUserID(ctx)
	if err != nil {
		return nil, err
	}
	s, err := uc.skillIn(ctx, "", id)
	if err != nil {
		return nil, err
	}
	if !usableBy(s.UserID, s.Share, s.ShareUsers, userID) && (s.UserID != agentSharedOwner || !uc.isAdmin(ctx)) {
		return nil, ErrAgentSkillNotFound
	}
	resp := &v1.GetAgentSkillResponse{Skill: skillPB(s)}
	for _, f := range uc.skillStore(s.UserID).Read(s.Name) {
		resp.Files = append(resp.Files, &v1.AgentSkillFile{Path: f.Path, Content: f.Content, Binary: f.Binary})
	}
	return resp, nil
}

func agentSkillFiles(in []*v1.AgentSkillFile) []agentskillpkg.File {
	out := make([]agentskillpkg.File, 0, len(in))
	for _, f := range in {
		out = append(out, agentskillpkg.File{Path: f.GetPath(), Content: f.GetContent(), Binary: f.GetBinary()})
	}
	return out
}

// saveSkill 新建或更新一个技能：名称不能和同一个人的别的技能重复。
func (uc *AgentUsecase) saveSkill(ctx context.Context, s *gen.AgentSkill, name, description string, enabled bool, files []*v1.AgentSkillFile) (*v1.AgentSkill, error) {
	name, description = strings.TrimSpace(name), strings.TrimSpace(description)
	if name == "" {
		return nil, response.BadRequest("先填一个名称")
	}
	if other, err := uc.repo.FindAgentSkillByName(ctx, s.UserID, name); err == nil && other.ID != s.ID {
		return nil, response.BadRequest("已经有一个叫 " + name + " 的技能了")
	}
	count, err := uc.skillStore(s.UserID).Write(name, description, s.Name, agentSkillFiles(files))
	if err != nil {
		return nil, agentErr(err)
	}
	s.Name, s.Description, s.Enabled, s.FileCount = name, description, enabled, count
	if err := uc.repo.SaveAgentSkill(ctx, s); err != nil {
		return nil, ErrSystem(err)
	}
	uc.configChanged()
	return skillPB(s), nil
}

// CreateSkill 新建一个自己的技能；带共享范围时（Agent 管理页，要管理员）新建成共享的。
func (uc *AgentUsecase) CreateSkill(ctx context.Context, req *v1.CreateAgentSkillRequest) (*v1.AgentSkill, error) {
	userID, err := agentUserID(ctx)
	if err != nil {
		return nil, err
	}
	s := &gen.AgentSkill{ID: "skill_" + uuid.NewString(), UserID: userID, Source: agentskill.SourceLocal}
	if req.GetShare() != "" {
		if err := uc.requireAdmin(ctx); err != nil {
			return nil, err
		}
		s.UserID = agentSharedOwner
		if s.Share, s.ShareUsers, err = uc.shareOf(ctx, req.GetShare(), req.GetShareUserIds()); err != nil {
			return nil, err
		}
	}
	return uc.saveSkill(ctx, s, req.Name, req.Description, req.Enabled, req.Files)
}

// UpdateSkill 改自己的技能；共享的要管理员，带了共享范围就一起改。
func (uc *AgentUsecase) UpdateSkill(ctx context.Context, req *v1.UpdateAgentSkillRequest) (*v1.AgentSkill, error) {
	s, err := uc.editableSkill(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if s.UserID == agentSharedOwner && req.GetShare() != "" {
		if s.Share, s.ShareUsers, err = uc.shareOf(ctx, req.GetShare(), req.GetShareUserIds()); err != nil {
			return nil, err
		}
	}
	return uc.saveSkill(ctx, s, req.Name, req.Description, req.Enabled, req.Files)
}

// DeleteSkill 删除技能和它的文件；有 agent:admin 时也能删别人的（Agent 管理页）。
func (uc *AgentUsecase) DeleteSkill(ctx context.Context, id string) error {
	scope, err := uc.cleanScope(ctx)
	if err != nil {
		return err
	}
	s, err := uc.skillIn(ctx, scope, id)
	if err != nil {
		return err
	}
	if err := uc.repo.DeleteAgentSkill(ctx, id); err != nil {
		return ErrSystem(err)
	}
	uc.configChanged()
	uc.skillStore(s.UserID).Remove(s.Name)
	return nil
}

// SetSkillEnabled 启用、停用；共享的是管理员替被共享到的人定。
func (uc *AgentUsecase) SetSkillEnabled(ctx context.Context, id string, enabled bool) error {
	s, err := uc.editableSkill(ctx, id)
	if err != nil {
		return err
	}
	s.Enabled = enabled
	if err := uc.repo.SaveAgentSkill(ctx, s); err != nil {
		return ErrSystem(err)
	}
	uc.configChanged()
	return nil
}

// ImportSkill 从上传的压缩包、压缩包链接、Git 地址或服务器目录导入技能。一个来源里可以有多个技能，同名的覆盖。
// 带共享范围时（Agent 管理页，要管理员）导入成共享的，覆盖的同名共享技能也改成这个范围。
func (uc *AgentUsecase) ImportSkill(ctx context.Context, req *v1.ImportAgentSkillRequest) ([]*v1.AgentSkill, error) {
	userID, err := agentUserID(ctx)
	if err != nil {
		return nil, err
	}
	share, shareUsers := req.GetShare(), req.GetShareUserIds()
	if share != "" {
		if err := uc.requireAdmin(ctx); err != nil {
			return nil, err
		}
		if share, shareUsers, err = uc.shareOf(ctx, share, shareUsers); err != nil {
			return nil, err
		}
		userID = agentSharedOwner
	}
	bg := context.WithoutCancel(ctx)
	var source agentskill.Source
	var root, location, rootName string
	switch req.Source {
	case "zip", "url", "git":
		root = uc.dir("tmp", "skill-"+uuid.NewString()[:8])
		defer agentfs.RemoveAll(root)
	}
	switch req.Source {
	case "zip":
		if err := agentskillpkg.ExtractZip(bg, req.Data, root); err != nil {
			return nil, agentErr(err)
		}
		source = agentskill.SourceZip
	case "url":
		if location = strings.TrimSpace(req.Location); location == "" {
			return nil, response.BadRequest("先填压缩包链接")
		}
		fetchCtx, cancel := context.WithTimeout(bg, 5*time.Minute)
		err := agentskillpkg.FetchZip(fetchCtx, location, root)
		cancel()
		if err != nil {
			return nil, agentErr(err)
		}
		source = agentskill.SourceZip
	case "git":
		if location = strings.TrimSpace(req.Location); location == "" {
			return nil, response.BadRequest("先填 Git 地址")
		}
		// 克隆进一个还不存在的子目录：git clone 要求目标目录不存在或为空。
		dest := filepath.Join(root, "repo")
		gitCtx, cancel := context.WithTimeout(bg, 5*time.Minute)
		err := agentgit.Clone(gitCtx, location, "", dest, 1)
		cancel()
		if err != nil {
			return nil, response.BadRequest("克隆失败：" + err.Error())
		}
		source, root, rootName = agentskill.SourceGit, dest, agentgit.RepoName(location)
	case "dir":
		clean, err := agentfs.CleanDir(req.Location)
		if err != nil {
			return nil, agentErr(err)
		}
		if !agentfs.IsDir(clean) {
			return nil, response.BadRequest("服务器上没有这个目录")
		}
		source, root, location = agentskill.SourceDir, clean, clean
	default:
		return nil, response.BadRequest("不认识的导入方式")
	}
	store := uc.skillStore(userID)
	var out []*v1.AgentSkill
	for _, found := range agentskillpkg.Find(root, rootName) {
		count, err := store.Import(found)
		if err != nil {
			return nil, response.BadRequest("复制技能 " + found.Name + " 失败：" + err.Error())
		}
		s, err := uc.repo.FindAgentSkillByName(bg, userID, found.Name)
		if err != nil {
			s = &gen.AgentSkill{ID: "skill_" + uuid.NewString(), UserID: userID, Name: found.Name, Enabled: true}
		}
		s.Description, s.Source, s.Location, s.FileCount = found.Description, source, location, count
		if share != "" {
			s.Share, s.ShareUsers = share, shareUsers
		}
		if err := uc.repo.SaveAgentSkill(bg, s); err != nil {
			return nil, ErrSystem(err)
		}
		out = append(out, skillPB(s))
	}
	uc.configChanged()
	if len(out) == 0 {
		return nil, response.BadRequest("里面没有找到技能（需要带 SKILL.md 的目录）")
	}
	return out, nil
}

// enabledSkills 这个人现在启用着的技能，包括共享给他的：名称 → 它在技能库里的目录和版本（保存时间，内容变了它就变）。
// 名字撞了用他自己的。
func (uc *AgentUsecase) enabledSkills(ctx context.Context, userID string) map[string]agentskillpkg.Enabled {
	out := map[string]agentskillpkg.Enabled{}
	if shared, err := uc.sharedSkillsFor(ctx, userID, true); err == nil {
		store := uc.skillStore(agentSharedOwner)
		for _, s := range shared {
			out[s.Name] = agentskillpkg.Enabled{Dir: store.Dir(s.Name), Version: strconv.FormatInt(s.UpdateTime.UnixNano(), 10)}
		}
	}
	rows, err := uc.repo.ListEnabledAgentSkills(ctx, userID)
	if err != nil {
		return out
	}
	store := uc.skillStore(userID)
	for _, s := range rows {
		out[s.Name] = agentskillpkg.Enabled{Dir: store.Dir(s.Name), Version: strconv.FormatInt(s.UpdateTime.UnixNano(), 10)}
	}
	return out
}
