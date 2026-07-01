package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"momoko/internal/data/ent/schema/sharetype"
)

// FileShare 记录一条对外分享，通过随机 token 公开访问，
// 可设提取码、有效期、下载次数上限，并可随时启用/停用、二次编辑。
type FileShare struct {
	ent.Schema
}

func (FileShare) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Comment("id"),
		field.String("user_id").Comment("创建者用户id"),
		field.String("name").Comment("展示名称"),
		field.JSON("items", []sharetype.Item{}).Comment("被分享的条目（可跨来源）：来源id + 来源内路径 + 缓存的名称/类型/大小/修改时间"),
		field.String("token").Unique().Comment("公开访问令牌"),
		field.String("code").Default("").Comment("提取码，空=无需"),
		field.Time("expires_at").Optional().Nillable().Comment("过期时间，空=永久"),
		field.Int64("max_downloads").Default(0).Comment("下载次数上限，0=不限"),
		field.Int64("download_count").Default(0).Comment("已下载次数"),
		field.Bool("enabled").Default(true).Comment("是否启用"),
	}
}

func (FileShare) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Field("user_id").
			Unique().
			Required().
			Comment("创建者"),
	}
}

func (FileShare) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
	}
}

func (FileShare) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
