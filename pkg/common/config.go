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
