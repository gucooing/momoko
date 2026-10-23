package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// IQCheckResult 一次检测中某个模型的回复结果；一轮检测对站点的每个模型各产生一条。
type IQCheckResult struct {
	ent.Schema
}

func (IQCheckResult) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("结果 id"),
		field.String("site_id").NotEmpty().Comment("站点 id"),
		field.String("round_id").Default("").Comment("同一轮检测的编号"),
		field.String("model").NotEmpty().Comment("模型"),
		field.String("effort").Default("").Comment("推理强度"),
		field.String("prompt_id").Default("").Comment("提示词 id"),
		field.String("prompt_title").Default("").Comment("提示词标题"),
		field.Text("prompt_content").Default("").Comment("请求发送的提示词"),
		field.Text("response_text").Default("").Comment("回复原始文本"),
		field.Enum("status").
			Values("success", "failed").
			Default("success").
			Comment("状态"),
		field.Text("error").Default("").Comment("失败原因"),
		field.Int64("duration_ms").Default(0).Comment("耗时（毫秒）"),
		field.Int64("input_tokens").Default(0).Comment("输入 token 数"),
		field.Int64("output_tokens").Default(0).Comment("输出 token 数"),
		field.Time("request_at").Default(time.Now).Comment("检测时间"),
	}
}

func (IQCheckResult) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("site", IQCheckSite.Type).
			Ref("results").
			Field("site_id").
			Unique().
			Required().
			Comment("所属站点"),
	}
}

func (IQCheckResult) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("site_id", "request_at"),
		index.Fields("round_id"),
	}
}

func (IQCheckResult) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Annotation{Table: "iq_check_results"},
	}
}

func (IQCheckResult) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
