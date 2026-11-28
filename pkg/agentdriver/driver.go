// Package agentdriver 定义「一系 Agent」和别家不一样的地方：适配器叫什么、怎么启动、第三方接口的环境变量怎么给、
// 会话记录放在哪、能不能由 Momoko 下载、工具调用怎么读。官方账号由用户自己在服务器上登录命令行，Momoko 不管。
//
// 每一系在自己的子包里实现 Driver（claude、codex、antigravity），对某一家的专门处理只写在它自己的子包里；
// 认不出是哪一系的 ACP Agent 用 Generic，只按协议正文办事。会话引擎（pkg/agentrun）只通过这个接口问这些事。
package agentdriver

import (
	"encoding/json"

	"momoko/pkg/agentinstall"
	"momoko/pkg/agentsandbox"
)

// Launch 怎么启动一个 Agent 进程。
type Launch struct {
	Command string
	Args    []string
	// Env 启动命令自带的环境（在当前进程环境之上）。
	Env map[string]string
	// Sandbox 不为空时在沙箱里启动（连同它启动的所有子进程），见 agentproc.Config.Sandbox。
	Sandbox *agentsandbox.Config
}

// API 第三方接口的配置。
type API struct {
	Name    string
	BaseURL string
	Key     string
	// Model 这次启动用的模型；FastModel 是做小任务用的模型，空则同 Model。
	Model     string
	FastModel string
	// HasLevel 为 true 时带上推理等级：Level 是模型目录里声明的 API 参数值，空 = 不带这个参数。
	HasLevel bool
	Level    string
	// ContextWindow / MaxOutput 这个模型的最长上下文和最长输出（token 数）。
	ContextWindow int64
	MaxOutput     int64
	// Extra 额外环境变量，最后覆盖上去。
	Extra map[string]string
}

// Driver 一系 Agent 的专门知识。
type Driver interface {
	// Kind claude | codex | antigravity；Generic 为 custom。
	Kind() string
	// Protocol 这一系接第三方接口用的协议：anthropic | openai（见 agentapi.Protocol*）；Generic 为空。
	Protocol() string
	// Adapter 在服务器 PATH 里找的 ACP 适配器命令；Generic 为空。
	Adapter() string
	// Launch 用找到的适配器启动 ACP 服务：path 是 Momoko 下载的那一份（managed 为 true）或服务器 PATH 里的 Adapter()。
	// 有的适配器要带子命令和参数才是 ACP 服务（agy acp），下载的那一份可能还要关掉它自己的自动更新。
	Launch(path string, managed bool) Launch
	// SessionData 把这个会话 Agent 自己的数据（会话记录、生成的文件）放进 dir 时，启动要多带的参数；
	// 要先在 dir 里放好的（比如链到默认目录的登录凭据）也由驱动放好。dir 是 Momoko 给这个会话的目录，删会话时整个删掉。
	// 放不了的为 nil：数据照旧在 Agent 自己的配置目录里。
	SessionData(dir string) []string
	// Unrestricted 启动时就让 Agent 不再询问任何操作（Antigravity 的 Turbo）要多带的参数；只能在开始会话前选，
	// 会盖过会话里的权限模式。没有这种启动方式的为 nil。
	Unrestricted() []string
	// Detect Agent 初始化时自报的名字是不是这一系的。
	Detect(agentName string) bool
	// Release 能由 Momoko 下载的发行包；这一系（或这个系统）没有时 ok 为 false。
	Release() (agentinstall.Release, bool)

	// APIEnv 用第三方接口启动时的环境：接口地址、密钥、模型都靠环境变量传进去。
	APIEnv(api API) map[string]string
	// LoginLimits 用服务器上的登录启动时，怎么把用户给模型设的最长上下文和最长输出（token 数，0 = 不设）交给 Agent：
	// 启动时带的环境变量。登录和模型都还是 Agent 自己的。
	LoginLimits(contextWindow, maxOutput int64) map[string]string
	// RemoveTranscript 删掉 Agent 侧这个会话的记录（在默认配置目录里按会话 id 找）。cwd 是会话的工作目录。
	RemoveTranscript(cwd, sessionID string)

	// Login 在终端里登录服务器上的账号用的命令：adapter 是这一系 ACP 适配器的可执行文件。
	// 这一系登录不了（或者服务器上没有登录要用的命令）时 ok 为 false。
	Login(adapter string) (launch Launch, ok bool)
	// SandboxEnv 在沙箱里起时另外带上的环境：关掉用不着的联网（遥测、错误上报、自动更新）。
	SandboxEnv() map[string]string

	// Command 从工具调用的原始输入里取要执行的命令；不是命令类工具为空。
	Command(rawInput json.RawMessage) string
	// ToolOutput 这一系放在 content 以外的工具结果：从工具调用的 rawOutput 里取给人看的文字，替换已有输出；没有为空。
	ToolOutput(rawOutput json.RawMessage) string
}

// Generic 认不出是哪一系的 ACP Agent：用它自己的登录和配置，只按协议正文办事。
type Generic struct{}

func (Generic) Kind() string                               { return "custom" }
func (Generic) Protocol() string                           { return "" }
func (Generic) Adapter() string                            { return "" }
func (Generic) Launch(path string, _ bool) Launch          { return Launch{Command: path} }
func (Generic) SessionData(string) []string                { return nil }
func (Generic) Unrestricted() []string                     { return nil }
func (Generic) Detect(string) bool                         { return false }
func (Generic) Release() (agentinstall.Release, bool)      { return agentinstall.Release{}, false }
func (Generic) APIEnv(API) map[string]string               { return nil }
func (Generic) LoginLimits(int64, int64) map[string]string { return nil }
func (Generic) RemoveTranscript(_, _ string)               {}
func (Generic) Login(string) (Launch, bool)                { return Launch{}, false }
func (Generic) SandboxEnv() map[string]string              { return nil }
func (Generic) Command(json.RawMessage) string             { return "" }
func (Generic) ToolOutput(json.RawMessage) string          { return "" }
