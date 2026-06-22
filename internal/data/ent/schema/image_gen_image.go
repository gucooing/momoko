package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ImageGenImage 生图结果图片，字节存盘，DB 仅存路径。
type ImageGenImage struct {
	ent.Schema
}

func (ImageGenImage) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("图片ID"),
		field.String("generation_id").NotEmpty().Comment("所属任务ID"),
		field.String("user_id").NotEmpty().Comment("sub2api 用户ID（冗余）"),
		field.String("path").Comment("磁盘相对路径（相对 servers/imagine 根）"),
		field.String("filename").Comment("文件名"),
		field.Int64("size_bytes").Default(0).Comment("文件大小（字节）"),
		field.Int("width").Default(0).Comment("宽度"),
		field.Int("height").Default(0).Comment("高度"),
		field.String("mime_type").Default("image/png").Comment("MIME 类型"),
		field.Text("revised_prompt").Optional().Comment("修订后的提示词"),
		field.Int("index").Default(0).Comment("本次生成中的序号"),
	}
}

func (ImageGenImage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("generation_id"),
		index.Fields("user_id"),
	}
}

func (ImageGenImage) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
