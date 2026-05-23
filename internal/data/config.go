package data

import (
	"context"
	"sync"

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
