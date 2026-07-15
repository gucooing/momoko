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
)

// Manager 是任务运行时：持久化 + 派生非超时 ctx + 进度/事件订阅 + 开机注入。
type Manager struct {
	mu      sync.RWMutex
	store   Store
	base    context.Context
	cancel  context.CancelFunc
	live    map[string]*liveState
	factory map[string]Factory
	wg      sync.WaitGroup
}

// liveState 是一个在跑任务的内存态：当前视图 + 取消句柄 + 事件回放缓冲 + 订阅者。
type liveState struct {
	info     *Info
	cancel   context.CancelFunc
	events   []Event
	subs     map[uint64]chan Event
	nextID   uint64
	canceled bool
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

// bgCtx 返回一个不随 Stop 取消的 ctx，用于持久化写，保证终态也能落库。
func (m *Manager) bgCtx() context.Context { return context.WithoutCancel(m.base) }

// Register 注册某类型任务的重建工厂（开机/重试用）。
func (m *Manager) Register(typ string, f Factory) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.factory[typ] = f
}

// Submit 持久化并启动一个一次性/常驻任务，返回任务 id。
func (m *Manager) Submit(ctx context.Context, t Task) (string, error) {
	meta := t.Meta()
	if meta.ID == "" {
		meta.ID = uuid.NewString()
	}
	rec := newRecord(meta, t)
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
	// 保留已存在行的创建时间。
	if old, err := m.store.Get(m.bgCtx(), meta.ID); err == nil {
		rec.CreateTime = old.CreateTime
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
			now := time.Now()
			_ = m.store.SetStatus(m.bgCtx(), rec.ID, StatusFailed, "", "服务重启，任务已中断", &now)
		case ResumeRerun:
			m.mu.RLock()
			f, ok := m.factory[rec.Type]
			m.mu.RUnlock()
			if !ok {
				continue
			}
			t, ferr := f(ctx, rec)
			if ferr != nil {
				now := time.Now()
				_ = m.store.SetStatus(m.bgCtx(), rec.ID, StatusFailed, "", ferr.Error(), &now)
				continue
			}
			m.launch(t, rec)
		}
	}
	return nil
}

// launch 按种类启动任务；若已在跑则幂等跳过。
func (m *Manager) launch(t Task, rec *Record) {
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
		info:   infoFromRecord(rec),
		cancel: cancel,
		subs:   make(map[uint64]chan Event),
	}
	ls.info.Status = StatusRunning
	m.live[rec.ID] = ls
	m.mu.Unlock()

	rep := &reporter{m: m, id: rec.ID}
	_ = m.store.SetStatus(m.bgCtx(), rec.ID, StatusRunning, "", "", nil)

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
// （抽奖结算等依赖「定时器触发 + 墙钟条件」双重门槛）。
func (m *Manager) runScheduled(ctx context.Context, t Task, rec *Record, rep *reporter) {
	interval := time.Duration(rec.IntervalMS) * time.Millisecond
	if interval <= 0 {
		interval = defaultInterval
	}
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
			func() {
				defer func() { _ = recover() }()
				_ = t.Run(ctx, rep)
			}()
		}
	}
}

// Cancel 取消运行中的任务（尽力而为：本地复制等不接 ctx 的操作中途不可打断）。
func (m *Manager) Cancel(id string) bool {
	m.mu.Lock()
	ls, ok := m.live[id]
	if ok {
		ls.canceled = true
	}
	m.mu.Unlock()
	if !ok {
		return false
	}
	ls.cancel()
	return true
}

// Retry 重投一个终态任务（按工厂重建）。
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
		return nil, errors.New("任务类型未注册")
	}
	t, err := f(ctx, rec)
	if err != nil {
		return nil, err
	}
	rec.Status = StatusPending
	rec.Error = ""
	rec.Message = ""
	rec.Finished = 0
	rec.Results = nil
	rec.EndTime = nil
	if err := m.store.Upsert(m.bgCtx(), rec); err != nil {
		return nil, err
	}
	m.launch(t, rec)
	info, _ := m.Get(id)
	return info, nil
}

// Delete 删除一条终态任务行（运行中需先取消）。
func (m *Manager) Delete(id string) error {
	m.mu.RLock()
	_, live := m.live[id]
	m.mu.RUnlock()
	if live {
		return errors.New("任务运行中，请先取消")
	}
	return m.store.Delete(m.bgCtx(), id)
}

// Get 取单个任务视图（优先内存态）。
func (m *Manager) Get(id string) (*Info, bool) {
	m.mu.RLock()
	ls, ok := m.live[id]
	if ok {
		info := cloneInfo(ls.info)
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

// List 分页查询任务；运行中的任务用内存态覆盖以获得最新进度。
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
			infos = append(infos, cloneInfo(ls.info))
		} else {
			infos = append(infos, infoFromRecord(rec))
		}
	}
	return infos, total, nil
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

func (m *Manager) setProgress(id string, finished, total int64, message string) {
	m.mu.Lock()
	ls, ok := m.live[id]
	if ok {
		ls.info.Finished = finished
		ls.info.Total = total
		if message != "" {
			ls.info.Message = message
		}
	}
	m.mu.Unlock()
	if ok {
		_ = m.store.SetProgress(m.bgCtx(), id, finished, total, message)
	}
}

func (m *Manager) emit(id string, e Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ls, ok := m.live[id]
	if !ok {
		return
	}
	if e.Message != "" {
		ls.info.Message = e.Message
	}
	if e.Error != "" {
		ls.info.Error = e.Error
	}
	ls.events = append(ls.events, e)
	for _, ch := range ls.subs {
		select {
		case ch <- e:
		default:
		}
	}
}

func (m *Manager) appendResult(id string, res Result) {
	m.mu.Lock()
	ls, ok := m.live[id]
	var results []Result
	if ok {
		ls.info.Results = append(ls.info.Results, res)
		ls.info.Finished = int64(len(ls.info.Results))
		results = append([]Result(nil), ls.info.Results...)
	}
	m.mu.Unlock()
	if ok {
		_ = m.store.SaveResults(m.bgCtx(), id, results)
	}
}

func (m *Manager) checkpoint(id string, state any) error {
	b, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return m.store.SaveState(m.bgCtx(), id, b)
}

// finish 落终态：更新内存、关订阅、删 live、持久化。
func (m *Manager) finish(id string, s Status, errText string) {
	now := time.Now()
	var closeSubs []chan Event
	m.mu.Lock()
	ls, ok := m.live[id]
	if ok {
		ls.info.Status = s
		ls.info.EndTime = &now
		if errText != "" {
			ls.info.Error = errText
		}
		for _, ch := range ls.subs {
			closeSubs = append(closeSubs, ch)
		}
		delete(m.live, id)
	}
	m.mu.Unlock()
	for _, ch := range closeSubs {
		close(ch)
	}
	message := ""
	if s == StatusSuccess {
		message = "完成"
	}
	_ = m.store.SetStatus(m.bgCtx(), id, s, message, errText, &now)
}

// drop 仅移除内存态（不改 DB 状态），用于 app 停止时退出常驻/定时任务。
func (m *Manager) drop(id string) {
	m.mu.Lock()
	if ls, ok := m.live[id]; ok {
		for _, ch := range ls.subs {
			close(ch)
		}
		delete(m.live, id)
	}
	m.mu.Unlock()
}

// reporter 实现 Reporter，把上报转发给管理器。
type reporter struct {
	m  *Manager
	id string
}

func (r *reporter) SetProgress(finished, total int64, message string) {
	r.m.setProgress(r.id, finished, total, message)
}
func (r *reporter) Emit(e Event)            { r.m.emit(r.id, e) }
func (r *reporter) AppendResult(res Result) { r.m.appendResult(r.id, res) }
func (r *reporter) Checkpoint(state any) error {
	return r.m.checkpoint(r.id, state)
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
		UserID:     meta.UserID,
		Total:      meta.Total,
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

func infoFromRecord(rec *Record) *Info {
	return &Info{
		ID:         rec.ID,
		Type:       rec.Type,
		Kind:       rec.Kind,
		Title:      rec.Title,
		UserID:     rec.UserID,
		Status:     rec.Status,
		Total:      rec.Total,
		Finished:   rec.Finished,
		Message:    rec.Message,
		Error:      rec.Error,
		Results:    append([]Result(nil), rec.Results...),
		CreateTime: rec.CreateTime,
		EndTime:    rec.EndTime,
	}
}

func cloneInfo(in *Info) *Info {
	out := *in
	out.Results = append([]Result(nil), in.Results...)
	if in.EndTime != nil {
		t := *in.EndTime
		out.EndTime = &t
	}
	return &out
}

// MatchFilter 报告一条记录是否命中过滤（供数据层无法表达的内存态复用；当前数据层已实现过滤）。
func MatchFilter(rec *Record, f Filter) bool {
	if f.UserID != "" && rec.UserID != f.UserID {
		return false
	}
	if f.TypePrefix != "" && !strings.HasPrefix(rec.Type, f.TypePrefix) {
		return false
	}
	if f.Status != "" && rec.Status != f.Status {
		return false
	}
	if f.Kind != "" && rec.Kind != f.Kind {
		return false
	}
	if f.Keywords != "" && !strings.Contains(rec.Title, f.Keywords) && !strings.Contains(rec.Type, f.Keywords) {
		return false
	}
	return true
}
