package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Session struct {
	ent.Schema
}

func (Session) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("会话id"),
		field.String("device_id").NotEmpty().Comment("设备id"),
		field.String("device").NotEmpty().Comment("登录设备"),
		field.String("user_id").NotEmpty().Comment("所属用户id"),
		field.String("ip").Comment("登录ip"),
		field.String("access_noise").NotEmpty().Comment("access token 随机噪声"),
		field.String("refresh_noise").NotEmpty().Comment("refresh token 随机噪声"),
		field.Time("expires_at").Comment("会话（refresh）过期时间"),
	}
}

func (Session) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Field("user_id").
			Unique().
			Required().
			Comment("所属用户"),
	}
}

func (Session) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("expires_at"),
		index.Fields("device_id", "user_id").Unique(), // 同一个用户在同一个设备上只能有一条登录记录
	}
}

func (Session) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
