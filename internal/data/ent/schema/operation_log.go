package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type OperationLog struct {
	ent.Schema
}

func (OperationLog) Fields() []ent.Field {
	return []ent.Field{
		field.String("user_id").Optional().Nillable().Comment("操作用户id"),
		field.String("operation_type").NotEmpty().Comment("操作类型"),
		field.Bool("success").Default(true).Comment("操作结果是否成功"),
		field.Text("detail").Optional().Comment("操作详情"),
		field.String("ip").Default("").Comment("客户端IP"),
		field.String("user_agent").Default("").Comment("客户端User-Agent"),
		field.String("path").Default("").Comment("请求路径"),
		field.Int64("duration_ms").Default(0).Comment("操作耗时毫秒"),
		field.Time("operation_time").
			Default(time.Now).
			Immutable().
			Comment("操作时间"),
	}
}

func (OperationLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Field("user_id").
			Unique().
			Comment("操作用户"),
	}
}

func (OperationLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("operation_type"),
		index.Fields("path"),
		index.Fields("operation_time"),
	}
}
