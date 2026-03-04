package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Auth struct {
	ent.Schema
}

func (Auth) Fields() []ent.Field {
	return []ent.Field{
		field.String("session_id").Unique().NotEmpty().Comment("会话id"),
		field.String("device_id").NotEmpty().Comment("设备id"),
		field.String("user_id").NotEmpty().Comment("所属用户id"),
		field.String("ip").Comment("登录ip"),
		field.Time("create_time").Default(time.Now).Immutable().Comment("创建时间"),
		field.Time("update_time").Default(time.Now).UpdateDefault(time.Now).Comment("更新时间"),
		field.Enum("type").Values("token", "refresh_token").Comment("token类型"),
	}
}

func (Auth) Indexes() []ent.Index {
	return []ent.Index{
		// 索引加速
		index.Fields("session_id"),
		index.Fields("device_id"),
		index.Fields("user_id"),
		// 唯一会话绑定
		index.Fields("session_id", "device_id", "type").Unique(),
	}
}
