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
		field.Bool("auto_restart").Default(false).Comment("崩溃后自动重启"),
		field.Int32("stop_timeout").Default(0).Comment("停止超时(秒)，<=0 使用默认值"),
		field.JSON("env", []string{}).Optional().Comment("环境变量（KEY=VALUE）"),
		field.Strings("secret_env_keys").Optional().Comment("按密钥保存的环境变量名：值只写不读，控制台输出里打码"),
		field.Bool("maintenance").Default(false).Comment("维护中：不自启动、不自动重启、不允许启动"),
		field.Time("maintenance_time").Optional().Nillable().Comment("进入维护的时间"),
		field.Time("last_stop_time").Optional().Nillable().Comment("最近一次进程退出时间"),
	}
}

func (Instance) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).Unique().Required().Comment("实例所属用户"),
		edge.To("users", User.Type).Comment("分配用户"),
		edge.To("type", InstanceType.Type).Unique().Comment("实例类型（类型被删除后为空，即未分类）"),
	}
}

func (Instance) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
