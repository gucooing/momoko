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
	}
}
