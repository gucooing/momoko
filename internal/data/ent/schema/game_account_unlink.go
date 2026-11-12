package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GameAccountUnlink 解绑记录：解绑后 7 天内不能在同一服务器绑定别的游戏账号。
type GameAccountUnlink struct {
	ent.Schema
}

func (GameAccountUnlink) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("记录 id"),
		field.String("passport_user_id").NotEmpty().Comment("通行证用户 id（passport_users）"),
		field.String("server_id").NotEmpty().Comment("服务器 id"),
		field.String("player_id").Default("").Comment("解绑前的玩家 id"),
		field.String("nickname").Default("").Comment("解绑前的昵称快照"),
		field.Enum("by").Values("self", "admin").Default("self").Comment("谁解绑的"),
		field.Time("unlinked_at").Default(time.Now).Comment("解绑时间"),
	}
}

func (GameAccountUnlink) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("passport_user_id", "server_id", "unlinked_at"),
	}
}
