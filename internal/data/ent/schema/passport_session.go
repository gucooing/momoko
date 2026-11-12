package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PassportSession 用户通行证的登录会话（与管理后台的 sessions 表无关）。
type PassportSession struct {
	ent.Schema
}

func (PassportSession) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("会话 id"),
		field.String("user_id").NotEmpty().Comment("通行证用户 id"),
		field.String("device_id").NotEmpty().Comment("设备 id"),
		field.String("device").Default("").Comment("登录设备"),
		field.String("user_agent").Default("").Comment("登录时的 User-Agent"),
		field.String("client_id").Default("").Comment("经 OIDC 登录时的应用 Client ID"),
		field.String("login_method").Default("").Comment("登录方式：password / email_code / oidc"),
		field.String("ip").Default("").Comment("登录 IP"),
		field.String("last_ip").Default("").Comment("最近活跃 IP"),
		field.Time("last_active_at").Optional().Nillable().Comment("最近活跃时间"),
		field.String("access_noise").NotEmpty().Sensitive().Comment("访问令牌随机噪声"),
		field.String("refresh_noise").NotEmpty().Sensitive().Comment("刷新令牌随机噪声"),
		field.Bool("persistent").Default(true).Comment("保持登录：刷新时按通行证设置的会话天数续期，否则最长 1 天"),
		field.Time("expires_at").Comment("会话（刷新令牌）过期时间"),
	}
}

func (PassportSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("expires_at"),
		index.Fields("device_id", "user_id").Unique(),
	}
}

func (PassportSession) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
