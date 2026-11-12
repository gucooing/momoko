package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PassportLoginLog 通行证登录记录（成功与失败；「最近活动」与安全提醒用）。
type PassportLoginLog struct {
	ent.Schema
}

func (PassportLoginLog) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("记录 id"),
		field.String("user_id").NotEmpty().Comment("通行证用户 id"),
		field.String("method").Default("").Comment("登录方式：password / email_code / register / oidc"),
		field.Bool("success").Default(true).Comment("是否成功"),
		field.String("error_code").Default("").Comment("失败原因代码"),
		field.String("ip").Default("").Comment("IP"),
		field.String("device_id").Default("").Comment("设备 id"),
		field.String("device").Default("").Comment("设备"),
		field.String("user_agent").Default("").Comment("User-Agent"),
	}
}

func (PassportLoginLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "create_time"),
		index.Fields("create_time"),
		index.Fields("user_id", "device_id"),
	}
}

func (PassportLoginLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
