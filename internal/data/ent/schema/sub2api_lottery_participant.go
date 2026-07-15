package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Sub2APILotteryParticipant 一位用户在某一轮次的报名/中奖记录。
type Sub2APILotteryParticipant struct {
	ent.Schema
}

func (Sub2APILotteryParticipant) Fields() []ent.Field {
	return []ent.Field{
		field.String("round_id").NotEmpty().Comment("关联 Sub2APILotteryRound.id（轮次日期）"),
		field.Int64("sub2api_user_id").Comment("Sub2API 用户 ID（上游 int64）"),
		field.String("user_name").Optional().Comment("Sub2API 用户名（冗余展示）"),
		field.Float("spend_snapshot").Default(0).Comment("报名时来源日扣费快照（用于展示/审计）"),
		field.Time("registered_time").Comment("报名时间"),
		field.Bool("is_winner").Default(false).Comment("是否中奖"),
		field.Float("prize_amount").Default(0).Comment("中奖金额"),
		field.Enum("payout_status").
			Values("none", "pending", "paid", "failed").
			Default("none").
			Comment("发放状态：none / pending / paid / failed"),
		field.String("payout_idempotency_key").Optional().Comment("派奖幂等键（sub2api Idempotency-Key）"),
		field.Text("payout_error").Optional().Comment("派奖失败原因"),
	}
}

func (Sub2APILotteryParticipant) Indexes() []ent.Index {
	return []ent.Index{
		// 每人每轮仅一条报名记录
		index.Fields("round_id", "sub2api_user_id").Unique(),
		index.Fields("round_id", "is_winner"),
		index.Fields("sub2api_user_id"),
	}
}

func (Sub2APILotteryParticipant) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
