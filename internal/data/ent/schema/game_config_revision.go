package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GameConfigRevision 运营配置版本（卡池、公告等配置域）：草稿与已发布的历史，可对比、回滚。
// 每个服务器每个配置域最多一份草稿（state=draft）；发布 = 带 revision 调 MGP PUT /configs/{domain}。
type GameConfigRevision struct {
	ent.Schema
}

func (GameConfigRevision) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("版本 id"),
		field.String("server_id").NotEmpty().Comment("服务器 id"),
		field.String("domain").NotEmpty().Comment("配置域 id"),
		field.Int("seq").Default(0).Comment("Momoko 侧版本号（每个配置域递增，r1、r2…；草稿为 0）"),
		field.String("revision").Default("").Comment("服务端 revision（内容 SHA-256；草稿为空）"),
		field.String("base_revision").Default("").Comment("编辑所基于的服务端 revision"),
		field.Text("content").Comment("配置内容（JSON）"),
		field.String("author_id").Default("").Comment("作者用户 id"),
		field.String("author_name").Default("").Comment("作者名称快照"),
		field.String("comment").Default("").Comment("说明"),
		field.Enum("state").Values("draft", "published").Default("draft").Comment("状态：草稿 / 已发布"),
		field.Time("published_at").Optional().Nillable().Comment("发布时间"),
		field.String("execution_id").Default("").Comment("发布对应的 GM 记录 id"),
	}
}

func (GameConfigRevision) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("server_id", "domain", "state"),
		index.Fields("server_id", "domain", "create_time"),
	}
}

func (GameConfigRevision) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
