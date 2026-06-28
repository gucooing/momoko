// Package task 是一个通用的异步任务管理器：统一托管「超时型一次性任务、定时任务、常驻任务」三类，
// 负责入库持久化、开机注入、进度/事件上报与订阅，但任务的具体执行逻辑仍留在各自业务 pkg 内
// （本包零业务依赖，由 pkg/file、pkg/docker 等反向 import）。
package task

import (
	"context"
	"time"
)

// Kind 决定任务的调度语义。
type Kind string

const (
	KindOneShot   Kind = "oneshot"   // 超时型一次性：在脱离请求、可选超时的 ctx 下执行一次
	KindScheduled Kind = "scheduled" // 定时触发：每 Interval 执行一次，直到取消/停止
	KindDaemon    Kind = "daemon"    // 常驻：开机执行一次，Run 阻塞至 ctx 取消
)

// ResumePolicy 决定开机时对一条历史任务行的处理方式。
type ResumePolicy string

const (
	ResumeNone   ResumePolicy = "none"   // 未完成即标记失败（不可重入，如复制/剪切）
	ResumeRerun  ResumePolicy = "rerun"  // 未完成则重新执行（幂等收尾，如远端上传收尾）
	ResumeAlways ResumePolicy = "always" // 默认开启的单例（GC/守护），由 EnsureSingleton 维护
)

// Status 是任务状态。
type Status string

const (
	StatusPending  Status = "pending"
	StatusRunning  Status = "running"
	StatusSuccess  Status = "success"
	StatusFailed   Status = "failed"
	StatusCanceled Status = "canceled"
)

// Terminal 报告该状态是否为终态。
func (s Status) Terminal() bool {
	return s == StatusSuccess || s == StatusFailed || s == StatusCanceled
}

// Meta 描述一个任务的身份与调度参数（由 Task.Meta() 返回，并据此建行）。
type Meta struct {
	ID       string        // 稳定 id；单例任务取其 Type 以便幂等 upsert，一次性任务为 uuid
	Type     string        // 注册类型键，用于开机按工厂重建
	Kind     Kind          // 调度语义
	Title    string        // 展示标题
	UserID   string        // 发起用户；空=系统任务
	Total    int64         // 初始总进度（如批量复制的文件数），用于提交即可见
	Timeout  time.Duration // KindOneShot 的超时；0=不超时
	Interval time.Duration // KindScheduled 的触发周期
	Resume   ResumePolicy  // 开机注入策略
}

// Result 是任务的单项执行结果（如批量复制中某个文件的成败）。
type Result struct {
	Path    string `json:"path"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Event 是运行期向订阅者实时广播的事件（如 Docker 拉取日志行）。仅在内存广播，不入库。
type Event struct {
	Message  string
	Progress string
	Error    string
	Status   Status
}

// Task 是唯一的任务接口，三类任务都实现它；执行逻辑留在各自 pkg。
type Task interface {
	// Meta 返回任务身份与调度参数。
	Meta() Meta
	// Run 执行任务。ctx 已脱离请求生命周期（见 Manager 的派生），KindOneShot 叠加超时，
	// KindScheduled 每次触发调用一次，KindDaemon 应阻塞至 ctx 取消并在取消时返回 ctx.Err()。
	Run(ctx context.Context, r Reporter) error
	// Payload 返回可 JSON 序列化的重建参数，落库后供 Factory 在重启后重建本任务；无需重建可返回 nil。
	Payload() any
}

// Reporter 是任务运行期上报进度/事件/结果与持久化断点的句柄；并发安全，终态后为 no-op。
type Reporter interface {
	SetProgress(finished, total int64, message string) // 粗粒度进度（入库）
	Emit(e Event)                                      // 实时事件（仅内存广播 + 回放缓冲）
	AppendResult(r Result)                             // 追加单项结果（入库）
	Checkpoint(state any) error                        // 持久化断点续传状态（入库）
}

// Factory 在开机/重试时按持久化记录重建一个 Task；在业务层以闭包形式注册，
// 从而把「解密/重建 Store」等业务细节留在业务层。
type Factory func(ctx context.Context, rec *Record) (Task, error)

// Info 是对外只读视图（业务层据此映射为 proto）。
type Info struct {
	ID         string
	Type       string
	Kind       Kind
	Title      string
	UserID     string
	Status     Status
	Total      int64
	Finished   int64
	Message    string
	Error      string
	Results    []Result
	CreateTime time.Time
	EndTime    *time.Time
}

// Filter 用于列表查询。
type Filter struct {
	UserID     string
	TypePrefix string
	Status     Status
	Kind       Kind
	Keywords   string
}
