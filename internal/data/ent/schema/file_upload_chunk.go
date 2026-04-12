package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type FileUploadChunk struct {
	ent.Schema
}

func (FileUploadChunk) Fields() []ent.Field {
	return []ent.Field{
		field.String("upload_id").
			Comment("所属上传任务id"),

		field.Uint64("chunk").
			Comment("分片编号，从1开始"),

		field.String("hash").
			Comment("分片hash"),

		field.Uint64("size").
			Comment("分片大小"),
	}
}

func (FileUploadChunk) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("upload", FileUpload.Type).
			Field("upload_id").
			Unique().
			Required().
			Comment("所属上传任务"),
	}
}

func (FileUploadChunk) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("upload_id", "chunk").
			Unique(),
	}
}

func (FileUploadChunk) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
