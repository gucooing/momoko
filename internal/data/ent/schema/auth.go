package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Auth struct {
	ent.Schema
}

func (Auth) Fields() []ent.Field {
	return []ent.Field{
		field.String("session_id").NotEmpty().Comment("会话id"),
		field.String("device_id").NotEmpty().Comment("设备id"),
		field.String("device").NotEmpty().Comment("登录设备"),
		field.String("user_id").NotEmpty().Comment("所属用户id"),
		field.String("ip").Comment("登录ip"),
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
		index.Fields("session_id", "type").Unique(),
		index.Fields("device_id", "type").Unique(),
	}
}

func (Auth) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
