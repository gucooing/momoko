package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"time"
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
		field.Enum("status").Values("active", "inactive").Default("active").Comment("启用状态"),
		field.String("avatar").Comment("头像"),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("roles", Role.Type).Comment("关联的角色"),
	}
}
