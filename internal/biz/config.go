package biz

import (
	"context"
	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/common"
	"momoko/pkg/response"
	"net/mail"
	"strings"
)

type ConfigRepo interface {
	LoginConfig(ctx context.Context) (*v1.LoginConfig, error)
	UpdateLoginConfig(ctx context.Context, req *v1.UpdateLoginConfigRequest) (*v1.LoginConfig, error)
	EmailConfig(ctx context.Context) (*v1.EmailConfig, error)
	UpdateEmailConfig(ctx context.Context, req *v1.UpdateEmailConfigRequest) (*v1.EmailConfig, error)
	UpdateEmailTemplate(ctx context.Context, req *v1.UpdateEmailTemplateRequest) (*gen.EmailTemplate, error)
	EmailTemplate(ctx context.Context, templateType v1.EmailTemplateType) (*gen.EmailTemplate, error)
	DockerConfig(ctx context.Context) (*v1.DockerConfigInfo, error)
	UpdateDockerConfig(ctx context.Context, req *v1.DockerConfigInfo) (*v1.DockerConfigInfo, error)
	Get(ctx context.Context, key common.ConfigKey) (string, error)
	BatchUpdate(ctx context.Context, configs map[common.ConfigKey]string) error
}

type ConfigUsecase struct {
	config ConfigRepo
}

func NewConfigUsecase(config ConfigRepo) *ConfigUsecase {
	return &ConfigUsecase{
		config: config,
	}
}

func (c *ConfigUsecase) LoginConfig(ctx context.Context) (*v1.LoginConfig, error) {
	config, err := c.config.LoginConfig(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return config, nil
}

func (c *ConfigUsecase) UpdateLoginConfig(ctx context.Context, req *v1.UpdateLoginConfigRequest) (*v1.LoginConfig, error) {
	updated, err := c.config.UpdateLoginConfig(ctx, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return updated, nil
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
		if req.UseTls {
			req.Port = 465
		} else {
			req.Port = 25
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
		Subject:  data.Subject,
		Template: data.Template,
		Type:     emailTemplateTypeFromString(data.Type),
	}
}

func isEmailTemplateTypeValid(data v1.EmailTemplateType) bool {
	switch data {
	case v1.EmailTemplateType_EmailTemplateType_Register:
		return true
	case v1.EmailTemplateType_EmailTemplateType_Login:
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
