package common

type ConfigKey string

const (
	ConfigLoginRegisterEnabled                   ConfigKey = "login.register_enabled"
	ConfigLoginUsernameLoginEnabled              ConfigKey = "login.username_login_enabled"
	ConfigLoginEmailLoginEnabled                 ConfigKey = "login.email_login_enabled"
	ConfigLoginRegisterEmailVerificationRequired ConfigKey = "login.register_email_verification_required"
	ConfigLoginAccessTokenTTLMinutes             ConfigKey = "login.access_token_ttl_minutes"
	ConfigLoginRefreshTokenTTLDays               ConfigKey = "login.refresh_token_ttl_days"
	ConfigLoginLockoutThreshold                  ConfigKey = "login.lockout_threshold"
	ConfigLoginLockoutMinutes                    ConfigKey = "login.lockout_minutes"

	ConfigSiteName            ConfigKey = "site.name"
	ConfigSiteExternalURL     ConfigKey = "site.external_url"
	ConfigSiteDefaultLanguage ConfigKey = "site.default_language"
	ConfigSiteSettingsAudit   ConfigKey = "site.settings_audit"
	// ConfigSiteTheme 全站默认外观主题（v1.SiteTheme 的 protojson，空表示没设置）。
	ConfigSiteTheme       ConfigKey = "site.theme"
	ConfigUpdateAutoCheck ConfigKey = "update.auto_check"
	ConfigUpdateChannel   ConfigKey = "update.channel"
	ConfigUpdateLastCheck ConfigKey = "update.last_check"
	// ConfigAuditRetentionDays 操作记录保留天数（0=永久保留）。
	ConfigAuditRetentionDays ConfigKey = "audit.retention_days"

	// 持久性能监控（只对当前节点生效）
	ConfigMonitorRecordEnabled  ConfigKey = "monitor.record_enabled"
	ConfigMonitorInterval       ConfigKey = "monitor.interval_seconds"
	ConfigMonitorRetentionDays  ConfigKey = "monitor.retention_days"
	ConfigMonitorCollectGPU     ConfigKey = "monitor.collect_gpu"
	ConfigMonitorCollectDisk    ConfigKey = "monitor.collect_disk"
	ConfigMonitorCollectNetwork ConfigKey = "monitor.collect_network"
	ConfigMonitorCollectApps    ConfigKey = "monitor.collect_apps"

	ConfigOIDCEnabled                     ConfigKey = "oidc.enabled"
	ConfigOIDCIssuerURL                   ConfigKey = "oidc.issuer_url"
	ConfigOIDCAccessTokenTTLSeconds       ConfigKey = "oidc.access_token_ttl_seconds"
	ConfigOIDCIDTokenTTLSeconds           ConfigKey = "oidc.id_token_ttl_seconds"
	ConfigOIDCAuthorizationCodeTTLSeconds ConfigKey = "oidc.authorization_code_ttl_seconds"
	ConfigOIDCPrivateKey                  ConfigKey = "oidc.private_key"
	ConfigOIDCPrivateKeyRotatedAt         ConfigKey = "oidc.private_key_rotated_at"

	ConfigEmailEnabled        ConfigKey = "email.enabled"
	ConfigEmailHost           ConfigKey = "email.host"
	ConfigEmailPort           ConfigKey = "email.port"
	ConfigEmailUsername       ConfigKey = "email.username"
	ConfigEmailPassword       ConfigKey = "email.password"
	ConfigEmailFrom           ConfigKey = "email.from"
	ConfigEmailFromName       ConfigKey = "email.from_name"
	ConfigEmailEncryption     ConfigKey = "email.encryption"
	ConfigEmailLastTest       ConfigKey = "email.last_test"
	ConfigEmailTimeoutSeconds ConfigKey = "email.timeout_seconds"
	ConfigEmailCcsN           ConfigKey = "email.ccs_n"

	ConfigDockerEnabled               ConfigKey = "docker.enabled"
	ConfigDockerHost                  ConfigKey = "docker.host"
	ConfigDockerTLSEnabled            ConfigKey = "docker.tls_enabled"
	ConfigDockerTLSCA                 ConfigKey = "docker.tls_ca"
	ConfigDockerTLSCert               ConfigKey = "docker.tls_cert"
	ConfigDockerTLSKey                ConfigKey = "docker.tls_key"
	ConfigDockerTLSSkipVerify         ConfigKey = "docker.tls_skip_verify"
	ConfigDockerAPIVersion            ConfigKey = "docker.api_version"
	ConfigDockerRequestTimeoutSeconds ConfigKey = "docker.request_timeout_seconds"
	ConfigDockerDefaultPlatform       ConfigKey = "docker.default_platform"
	ConfigDockerTaskTimeoutSeconds    ConfigKey = "docker.task_timeout_seconds"
	ConfigDockerRegistryAuths         ConfigKey = "docker.registry_auths"

	// frps（内网穿透）实例级配置（无统一 auth_token，见隧道逐条 credential）。
	ConfigFrpsEnabled            ConfigKey = "frps.enabled"
	ConfigFrpsBindAddr           ConfigKey = "frps.bind_addr"
	ConfigFrpsBindPort           ConfigKey = "frps.bind_port"
	ConfigFrpsVhostHTTPPort      ConfigKey = "frps.vhost_http_port"
	ConfigFrpsVhostHTTPSPort     ConfigKey = "frps.vhost_https_port"
	ConfigFrpsKCPBindPort        ConfigKey = "frps.kcp_bind_port"
	ConfigFrpsQUICBindPort       ConfigKey = "frps.quic_bind_port"
	ConfigFrpsSubdomainHost      ConfigKey = "frps.subdomain_host"
	ConfigFrpsStatSampleInterval ConfigKey = "frps.stat_sample_interval"
	ConfigFrpsServerAddr         ConfigKey = "frps.server_addr"
	ConfigFrpsTLSForce           ConfigKey = "frps.tls_force"

	// ConfigStorageAvatar 头像储存（S3 兼容）设置：v1.AvatarStorageConfig 的 protojson（Secret Key 加密存放），空表示没配置过。
	ConfigStorageAvatar ConfigKey = "storage.avatar"

	// ConfigPassportSettings 用户通行证设置：v1.PassportSettings 的 protojson，空表示用默认值。
	ConfigPassportSettings ConfigKey = "passport.settings"
	// ConfigPassportOIDCPrivateKey 用户通行证 OIDC Provider 的 RS256 签名私钥（PEM，首次使用时生成；与管理后台 OIDC 的密钥无关）。
	ConfigPassportOIDCPrivateKey ConfigKey = "passport.oidc_private_key"
)

var configDefaults = map[ConfigKey]string{
	ConfigLoginRegisterEnabled:                   "false",
	ConfigLoginUsernameLoginEnabled:              "true",
	ConfigLoginEmailLoginEnabled:                 "false",
	ConfigLoginRegisterEmailVerificationRequired: "false",
	ConfigLoginAccessTokenTTLMinutes:             "60",
	ConfigLoginRefreshTokenTTLDays:               "14",
	ConfigLoginLockoutThreshold:                  "5",
	ConfigLoginLockoutMinutes:                    "15",

	ConfigSiteName:            "Momoko",
	ConfigSiteExternalURL:     "",
	ConfigSiteDefaultLanguage: "auto",
	ConfigSiteSettingsAudit:   "",
	ConfigSiteTheme:           "",
	ConfigUpdateAutoCheck:     "true",
	ConfigUpdateChannel:       "UpdateChannel_Stable",
	ConfigUpdateLastCheck:     "",
	ConfigAuditRetentionDays:  "180",

	ConfigMonitorRecordEnabled:  "true",
	ConfigMonitorInterval:       "10",
	ConfigMonitorRetentionDays:  "30",
	ConfigMonitorCollectGPU:     "true",
	ConfigMonitorCollectDisk:    "true",
	ConfigMonitorCollectNetwork: "true",
	ConfigMonitorCollectApps:    "true",

	ConfigOIDCEnabled:                     "false",
	ConfigOIDCIssuerURL:                   "",
	ConfigOIDCAccessTokenTTLSeconds:       "3600",
	ConfigOIDCIDTokenTTLSeconds:           "3600",
	ConfigOIDCAuthorizationCodeTTLSeconds: "300",
	ConfigOIDCPrivateKey:                  "",
	ConfigOIDCPrivateKeyRotatedAt:         "",

	ConfigEmailEnabled:        "false",
	ConfigEmailHost:           "",
	ConfigEmailPort:           "465",
	ConfigEmailUsername:       "",
	ConfigEmailPassword:       "",
	ConfigEmailFrom:           "",
	ConfigEmailFromName:       "",
	ConfigEmailEncryption:     "EmailEncryption_SSL",
	ConfigEmailLastTest:       "",
	ConfigEmailTimeoutSeconds: "10",
	ConfigEmailCcsN:           "5",

	ConfigDockerEnabled:               "false",
	ConfigDockerHost:                  "",
	ConfigDockerTLSEnabled:            "false",
	ConfigDockerTLSCA:                 "",
	ConfigDockerTLSCert:               "",
	ConfigDockerTLSKey:                "",
	ConfigDockerTLSSkipVerify:         "false",
	ConfigDockerAPIVersion:            "",
	ConfigDockerRequestTimeoutSeconds: "30",
	ConfigDockerDefaultPlatform:       "",
	ConfigDockerTaskTimeoutSeconds:    "1800",
	ConfigDockerRegistryAuths:         "[]",

	ConfigFrpsEnabled:            "false",
	ConfigFrpsBindAddr:           "0.0.0.0",
	ConfigFrpsBindPort:           "7000",
	ConfigFrpsVhostHTTPPort:      "0",
	ConfigFrpsVhostHTTPSPort:     "0",
	ConfigFrpsKCPBindPort:        "0",
	ConfigFrpsQUICBindPort:       "0",
	ConfigFrpsSubdomainHost:      "",
	ConfigFrpsStatSampleInterval: "30",
	ConfigFrpsServerAddr:         "",
	ConfigFrpsTLSForce:           "false",

	ConfigStorageAvatar: "",

	ConfigPassportSettings:       "",
	ConfigPassportOIDCPrivateKey: "",
}

func (k ConfigKey) String() string {
	return string(k)
}

func ConfigDefault(key ConfigKey) (string, bool) {
	value, ok := configDefaults[key]
	return value, ok
}

func ConfigsLen() int {
	return len(configDefaults)
}
