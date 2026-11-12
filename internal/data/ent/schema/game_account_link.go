package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GameAccountLink 通行证账号（passport_users，与 Momoko 管理后台用户无关）↔ 游戏账号。每个用户在每台服务器只能绑一个游戏账号，
// 一个游戏账号也只能绑到一个通行证。state=pending 是等待管理员确认的绑定申请（如白名单）。
type GameAccountLink struct {
	ent.Schema
}

func (GameAccountLink) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("绑定 id"),
		field.String("passport_user_id").NotEmpty().Comment("通行证用户 id（passport_users）"),
		field.String("server_id").NotEmpty().Comment("服务器 id"),
		field.String("game").Default("").Comment("游戏（冗余，按游戏分组用）"),
		field.String("player_id").Default("").Comment("游戏内玩家 id（MGP pid）；申请未通过时可为空"),
		field.String("display_id").Default("").Comment("游戏内显示的 ID"),
		field.String("account").Default("").Comment("游戏账号名 / 玩家代号"),
		field.String("nickname").Default("").Comment("昵称快照"),
		field.String("avatar_asset").Default("").Comment("头像资源 ID（偶像卡 / 角色）"),
		field.Int("level").Default(0).Comment("等级快照"),
		field.Enum("method").Values("password", "code", "oidc", "admin", "request").Comment("绑定方式：密码 / 绑定码 / 通行证登录 / 管理员 / 申请"),
		field.Enum("state").Values("active", "pending", "rejected").Default("active").Comment("状态"),
		field.String("review_note").Default("").Comment("申请被驳回的原因"),
		field.String("reviewed_by").Default("").Comment("审批的管理员（Momoko 用户）id"),
		field.Bool("is_primary").Default(false).Comment("该游戏下的常用账号"),
		field.Time("linked_at").Default(time.Now).Comment("绑定时间"),
		field.Time("last_seen_at").Optional().Nillable().Comment("最近在通行证里使用（进入游戏空间）的时间"),
	}
}

func (GameAccountLink) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("passport_user_id", "server_id").Unique(),
		index.Fields("server_id", "player_id"),
		index.Fields("server_id", "state"),
	}
}

func (GameAccountLink) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
