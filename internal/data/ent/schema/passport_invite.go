package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PassportInvite 通行证注册邀请码（注册方式为「仅邀请」时使用；也可附带一台服务器的绑定资格）。
type PassportInvite struct {
	ent.Schema
}

func (PassportInvite) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("id"),
		field.String("code").NotEmpty().Unique().Comment("邀请码"),
		field.String("email").Default("").Comment("限定邮箱（空不限）"),
		field.String("server_id").Default("").Comment("附带的服务器（空不限）"),
		field.Int("max_uses").Default(1).Comment("可用次数"),
		field.Int("used").Default(0).Comment("已用次数"),
		field.Time("expires_at").Optional().Nillable().Comment("过期时间"),
		field.String("created_by").Default("").Comment("创建者（Momoko 管理员）id"),
		field.String("created_by_name").Default("").Comment("创建者名称快照"),
		field.String("note").Default("").Comment("备注（只给管理员看）"),
		field.Bool("revoked").Default(false).Comment("已作废"),
	}
}

func (PassportInvite) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("create_time"),
	}
}

func (PassportInvite) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
