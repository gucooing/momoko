// Package codex 是 Codex 一系的驱动：适配器 codex-acp（用户自己在服务器上安装，自带 Codex CLI），
// 配置目录 CODEX_HOME，第三方接口靠 CODEX_CONFIG 声明一个 model provider，
// 会话记录在 sessions/年/月/日/rollout-<时间>-<会话 id>.jsonl。
package codex

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"momoko/pkg/agentdriver"
	"momoko/pkg/agentinstall"
	"momoko/pkg/agentproc"
)

// Kind 这一系的标识。
const Kind = "codex"

// Driver Codex 的驱动。
type Driver struct{}

var _ agentdriver.Driver = Driver{}

func (Driver) Kind() string     { return Kind }
func (Driver) Protocol() string { return "openai" }
func (Driver) Adapter() string  { return "codex-acp" }

func (Driver) Detect(agentName string) bool {
	return strings.Contains(strings.ToLower(agentName), "codex")
}

// Release codex-acp 没有单文件的发行包，由用户自己安装。
func (Driver) Release() (agentinstall.Release, bool) { return agentinstall.Release{}, false }

// Home Codex 默认的配置目录：服务器上的登录和会话记录都在这里。
func (Driver) Home() string {
	if dir := os.Getenv("CODEX_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codex")
}

// contextLimits 最长上下文写进 Codex 配置：model_context_window，Codex 会把它夹到服务端给这个模型的上限（max_context_window）；
// 自动压缩阈值 model_auto_compact_token_limit 一起设成它的 90%（Codex 允许的最大值），不然还会按服务端给这个模型的阈值提前压缩。
func contextLimits(cfg map[string]any, contextWindow int64) {
	if contextWindow > 0 {
		cfg["model_context_window"] = contextWindow
		cfg["model_auto_compact_token_limit"] = contextWindow * 9 / 10
	}
}

// APIEnv 通过 CODEX_CONFIG 声明一个叫 momoko 的 model provider，密钥放在它指定的环境变量里；推理等级是 model_reasoning_effort，
// 最长上下文见 contextLimits。Codex 没有设定最长输出的配置，模型目录里的这一项不交给它。
func (Driver) APIEnv(api agentdriver.API) map[string]string {
	cfg := map[string]any{
		"model":          api.Model,
		"model_provider": "momoko",
		"model_providers": map[string]any{
			"momoko": map[string]any{
				"name":     api.Name,
				"base_url": api.BaseURL,
				"env_key":  "MOMOKO_PROVIDER_API_KEY",
				"wire_api": "responses",
			},
		},
	}
	if api.HasLevel && api.Level != "" {
		cfg["model_reasoning_effort"] = api.Level
	}
	contextLimits(cfg, api.ContextWindow)
	raw, _ := json.Marshal(cfg)
	env := map[string]string{
		"CODEX_CONFIG":            string(raw),
		"MODEL_PROVIDER":          "momoko",
		"MOMOKO_PROVIDER_API_KEY": api.Key,
		"OPENAI_API_KEY":          "",
		"CODEX_API_KEY":           "",
	}
	for k, v := range api.Extra {
		if k != "" {
			env[k] = v
		}
	}
	return env
}

// LoginLimits codex-acp 不读自己的启动参数（-c 覆盖项到不了 Codex），配置只认 CODEX_CONFIG：一段 JSON，
// 作为每个会话的配置覆盖交给 Codex。最长上下文见 contextLimits；Codex 没有设定最长输出的配置，maxOutput 不用。
func (Driver) LoginLimits(contextWindow, _ int64) map[string]string {
	if contextWindow <= 0 {
		return nil
	}
	cfg := map[string]any{}
	contextLimits(cfg, contextWindow)
	raw, _ := json.Marshal(cfg)
	return map[string]string{"CODEX_CONFIG": string(raw)}
}

// Login codex-acp 自己没有登录命令（它的浏览器登录要在服务器本机开网页），用服务器上的 Codex CLI：
// codex login --device-auth 给一个网址和一次性的码，在哪台电脑的浏览器上打开都行。登录信息写进默认的配置目录，和 codex-acp 共用。
func (Driver) Login(string) (agentdriver.Launch, bool) {
	path, ok := agentproc.Lookup("codex")
	if !ok {
		return agentdriver.Launch{}, false
	}
	return agentdriver.Launch{Command: path, Args: []string{"login", "--device-auth"}}, true
}

// ProxyEnv Codex 默认不发遥测，不用另外关。
func (Driver) ProxyEnv() map[string]string { return nil }

// RemoveTranscript 会话记录按日期分目录放在 sessions 下（归档的在 archived_sessions 下），文件名以会话 id 结尾；
// 生成的图片在 generated_images/<会话 id>/ 下（id 里字母数字和 - _ 以外的字符换成 _），一起删掉。
func (d Driver) RemoveTranscript(_, sessionID string) {
	// 空的 id 会匹配到所有记录
	if sessionID == "" {
		return
	}
	imageDir := strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || ('0' <= r && r <= '9') {
			return r
		}
		return '_'
	}, sessionID)
	_ = os.RemoveAll(filepath.Join(d.Home(), "generated_images", imageDir))
	suffix := "-" + sessionID + ".jsonl"
	for _, sub := range []string{"sessions", "archived_sessions"} {
		_ = filepath.WalkDir(filepath.Join(d.Home(), sub), func(path string, e fs.DirEntry, err error) error {
			if err == nil && !e.IsDir() && strings.HasSuffix(e.Name(), suffix) {
				_ = os.Remove(path)
			}
			return nil
		})
	}
}

// Command Codex 的命令工具：原始输入里的 command 是参数数组，常见的是 ["bash", "-lc", "<命令>"]，只显示真正的命令。
func (Driver) Command(raw json.RawMessage) string {
	var input struct {
		Command []string `json:"command"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &input) != nil || len(input.Command) == 0 {
		return ""
	}
	parts := input.Command
	if len(parts) >= 3 && (parts[1] == "-lc" || parts[1] == "-c" || strings.EqualFold(parts[1], "/c") || strings.EqualFold(parts[1], "-Command")) {
		parts = parts[2:]
	}
	return strings.Join(parts, " ")
}

// ToolOutput codex-acp 的 MCP 工具结果只在 rawOutput 的 {result, error} 里（v1 v2 都是）：取 result.content 里的文字，
// 没有文字时用 result 的 JSON，出错时用 error.message；有的工具的 rawOutput 直接是一段文字。
// 命令输出走 v2 的终端更新，不在这里；非终端命令结束时的 rawOutput {exit_code} 不是输出，不要。
func (Driver) ToolOutput(rawOutput json.RawMessage) string { return rawOutputText(rawOutput) }

// rawOutputText rawOutput 里给人看的文字：一段文字原样用；MCP 的 {result, error} 见 ToolOutput；别的为空。
func rawOutputText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	var mcp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &mcp) != nil {
		return ""
	}
	if mcp.Error != nil && mcp.Error.Message != "" {
		return mcp.Error.Message
	}
	if len(mcp.Result) == 0 || string(mcp.Result) == "null" {
		return ""
	}
	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if json.Unmarshal(mcp.Result, &result) == nil {
		var texts []string
		for _, c := range result.Content {
			if c.Type == "text" && c.Text != "" {
				texts = append(texts, c.Text)
			}
		}
		if len(texts) > 0 {
			return strings.Join(texts, "\n")
		}
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, mcp.Result, "", "  ") != nil {
		return string(mcp.Result)
	}
	return pretty.String()
}
