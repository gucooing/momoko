package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentChoice Agent 给的一个可选值（推理强度）。
type AgentChoice struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// AgentRuntimeModel 服务器上的登录能用的一个模型。名称和推理强度在检查登录时从 Agent 同步过来；
// 最长上下文和最长输出是用户在管理页设的，同步不动它们，起进程时按（运行时，模型）查出来交给 Agent。
type AgentRuntimeModel struct {
	ent.Schema
}

func (AgentRuntimeModel) Fields() []ent.Field {
	return []ent.Field{
		field.String("runtime_id").NotEmpty().Comment("运行时 id"),
		field.String("model").NotEmpty().Comment("Agent 给的模型 id"),
		field.String("name").Default("").Comment("Agent 给的名称"),
		field.JSON("efforts", []AgentChoice{}).Optional().Comment("能选的推理强度"),
		field.String("default_effort").Default("").Comment("默认的推理强度"),
		field.Int64("context_window").Default(0).Comment("用户设的最长上下文（token），0 = 用 Agent 自己的"),
		field.Int64("max_output").Default(0).Comment("用户设的最长输出（token），0 = 用 Agent 自己的"),
		field.Int("sort").Default(0).Comment("Agent 列出的顺序"),
	}
}

func (AgentRuntimeModel) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("runtime_id", "model").Unique(),
	}
}

func (AgentRuntimeModel) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Annotation{Table: "agent_runtime_models"},
	}
}

func (AgentRuntimeModel) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
