package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// 用户信息
type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().NotEmpty().Comment("用户id"),
		field.String("username").Unique().NotEmpty().Comment("用户名"),
		field.String("password").Comment("密码"),
		field.String("email").Unique().Comment("邮箱"),
		field.Enum("status").Values("inactive", "active").Default("inactive").Comment("账号状态"),
		field.String("avatar").Default("").Comment("头像图片链接（对象存储里的公开链接）"),
		field.String("avatar_key").Default("").Comment("头像在对象存储里的对象 key（换头像、删用户时按它删除旧对象）"),
		field.String("bio").Optional().Comment("个人简介"),
		field.String("name").Default("gucooing").Comment("昵称"),
		field.String("tags").Optional().Comment("标签"),
		field.String("phone").Default("").Comment("手机号"),
		field.Time("last_login_at").Optional().Nillable().Comment("最近一次登录时间"),
		field.Time("password_updated_at").Optional().Nillable().Comment("最近一次修改密码时间"),
		field.String("last_login_ip").Optional().Comment("最近一次登录 IP"),
		field.String("last_login_method").Optional().Comment("最近一次登录方式：password / email_code"),
		field.Text("appearance").Optional().Comment("外观偏好（JSON：当前主题 / 深浅色 / 紧凑 / 减少动效 / 圆角 / 标签栏 / 未存进主题的背景）"),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("roles", Role.Type).StorageKey(edge.Table("user_roles"), edge.Columns("user_id", "role_id")).Comment("拥有的角色（多个，权限取并集）"),
		edge.From("shared_ssh_hosts", SSHHost.Type).Ref("shared_users").Comment("被分享的SSH服务端"),
		edge.From("sessions", Session.Type).Ref("user").Comment("登录会话"),
	}
}

func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
