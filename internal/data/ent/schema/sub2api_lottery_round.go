package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Sub2APILotteryRound 每日一期抽奖轮次。
// id = round_date（YYYY-MM-DD，Asia/Shanghai）= 报名/开奖当天（day D）；
// 奖池与资格来自 D-1 扣费。仅定时器在 [00:00,12:00) 且当日未结算时 Settle；
// 仅定时器在 ≥12:00 且轮次仍 registering 时 Draw。重启不做任何结算/开奖。
type Sub2APILotteryRound struct {
	ent.Schema
}

func (Sub2APILotteryRound) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Immutable().Comment("轮次日期 YYYY-MM-DD（报名/开奖当天）"),
		field.String("source_date").Comment("扣费来源日期 YYYY-MM-DD（前一天）"),
		field.Time("settle_time").Comment("结算时间（当天 00:00 +08）"),
		field.Time("draw_time").Comment("开奖时间（当天 12:00 +08）"),
		field.Enum("status").
			Values("registering", "drawn").
			Default("registering").
			Comment("状态：registering / drawn"),
		// 结算快照（便于历史审计，配置改动不影响已结算轮次）
		field.Float("pool_ratio").Comment("本轮奖池比例快照"),
		field.Float("threshold").Comment("本轮报名门槛快照"),
		field.Int("base_winners").Comment("本轮基准中奖人数快照"),
		field.Int("max_winners").Comment("本轮最大中奖人数快照（0=无限）"),
		field.Float("group_spend_total").Default(0).Comment("来源日展示分组扣费之和"),
		field.Float("carry_in").Default(0).Comment("上一期结转进来的奖池"),
		field.Float("pool_amount").Default(0).Comment("本轮奖池 = 比例×扣费和 + carry_in"),
		field.Int("eligible_count").Default(0).Comment("符合报名资格的用户数（来源日扣费≥门槛）"),
		field.Int("registered_count").Default(0).Comment("已报名人数"),
		field.Int("winner_count").Default(0).Comment("实际中奖人数"),
		field.Float("per_winner_amount").Default(0).Comment("每位中奖者金额"),
		field.Float("carry_out").Default(0).Comment("未发放、结转到次日的金额"),
		field.Bool("auto_payout").Default(true).Comment("本轮是否自动发放（快照）"),
		field.Bool("distributed").Default(false).Comment("奖金是否已发放（自动或手动）"),
	}
}

func (Sub2APILotteryRound) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("draw_time"),
		index.Fields("source_date"),
	}
}

func (Sub2APILotteryRound) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
