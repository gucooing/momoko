package sub2api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ImagineClient 调用 sub2api 的密钥/模型/生图接口。
// 出站统一走包级 Get/Post/DoBytes（SharedHTTPClient），不自建 Request/Client。
type ImagineClient struct{}

func NewImagineClient() *ImagineClient { return &ImagineClient{} }

// NewImagineClientWithClient 保留兼容；忽略 client，统一共享出站。
func NewImagineClientWithClient(_ *http.Client) *ImagineClient { return &ImagineClient{} }

// ImagineKey 完整缓存项（含原文 key，仅服务端持有）。
type ImagineKey struct {
	ID        string
	RawKey    string
	Name      string
	GroupID   string
	GroupName string
	Platform  string
}

// ListKeys 用 sub2api JWT 拉取用户 apikey 列表；返回 (keys, trustedUserID, error)。
func (c *ImagineClient) ListKeys(_ context.Context, srcHost, jwt string) ([]ImagineKey, string, error) {
	endpoint, err := joinURL(srcHost, imagineKeysPath)
	if err != nil {
		return nil, "", err
	}
	q := endpoint.Query()
	q.Set("page", "1")
	q.Set("page_size", "100")
	endpoint.RawQuery = q.Encode()

	body, status, err := DoBytes(http.MethodGet, endpoint.String(), map[string]string{
		"Authorization": "Bearer " + jwt,
	}, nil)
	if err != nil {
		return nil, "", err
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return nil, "", decodeImageError(body, status)
	}
	var resp imagineKeysResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, "", fmt.Errorf("解析 sub2api 密钥响应失败: %w", err)
	}
	if resp.Code != 0 && resp.Code != http.StatusOK {
		return nil, "", &ImagineRequestError{HTTPStatus: http.StatusUnauthorized, Code: strconv.Itoa(resp.Code), Message: resp.Message}
	}

	keys := make([]ImagineKey, 0, len(resp.Data.Items))
	trustedUserID := ""
	for _, item := range resp.Data.Items {
		if item.Key == "" || item.Status == "deleted" {
			continue
		}
		if trustedUserID == "" {
			trustedUserID = item.UserID.String()
		}
		keys = append(keys, ImagineKey{
			ID:        item.ID.String(),
			RawKey:    item.Key,
			Name:      item.Name,
			GroupID:   item.GroupID.String(),
			GroupName: item.Group.Name,
			Platform:  item.Group.Platform,
		})
	}
	if trustedUserID == "" {
		return nil, "", ErrImagineTokenInvalid
	}
	return keys, trustedUserID, nil
}

// FindKey 按 apikey id 在 ListKeys 结果中查找完整密钥项（含原文 key）。
func FindKey(keys []ImagineKey, id string) (ImagineKey, bool) {
	for _, k := range keys {
		if k.ID == id {
			return k, true
		}
	}
	return ImagineKey{}, false
}

// ListModels 用 apikey 拉取可用模型列表。
func (c *ImagineClient) ListModels(_ context.Context, srcHost, apiKey string) ([]imagineModelDTO, error) {
	endpoint, err := joinURL(srcHost, imagineModelsPath)
	if err != nil {
		return nil, err
	}
	body, status, err := DoBytes(http.MethodGet, endpoint.String(), map[string]string{
		"Authorization": "Bearer " + apiKey,
	}, nil)
	if err != nil {
		return nil, err
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return nil, decodeImageError(body, status)
	}
	var resp imagineModelsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析 sub2api 模型响应失败: %w", err)
	}
	return resp.Data, nil
}

func (c *ImagineClient) doImage(_ context.Context, srcHost, apiKey, path string, req *imagineImageRequest) (*imagineImageResponse, error) {
	endpoint, err := joinURL(srcHost, path)
	if err != nil {
		return nil, err
	}
	body, status, err := DoBytes(http.MethodPost, endpoint.String(), map[string]string{
		"Authorization": "Bearer " + apiKey,
	}, req)
	if err != nil {
		return nil, err
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return nil, decodeImageError(body, status)
	}
	var resp imagineImageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析 sub2api 生图响应失败: %w", err)
	}
	return &resp, nil
}

// decodeImageError 将 sub2api 网关错误（OpenAI 形）解析为带详情的 ImagineRequestError。
func decodeImageError(body []byte, status int) error {
	var env openAIErrorEnvelope
	if err := json.Unmarshal(body, &env); err == nil && env.Error.Message != "" {
		return &ImagineRequestError{
			HTTPStatus: status,
			Type:       env.Error.Type,
			Code:       env.Error.Code,
			Message:    env.Error.Message,
		}
	}
	return &ImagineRequestError{HTTPStatus: status, Message: fmt.Sprintf("sub2api 返回 HTTP %d", status)}
}

// ClassifyImagineError 将 ImagineRequestError 归类为语义 sentinel，其余原样返回。
func ClassifyImagineError(err error) error {
	if err == nil {
		return nil
	}
	var e *ImagineRequestError
	if !errors.As(err, &e) {
		return err
	}
	switch {
	case e.Code == "INVALID_API_KEY" || e.Code == "API_KEY_DISABLED" || e.HTTPStatus == http.StatusUnauthorized:
		return e
	case e.Code == "API_KEY_EXPIRED" || e.Code == "INSUFFICIENT_BALANCE" || e.HTTPStatus == http.StatusForbidden:
		return ErrImagineApiKeyForbidden
	case e.Code == "API_KEY_QUOTA_EXHAUSTED" || e.HTTPStatus == http.StatusTooManyRequests:
		return ErrImagineQuotaExhausted
	default:
		return e
	}
}

// DecodeB64Image 解析 sub2api 返回的 base64 图片为字节；url(data: 形) 也兼容。
func DecodeB64Image(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	if strings.HasPrefix(s, "data:") {
		if i := strings.Index(s, ","); i >= 0 {
			s = s[i+1:]
		}
	}
	return base64.StdEncoding.DecodeString(s)
}

func joinURL(baseURL, path string) (*url.URL, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("sub2api 地址不能为空")
	}
	raw, err := url.JoinPath(strings.TrimRight(baseURL, "/"), path)
	if err != nil {
		return nil, err
	}
	return url.Parse(raw)
}
