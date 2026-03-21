package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type SystemConfig struct {
	ent.Schema
}

func (SystemConfig) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").Unique().NotEmpty().Comment("配置键"),
		field.String("value").Default("").Comment("配置值"),
	}
}

func (SystemConfig) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Annotation{Table: "configs"},
	}
}

func (SystemConfig) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
