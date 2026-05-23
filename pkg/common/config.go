package common

type ConfigKey string

const (
	ConfigLoginRegisterEnabled      ConfigKey = "login.register_enabled"
	ConfigLoginUsernameLoginEnabled ConfigKey = "login.username_login_enabled"
	ConfigLoginEmailLoginEnabled    ConfigKey = "login.email_login_enabled"
)

var configDefaults = map[ConfigKey]string{
	ConfigLoginRegisterEnabled:      "false",
	ConfigLoginUsernameLoginEnabled: "true",
	ConfigLoginEmailLoginEnabled:    "false",
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
