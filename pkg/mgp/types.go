package mgp

import (
	"context"
	"encoding/json"
)

// Manifest 是 GET /manifest 中各游戏共有的部分：协议版本、游戏与服务端版本；其余字段由游戏模块自己解析（Raw）。
type Manifest struct {
	Protocol int `json:"protocol"`
	Game     struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Theme string `json:"theme"`
		Icon  string `json:"icon"`
	} `json:"game"`
	Server struct {
		Version  string          `json:"version"`
		Resource json.RawMessage `json:"resource"`
	} `json:"server"`
	Raw json.RawMessage `json:"-"`
}

// ParseManifest 解析 manifest 原始 JSON。
func ParseManifest(raw json.RawMessage) (*Manifest, error) {
	m := &Manifest{}
	if err := json.Unmarshal(raw, m); err != nil {
		return nil, err
	}
	m.Raw = raw
	return m, nil
}

// Status 是 GET /status 中 Momoko 需要理解的部分；完整内容保存在 Raw。
type Status struct {
	Online        int    `json:"online"`
	OnlinePeak    int    `json:"onlinePeak"`
	Registered    int64  `json:"registered"`
	NewToday      int64  `json:"newToday"`
	UptimeSeconds int64  `json:"uptimeSeconds"`
	Now           int64  `json:"now"`
	Version       string `json:"version"`
	Maintenance   struct {
		Enabled bool `json:"enabled"`
	} `json:"maintenance"`
	Health []struct {
		Name string `json:"name"`
		OK   bool   `json:"ok"`
	} `json:"health"`
	Raw json.RawMessage `json:"-"`
}

// ParseStatus 解析 status 原始 JSON。
func ParseStatus(raw json.RawMessage) (*Status, error) {
	s := &Status{}
	if err := json.Unmarshal(raw, s); err != nil {
		return nil, err
	}
	s.Raw = raw
	return s, nil
}

// Healthy 判断全部依赖是否健康。
func (s *Status) Healthy() bool {
	if s == nil {
		return false
	}
	for _, h := range s.Health {
		if !h.OK {
			return false
		}
	}
	return true
}

// Manifest 读取 GET /manifest。
func (c *Client) Manifest(ctx context.Context) (*Manifest, error) {
	raw, err := c.Get(ctx, "/manifest", nil)
	if err != nil {
		return nil, err
	}
	return ParseManifest(raw)
}

// Status 读取 GET /status。
func (c *Client) Status(ctx context.Context) (*Status, error) {
	raw, err := c.Get(ctx, "/status", nil)
	if err != nil {
		return nil, err
	}
	return ParseStatus(raw)
}
