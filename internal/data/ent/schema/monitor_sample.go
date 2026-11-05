package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// MonitorSample 持久性能监控的采样点（分级降采样）。
// tier：0=原始采样，1=1 分钟，2=10 分钟，3=1 小时；sample_time 为采样槽 / 聚合桶的开始时间。
// data 为 JSON：指标键 → [平均值, 峰值, 样本数]，原始采样三者为 [值, 值, 1]。
type MonitorSample struct {
	ent.Schema
}

func (MonitorSample) Fields() []ent.Field {
	return []ent.Field{
		field.Int8("tier").Default(0).Comment("精度级别：0 原始 / 1 一分钟 / 2 十分钟 / 3 一小时"),
		field.Time("sample_time").Comment("采样槽或聚合桶开始时间"),
		field.Text("data").Comment("指标 JSON：键 → [平均, 峰值, 样本数]"),
	}
}

func (MonitorSample) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tier", "sample_time").Unique(),
	}
}
