package biz

import (
	"context"
	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/auth"
	"momoko/pkg/common"
	"momoko/pkg/response"
	"net/mail"
	"net/url"
	"sort"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type ConfigRepo interface {
	LoginConfig(ctx context.Context) (*v1.LoginConfig, error)
	UpdateLoginConfig(ctx context.Context, req *v1.UpdateLoginConfigRequest) (*v1.LoginConfig, error)
	EmailConfig(ctx context.Context) (*v1.EmailConfig, error)
	UpdateEmailConfig(ctx context.Context, req *v1.UpdateEmailConfigRequest) (*v1.EmailConfig, error)
	UpdateEmailTemplate(ctx context.Context, req *v1.UpdateEmailTemplateRequest) (*gen.EmailTemplate, error)
	EmailTemplate(ctx context.Context, templateType v1.EmailTemplateType) (*gen.EmailTemplate, error)
	ListEmailTemplates(ctx context.Context) ([]*gen.EmailTemplate, error)
	SaveEmailTestResult(ctx context.Context, result *v1.EmailTestResult) error
	SiteConfig(ctx context.Context) (*v1.SiteConfig, error)
	UpdateSiteConfig(ctx context.Context, req *v1.UpdateSiteConfigRequest) (*v1.SiteConfig, error)
	SaveUpdateCheck(ctx context.Context, result *v1.UpdateCheckResult) error
	SaveSettingsAudit(ctx context.Context, audit *v1.SettingsAudit) error
	DockerConfig(ctx context.Context) (*v1.DockerConfigInfo, error)
	UpdateDockerConfig(ctx context.Context, req *v1.DockerConfigInfo) (*v1.DockerConfigInfo, error)
	FrpsConfig(ctx context.Context) (*v1.FrpsConfig, error)
	UpdateFrpsConfig(ctx context.Context, req *v1.FrpsConfig) (*v1.FrpsConfig, error)
	Get(ctx context.Context, key common.ConfigKey) (string, error)
	BatchUpdate(ctx context.Context, configs map[common.ConfigKey]string) error
}

type ConfigUsecase struct {
	config ConfigRepo
}

func NewConfigUsecase(config ConfigRepo) *ConfigUsecase {
	uc := &ConfigUsecase{
		config: config,
	}
	// 启动时按已保存的登录设置应用令牌有效期
	if cfg, err := config.LoginConfig(context.Background()); err == nil {
		applyTokenTTL(cfg)
	}
	return uc
}

// applyTokenTTL 把登录设置里的令牌有效期写入签发逻辑。
func applyTokenTTL(cfg *v1.LoginConfig) {
	auth.SetTokenTTL(
		time.Duration(cfg.GetAccessTokenTtlMinutes())*time.Minute,
		time.Duration(cfg.GetRefreshTokenTtlDays())*24*time.Hour,
	)
}

func (c *ConfigUsecase) LoginConfig(ctx context.Context) (*v1.LoginConfig, error) {
	config, err := c.config.LoginConfig(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return config, nil
}

func (c *ConfigUsecase) UpdateLoginConfig(ctx context.Context, req *v1.UpdateLoginConfigRequest) (*v1.LoginConfig, error) {
	if req.AccessTokenTtlMinutes < 5 || req.AccessTokenTtlMinutes > 24*60 {
		return nil, response.BadRequest(400, "访问令牌有效期需在 5 到 1440 分钟之间")
	}
	if req.RefreshTokenTtlDays < 1 || req.RefreshTokenTtlDays > 365 {
		return nil, response.BadRequest(400, "刷新令牌有效期需在 1 到 365 天之间")
	}
	if req.RefreshTokenTtlDays*24*60 < req.AccessTokenTtlMinutes {
		return nil, response.BadRequest(400, "刷新令牌有效期不能短于访问令牌有效期")
	}
	if req.LockoutThreshold < 0 || req.LockoutThreshold > 100 {
		return nil, response.BadRequest(400, "登录失败锁定次数需在 0 到 100 之间")
	}
	if req.LockoutThreshold > 0 && (req.LockoutMinutes < 1 || req.LockoutMinutes > 24*60) {
		return nil, response.BadRequest(400, "锁定时长需在 1 到 1440 分钟之间")
	}
	if req.LockoutMinutes <= 0 {
		req.LockoutMinutes = 15
	}
	updated, err := c.config.UpdateLoginConfig(ctx, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	applyTokenTTL(updated)
	return updated, nil
}

// siteLanguages 站点默认语言可选值（auto 跟随浏览器）。
var siteLanguages = map[string]struct{}{"auto": {}, "zh-CN": {}, "zh-HK": {}, "en-US": {}}

func (c *ConfigUsecase) SiteConfig(ctx context.Context) (*v1.SiteConfig, error) {
	cfg, err := c.config.SiteConfig(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return cfg, nil
}

func (c *ConfigUsecase) UpdateSiteConfig(ctx context.Context, req *v1.UpdateSiteConfigRequest) (*v1.SiteConfig, error) {
	req.SiteName = strings.TrimSpace(req.SiteName)
	req.ExternalUrl = strings.TrimRight(strings.TrimSpace(req.ExternalUrl), "/")
	if req.SiteName == "" {
		return nil, response.BadRequest(400, "站点名称不能为空")
	}
	if len([]rune(req.SiteName)) > 40 {
		return nil, response.BadRequest(400, "站点名称不能超过 40 个字")
	}
	if req.ExternalUrl != "" {
		u, err := url.Parse(req.ExternalUrl)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return nil, response.BadRequest(400, "对外地址需是 http:// 或 https:// 开头的完整地址")
		}
	}
	if _, ok := siteLanguages[req.DefaultLanguage]; !ok {
		req.DefaultLanguage = "auto"
	}
	if _, ok := v1.UpdateChannel_name[int32(req.UpdateChannel)]; !ok {
		req.UpdateChannel = v1.UpdateChannel_UpdateChannel_Stable
	}
	cfg, err := c.config.UpdateSiteConfig(ctx, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return cfg, nil
}

// TouchSettings 记录系统设置最近一次修改（失败只影响展示）。
func (c *ConfigUsecase) TouchSettings(ctx context.Context, userName string) {
	_ = c.config.SaveSettingsAudit(context.WithoutCancel(ctx), &v1.SettingsAudit{
		Time:     timestamppb.Now(),
		UserName: userName,
	})
}

func (c *ConfigUsecase) EmailConfig(ctx context.Context) (*v1.EmailConfig, error) {
	config, err := c.config.EmailConfig(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return config, nil
}

func (c *ConfigUsecase) UpdateEmailConfig(ctx context.Context, req *v1.UpdateEmailConfigRequest) (*v1.EmailConfig, error) {
	if err := validateEmailConfig(req); err != nil {
		return nil, err
	}

	updated, err := c.config.UpdateEmailConfig(ctx, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return updated, nil
}

func (c *ConfigUsecase) UpdateEmailTemplate(ctx context.Context, req *v1.UpdateEmailTemplateRequest) (*v1.EmailTemplate, error) {
	if err := validateEmailTemplate(req); err != nil {
		return nil, err
	}

	updated, err := c.config.UpdateEmailTemplate(ctx, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toEmailTemplate(updated), nil
}

func (c *ConfigUsecase) EmailTemplate(ctx context.Context, templateType v1.EmailTemplateType) (*v1.EmailTemplate, error) {
	if !isEmailTemplateTypeValid(templateType) {
		return nil, ErrEmailTemplateType
	}

	template, err := c.config.EmailTemplate(ctx, templateType)
	if err != nil {
		if gen.IsNotFound(err) {
			return new(v1.EmailTemplate), nil
		}
		return nil, ErrSystem(err)
	}
	return toEmailTemplate(template), nil
}

func validateEmailConfig(req *v1.UpdateEmailConfigRequest) error {
	req.Host = strings.TrimSpace(req.Host)
	req.Username = strings.TrimSpace(req.Username)
	req.From = strings.TrimSpace(req.From)
	req.FromName = strings.TrimSpace(req.FromName)
	if req.Port <= 0 {
		switch req.Encryption {
		case v1.EmailEncryption_EmailEncryption_STARTTLS:
			req.Port = 587
		case v1.EmailEncryption_EmailEncryption_NONE:
			req.Port = 25
		default:
			req.Port = 465
		}
	}
	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = 10
	}
	if req.CcsN <= 0 {
		req.CcsN = 5
	}
	if req.Enabled {
		if req.Host == "" {
			return response.BadRequest(400, "邮件服务地址不能为空")
		}
		if req.From == "" {
			return response.BadRequest(400, "发件邮箱不能为空")
		}
		if _, err := mail.ParseAddress(req.From); err != nil {
			return response.BadRequest(400, "发件邮箱格式不正确")
		}
	}
	return nil
}

func validateEmailTemplate(req *v1.UpdateEmailTemplateRequest) error {
	req.Subject = strings.TrimSpace(req.Subject)
	req.Template = strings.TrimSpace(req.Template)

	if !isEmailTemplateTypeValid(req.Type) {
		return ErrEmailTemplateType
	}
	if req.Subject == "" {
		return ErrEmailTemplateSubject
	}
	if req.Template == "" {
		return ErrEmailTemplateContent
	}
	return nil
}

func toEmailTemplate(data *gen.EmailTemplate) *v1.EmailTemplate {
	if data == nil {
		return nil
	}
	return &v1.EmailTemplate{
		Subject:    data.Subject,
		Template:   data.Template,
		Type:       emailTemplateTypeFromString(data.Type),
		Customized: true,
		UpdateTime: timestamppb.New(data.UpdateTime),
	}
}

// ListEmailTemplates 按模板类型顺序返回全部模板；未自定义的类型返回空模板（customized=false）。
func (c *ConfigUsecase) ListEmailTemplates(ctx context.Context) ([]*v1.EmailTemplate, error) {
	saved, err := c.config.ListEmailTemplates(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	byType := make(map[v1.EmailTemplateType]*gen.EmailTemplate, len(saved))
	for _, item := range saved {
		byType[emailTemplateTypeFromString(item.Type)] = item
	}
	types := make([]int, 0, len(v1.EmailTemplateType_name))
	for value := range v1.EmailTemplateType_name {
		types = append(types, int(value))
	}
	sort.Ints(types)
	out := make([]*v1.EmailTemplate, 0, len(types))
	for _, value := range types {
		typ := v1.EmailTemplateType(value)
		if !isEmailTemplateTypeValid(typ) {
			continue
		}
		if item, ok := byType[typ]; ok {
			out = append(out, toEmailTemplate(item))
			continue
		}
		out = append(out, &v1.EmailTemplate{Type: typ})
	}
	return out, nil
}

// SaveEmailTestResult 记录最近一次测试发送结果；失败只影响展示，不影响测试本身。
func (c *ConfigUsecase) SaveEmailTestResult(ctx context.Context, result *v1.EmailTestResult) {
	_ = c.config.SaveEmailTestResult(ctx, result)
}

func isEmailTemplateTypeValid(data v1.EmailTemplateType) bool {
	switch data {
	case v1.EmailTemplateType_EmailTemplateType_Register:
		return true
	case v1.EmailTemplateType_EmailTemplateType_Login:
		return true
	case v1.EmailTemplateType_EmailTemplateType_ResetPassword:
		return true
	default:
		return false
	}
}

func emailTemplateTypeFromString(data string) v1.EmailTemplateType {
	if value, ok := v1.EmailTemplateType_value[data]; ok {
		return v1.EmailTemplateType(value)
	}
	return v1.EmailTemplateType_EmailTemplateType_Register
}
