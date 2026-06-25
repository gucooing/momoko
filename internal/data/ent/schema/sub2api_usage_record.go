package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Sub2APIUsageRecord 本地缓存的 Sub2API 使用记录，用于聚合首页用量看板。
type Sub2APIUsageRecord struct {
	ent.Schema
}

func (Sub2APIUsageRecord) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("记录唯一标识（来源记录ID或内容哈希）"),
		field.Time("request_time").Comment("请求时间"),
		field.String("request_date").Comment("请求日期（本地时区 YYYY-MM-DD，用于按日聚合）"),
		field.String("model").Optional().Comment("模型名称"),
		field.String("endpoint").Optional().Comment("接口/通道"),
		field.String("user_agent").Optional().Comment("客户端 User-Agent（用于 UA 维度统计）"),
		field.String("status").Optional().Comment("状态"),
		field.Bool("success").Default(false).Comment("是否成功"),
		field.Int64("latency_ms").Default(0).Comment("延迟（毫秒）"),
		field.Int64("token_count").Default(0).Comment("Token 总数（含输入/输出/缓存，用于用量统计）"),
		field.Int64("output_tokens").Default(0).Comment("输出 token 数（不含输入与缓存）"),
		field.Float("tps").Default(0).Comment("单请求 token 生成速度（输出token/秒，按请求计，不含缓存）"),
		// 详情字段：供最近请求详情展示
		field.Float("cost").Default(0).Comment("费用（USD，total_cost）"),
		field.Int64("first_token_ms").Default(0).Comment("首 token 延迟（毫秒）"),
		field.String("reasoning_effort").Optional().Comment("推理强度"),
		field.String("account_name").Optional().Comment("账号名称"),
		field.Text("error_message").Optional().Comment("错误详情（失败/上游错误请求）"),
		field.Int("http_status").Default(0).Comment("HTTP 状态码"),
	}
}

func (Sub2APIUsageRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("request_time"),
		index.Fields("request_date"),
		index.Fields("success"),
		index.Fields("status"),
		index.Fields("model"),
		index.Fields("endpoint"),
	}
}

func (Sub2APIUsageRecord) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
