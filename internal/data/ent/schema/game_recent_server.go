package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GameRecentServer 最近查看的游戏服务器（服务器搜索下拉的「最近查看」）。
// scope=admin 时 user_id 是 Momoko 管理员；scope=passport 时是通行证用户（passport_users）。
type GameRecentServer struct {
	ent.Schema
}

func (GameRecentServer) Fields() []ent.Field {
	return []ent.Field{
		field.String("user_id").NotEmpty().Comment("用户 id"),
		field.String("server_id").NotEmpty().Comment("服务器 id"),
		field.Enum("scope").Values("admin", "passport").Default("admin").Comment("在哪一端查看：admin 管理端 / passport 通行证"),
		field.Time("viewed_at").Default(time.Now).Comment("最近查看时间"),
	}
}

func (GameRecentServer) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "scope", "server_id").Unique(),
		index.Fields("user_id", "scope", "viewed_at"),
	}
}
