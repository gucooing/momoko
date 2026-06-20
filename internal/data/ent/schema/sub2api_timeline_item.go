package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Sub2APITimelineItem 首页更新时间线，独立于 Sub2API 连接配置管理。
type Sub2APITimelineItem struct {
	ent.Schema
}

func (Sub2APITimelineItem) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("id"),
		field.String("title").Optional().Comment("标题"),
		field.String("content").Optional().Comment("内容"),
		field.String("category").Default("更新").Comment("分类"),
		field.Time("published_at").Optional().Nillable().Comment("发布时间"),
	}
}

func (Sub2APITimelineItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("published_at"),
	}
}

func (Sub2APITimelineItem) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
