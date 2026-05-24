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
