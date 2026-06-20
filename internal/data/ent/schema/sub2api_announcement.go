package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Sub2APIAnnouncement 首页公告，独立于 Sub2API 连接配置管理。
type Sub2APIAnnouncement struct {
	ent.Schema
}

func (Sub2APIAnnouncement) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("id"),
		field.String("title").Optional().Comment("标题"),
		field.String("content").Optional().Comment("内容"),
		field.String("level").Default("info").Comment("级别：info/success/warning/danger"),
		field.Bool("pinned").Default(false).Comment("是否置顶"),
		field.Time("published_at").Optional().Nillable().Comment("发布时间"),
	}
}

func (Sub2APIAnnouncement) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("pinned"),
		index.Fields("published_at"),
	}
}

func (Sub2APIAnnouncement) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
