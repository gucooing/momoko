package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"time"
)

type Menu struct {
	ent.Schema
}

func (Menu) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Comment("菜单id"),
		field.Enum("type").Values("directory", "menu", "button").Comment("菜单属性"),
		field.String("title").NotEmpty().Comment("标题名称"),
		field.String("icon").NotEmpty().Comment("图标"),
		field.Bool("is_builtin").Default(false).Comment("是否内置角色(不可修改)"),
		field.Time("create_time").Default(time.Now).Immutable().Comment("创建时间"),
		field.Time("update_time").Default(time.Now).UpdateDefault(time.Now).Comment("更新时间"),
		field.Enum("status").Values("active", "inactive").Default("active").Comment("启用状态"),
		field.String("parent_id").Nillable().Comment("父菜单id"),
		field.Int("order").Comment("排序"),
		field.String("permission").Comment("权限标识"),
	}
}
