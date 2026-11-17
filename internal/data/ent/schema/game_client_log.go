package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GameClientLog 游戏客户端上传的错误日志（凭游戏服务端签发的上传票据传到通行证 server）。
// 每个游戏账号（server_id + player_id）只保留最新 5 份；日志原文 gzip 压缩后保存。
// 管理端的客户端日志页按上传时间倒序列出全部服务器的日志，可按服务器、玩家、原因、上传时间筛选。
type GameClientLog struct {
	ent.Schema
}

func (GameClientLog) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Immutable().Comment("日志 id（uuid）"),
		field.String("server_id").NotEmpty().Immutable().Comment("服务器 id"),
		field.String("player_id").NotEmpty().Immutable().Comment("游戏内玩家 id（MGP pid）"),
		field.String("reason").Default("").Immutable().Comment("上传原因：exception 异常 / crash 崩溃 / fatal 致命错误（客户端自动上传）/ manual 玩家手动上传"),
		field.Text("summary").Default("").Immutable().Comment("摘要（最多 512 字）"),
		field.Text("note").Default("").Immutable().Comment("玩家填写的说明（最多 500 字）"),
		field.String("device").Default("").Immutable().Comment("设备（最多 200 字）"),
		field.String("client_version").Default("").Immutable().Comment("客户端版本（最多 64 字）"),
		field.Int("size").Default(0).Immutable().Comment("日志原文大小（字节，未压缩）"),
		field.Bytes("content").
			Immutable().
			SchemaType(map[string]string{dialect.MySQL: "longblob"}).
			Comment("日志原文（gzip 压缩）"),
		field.Time("occurred_at").Immutable().Comment("客户端记录日志的时间"),
		field.Time("created_at").Default(time.Now).Immutable().Comment("上传时间"),
	}
}

func (GameClientLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("server_id", "player_id", "created_at"),
		index.Fields("server_id", "created_at"),
		index.Fields("reason", "created_at"),
		index.Fields("created_at"),
	}
}
