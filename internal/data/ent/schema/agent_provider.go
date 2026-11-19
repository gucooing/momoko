package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// AgentModelLevel 推理等级：Name 是会话里能选的等级，Value 是选中它时实际发给接口的值（空 = 不带这个参数）。
type AgentModelLevel struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// AgentModel 模型目录里的一个模型：推理等级，以及最长上下文和最长输出（token 数，启动 Agent 时交给它）。
type AgentModel struct {
	ID            string            `json:"id"`
	Levels        []AgentModelLevel `json:"levels"`
	ContextWindow int64             `json:"contextWindow"`
	MaxOutput     int64             `json:"maxOutput"`
}

// AgentEnv 启动 Agent 进程时额外带上的环境变量。
type AgentEnv struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// AgentProvider 模型供应商：一个第三方接口（Anthropic 兼容给 Claude 系，OpenAI 兼容给 Codex 系）。
// 官方账号不在这里：用户自己在服务器上登录命令行，Momoko 只检查（见运行时的检测）。
type AgentProvider struct {
	ent.Schema
}

func (AgentProvider) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("供应商 id"),
		field.String("vendor").NotEmpty().Comment("接口协议：anthropic / openai"),
		field.String("name").NotEmpty().Comment("名称"),
		field.String("status").Default("untested").Comment("ok / bad / untested"),
		field.String("status_text").Default("").Comment("测试失败的原因"),
		field.String("base_url").Default("").Comment("接口地址"),
		field.Text("api_key").Default("").Sensitive().Comment("API Key（secretbox 加密）"),
		field.JSON("models", []AgentModel{}).Optional().Comment("模型目录"),
		field.String("main_model").Default("").Comment("主模型 / 默认模型"),
		field.String("fast_model").Default("").Comment("快速模型"),
		field.JSON("env", []AgentEnv{}).Optional().Comment("额外环境变量"),
		field.Int64("test_latency_ms").Default(0).Comment("最近一次测试的耗时"),
		field.Time("tested_at").Optional().Nillable().Comment("最近一次测试时间"),
	}
}

func (AgentProvider) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Annotation{Table: "agent_providers"},
	}
}

func (AgentProvider) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
