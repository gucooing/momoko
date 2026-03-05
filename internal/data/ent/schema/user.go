package schema

import (
	"time"

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
		field.String("email").Comment("邮箱"),
		field.Time("create_time").Default(time.Now).Immutable().Comment("创建时间"),
		field.Time("update_time").Default(time.Now).UpdateDefault(time.Now).Comment("更新时间"),
		field.Enum("status").Values("inactive", "active", "freeze").Default("inactive").Comment("账号状态"),
		field.String("avatar").Default("").Comment("头像"),
		field.String("bio").NotEmpty().Comment("个人简介"),
		field.String("name").Default("gucooing").Comment("昵称"),
		field.String("tags").NotEmpty().Comment("标签"),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("role", Role.Type).Unique().Comment("关联的角色"),
	}
}
