package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserMessage 站内消息（每个接收者一行）。
type UserMessage struct {
	ent.Schema
}

func (UserMessage) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("消息id"),
		field.String("user_id").NotEmpty().Comment("接收者用户id"),
		field.String("category").Default("system").Comment("分类：security / system / notice"),
		field.String("kind").NotEmpty().Comment("消息种类（前端据此本地化）"),
		field.String("title").Default("").Comment("标题"),
		field.Text("content").Default("").Comment("内容"),
		field.JSON("params", map[string]string{}).Optional().Comment("模板参数"),
		field.String("sender_id").Default("").Comment("发布者用户id（notice）"),
		field.Time("read_at").Optional().Nillable().Comment("已读时间"),
	}
}

func (UserMessage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "create_time"),
		index.Fields("user_id", "read_at"),
	}
}

func (UserMessage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
