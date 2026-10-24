package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Node 记录主控接入的远程节点：通过节点 gRPC 端口与节点上创建的 API Key 管理该节点。
type Node struct {
	ent.Schema
}

func (Node) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("节点id"),
		field.String("name").NotEmpty().Comment("名称"),
		field.String("address").NotEmpty().Comment("节点 gRPC 地址（host:port）"),
		field.Text("api_key").NotEmpty().Sensitive().Comment("加密后的 API Key（签名种子）"),
		field.String("key_id").NotEmpty().Comment("由种子派生的公开密钥标识"),
		field.Bool("tls").Default(false).Comment("使用 TLS 连接"),
		field.Bool("tls_skip_verify").Default(false).Comment("跳过 TLS 证书校验"),
		field.String("remark").Optional().Comment("备注"),
	}
}

func (Node) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
