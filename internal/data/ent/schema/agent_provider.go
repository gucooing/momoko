package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
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

// AgentProvider 模型供应商：一个第三方接口。按类型（pkg/agentapi 的 Vendor）决定给哪几系 Agent 用、地址填不填。
// 属于添加它的用户，只有他自己用得上；管理员在 Agent 管理页加的是共享的（主人 @shared），共享给所有人或指定的人。
// 官方账号不在这里：用户自己在服务器上登录命令行，Momoko 只检查（见运行时的检测）。
type AgentProvider struct {
	ent.Schema
}

func (AgentProvider) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("供应商 id"),
		field.String("user_id").Default("").Comment("所属用户 id：每个人只看得到、只用得上自己的（以前没有主人的升级时改成共享给所有人）"),
		field.String("vendor").NotEmpty().Comment("供应商类型：anthropic / openai / deepseek …（见 pkg/agentapi）"),
		field.String("name").NotEmpty().Comment("名称"),
		field.String("status").Default("untested").Comment("ok / bad / untested"),
		field.String("status_text").Default("").Comment("测试失败的原因"),
		field.String("base_url").Default("").Comment("用户填的接口地址；地址固定的类型为空"),
		field.Text("api_key").Default("").Sensitive().Comment("API Key（secretbox 加密）"),
		field.JSON("models", []AgentModel{}).Optional().Comment("模型目录"),
		field.String("main_model").Default("").Comment("主模型 / 默认模型"),
		field.String("fast_model").Default("").Comment("快速模型"),
		field.JSON("env", []AgentEnv{}).Optional().Comment("额外环境变量"),
		field.Int64("test_latency_ms").Default(0).Comment("最近一次测试的耗时"),
		field.Time("tested_at").Optional().Nillable().Comment("最近一次测试时间"),
		field.String("share").Default("").Comment("共享范围：空 = 只有主人自己用；all = 所有人；users = share_users 里的人。共享的主人是 @shared，在 Agent 管理页里管"),
		field.Strings("share_users").Optional().Comment("share 为 users 时能用的用户 id"),
		field.Text("proxy").Default("").Sensitive().Comment("代理节点（secretbox 加密的 JSON：分享链接或 sing-box 出站，连同沙箱的时区和语言）；空 = 不走代理。只有 Linux 能配，见 pkg/agentsandbox"),
	}
}

func (AgentProvider) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
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
