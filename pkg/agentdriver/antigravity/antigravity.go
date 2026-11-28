// Package antigravity 是 Google Antigravity（反重力）一系的驱动：Antigravity CLI 自带 ACP 服务（agy acp），
// 发行包在 GitHub google-antigravity/antigravity-cli 上，里面只有一个可执行文件 antigravity。
// 登录用 agy acp --login（打出网址，在浏览器里拿到码贴回终端）。每个会话的数据（会话记录、生成的图片和计划）
// 用 --gemini-dir 放进 Momoko 给这个会话的目录。
package antigravity

import (
	"encoding/json"
	"runtime"
	"strings"

	"momoko/pkg/agentdriver"
	"momoko/pkg/agentinstall"
)

// Kind 这一系的标识。
const Kind = "antigravity"

// noAutoUpdate 关掉 CLI 自己在后台的自动更新。
var noAutoUpdate = map[string]string{"AGY_CLI_DISABLE_AUTO_UPDATE": "true"}

// Driver Antigravity 的驱动。
type Driver struct{}

var _ agentdriver.Driver = Driver{}

func (Driver) Kind() string { return Kind }

// Protocol 没有第三方接口：它的 API Key 模式要改 ~/.gemini/antigravity-cli/settings.json 里的 modelProvider，全服务器共用一份。
func (Driver) Protocol() string { return "" }

// Adapter 安装脚本把 CLI 装成 agy。
func (Driver) Adapter() string { return "agy" }

// Launch agy acp 才是 ACP 服务。--process-per-session：Momoko 交给它的 HTTP MCP 服务器要每个会话单独一个进程才能用；
// --disable-telemetry：在服务器上替人跑，不发遥测。Momoko 下载的那一份由 Momoko 更新，关掉它自己的自动更新。
func (Driver) Launch(path string, managed bool) agentdriver.Launch {
	launch := agentdriver.Launch{Command: path, Args: []string{"acp", "--process-per-session", "--disable-telemetry"}}
	if managed {
		launch.Env = noAutoUpdate
	}
	return launch
}

// Detect 握手时报的名字是 Antigravity ACP Server。
func (Driver) Detect(agentName string) bool {
	return strings.Contains(strings.ToLower(agentName), "antigravity")
}

// Release GitHub 发布页上的 agy_cli_<系统>_<架构>[_musl]：macOS 叫 mac；Linux 是 tar.gz（另有 musl 的），Windows 是 zip。
// 包里没有目录，只有 antigravity（Windows 是 antigravity.exe）。
func (Driver) Release() (agentinstall.Release, bool) {
	goos, arch, ok := agentinstall.Platform()
	if !ok {
		return agentinstall.Release{}, false
	}
	if goos == "darwin" {
		goos = "mac"
	}
	asset := "agy_cli_" + goos + "_" + arch
	switch {
	case runtime.GOOS == "windows":
		asset += ".zip"
	case runtime.GOOS == "linux" && agentinstall.Musl():
		asset += "_musl.tar.gz"
	default:
		asset += ".tar.gz"
	}
	return agentinstall.Release{
		Name:  "Antigravity",
		Repo:  "google-antigravity/antigravity-cli",
		Asset: asset,
		Files: []string{"antigravity"},
	}, true
}

// APIEnv 没有第三方接口，见 Protocol。
func (Driver) APIEnv(agentdriver.API) map[string]string { return nil }

// LoginLimits 它没有设定最长上下文和最长输出的配置。
func (Driver) LoginLimits(int64, int64) map[string]string { return nil }

// SessionData --gemini-dir 换掉 ~/.gemini：会话记录（conversations/<会话 id>.db）和生成的文件（brain/<会话 id>/）
// 都在 dir/antigravity-cli 下，不再写到用户目录里。
func (Driver) SessionData(dir string) []string { return []string{"--gemini-dir", dir} }

// Unrestricted Turbo：--dangerously-skip-permissions 让所有工具调用都不再询问（settings.json 里的 toolPermission always-proceed），
// 经 ACP 给的权限模式里没有它，只能在启动时带上。
func (Driver) Unrestricted() []string { return []string{"--dangerously-skip-permissions"} }

// RemoveTranscript 会话的数据都在 SessionData 给的目录里，跟着会话目录一起删。
func (Driver) RemoveTranscript(_, _ string) {}

// Login agy acp --login：在终端里打出网址，浏览器里登录后把码贴回来，然后退出。SSH 上也能用，登录信息和 agy acp 共用。
// 登录这一下不用自动更新。
func (Driver) Login(adapter string) (agentdriver.Launch, bool) {
	if adapter == "" {
		return agentdriver.Launch{}, false
	}
	return agentdriver.Launch{Command: adapter, Args: []string{"acp", "--login"}, Env: noAutoUpdate}, true
}

// SandboxEnv 沙箱里用不着自动更新。遥测在 Launch 里已经关了。
func (Driver) SandboxEnv() map[string]string { return noAutoUpdate }

// Command 工具调用的原始输入格式文档没写，命令看工具调用的标题。
func (Driver) Command(json.RawMessage) string { return "" }

// ToolOutput 工具输出都在 content 里。
func (Driver) ToolOutput(json.RawMessage) string { return "" }
