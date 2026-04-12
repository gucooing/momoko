package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type FileUpload struct {
	ent.Schema
}

func (FileUpload) Fields() []ent.Field {
	// 上传id
	return []ent.Field{
		field.String("id").Unique().Comment("id"),
		field.String("hash").Comment("快速hash"),
		field.String("path").Comment("文件路径"),
		field.String("file_name").Comment("文件名称"),
		field.Uint64("file_size").Comment("文件大小"),
		field.Uint64("chunk_size").Comment("分片大小"),
		field.Uint64("total_chunks").Comment("分片数量"),
		field.Bool("completed").Default(false).Comment("是否完成"),
		field.Bool("cancel").Default(false).Comment("是否取消"),
		field.String("user_id").Comment("上传发起人"),
	}
}

func (FileUpload) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Field("user_id").
			Unique().
			Required().
			Comment("上传发起用户"),

		edge.From("chunks", FileUploadChunk.Type).
			Ref("upload").
			Comment("上传分片列表"),
	}
}

func (FileUpload) Indexes() []ent.Index {
	return []ent.Index{
		// 保证同一路径下，同一 hash 文件只能有一条未完成且未取消的上传会话
		index.Fields("path", "hash").
			Unique().
			Annotations(
				entsql.IndexWhere("NOT completed AND NOT cancel"),
			),
	}
}

func (FileUpload) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
