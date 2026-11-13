package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GmSelfService 玩家自助开放：每台服务器、每条指令一行。只能开放 manifest 里标了 selfService 的指令。
// 设置：开放 / 每 period_days 天 use_limit 次（0 点刷新，period_days 为 0 时永久不刷新）/ 需要审批 / 玩家看到的说明。
type GmSelfService struct {
	ent.Schema
}

func (GmSelfService) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("id"),
		field.String("server_id").NotEmpty().Comment("服务器 id"),
		field.String("command").NotEmpty().Comment("指令 id"),
		field.Bool("enabled").Default(false).Comment("是否开放给玩家"),
		field.Int("use_limit").Default(0).Comment("每个玩家每个周期的次数（0 不限）"),
		field.Int("period_days").Default(1).Comment("次数的刷新周期天数，服务器时区 0 点刷新（0 永久不刷新）"),
		field.Bool("need_approval").Default(false).Comment("需要管理员审批"),
		field.String("player_note").Default("").Comment("玩家看到的说明"),
		field.String("updated_by").Default("").Comment("最后修改的管理员 id"),
		field.String("updated_by_name").Default("").Comment("最后修改的管理员名称快照"),
	}
}

func (GmSelfService) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("server_id", "command").Unique(),
	}
}

func (GmSelfService) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
