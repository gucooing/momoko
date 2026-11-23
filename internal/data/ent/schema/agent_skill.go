package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentSkill 技能，属于创建它的用户；管理员在全员 Agent 页加的是共享的（主人 @shared），共享给所有人或指定的人。
// 文件放在主人的技能目录里（以 name 为目录名），Claude Code 和 Codex 共用一份。
type AgentSkill struct {
	ent.Schema
}

func (AgentSkill) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("技能 id"),
		field.String("user_id").Default("").Comment("所属用户 id：每个人只看得到、只用得上自己的"),
		field.String("name").NotEmpty().Comment("名称，同时是技能目录名；同一个人的不重名"),
		field.String("description").Default("").Comment("说明"),
		field.Enum("source").Values("local", "git", "zip", "dir").Default("local").Comment("来源"),
		field.String("location").Default("").Comment("导入来源：Git 地址或服务器目录"),
		field.Bool("enabled").Default(true).Comment("是否启用"),
		field.Int("file_count").Default(0).Comment("文件数"),
		field.String("share").Default("").Comment("共享范围：空 = 只有主人自己用；all = 所有人；users = share_users 里的人。共享的主人是 @shared，在全员 Agent 页里管"),
		field.Strings("share_users").Optional().Comment("share 为 users 时能用的用户 id"),
	}
}

func (AgentSkill) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "name").Unique(),
	}
}

func (AgentSkill) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Annotation{Table: "agent_skills"},
	}
}

func (AgentSkill) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
