package sub2api

import (
	"context"
	"strings"
	"time"

	"momoko/pkg/common"
)

// Sub2API 管理端接口路径。
const (
	DefaultAdminUsagePath          = "/api/v1/admin/usage"
	DefaultAdminUpstreamErrorsPath = "/api/v1/admin/ops/upstream-errors"
	DefaultAdminGroupsPath         = "/api/v1/admin/groups/all"
	StatusUpstreamError            = "upstream_error"
)

// 同步参数默认值与边界。
const (
	DefaultSyncIntervalMinutes int32 = 10
	DefaultHistoryDays         int32 = 30
	DefaultPageSize            int32 = 500
	MinSyncIntervalMinutes     int32 = 1
	MaxSyncIntervalMinutes     int32 = 1440
	MinHistoryDays             int32 = 1
	MaxHistoryDays             int32 = 365
	MinPageSize                int32 = 50
	MaxPageSize                int32 = 1000
	SyncTimeout                      = 9 * time.Minute
)

// 同步状态。
const (
	SyncStatusIdle    = "idle"
	SyncStatusSyncing = "syncing"
	SyncStatusSuccess = "success"
	SyncStatusError   = "error"
)

// GroupField 聚合维度。
type GroupField string

const (
	GroupByModel     GroupField = "model"
	GroupByUserAgent GroupField = "user_agent"
	GroupByGroup     GroupField = "group"
)

// ClientConfig 调用 Sub2API 管理端所需的连接信息。
type ClientConfig struct {
	BaseURL     string
	AdminAPIKey string
}

func NewClientConfig(baseURL, adminAPIKey string) ClientConfig {
	return ClientConfig{
		BaseURL:     strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		AdminAPIKey: strings.TrimSpace(adminAPIKey),
	}
}

func (c ClientConfig) Configured() bool {
	return c.BaseURL != "" && c.AdminAPIKey != ""
}

// UsageRecord 单条使用记录领域模型。
type UsageRecord struct {
	ID           string
	RequestTime  time.Time
	RequestDate  string
	Model        string
	Endpoint     string
	GroupID      string // 关联本地 Sub2APIGroup.id
	GroupName    string // Sub2API 分组名称
	UserID       int64  // Sub2API 用户 ID（上游 int64，不转换）
	UserName     string // Sub2API 用户名
	UserAgent    string
	Status       string
	Success      bool
	LatencyMS    int64
	TokenCount   int64
	OutputTokens int64
	TPS          float64 // 吐出速度(token/s)：output_tokens*1000/(duration_ms-first_token_ms)，时间按 ms
	// 详情字段：供最近请求详情展示
	Cost            float64 // 费用（USD）
	FirstTokenMS    int64   // 首 token / 首字节延迟（毫秒）
	ReasoningEffort string  // 推理强度
	AccountName     string  // 账号名称
	ErrorMessage    string  // 错误详情（失败/上游错误请求）
	HTTPStatus      int     // HTTP 状态码
}

// Group 本地分组领域模型。
type Group struct {
	ID            string
	Name          string
	PublicEnabled bool
	Deleted       bool
}

// DeletedGroupPublicKey 配置页「已删除分组」合并选项的约定 key。
// 勾选后，所有 deleted=true 的分组 public_enabled 一并开启。
const DeletedGroupPublicKey = "__deleted__"

// SyncState 最近一次同步的状态。
type SyncState struct {
	Status           string     `json:"status"`
	Error            string     `json:"error,omitempty"`
	LastSyncTime     *time.Time `json:"last_sync_time,omitempty"`
	NextSyncTime     *time.Time `json:"next_sync_time,omitempty"`
	LatestRecordTime *time.Time `json:"latest_record_time,omitempty"`
	SyncedRecords    int64      `json:"synced_records,omitempty"`
}

// 同步选项与结果。
type SyncOptions struct {
	Full     bool
	PageSize int
}

type SyncResult struct {
	SyncedRecords    int64
	LatestRecordTime *time.Time
}

// UsageListOptions 列表查询选项（传递给 Sub2API 管理端）。
type UsageListOptions struct {
	Page      int
	PageSize  int
	SortBy    string
	SortDesc  bool
	StartTime *time.Time
	EndTime   *time.Time
}

type UsageListResult struct {
	Records []*UsageRecord
	Total   int
}

// Totals 区间/全量聚合结果（由数据层在 DB 侧 Count/Sum/Mean 算出）。
type Totals struct {
	TotalCount       int64
	RequestCount     int64
	SuccessCount     int64
	TokenCount       int64
	AverageLatencyMS float64
	AverageTPS       float64
}

// MinTPSOutputTokens 生成速度统计的最小输出 token 阈值：输出过短的请求速率无统计意义，
// 输出 token 少于该值的请求不计入平均生成速度（DB 侧 WHERE 过滤）。
const MinTPSOutputTokens = 20

// StatsWindow 聚合查询的时间窗口与可见性过滤。
// Start 为 nil 表示不限起点；End 为 nil 表示不限终点；PublicOnly 仅统计公开启用分组的记录。
type StatsWindow struct {
	Start      *time.Time
	End        *time.Time
	PublicOnly bool
}

// TopStat 单维度（模型/接口/分组）用量聚合（数据层 GroupBy 结果）。
type TopStat struct {
	Name    string
	Request int64   // 计费请求数
	Token   int64   // 计费请求 token 之和
	Success int64   // 成功请求数
	AvgTPS  float64 // 达标请求平均生成速度
}

// DayStat 单日聚合（数据层 GroupBy(request_date) 结果）。
type DayStat struct {
	Date         string
	Request      int64
	Success      int64
	Token        int64
	AvgLatencyMS float64
}

// BucketStat 15 分钟桶聚合（数据层 GroupBy(bucket15m) 结果）。
type BucketStat struct {
	Bucket       time.Time
	Total        int64 // 全部记录数
	Request      int64 // 计费请求数
	Success      int64
	Token        int64
	AvgLatencyMS float64
	AvgTPS       float64
}

// RecordFilter 最近请求多维度筛选；指针字段为 nil 表示该维度不参与过滤。
type RecordFilter struct {
	Model       *string // 精确匹配
	GroupName   *string // 精确匹配
	AccountName *string // 子串（不区分大小写）
	Outcome     *string // "success" | "failed"
}

// UsageStore 数据层需实现的持久化与聚合接口。
// 统计聚合全部下沉到数据库侧（ent Count/Sum/Mean/GroupBy），不在内存中遍历记录聚合；
// 公开页读路径必须在数据库侧按 public_enabled 分组过滤，保证全局一致。
type UsageStore interface {
	SaveUsageRecords(ctx context.Context, records []*UsageRecord) error
	ClearUsageRecords(ctx context.Context) error
	LatestUsageRecordTime(ctx context.Context) (*time.Time, error)
	LatestUpstreamErrorRecordTime(ctx context.Context) (*time.Time, error)
	// RecordsPage 按时间倒序（最新在前）返回 [start, end] 区间内分页的记录及区间总数，
	// 支持多维度筛选（RecordFilter）。publicOnly=true 仅返回公开启用分组的记录。
	RecordsPage(ctx context.Context, start, end *time.Time, offset, limit int, publicOnly bool, filter RecordFilter) ([]*UsageRecord, int, error)
	// AggregateTotals 汇总窗口内标量指标（DB 侧 Count/Sum/Mean，含 TPS 达标过滤）。
	AggregateTotals(ctx context.Context, w StatsWindow) (Totals, error)
	// PublicOverview 公开首页今日概览：严格 2 次 ent GroupBy（桶请求/token + 桶成功/TPS），
	// 总量由桶结果汇总；不走完整快照、不扫明细行。
	PublicOverview(ctx context.Context, w StatsWindow) (Totals, []BucketStat, error)
	// TopItems 按维度分组的用量排行（DB 侧 GroupBy），按 token 降序；limit<=0 返回全部。
	TopItems(ctx context.Context, w StatsWindow, field GroupField, limit int) ([]TopStat, error)
	// DailyTrend 按自然日聚合（DB 侧 GroupBy(request_date)），按日期升序。
	DailyTrend(ctx context.Context, w StatsWindow) ([]DayStat, error)
	// IntradaySeries 按 15 分钟桶聚合（DB 侧 GroupBy(bucket15m)），按桶升序。
	IntradaySeries(ctx context.Context, w StatsWindow) ([]BucketStat, error)
	// ListGroups 返回本地分组列表（活跃在前，按名称排序）。
	ListGroups(ctx context.Context) ([]*Group, error)
	// SyncGroups 用上游存活分组 ID 集合刷新本地：命中的标未删除并更新名；本地多出的标 deleted。
	SyncGroups(ctx context.Context, live []*Group) error
	// SetPublicGroups 设置公开启用。
	// names 为活跃分组 name；若包含 DeletedGroupPublicKey，则所有已删除分组一并启用。
	SetPublicGroups(ctx context.Context, names []string) error
}

// ConfigStore KV 配置存储（由数据层 ConfigRepo 适配）。
type ConfigStore interface {
	Get(ctx context.Context, key common.ConfigKey) (string, error)
	BatchUpdate(ctx context.Context, configs map[common.ConfigKey]string) error
}
