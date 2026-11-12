package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PassportMessage 通行证玩家的消息（新设备登录、密码与邮箱变更、管理员替你绑定 / 解绑、自助申请的审批结果…）。
// 只存种类与参数，前端本地化。
type PassportMessage struct {
	ent.Schema
}

func (PassportMessage) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("消息 id"),
		field.String("user_id").NotEmpty().Comment("通行证用户 id"),
		field.String("kind").NotEmpty().Comment("种类（前端据此本地化）"),
		field.String("server_id").Default("").Comment("相关的游戏服务器 id"),
		field.JSON("params", map[string]string{}).Optional().Comment("参数"),
		field.Time("read_at").Optional().Nillable().Comment("已读时间"),
	}
}

func (PassportMessage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "create_time"),
		index.Fields("user_id", "read_at"),
	}
}

func (PassportMessage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
