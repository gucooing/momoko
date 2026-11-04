package task

import (
	"context"
	"encoding/json"
	"time"
)

// Record 是一条任务的完整持久化形态（由数据层映射到 ent task 表）。
type Record struct {
	ID         string
	Type       string
	Kind       Kind
	Status     Status
	Resume     ResumePolicy
	Title      string
	Target     string
	Source     string
	Trigger    Trigger
	UserID     string
	Operator   string
	Payload    json.RawMessage // 重建参数
	State      json.RawMessage // 断点续传状态
	Results    []Result        // 执行结果明细
	Steps      []Step          // 执行步骤
	Logs       []LogEntry      // 任务日志
	Total      int64
	Finished   int64
	Message    string
	Error      string
	CanceledBy string
	IntervalMS int64
	TimeoutMS  int64
	CreateTime time.Time
	StartTime  *time.Time
	EndTime    *time.Time
}

// Snapshot 是运行期需要落库的一组可变字段（进度/消息/步骤/日志），由管理器节流写入。
type Snapshot struct {
	Total    int64
	Finished int64
	Message  string
	Steps    []Step
	Logs     []LogEntry
}

// Store 是任务持久化能力（由 internal/data 实现），让任务在重启后不丢失、可开机注入。
type Store interface {
	// Upsert 创建或整体覆盖一条任务行（按 id）。
	Upsert(ctx context.Context, r *Record) error
	// SetRunning 标记开始执行并写入开始时间。
	SetRunning(ctx context.Context, id string, start time.Time) error
	// SetStatus 只更新状态（定时任务在两次触发之间回到等待）。
	SetStatus(ctx context.Context, id string, s Status) error
	// SaveSnapshot 写入进度/消息/步骤/日志。
	SaveSnapshot(ctx context.Context, id string, s Snapshot) error
	// Finish 写入终态：状态、错误、取消人、结束时间以及最终的进度/步骤/日志快照。
	Finish(ctx context.Context, id string, s Status, errText, canceledBy string, end time.Time, snap Snapshot) error
	// SaveState 持久化断点续传状态。
	SaveState(ctx context.Context, id string, state json.RawMessage) error
	// SaveResults 持久化执行结果明细。
	SaveResults(ctx context.Context, id string, results []Result) error
	// Get 按 id 取一条任务。
	Get(ctx context.Context, id string) (*Record, error)
	// List 分页查询（page<1 视为 1），按创建时间倒序；返回的记录不带日志。
	List(ctx context.Context, f Filter, page, pageSize int64) ([]*Record, int64, error)
	// LoadResumable 返回所有「未完成」任务行（status pending/running），供开机注入决策。
	LoadResumable(ctx context.Context) ([]*Record, error)
	// Delete 删除若干任务行，返回删除行数。
	Delete(ctx context.Context, ids []string) (int64, error)
	// CountByStatus 按状态统计满足过滤条件（忽略 Filter.Status）的任务数。
	CountByStatus(ctx context.Context, f Filter) (map[Status]int64, error)
	// CountByType 统计全部任务里每个类型的数量。
	CountByType(ctx context.Context) (map[string]int64, error)
	// DeleteByStatus 批量删除处于给定状态的任务行，返回删除行数。
	DeleteByStatus(ctx context.Context, statuses []Status) (int64, error)
	// DeleteFinishedBefore 删除结束时间早于 before 的终态任务行，返回删除行数。
	DeleteFinishedBefore(ctx context.Context, before time.Time) (int64, error)
}
