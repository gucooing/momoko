package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ThemeAsset 主题引用的资源文件（背景图 / 横幅图转成的 WebP、上传的 woff2 字体）。
// 文件保存在 data/theme-assets/<id>，公开访问地址 /api/v1/theme-asset/<id>。
type ThemeAsset struct {
	ent.Schema
}

func (ThemeAsset) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Immutable().Comment("资源文件名：32 位小写十六进制 + 扩展名（webp / woff2）"),
		field.String("user_id").NotEmpty().Comment("上传人 id"),
		field.Enum("kind").Values("image", "font").Comment("类型：image 图片 / font 字体"),
		field.String("name").Default("").Comment("上传时的原文件名"),
		field.Int64("size").Default(0).Comment("保存的文件大小（字节）"),
		field.Int64("original_size").Default(0).Comment("原始大小（字节）"),
		field.Int32("width").Default(0).Comment("图片宽度（缩放后）"),
		field.Int32("height").Default(0).Comment("图片高度（缩放后）"),
	}
}

func (ThemeAsset) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
	}
}

func (ThemeAsset) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
