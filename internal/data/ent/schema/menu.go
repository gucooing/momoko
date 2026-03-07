package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Menu struct {
	ent.Schema
}

func (Menu) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Comment("菜单id"),
		field.Enum("type").Values("directory", "menu", "button").Comment("菜单属性"),
		field.String("path").NotEmpty().Comment("路径"),
		field.String("title").NotEmpty().Comment("名称"),
		field.String("permission").NotEmpty().Comment("权限标识"),
		field.Int("order").Comment("排序"),
		field.String("icon").NotEmpty().Comment("图标"),
		field.Bool("is_system").Default(false).Comment("是否系统默认菜单(不可修改)"),
		field.Enum("status").Values("active", "inactive").Default("active").Comment("启用状态"),
		field.String("parent_id").Nillable().Comment("父菜单id"),
	}
}

func (Menu) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
