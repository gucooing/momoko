package data

import (
	"context"
	"strconv"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/common"
)

// SiteConfig 站点常规配置：名称 / 对外地址 / 默认语言 / 版本更新偏好，附带只读的最近检查与最近修改记录。
func (c *ConfigRepo) SiteConfig(ctx context.Context) (*v1.SiteConfig, error) {
	name, err := c.getStringConfig(ctx, common.ConfigSiteName)
	if err != nil {
		return nil, err
	}
	externalURL, err := c.getStringConfig(ctx, common.ConfigSiteExternalURL)
	if err != nil {
		return nil, err
	}
	lang, err := c.getStringConfig(ctx, common.ConfigSiteDefaultLanguage)
	if err != nil {
		return nil, err
	}
	autoCheck, err := c.getBoolConfig(ctx, common.ConfigUpdateAutoCheck)
	if err != nil {
		return nil, err
	}
	channelRaw, err := c.getStringConfig(ctx, common.ConfigUpdateChannel)
	if err != nil {
		return nil, err
	}
	cfg := &v1.SiteConfig{
		SiteName:        name,
		ExternalUrl:     externalURL,
		DefaultLanguage: lang,
		AutoCheckUpdate: autoCheck,
		UpdateChannel:   v1.UpdateChannel(v1.UpdateChannel_value[channelRaw]),
	}
	lastCheck := new(v1.UpdateCheckResult)
	if ok, err := c.getProtoConfig(ctx, common.ConfigUpdateLastCheck, lastCheck); err != nil {
		return nil, err
	} else if ok {
		cfg.LastUpdateCheck = lastCheck
	}
	audit := new(v1.SettingsAudit)
	if ok, err := c.getProtoConfig(ctx, common.ConfigSiteSettingsAudit, audit); err != nil {
		return nil, err
	} else if ok {
		cfg.LastModified = audit
	}
	return cfg, nil
}

func (c *ConfigRepo) UpdateSiteConfig(ctx context.Context, req *v1.UpdateSiteConfigRequest) (*v1.SiteConfig, error) {
	configs := map[common.ConfigKey]string{
		common.ConfigSiteName:            req.SiteName,
		common.ConfigSiteExternalURL:     req.ExternalUrl,
		common.ConfigSiteDefaultLanguage: req.DefaultLanguage,
		common.ConfigUpdateAutoCheck:     strconv.FormatBool(req.AutoCheckUpdate),
		common.ConfigUpdateChannel:       req.UpdateChannel.String(),
	}
	if err := c.BatchUpdate(ctx, configs); err != nil {
		return nil, err
	}
	return c.SiteConfig(ctx)
}

// SaveUpdateCheck 记录最近一次检查更新结果（手动或每日自动）。
func (c *ConfigRepo) SaveUpdateCheck(ctx context.Context, result *v1.UpdateCheckResult) error {
	return c.saveProtoConfig(ctx, common.ConfigUpdateLastCheck, result)
}

// SaveSettingsAudit 记录系统设置最近一次修改（时间 + 修改人）。
func (c *ConfigRepo) SaveSettingsAudit(ctx context.Context, audit *v1.SettingsAudit) error {
	return c.saveProtoConfig(ctx, common.ConfigSiteSettingsAudit, audit)
}

func (c *ConfigRepo) getProtoConfig(ctx context.Context, key common.ConfigKey, msg proto.Message) (bool, error) {
	raw, err := c.getStringConfig(ctx, key)
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(raw) == "" {
		return false, nil
	}
	if err := protojson.Unmarshal([]byte(raw), msg); err != nil {
		// 只读展示数据，损坏时当作没有
		return false, nil
	}
	return true, nil
}

func (c *ConfigRepo) saveProtoConfig(ctx context.Context, key common.ConfigKey, msg proto.Message) error {
	raw, err := protojson.Marshal(msg)
	if err != nil {
		return err
	}
	return c.BatchUpdate(ctx, map[common.ConfigKey]string{key: string(raw)})
}

// SiteTheme 全站默认外观主题；没设置过（或内容损坏）返回 nil。
func (c *ConfigRepo) SiteTheme(ctx context.Context) (*v1.SiteTheme, error) {
	theme := new(v1.SiteTheme)
	ok, err := c.getProtoConfig(ctx, common.ConfigSiteTheme, theme)
	if err != nil || !ok {
		return nil, err
	}
	return theme, nil
}

// SaveSiteTheme 保存全站默认外观主题；nil 表示取消。
func (c *ConfigRepo) SaveSiteTheme(ctx context.Context, theme *v1.SiteTheme) error {
	if theme == nil {
		return c.BatchUpdate(ctx, map[common.ConfigKey]string{common.ConfigSiteTheme: ""})
	}
	return c.saveProtoConfig(ctx, common.ConfigSiteTheme, theme)
}

// AvatarStorage 头像储存设置（Secret Key 为加密态）；没配置过返回空配置。
func (c *ConfigRepo) AvatarStorage(ctx context.Context) (*v1.AvatarStorageConfig, error) {
	cfg := new(v1.AvatarStorageConfig)
	if _, err := c.getProtoConfig(ctx, common.ConfigStorageAvatar, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// SaveAvatarStorage 保存头像储存设置（调用方负责把 Secret Key 加密后再传入）。
func (c *ConfigRepo) SaveAvatarStorage(ctx context.Context, cfg *v1.AvatarStorageConfig) error {
	return c.saveProtoConfig(ctx, common.ConfigStorageAvatar, cfg)
}
