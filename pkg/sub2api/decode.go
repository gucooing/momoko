package sub2api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func decodeUsageList(body []byte) (*UsageListResult, error) {
	return decodeRecordList(body, decodeUsageRecord)
}

func decodeUpstreamErrorList(body []byte) (*UsageListResult, error) {
	return decodeRecordList(body, decodeUpstreamErrorRecord)
}

func decodeRecordList(body []byte, decode func(json.RawMessage) *UsageRecord) (*UsageListResult, error) {
	var envelope struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Data) > 0 {
		if envelope.Code != 0 && envelope.Code != http.StatusOK {
			return nil, errors.New(envelope.Message)
		}
		return decodeRecordListData(envelope.Data, decode)
	}
	return decodeRecordListData(body, decode)
}

func decodeRecordListData(raw json.RawMessage, decode func(json.RawMessage) *UsageRecord) (*UsageListResult, error) {
	var list []json.RawMessage
	if err := json.Unmarshal(raw, &list); err == nil {
		return decodeRecordItems(list, len(list), decode), nil
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	for _, key := range []string{"items", "list", "records", "data", "logs", "usages", "errors"} {
		value, ok := obj[key]
		if !ok {
			continue
		}
		if err := json.Unmarshal(value, &list); err == nil {
			return decodeRecordItems(list, jsonNumberFromRaw(obj, "total", len(list)), decode), nil
		}
	}
	return nil, errors.New("无法解析 Sub2API 列表响应")
}

func decodeRecordItems(items []json.RawMessage, total int, decode func(json.RawMessage) *UsageRecord) *UsageListResult {
	records := make([]*UsageRecord, 0, len(items))
	for _, item := range items {
		record := decode(item)
		if record == nil {
			continue
		}
		records = append(records, record)
	}
	return &UsageListResult{Records: records, Total: total}
}

func decodeUsageRecord(raw json.RawMessage) *UsageRecord {
	data := decodeJSONMap(raw)
	if data == nil {
		return nil
	}

	requestTime := jsonTime(data, "created_at", "create_time", "request_time", "timestamp", "time")
	if requestTime.IsZero() {
		requestTime = time.Now()
	}
	status := jsonString(data, "status", "state")
	statusCode := jsonInt(data, "status_code", "code", "http_status")
	success, ok := jsonBool(data, "success", "is_success", "ok")
	if !ok {
		success = statusCode == 0 || (statusCode >= 200 && statusCode < 400) || isSuccessStatus(status)
	}
	if status == "" {
		if success {
			status = "success"
		} else {
			status = "failed"
		}
	}

	id := jsonString(data, "id", "request_id", "log_id")
	if id == "" {
		id = stableRawID(raw)
	}
	latency := jsonInt(data, "duration_ms", "latency_ms", "request_duration_ms", "response_time_ms", "latency", "duration")
	output := jsonOutputTokens(data)
	return &UsageRecord{
		ID:           id,
		RequestTime:  requestTime,
		RequestDate:  requestTime.In(time.Local).Format("2006-01-02"),
		Model:        jsonString(data, "model", "requested_model", "upstream_model", "model_name"),
		Endpoint:     jsonString(data, "inbound_endpoint", "upstream_endpoint", "endpoint", "path", "route", "api_path", "request_path", "channel", "provider"),
		UserAgent:    jsonString(data, "user_agent", "userAgent", "ua", "client"),
		Status:       status,
		Success:      success,
		LatencyMS:    latency,
		TokenCount:   jsonTokenCount(data),
		OutputTokens: output,
		TPS:          perRequestTPS(output, latency),
		// 详情字段
		Cost:            jsonFloat(data, "total_cost", "actual_cost", "cost"),
		FirstTokenMS:    jsonInt(data, "first_token_ms", "time_to_first_token_ms", "ttft_ms"),
		ReasoningEffort: jsonString(data, "reasoning_effort"),
		AccountName:     jsonString(data, "account.name", "account_name"),
		HTTPStatus:      int(statusCode),
	}
}

// jsonOutputTokens 仅取实际生成的输出 token，排除输入与缓存 token。
func jsonOutputTokens(data map[string]any) int64 {
	return jsonInt(data,
		"output_tokens",
		"completion_tokens",
		"usage.output_tokens",
		"usage.completion_tokens",
	)
}

// perRequestTPS 计算单请求 token 生成速度（输出token/秒）。无输出或无耗时则为 0。
func perRequestTPS(outputTokens, latencyMS int64) float64 {
	if outputTokens <= 0 || latencyMS <= 0 {
		return 0
	}
	return float64(outputTokens) / (float64(latencyMS) / 1000)
}

func decodeUpstreamErrorRecord(raw json.RawMessage) *UsageRecord {
	data := decodeJSONMap(raw)
	if data == nil {
		return nil
	}

	requestTime := jsonTime(data, "created_at", "create_time", "request_time", "timestamp", "time")
	if requestTime.IsZero() {
		requestTime = time.Now()
	}

	id := jsonString(data, "id")
	if id != "" {
		id = "upstream:" + id
	} else if requestID := jsonString(data, "request_id", "client_request_id"); requestID != "" {
		id = "upstream:" + requestID
	} else {
		id = "upstream:" + stableRawID(raw)
	}

	return &UsageRecord{
		ID:          id,
		RequestTime: requestTime,
		RequestDate: requestTime.In(time.Local).Format("2006-01-02"),
		Model:       jsonString(data, "upstream_model", "requested_model", "model"),
		Endpoint:    jsonString(data, "upstream_endpoint", "inbound_endpoint", "request_path", "endpoint", "path"),
		Status:      StatusUpstreamError,
		Success:     false,
		LatencyMS:   jsonInt(data, "upstream_latency_ms", "response_latency_ms", "duration_ms", "latency_ms"),
		TokenCount:  0,
		// 详情字段：上游错误带有状态码与错误信息
		AccountName:  jsonString(data, "account_name"),
		ErrorMessage: truncateError(jsonString(data, "message", "error_message", "error", "upstream_error_message", "upstream_error_detail")),
		HTTPStatus:   int(jsonInt(data, "status_code", "upstream_status_code", "http_status", "code")),
	}
}

// maxErrorMessageLen 限制错误详情入库长度，避免存储超大响应体。
const maxErrorMessageLen = 4000

func truncateError(msg string) string {
	if len(msg) <= maxErrorMessageLen {
		return msg
	}
	return msg[:maxErrorMessageLen] + "…"
}

func decodeJSONMap(raw json.RawMessage) map[string]any {
	var data map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		return nil
	}
	return data
}

func jsonNumberFromRaw(obj map[string]json.RawMessage, key string, fallback int) int {
	raw, ok := obj[key]
	if !ok {
		return fallback
	}
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return int(f)
	}
	return fallback
}

func jsonString(data map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := lookupJSONValue(data, key)
		if !ok || value == nil {
			continue
		}
		if reflect.DeepEqual(value, reflect.Zero(reflect.TypeOf(value)).Interface()) {
			continue
		}
		switch v := value.(type) {
		case string:
			return strings.TrimSpace(v)
		case json.Number:
			return v.String()
		default:
			return strings.TrimSpace(fmt.Sprint(v))
		}
	}
	return ""
}

func jsonBool(data map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		value, ok := lookupJSONValue(data, key)
		if !ok || value == nil {
			continue
		}
		switch v := value.(type) {
		case bool:
			return v, true
		case string:
			parsed, err := strconv.ParseBool(strings.TrimSpace(v))
			if err == nil {
				return parsed, true
			}
		case float64:
			return v != 0, true
		case json.Number:
			n, err := v.Int64()
			if err == nil {
				return n != 0, true
			}
		case int:
			return v != 0, true
		case int64:
			return v != 0, true
		}
	}
	return false, false
}

func jsonInt(data map[string]any, keys ...string) int64 {
	for _, key := range keys {
		value, ok := lookupJSONValue(data, key)
		if !ok || value == nil {
			continue
		}
		if n, ok := anyInt(value); ok {
			return n
		}
	}
	return 0
}

func jsonFloat(data map[string]any, keys ...string) float64 {
	for _, key := range keys {
		value, ok := lookupJSONValue(data, key)
		if !ok || value == nil {
			continue
		}
		if f, ok := anyFloat(value); ok {
			return f
		}
	}
	return 0
}

func anyFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		if err == nil {
			return f, true
		}
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			return f, true
		}
	}
	return 0, false
}

func jsonTokenCount(data map[string]any) int64 {
	splitTotal := sumPresentJSONInts(data,
		"input_tokens",
		"output_tokens",
		"cache_creation_tokens",
		"cache_read_tokens",
		"cache_creation_input_tokens",
		"cache_read_input_tokens",
		"image_output_tokens",
	)
	if splitTotal > 0 {
		return splitTotal
	}

	cacheCreation := jsonInt(data, "cache_creation_input_tokens", "usage.cache_creation_input_tokens", "input_tokens_details.cache_creation_tokens")
	if cacheCreation == 0 {
		cacheCreation = sumPresentJSONInts(data, "cache_creation_5m_tokens", "cache_creation_1h_tokens")
	}
	cacheRead := jsonInt(data,
		"cache_read_input_tokens",
		"usage.cache_read_input_tokens",
		"prompt_tokens_details.cached_tokens",
		"input_tokens_details.cached_tokens",
		"usage.prompt_tokens_details.cached_tokens",
		"usage.input_tokens_details.cached_tokens",
	)
	anthropicStyleTotal := sumPresentJSONInts(data,
		"input_tokens",
		"output_tokens",
		"usage.input_tokens",
		"usage.output_tokens",
	) + cacheCreation + cacheRead
	if anthropicStyleTotal > 0 {
		return anthropicStyleTotal
	}

	total := jsonInt(data, "total_tokens", "token_count", "total_token_count", "usage.total_tokens")
	if total > 0 {
		return total
	}
	return jsonInt(data, "prompt_tokens", "usage.prompt_tokens") +
		jsonInt(data, "completion_tokens", "usage.completion_tokens")
}

func sumPresentJSONInts(data map[string]any, keys ...string) int64 {
	var total int64
	var found bool
	for _, key := range keys {
		value, ok := lookupJSONValue(data, key)
		if !ok || value == nil {
			continue
		}
		n, ok := anyInt(value)
		if !ok {
			continue
		}
		total += n
		found = true
	}
	if !found {
		return 0
	}
	return total
}

func anyInt(value any) (int64, bool) {
	switch v := value.(type) {
	case float64:
		return int64(v), true
	case float32:
		return int64(v), true
	case int:
		return int64(v), true
	case int64:
		return v, true
	case json.Number:
		n, err := v.Int64()
		if err == nil {
			return n, true
		}
		f, err := strconv.ParseFloat(v.String(), 64)
		if err == nil {
			return int64(f), true
		}
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			return int64(n), true
		}
	}
	return 0, false
}

func lookupJSONValue(data map[string]any, key string) (any, bool) {
	if value, ok := data[key]; ok {
		return value, true
	}
	if !strings.Contains(key, ".") {
		return nil, false
	}
	var current any = data
	for _, part := range strings.Split(key, ".") {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = obj[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func jsonTime(data map[string]any, keys ...string) time.Time {
	for _, key := range keys {
		value, ok := lookupJSONValue(data, key)
		if !ok || value == nil {
			continue
		}
		switch v := value.(type) {
		case string:
			if t, ok := parseTime(strings.TrimSpace(v)); ok {
				return t
			}
		case float64:
			return unixFlexibleTime(int64(v))
		case json.Number:
			n, _ := v.Int64()
			return unixFlexibleTime(n)
		case int64:
			return unixFlexibleTime(v)
		case int:
			return unixFlexibleTime(int64(v))
		}
	}
	return time.Time{}
}

func parseTime(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, true
		}
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func unixFlexibleTime(n int64) time.Time {
	switch {
	case n > 1e17:
		return time.Unix(0, n)
	case n > 1e12:
		return time.UnixMilli(n)
	default:
		return time.Unix(n, 0)
	}
}

func isSuccessStatus(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	return status == "" || status == "success" || status == "succeeded" || status == "ok" || status == "completed"
}

func stableRawID(raw []byte) string {
	normalized := bytes.TrimSpace(raw)
	sum := sha256.Sum256(normalized)
	return hex.EncodeToString(sum[:])
}
