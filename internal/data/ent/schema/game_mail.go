package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GameReward 一项奖励（与 MGP gift.send / mail.broadcast 的 rewards 相同）。
type GameReward struct {
	ResourceType int32  `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	Quantity     int64  `json:"quantity"`
}

// GameMailCondition 按条件发送的收件人条件（与 MGP mail.broadcast 的 condition 相同，Unix 毫秒，0 不限）。
type GameMailCondition struct {
	CreatedSince    int64 `json:"createdSince,omitempty"`
	CreatedBefore   int64 `json:"createdBefore,omitempty"`
	LastLoginSince  int64 `json:"lastLoginSince,omitempty"`
	LastLoginBefore int64 `json:"lastLoginBefore,omitempty"`
}

// GameMail 邮件补偿：把奖励发进玩家礼物箱（指令 mail.broadcast；撤回 mail.revoke）。
type GameMail struct {
	ent.Schema
}

func (GameMail) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("邮件 id（同时作为 MGP mailId）"),
		field.String("server_id").NotEmpty().Comment("服务器 id"),
		field.String("title").Default("").Comment("标题（玩家可见，作为礼物箱文案）"),
		field.Text("body").Default("").Comment("正文（管理员备注；游戏礼物箱只显示标题）"),
		field.JSON("rewards", []GameReward{}).Optional().Comment("奖励"),
		field.Enum("target").Values("players", "condition", "all").Default("all").Comment("收件人：指定玩家 / 按条件 / 全服"),
		field.JSON("players", []GmTarget{}).Optional().Comment("指定玩家"),
		field.JSON("condition", GameMailCondition{}).Optional().Comment("按条件"),
		field.Bool("exclude_banned").Default(true).Comment("跳过封禁玩家"),
		field.Int("expire_days").Default(14).Comment("领取期限（天，0 不过期）"),
		field.Time("scheduled_at").Optional().Nillable().Comment("定时发送时间"),
		field.Enum("status").Values("draft", "scheduled", "sending", "sent", "failed", "revoking", "revoked").Default("draft").Comment("状态（sending / revoking 为后台投递、撤回中）"),
		field.Int("matched_count").Default(0).Comment("最近一次试运行匹配的人数（收件人改动后清零）"),
		field.Time("matched_at").Optional().Nillable().Comment("最近一次试运行时间"),
		field.Int("recipient_count").Default(0).Comment("投递人数（发送后）"),
		field.Int("failed_count").Default(0).Comment("投递失败人数"),
		field.Int("claimed_count").Default(0).Comment("已领取人数（缓存，来自 MGP gifts 报表）"),
		field.Time("stats_at").Optional().Nillable().Comment("领取统计更新时间"),
		field.String("error").Default("").Comment("发送失败原因"),
		field.String("execution_id").Default("").Comment("发送对应的 GM 记录 id"),
		field.String("created_by").Default("").Comment("创建者用户 id"),
		field.String("created_by_name").Default("").Comment("创建者名称快照"),
		field.String("updated_by_name").Default("").Comment("最后编辑者名称快照"),
		field.String("sent_by").Default("").Comment("发送者用户 id（定时发送到点时以他的名义执行）"),
		field.String("sent_by_name").Default("").Comment("发送者名称快照"),
		field.Time("sent_at").Optional().Nillable().Comment("发送时间"),
		field.String("revoked_by_name").Default("").Comment("撤回者名称快照"),
		field.Time("revoked_at").Optional().Nillable().Comment("撤回时间"),
		field.Int("revoked_count").Default(0).Comment("撤回了未领取礼物的玩家数"),
		field.String("revoke_execution_id").Default("").Comment("撤回对应的 GM 记录 id"),
	}
}

func (GameMail) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("server_id", "status"),
		index.Fields("server_id", "create_time"),
		index.Fields("status", "scheduled_at"),
	}
}

func (GameMail) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
