package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PassportOidcClient 用户通行证 OIDC Provider 的客户端（游戏 SDK「用户通行证登录」等），由管理员在通行证后台管理。
// 与 Momoko 管理后台的 OIDC 客户端（oidc_clients）完全独立。
type PassportOidcClient struct {
	ent.Schema
}

func (PassportOidcClient) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("id"),
		field.String("name").NotEmpty().Comment("应用名称（授权确认页展示）"),
		field.String("client_id").NotEmpty().Unique().Comment("Client ID"),
		field.String("client_secret").NotEmpty().Sensitive().Comment("Client Secret"),
		field.JSON("redirect_uris", []string{}).Comment("回调地址"),
		field.JSON("scopes", []string{}).Comment("允许的授权范围"),
		field.Bool("active").Default(true).Comment("是否启用"),
		field.String("description").Default("").Comment("描述"),
		field.String("server_id").Default("").Comment("关联的游戏服务器（展示游戏图标用，可空）"),
		field.String("created_by_name").Default("").Comment("创建者名称快照"),
		field.Time("last_used_at").Optional().Nillable().Comment("最近一次成功换取令牌的时间"),
	}
}

func (PassportOidcClient) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// PassportOidcGrant 玩家授权过的应用（「授权的应用」列表）；撤销即删除，下次登录需重新确认，
// 已签发的访问令牌在 userinfo 校验时随之失效。
type PassportOidcGrant struct {
	ent.Schema
}

func (PassportOidcGrant) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("id"),
		field.String("user_id").NotEmpty().Comment("通行证用户 id"),
		field.String("client_id").NotEmpty().Comment("客户端（passport_oidc_clients.id）"),
		field.Strings("scopes").Optional().Comment("已同意的授权范围"),
		field.Time("last_used_at").Default(time.Now).Comment("最近一次使用（授权或换取令牌）的时间"),
	}
}

func (PassportOidcGrant) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "client_id").Unique(),
		index.Fields("client_id"),
	}
}

func (PassportOidcGrant) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
