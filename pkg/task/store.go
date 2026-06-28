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
	UserID     string
	Payload    json.RawMessage // 重建参数
	State      json.RawMessage // 断点续传状态
	Results    []Result        // 执行结果明细
	Total      int64
	Finished   int64
	Message    string
	Error      string
	IntervalMS int64
	TimeoutMS  int64
	CreateTime time.Time
	EndTime    *time.Time
}

// Store 是任务持久化能力（由 internal/data 实现），让任务在重启后不丢失、可开机注入。
type Store interface {
	// Upsert 创建或整体覆盖一条任务行（按 id）。
	Upsert(ctx context.Context, r *Record) error
	// SetStatus 更新状态/消息/错误，end 非空时写入结束时间。
	SetStatus(ctx context.Context, id string, s Status, message, errText string, end *time.Time) error
	// SetProgress 更新进度与消息。
	SetProgress(ctx context.Context, id string, finished, total int64, message string) error
	// SaveState 持久化断点续传状态。
	SaveState(ctx context.Context, id string, state json.RawMessage) error
	// SaveResults 持久化执行结果明细。
	SaveResults(ctx context.Context, id string, results []Result) error
	// Get 按 id 取一条任务。
	Get(ctx context.Context, id string) (*Record, error)
	// List 分页查询（page<1 视为 1）。
	List(ctx context.Context, f Filter, page, pageSize int64) ([]*Record, int64, error)
	// LoadResumable 返回所有「未完成」任务行（status pending/running），供开机注入决策。
	LoadResumable(ctx context.Context) ([]*Record, error)
	// Delete 删除一条任务行。
	Delete(ctx context.Context, id string) error
}
