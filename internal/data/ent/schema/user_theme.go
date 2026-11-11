package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserTheme 「我的主题」：用户自建或导入的外观主题，只对自己生效。
// content 为 v1.ThemeContent 的 protojson（已由 biz 规范化）。
type UserTheme struct {
	ent.Schema
}

func (UserTheme) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("主题 id（uuid）"),
		field.String("user_id").NotEmpty().Comment("所属用户 id"),
		field.String("name").NotEmpty().Comment("名称"),
		field.String("author").Default("").Comment("作者：自建为创建人昵称，导入为主题包里的作者"),
		field.String("version").Default("1.0").Comment("版本"),
		field.String("description").Default("").Comment("说明"),
		field.Enum("source").Values("custom", "import").Default("custom").Comment("来源：custom 自建 / import 导入"),
		field.Text("content").Comment("主题内容（v1.ThemeContent 的 protojson）"),
	}
}

func (UserTheme) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
	}
}

func (UserTheme) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
