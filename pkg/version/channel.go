package version

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const releasesAPI = "https://api.github.com/repos/gucooing/momoko/releases?per_page=10"

// CheckLatestChannel 按更新通道查询最新发行版：beta=false 只看正式版（同 CheckLatest），
// beta=true 时包含预发布版本，取最新发布的一个（跳过草稿）。
func CheckLatestChannel(ctx context.Context, beta bool) (*Release, error) {
	if !beta {
		return CheckLatest(ctx)
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesAPI, nil)
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
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("查询更新失败：GitHub 返回状态 %d", resp.StatusCode)
	}
	var list []struct {
		Release
		Draft bool `json:"draft"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}
	var latest *Release
	for i := range list {
		if list[i].Draft {
			continue
		}
		if latest == nil || list[i].PublishedAt.After(latest.PublishedAt) {
			rel := list[i].Release
			latest = &rel
		}
	}
	if latest == nil {
		return nil, errors.New("尚无可用发行版")
	}
	return latest, nil
}
