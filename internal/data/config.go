package data

import (
	"context"
	"encoding/json"
	"fmt"
	authpkg "momoko/pkg/auth"
	"momoko/pkg/secretbox"
	"strconv"
	"strings"
	"sync"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/emailtemplate"
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
	registerEmailVerificationRequired, err := c.getBoolConfig(ctx, common.ConfigLoginRegisterEmailVerificationRequired)
	if err != nil {
		return nil, err
	}

	return &v1.LoginConfig{
		RegisterEnabled:                   registerEnabled,
		UsernameLoginEnabled:              usernameLoginEnabled,
		EmailLoginEnabled:                 emailLoginEnabled,
		RegisterEmailVerificationRequired: registerEmailVerificationRequired,
	}, nil
}

func (c *ConfigRepo) UpdateLoginConfig(ctx context.Context, req *v1.UpdateLoginConfigRequest) (*v1.LoginConfig, error) {
	configs := map[common.ConfigKey]string{
		common.ConfigLoginRegisterEnabled:                   strconv.FormatBool(req.RegisterEnabled),
		common.ConfigLoginUsernameLoginEnabled:              strconv.FormatBool(req.UsernameLoginEnabled),
		common.ConfigLoginEmailLoginEnabled:                 strconv.FormatBool(req.EmailLoginEnabled),
		common.ConfigLoginRegisterEmailVerificationRequired: strconv.FormatBool(req.RegisterEmailVerificationRequired),
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

func (c *ConfigRepo) UpdateEmailTemplate(ctx context.Context, req *v1.UpdateEmailTemplateRequest) (*gen.EmailTemplate, error) {
	templateType := req.Type.String()

	err := c.data.db.EmailTemplate.Create().
		SetType(templateType).
		SetSubject(req.Subject).
		SetTemplate(req.Template).
		OnConflictColumns(emailtemplate.FieldType).
		UpdateNewValues().
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	return c.data.db.EmailTemplate.Query().
		Where(emailtemplate.TypeEQ(templateType)).
		Only(ctx)
}

func (c *ConfigRepo) EmailTemplate(ctx context.Context, templateType v1.EmailTemplateType) (*gen.EmailTemplate, error) {
	return c.data.db.EmailTemplate.Query().
		Where(emailtemplate.TypeEQ(templateType.String())).
		Only(ctx)
}

func (c *ConfigRepo) DockerConfig(ctx context.Context) (*v1.DockerConfigInfo, error) {
	enabled, err := c.getBoolConfig(ctx, common.ConfigDockerEnabled)
	if err != nil {
		return nil, err
	}
	tlsEnabled, err := c.getBoolConfig(ctx, common.ConfigDockerTLSEnabled)
	if err != nil {
		return nil, err
	}
	requestTimeout, err := c.getInt32Config(ctx, common.ConfigDockerRequestTimeoutSeconds)
	if err != nil {
		return nil, err
	}
	logTail, err := c.getInt32Config(ctx, common.ConfigDockerDefaultLogTail)
	if err != nil {
		return nil, err
	}
	taskTimeout, err := c.getInt32Config(ctx, common.ConfigDockerTaskTimeoutSeconds)
	if err != nil {
		return nil, err
	}
	host, err := c.getStringConfig(ctx, common.ConfigDockerHost)
	if err != nil {
		return nil, err
	}
	caPath, err := c.getStringConfig(ctx, common.ConfigDockerTLSCAPath)
	if err != nil {
		return nil, err
	}
	certPath, err := c.getStringConfig(ctx, common.ConfigDockerTLSCertPath)
	if err != nil {
		return nil, err
	}
	keyPath, err := c.getStringConfig(ctx, common.ConfigDockerTLSKeyPath)
	if err != nil {
		return nil, err
	}
	apiVersion, err := c.getStringConfig(ctx, common.ConfigDockerAPIVersion)
	if err != nil {
		return nil, err
	}
	defaultPlatform, err := c.getStringConfig(ctx, common.ConfigDockerDefaultPlatform)
	if err != nil {
		return nil, err
	}
	authsRaw, err := c.getRegistryAuth(ctx, common.ConfigDockerRegistryAuths)
	if err != nil {
		return nil, err
	}

	return &v1.DockerConfigInfo{
		Enabled:               enabled,
		Host:                  host,
		TlsEnabled:            tlsEnabled,
		TlsCaPath:             caPath,
		TlsCertPath:           certPath,
		TlsKeyPath:            keyPath,
		ApiVersion:            apiVersion,
		RequestTimeoutSeconds: requestTimeout,
		DefaultPlatform:       defaultPlatform,
		DefaultLogTail:        logTail,
		TaskTimeoutSeconds:    taskTimeout,
		RegistryAuths:         authsRaw,
	}, nil
}

func (c *ConfigRepo) UpdateDockerConfig(ctx context.Context, req *v1.DockerConfigInfo) (*v1.DockerConfigInfo, error) {
	authsRaw, err := c.saveRegistryAuth(req.RegistryAuths)
	if err != nil {
		return nil, err
	}

	configs := map[common.ConfigKey]string{
		common.ConfigDockerEnabled:               strconv.FormatBool(req.Enabled),
		common.ConfigDockerTLSEnabled:            strconv.FormatBool(req.TlsEnabled),
		common.ConfigDockerRequestTimeoutSeconds: strconv.FormatInt(int64(req.RequestTimeoutSeconds), 10),
		common.ConfigDockerDefaultLogTail:        strconv.FormatInt(int64(req.DefaultLogTail), 10),
		common.ConfigDockerTaskTimeoutSeconds:    strconv.FormatInt(int64(req.TaskTimeoutSeconds), 10),
		common.ConfigDockerHost:                  req.Host,
		common.ConfigDockerTLSCAPath:             req.TlsCaPath,
		common.ConfigDockerTLSCertPath:           req.TlsCertPath,
		common.ConfigDockerTLSKeyPath:            req.TlsKeyPath,
		common.ConfigDockerAPIVersion:            req.ApiVersion,
		common.ConfigDockerDefaultPlatform:       req.DefaultPlatform,
		common.ConfigDockerRegistryAuths:         authsRaw,
	}
	if err = c.BatchUpdate(ctx, configs); err != nil {
		return nil, err
	}
	return c.DockerConfig(ctx)
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

func (c *ConfigRepo) getRegistryAuth(ctx context.Context, key common.ConfigKey) ([]*v1.DockerRegistryAuth, error) {
	value, err := c.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var auths []*v1.DockerRegistryAuth
	err = json.Unmarshal([]byte(value), &auths)
	if err != nil {
		return nil, err
	}
	box := secretbox.New(authpkg.AuthSecretKey)
	for i := range auths {
		password, err := box.Encrypt(auths[i].Password)
		if err != nil {
			return nil, err
		}
		token, err := box.Encrypt(auths[i].Token)
		if err != nil {
			return nil, err
		}
		auths[i].Password = password
		auths[i].Token = token
	}
	return auths, nil
}

func (c *ConfigRepo) saveRegistryAuth(auths []*v1.DockerRegistryAuth) (string, error) {
	box := secretbox.New(authpkg.AuthSecretKey)
	for i := range auths {
		if strings.HasPrefix(auths[i].Password, "v1:") {
			password, err := box.Decrypt(auths[i].Password)
			if err != nil {
				return "", err
			}
			auths[i].Password = password
		}
		if strings.HasPrefix(auths[i].Token, "v1:") {
			token, err := box.Decrypt(auths[i].Token)
			if err != nil {
				return "", err
			}
			auths[i].Token = token
		}
	}
	authsRaw, err := json.Marshal(auths)
	if err != nil {
		return "", err
	}
	return string(authsRaw), nil
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
