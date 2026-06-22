package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ImageGenGeneration 一次生图任务（文生图/图生图），异步执行。
type ImageGenGeneration struct {
	ent.Schema
}

func (ImageGenGeneration) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("任务ID"),
		field.String("user_id").NotEmpty().Comment("sub2api 用户ID（来自 /keys 校验）"),
		field.String("src_host").NotEmpty().Comment("sub2api 站点 origin，与 user_id 组成复合归属键"),
		field.String("mode").Comment("生成模式：text2image | image2image"),
		field.String("api_key_id").NotEmpty().Comment("使用的 apikey id（不存原文）"),
		field.String("model").Comment("模型"),
		field.Text("prompt").Comment("提示词"),
		field.String("size").Comment("分辨率"),
		field.Int("n").Default(1).Comment("张数"),
		field.String("quality").Optional().Comment("质量"),
		field.String("output_format").Optional().Comment("输出格式"),
		field.String("status").Default("pending").Comment("状态：pending | completed | failed"),
		field.Text("error_message").Optional().Comment("失败原因"),
		field.Int64("latency_ms").Default(0).Comment("耗时（毫秒）"),
		field.Int("result_count").Default(0).Comment("结果图片数"),
		field.Time("started_at").Optional().Nillable().Comment("开始时间"),
		field.Time("finished_at").Optional().Nillable().Comment("完成时间"),
	}
}

func (ImageGenGeneration) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "src_host"),
		index.Fields("status"),
		index.Fields("started_at"),
	}
}

func (ImageGenGeneration) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
