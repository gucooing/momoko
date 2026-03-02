package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

type Auth struct {
	ent.Schema
}

func (Auth) Fields() []ent.Field {
	return []ent.Field{
		field.String("session_id").Unique().NotEmpty().Comment("会话id"),
		field.String("device_id").Unique().NotEmpty().Comment("设备id"),
		field.String("user_id").NotEmpty().Comment("所属用户id"),
		field.Time("create_time").Default(time.Now).Immutable().Comment("创建时间"),
		field.Time("update_time").Default(time.Now).UpdateDefault(time.Now).Comment("更新时间"),
		field.Enum("type").Values("token", "refresh_token").Comment("token类型"),
	}
}

func (Auth) Indexes() []ent.Index {
	return []ent.Index{
		// 唯一会话绑定
		index.Fields("session_id", "device_id", "type").Unique(),
	}
}
