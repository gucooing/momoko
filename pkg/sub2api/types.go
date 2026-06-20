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
	GroupByModel    GroupField = "model"
	GroupByEndpoint GroupField = "endpoint"
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
	Status       string
	Success      bool
	LatencyMS    int64
	TokenCount   int64
	OutputTokens int64
	TPS          float64
}

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
type UsageStore interface {
	SaveUsageRecords(ctx context.Context, records []*UsageRecord) error
	ClearUsageRecords(ctx context.Context) error
	LatestUsageRecordTime(ctx context.Context) (*time.Time, error)
	LatestUpstreamErrorRecordTime(ctx context.Context) (*time.Time, error)
	// RecordsSince 按时间升序返回 start（含）之后的记录；start 为 nil 时返回全部记录。
	RecordsSince(ctx context.Context, start *time.Time) ([]*UsageRecord, error)
}

// ConfigStore KV 配置存储（由数据层 ConfigRepo 适配）。
type ConfigStore interface {
	Get(ctx context.Context, key common.ConfigKey) (string, error)
	BatchUpdate(ctx context.Context, configs map[common.ConfigKey]string) error
}
