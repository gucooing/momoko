package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// AgentRuntime 运行时：一个能以 ACP 方式启动的 Agent。claude 和 codex 两条内置，其它是用户添加的。
type AgentRuntime struct {
	ent.Schema
}

func (AgentRuntime) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("运行时 id：claude / codex / 自定义的随机 id"),
		field.String("name").NotEmpty().Comment("名称"),
		field.Enum("kind").Values("claude", "codex", "custom").Default("custom").Comment("类型"),
		field.Bool("managed").Default(false).Comment("由 Momoko 下载并更新"),
		field.String("command").Default("").Comment("启动命令"),
		field.String("version").Default("").Comment("Momoko 下载的版本"),
		field.Bool("ready").Default(false).Comment("最近一次检测是否通过"),
		field.String("agent_name").Default("").Comment("Agent 自报的名字"),
		field.String("agent_version").Default("").Comment("Agent 自报的版本"),
		field.Text("error").Default("").Comment("检测失败原因"),
		field.Time("detected_at").Optional().Nillable().Comment("最近一次检测时间"),
		field.Text("caps").Default("").Comment("Agent 声明的能力（JSON）"),
		field.String("auth_label").Default("").Comment("服务器上已有的登录：账号名称"),
		field.String("auth_detail").Default("").Comment("服务器上已有的登录：说明"),
		field.Bool("logged_in").Default(false).Comment("最近一次检查时服务器上已经登录（能开会话）"),
		field.Text("catalog").Default("").Comment("Agent 给出的权限模式和默认模型（JSON）；模型清单在 agent_runtime_models"),
		field.Int("sort").Default(0).Comment("排序"),
		field.Text("proxy").Default("").Sensitive().Comment("服务器上的登录走的代理 http:// 或 socks5://（secretbox 加密，可能带账号密码）；空 = 不走代理。只有 Linux 能配，见 pkg/agentsandbox"),
	}
}

func (AgentRuntime) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Annotation{Table: "agent_runtimes"},
	}
}

func (AgentRuntime) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
