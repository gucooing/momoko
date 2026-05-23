package biz

import (
	"context"
	"fmt"
	"strconv"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/common"
)

type ConfigRepo interface {
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
	registerEnabled, err := c.getBoolConfig(ctx, common.ConfigLoginRegisterEnabled)
	if err != nil {
		return nil, err
	}
	usernameLoginEnabled, err := c.getBoolConfig(ctx, common.ConfigLoginUsernameLoginEnabled)
	if err != nil {
		return nil, err
	}
	emailLoginEnabled, err := c.getBoolConfig(ctx, common.ConfigLoginEmailLoginEnabled)
	if err != nil {
		return nil, err
	}

	return &v1.LoginConfig{
		RegisterEnabled:      registerEnabled,
		UsernameLoginEnabled: usernameLoginEnabled,
		EmailLoginEnabled:    emailLoginEnabled,
	}, nil
}

func (c *ConfigUsecase) UpdateLoginConfig(ctx context.Context, req *v1.UpdateLoginConfigRequest) (*v1.LoginConfig, error) {
	configs := map[common.ConfigKey]string{
		common.ConfigLoginRegisterEnabled:      strconv.FormatBool(req.RegisterEnabled),
		common.ConfigLoginUsernameLoginEnabled: strconv.FormatBool(req.UsernameLoginEnabled),
		common.ConfigLoginEmailLoginEnabled:    strconv.FormatBool(req.EmailLoginEnabled),
	}
	if err := c.config.BatchUpdate(ctx, configs); err != nil {
		return nil, ErrSystem(err)
	}
	return c.LoginConfig(ctx)
}

func (c *ConfigUsecase) getBoolConfig(ctx context.Context, key common.ConfigKey) (bool, error) {
	value, err := c.config.Get(ctx, key)
	if err != nil {
		return false, ErrSystem(err)
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return false, ErrSystem(fmt.Errorf("invalid bool config %s=%q: %w", key, value, err))
	}
	return enabled, nil
}
