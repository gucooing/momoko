// Package claude 是 Claude Code 一系的驱动：适配器 claude-agent-acp（有打包了 Claude CLI 的单文件发行包），
// 配置目录 CLAUDE_CONFIG_DIR，第三方接口走 ANTHROPIC_* 这一组环境变量，最长上下文和最长输出走 CLAUDE_CODE_MAX_*，
// 会话记录在 projects/<目录>/<会话 id>.jsonl。
package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"momoko/pkg/agentdriver"
	"momoko/pkg/agentfs"
	"momoko/pkg/agentinstall"
)

// Kind 这一系的标识。
const Kind = "claude"

// Driver Claude Code 的驱动。
type Driver struct{}

var _ agentdriver.Driver = Driver{}

func (Driver) Kind() string    { return Kind }
func (Driver) Vendor() string  { return "anthropic" }
func (Driver) Adapter() string { return "claude-agent-acp" }

func (Driver) Detect(agentName string) bool {
	return strings.Contains(strings.ToLower(agentName), "claude")
}

// Release claude-agent-acp 的发行包：适配器和它要用的 Claude CLI 两个文件。Linux 是 tar.gz（另有 musl 的），其它是 zip。
func (Driver) Release() (agentinstall.Release, bool) {
	goos, arch, ok := agentinstall.Platform()
	if !ok {
		return agentinstall.Release{}, false
	}
	asset := "claude-agent-acp-" + goos + "-" + arch
	switch {
	case runtime.GOOS != "linux":
		asset += ".zip"
	case agentinstall.Musl():
		asset += "-musl.tar.gz"
	default:
		asset += ".tar.gz"
	}
	return agentinstall.Release{
		Name:  "Claude Code",
		Repo:  "agentclientprotocol/claude-agent-acp",
		Asset: asset,
		Files: []string{"claude-agent-acp", "claude"},
	}, true
}

// Home Claude Code 默认的配置目录：服务器上的登录和会话记录都在这里。
func (Driver) Home() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

// APIEnv 接口地址和令牌走 ANTHROPIC_BASE_URL / ANTHROPIC_AUTH_TOKEN，模型靠 ANTHROPIC_MODEL 这一组变量，
// 推理等级走 CLAUDE_CODE_EFFORT_LEVEL（声明成不带参数的等级就关掉思考），
// 最长上下文和最长输出走 CLAUDE_CODE_MAX_CONTEXT_TOKENS / CLAUDE_CODE_MAX_OUTPUT_TOKENS（Claude Code 不认识第三方模型的窗口）。
func (Driver) APIEnv(api agentdriver.API) map[string]string {
	fast := api.FastModel
	if fast == "" {
		fast = api.Model
	}
	env := map[string]string{
		"ANTHROPIC_BASE_URL":                       api.BaseURL,
		"ANTHROPIC_AUTH_TOKEN":                     api.Key,
		"ANTHROPIC_API_KEY":                        "",
		"CLAUDE_CODE_OAUTH_TOKEN":                  "",
		"ANTHROPIC_MODEL":                          api.Model,
		"ANTHROPIC_CUSTOM_MODEL_OPTION":            api.Model,
		"ANTHROPIC_DEFAULT_OPUS_MODEL":             api.Model,
		"ANTHROPIC_DEFAULT_SONNET_MODEL":           api.Model,
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":            fast,
		"ANTHROPIC_SMALL_FAST_MODEL":               fast,
		"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
	}
	limits, _ := Driver{}.LoginLimits(api.ContextWindow, api.MaxOutput)
	for k, v := range limits {
		env[k] = v
	}
	if api.HasLevel {
		if api.Level == "" {
			env["MAX_THINKING_TOKENS"] = "0"
		} else {
			env["CLAUDE_CODE_EFFORT_LEVEL"] = api.Level
		}
	}
	for k, v := range api.Extra {
		if k != "" {
			env[k] = v
		}
	}
	return env
}

// LoginLimits 最长上下文和最长输出都靠环境变量：CLAUDE_CODE_MAX_CONTEXT_TOKENS、CLAUDE_CODE_MAX_OUTPUT_TOKENS。
func (Driver) LoginLimits(contextWindow, maxOutput int64) (map[string]string, []string) {
	env := map[string]string{}
	if contextWindow > 0 {
		env["CLAUDE_CODE_MAX_CONTEXT_TOKENS"] = strconv.FormatInt(contextWindow, 10)
	}
	if maxOutput > 0 {
		env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = strconv.FormatInt(maxOutput, 10)
	}
	return env, nil
}

// projectDirMax Claude CLI 给项目目录名的长度上限：超过的截断后再接一段它自己算的散列。
const projectDirMax = 200

// projectDir 工作目录对应的 projects/<目录名>：目录名是把工作目录里字母数字以外的字符都换成「-」，
// 太长时 Claude CLI 截断后接一段散列，这时按前缀找。找不到时为空。
func projectDir(home, cwd string) string {
	encoded := strings.Map(func(r rune) rune {
		if r < 128 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, cwd)
	root := filepath.Join(home, "projects")
	if len(encoded) <= projectDirMax {
		return filepath.Join(root, encoded)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	prefix := encoded[:projectDirMax] + "-"
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), prefix) {
			return filepath.Join(root, e.Name())
		}
	}
	return ""
}

// RemoveTranscript 这个会话在配置目录里的东西都以会话 id 命名：projects/<目录名>/<会话 id>.jsonl 和同名目录（子 Agent 的记录）、
// session-env/<会话 id>、file-history/<会话 id>。项目目录删空了也一起删掉。
func (d Driver) RemoveTranscript(cwd, sessionID string) {
	// 空的或带路径符号的 id 拼出来会指到整个目录
	if sessionID == "" || strings.ContainsAny(sessionID, `/\.`) {
		return
	}
	home := d.Home()
	if project := projectDir(home, cwd); project != "" {
		agentfs.RemoveAll(filepath.Join(project, sessionID+".jsonl"))
		agentfs.RemoveAll(filepath.Join(project, sessionID))
		_ = os.Remove(project)
	}
	agentfs.RemoveAll(filepath.Join(home, "session-env", sessionID))
	agentfs.RemoveAll(filepath.Join(home, "file-history", sessionID))
}

// Command Claude 的 Bash 工具：原始输入里的 command 是一整行命令。
func (Driver) Command(raw json.RawMessage) string {
	var input struct {
		Command string `json:"command"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &input) != nil {
		return ""
	}
	return input.Command
}
