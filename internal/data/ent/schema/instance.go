package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
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
		field.String("remark").Optional().Comment("备注"),
		field.String("tags").Optional().Comment("标签"),
		field.Bool("is_system").Default(false).Comment("是否内置实例"),
		// 实例设置
		field.String("path").NotEmpty().Default("./servers").Comment("工作目录"),
		field.String("start_command").NotEmpty().Comment("启动命令"),
		field.String("stop_command").Default("exit").Comment("停止命令"),
		field.Bool("auto_start").Default(false).Comment("是否自启动"),
		field.JSON("env", []string{}).Optional().Comment("环境变量"),
	}
}

func (Instance) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).Unique().Required().Comment("实例所属用户"),
		edge.To("users", User.Type).Comment("分配用户"),
		edge.To("type", InstanceType.Type).Unique().Required().Comment("实例类型"),
	}
}

func (Instance) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
