package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserAPIKey 记录用户创建的 API Key。
// API Key 是节点鉴权的签名种子：主控只发送由种子派生的 key_id 与时间戳签名，种子本身不在网络上传输。
type UserAPIKey struct {
	ent.Schema
}

func (UserAPIKey) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("API Key id"),
		field.String("name").NotEmpty().Comment("名称"),
		field.String("api_key").NotEmpty().Unique().Sensitive().Comment("API Key（签名种子）"),
		field.String("key_id").Optional().Unique().Comment("由种子派生的公开密钥标识，签名请求据此定位密钥"),
		field.JSON("permissions", []string{}).Optional().Comment("授予的节点权限标识"),
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
