package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Role struct {
	ent.Schema
}

func (Role) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().NotEmpty().Comment("角色id"),
		field.String("name").Unique().NotEmpty().Comment("角色名称"),
		field.String("description").NotEmpty().Comment("角色描述"),
		field.Bool("is_builtin").Default(false).Comment("是否内置角色(不可修改)"),
		field.Enum("status").Values("active", "inactive").Default("active").Comment("启用状态"),
	}
}

func (Role) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("menus", Menu.Type).Comment("关联的权限"),
	}
}

func (Role) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
