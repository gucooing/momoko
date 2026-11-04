package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// IQCheckPrompt 智商检测提示词：每轮检测把全部启用的提示词逐个发给站点的每个模型。
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
		field.Enum("category").
			Values("other", "logic", "math", "coding", "drawing", "writing").
			Default("other").
			Comment("分类：other=其他，logic=逻辑推理，math=数学，coding=编程，drawing=画图，writing=写作"),
		field.String("grading").Default("").Comment("评分方式说明"),
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
