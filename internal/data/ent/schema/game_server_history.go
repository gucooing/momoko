package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GameServerHistory 游戏服务器的更新、资源、配置与密钥变更记录（服务器详情「更新记录」）。
type GameServerHistory struct {
	ent.Schema
}

func (GameServerHistory) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("记录 id"),
		field.String("server_id").NotEmpty().Comment("服务器 id"),
		field.String("kind").NotEmpty().Comment("类型：server 服务端 / resource 资源 / config 配置 / backup 备份 / key 密钥"),
		field.String("action").Default("").Comment("动作（前端据此本地化）"),
		field.JSON("params", map[string]string{}).Optional().Comment("参数（版本、键名与新旧值等）"),
		field.Enum("result").Values("success", "failed", "rolled_back", "running").Default("success").Comment("结果"),
		field.String("message").Default("").Comment("失败原因等补充说明"),
		field.String("operator_id").Default("").Comment("操作者用户 id（自动任务为空）"),
		field.String("operator_name").Default("").Comment("操作者名称快照"),
	}
}

func (GameServerHistory) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("server_id", "create_time"),
	}
}

func (GameServerHistory) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
