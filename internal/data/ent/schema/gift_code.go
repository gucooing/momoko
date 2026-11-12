package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GiftCodeBatch 兑换码批次：通用码（一个码多人用）或一次性码（生成 N 个码，每个只能用一次）。
// 兑换成功时以 gift.send（sourceKind=code、sourceId=批次 id）把奖励发进礼物箱。
type GiftCodeBatch struct {
	ent.Schema
}

func (GiftCodeBatch) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("批次 id"),
		field.String("server_id").NotEmpty().Comment("服务器 id"),
		field.String("name").NotEmpty().Comment("批次名称（只给管理员看）"),
		field.Enum("kind").Values("universal", "unique").Comment("类型：通用码 / 一次性码"),
		field.String("code").Default("").Comment("通用码（kind=universal）"),
		field.String("prefix").Default("").Comment("一次性码前缀"),
		field.Int("quantity").Default(0).Comment("一次性码生成数量"),
		field.JSON("rewards", []GameReward{}).Optional().Comment("奖励"),
		field.String("message").Default("").Comment("礼物箱文案"),
		field.Int("total_limit").Default(0).Comment("总次数上限（0 不限；一次性码等于数量）"),
		field.Int("per_player_limit").Default(1).Comment("每个玩家可用次数"),
		field.Time("start_at").Optional().Nillable().Comment("开始时间"),
		field.Time("end_at").Optional().Nillable().Comment("结束时间（空为长期）"),
		field.Enum("eligibility").Values("all", "condition").Default("all").Comment("适用玩家：所有玩家 / 按注册时间"),
		field.Int("registered_within_days").Default(0).Comment("按条件：注册 N 天内的玩家"),
		field.Strings("channels").Optional().Comment("兑换入口：passport 通行证自助服务 / game 游戏内"),
		field.Int("used_count").Default(0).Comment("已成功兑换次数"),
		field.Time("disabled_at").Optional().Nillable().Comment("停用时间"),
		field.String("disabled_by_name").Default("").Comment("停用者名称快照"),
		field.String("created_by").Default("").Comment("创建者用户 id"),
		field.String("created_by_name").Default("").Comment("创建者名称快照"),
	}
}

func (GiftCodeBatch) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("server_id", "create_time"),
		index.Fields("server_id", "code"),
	}
}

func (GiftCodeBatch) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// GiftCode 一次性码（每个码只能用一次）。通用码不在这张表里。
type GiftCode struct {
	ent.Schema
}

func (GiftCode) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("id"),
		field.String("batch_id").NotEmpty().Comment("批次 id"),
		field.String("server_id").NotEmpty().Comment("服务器 id"),
		field.String("code").NotEmpty().Comment("兑换码"),
		field.String("used_by_player").Default("").Comment("使用者 pid"),
		field.String("used_by_name").Default("").Comment("使用者昵称快照"),
		field.Time("used_at").Optional().Nillable().Comment("使用时间"),
	}
}

func (GiftCode) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("server_id", "code").Unique(),
		index.Fields("batch_id"),
	}
}

func (GiftCode) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// GiftCodeRedeem 兑换记录（成功与失败都记）。
type GiftCodeRedeem struct {
	ent.Schema
}

func (GiftCodeRedeem) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("id"),
		field.String("server_id").NotEmpty().Comment("服务器 id"),
		field.String("batch_id").Default("").Comment("批次 id（码不存在时为空）"),
		field.String("code").NotEmpty().Comment("输入的兑换码"),
		field.String("player_id").Default("").Comment("玩家 pid"),
		field.String("nickname").Default("").Comment("玩家昵称快照"),
		field.String("passport_user_id").Default("").Comment("通行证用户 id（passport_users）"),
		field.Enum("channel").Values("passport", "game", "admin").Default("passport").Comment("兑换入口"),
		field.String("platform").Default("").Comment("设备"),
		field.Enum("result").Values("success", "already_redeemed", "expired", "not_started", "exhausted", "disabled", "not_eligible", "banned", "rate_limited", "not_found", "code_used", "channel_closed", "failed").Comment("结果"),
		field.String("error_message").Default("").Comment("失败详情"),
		field.String("execution_id").Default("").Comment("发放对应的 GM 记录 id"),
	}
}

func (GiftCodeRedeem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("server_id", "create_time"),
		index.Fields("batch_id", "create_time"),
		index.Fields("server_id", "player_id", "batch_id"),
		index.Fields("passport_user_id", "create_time"),
	}
}

func (GiftCodeRedeem) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
