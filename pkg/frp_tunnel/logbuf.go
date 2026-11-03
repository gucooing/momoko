package frp_tunnel

import (
	"strings"
	"sync"
	"time"

	frplog "github.com/fatedier/frp/pkg/util/log"
	golog "github.com/fatedier/golib/log"
	klog "github.com/go-kratos/kratos/v2/log"
)

// logCapacity 是进程内保留的 frps 日志行数上限。
const logCapacity = 1000

// LogEntry 是一行 frps 日志。
type LogEntry struct {
	Time    time.Time
	Level   string // trace/debug/info/warn/error
	Message string
}

// logRing 是 frps 日志的环形缓冲，实现 golib log.Writer（带级别与时间）。
type logRing struct {
	mu      sync.Mutex
	entries []LogEntry
	next    int
	full    bool
}

var (
	frpsLogs       = &logRing{entries: make([]LogEntry, logCapacity)}
	logCaptureOnce sync.Once
)

// installLogCapture 把 frp 的全局 logger 输出改写到环形缓冲（只做一次）。
func installLogCapture() {
	logCaptureOnce.Do(func() {
		frplog.Logger = frplog.Logger.WithOptions(
			golog.WithOutput(frpsLogs),
			golog.WithLevel(golog.InfoLevel),
			golog.WithCaller(false),
		)
	})
}

func levelName(l golog.Level) string {
	switch l {
	case golog.TraceLevel:
		return "trace"
	case golog.DebugLevel:
		return "debug"
	case golog.WarnLevel:
		return "warn"
	case golog.ErrorLevel:
		return "error"
	default:
		return "info"
	}
}

// Write 实现 io.Writer（无级别信息时按 info 记录）。
func (r *logRing) Write(p []byte) (int, error) {
	return r.WriteLog(p, golog.InfoLevel, time.Now())
}

// WriteLog 实现 golib log.Writer。
func (r *logRing) WriteLog(p []byte, level golog.Level, when time.Time) (int, error) {
	line := strings.TrimRight(string(p), "\r\n")
	// 去掉 golib 自带的「时间 [级别] 」前缀，时间与级别单独存放。
	if idx := strings.Index(line, "] "); idx > 0 && idx < 32 && strings.Contains(line[:idx+1], " [") {
		line = line[idx+2:]
	}
	entry := LogEntry{Time: when, Level: levelName(level), Message: line}
	r.mu.Lock()
	r.entries[r.next] = entry
	r.next = (r.next + 1) % len(r.entries)
	if r.next == 0 {
		r.full = true
	}
	r.mu.Unlock()
	if level >= golog.WarnLevel {
		klog.Warnf("[frps] %s", line)
	}
	return len(p), nil
}

// tail 返回最近 limit 行（按时间升序）。
func (r *logRing) tail(limit int) []LogEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := r.next
	if r.full {
		n = len(r.entries)
	}
	if limit <= 0 || limit > n {
		limit = n
	}
	out := make([]LogEntry, 0, limit)
	start := r.next - limit
	for i := 0; i < limit; i++ {
		idx := (start + i + len(r.entries)) % len(r.entries)
		out = append(out, r.entries[idx])
	}
	return out
}

// Logs 返回最近 limit 行 frps 日志（按时间升序）。
func (m *Manager) Logs(limit int) []LogEntry { return frpsLogs.tail(limit) }
