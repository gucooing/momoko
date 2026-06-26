package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FrpTunnel 记录用户配置的内网穿透（frp）隧道规则。
// 与端口转发不同：隧道监听由远端 frpc 声明 NewProxy 后才开启，momoko 仅做授权与统计。
type FrpTunnel struct {
	ent.Schema
}

func (FrpTunnel) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("id"),
		field.String("name").NotEmpty().Unique().Comment("frp proxy 名称，frpc 声明需一致，全局唯一"),
		field.String("user_id").NotEmpty().Comment("所属用户id"),
		field.Enum("proxy_type").
			Values("tcp", "udp", "http", "https", "stcp", "xtcp", "tcpmux").
			Default("tcp").
			Comment("代理类型"),
		field.Int("remote_port").Default(0).Comment("tcp/udp/tcpmux 公共端口"),
		field.String("custom_domains").Default("").Comment("http/https 自定义域名，逗号分隔"),
		field.String("subdomain").Default("").Comment("http/https 子域名"),
		field.String("local_ip").Default("127.0.0.1").Comment("frpc 端本地地址（生成 frpc 配置用）"),
		field.Int("local_port").Default(0).Comment("frpc 端本地端口（生成 frpc 配置用）"),
		field.String("credential").NotEmpty().Comment("每隧道独立认证凭证，置于 frpc metadatas.credential"),
		field.String("allow_users").Default("").Comment("允许的 frpc user，逗号分隔；空=不限制"),
		field.Bool("is_enable").Default(false).Comment("是否启用"),
		field.String("max_bandwidth").Default("").Comment("带宽上限/限速(如 1MB)；客户端声明带宽不得超过；空=不限"),
		field.Int("max_active_conns").Default(0).Comment("活跃连接数上限，0=不限；momoko 在每条用户连接时校验"),
	}
}

func (FrpTunnel) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Field("user_id").
			Unique().
			Required().
			Comment("所属用户"),
	}
}

func (FrpTunnel) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("is_enable"),
	}
}

func (FrpTunnel) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
