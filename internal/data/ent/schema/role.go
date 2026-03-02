package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"time"
)

type Role struct {
	ent.Schema
}

func (Role) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().NotEmpty().Comment("角色id"),
		field.String("name").Unique().NotEmpty().Comment("角色名称"),
		field.Bool("is_builtin").Default(false).Comment("是否内置角色(不可修改)"),
		field.Time("create_time").Default(time.Now).Immutable().Comment("创建时间"),
		field.Time("update_time").Default(time.Now).UpdateDefault(time.Now).Comment("更新时间"),
		field.Enum("status").Values("active", "inactive").Default("active").Comment("启用状态"),
	}
}

func (Role) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("menus", Menu.Type).Comment("关联的页面"),
	}
}
