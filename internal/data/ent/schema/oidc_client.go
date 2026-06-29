package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type OIDCClient struct {
	ent.Schema
}

func (OIDCClient) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("OIDC 客户端 id"),
		field.String("name").NotEmpty().Comment("名称"),
		field.String("client_id").NotEmpty().Unique().Comment("Client ID"),
		field.String("client_secret").NotEmpty().Sensitive().Comment("Client Secret"),
		field.JSON("redirect_uris", []string{}).Comment("回调地址"),
		field.JSON("scopes", []string{}).Comment("授权范围"),
		field.Bool("active").Default(true).Comment("是否启用"),
	}
}

func (OIDCClient) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("client_id"),
	}
}

func (OIDCClient) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
