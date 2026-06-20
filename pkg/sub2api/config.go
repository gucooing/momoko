package sub2api

import (
	"context"
	"encoding/json"
	"strings"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/common"

	"google.golang.org/protobuf/encoding/protojson"
)

// DefaultConfig 返回带默认值的配置。
func DefaultConfig() *v1.Sub2APIConfig {
	return &v1.Sub2APIConfig{
		HomeEnabled:         false,
		SyncEnabled:         true,
		Title:               "Sub2API",
		Subtitle:            "统一订阅转换与模型调用看板",
		SyncIntervalMinutes: DefaultSyncIntervalMinutes,
		HistoryDays:         DefaultHistoryDays,
		PageSize:            DefaultPageSize,
	}
}

// LoadConfig 从 KV 配置读取并规范化 Sub2API 配置。
func LoadConfig(ctx context.Context, store ConfigStore) (*v1.Sub2APIConfig, error) {
	raw, err := store.Get(ctx, common.ConfigSub2APIConfig)
	if err != nil {
		return nil, err
	}
	cfg := DefaultConfig()
	if strings.TrimSpace(raw) != "" {
		parsed := &v1.Sub2APIConfig{}
		if err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal([]byte(raw), parsed); err != nil {
			return nil, err
		}
		cfg = parsed
	}
	NormalizeConfig(cfg)
	return cfg, nil
}

// SaveConfig 规范化并持久化 Sub2API 配置。
func SaveConfig(ctx context.Context, store ConfigStore, cfg *v1.Sub2APIConfig) error {
	NormalizeConfig(cfg)
	data, err := protojson.Marshal(cfg)
	if err != nil {
		return err
	}
	return store.BatchUpdate(ctx, map[common.ConfigKey]string{
		common.ConfigSub2APIConfig: string(data),
	})
}

// NormalizeConfig 清洗字段并将数值约束在合理范围内。
func NormalizeConfig(cfg *v1.Sub2APIConfig) {
	if cfg == nil {
		return
	}
	cfg.BaseUrl = strings.TrimRight(strings.TrimSpace(cfg.BaseUrl), "/")
	cfg.AdminApiKey = strings.TrimSpace(cfg.AdminApiKey)
	cfg.ConsoleUrl = strings.TrimRight(strings.TrimSpace(cfg.ConsoleUrl), "/")
	cfg.Title = strings.TrimSpace(cfg.Title)
	cfg.Subtitle = strings.TrimSpace(cfg.Subtitle)
	cfg.Introduction = strings.TrimSpace(cfg.Introduction)
	if cfg.Title == "" {
		cfg.Title = "Sub2API"
	}
	cfg.SyncIntervalMinutes = clampInt32(cfg.SyncIntervalMinutes, MinSyncIntervalMinutes, MaxSyncIntervalMinutes, DefaultSyncIntervalMinutes)
	cfg.HistoryDays = clampInt32(cfg.HistoryDays, MinHistoryDays, MaxHistoryDays, DefaultHistoryDays)
	cfg.PageSize = clampInt32(cfg.PageSize, MinPageSize, MaxPageSize, DefaultPageSize)
}

// ClientConfigFromConfig 提取调用 Sub2API 所需的连接信息。
func ClientConfigFromConfig(cfg *v1.Sub2APIConfig) ClientConfig {
	if cfg == nil {
		return ClientConfig{}
	}
	return NewClientConfig(cfg.BaseUrl, cfg.AdminApiKey)
}

// LoadSyncState 读取最近一次同步状态。
func LoadSyncState(ctx context.Context, store ConfigStore) (SyncState, error) {
	raw, err := store.Get(ctx, common.ConfigSub2APISyncState)
	if err != nil {
		return SyncState{}, err
	}
	state := SyncState{Status: SyncStatusIdle}
	if strings.TrimSpace(raw) != "" {
		if err = json.Unmarshal([]byte(raw), &state); err != nil {
			return SyncState{}, err
		}
	}
	if state.Status == "" {
		state.Status = SyncStatusIdle
	}
	return state, nil
}

// SaveSyncState 持久化同步状态。
func SaveSyncState(ctx context.Context, store ConfigStore, state SyncState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return store.BatchUpdate(ctx, map[common.ConfigKey]string{
		common.ConfigSub2APISyncState: string(data),
	})
}

// ResetSyncState 清空同步状态。
func ResetSyncState(ctx context.Context, store ConfigStore) error {
	return SaveSyncState(ctx, store, SyncState{Status: SyncStatusIdle})
}

func clampInt32(value, min, max, def int32) int32 {
	if value <= 0 {
		return def
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
