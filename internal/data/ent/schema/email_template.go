package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type EmailTemplate struct {
	ent.Schema
}

func (EmailTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.String("type").
			NotEmpty().
			Unique().
			Comment("邮件模板类型"),
		field.String("subject").
			Default("").
			Comment("邮件主题 TEXT 模板"),
		field.Text("template").
			Default("").
			Comment("邮件内容 HTML 模板"),
	}
}

func (EmailTemplate) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Annotation{Table: "email_templates"},
	}
}

func (EmailTemplate) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
