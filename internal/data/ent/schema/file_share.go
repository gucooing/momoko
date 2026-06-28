package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FileShare 记录一条对外分享（文件或文件夹），通过随机 token 公开访问，
// 可设提取码、有效期、下载次数上限，并可随时启用/停用、二次编辑。
type FileShare struct {
	ent.Schema
}

func (FileShare) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Comment("id"),
		field.String("user_id").Comment("创建者用户id"),
		field.String("name").Comment("展示名称"),
		field.String("target_path").Comment("被分享的文件/文件夹路径（来源内逻辑路径）"),
		// 文件来源：空=本地磁盘，否则为 file_source.id；分享统一走 Store 接口，支持任意来源。
		field.String("source_id").Default("").Comment("文件来源id，空=本地"),
		field.Bool("is_dir").Default(false).Comment("是否文件夹"),
		field.Uint64("size").Default(0).Comment("文件大小快照(单文件分享，供公开元信息展示)"),
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
