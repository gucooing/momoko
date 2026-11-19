package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/constant"
)

// Agent 两个 Agent 工具真正要做的事，由外面注入。
type Agent interface {
	// ImportSkill 装技能：Source 为 url 时由实现去下载 Location 指向的 zip，git 时克隆仓库。
	ImportSkill(ctx context.Context, req *v1.ImportAgentSkillRequest) ([]*v1.AgentSkill, error)
	CreateMcpServer(ctx context.Context, in *v1.AgentMcpServer) (*v1.AgentMcpServer, error)
}

// agentTools 把技能和 MCP 服务器装进 Momoko 的 Agent 配置。
func agentTools(agent Agent) []Tool {
	return []Tool{
		{
			Name:        "install_skill",
			Description: "Install a skill into Momoko from a Git repository URL or a .zip download link. A source may contain several skills (directories with a SKILL.md); all of them are installed and enabled.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"source":   map[string]any{"type": "string", "enum": []string{"git", "zip"}, "description": "git: clone a repository; zip: download an archive"},
					"location": map[string]any{"type": "string", "description": "Git URL, or https link to the .zip file"},
				},
				"required": []string{"source", "location"},
			},
			Permission: constant.AgentManage,
			call: func(ctx context.Context, raw json.RawMessage) (string, error) {
				return installSkill(ctx, agent, raw)
			},
		},
		{
			Name:        "add_mcp_server",
			Description: "Register an MCP server in Momoko so it can be enabled for sessions. Use transport stdio with command/args/env, or transport http with url/headers.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":        map[string]any{"type": "string", "description": "Short name without spaces or slashes"},
					"description": map[string]any{"type": "string"},
					"transport":   map[string]any{"type": "string", "enum": []string{"stdio", "http"}},
					"command":     map[string]any{"type": "string", "description": "stdio: the executable"},
					"args":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"env":         map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}},
					"url":         map[string]any{"type": "string", "description": "http: the server URL"},
					"headers":     map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}},
				},
				"required": []string{"name", "transport"},
			},
			Permission: constant.AgentManage,
			call: func(ctx context.Context, raw json.RawMessage) (string, error) {
				return addMcpServer(ctx, agent, raw)
			},
		},
	}
}

// skillSources 工具参数里的来源 → 导入技能时的来源：zip 链接交给 url，由实现去下载。
var skillSources = map[string]string{"git": "git", "zip": "url"}

func installSkill(ctx context.Context, agent Agent, raw json.RawMessage) (string, error) {
	var args struct {
		Source   string `json:"source"`
		Location string `json:"location"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", errors.New("参数不对：" + err.Error())
	}
	source, ok := skillSources[args.Source]
	if !ok {
		return "", errors.New("source 只能是 git 或 zip")
	}
	skills, err := agent.ImportSkill(ctx, &v1.ImportAgentSkillRequest{Source: source, Location: args.Location})
	if err != nil {
		return "", err
	}
	names := make([]string, 0, len(skills))
	for _, s := range skills {
		names = append(names, s.Name)
	}
	return "Installed skills: " + strings.Join(names, ", "), nil
}

func addMcpServer(ctx context.Context, agent Agent, raw json.RawMessage) (string, error) {
	var args struct {
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Transport   string            `json:"transport"`
		Command     string            `json:"command"`
		Args        []string          `json:"args"`
		Env         map[string]string `json:"env"`
		URL         string            `json:"url"`
		Headers     map[string]string `json:"headers"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", errors.New("参数不对：" + err.Error())
	}
	in := &v1.AgentMcpServer{
		Name:        args.Name,
		Description: args.Description,
		Transport:   args.Transport,
		Command:     args.Command,
		Args:        args.Args,
		Url:         args.URL,
		Env:         secretKVs(args.Env),
		Headers:     secretKVs(args.Headers),
	}
	created, err := agent.CreateMcpServer(ctx, in)
	if err != nil {
		return "", err
	}
	return "Added MCP server: " + created.Name, nil
}

// secretKVs 键值对按键排好，转成接口里的列表。
func secretKVs(m map[string]string) []*v1.AgentSecretKV {
	out := make([]*v1.AgentSecretKV, 0, len(m))
	for _, key := range slices.Sorted(maps.Keys(m)) {
		out = append(out, &v1.AgentSecretKV{Key: key, Value: m[key]})
	}
	return out
}
