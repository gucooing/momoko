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
		field.String("user_agent").Default("").Comment("登录时的 User-Agent"),
		field.String("login_method").Default("").Comment("登录方式：password / email_code"),
		field.Time("last_active_at").Optional().Nillable().Comment("最近活跃时间"),
		field.String("last_ip").Default("").Comment("最近活跃 IP"),
		field.Bool("persistent").Default(true).Comment("保持登录：false 时会话有效期不超过 1 天（不活动即失效）"),
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
