package agentrun

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrPoolFull 同时运行的进程到了上限，而且都在忙。
var ErrPoolFull = errors.New("同时运行的 Agent 进程已经到上限，等别的会话空闲或结束")

// Worker 进程池里的一个成员（调用方在 Session 外面包一层自己的东西）。
type Worker interface {
	Busy() bool
	ActiveAt() time.Time
	Shutdown()
	// Alive 进程还在。进程刚起就退了、或者退出时还没登记进池子，池子靠它认出不能再用的成员。
	Alive() bool
}

// Pool 有进程在跑的会话：同一个会话同时只启动一次，进程数有上限（到上限时结束空闲最久的那个），
// 空闲太久的自动结束（下次发消息时再接回来）。回收的协程只在池里有进程时跑。
type Pool[W Worker] struct {
	mu       sync.Mutex
	live     map[string]W
	starting map[string]chan struct{}
	limit    int
	idle     time.Duration
	reaping  bool
	closed   bool
}

// NewPool limit 是进程数上限（至少 1），idle 是空闲多久后结束（0 = 不自动结束）。
func NewPool[W Worker](limit int, idle time.Duration) *Pool[W] {
	p := &Pool[W]{live: map[string]W{}, starting: map[string]chan struct{}{}}
	p.SetLimits(limit, idle)
	return p
}

// SetLimits 改进程数上限和空闲时长。
func (p *Pool[W]) SetLimits(limit int, idle time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.limit, p.idle = max(limit, 1), idle
	p.startReaper()
}

// Get 这个会话正在跑的进程。
func (p *Pool[W]) Get(id string) (W, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.aliveLocked(id)
}

// aliveLocked 登记着、而且进程还在的成员；进程已经没了的顺手移除。
func (p *Pool[W]) aliveLocked(id string) (W, bool) {
	w, ok := p.live[id]
	if ok && !w.Alive() {
		delete(p.live, id)
		var zero W
		return zero, false
	}
	return w, ok
}

// Snapshot 现在所有在跑的进程。
func (p *Pool[W]) Snapshot() map[string]W {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make(map[string]W, len(p.live))
	for id, w := range p.live {
		out[id] = w
	}
	return out
}

// Ensure 保证这个会话有进程：有就直接用；没有就腾出位置后调用 start 起一个。
// 同一个会话同时只起一次，后来的等前面那次的结果。
func (p *Pool[W]) Ensure(ctx context.Context, id string, start func() (W, error)) (W, error) {
	var zero W
	for {
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return zero, errors.New("Momoko 正在停止")
		}
		if w, ok := p.aliveLocked(id); ok {
			p.mu.Unlock()
			return w, nil
		}
		wait := p.starting[id]
		if wait == nil {
			done := make(chan struct{})
			p.starting[id] = done
			victim, full := p.victimLocked()
			p.mu.Unlock()
			var (
				w   W
				err error
			)
			switch {
			case full:
				err = ErrPoolFull
			default:
				if victim != nil {
					(*victim).Shutdown()
				}
				w, err = start()
			}
			p.mu.Lock()
			delete(p.starting, id)
			if err == nil && w.Alive() {
				p.live[id] = w
				p.startReaper()
			}
			p.mu.Unlock()
			close(done)
			return w, err
		}
		p.mu.Unlock()
		select {
		case <-wait:
		case <-ctx.Done():
			return zero, ctx.Err()
		}
	}
}

// victimLocked 进程数（算上正在起的）超过上限时，挑空闲最久的那个让位；都在忙时 full 为 true。
func (p *Pool[W]) victimLocked() (victim *W, full bool) {
	if len(p.live)+len(p.starting) <= p.limit {
		return nil, false
	}
	for id := range p.live {
		w := p.live[id]
		if !w.Busy() && (victim == nil || w.ActiveAt().Before((*victim).ActiveAt())) {
			victim = &w
		}
	}
	return victim, victim == nil
}

// Remove 进程结束了：w 还是这个会话登记的那个时才移除（同一个会话可能已经起了新的）。
func (p *Pool[W]) Remove(id string, w W) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if cur, ok := p.live[id]; ok && any(cur) == any(w) {
		delete(p.live, id)
	}
}

// Close 结束所有进程，之后不能再起。
func (p *Pool[W]) Close() {
	p.mu.Lock()
	p.closed = true
	live := make([]W, 0, len(p.live))
	for _, w := range p.live {
		live = append(live, w)
	}
	p.mu.Unlock()
	for _, w := range live {
		w.Shutdown()
	}
}

// startReaper 池里有进程、设了空闲时长、回收协程没在跑时起一个。在 p.mu 里调用。
func (p *Pool[W]) startReaper() {
	if p.reaping || p.closed || p.idle <= 0 || len(p.live) == 0 {
		return
	}
	p.reaping = true
	go p.reap()
}

// reap 每分钟看一遍，结束空闲超过设定时长的进程；池空了或不再需要回收就退出。
func (p *Pool[W]) reap() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		p.mu.Lock()
		if p.closed || p.idle <= 0 || len(p.live) == 0 {
			p.reaping = false
			p.mu.Unlock()
			return
		}
		var stale []W
		for _, w := range p.live {
			if !w.Busy() && time.Since(w.ActiveAt()) > p.idle {
				stale = append(stale, w)
			}
		}
		p.mu.Unlock()
		for _, w := range stale {
			w.Shutdown()
		}
	}
}
