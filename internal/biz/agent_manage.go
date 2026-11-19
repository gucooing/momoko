package biz

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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
	"momoko/pkg/agentrun"
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
				rt.AuthLabel, rt.AuthDetail = result.Auth.Label, result.Auth.Description
			}
			if !result.NeedAuth {
				rt.Catalog = agentJSON(agentrun.MergeCatalog(agentParse[agentrun.Catalog](rt.Catalog), result.Catalog, false))
			}
		}
	}
	if err != nil {
		rt.Ready, rt.LoggedIn, rt.Error = false, false, err.Error()
	}
	if err := uc.repo.SaveAgentRuntime(bg, rt); err != nil {
		return nil, ErrSystem(err)
	}
	uc.configChanged()
	return rt, nil
}

// activeByRuntime 每个运行时现在有多少个会话的进程在跑。
func (uc *AgentUsecase) activeByRuntime() map[string]int {
	out := map[string]int{}
	for _, l := range uc.pool.Snapshot() {
		out[l.runtimeID]++
	}
	return out
}

func (uc *AgentUsecase) runtimePB(rt *gen.AgentRuntime, active map[string]int) *v1.AgentRuntime {
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
		Models:         agentModelsPB(agentParse[agentrun.Catalog](rt.Catalog).Models),
	}
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
	active := uc.activeByRuntime()
	resp := &v1.ListAgentRuntimesResponse{Settings: uc.runSettings(ctx)}
	for _, rt := range rts {
		resp.Runtimes = append(resp.Runtimes, uc.runtimePB(rt, active))
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
	return uc.runtimePB(rt, uc.activeByRuntime()), nil
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
	return uc.runtimePB(rt, uc.activeByRuntime()), nil
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
	resp.Runtime = uc.runtimePB(rt, uc.activeByRuntime())
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
	return uc.runtimePB(rt, nil), nil
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

func (uc *AgentUsecase) ListProviders(ctx context.Context) ([]*v1.AgentProvider, error) {
	rows, err := uc.repo.ListAgentProviders(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := make([]*v1.AgentProvider, 0, len(rows))
	for _, p := range rows {
		out = append(out, uc.providerPB(p))
	}
	return out, nil
}

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

// applyAPIProvider 把表单里的第三方接口配置写到记录上；API Key 留空表示保留原值。
func (uc *AgentUsecase) applyAPIProvider(p *gen.AgentProvider, in *v1.AgentProvider, creating bool) error {
	name := strings.TrimSpace(in.GetName())
	if name == "" {
		return response.BadRequest("先填一个名称")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(in.GetBaseUrl()), "/")
	if u, err := url.Parse(baseURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return response.BadRequest("接口地址要以 http:// 或 https:// 开头")
	}
	if creating {
		if _, ok := uc.driverForVendor(in.Vendor); !ok {
			return response.BadRequest("接口协议只能是 Anthropic 兼容或 OpenAI 兼容")
		}
		p.Vendor = in.Vendor
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
	p.MainModel, p.FastModel = pick(in.MainModel), ""
	if p.Vendor == agentapi.Anthropic {
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
	return nil
}

func (uc *AgentUsecase) CreateProvider(ctx context.Context, in *v1.AgentProvider) (*v1.AgentProvider, error) {
	p := &gen.AgentProvider{ID: "prov_" + uuid.NewString(), Status: "untested"}
	if err := uc.applyAPIProvider(p, in, true); err != nil {
		return nil, err
	}
	if err := uc.saveProvider(ctx, p); err != nil {
		return nil, err
	}
	return uc.providerPB(p), nil
}

func (uc *AgentUsecase) UpdateProvider(ctx context.Context, id string, in *v1.AgentProvider) (*v1.AgentProvider, error) {
	p, err := uc.getProvider(ctx, id)
	if err != nil {
		return nil, err
	}
	before := p.BaseURL + "\x00" + p.APIKey
	if err := uc.applyAPIProvider(p, in, false); err != nil {
		return nil, err
	}
	// 地址或密钥变了，之前的测试结果就不作数了。
	if before != p.BaseURL+"\x00"+p.APIKey {
		p.Status, p.StatusText, p.TestLatencyMs, p.TestedAt = "untested", "", 0, nil
	}
	if err := uc.saveProvider(ctx, p); err != nil {
		return nil, err
	}
	return uc.providerPB(p), nil
}

func (uc *AgentUsecase) DeleteProvider(ctx context.Context, id string) error {
	if _, err := uc.getProvider(ctx, id); err != nil {
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

// draftProvider 「测试连接」「获取可用模型」用的配置：表单里的草稿，留空的地址和密钥用已保存的。
func (uc *AgentUsecase) draftProvider(ctx context.Context, id, vendor, baseURL, apiKey string) (string, string, string, *gen.AgentProvider, error) {
	var stored *gen.AgentProvider
	if id != "" {
		p, err := uc.getProvider(ctx, id)
		if err != nil {
			return "", "", "", nil, err
		}
		stored = p
	}
	baseURL, apiKey = strings.TrimRight(strings.TrimSpace(baseURL), "/"), strings.TrimSpace(apiKey)
	if stored != nil {
		if vendor == "" {
			vendor = stored.Vendor
		}
		if baseURL == "" {
			baseURL = stored.BaseURL
		}
		if apiKey == "" {
			apiKey = uc.decrypt(stored.APIKey)
		}
	}
	if baseURL == "" {
		return "", "", "", nil, response.BadRequest("先填接口地址")
	}
	if apiKey == "" {
		return "", "", "", nil, response.BadRequest("先填 API Key")
	}
	if _, ok := uc.driverForVendor(vendor); !ok {
		return "", "", "", nil, response.BadRequest("接口协议只能是 Anthropic 兼容或 OpenAI 兼容")
	}
	return vendor, baseURL, apiKey, stored, nil
}

// TestProvider 测试连接。测的是已保存的那一份时，把结果记到记录上。
func (uc *AgentUsecase) TestProvider(ctx context.Context, req *v1.TestAgentProviderRequest) (*v1.TestAgentProviderResponse, error) {
	draft := req.GetDraft()
	vendor, baseURL, apiKey, stored, err := uc.draftProvider(ctx, req.GetId(), draft.GetVendor(), draft.GetBaseUrl(), draft.GetApiKey())
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
	bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), 45*time.Second)
	defer cancel()
	latency, testErr := agentapi.Test(bg, vendor, baseURL, apiKey, model)
	resp := &v1.TestAgentProviderResponse{Model: model, LatencyMs: latency.Milliseconds(), Ok: testErr == nil}
	if testErr != nil {
		resp.Message = testErr.Error()
	}
	if stored != nil && baseURL == stored.BaseURL && apiKey == uc.decrypt(stored.APIKey) {
		now := time.Now()
		stored.TestedAt, stored.TestLatencyMs = &now, resp.LatencyMs
		stored.Status, stored.StatusText = "ok", ""
		if !resp.Ok {
			stored.Status, stored.StatusText = "bad", resp.Message
		}
		_ = uc.saveProvider(context.WithoutCancel(ctx), stored)
	}
	return resp, nil
}

func (uc *AgentUsecase) FetchProviderModels(ctx context.Context, req *v1.FetchAgentProviderModelsRequest) ([]string, error) {
	vendor, baseURL, apiKey, _, err := uc.draftProvider(ctx, req.GetId(), req.GetVendor(), req.GetBaseUrl(), req.GetApiKey())
	if err != nil {
		return nil, err
	}
	bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	models, err := agentapi.Models(bg, vendor, baseURL, apiKey)
	return models, agentErr(err)
}

// providerEnv 用某个供应商启动 Agent 进程时的环境变量。配置目录一直是 Agent 默认的，换供应商只是换这一组参数：
//   - 没有供应商：服务器上已有的登录，什么都不传；
//   - 第三方接口：接口地址、密钥、模型、最长上下文和输出靠环境注入（byEnv），换模型要重启进程。
func (uc *AgentUsecase) providerEnv(d agentdriver.Driver, p *gen.AgentProvider, model, effort string) (env map[string]string, byEnv bool, err error) {
	if p == nil {
		return nil, false, nil
	}
	if p.Vendor != d.Vendor() {
		return nil, false, response.BadRequest("这个供应商的接口和这个 Agent 不兼容")
	}
	key := uc.decrypt(p.APIKey)
	if key == "" {
		return nil, false, response.BadRequest("供应商「" + p.Name + "」没有可用的 API Key")
	}
	if model == "" {
		model = p.MainModel
	}
	api := agentdriver.API{Name: p.Name, BaseURL: p.BaseURL, Key: key, Model: model, FastModel: p.FastModel, Extra: map[string]string{}}
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

func (uc *AgentUsecase) ListWorkspaces(ctx context.Context) ([]*v1.AgentWorkspace, error) {
	rows, err := uc.repo.ListAgentWorkspaces(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	ids := make([]string, 0, len(rows))
	for _, w := range rows {
		ids = append(ids, w.ID)
	}
	stats, err := uc.repo.AgentWorkspaceStats(ctx, ids)
	if err != nil {
		return nil, ErrSystem(err)
	}
	names := uc.loadNames(ctx)
	out := make([]*v1.AgentWorkspace, 0, len(rows))
	for _, w := range rows {
		out = append(out, uc.workspacePB(w, names, stats))
	}
	return out, nil
}

func (uc *AgentUsecase) getWorkspace(ctx context.Context, id string) (*gen.AgentWorkspace, error) {
	w, err := uc.repo.GetAgentWorkspace(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrAgentWorkspaceNotFound
		}
		return nil, ErrSystem(err)
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

// checkDefaults 校验工作区的默认模型：模型定了，Agent 和供应商也就定了，两者要对得上。
func (uc *AgentUsecase) checkDefaults(ctx context.Context, runtimeID, providerID string) error {
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
	p, err := uc.getProvider(ctx, providerID)
	if err != nil {
		return err
	}
	if p.Vendor != uc.driverOf(string(rt.Kind)).Vendor() {
		return response.BadRequest("这个供应商的接口和这个 Agent 不兼容")
	}
	return nil
}

// pathFree 一个目录只能属于一个工作区。
func (uc *AgentUsecase) pathFree(ctx context.Context, path, exceptID string) error {
	existing, err := uc.repo.FindAgentWorkspaceByPath(ctx, path)
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
	if err := uc.checkDefaults(ctx, req.RuntimeId, req.ProviderId); err != nil {
		return nil, err
	}
	if err := checkAgentDocs(req.Docs); err != nil {
		return nil, err
	}
	w := &gen.AgentWorkspace{
		ID: "ws_" + uuid.NewString(), UserID: userID, Name: strings.TrimSpace(req.Name),
		RuntimeID: req.RuntimeId, ProviderID: req.ProviderId, Model: req.Model, Effort: req.Effort, Mode: req.Mode, McpIds: req.McpIds,
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
		if err := uc.pathFree(ctx, w.Path, ""); err != nil {
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
		if err := uc.pathFree(ctx, w.Path, ""); err != nil {
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
	if err := uc.checkDefaults(ctx, req.RuntimeId, req.ProviderId); err != nil {
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
			if err := uc.pathFree(ctx, clean, w.ID); err != nil {
				return nil, err
			}
			w.Path = clean
		}
	}
	w.RuntimeID, w.ProviderID, w.Model, w.Effort, w.Mode, w.McpIds = req.RuntimeId, req.ProviderId, req.Model, req.Effort, req.Mode, req.McpIds
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
func (uc *AgentUsecase) DeleteWorkspace(ctx context.Context, id string) error {
	if _, err := uc.getWorkspace(ctx, id); err != nil {
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
	return agentEntriesPB(entries), agentErr(err)
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
	}
	if m.TestedAt != nil {
		out.TestedAt = timestamppb.New(*m.TestedAt)
	}
	return out
}

func (uc *AgentUsecase) ListMcpServers(ctx context.Context) ([]*v1.AgentMcpServer, error) {
	rows, err := uc.repo.ListAgentMcpServers(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := make([]*v1.AgentMcpServer, 0, len(rows))
	for _, m := range rows {
		out = append(out, uc.mcpPB(m))
	}
	return out, nil
}

func (uc *AgentUsecase) getMcp(ctx context.Context, id string) (*gen.AgentMcpServer, error) {
	m, err := uc.repo.GetAgentMcpServer(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrAgentMcpNotFound
		}
		return nil, ErrSystem(err)
	}
	return m, nil
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
	if other, err := uc.repo.FindAgentMcpServerByName(ctx, name); err == nil && other.ID != m.ID {
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

func (uc *AgentUsecase) CreateMcpServer(ctx context.Context, in *v1.AgentMcpServer) (*v1.AgentMcpServer, error) {
	m := &gen.AgentMcpServer{ID: "mcp_" + uuid.NewString(), Status: "untested"}
	if err := uc.applyMcp(ctx, m, in); err != nil {
		return nil, err
	}
	if err := uc.repo.SaveAgentMcpServer(ctx, m); err != nil {
		return nil, ErrSystem(err)
	}
	uc.configChanged()
	return uc.mcpPB(m), nil
}

func (uc *AgentUsecase) UpdateMcpServer(ctx context.Context, id string, in *v1.AgentMcpServer) (*v1.AgentMcpServer, error) {
	m, err := uc.getMcp(ctx, id)
	if err != nil {
		return nil, err
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

func (uc *AgentUsecase) DeleteMcpServer(ctx context.Context, id string) error {
	if _, err := uc.getMcp(ctx, id); err != nil {
		return err
	}
	if err := uc.repo.DeleteAgentMcpServer(ctx, id); err != nil {
		return ErrSystem(err)
	}
	uc.configChanged()
	return nil
}

func (uc *AgentUsecase) SetMcpDefault(ctx context.Context, id string, on bool) error {
	m, err := uc.getMcp(ctx, id)
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
		m, err := uc.getMcp(ctx, req.GetId())
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

// ImportMcpServers 粘贴 {"mcpServers": {...}} 导入。重名的自动加序号。
func (uc *AgentUsecase) ImportMcpServers(ctx context.Context, raw string) ([]*v1.AgentMcpServer, error) {
	names, servers, err := mcpprobe.ParseConfig(raw)
	if err != nil {
		return nil, agentErr(err)
	}
	existing, err := uc.repo.AgentMcpServerNames(ctx)
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
		in := &v1.AgentMcpServer{Name: unique, Transport: s.Transport, Command: s.Command, Args: s.Args, Env: kvs(s.Env), Url: s.URL, Headers: kvs(s.Headers)}
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

// sessionMcpServers 会话开启的 MCP 服务器，交给引擎（Agent 不支持的传输方式由引擎去掉）。
func (uc *AgentUsecase) sessionMcpServers(ctx context.Context, ids []string) ([]agentrun.McpServer, error) {
	rows, err := uc.repo.ListAgentMcpServersByIDs(ctx, ids)
	if err != nil {
		return nil, ErrSystem(err)
	}
	servers := make([]agentrun.McpServer, 0, len(rows))
	for _, m := range rows {
		if m.Transport == agentmcpserver.TransportHTTP {
			servers = append(servers, agentrun.McpServer{Name: m.Name, HTTP: true, URL: m.URL, Headers: uc.secrets(m.Headers)})
			continue
		}
		servers = append(servers, agentrun.McpServer{Name: m.Name, Command: m.Command, Args: m.Args, Env: uc.secrets(m.Env)})
	}
	return servers, nil
}

// ———————————————————————————— 技能 ————————————————————————————

func (uc *AgentUsecase) skillStore() agentskillpkg.Store {
	return agentskillpkg.Store{Root: filepath.Join(uc.root, "skills")}
}

func skillPB(s *gen.AgentSkill) *v1.AgentSkill {
	return &v1.AgentSkill{Id: s.ID, Name: s.Name, Description: s.Description, Source: string(s.Source), Enabled: s.Enabled, FileCount: int32(s.FileCount)}
}

// ListSkills enabledOnly 为 true 时只要启用着的（筛选在 SQL 里）。
func (uc *AgentUsecase) ListSkills(ctx context.Context, enabledOnly bool) ([]*v1.AgentSkill, error) {
	list := uc.repo.ListAgentSkills
	if enabledOnly {
		list = uc.repo.ListEnabledAgentSkills
	}
	rows, err := list(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := make([]*v1.AgentSkill, 0, len(rows))
	for _, s := range rows {
		out = append(out, skillPB(s))
	}
	return out, nil
}

func (uc *AgentUsecase) getSkill(ctx context.Context, id string) (*gen.AgentSkill, error) {
	s, err := uc.repo.GetAgentSkill(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrAgentSkillNotFound
		}
		return nil, ErrSystem(err)
	}
	return s, nil
}

func (uc *AgentUsecase) GetSkill(ctx context.Context, id string) (*v1.GetAgentSkillResponse, error) {
	s, err := uc.getSkill(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := &v1.GetAgentSkillResponse{Skill: skillPB(s)}
	for _, f := range uc.skillStore().Read(s.Name) {
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

// saveSkill 新建或更新一个技能：名称不能和别的技能重复。
func (uc *AgentUsecase) saveSkill(ctx context.Context, s *gen.AgentSkill, name, description string, enabled bool, files []*v1.AgentSkillFile) (*v1.AgentSkill, error) {
	name, description = strings.TrimSpace(name), strings.TrimSpace(description)
	if name == "" {
		return nil, response.BadRequest("先填一个名称")
	}
	if other, err := uc.repo.FindAgentSkillByName(ctx, name); err == nil && other.ID != s.ID {
		return nil, response.BadRequest("已经有一个叫 " + name + " 的技能了")
	}
	count, err := uc.skillStore().Write(name, description, s.Name, agentSkillFiles(files))
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

func (uc *AgentUsecase) CreateSkill(ctx context.Context, req *v1.CreateAgentSkillRequest) (*v1.AgentSkill, error) {
	s := &gen.AgentSkill{ID: "skill_" + uuid.NewString(), Source: agentskill.SourceLocal}
	return uc.saveSkill(ctx, s, req.Name, req.Description, req.Enabled, req.Files)
}

func (uc *AgentUsecase) UpdateSkill(ctx context.Context, req *v1.UpdateAgentSkillRequest) (*v1.AgentSkill, error) {
	s, err := uc.getSkill(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return uc.saveSkill(ctx, s, req.Name, req.Description, req.Enabled, req.Files)
}

func (uc *AgentUsecase) DeleteSkill(ctx context.Context, id string) error {
	s, err := uc.getSkill(ctx, id)
	if err != nil {
		return err
	}
	if err := uc.repo.DeleteAgentSkill(ctx, id); err != nil {
		return ErrSystem(err)
	}
	uc.configChanged()
	uc.skillStore().Remove(s.Name)
	return nil
}

func (uc *AgentUsecase) SetSkillEnabled(ctx context.Context, id string, enabled bool) error {
	s, err := uc.getSkill(ctx, id)
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
func (uc *AgentUsecase) ImportSkill(ctx context.Context, req *v1.ImportAgentSkillRequest) ([]*v1.AgentSkill, error) {
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
	store := uc.skillStore()
	var out []*v1.AgentSkill
	for _, found := range agentskillpkg.Find(root, rootName) {
		count, err := store.Import(found)
		if err != nil {
			return nil, response.BadRequest("复制技能 " + found.Name + " 失败：" + err.Error())
		}
		s, err := uc.repo.FindAgentSkillByName(bg, found.Name)
		if err != nil {
			s = &gen.AgentSkill{ID: "skill_" + uuid.NewString(), Name: found.Name, Enabled: true}
		}
		s.Description, s.Source, s.Location, s.FileCount = found.Description, source, location, count
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

// enabledSkills 现在启用着的技能：名称 → 它在技能库里的目录和版本（保存时间，内容变了它就变）。
func (uc *AgentUsecase) enabledSkills(ctx context.Context) map[string]agentskillpkg.Enabled {
	out := map[string]agentskillpkg.Enabled{}
	rows, err := uc.repo.ListEnabledAgentSkills(ctx)
	if err != nil {
		return out
	}
	store := uc.skillStore()
	for _, s := range rows {
		out[s.Name] = agentskillpkg.Enabled{Dir: store.Dir(s.Name), Version: strconv.FormatInt(s.UpdateTime.UnixNano(), 10)}
	}
	return out
}
