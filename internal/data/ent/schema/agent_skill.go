package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// AgentSkill 技能。文件放在服务器上的技能目录里（以 name 为目录名），Claude Code 和 Codex 共用一份。
type AgentSkill struct {
	ent.Schema
}

func (AgentSkill) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("技能 id"),
		field.String("name").NotEmpty().Unique().Comment("名称，同时是技能目录名"),
		field.String("description").Default("").Comment("说明"),
		field.Enum("source").Values("local", "git", "zip", "dir").Default("local").Comment("来源"),
		field.String("location").Default("").Comment("导入来源：Git 地址或服务器目录"),
		field.Bool("enabled").Default(true).Comment("是否启用"),
		field.Int("file_count").Default(0).Comment("文件数"),
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
