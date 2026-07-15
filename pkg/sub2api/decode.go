package sub2api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Sub2API 管理端统一响应外壳。
// 见 sub2api/backend/internal/pkg/response.Response。
type apiEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// 分页 data 外壳。
// 见 sub2api/backend/internal/pkg/response.PaginatedData。
type paginatedData struct {
	Items    json.RawMessage `json:"items"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	Pages    int             `json:"pages"`
}

// adminUsageLog 对齐 sub2api AdminUsageLog / UsageLog JSON。
// 见 sub2api/backend/internal/handler/dto/types.go。
type adminUsageLog struct {
	ID              int64   `json:"id"`
	RequestID       string  `json:"request_id"`
	Model           string  `json:"model"`
	ReasoningEffort *string `json:"reasoning_effort,omitempty"`
	InboundEndpoint *string `json:"inbound_endpoint,omitempty"`
	// UpstreamEndpoint 在 AdminUsageLog 上通过嵌入 + 覆盖写入。
	UpstreamEndpoint *string `json:"upstream_endpoint,omitempty"`

	GroupID *int64      `json:"group_id"`
	Group   *adminGroup `json:"group,omitempty"`

	UserID int64             `json:"user_id"`
	User   *adminUserSummary `json:"user,omitempty"`

	InputTokens           int `json:"input_tokens"`
	OutputTokens          int `json:"output_tokens"`
	CacheCreationTokens   int `json:"cache_creation_tokens"`
	CacheReadTokens       int `json:"cache_read_tokens"`
	CacheCreation5mTokens int `json:"cache_creation_5m_tokens"`
	CacheCreation1hTokens int `json:"cache_creation_1h_tokens"`
	ImageOutputTokens     int `json:"image_output_tokens"`

	TotalCost  float64 `json:"total_cost"`
	ActualCost float64 `json:"actual_cost"`

	DurationMs   *int `json:"duration_ms"`
	FirstTokenMs *int `json:"first_token_ms"`

	UserAgent *string   `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`

	Account *adminAccountSummary `json:"account,omitempty"`
}

type adminGroup struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type adminAccountSummary struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type adminUserSummary struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

// opsErrorLog 对齐 sub2api OpsErrorLog JSON（上游错误列表项）。
// 见 sub2api/backend/internal/service/ops_models.go。
type opsErrorLog struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`

	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`

	ClientRequestID string `json:"client_request_id"`
	RequestID       string `json:"request_id"`

	AccountName string `json:"account_name"`
	GroupID     *int64 `json:"group_id"`
	GroupName   string `json:"group_name"`

	RequestPath      string `json:"request_path"`
	InboundEndpoint  string `json:"inbound_endpoint"`
	UpstreamEndpoint string `json:"upstream_endpoint"`
	RequestedModel   string `json:"requested_model"`
	UpstreamModel    string `json:"upstream_model"`
	Model            string `json:"model"`
	UserAgent        string `json:"user_agent"`

	// 详情接口才有的耗时字段；列表项通常为空，兼容一并解析。
	UpstreamLatencyMs  *int64 `json:"upstream_latency_ms"`
	ResponseLatencyMs  *int64 `json:"response_latency_ms"`
	TimeToFirstTokenMs *int64 `json:"time_to_first_token_ms"`
}

func decodeUsageList(body []byte) (*UsageListResult, error) {
	items, total, err := decodePaginatedItems(body)
	if err != nil {
		return nil, err
	}
	records := make([]*UsageRecord, 0, len(items))
	for _, raw := range items {
		var item adminUsageLog
		if err := json.Unmarshal(raw, &item); err != nil {
			continue
		}
		if rec := usageLogToRecord(item); rec != nil {
			records = append(records, rec)
		}
	}
	return &UsageListResult{Records: records, Total: total}, nil
}

func decodeUpstreamErrorList(body []byte) (*UsageListResult, error) {
	items, total, err := decodePaginatedItems(body)
	if err != nil {
		return nil, err
	}
	records := make([]*UsageRecord, 0, len(items))
	for _, raw := range items {
		var item opsErrorLog
		if err := json.Unmarshal(raw, &item); err != nil {
			continue
		}
		if rec := opsErrorToRecord(item); rec != nil {
			records = append(records, rec)
		}
	}
	return &UsageListResult{Records: records, Total: total}, nil
}

// decodePaginatedItems 解析 Sub2API 分页响应：
// { code, message, data: { items, total, ... } }
func decodePaginatedItems(body []byte) ([]json.RawMessage, int, error) {
	var env apiEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, 0, fmt.Errorf("解析 Sub2API 响应失败: %w", err)
	}
	if env.Code != 0 && env.Code != http.StatusOK {
		msg := strings.TrimSpace(env.Message)
		if msg == "" {
			msg = fmt.Sprintf("Sub2API 返回错误码 %d", env.Code)
		}
		return nil, 0, errors.New(msg)
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return nil, 0, errors.New("Sub2API 响应缺少 data")
	}

	var page paginatedData
	if err := json.Unmarshal(env.Data, &page); err != nil {
		return nil, 0, fmt.Errorf("解析 Sub2API 分页数据失败: %w", err)
	}
	if len(page.Items) == 0 || string(page.Items) == "null" {
		return nil, int(page.Total), nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(page.Items, &items); err != nil {
		return nil, 0, fmt.Errorf("解析 Sub2API items 失败: %w", err)
	}
	return items, int(page.Total), nil
}

func usageLogToRecord(item adminUsageLog) *UsageRecord {
	requestTime := item.CreatedAt
	if requestTime.IsZero() {
		requestTime = time.Now()
	}

	id := strings.TrimSpace(item.RequestID)
	if id == "" && item.ID > 0 {
		id = strconv.FormatInt(item.ID, 10)
	}
	if id == "" {
		return nil
	}

	latency := int64FromPtr(item.DurationMs)
	output := int64(item.OutputTokens)
	if output == 0 && item.ImageOutputTokens > 0 {
		output = int64(item.ImageOutputTokens)
	}
	tokenCount := int64(item.InputTokens + item.OutputTokens + item.CacheCreationTokens + item.CacheReadTokens)
	// 若仅有分桶缓存字段，补上（与上游 usage 拆分一致）。
	if tokenCount == 0 {
		tokenCount = int64(item.InputTokens + item.OutputTokens + item.CacheCreation5mTokens + item.CacheCreation1hTokens + item.CacheReadTokens)
	}

	endpoint := firstNonEmpty(derefString(item.InboundEndpoint), derefString(item.UpstreamEndpoint))
	cost := item.ActualCost
	if cost == 0 {
		cost = item.TotalCost
	}

	groupID, groupName := groupFromUsage(item.Group, item.GroupID)
	userName := ""
	if item.User != nil {
		userName = strings.TrimSpace(item.User.Username)
	}
	return &UsageRecord{
		ID:              id,
		RequestTime:     requestTime,
		RequestDate:     requestTime.In(time.Local).Format("2006-01-02"),
		Model:           strings.TrimSpace(item.Model),
		Endpoint:        endpoint,
		GroupID:         groupID,
		GroupName:       groupName,
		UserID:          item.UserID,
		UserName:        userName,
		UserAgent:       derefString(item.UserAgent),
		Status:          "success",
		Success:         true,
		LatencyMS:       latency,
		TokenCount:      tokenCount,
		OutputTokens:    output,
		TPS:             perRequestTPS(output, latency),
		Cost:            cost,
		FirstTokenMS:    int64FromPtr(item.FirstTokenMs),
		ReasoningEffort: derefString(item.ReasoningEffort),
		AccountName:     accountNameFrom(item.Account),
		HTTPStatus:      http.StatusOK,
	}
}

func opsErrorToRecord(item opsErrorLog) *UsageRecord {
	requestTime := item.CreatedAt
	if requestTime.IsZero() {
		requestTime = time.Now()
	}

	id := ""
	switch {
	case item.ID > 0:
		id = "upstream:" + strconv.FormatInt(item.ID, 10)
	case strings.TrimSpace(item.RequestID) != "":
		id = "upstream:" + strings.TrimSpace(item.RequestID)
	case strings.TrimSpace(item.ClientRequestID) != "":
		id = "upstream:" + strings.TrimSpace(item.ClientRequestID)
	default:
		sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%d", item.CreatedAt, item.Model, item.Message, item.StatusCode)))
		id = "upstream:" + hex.EncodeToString(sum[:])
	}

	latency := int64FromInt64Ptr(item.UpstreamLatencyMs)
	if latency == 0 {
		latency = int64FromInt64Ptr(item.ResponseLatencyMs)
	}

	groupID, groupName := groupFromError(item.GroupID, item.GroupName)
	return &UsageRecord{
		ID:           id,
		RequestTime:  requestTime,
		RequestDate:  requestTime.In(time.Local).Format("2006-01-02"),
		Model:        firstNonEmpty(item.UpstreamModel, item.RequestedModel, item.Model),
		Endpoint:     firstNonEmpty(item.UpstreamEndpoint, item.InboundEndpoint, item.RequestPath),
		GroupID:      groupID,
		GroupName:    groupName,
		UserAgent:    strings.TrimSpace(item.UserAgent),
		Status:       StatusUpstreamError,
		Success:      false,
		LatencyMS:    latency,
		TokenCount:   0,
		OutputTokens: 0,
		TPS:          0,
		FirstTokenMS: int64FromInt64Ptr(item.TimeToFirstTokenMs),
		AccountName:  strings.TrimSpace(item.AccountName),
		ErrorMessage: truncateError(strings.TrimSpace(item.Message)),
		HTTPStatus:   item.StatusCode,
	}
}

// groupFromUsage 提取上游 group_id + name。
func groupFromUsage(g *adminGroup, groupID *int64) (id, name string) {
	if g != nil {
		if g.ID > 0 {
			id = strconv.FormatInt(g.ID, 10)
		}
		name = strings.TrimSpace(g.Name)
	}
	if id == "" && groupID != nil && *groupID > 0 {
		id = strconv.FormatInt(*groupID, 10)
	}
	return id, name
}

func groupFromError(groupID *int64, groupName string) (id, name string) {
	if groupID != nil && *groupID > 0 {
		id = strconv.FormatInt(*groupID, 10)
	}
	return id, strings.TrimSpace(groupName)
}


func accountNameFrom(a *adminAccountSummary) string {
	if a == nil {
		return ""
	}
	return strings.TrimSpace(a.Name)
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func int64FromPtr(p *int) int64 {
	if p == nil {
		return 0
	}
	return int64(*p)
}

func int64FromInt64Ptr(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

// maxErrorMessageLen 限制错误详情入库长度，避免存储超大响应体。
const maxErrorMessageLen = 4000

func truncateError(msg string) string {
	if len(msg) <= maxErrorMessageLen {
		return msg
	}
	return msg[:maxErrorMessageLen] + "…"
}

// perRequestTPS 计算单请求 token 生成速度（输出token/秒）。无输出或无耗时则为 0。
func perRequestTPS(outputTokens, latencyMS int64) float64 {
	if outputTokens <= 0 || latencyMS <= 0 {
		return 0
	}
	return float64(outputTokens) / (float64(latencyMS) / 1000)
}

// adminGroupListItem 对齐 /admin/groups/all 的 AdminGroup JSON（只用 id/name）。
type adminGroupListItem struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func decodeGroupList(body []byte) ([]*Group, error) {
	var env apiEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("解析 Sub2API 分组响应失败: %w", err)
	}
	if env.Code != 0 && env.Code != http.StatusOK {
		msg := strings.TrimSpace(env.Message)
		if msg == "" {
			msg = fmt.Sprintf("Sub2API 返回错误码 %d", env.Code)
		}
		return nil, errors.New(msg)
	}
	raw := env.Data
	if len(raw) == 0 || string(raw) == "null" {
		return []*Group{}, nil
	}
	var items []adminGroupListItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("解析 Sub2API 分组列表失败: %w", err)
	}
	out := make([]*Group, 0, len(items))
	for _, it := range items {
		if it.ID <= 0 {
			continue
		}
		name := strings.TrimSpace(it.Name)
		id := strconv.FormatInt(it.ID, 10)
		if name == "" {
			name = id
		}
		out = append(out, &Group{ID: id, Name: name})
	}
	return out, nil
}
