package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PortForward 记录用户配置的协议转发规则。
type PortForward struct {
	ent.Schema
}

func (PortForward) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("id"),
		field.String("name").NotEmpty().Comment("名称"),
		field.String("user_id").NotEmpty().Comment("所属用户id"),
		field.Enum("protocol").Values("tcp", "udp").Comment("协议"),
		field.String("listen_address").Default("0.0.0.0").Comment("监听地址"),
		field.Int("listen_port").Default(0).Comment("监听端口"),
		field.String("target_address").Default("0.0.0.0").Comment("目标地址"),
		field.Int("target_port").Default(0).Comment("目标端口"),
		field.Bool("is_enable").Default(false).Comment("是否启用"),
		field.String("remark").Optional().Comment("备注"),
		field.String("tags").Optional().Comment("标签"),
	}
}

func (PortForward) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Field("user_id").
			Unique().
			Required().
			Comment("所属用户"),
	}
}

func (PortForward) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("is_enable"),
	}
}

func (PortForward) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
