package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// 实例类型表
type InstanceType struct {
	ent.Schema
}

func (InstanceType) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("类型id"),
		field.String("name").NotEmpty().Unique().Comment("类型名称"),
		field.Bool("is_system").Default(false).Comment("是否系统内置"),
		field.Bool("is_enable").Default(true).Comment("是否启用"),
		field.String("description").Default("").Comment("类型说明"),
		field.String("start_command").Default("").Comment("默认启动命令"),
		field.String("stop_command").Default("").Comment("默认停止命令"),
		field.Strings("quick_commands").Optional().Comment("控制台快捷命令（按顺序显示为一排按钮）"),
	}
}

func (InstanceType) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
