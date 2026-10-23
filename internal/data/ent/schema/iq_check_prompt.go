package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// IQCheckPrompt 智商检测提示词：定时检测时从启用的提示词中随机选一条发给上游模型。
type IQCheckPrompt struct {
	ent.Schema
}

func (IQCheckPrompt) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("提示词 id"),
		field.String("title").NotEmpty().Comment("标题"),
		field.Text("content").NotEmpty().Comment("提示词内容"),
		field.Bool("enabled").Default(true).Comment("是否启用"),
		field.Int("order").Default(0).Comment("排序，越小越靠前"),
	}
}

func (IQCheckPrompt) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("enabled"),
	}
}

func (IQCheckPrompt) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Annotation{Table: "iq_check_prompts"},
	}
}

func (IQCheckPrompt) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
