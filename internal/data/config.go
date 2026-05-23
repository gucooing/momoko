package data

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/systemconfig"
	"momoko/pkg/common"
	"momoko/pkg/response"
)

type ConfigRepo struct {
	data *Data

	sync  sync.RWMutex
	cache map[common.ConfigKey]string
}

func NewConfigRepo(data *Data) biz.ConfigRepo {
	return &ConfigRepo{
		data:  data,
		sync:  sync.RWMutex{},
		cache: make(map[common.ConfigKey]string, common.ConfigsLen()),
	}
}

func (c *ConfigRepo) cacheGet(key common.ConfigKey) (string, bool) {
	c.sync.RLock()
	defer c.sync.RUnlock()
	v, ok := c.cache[key]
	return v, ok
}

func (c *ConfigRepo) Get(ctx context.Context, key common.ConfigKey) (string, error) {
	if value, ok := c.cacheGet(key); ok {
		return value, nil
	}

	c.sync.Lock()
	defer c.sync.Unlock()
	value, err := c.data.db.SystemConfig.Query().
		Where(systemconfig.KeyEQ(key.String())).
		Select(systemconfig.FieldValue).
		String(ctx)
	if err == nil {
		c.cache[key] = value
		return value, nil
	}
	if !gen.IsNotFound(err) {
		return "", err
	}

	value, err = c.GetWithDefault(ctx, key)
	if err != nil {
		return "", err
	}
	c.cache[key] = value
	return value, nil
}

func (c *ConfigRepo) LoginConfig(ctx context.Context) (*v1.LoginConfig, error) {
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

func (c *ConfigRepo) UpdateLoginConfig(ctx context.Context, req *v1.UpdateLoginConfigRequest) (*v1.LoginConfig, error) {
	configs := map[common.ConfigKey]string{
		common.ConfigLoginRegisterEnabled:      strconv.FormatBool(req.RegisterEnabled),
		common.ConfigLoginUsernameLoginEnabled: strconv.FormatBool(req.UsernameLoginEnabled),
		common.ConfigLoginEmailLoginEnabled:    strconv.FormatBool(req.EmailLoginEnabled),
	}
	if err := c.BatchUpdate(ctx, configs); err != nil {
		return nil, err
	}
	return c.LoginConfig(ctx)
}

func (c *ConfigRepo) EmailConfig(ctx context.Context) (*v1.EmailConfig, error) {
	enabled, err := c.getBoolConfig(ctx, common.ConfigEmailEnabled)
	if err != nil {
		return nil, err
	}
	host, err := c.getStringConfig(ctx, common.ConfigEmailHost)
	if err != nil {
		return nil, err
	}
	port, err := c.getInt32Config(ctx, common.ConfigEmailPort)
	if err != nil {
		return nil, err
	}
	username, err := c.getStringConfig(ctx, common.ConfigEmailUsername)
	if err != nil {
		return nil, err
	}
	password, err := c.getStringConfig(ctx, common.ConfigEmailPassword)
	if err != nil {
		return nil, err
	}
	from, err := c.getStringConfig(ctx, common.ConfigEmailFrom)
	if err != nil {
		return nil, err
	}
	fromName, err := c.getStringConfig(ctx, common.ConfigEmailFromName)
	if err != nil {
		return nil, err
	}
	useTLS, err := c.getBoolConfig(ctx, common.ConfigEmailUseTLS)
	if err != nil {
		return nil, err
	}
	timeoutSeconds, err := c.getInt32Config(ctx, common.ConfigEmailTimeoutSeconds)
	if err != nil {
		return nil, err
	}
	ccsN, err := c.getInt32Config(ctx, common.ConfigEmailCcsN)
	if err != nil {
		return nil, err
	}

	return &v1.EmailConfig{
		Enabled:        enabled,
		Host:           host,
		Port:           port,
		Username:       username,
		Password:       password,
		From:           from,
		FromName:       fromName,
		UseTls:         useTLS,
		TimeoutSeconds: timeoutSeconds,
		CcsN:           ccsN,
	}, nil
}

func (c *ConfigRepo) UpdateEmailConfig(ctx context.Context, req *v1.UpdateEmailConfigRequest) (*v1.EmailConfig, error) {
	configs := map[common.ConfigKey]string{
		common.ConfigEmailEnabled:        strconv.FormatBool(req.Enabled),
		common.ConfigEmailHost:           req.Host,
		common.ConfigEmailPort:           strconv.FormatInt(int64(req.Port), 10),
		common.ConfigEmailUsername:       req.Username,
		common.ConfigEmailPassword:       req.Password,
		common.ConfigEmailFrom:           req.From,
		common.ConfigEmailFromName:       req.FromName,
		common.ConfigEmailUseTLS:         strconv.FormatBool(req.UseTls),
		common.ConfigEmailTimeoutSeconds: strconv.FormatInt(int64(req.TimeoutSeconds), 10),
		common.ConfigEmailCcsN:           strconv.FormatInt(int64(req.CcsN), 10),
	}
	if err := c.BatchUpdate(ctx, configs); err != nil {
		return nil, err
	}
	return c.EmailConfig(ctx)
}

func (c *ConfigRepo) getBoolConfig(ctx context.Context, key common.ConfigKey) (bool, error) {
	value, err := c.Get(ctx, key)
	if err != nil {
		return false, err
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid bool config %s=%q: %w", key, value, err)
	}
	return enabled, nil
}

func (c *ConfigRepo) getInt32Config(ctx context.Context, key common.ConfigKey) (int32, error) {
	value, err := c.Get(ctx, key)
	if err != nil {
		return 0, err
	}
	number, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid int config %s=%q: %w", key, value, err)
	}
	return int32(number), nil
}

func (c *ConfigRepo) getStringConfig(ctx context.Context, key common.ConfigKey) (string, error) {
	return c.Get(ctx, key)
}

func (c *ConfigRepo) GetWithDefault(ctx context.Context, key common.ConfigKey) (string, error) {
	defaultValue, ok := common.ConfigDefault(key)
	if !ok {
		return "", response.BadRequest(500, "配置不存在")
	}

	err := c.data.db.SystemConfig.Create().
		SetKey(key.String()).
		SetValue(defaultValue).
		OnConflictColumns(systemconfig.FieldKey).
		UpdateNewValues().
		Exec(ctx)
	if err != nil {
		return "", err
	}
	return defaultValue, nil
}

func (c *ConfigRepo) Update(ctx context.Context, key common.ConfigKey, value string) error {
	c.sync.Lock()
	defer c.sync.Unlock()

	n, err := c.data.db.SystemConfig.Update().
		Where(systemconfig.KeyEQ(key.String())).
		SetValue(value).
		Save(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return response.BadRequest(500, "更新配置失败")
	}
	c.cache[key] = value

	return nil
}

func (c *ConfigRepo) BatchUpdate(ctx context.Context, configs map[common.ConfigKey]string) error {
	if len(configs) == 0 {
		return nil
	}

	c.sync.Lock()
	defer c.sync.Unlock()

	builders := make([]*gen.SystemConfigCreate, 0, len(configs))
	for key, value := range configs {
		builders = append(builders,
			c.data.db.SystemConfig.
				Create().
				SetKey(key.String()).
				SetValue(value),
		)
	}

	err := c.data.db.SystemConfig.
		CreateBulk(builders...).
		OnConflictColumns(systemconfig.FieldKey).
		UpdateNewValues().
		Exec(ctx)
	if err != nil {
		return err
	}

	for key, value := range configs {
		c.cache[key] = value
	}

	return nil
}

func (c *ConfigRepo) Delete(ctx context.Context, key common.ConfigKey) error {
	c.sync.Lock()
	defer c.sync.Unlock()

	_, err := c.data.db.SystemConfig.Delete().
		Where(systemconfig.KeyEQ(key.String())).
		Exec(ctx)
	if err != nil {
		return err
	}
	if _, ok := c.cache[key]; ok {
		delete(c.cache, key)
	}

	return nil
}
