package port_forward

import (
	"context"
	"errors"
	"sync"
	"time"
)

const (
	// RateInterval 实时速率的采样间隔。
	RateInterval = 2 * time.Second
	// rateHistorySize 每个实例保留的速率采样点数（约 1 分钟）。
	rateHistorySize = 30
)

var (
	ExampleREPEAT  = errors.New("端口转发已启动")
	NotExample     = errors.New("端口转发未启用")
	NOTProtocol    = errors.New("暂未实现该协议")
	NETListenError = errors.New("端口被占用或不可用")
)

type Protocol int

const (
	TCP Protocol = iota
	UDP
)

type Option struct {
	ID            string // 唯一id
	Protocol      Protocol
	ListenAddress string // 监听地址
	ListenPort    int    // 监听端口
	TargetAddress string // 目标地址
	TargetPort    int    // 目标端口
}

type PortForward interface {
	Option() *Option
	Start() error
	Stop()
	Stats() *Stats
}

// Rate 是某实例的实时速率（字节/秒）与最近的总速率采样。
type Rate struct {
	In      int64
	Out     int64
	History []int64
}

type rateState struct {
	stats   *Stats
	lastIn  int64
	lastOut int64
	lastAt  time.Time
	rate    Rate
}

// 实例管理
type Manager struct {
	sync     sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	examples map[string]PortForward // 全部已启动的实例
	errors   map[string]string      // 启动失败或运行中退出的实例 → 原因
	rates    map[string]*rateState
}

func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		sync:     sync.RWMutex{},
		ctx:      ctx,
		cancel:   cancel,
		examples: make(map[string]PortForward),
		errors:   make(map[string]string),
		rates:    make(map[string]*rateState),
	}
	go m.rateLoop()
	return m
}

// rateLoop 每 RateInterval 采样一次全部实例的累计流量，换算为实时速率。
func (m *Manager) rateLoop() {
	ticker := time.NewTicker(RateInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case now := <-ticker.C:
			m.sync.Lock()
			for id, example := range m.examples {
				stats := example.Stats()
				snap := stats.Snapshot()
				st := m.rates[id]
				if st == nil || st.stats != stats {
					m.rates[id] = &rateState{stats: stats, lastIn: snap.BytesIn, lastOut: snap.BytesOut, lastAt: now}
					continue
				}
				dt := now.Sub(st.lastAt).Seconds()
				if dt <= 0 {
					continue
				}
				in := int64(float64(snap.BytesIn-st.lastIn) / dt)
				out := int64(float64(snap.BytesOut-st.lastOut) / dt)
				st.lastIn, st.lastOut, st.lastAt = snap.BytesIn, snap.BytesOut, now
				st.rate.In, st.rate.Out = max(in, 0), max(out, 0)
				st.rate.History = append(st.rate.History, st.rate.In+st.rate.Out)
				if len(st.rate.History) > rateHistorySize {
					st.rate.History = st.rate.History[len(st.rate.History)-rateHistorySize:]
				}
			}
			for id := range m.rates {
				if _, ok := m.examples[id]; !ok {
					delete(m.rates, id)
				}
			}
			m.sync.Unlock()
		}
	}
}

// Rate 返回实例的实时速率；未运行时 ok 为 false。
func (m *Manager) Rate(id string) (Rate, bool) {
	m.sync.RLock()
	defer m.sync.RUnlock()
	st, ok := m.rates[id]
	if !ok {
		return Rate{}, false
	}
	return Rate{In: st.rate.In, Out: st.rate.Out, History: append([]int64(nil), st.rate.History...)}, true
}

// Error 返回实例最近一次启动失败或异常退出的原因（成功启动或卸载后清空）。
func (m *Manager) Error(id string) string {
	m.sync.RLock()
	defer m.sync.RUnlock()
	return m.errors[id]
}

// Connections 返回实例的活动连接；未运行时为空。
func (m *Manager) Connections(id string) []ConnSnapshot {
	m.sync.RLock()
	example, ok := m.examples[id]
	m.sync.RUnlock()
	if !ok {
		return nil
	}
	return example.Stats().Connections()
}

// CloseConnections 断开实例的指定活动连接，ids 为空时断开全部。
func (m *Manager) CloseConnections(id string, ids []string) int {
	m.sync.RLock()
	example, ok := m.examples[id]
	m.sync.RUnlock()
	if !ok {
		return 0
	}
	return example.Stats().CloseConnections(ids)
}

// 注册端口转发
func (m *Manager) RegisterExample(opt *Option) error {
	m.sync.RLock()
	if _, ok := m.examples[opt.ID]; ok {
		m.sync.RUnlock()
		return ExampleREPEAT
	}
	m.sync.RUnlock()

	var example PortForward
	var err error
	switch opt.Protocol {
	case TCP:
		example, err = NewTCPPortForward(opt, m.ctx)
	case UDP:
		example, err = NewUDPPortForward(opt, m.ctx)
	default:
		return NOTProtocol
	}
	if err != nil {
		m.sync.Lock()
		m.errors[opt.ID] = err.Error()
		m.sync.Unlock()
		return err
	}
	m.sync.Lock()
	m.examples[opt.ID] = example
	delete(m.errors, opt.ID)
	m.sync.Unlock()
	go m.Run(example)
	return nil
}

// 卸载端口转发
func (m *Manager) UnRegisterExample(id string) {
	m.sync.Lock()
	delete(m.errors, id)
	example, ok := m.examples[id]
	if !ok {
		m.sync.Unlock()
		return
	}
	delete(m.examples, id)
	m.sync.Unlock()
	example.Stop()
}

// 重载端口转发
func (m *Manager) Retry(opt *Option) error {
	m.sync.Lock()
	example, ok := m.examples[opt.ID]
	if ok {
		example.Stop()
		delete(m.examples, opt.ID)
	}
	m.sync.Unlock()

	return m.RegisterExample(opt)
}

func (m *Manager) Run(example PortForward) {
	select {
	case <-m.ctx.Done():
		return
	default:
	}

	err := example.Start()
	if err != nil {
		// 监听意外中断：移出运行列表并记录原因，状态显示为异常。
		id := example.Option().ID
		m.sync.Lock()
		if m.examples[id] == example {
			delete(m.examples, id)
			m.errors[id] = err.Error()
		}
		m.sync.Unlock()
		example.Stop()
		return
	}
}

func (m *Manager) Running(id string) bool {
	m.sync.RLock()
	defer m.sync.RUnlock()
	_, ok := m.examples[id]
	return ok
}

func (m *Manager) Stop() {
	m.cancel()
}

// Stats 返回指定实例的实时统计快照；实例未运行时 ok 为 false。
func (m *Manager) Stats(id string) (Snapshot, bool) {
	m.sync.RLock()
	example, ok := m.examples[id]
	m.sync.RUnlock()
	if !ok {
		return Snapshot{}, false
	}
	return example.Stats().Snapshot(), true
}

// SnapshotAll 返回当前所有运行中实例的实时统计快照，按实例 id 索引。
// 供上层周期性采样并持久化。
func (m *Manager) SnapshotAll() map[string]Snapshot {
	m.sync.RLock()
	defer m.sync.RUnlock()

	out := make(map[string]Snapshot, len(m.examples))
	for id, example := range m.examples {
		out[id] = example.Stats().Snapshot()
	}
	return out
}
