package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PassportUser 用户通行证账号（玩家）。与 Momoko 管理后台的用户（users 表）完全独立：
// 独立的注册、登录、会话与令牌，只在用户通行证 server 上使用。
type PassportUser struct {
	ent.Schema
}

func (PassportUser) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("通行证用户 id"),
		field.String("username").NotEmpty().Unique().Comment("用户名（登录用，不能修改）"),
		field.String("email").Default("").Comment("邮箱（找回密码、验证码登录）"),
		field.Bool("email_verified").Default(false).Comment("邮箱已验证"),
		field.String("password").Default("").Sensitive().Comment("密码（bcrypt）"),
		field.Time("password_updated_at").Optional().Nillable().Comment("最近一次修改密码的时间"),
		field.String("name").Default("").Comment("昵称"),
		field.String("bio").Default("").Comment("个人简介"),
		field.String("avatar").Default("").Comment("头像（上传的图片地址）"),
		field.String("avatar_key").Default("").Comment("上传头像在对象存储里的 key（换头像时删除旧对象）"),
		field.String("avatar_asset").Default("").Comment("用游戏素材做头像：<game>/<asset key>"),
		field.String("language").Default("").Comment("语言偏好（空为跟随浏览器）"),
		field.String("timezone").Default("").Comment("时区（IANA，空为跟随浏览器）"),
		field.Bool("email_code_login").Default(true).Comment("允许用邮箱验证码登录"),
		field.Text("totp_secret").Default("").Sensitive().Comment("两步验证 TOTP 密钥（加密；空为未开启）"),
		field.Time("totp_enabled_at").Optional().Nillable().Comment("开启两步验证的时间"),
		field.Strings("recovery_codes").Optional().Sensitive().Comment("两步验证恢复码（bcrypt 哈希，用过即删）"),
		field.Enum("status").Values("active", "disabled").Default("active").Comment("状态"),
		field.Time("last_login_at").Optional().Nillable().Comment("最近一次登录时间"),
		field.String("last_login_ip").Default("").Comment("最近一次登录 IP"),
		field.String("last_login_device").Default("").Comment("最近一次登录设备"),
		field.Time("deletion_scheduled_at").Optional().Nillable().Comment("申请注销后计划删除的时间（7 天内可撤销）"),
		field.String("invite_id").Default("").Comment("注册时使用的邀请码 id（passport_invites）"),
		field.String("register_ip").Default("").Comment("注册时的 IP"),
		field.Time("disabled_at").Optional().Nillable().Comment("被管理员停用的时间"),
		field.String("disabled_by_name").Default("").Comment("停用的管理员名称快照"),
		field.Time("reviews_synced_at").Optional().Nillable().Comment("已为之生成消息的自助申请审批时间上限（消息同步游标）"),
	}
}

func (PassportUser) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("email"),
		index.Fields("create_time"),
		index.Fields("status"),
		index.Fields("deletion_scheduled_at"),
	}
}

func (PassportUser) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
