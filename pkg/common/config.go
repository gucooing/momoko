package common

type ConfigKey string

const (
	ConfigLoginRegisterEnabled                   ConfigKey = "login.register_enabled"
	ConfigLoginUsernameLoginEnabled              ConfigKey = "login.username_login_enabled"
	ConfigLoginEmailLoginEnabled                 ConfigKey = "login.email_login_enabled"
	ConfigLoginRegisterEmailVerificationRequired ConfigKey = "login.register_email_verification_required"

	ConfigOIDCEnabled                     ConfigKey = "oidc.enabled"
	ConfigOIDCIssuerURL                   ConfigKey = "oidc.issuer_url"
	ConfigOIDCAccessTokenTTLSeconds       ConfigKey = "oidc.access_token_ttl_seconds"
	ConfigOIDCIDTokenTTLSeconds           ConfigKey = "oidc.id_token_ttl_seconds"
	ConfigOIDCAuthorizationCodeTTLSeconds ConfigKey = "oidc.authorization_code_ttl_seconds"
	ConfigOIDCPrivateKey                  ConfigKey = "oidc.private_key"

	ConfigEmailEnabled        ConfigKey = "email.enabled"
	ConfigEmailHost           ConfigKey = "email.host"
	ConfigEmailPort           ConfigKey = "email.port"
	ConfigEmailUsername       ConfigKey = "email.username"
	ConfigEmailPassword       ConfigKey = "email.password"
	ConfigEmailFrom           ConfigKey = "email.from"
	ConfigEmailFromName       ConfigKey = "email.from_name"
	ConfigEmailUseTLS         ConfigKey = "email.use_tls"
	ConfigEmailTimeoutSeconds ConfigKey = "email.timeout_seconds"
	ConfigEmailCcsN           ConfigKey = "email.ccs_n"

	ConfigDockerEnabled               ConfigKey = "docker.enabled"
	ConfigDockerHost                  ConfigKey = "docker.host"
	ConfigDockerTLSEnabled            ConfigKey = "docker.tls_enabled"
	ConfigDockerTLSCAPath             ConfigKey = "docker.tls_ca_path"
	ConfigDockerTLSCertPath           ConfigKey = "docker.tls_cert_path"
	ConfigDockerTLSKeyPath            ConfigKey = "docker.tls_key_path"
	ConfigDockerAPIVersion            ConfigKey = "docker.api_version"
	ConfigDockerRequestTimeoutSeconds ConfigKey = "docker.request_timeout_seconds"
	ConfigDockerDefaultPlatform       ConfigKey = "docker.default_platform"
	ConfigDockerTaskTimeoutSeconds    ConfigKey = "docker.task_timeout_seconds"
	ConfigDockerRegistryAuths         ConfigKey = "docker.registry_auths"

	// Sub2API 配置（按字段拆分为独立 KV）
	ConfigSub2APIHomeEnabled         ConfigKey = "sub2api.home_enabled"
	ConfigSub2APISyncEnabled         ConfigKey = "sub2api.sync_enabled"
	ConfigSub2APIBaseURL             ConfigKey = "sub2api.base_url"
	ConfigSub2APIAdminAPIKey         ConfigKey = "sub2api.admin_api_key"
	ConfigSub2APIConsoleURL          ConfigKey = "sub2api.console_url"
	ConfigSub2APITitle               ConfigKey = "sub2api.title"
	ConfigSub2APISubtitle            ConfigKey = "sub2api.subtitle"
	ConfigSub2APIIntroduction        ConfigKey = "sub2api.introduction"
	ConfigSub2APISyncIntervalMinutes ConfigKey = "sub2api.sync_interval_minutes"
	ConfigSub2APIHistoryDays         ConfigKey = "sub2api.history_days"
	ConfigSub2APIPageSize            ConfigKey = "sub2api.page_size"
	// ConfigSub2APISyncState 存储 Sub2API 最近一次同步状态（运行时状态，JSON）。
	ConfigSub2APISyncState ConfigKey = "sub2api.sync_state"
	// ConfigSub2APIAllowedSrcHosts 允许调用 momoko 生图的 sub2api 站点 origin 列表（JSON 数组）。
	// 留空表示不限制（仅用于开发/调试）。
	ConfigSub2APIAllowedSrcHosts ConfigKey = "sub2api.allowed_src_hosts"
	// ConfigSub2APIImageEnabled 是否启用生图功能。
	ConfigSub2APIImageEnabled ConfigKey = "sub2api.image_enabled"
	// ConfigSub2APISrcHostWhitelistEnabled 是否开启站点白名单校验。
	ConfigSub2APISrcHostWhitelistEnabled ConfigKey = "sub2api.src_host_whitelist_enabled"

	// 每日抽奖活动配置（存 KV，不单独建表）。
	ConfigSub2APILotteryEnabled     ConfigKey = "sub2api.lottery_enabled"      // 活动是否启用
	ConfigSub2APILotteryPoolRatio   ConfigKey = "sub2api.lottery_pool_ratio"   // 奖池比例（默认 0.05）
	ConfigSub2APILotteryThreshold   ConfigKey = "sub2api.lottery_threshold"    // 报名门槛金额（默认 2）
	ConfigSub2APILotteryBaseWinners ConfigKey = "sub2api.lottery_base_winners" // 基准中奖人数（默认 10）
	ConfigSub2APILotteryMaxWinners  ConfigKey = "sub2api.lottery_max_winners"  // 最大中奖人数（0=无限）
	ConfigSub2APILotteryAutoPayout  ConfigKey = "sub2api.lottery_auto_payout"  // 是否自动发放

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
)

var configDefaults = map[ConfigKey]string{
	ConfigLoginRegisterEnabled:                   "false",
	ConfigLoginUsernameLoginEnabled:              "true",
	ConfigLoginEmailLoginEnabled:                 "false",
	ConfigLoginRegisterEmailVerificationRequired: "false",

	ConfigOIDCEnabled:                     "false",
	ConfigOIDCIssuerURL:                   "",
	ConfigOIDCAccessTokenTTLSeconds:       "3600",
	ConfigOIDCIDTokenTTLSeconds:           "3600",
	ConfigOIDCAuthorizationCodeTTLSeconds: "300",
	ConfigOIDCPrivateKey:                  "",

	ConfigEmailEnabled:        "false",
	ConfigEmailHost:           "",
	ConfigEmailPort:           "465",
	ConfigEmailUsername:       "",
	ConfigEmailPassword:       "",
	ConfigEmailFrom:           "",
	ConfigEmailFromName:       "",
	ConfigEmailUseTLS:         "true",
	ConfigEmailTimeoutSeconds: "10",
	ConfigEmailCcsN:           "5",

	ConfigDockerEnabled:               "false",
	ConfigDockerHost:                  "",
	ConfigDockerTLSEnabled:            "false",
	ConfigDockerTLSCAPath:             "",
	ConfigDockerTLSCertPath:           "",
	ConfigDockerTLSKeyPath:            "",
	ConfigDockerAPIVersion:            "",
	ConfigDockerRequestTimeoutSeconds: "30",
	ConfigDockerDefaultPlatform:       "",
	ConfigDockerTaskTimeoutSeconds:    "1800",
	ConfigDockerRegistryAuths:         "[]",

	ConfigSub2APIHomeEnabled:             "false",
	ConfigSub2APISyncEnabled:             "true",
	ConfigSub2APIBaseURL:                 "",
	ConfigSub2APIAdminAPIKey:             "",
	ConfigSub2APIConsoleURL:              "",
	ConfigSub2APITitle:                   "Sub2API",
	ConfigSub2APISubtitle:                "统一订阅转换与模型调用看板",
	ConfigSub2APIIntroduction:            "",
	ConfigSub2APISyncIntervalMinutes:     "10",
	ConfigSub2APIHistoryDays:             "30",
	ConfigSub2APIPageSize:                "500",
	ConfigSub2APISyncState:               "",
	ConfigSub2APIAllowedSrcHosts:         "[]",
	ConfigSub2APIImageEnabled:            "true",
	ConfigSub2APISrcHostWhitelistEnabled: "false",
	ConfigSub2APILotteryEnabled:          "false",
	ConfigSub2APILotteryPoolRatio:        "0.05",
	ConfigSub2APILotteryThreshold:        "2",
	ConfigSub2APILotteryBaseWinners:      "10",
	ConfigSub2APILotteryMaxWinners:       "0",
	ConfigSub2APILotteryAutoPayout:       "true",

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
