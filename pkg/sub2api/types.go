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
	UserAgent    string
	Status       string
	Success      bool
	LatencyMS    int64
	TokenCount   int64
	OutputTokens int64
	TPS          float64
	// 详情字段：供最近请求详情展示
	Cost            float64 // 费用（USD）
	FirstTokenMS    int64   // 首 token 延迟（毫秒）
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

// Totals 区间/全量聚合结果（由 BuildSnapshot/BuildStats 读取记录后在内存中算出）。
type Totals struct {
	TotalCount       int64
	RequestCount     int64
	SuccessCount     int64
	TokenCount       int64
	AverageLatencyMS float64
	AverageTPS       float64
}

// UsageStore 数据层需实现的持久化与读取接口（仅做 ent CRUD/读取）。
// 统计聚合改为读取记录后在内存中计算，数据层只需提供一次性读取入口，
// 避免一个页面触发数十次聚合查询。
//
// 公开页读路径必须在数据库侧按 public_enabled 分组过滤，保证全局一致。
type UsageStore interface {
	SaveUsageRecords(ctx context.Context, records []*UsageRecord) error
	ClearUsageRecords(ctx context.Context) error
	LatestUsageRecordTime(ctx context.Context) (*time.Time, error)
	LatestUpstreamErrorRecordTime(ctx context.Context) (*time.Time, error)
	// RecordsSince 按时间升序返回 start（含）之后的记录；start 为 nil 时返回全部记录。
	// publicOnly=true 时仅返回「关联分组 public_enabled=true」的记录（DB 过滤）。
	RecordsSince(ctx context.Context, start *time.Time, publicOnly bool) ([]*UsageRecord, error)
	// RecordsPage 按时间倒序（最新在前）返回 [start, end] 区间内分页的记录及区间总数。
	// publicOnly 语义同 RecordsSince。
	RecordsPage(ctx context.Context, start, end *time.Time, offset, limit int, publicOnly bool) ([]*UsageRecord, int, error)
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
