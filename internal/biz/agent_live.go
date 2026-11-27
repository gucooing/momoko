package biz

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/agentsession"
	"momoko/pkg/agentdriver"
	"momoko/pkg/agentrun"
	"momoko/pkg/agentskill"
	"momoko/pkg/response"
)

var agentJSONOptions = protojson.MarshalOptions{EmitUnpopulated: true}

// agentLive 一个有进程在跑的会话。它是会话引擎的回调（agentrun.Sink），也是进程池里的成员（agentrun.Worker）。
// 引擎的回调在引擎的锁里，不能做慢事：推给页面直接发，落库交给这个会话自己的写入队列按顺序做。
type agentLive struct {
	uc         *AgentUsecase
	id         string
	userID     string
	runtimeID  string
	providerID string
	// byEnv 模型和推理强度是启动时靠环境定下来的（第三方接口），Agent 报的值不算数。
	byEnv  bool
	launch agentdriver.Launch
	// session 启动完成后才有；启动过程中引擎就可能报事件上来，那些回调不碰它。
	session *agentrun.Session
	w       *agentWriter
	// dead 会话已经删除：进程收尾时引擎报上来的东西不再落库、不再推送。
	dead atomic.Bool
	// exited 进程已经没了。
	exited atomic.Bool

	stateMu     sync.Mutex
	state       *agentrun.State
	stateQueued bool
}

func (l *agentLive) Busy() bool          { return l.session.Busy() }
func (l *agentLive) ActiveAt() time.Time { return l.session.ActiveAt() }
func (l *agentLive) Shutdown()           { l.session.Shutdown() }
func (l *agentLive) Alive() bool         { return !l.exited.Load() }

// push 推一条对话流里的消息给打开着这个会话的页面。
func (l *agentLive) push(msg map[string]any) {
	if l.dead.Load() {
		return
	}
	msg["id"] = l.id
	if raw, err := json.Marshal(msg); err == nil {
		l.uc.hub.ToSession(l.id, raw)
	}
}

func (l *agentLive) Item(item *agentrun.Item, push bool) {
	if l.dead.Load() {
		return
	}
	payload := item.Payload()
	l.w.event(&gen.AgentEvent{SessionID: l.id, Seq: item.Seq, Rev: item.Rev, Type: item.Type, Sub: item.Sub, Payload: payload})
	if push {
		l.push(map[string]any{"t": "item", "item": json.RawMessage(payload)})
	}
}

func (l *agentLive) Delta(seq int64, text string) {
	l.push(map[string]any{"t": "delta", "seq": seq, "text": text})
}

// State 推给页面；落库合并着做：队列里已经排着一次写状态的，就只换成最新的这份。
func (l *agentLive) State(state agentrun.State) {
	l.push(map[string]any{"t": "state", "state": state})
	l.stateMu.Lock()
	l.state = &state
	queued := l.stateQueued
	l.stateQueued = true
	l.stateMu.Unlock()
	if !queued {
		l.w.do(func(ctx context.Context) {
			l.stateMu.Lock()
			state := l.state
			l.stateQueued = false
			l.stateMu.Unlock()
			_, _ = l.uc.repo.UpdateAgentSession(ctx, l.id, func(u *gen.AgentSessionUpdateOne) { u.SetState(agentJSON(state)) })
		})
	}
}

// Usage 上下文用量只推给页面，一轮结束时随会话一起落库。
func (l *agentLive) Usage(used, size int64) {
	l.push(map[string]any{"t": "usage", "contextUsed": used, "contextSize": size})
}

func (l *agentLive) Title(title string) {
	l.w.update(func(u *gen.AgentSessionUpdateOne) { u.SetTitle(title) })
}

func (l *agentLive) Config(mode, model, effort string) {
	l.w.update(func(u *gen.AgentSessionUpdateOne) {
		if mode != "" {
			u.SetMode(mode)
		}
		if !l.byEnv && model != "" {
			u.SetModel(model).SetEffort(effort)
		}
	})
}

func (l *agentLive) Asked(p *agentrun.Permission) {
	if l.dead.Load() {
		return
	}
	l.w.update(func(u *gen.AgentSessionUpdateOne) { u.SetStatus(agentsession.StatusWaiting) })
	approval := l.uc.approvals.add(l.id, p)
	detail := p.Command
	if detail == "" {
		detail = p.Title
	}
	go l.uc.notify(l.id, "ask", detail, approval)
}

// Questioned Agent 向人提问：和等批准一样，会话改成「等待」并通知人。问题只能在页面上答，
// 外部渠道的通知只带问题和打开会话的链接，没有回答用的按钮。
func (l *agentLive) Questioned(q *agentrun.Question) {
	if l.dead.Load() {
		return
	}
	l.w.update(func(u *gen.AgentSessionUpdateOne) { u.SetStatus(agentsession.StatusWaiting) })
	go l.uc.notify(l.id, "question", q.Message, nil)
}

func (l *agentLive) Answered(id, kind string, waiting bool) {
	if kind == "" {
		l.uc.settleApproval(l.id, id, "")
	}
	if !waiting {
		// 只改「等待批准」：还有一轮在跑、或者有子 Agent 在干活就回到「运行中」，都没有（v2 后台的工作要的批准）就是「空闲」；
		// 这一轮要是刚好结束了，状态以结束时写的为准。
		next := agentsession.StatusIdle
		if l.session.Working() {
			next = agentsession.StatusRunning
		}
		l.w.update(func(u *gen.AgentSessionUpdateOne) {
			u.Where(agentsession.StatusEQ(agentsession.StatusWaiting)).SetStatus(next)
		})
	}
}

// TurnStarted 一轮开始了（发消息开始的、排队的接着发的、v2 里 Agent 自己开始的）。空闲或出错的会话改成运行中；
// 发消息时已经改过的，这里不再动。
func (l *agentLive) TurnStarted() {
	if l.dead.Load() {
		return
	}
	l.w.update(func(u *gen.AgentSessionUpdateOne) {
		u.Where(agentsession.StatusIn(agentsession.StatusIdle, agentsession.StatusError)).
			SetStatus(agentsession.StatusRunning).SetError("").SetActiveAt(time.Now())
	})
}

// SubagentsWorking 主会话没有一轮在跑时子 Agent 开始干活、或者都停下了：空闲的改成运行中，运行中的改回空闲。
// 等人回应的不动（回应完按那时的样子改）；出错的不动，留着给人看。
func (l *agentLive) SubagentsWorking(on bool) {
	if l.dead.Load() {
		return
	}
	from, to := agentsession.StatusRunning, agentsession.StatusIdle
	if on {
		from, to = to, from
	}
	l.w.update(func(u *gen.AgentSessionUpdateOne) { u.Where(agentsession.StatusEQ(from)).SetStatus(to) })
}

func (l *agentLive) TurnEnded(end agentrun.TurnEnd) {
	if l.dead.Load() {
		return
	}
	failed := end.Turn.Stop == "error"
	l.w.update(func(u *gen.AgentSessionUpdateOne) {
		// 主会话这一轮完了，子 Agent 还在干活的会话还算运行中
		u.SetStatus(agentsession.StatusIdle).SetError("")
		if end.Working {
			u.SetStatus(agentsession.StatusRunning)
		}
		if failed {
			u.SetStatus(agentsession.StatusError).SetError(end.Turn.Error)
		}
		u.SetActiveAt(time.Now()).AddTurns(1).
			SetState(agentJSON(end.State)).
			SetContextUsed(end.ContextUsed).SetContextSize(end.ContextSize)
		// 用量 Agent 给了才累加。
		if usage := end.Turn.Usage; usage != nil {
			u.AddUsageTotal(usage.Total).AddUsageInput(usage.Input).AddUsageOutput(usage.Output).
				AddUsageCacheRead(usage.CacheRead).AddUsageCacheWrite(usage.CacheWrite).AddUsageThought(usage.Thought).
				SetLastUsage(agentJSON(usage))
		}
	})
	if end.Turn.Code == agentrun.TurnAuthRequired && l.providerID == "" {
		l.w.do(func(ctx context.Context) { l.uc.markLoggedOut(ctx, l.runtimeID) })
	}
	switch {
	case failed:
		go l.uc.notify(l.id, "error", end.Turn.Error, nil)
	case end.Turn.Stop != "cancelled":
		go l.uc.notify(l.id, "done", agentrun.Summary(end.LastText), nil)
	}
}

func (l *agentLive) Exited() {
	l.exited.Store(true)
	uc := l.uc
	uc.pool.Remove(l.id, l)
	uc.approvals.dropSession(l.id)
	uc.telegram.idle(uc)
	// 进程没了还显示「运行中 / 等待批准」的收成空闲；一轮里出错的在结束时已经记成出错。
	// 状态没变也推一次：页面上「有进程在跑」要跟着变。
	l.w.do(func(ctx context.Context) {
		if l.dead.Load() {
			return
		}
		s, err := uc.repo.UpdateAgentSession(ctx, l.id, func(u *gen.AgentSessionUpdateOne) {
			u.Where(agentsession.StatusIn(agentsession.StatusRunning, agentsession.StatusWaiting)).SetStatus(agentsession.StatusIdle)
		})
		if gen.IsNotFound(err) {
			s, err = uc.repo.GetAgentSession(ctx, l.id)
		}
		if err == nil {
			uc.pushSession(ctx, s)
		}
	})
	l.w.close()
}

// —— 写入队列 ——

// agentWriter 一个活着的会话的落库队列：事件、会话的改动按引擎报上来的顺序写，连着的事件合成一批写；
// 会话改了之后把改后的样子推给页面，推送的先后和改动的先后一致。
type agentWriter struct {
	uc   *AgentUsecase
	l    *agentLive
	mu   sync.Mutex
	ch   chan agentWrite
	shut bool
}

type agentWrite struct {
	event *gen.AgentEvent
	run   func(ctx context.Context)
}

func newAgentWriter(l *agentLive) *agentWriter {
	w := &agentWriter{uc: l.uc, l: l, ch: make(chan agentWrite, 1024)}
	go w.loop()
	return w
}

func (w *agentWriter) send(op agentWrite) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.shut {
		return false
	}
	w.ch <- op
	return true
}

func (w *agentWriter) event(e *gen.AgentEvent) { w.send(agentWrite{event: e}) }

func (w *agentWriter) do(fn func(ctx context.Context)) {
	if !w.send(agentWrite{run: fn}) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		fn(ctx)
	}
}

// update 改会话并把改后的样子推给页面。
func (w *agentWriter) update(fn func(u *gen.AgentSessionUpdateOne)) {
	w.do(func(ctx context.Context) {
		if w.l.dead.Load() {
			return
		}
		s, err := w.uc.repo.UpdateAgentSession(ctx, w.l.id, fn)
		if err != nil {
			if !gen.IsNotFound(err) {
				w.uc.log.Warnf("update agent session %s: %v", w.l.id, err)
			}
			return
		}
		w.uc.pushSession(ctx, s)
	})
}

// call 在队列里改会话（和引擎报上来的改动排在一起），等它改完，返回改后的样子。
func (w *agentWriter) call(ctx context.Context, fn func(u *gen.AgentSessionUpdateOne)) (*gen.AgentSession, error) {
	type result struct {
		s   *gen.AgentSession
		err error
	}
	done := make(chan result, 1)
	run := func(ctx context.Context) {
		s, err := w.uc.repo.UpdateAgentSession(ctx, w.l.id, fn)
		if err == nil {
			w.uc.pushSession(ctx, s)
		}
		done <- result{s, err}
	}
	if !w.send(agentWrite{run: run}) {
		run(ctx)
	}
	r := <-done
	return r.s, r.err
}

// flush 等队列里已经排着的都写完。
func (w *agentWriter) flush() {
	done := make(chan struct{})
	if w.send(agentWrite{run: func(context.Context) { close(done) }}) {
		<-done
	}
}

// close 不再收新的，排着的写完后队列结束。
func (w *agentWriter) close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.shut {
		w.shut = true
		close(w.ch)
	}
}

func (w *agentWriter) loop() {
	ctx := context.Background()
	var batch []*gen.AgentEvent
	flush := func() {
		if len(batch) == 0 || w.l.dead.Load() {
			batch = batch[:0]
			return
		}
		writeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		if err := w.uc.repo.SaveAgentEvents(writeCtx, batch); err != nil {
			w.uc.log.Warnf("save agent events %s: %v", w.l.id, err)
		}
		cancel()
		batch = batch[:0]
	}
	for op := range w.ch {
		// 把已经排着的一并取出来：连着的事件合成一批，遇到别的操作先把这一批写掉，保持先后。
		ops := []agentWrite{op}
		for more := true; more && len(ops) < 256; {
			select {
			case next, ok := <-w.ch:
				if ok {
					ops = append(ops, next)
				} else {
					more = false
				}
			default:
				more = false
			}
		}
		for _, op := range ops {
			if op.event != nil {
				// 同一条事件在一批里改了好几次，只写最后那次。
				if n := len(batch); n > 0 && batch[n-1].Seq == op.event.Seq {
					batch[n-1] = op.event
				} else {
					batch = append(batch, op.event)
				}
				continue
			}
			flush()
			runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			op.run(runCtx)
			cancel()
		}
		flush()
	}
}

// —— 启动 ——

// live 这个会话正在跑的进程；没有返回 nil。
func (uc *AgentUsecase) live(id string) *agentLive {
	l, _ := uc.pool.Get(id)
	return l
}

// updateSession 改一个会话并推给页面。有进程在跑时经它的写入队列，和引擎报上来的改动保持先后。
func (uc *AgentUsecase) updateSession(ctx context.Context, id string, fn func(u *gen.AgentSessionUpdateOne)) (*gen.AgentSession, error) {
	if l := uc.live(id); l != nil {
		return l.w.call(ctx, fn)
	}
	s, err := uc.repo.UpdateAgentSession(ctx, id, fn)
	if err != nil {
		return nil, err
	}
	uc.pushSession(ctx, s)
	return s, nil
}

// pushSession 会话的状态、标题或选项变了：推给它主人的会话栏、开着它的页面和管理页。
func (uc *AgentUsecase) pushSession(ctx context.Context, s *gen.AgentSession) {
	pb, err := agentJSONOptions.Marshal(uc.sessionPB(s, uc.loadNames(ctx)))
	if err != nil {
		return
	}
	if raw, err := json.Marshal(map[string]any{"t": "session", "id": s.ID, "session": json.RawMessage(pb)}); err == nil {
		uc.hub.ToSessionOrUser(s.ID, s.UserID, raw)
	}
}

// ensureLive 保证这个会话有进程：有就直接用；没有就（必要时让出位置后）起一个，并把 Agent 侧的上下文接回来。
func (uc *AgentUsecase) ensureLive(ctx context.Context, s *gen.AgentSession) (*agentLive, error) {
	l, err := uc.pool.Ensure(ctx, s.ID, func() (*agentLive, error) { return uc.startLive(ctx, s) })
	if errors.Is(err, agentrun.ErrPoolFull) {
		return nil, agentErr(err)
	}
	return l, err
}

func (uc *AgentUsecase) startLive(ctx context.Context, s *gen.AgentSession) (*agentLive, error) {
	bg := context.WithoutCancel(ctx)
	rt, err := uc.getRuntime(bg, s.RuntimeID)
	if err != nil {
		return nil, err
	}
	if !rt.Ready {
		return nil, ErrAgentRuntimeNotReady
	}
	driver := uc.driverOf(string(rt.Kind))
	launch, err := uc.runtimeLaunch(rt)
	if err != nil {
		return nil, err
	}
	var provider *gen.AgentProvider
	if s.ProviderID != "" {
		// 第三方接口要还是会话主人用得上的：删掉了、或者管理员不再共享给他了，就起不来
		if provider, err = uc.usableProvider(bg, s.UserID, s.ProviderID); err != nil {
			return nil, err
		}
	}
	env, byEnv, err := uc.providerEnv(driver, provider, s.Model, s.Effort)
	if err != nil {
		return nil, err
	}
	// 服务器上的登录：用户在模型表里给这个模型设的最长上下文、最长输出，启动时交给 Agent
	if provider == nil {
		m, err := uc.loginModel(bg, rt, s.Model)
		if err != nil {
			return nil, err
		}
		env = driver.LoginLimits(loginLimits(m))
	}
	// 代理：有就整个进程树在沙箱里，只经它联网（见 effectiveProxy）
	proxy, err := uc.effectiveProxy(rt, provider)
	if err != nil {
		return nil, err
	}
	sb, err := uc.sandboxOf(proxy)
	if err != nil {
		return nil, err
	}
	launch, env = sandboxed(driver, launch, env, sb)
	// 技能放进 Agent 的配置目录：这个目录是服务器上所有会话共用的，起进程前换成这个会话主人自己启用的技能
	if home := driver.Home(); home != "" {
		agentskill.Sync(home, uc.enabledSkills(bg, s.UserID))
	}
	servers, err := uc.sessionMcpServers(bg, s.UserID, s.McpIds)
	if err != nil {
		return nil, err
	}
	seq, rev, err := uc.repo.MaxAgentEvent(bg, s.ID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	l := &agentLive{uc: uc, id: s.ID, userID: s.UserID, runtimeID: s.RuntimeID, providerID: s.ProviderID, byEnv: byEnv, launch: launch}
	l.w = newAgentWriter(l)
	session, started, err := agentrun.Start(bg, agentrun.Config{
		Launch:      launch,
		Driver:      driver,
		Env:         env,
		Cwd:         s.Cwd,
		ExtraDirs:   uc.agentDirs(s.ID, s.ExtraDirs),
		TempDir:     uc.sessionDir(s.ID, "tmp"),
		MediaDir:    uc.sessionDir(s.ID, "uploads"),
		McpServers:  servers,
		ResumeID:    s.AcpSessionID,
		Mode:        s.Mode,
		Model:       s.Model,
		Effort:      s.Effort,
		ModelByEnv:  byEnv,
		Seq:         seq,
		Rev:         rev,
		State:       agentParse[agentrun.State](s.State),
		TitleCustom: s.TitleCustom,
		ContextUsed: s.ContextUsed,
		ContextSize: s.ContextSize,
		Sink:        l,
	})
	if err != nil {
		l.w.close()
		if errors.Is(err, agentrun.ErrAuthRequired) {
			if provider != nil {
				return nil, response.BadRequest("供应商「" + provider.Name + "」的接口不接受这个 API Key，到「模型供应商」里检查")
			}
			uc.markLoggedOut(bg, rt.ID)
			return nil, response.BadRequest(rt.Name + " 在服务器上还没有登录：在服务器上登录后，到「模型供应商」里点「检查」")
		}
		return nil, agentErr(err)
	}
	if !session.Alive() {
		l.w.close()
		return nil, response.BadRequest("Agent 进程刚启动就退出了，再试一次")
	}
	l.session = session
	l.w.update(func(u *gen.AgentSessionUpdateOne) {
		u.SetAcpSessionID(started.SessionID)
		if started.Mode != "" {
			u.SetMode(started.Mode)
		}
		if !byEnv && started.Model != "" {
			u.SetModel(started.Model).SetEffort(started.Effort)
		}
	})
	uc.refreshModes(bg, rt, session.Catalog())
	return l, nil
}

// refreshModes 权限模式的清单是 Agent 说了算：起进程时记下它这次给的。只用第三方接口、服务器上没登录时，
// 检查开不了会话，权限模式只能这样拿到。模型不在这里记：服务器上的登录的模型只在检查时同步进模型表。
func (uc *AgentUsecase) refreshModes(ctx context.Context, rt *gen.AgentRuntime, fresh agentrun.Catalog) {
	if len(fresh.Modes) == 0 {
		return
	}
	catalog := agentParse[agentRuntimeCatalog](rt.Catalog)
	catalog.Modes = fresh.Modes
	if catalog.DefaultMode == "" {
		catalog.DefaultMode = fresh.DefaultMode
	}
	if raw := agentJSON(catalog); raw != rt.Catalog {
		rt.Catalog = raw
		_ = uc.repo.SaveAgentRuntime(ctx, rt)
		uc.configChanged()
	}
}
