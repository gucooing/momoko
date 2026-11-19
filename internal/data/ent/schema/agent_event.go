package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentEvent 会话里的一条对话事件（消息、思考、工具调用、批准、一轮结束）。
// seq 是它在会话里的位置，rev 是它最后一次变化的修订号：断线重连时补发 rev 大于客户端已知值的事件。
type AgentEvent struct {
	ent.Schema
}

func (AgentEvent) Fields() []ent.Field {
	return []ent.Field{
		field.String("session_id").NotEmpty().Comment("会话 id"),
		field.Int64("seq").Comment("序号"),
		field.Int64("rev").Comment("修订号"),
		field.String("type").NotEmpty().Comment("类型：user / assistant / thought / tool / permission / turn / notice / subagent"),
		field.String("sub").Default("").Comment("子 Agent 的会话 id，主会话为空"),
		field.Text("payload").Default("").Comment("内容（JSON）"),
		field.Time("create_time").Default(time.Now).Immutable().Comment("创建时间"),
	}
}

func (AgentEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("session_id", "seq").Unique(),
		index.Fields("session_id", "rev"),
	}
}

func (AgentEvent) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Annotation{Table: "agent_events"},
	}
}
