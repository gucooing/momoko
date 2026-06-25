package port_forward

import (
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
}

// Stats 收集单个端口转发实例的实时计数。
// 所有计数字段均通过原子操作读写；时间序列由上层定期读取 Snapshot 并持久化。
type Stats struct {
	activeConns int64
	totalConns  int64
	bytesIn     int64
	bytesOut    int64
	startTime   time.Time
}

// NewStats 创建一个统计实例，并记录启动时间。
func NewStats() *Stats {
	return &Stats{startTime: time.Now()}
}

// connOpened 在一条连接（或 UDP 会话）建立时调用。
func (s *Stats) connOpened() {
	atomic.AddInt64(&s.activeConns, 1)
	atomic.AddInt64(&s.totalConns, 1)
}

// connClosed 在一条连接（或 UDP 会话）结束时调用。
func (s *Stats) connClosed() {
	atomic.AddInt64(&s.activeConns, -1)
}

// addIn 累计入站字节数。
func (s *Stats) addIn(n int64) { atomic.AddInt64(&s.bytesIn, n) }

// addOut 累计出站字节数。
func (s *Stats) addOut(n int64) { atomic.AddInt64(&s.bytesOut, n) }

// Snapshot 返回当前的实时统计快照。
func (s *Stats) Snapshot() Snapshot {
	return Snapshot{
		ActiveConns: atomic.LoadInt64(&s.activeConns),
		TotalConns:  atomic.LoadInt64(&s.totalConns),
		BytesIn:     atomic.LoadInt64(&s.bytesIn),
		BytesOut:    atomic.LoadInt64(&s.bytesOut),
		StartTime:   s.startTime,
	}
}
