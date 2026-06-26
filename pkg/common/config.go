package common

type ConfigKey string

const (
	ConfigLoginRegisterEnabled                   ConfigKey = "login.register_enabled"
	ConfigLoginUsernameLoginEnabled              ConfigKey = "login.username_login_enabled"
	ConfigLoginEmailLoginEnabled                 ConfigKey = "login.email_login_enabled"
	ConfigLoginRegisterEmailVerificationRequired ConfigKey = "login.register_email_verification_required"

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
	ConfigFrpsMaxPoolCount       ConfigKey = "frps.max_pool_count"
	ConfigFrpsMaxPortsPerClient  ConfigKey = "frps.max_ports_per_client"
	ConfigFrpsHeartbeatTimeout   ConfigKey = "frps.heartbeat_timeout"
	ConfigFrpsTCPMux             ConfigKey = "frps.tcp_mux"
	ConfigFrpsUDPPacketSize      ConfigKey = "frps.udp_packet_size"
)

var configDefaults = map[ConfigKey]string{
	ConfigLoginRegisterEnabled:                   "false",
	ConfigLoginUsernameLoginEnabled:              "true",
	ConfigLoginEmailLoginEnabled:                 "false",
	ConfigLoginRegisterEmailVerificationRequired: "false",

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
	ConfigFrpsMaxPoolCount:       "5",
	ConfigFrpsMaxPortsPerClient:  "0",
	ConfigFrpsHeartbeatTimeout:   "-1",
	ConfigFrpsTCPMux:             "true",
	ConfigFrpsUDPPacketSize:      "1500",
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
