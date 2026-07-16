package sub2api

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Sub2API 管理端路径。
const (
	DefaultUserProfilePath  = "/api/v1/user/profile"
	adminUserBalancePathFmt = "/api/v1/admin/users/%d/balance"
	adminUserByIDPathFmt    = "/api/v1/admin/users/%d"
)

// Sub2APIUserInfo 管理端用户信息（抽奖报名者详情展示用）。
type Sub2APIUserInfo struct {
	ID        int64
	Username  string
	Email     string
	Role      string
	Balance   float64
	Status    string
	CreatedAt time.Time
}

// Manager 管理端 API 门面：出站一律走包级泛型 Get/Post/DoEnvelope，不自建 Request/Client。
type Manager struct{}

func NewManager() *Manager { return &Manager{} }

// NewManagerWithClient 保留签名以兼容旧调用；client 参数忽略（统一 SharedHTTPClient）。
func NewManagerWithClient(_ *http.Client) *Manager { return &Manager{} }

func adminHeaders(cfg ClientConfig) map[string]string {
	return map[string]string{"x-api-key": cfg.AdminAPIKey}
}

func bearerHeaders(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

// Test 通过拉取一条记录验证连接与密钥。
func (m *Manager) Test(cfg ClientConfig) error {
	_, err := m.ListUsage(cfg, UsageListOptions{Page: 1, PageSize: 1})
	return err
}

func (m *Manager) ListUsage(cfg ClientConfig, opts UsageListOptions) (*UsageListResult, error) {
	endpoint, err := adminURL(cfg.BaseURL, DefaultAdminUsagePath)
	if err != nil {
		return nil, err
	}
	applyListQuery(endpoint, opts)
	// 分页列表解码逻辑在 decodeUsageList（保留历史字段兼容），因此用 DoBytes。
	body, status, err := DoBytes(http.MethodGet, endpoint.String(), adminHeaders(cfg), nil)
	if err != nil {
		return nil, err
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Sub2API 返回 HTTP %d", status)
	}
	return decodeUsageList(body)
}

func (m *Manager) ListUpstreamErrors(cfg ClientConfig, opts UsageListOptions) (*UsageListResult, error) {
	endpoint, err := adminURL(cfg.BaseURL, DefaultAdminUpstreamErrorsPath)
	if err != nil {
		return nil, err
	}
	applyListQuery(endpoint, opts)
	body, status, err := DoBytes(http.MethodGet, endpoint.String(), adminHeaders(cfg), nil)
	if err != nil {
		return nil, err
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Sub2API 返回 HTTP %d", status)
	}
	return decodeUpstreamErrorList(body)
}

// ListGroups 拉取管理端全部活跃分组。
func (m *Manager) ListGroups(cfg ClientConfig) ([]*Group, error) {
	endpoint, err := adminURL(cfg.BaseURL, DefaultAdminGroupsPath)
	if err != nil {
		return nil, err
	}
	body, status, err := DoBytes(http.MethodGet, endpoint.String(), adminHeaders(cfg), nil)
	if err != nil {
		return nil, err
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Sub2API 返回 HTTP %d", status)
	}
	return decodeGroupList(body)
}

// userProfileDTO 对齐 GET /user/profile 的 data。
type userProfileDTO struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

// ValidateUserToken 用用户 JWT 拉 profile，返回上游 int64 user_id。
func (m *Manager) ValidateUserToken(cfg ClientConfig, jwt string) (userID int64, userName string, err error) {
	jwt = strings.TrimSpace(jwt)
	if jwt == "" {
		return 0, "", fmt.Errorf("缺少用户令牌")
	}
	endpoint, err := adminURL(cfg.BaseURL, DefaultUserProfilePath)
	if err != nil {
		return 0, "", err
	}
	profile, err := GetEnvelope[userProfileDTO](endpoint.String(), bearerHeaders(jwt))
	if err != nil {
		return 0, "", fmt.Errorf("令牌无效")
	}
	if profile.ID <= 0 {
		return 0, "", fmt.Errorf("令牌无效")
	}
	return profile.ID, strings.TrimSpace(profile.Username), nil
}

// adminUserDTO 对齐 GET /admin/users/:id 的 data（仅取报名者展示所需字段）。
type adminUserDTO struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Balance   float64   `json:"balance"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// GetUserByID 用管理密钥拉取单个用户信息（抽奖报名者详情）。
func (m *Manager) GetUserByID(cfg ClientConfig, userID int64) (*Sub2APIUserInfo, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("用户 ID 不能为空")
	}
	endpoint, err := adminURL(cfg.BaseURL, fmt.Sprintf(adminUserByIDPathFmt, userID))
	if err != nil {
		return nil, err
	}
	u, err := GetEnvelope[adminUserDTO](endpoint.String(), adminHeaders(cfg))
	if err != nil {
		return nil, err
	}
	return &Sub2APIUserInfo{
		ID:        u.ID,
		Username:  strings.TrimSpace(u.Username),
		Email:     strings.TrimSpace(u.Email),
		Role:      u.Role,
		Balance:   u.Balance,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
	}, nil
}

// balanceAdjustBody 对齐 POST /admin/users/:id/balance。
type balanceAdjustBody struct {
	Balance   float64 `json:"balance"`
	Operation string  `json:"operation"`
	Notes     string  `json:"notes"`
}

// AddUserBalance 给用户加余额（派奖）。
func (m *Manager) AddUserBalance(cfg ClientConfig, userID int64, amount float64, notes, idemKey string) error {
	if userID <= 0 {
		return fmt.Errorf("用户 ID 不能为空")
	}
	endpoint, err := adminURL(cfg.BaseURL, fmt.Sprintf(adminUserBalancePathFmt, userID))
	if err != nil {
		return err
	}
	headers := adminHeaders(cfg)
	if idemKey != "" {
		headers["Idempotency-Key"] = idemKey
	}
	// 响应 data 可忽略，只关心是否成功（信封 code）。
	_, err = PostEnvelope[map[string]any](endpoint.String(), headers, balanceAdjustBody{
		Balance:   amount,
		Operation: "add",
		Notes:     notes,
	})
	if err != nil {
		return fmt.Errorf("派奖失败：%w", err)
	}
	return nil
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
