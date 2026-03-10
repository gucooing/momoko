package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// 实例表
type Instance struct {
	ent.Schema
}

func (Instance) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("实例id"),
		field.String("name").NotEmpty().Comment("实例名称"),
		field.Bool("is_system").Default(false).Comment("是否内置实例"),
		field.String("user_id").NotEmpty().Comment("实例所属用户"),
		// 实例设置
		field.String("path").NotEmpty().Default("./servers").Comment("工作目录"),
		field.String("start_command").NotEmpty().Comment("启动命令"),
		// 终端设置
	}
}
