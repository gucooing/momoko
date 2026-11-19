// Package agentdriver 定义「一系 Agent」和别家不一样的地方：适配器叫什么、配置目录和第三方接口的环境变量怎么给、
// 会话记录放在哪、能不能由 Momoko 下载、工具调用怎么读。官方账号由用户自己在服务器上登录命令行，Momoko 不管。
//
// 每一系在自己的子包里实现 Driver（claude、codex），对某一家的专门处理只写在它自己的子包里；
// 认不出是哪一系的 ACP Agent 用 Generic，只按协议正文办事。会话引擎（pkg/agentrun）只通过这个接口问这些事。
package agentdriver

import (
	"encoding/json"

	"momoko/pkg/agentinstall"
)

// Launch 怎么启动一个 Agent 进程。
type Launch struct {
	Command string
	Args    []string
	// Env 启动命令自带的环境（在当前进程环境之上）。
	Env map[string]string
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
	// Kind claude | codex；Generic 为 custom。
	Kind() string
	// Vendor 这一系能接的第三方接口协议：anthropic | openai；Generic 为空。
	Vendor() string
	// Adapter 在服务器 PATH 里找的 ACP 适配器命令；Generic 为空。
	Adapter() string
	// Detect Agent 初始化时自报的名字是不是这一系的。
	Detect(agentName string) bool
	// Release 能由 Momoko 下载的发行包；这一系（或这个系统）没有时 ok 为 false。
	Release() (agentinstall.Release, bool)

	// Home Agent 默认的配置目录：服务器上的登录和它自己的会话记录都在这里。所有会话都用它，不另外指定。
	Home() string
	// APIEnv 用第三方接口启动时的环境：接口地址、密钥、模型都靠环境变量传进去。
	APIEnv(api API) map[string]string
	// LoginLimits 用服务器上的登录启动时，怎么把用户给模型设的最长上下文和最长输出（token 数，0 = 不设）交给 Agent：
	// 环境变量和追加的启动参数。登录和模型都还是 Agent 自己的。
	LoginLimits(contextWindow, maxOutput int64) (env map[string]string, args []string)
	// RemoveTranscript 删掉 Agent 侧这个会话的记录（在默认配置目录里按会话 id 找）。cwd 是会话的工作目录。
	RemoveTranscript(cwd, sessionID string)

	// Command 从工具调用的原始输入里取要执行的命令；不是命令类工具为空。
	Command(rawInput json.RawMessage) string
}

// Generic 认不出是哪一系的 ACP Agent：用它自己的登录和配置，只按协议正文办事。
type Generic struct{}

func (Generic) Kind() string                                           { return "custom" }
func (Generic) Vendor() string                                         { return "" }
func (Generic) Adapter() string                                        { return "" }
func (Generic) Detect(string) bool                                     { return false }
func (Generic) Release() (agentinstall.Release, bool)                  { return agentinstall.Release{}, false }
func (Generic) Home() string                                           { return "" }
func (Generic) APIEnv(API) map[string]string                           { return nil }
func (Generic) LoginLimits(int64, int64) (map[string]string, []string) { return nil, nil }
func (Generic) RemoveTranscript(_, _ string)                           {}
func (Generic) Command(json.RawMessage) string                         { return "" }
