package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// SSHHost 表示可由浏览器连接的 SSH 服务端。
type SSHHost struct {
	ent.Schema
}

func (SSHHost) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("SSH服务端id"),
		field.String("name").NotEmpty().Comment("名称"),
		field.String("host").NotEmpty().Comment("主机地址"),
		field.Int("port").Default(22).Positive().Comment("端口"),
		field.String("username").NotEmpty().Comment("用户名"),
		field.Enum("auth_type").Values("password", "key").Comment("认证方式"),
		field.Text("credential").Sensitive().Comment("加密后的密码或私钥"),
		field.String("passphrase").Optional().Sensitive().Comment("加密后的私钥口令"),
		field.String("fingerprint").Optional().Comment("SSH主机指纹"),
		field.String("remark").Optional().Comment("备注"),
		field.String("tags").Optional().Comment("标签"),
		field.Enum("status").Values("unknown", "online", "offline").Default("unknown").Comment("连接状态"),
	}
}

func (SSHHost) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("owner", User.Type).Unique().Required().Comment("创建者"),
		edge.To("shared_users", User.Type).
			StorageKey(edge.Table("ssh_host_shares"), edge.Columns("ssh_host_id", "user_id")).
			Comment("分享用户"),
	}
}

func (SSHHost) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
