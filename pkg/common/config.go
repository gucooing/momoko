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
	ConfigDockerDefaultLogTail        ConfigKey = "docker.default_log_tail"
	ConfigDockerTaskTimeoutSeconds    ConfigKey = "docker.task_timeout_seconds"
	ConfigDockerRegistryAuths         ConfigKey = "docker.registry_auths"

	// ConfigSub2APIConfig 存储 Sub2API 模块的完整配置（JSON）。
	ConfigSub2APIConfig ConfigKey = "sub2api.config"
	// ConfigSub2APISyncState 存储 Sub2API 最近一次同步状态（JSON）。
	ConfigSub2APISyncState ConfigKey = "sub2api.sync_state"
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
	ConfigDockerDefaultLogTail:        "200",
	ConfigDockerTaskTimeoutSeconds:    "1800",
	ConfigDockerRegistryAuths:         "[]",

	ConfigSub2APIConfig:    "",
	ConfigSub2APISyncState: "",
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
