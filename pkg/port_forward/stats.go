package port_forward

import (
	"net"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Snapshot 表示某一时刻的实时统计快照。
type Snapshot struct {
	ActiveConns int64     // 当前活跃连接（TCP 连接 / UDP 会话）数
	TotalConns  int64     // 自本次启动以来的累计连接数
	BytesIn     int64     // 累计入站流量（客户端→目标）
	BytesOut    int64     // 累计出站流量（目标→客户端）
	StartTime   time.Time // 本次启动时间
	DialError   string    // 最近一次连接目标失败的原因；之后成功连上目标即清空
}

// ConnSnapshot 是一条活动连接（TCP 连接 / UDP 会话）的快照。
type ConnSnapshot struct {
	ID         string
	RemoteAddr string
	StartTime  time.Time
	BytesIn    int64
	BytesOut   int64
}

// conn 是一条活动连接的实时计数，close 用于主动断开。
type conn struct {
	id         string
	remoteAddr string
	startTime  time.Time
	bytesIn    int64
	bytesOut   int64
	close      func()
}

func (c *conn) addIn(n int64)  { atomic.AddInt64(&c.bytesIn, n) }
func (c *conn) addOut(n int64) { atomic.AddInt64(&c.bytesOut, n) }

// Stats 收集单个端口转发实例的实时计数。
// 所有计数字段均通过原子操作读写；时间序列由上层定期读取 Snapshot 并持久化。
type Stats struct {
	activeConns int64
	totalConns  int64
	bytesIn     int64
	bytesOut    int64
	startTime   time.Time
	seq         uint64

	mu        sync.Mutex
	conns     map[string]*conn
	dialError string
}

// NewStats 创建一个统计实例，并记录启动时间。
func NewStats() *Stats {
	return &Stats{startTime: time.Now(), conns: make(map[string]*conn)}
}

// connOpened 在一条连接（或 UDP 会话）建立时调用，返回该连接的计数器。
func (s *Stats) connOpened(remote net.Addr, closeFn func()) *conn {
	atomic.AddInt64(&s.activeConns, 1)
	atomic.AddInt64(&s.totalConns, 1)
	c := &conn{
		id:        strconv.FormatUint(atomic.AddUint64(&s.seq, 1), 10),
		startTime: time.Now(),
		close:     closeFn,
	}
	if remote != nil {
		c.remoteAddr = remote.String()
	}
	s.mu.Lock()
	s.conns[c.id] = c
	s.dialError = ""
	s.mu.Unlock()
	return c
}

// connClosed 在一条连接（或 UDP 会话）结束时调用。
func (s *Stats) connClosed(c *conn) {
	atomic.AddInt64(&s.activeConns, -1)
	if c == nil {
		return
	}
	s.mu.Lock()
	delete(s.conns, c.id)
	s.mu.Unlock()
}

// dialFailed 记录连接目标失败的原因。
func (s *Stats) dialFailed(err error) {
	s.mu.Lock()
	s.dialError = err.Error()
	s.mu.Unlock()
}

// addIn 累计入站字节数。
func (s *Stats) addIn(n int64) { atomic.AddInt64(&s.bytesIn, n) }

// addOut 累计出站字节数。
func (s *Stats) addOut(n int64) { atomic.AddInt64(&s.bytesOut, n) }

// Snapshot 返回当前的实时统计快照。
func (s *Stats) Snapshot() Snapshot {
	s.mu.Lock()
	dialError := s.dialError
	s.mu.Unlock()
	return Snapshot{
		ActiveConns: atomic.LoadInt64(&s.activeConns),
		TotalConns:  atomic.LoadInt64(&s.totalConns),
		BytesIn:     atomic.LoadInt64(&s.bytesIn),
		BytesOut:    atomic.LoadInt64(&s.bytesOut),
		StartTime:   s.startTime,
		DialError:   dialError,
	}
}

// Connections 返回当前全部活动连接，按建立时间升序。
func (s *Stats) Connections() []ConnSnapshot {
	s.mu.Lock()
	out := make([]ConnSnapshot, 0, len(s.conns))
	for _, c := range s.conns {
		out = append(out, ConnSnapshot{
			ID:         c.id,
			RemoteAddr: c.remoteAddr,
			StartTime:  c.startTime,
			BytesIn:    atomic.LoadInt64(&c.bytesIn),
			BytesOut:   atomic.LoadInt64(&c.bytesOut),
		})
	}
	s.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].StartTime.Before(out[j].StartTime) })
	return out
}

// CloseConnections 断开指定 id 的活动连接；ids 为空时断开全部。返回实际断开数。
func (s *Stats) CloseConnections(ids []string) int {
	s.mu.Lock()
	targets := make([]*conn, 0)
	if len(ids) == 0 {
		for _, c := range s.conns {
			targets = append(targets, c)
		}
	} else {
		for _, id := range ids {
			if c, ok := s.conns[id]; ok {
				targets = append(targets, c)
			}
		}
	}
	s.mu.Unlock()
	for _, c := range targets {
		if c.close != nil {
			c.close()
		}
	}
	return len(targets)
}
