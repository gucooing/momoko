// Package version 持有当前编译版本号，并实现「检查更新」逻辑（查询 GitHub 最新发行版并比较）。
// 版本号由构建期 ldflags 注入 main.Version，启动时经 Set 写入本包。
package version

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

// ReleasesURL 是项目发行版列表页（无可用最新版信息时的兜底跳转地址）。
const (
	ReleasesURL      = "https://github.com/gucooing/momoko/releases"
	latestReleaseAPI = "https://api.github.com/repos/gucooing/momoko/releases/latest"
)

// current 为当前运行版本（构建期注入）；为空表示本地/开发构建。
var current string

// Set 写入当前版本（启动时由 main 注入 main.Version）。
func Set(v string) { current = strings.TrimSpace(v) }

// Current 返回当前版本；未注入时返回 "dev"。
func Current() string {
	if current == "" {
		return "dev"
	}
	return current
}

// Release 是 GitHub 发行版的精简视图。
type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	HTMLURL     string    `json:"html_url"`
	Body        string    `json:"body"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
}

// CheckLatest 查询 GitHub 最新（非预发布、非草稿）发行版。
func CheckLatest(ctx context.Context) (*Release, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "momoko-update-checker")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("尚无可用发行版")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("查询更新失败：GitHub 返回状态 %d", resp.StatusCode)
	}
	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// HasUpdate 比较当前版本与最新 tag（按 semver）；任一非法（如开发版）时无法判定，返回 false。
func HasUpdate(latestTag string) bool {
	cur := normalize(Current())
	latest := normalize(latestTag)
	if !semver.IsValid(cur) || !semver.IsValid(latest) {
		return false
	}
	return semver.Compare(latest, cur) > 0
}

// normalize 规整为 semver 可比较形式（确保有 v 前缀）。
func normalize(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}
