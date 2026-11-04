// Package task 是一个通用的异步任务管理器：统一托管「超时型一次性任务、定时任务、常驻任务」三类，
// 负责入库持久化、开机注入、来源记录、进度/步骤/日志上报与订阅，但任务的具体执行逻辑仍留在各自业务 pkg 内
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

// Trigger 是任务的触发方式（任务面板「触发方式」）。
type Trigger string

const (
	TriggerManual   Trigger = "manual"   // 用户在页面上手动发起
	TriggerSchedule Trigger = "schedule" // 定时/常驻单例，由系统按计划触发
	TriggerMaster   Trigger = "master"   // 远程主控经 API Key 转发到本节点
	TriggerSystem   Trigger = "system"   // 系统内部发起（无用户上下文）
)

// Origin 是从请求 ctx 里解析出的任务发起方（由业务层通过 Manager.SetOriginResolver 注入解析逻辑）。
type Origin struct {
	Trigger  Trigger
	UserID   string // 本节点上的执行身份（主控转发时为 API Key 所属用户）
	Operator string // 远程主控侧的操作者名称（仅主控转发时有值）
}

// OriginResolver 从提交任务时的 ctx 解析发起方。
type OriginResolver func(ctx context.Context) Origin

// Meta 描述一个任务的身份与调度参数（由 Task.Meta() 返回，并据此建行）。
type Meta struct {
	ID       string        // 稳定 id；单例任务取其 Type 以便幂等 upsert，一次性任务为 uuid
	Type     string        // 注册类型键，用于开机按工厂重建，也决定面板里的图标与类型名
	Kind     Kind          // 调度语义
	Title    string        // 展示标题
	Target   string        // 关联对象（镜像引用、容器/网络名、目标目录、文件名…）
	Source   string        // 发起入口（页面/模块键，如 docker.container、file、instance.file）
	UserID   string        // 发起用户；为空时由 OriginResolver 从 ctx 解析，仍为空=系统任务
	Total    int64         // 初始总进度（如批量复制的文件数），用于提交即可见
	Steps    []string      // 预定义执行步骤（可选）；Reporter.Step 依次推进，未执行到的在终态时标为跳过
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

// StepState 是执行步骤的状态。
type StepState string

const (
	StepWaiting StepState = "waiting"
	StepRunning StepState = "running"
	StepSuccess StepState = "success"
	StepFailed  StepState = "failed"
	StepSkipped StepState = "skipped"
)

// Step 是任务的一个执行步骤（任务面板「执行步骤」时间线）。
type Step struct {
	Title     string     `json:"title"`
	State     StepState  `json:"state"`
	Detail    string     `json:"detail,omitempty"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

// LogLevel 是任务日志行的级别。
type LogLevel string

const (
	LogInfo    LogLevel = "info"
	LogSuccess LogLevel = "success"
	LogWarn    LogLevel = "warn"
	LogError   LogLevel = "error"
)

// LogEntry 是任务日志的一行（任务面板「实时日志」）。
type LogEntry struct {
	Time    time.Time `json:"time"`
	Level   LogLevel  `json:"level"`
	Message string    `json:"message"`
}

// MaxLogEntries 是每个任务保留的日志行数上限（超出时丢弃最早的行）。
const MaxLogEntries = 500

// Event 是运行期向订阅者实时广播的事件（如 Docker 拉取日志行）。
// 事件只在内存广播；其中的消息/错误同时记为任务日志，随任务行入库。
type Event struct {
	Message  string
	Progress string
	Error    string
	Status   Status
	Time     time.Time // 事件时间；为零时由管理器补为当前时间
	Quiet    bool      // 只广播不记日志（调用方已用 Reporter.Log 记录清洗后的日志）
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

// Reporter 是任务运行期上报进度/步骤/日志/结果与持久化断点的句柄；并发安全，终态后为 no-op。
type Reporter interface {
	// SetProgress 上报粗粒度进度与当前进度说明（如「第 5 / 7 层 · 18.2 / 29.1 MB」）；入库有节流。
	SetProgress(finished, total int64, message string)
	// Step 进入名为 title 的步骤（上一个进行中的步骤记为成功）；title 与当前步骤相同时只更新 detail。
	Step(title, detail string)
	// Log 追加一行任务日志。
	Log(level LogLevel, message string)
	// Emit 广播实时事件（订阅者 + 回放缓冲），其中的消息/错误同时记为日志。
	Emit(e Event)
	// AppendResult 追加单项结果（入库）。
	AppendResult(r Result)
	// Checkpoint 持久化断点续传状态（入库）。
	Checkpoint(state any) error
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
	Target     string
	Source     string
	Trigger    Trigger
	UserID     string
	Operator   string
	Status     Status
	Total      int64
	Finished   int64
	Message    string
	Error      string
	Results    []Result
	Steps      []Step
	Logs       []LogEntry // 列表查询不带，Get 才返回
	LogCount   int
	Interval   time.Duration
	CanceledBy string
	CreateTime time.Time
	StartTime  *time.Time
	EndTime    *time.Time
	// NextRunTime 定时任务下次触发时间（仅运行时内存态，两次触发之间有值）
	NextRunTime *time.Time
}

// Filter 用于列表查询。
type Filter struct {
	UserID     string
	TypePrefix string
	Status     Status
	Kind       Kind
	Keywords   string // 匹配标题、类型、关联对象
}
