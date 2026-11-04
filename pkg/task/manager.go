package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	subscriberBuffer = 4096
	defaultInterval  = time.Hour
	// snapshotInterval 是运行期进度/步骤/日志落库的最小间隔（高频上报只写内存，按此节流入库）。
	snapshotInterval = time.Second
)

// ErrTaskRunning 表示任务仍在运行，不能删除。
var ErrTaskRunning = errors.New("任务运行中，请先取消")

// Manager 是任务运行时：持久化 + 派生非超时 ctx + 来源记录 + 进度/步骤/日志上报与订阅 + 开机注入。
type Manager struct {
	mu      sync.RWMutex
	store   Store
	base    context.Context
	cancel  context.CancelFunc
	live    map[string]*liveState
	factory map[string]Factory
	origin  OriginResolver
	wg      sync.WaitGroup
}

// liveState 是一个在跑任务的内存态：当前视图（含步骤与日志）+ 取消句柄 + 事件回放缓冲 + 订阅者 + 落库节流状态。
type liveState struct {
	info       *Info
	cancel     context.CancelFunc
	events     []Event
	subs       map[uint64]chan Event
	nextID     uint64
	canceled   bool
	canceledBy string

	// 落库节流：saveMu 串行化本任务的快照/终态写入，done 之后不再写快照。
	saveMu     sync.Mutex
	done       bool
	dirty      bool
	timerArmed bool
	lastSave   time.Time
}

// New 构造管理器；base ctx 为 app 生命周期，Stop 时取消以中止全部任务。
func New(store Store) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		store:   store,
		base:    ctx,
		cancel:  cancel,
		live:    make(map[string]*liveState),
		factory: make(map[string]Factory),
	}
}

// SetOriginResolver 注入「从请求 ctx 解析任务发起方」的逻辑（业务层在构造时调用一次）。
func (m *Manager) SetOriginResolver(f OriginResolver) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.origin = f
}

// bgCtx 返回一个不随 Stop 取消的 ctx，用于持久化写，保证终态也能落库。
func (m *Manager) bgCtx() context.Context { return context.WithoutCancel(m.base) }

// Register 注册某类型任务的重建工厂（开机/重试用）。
func (m *Manager) Register(typ string, f Factory) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.factory[typ] = f
}

// Submit 持久化并启动一个一次性/常驻任务，返回任务 id；发起方从 ctx 解析。
func (m *Manager) Submit(ctx context.Context, t Task) (string, error) {
	meta := t.Meta()
	if meta.ID == "" {
		meta.ID = uuid.NewString()
	}
	rec := newRecord(meta, t)
	m.applyOrigin(ctx, rec)
	if err := m.store.Upsert(m.bgCtx(), rec); err != nil {
		return "", err
	}
	m.launch(t, rec)
	return meta.ID, nil
}

// EnsureSingleton 幂等地确保一个单例任务（定时/常驻）存在并在运行：upsert 行 + 未在跑则启动。
func (m *Manager) EnsureSingleton(ctx context.Context, t Task) error {
	meta := t.Meta()
	if meta.ID == "" {
		meta.ID = meta.Type
	}
	rec := newRecord(meta, t)
	rec.Trigger = TriggerSchedule
	if meta.Kind == KindDaemon {
		rec.Trigger = TriggerSystem
	}
	// 保留已存在行的创建时间与历史日志。
	if old, err := m.store.Get(m.bgCtx(), meta.ID); err == nil {
		rec.CreateTime = old.CreateTime
		rec.Logs = old.Logs
	}
	if err := m.store.Upsert(m.bgCtx(), rec); err != nil {
		return err
	}
	m.launch(t, rec)
	return nil
}

// Resume 开机注入：把本域（types）未完成的一次性任务按策略重跑或标记失败。always 单例由 EnsureSingleton 维护。
func (m *Manager) Resume(ctx context.Context, types ...string) error {
	recs, err := m.store.LoadResumable(m.bgCtx())
	if err != nil {
		return err
	}
	want := make(map[string]struct{}, len(types))
	for _, t := range types {
		want[t] = struct{}{}
	}
	for _, rec := range recs {
		if _, ok := want[rec.Type]; !ok {
			continue
		}
		switch rec.Resume {
		case ResumeNone:
			m.failRecord(rec, "服务重启，任务已中断")
		case ResumeRerun:
			m.mu.RLock()
			f, ok := m.factory[rec.Type]
			m.mu.RUnlock()
			if !ok {
				continue
			}
			t, ferr := f(ctx, rec)
			if ferr != nil {
				m.failRecord(rec, ferr.Error())
				continue
			}
			resetRun(rec, t.Meta())
			rec.Logs = append(rec.Logs, LogEntry{Time: time.Now(), Level: LogWarn, Message: "服务重启，任务重新执行"})
			if err := m.store.Upsert(m.bgCtx(), rec); err != nil {
				continue
			}
			m.launch(t, rec)
		}
	}
	return nil
}

// failRecord 把一条未在跑的历史任务行直接落为失败终态。
func (m *Manager) failRecord(rec *Record, reason string) {
	now := time.Now()
	steps := finishSteps(rec.Steps, StatusFailed, now)
	logs := appendLog(rec.Logs, LogEntry{Time: now, Level: LogError, Message: reason})
	_ = m.store.Finish(m.bgCtx(), rec.ID, StatusFailed, reason, "", now, Snapshot{
		Total: rec.Total, Finished: rec.Finished, Message: rec.Message, Steps: steps, Logs: logs,
	})
}

// launch 按种类启动任务；若已在跑则幂等跳过。
func (m *Manager) launch(t Task, rec *Record) {
	now := time.Now()
	m.mu.Lock()
	if _, ok := m.live[rec.ID]; ok {
		m.mu.Unlock()
		return
	}
	var runCtx context.Context
	var cancel context.CancelFunc
	if rec.Kind == KindOneShot && rec.TimeoutMS > 0 {
		runCtx, cancel = context.WithTimeout(m.base, time.Duration(rec.TimeoutMS)*time.Millisecond)
	} else {
		runCtx, cancel = context.WithCancel(m.base)
	}
	ls := &liveState{
		info:     infoFromRecord(rec),
		cancel:   cancel,
		subs:     make(map[uint64]chan Event),
		lastSave: now,
	}
	// 定时任务在两次触发之间处于「等待」，开始时间保留上一次执行；其余任务立即进入运行。
	scheduled := rec.Kind == KindScheduled
	if scheduled {
		next := now.Add(scheduleInterval(rec))
		ls.info.Status = StatusPending
		ls.info.NextRunTime = &next
	} else {
		ls.info.Status = StatusRunning
		ls.info.StartTime = &now
	}
	m.live[rec.ID] = ls
	m.mu.Unlock()

	rep := &reporter{m: m, id: rec.ID, ls: ls}
	if scheduled {
		_ = m.store.SetStatus(m.bgCtx(), rec.ID, StatusPending)
	} else {
		_ = m.store.SetRunning(m.bgCtx(), rec.ID, now)
	}

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer cancel()
		if rec.Kind == KindScheduled {
			m.runScheduled(runCtx, t, rec, rep)
			return
		}
		m.runOnce(runCtx, t, rec, rep)
	}()
}

// runOnce 执行一次性/常驻任务并落终态。
func (m *Manager) runOnce(ctx context.Context, t Task, rec *Record, rep *reporter) {
	defer func() {
		if r := recover(); r != nil {
			m.finish(rec.ID, StatusFailed, fmt.Sprintf("panic: %v", r))
		}
	}()
	err := t.Run(ctx, rep)
	switch {
	case err == nil:
		m.finish(rec.ID, StatusSuccess, "")
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		m.finish(rec.ID, StatusCanceled, err.Error())
	default:
		m.finish(rec.ID, StatusFailed, err.Error())
	}
}

// runScheduled 按周期触发任务，直到取消/停止。
// 不在启动时立即 Run：重启本身不做业务动作，等 Interval 真正到点再执行
// （依赖「定时器触发 + 墙钟条件」双重门槛的任务）。
func (m *Manager) runScheduled(ctx context.Context, t Task, rec *Record, rep *reporter) {
	interval := scheduleInterval(rec)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			// 用户取消才标记终态；app 停止则留待下次开机由 EnsureSingleton 重启。
			m.mu.RLock()
			ls, ok := m.live[rec.ID]
			canceled := ok && ls.canceled
			m.mu.RUnlock()
			if canceled {
				m.finish(rec.ID, StatusCanceled, "")
			} else {
				m.drop(rec.ID)
			}
			return
		case <-ticker.C:
			m.setScheduleState(rep.ls, true, interval)
			func() {
				defer func() {
					if r := recover(); r != nil {
						rep.Log(LogError, fmt.Sprintf("panic: %v", r))
					}
				}()
				if err := t.Run(ctx, rep); err != nil && ctx.Err() == nil {
					rep.Log(LogError, err.Error())
				}
			}()
			m.setScheduleState(rep.ls, false, interval)
		}
	}
}

// scheduleInterval 返回定时任务的触发周期（未设置时用默认值）。
func scheduleInterval(rec *Record) time.Duration {
	if d := time.Duration(rec.IntervalMS) * time.Millisecond; d > 0 {
		return d
	}
	return defaultInterval
}

// setScheduleState 切换定时任务的一次触发：running=true 进入运行并记开始时间，false 回到等待并算出下次触发时间。
func (m *Manager) setScheduleState(ls *liveState, running bool, interval time.Duration) {
	now := time.Now()
	m.mu.Lock()
	if ls.done {
		m.mu.Unlock()
		return
	}
	id := ls.info.ID
	if running {
		ls.info.Status = StatusRunning
		ls.info.StartTime = &now
		ls.info.NextRunTime = nil
	} else {
		next := now.Add(interval)
		ls.info.Status = StatusPending
		ls.info.NextRunTime = &next
	}
	m.mu.Unlock()
	if running {
		_ = m.store.SetRunning(m.bgCtx(), id, now)
	} else {
		_ = m.store.SetStatus(m.bgCtx(), id, StatusPending)
		m.flush(ls)
	}
}

// Cancel 取消运行中的任务并记录取消人（尽力而为：本地复制等不接 ctx 的操作中途不可打断）。
func (m *Manager) Cancel(id, by string) bool {
	m.mu.Lock()
	ls, ok := m.live[id]
	if ok {
		ls.canceled = true
		ls.canceledBy = by
	}
	m.mu.Unlock()
	if !ok {
		return false
	}
	ls.cancel()
	return true
}

// Retry 重投一个终态任务（按工厂重建），步骤、日志、结果与取消人全部重置。
func (m *Manager) Retry(ctx context.Context, id string) (*Info, error) {
	rec, err := m.store.Get(m.bgCtx(), id)
	if err != nil {
		return nil, err
	}
	if !rec.Status.Terminal() {
		return nil, errors.New("任务未结束，无法重试")
	}
	m.mu.RLock()
	f, ok := m.factory[rec.Type]
	m.mu.RUnlock()
	if !ok {
		return nil, errors.New("该类型任务不支持重试")
	}
	t, err := f(ctx, rec)
	if err != nil {
		return nil, err
	}
	resetRun(rec, t.Meta())
	rec.Logs = nil
	if err := m.store.Upsert(m.bgCtx(), rec); err != nil {
		return nil, err
	}
	m.launch(t, rec)
	info, _ := m.Get(id)
	return info, nil
}

// CanRetry 报告该类型任务是否注册了重建工厂（可重试）。
func (m *Manager) CanRetry(typ string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.factory[typ]
	return ok
}

// Delete 删除若干终态任务行；运行中的任务跳过并在 running 中返回。
func (m *Manager) Delete(ids []string) (deleted int64, running []string, err error) {
	m.mu.RLock()
	targets := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, live := m.live[id]; live {
			running = append(running, id)
			continue
		}
		targets = append(targets, id)
	}
	m.mu.RUnlock()
	if len(targets) == 0 {
		return 0, running, nil
	}
	deleted, err = m.store.Delete(m.bgCtx(), targets)
	return deleted, running, err
}

// Get 取单个任务视图（优先内存态），带完整日志。
func (m *Manager) Get(id string) (*Info, bool) {
	m.mu.RLock()
	ls, ok := m.live[id]
	if ok {
		info := cloneInfo(ls.info, true)
		m.mu.RUnlock()
		return info, true
	}
	m.mu.RUnlock()
	rec, err := m.store.Get(m.bgCtx(), id)
	if err != nil {
		return nil, false
	}
	return infoFromRecord(rec), true
}

// List 分页查询任务；运行中的任务用内存态覆盖以获得最新进度。列表不带日志（只带行数）。
func (m *Manager) List(ctx context.Context, f Filter, page, pageSize int64) ([]*Info, int64, error) {
	recs, total, err := m.store.List(ctx, f, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	infos := make([]*Info, 0, len(recs))
	for _, rec := range recs {
		if ls, ok := m.live[rec.ID]; ok {
			infos = append(infos, cloneInfo(ls.info, false))
		} else {
			info := infoFromRecord(rec)
			info.Logs = nil
			infos = append(infos, info)
		}
	}
	return infos, total, nil
}

// CountByStatus 按状态统计任务数（忽略 f.Status），供列表页状态 chips 使用。
func (m *Manager) CountByStatus(ctx context.Context, f Filter) (map[Status]int64, error) {
	f.Status = ""
	return m.store.CountByStatus(ctx, f)
}

// CountByType 统计全部任务里每个类型的数量，供类型筛选使用。
func (m *Manager) CountByType(ctx context.Context) (map[string]int64, error) {
	return m.store.CountByType(ctx)
}

// Clear 批量删除终态任务行；statuses 为空时清理全部终态，非终态会被忽略（运行中的任务不能删）。
func (m *Manager) Clear(ctx context.Context, statuses ...Status) (int64, error) {
	if len(statuses) == 0 {
		statuses = []Status{StatusSuccess, StatusFailed, StatusCanceled}
	}
	terminal := make([]Status, 0, len(statuses))
	for _, s := range statuses {
		if s.Terminal() {
			terminal = append(terminal, s)
		}
	}
	if len(terminal) == 0 {
		return 0, nil
	}
	return m.store.DeleteByStatus(ctx, terminal)
}

// Subscribe 订阅某运行中任务的实时事件：先回放已缓冲事件，再流式推送；任务终态时关闭。
func (m *Manager) Subscribe(id string) (<-chan Event, func(), bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ls, ok := m.live[id]
	if !ok {
		return nil, nil, false
	}
	ch := make(chan Event, len(ls.events)+subscriberBuffer)
	for _, e := range ls.events {
		ch <- e
	}
	ls.nextID++
	subID := ls.nextID
	ls.subs[subID] = ch
	cancel := func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if ls, ok := m.live[id]; ok {
			if _, exists := ls.subs[subID]; exists {
				delete(ls.subs, subID)
				close(ch)
			}
		}
	}
	return ch, cancel, true
}

// Stop 取消 base ctx 并等待全部任务退出（接进 wire cleanup）。
func (m *Manager) Stop() {
	m.cancel()
	m.wg.Wait()
}

// ---- 内部上报 ----

// mutate 在持锁下修改内存态并按节流决定是否立即落库快照。
func (m *Manager) mutate(ls *liveState, f func(info *Info)) {
	m.mu.Lock()
	if ls.done {
		m.mu.Unlock()
		return
	}
	f(ls.info)
	ls.dirty = true
	saveNow := false
	if wait := snapshotInterval - time.Since(ls.lastSave); wait <= 0 {
		saveNow = true
	} else if !ls.timerArmed {
		ls.timerArmed = true
		time.AfterFunc(wait, func() { m.flush(ls) })
	}
	m.mu.Unlock()
	if saveNow {
		m.flush(ls)
	}
}

// flush 把脏的内存快照写库；终态之后不再写。
func (m *Manager) flush(ls *liveState) {
	ls.saveMu.Lock()
	defer ls.saveMu.Unlock()
	m.mu.Lock()
	ls.timerArmed = false
	if ls.done || !ls.dirty {
		m.mu.Unlock()
		return
	}
	snap := snapshotOf(ls.info)
	id := ls.info.ID
	ls.dirty = false
	ls.lastSave = time.Now()
	m.mu.Unlock()
	_ = m.store.SaveSnapshot(m.bgCtx(), id, snap)
}

func (m *Manager) appendResult(ls *liveState, res Result) {
	m.mu.Lock()
	if ls.done {
		m.mu.Unlock()
		return
	}
	ls.info.Results = append(ls.info.Results, res)
	ls.info.Finished = int64(len(ls.info.Results))
	results := append([]Result(nil), ls.info.Results...)
	id := ls.info.ID
	m.mu.Unlock()
	_ = m.store.SaveResults(m.bgCtx(), id, results)
	m.mutate(ls, func(*Info) {})
}

func (m *Manager) emit(ls *liveState, e Event) {
	if e.Time.IsZero() {
		e.Time = time.Now()
	}
	m.mutate(ls, func(info *Info) {
		if e.Quiet {
			return
		}
		if msg := strings.TrimRight(e.Message, "\r\n"); msg != "" {
			info.Logs = appendLog(info.Logs, LogEntry{Time: e.Time, Level: LogInfo, Message: msg})
		}
		if e.Error != "" {
			info.Logs = appendLog(info.Logs, LogEntry{Time: e.Time, Level: LogError, Message: e.Error})
		}
		info.LogCount = len(info.Logs)
	})
	m.mu.Lock()
	defer m.mu.Unlock()
	if ls.done {
		return
	}
	ls.events = append(ls.events, e)
	for _, ch := range ls.subs {
		select {
		case ch <- e:
		default:
		}
	}
}

func (m *Manager) checkpoint(id string, state any) error {
	b, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return m.store.SaveState(m.bgCtx(), id, b)
}

// finish 落终态：收尾步骤、补错误日志、关订阅、删 live、持久化。
func (m *Manager) finish(id string, s Status, errText string) {
	now := time.Now()
	m.mu.RLock()
	ls, ok := m.live[id]
	m.mu.RUnlock()
	if !ok {
		return
	}
	ls.saveMu.Lock()
	defer ls.saveMu.Unlock()

	var closeSubs []chan Event
	m.mu.Lock()
	ls.done = true
	info := ls.info
	info.Status = s
	info.EndTime = &now
	if errText != "" && !(s == StatusCanceled && ls.canceled) {
		info.Error = errText
		info.Logs = appendLog(info.Logs, LogEntry{Time: now, Level: LogError, Message: errText})
	}
	if s == StatusCanceled {
		info.CanceledBy = ls.canceledBy
		if ls.canceled {
			info.Error = ""
		}
	}
	info.Steps = finishSteps(info.Steps, s, now)
	info.LogCount = len(info.Logs)
	snap := snapshotOf(info)
	canceledBy := info.CanceledBy
	finalErr := info.Error
	for _, ch := range ls.subs {
		closeSubs = append(closeSubs, ch)
	}
	delete(m.live, id)
	m.mu.Unlock()
	for _, ch := range closeSubs {
		close(ch)
	}
	_ = m.store.Finish(m.bgCtx(), id, s, finalErr, canceledBy, now, snap)
}

// drop 仅移除内存态并落一次快照（不改 DB 状态），用于 app 停止时退出常驻/定时任务。
func (m *Manager) drop(id string) {
	m.mu.RLock()
	ls, ok := m.live[id]
	m.mu.RUnlock()
	if !ok {
		return
	}
	ls.saveMu.Lock()
	defer ls.saveMu.Unlock()
	m.mu.Lock()
	ls.done = true
	snap := snapshotOf(ls.info)
	for _, ch := range ls.subs {
		close(ch)
	}
	delete(m.live, id)
	m.mu.Unlock()
	_ = m.store.SaveSnapshot(m.bgCtx(), id, snap)
}

// applyOrigin 用 ctx 解析出的发起方补全任务记录。
func (m *Manager) applyOrigin(ctx context.Context, rec *Record) {
	m.mu.RLock()
	resolve := m.origin
	m.mu.RUnlock()
	var o Origin
	if resolve != nil && ctx != nil {
		o = resolve(ctx)
	}
	if rec.UserID == "" {
		rec.UserID = o.UserID
	}
	rec.Operator = o.Operator
	rec.Trigger = o.Trigger
	if rec.Trigger == "" {
		if rec.UserID != "" {
			rec.Trigger = TriggerManual
		} else {
			rec.Trigger = TriggerSystem
		}
	}
}

// reporter 实现 Reporter，把上报转发给管理器。
type reporter struct {
	m  *Manager
	id string
	ls *liveState
}

func (r *reporter) SetProgress(finished, total int64, message string) {
	r.m.mutate(r.ls, func(info *Info) {
		info.Finished = finished
		info.Total = total
		if message != "" {
			info.Message = message
		}
	})
}

func (r *reporter) Step(title, detail string) {
	now := time.Now()
	r.m.mutate(r.ls, func(info *Info) {
		info.Steps = advanceStep(info.Steps, title, detail, now)
		if detail != "" {
			info.Message = title + " · " + detail
		} else {
			info.Message = title
		}
	})
}

func (r *reporter) Log(level LogLevel, message string) {
	if message == "" {
		return
	}
	if level == "" {
		level = LogInfo
	}
	now := time.Now()
	r.m.mutate(r.ls, func(info *Info) {
		info.Logs = appendLog(info.Logs, LogEntry{Time: now, Level: level, Message: message})
		info.LogCount = len(info.Logs)
	})
}

func (r *reporter) Emit(e Event)            { r.m.emit(r.ls, e) }
func (r *reporter) AppendResult(res Result) { r.m.appendResult(r.ls, res) }
func (r *reporter) Checkpoint(state any) error {
	return r.m.checkpoint(r.id, state)
}

// ---- 步骤与日志 ----

// waitingSteps 把预定义步骤标题展开为等待态步骤。
func waitingSteps(titles []string) []Step {
	if len(titles) == 0 {
		return nil
	}
	steps := make([]Step, 0, len(titles))
	for _, t := range titles {
		steps = append(steps, Step{Title: t, State: StepWaiting})
	}
	return steps
}

// advanceStep 进入名为 title 的步骤：当前进行中的同名步骤只更新 detail；
// 否则上一步记为成功，并推进到后续第一个同名的等待步骤（中间未执行的等待步骤记为跳过），找不到则追加。
func advanceStep(steps []Step, title, detail string, now time.Time) []Step {
	cur := -1
	for i := len(steps) - 1; i >= 0; i-- {
		if steps[i].State == StepRunning {
			cur = i
			break
		}
	}
	if cur >= 0 && steps[cur].Title == title {
		steps[cur].Detail = detail
		return steps
	}
	if cur >= 0 {
		end := now
		steps[cur].State = StepSuccess
		steps[cur].EndTime = &end
	}
	start := now
	for i := cur + 1; i < len(steps); i++ {
		if steps[i].State == StepWaiting && steps[i].Title == title {
			for j := cur + 1; j < i; j++ {
				if steps[j].State == StepWaiting {
					steps[j].State = StepSkipped
				}
			}
			steps[i].State = StepRunning
			steps[i].Detail = detail
			steps[i].StartTime = &start
			return steps
		}
	}
	return append(steps, Step{Title: title, State: StepRunning, Detail: detail, StartTime: &start})
}

// finishSteps 按终态收尾步骤：成功时全部记为成功；失败/取消时进行中的记为失败、等待中的记为跳过。
func finishSteps(steps []Step, s Status, now time.Time) []Step {
	for i := range steps {
		switch steps[i].State {
		case StepRunning:
			end := now
			steps[i].EndTime = &end
			if s == StatusSuccess {
				steps[i].State = StepSuccess
			} else {
				steps[i].State = StepFailed
			}
		case StepWaiting:
			if s == StatusSuccess {
				steps[i].State = StepSuccess
			} else {
				steps[i].State = StepSkipped
			}
		}
	}
	return steps
}

// appendLog 追加一行日志：与上一行内容、级别都相同则只刷新时间；超出上限丢弃最早的行。
func appendLog(logs []LogEntry, e LogEntry) []LogEntry {
	if n := len(logs); n > 0 {
		last := &logs[n-1]
		if last.Message == e.Message && last.Level == e.Level {
			last.Time = e.Time
			return logs
		}
	}
	logs = append(logs, e)
	if over := len(logs) - MaxLogEntries; over > 0 {
		logs = append([]LogEntry(nil), logs[over:]...)
	}
	return logs
}

// ---- 映射helpers ----

func newRecord(meta Meta, t Task) *Record {
	rec := &Record{
		ID:         meta.ID,
		Type:       meta.Type,
		Kind:       meta.Kind,
		Status:     StatusPending,
		Resume:     meta.Resume,
		Title:      meta.Title,
		Target:     meta.Target,
		Source:     meta.Source,
		UserID:     meta.UserID,
		Total:      meta.Total,
		Steps:      waitingSteps(meta.Steps),
		IntervalMS: meta.Interval.Milliseconds(),
		TimeoutMS:  meta.Timeout.Milliseconds(),
		CreateTime: time.Now(),
	}
	if payload := t.Payload(); payload != nil {
		if b, err := json.Marshal(payload); err == nil {
			rec.Payload = b
		}
	}
	return rec
}

// resetRun 把一条记录重置为待执行（重试 / 开机重跑），步骤按新构造任务的预定义步骤重建。
func resetRun(rec *Record, meta Meta) {
	rec.Status = StatusPending
	rec.Error = ""
	rec.Message = ""
	rec.Finished = 0
	if meta.Total > 0 {
		rec.Total = meta.Total
	}
	rec.Results = nil
	rec.Steps = waitingSteps(meta.Steps)
	rec.CanceledBy = ""
	rec.StartTime = nil
	rec.EndTime = nil
}

func snapshotOf(info *Info) Snapshot {
	return Snapshot{
		Total:    info.Total,
		Finished: info.Finished,
		Message:  info.Message,
		Steps:    append([]Step(nil), info.Steps...),
		Logs:     append([]LogEntry(nil), info.Logs...),
	}
}

func infoFromRecord(rec *Record) *Info {
	return &Info{
		ID:         rec.ID,
		Type:       rec.Type,
		Kind:       rec.Kind,
		Title:      rec.Title,
		Target:     rec.Target,
		Source:     rec.Source,
		Trigger:    rec.Trigger,
		UserID:     rec.UserID,
		Operator:   rec.Operator,
		Status:     rec.Status,
		Total:      rec.Total,
		Finished:   rec.Finished,
		Message:    rec.Message,
		Error:      rec.Error,
		Results:    append([]Result(nil), rec.Results...),
		Steps:      append([]Step(nil), rec.Steps...),
		Logs:       append([]LogEntry(nil), rec.Logs...),
		LogCount:   len(rec.Logs),
		Interval:   time.Duration(rec.IntervalMS) * time.Millisecond,
		CanceledBy: rec.CanceledBy,
		CreateTime: rec.CreateTime,
		StartTime:  rec.StartTime,
		EndTime:    rec.EndTime,
	}
}

func cloneInfo(in *Info, withLogs bool) *Info {
	out := *in
	out.Results = append([]Result(nil), in.Results...)
	out.Steps = append([]Step(nil), in.Steps...)
	out.LogCount = len(in.Logs)
	if withLogs {
		out.Logs = append([]LogEntry(nil), in.Logs...)
	} else {
		out.Logs = nil
	}
	if in.StartTime != nil {
		t := *in.StartTime
		out.StartTime = &t
	}
	if in.EndTime != nil {
		t := *in.EndTime
		out.EndTime = &t
	}
	if in.NextRunTime != nil {
		t := *in.NextRunTime
		out.NextRunTime = &t
	}
	return &out
}
