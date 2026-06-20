package sub2api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const requestTimeout = 30 * time.Second

// Manager 负责与 Sub2API 管理端 HTTP 接口交互。
type Manager struct {
	client *http.Client
}

func NewManager() *Manager {
	return &Manager{client: &http.Client{Timeout: requestTimeout}}
}

// Test 通过拉取一条记录验证连接与密钥是否可用。
func (m *Manager) Test(ctx context.Context, cfg ClientConfig) error {
	_, err := m.ListUsage(ctx, cfg, UsageListOptions{Page: 1, PageSize: 1})
	return err
}

func (m *Manager) ListUsage(ctx context.Context, cfg ClientConfig, opts UsageListOptions) (*UsageListResult, error) {
	endpoint, err := adminURL(cfg.BaseURL, DefaultAdminUsagePath)
	if err != nil {
		return nil, err
	}
	applyListQuery(endpoint, opts)
	body, err := m.doAdminGet(ctx, cfg, endpoint)
	if err != nil {
		return nil, err
	}
	return decodeUsageList(body)
}

func (m *Manager) ListUpstreamErrors(ctx context.Context, cfg ClientConfig, opts UsageListOptions) (*UsageListResult, error) {
	endpoint, err := adminURL(cfg.BaseURL, DefaultAdminUpstreamErrorsPath)
	if err != nil {
		return nil, err
	}
	applyListQuery(endpoint, opts)
	body, err := m.doAdminGet(ctx, cfg, endpoint)
	if err != nil {
		return nil, err
	}
	return decodeUpstreamErrorList(body)
}

func applyListQuery(endpoint *url.URL, opts UsageListOptions) {
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PageSize <= 0 {
		opts.PageSize = 100
	}
	if opts.SortBy == "" {
		opts.SortBy = "created_at"
	}
	q := endpoint.Query()
	q.Set("page", strconv.Itoa(opts.Page))
	q.Set("page_size", strconv.Itoa(opts.PageSize))
	q.Set("sort_by", opts.SortBy)
	if opts.SortDesc {
		q.Set("sort_order", "desc")
	} else {
		q.Set("sort_order", "asc")
	}
	if opts.StartTime != nil && !opts.StartTime.IsZero() {
		q.Set("start_time", opts.StartTime.UTC().Format(time.RFC3339Nano))
	}
	if opts.EndTime != nil && !opts.EndTime.IsZero() {
		q.Set("end_time", opts.EndTime.UTC().Format(time.RFC3339Nano))
	}
	endpoint.RawQuery = q.Encode()
}

func (m *Manager) doAdminGet(ctx context.Context, cfg ClientConfig, endpoint *url.URL) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-key", cfg.AdminAPIKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Sub2API 返回 HTTP %d", resp.StatusCode)
	}
	return body, nil
}

func adminURL(baseURL, path string) (*url.URL, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("Sub2API 地址不能为空")
	}
	raw, err := url.JoinPath(strings.TrimRight(baseURL, "/"), path)
	if err != nil {
		return nil, err
	}
	return url.Parse(raw)
}
