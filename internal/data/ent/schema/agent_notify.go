package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// AgentNotifyChannel 一种通知方式（discord / telegram / feishu / qq）的配置：启用没有，和要填的配置项（密钥类存密文）。
type AgentNotifyChannel struct {
	Enabled bool              `json:"enabled"`
	Fields  map[string]string `json:"fields"`
}

// AgentNotify 一个人的 Agent 通知设置：三个时机开关和各种通知方式。每个人只管自己的，
// 他的会话到了时机就按他的设置发；管理员在 Agent 管理页能看到所有人的，能停用或移除。没设置过的人没有这一行，按默认（三个时机都开、没有通知方式）。
type AgentNotify struct {
	ent.Schema
}

func (AgentNotify) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("用户 id"),
		field.Bool("on_ask").Default(true).Comment("等待批准、有问题要问时通知"),
		field.Bool("on_done").Default(true).Comment("一轮结束时通知"),
		field.Bool("on_error").Default(true).Comment("出错时通知"),
		field.JSON("channels", map[string]AgentNotifyChannel{}).Optional().Comment("通知方式：discord / telegram / feishu / qq → 配置"),
	}
}

func (AgentNotify) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Annotation{Table: "agent_notify"},
	}
}

func (AgentNotify) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
