package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GameServerBackup 托管服务器的备份（数据库与数据目录 / 世界存档），可回档。
type GameServerBackup struct {
	ent.Schema
}

func (GameServerBackup) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("备份 id"),
		field.String("server_id").NotEmpty().Comment("服务器 id"),
		field.Enum("trigger").Values("manual", "schedule", "update", "restore").Default("manual").Comment("触发方式：手动 / 定时 / 更新前 / 回档前"),
		field.Strings("scope").Optional().Comment("备份的相对路径（适配器决定，如 data/、config.json）"),
		field.String("path").Default("").Comment("备份文件在节点上的路径"),
		field.Int64("size").Default(0).Comment("大小（字节）"),
		field.Enum("state").Values("running", "ready", "failed").Default("running").Comment("状态"),
		field.String("note").Default("").Comment("备注"),
		field.String("error").Default("").Comment("失败原因"),
		field.String("operator_id").Default("").Comment("操作者用户 id"),
		field.String("operator_name").Default("").Comment("操作者名称快照"),
	}
}

func (GameServerBackup) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("server_id", "create_time"),
	}
}

func (GameServerBackup) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
