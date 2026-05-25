package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserAPIKey 记录用户创建的 API Key。
type UserAPIKey struct {
	ent.Schema
}

func (UserAPIKey) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("API Key id"),
		field.String("name").NotEmpty().Comment("名称"),
		field.String("api_key").NotEmpty().Unique().Sensitive().Comment("API Key"),
		field.Time("expires_at").Optional().Nillable().Comment("过期时间，为空表示永久有效"),
		field.String("user_id").NotEmpty().Comment("所属用户id"),
	}
}

func (UserAPIKey) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Field("user_id").
			Unique().
			Required().
			Comment("所属用户"),
	}
}

func (UserAPIKey) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
	}
}

func (UserAPIKey) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
