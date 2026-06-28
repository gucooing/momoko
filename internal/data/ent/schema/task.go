package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Task 持久化通用任务管理器（pkg/task）中的任务，使其在 momoko 重启 / 前端刷新后不丢失，
// 并支持开机注入未完成任务、定时任务与常驻任务。任务的具体执行逻辑仍在各自 pkg 内。
type Task struct {
	ent.Schema
}

func (Task) Fields() []ent.Field {
	return []ent.Field{
		// 单例任务（GC/守护）id 取其 type 便于幂等 upsert；一次性任务为 uuid。
		field.String("id").Unique().Comment("任务id"),
		field.String("type").Comment("注册类型键，用于开机按工厂重建任务"),
		field.Enum("kind").
			Values("oneshot", "scheduled", "daemon").
			Comment("任务种类：超时型一次性/定时/常驻"),
		field.Enum("status").
			Values("pending", "running", "success", "failed", "canceled").
			Default("pending").
			Comment("任务状态"),
		field.Enum("resume_policy").
			Values("none", "rerun", "always").
			Default("none").
			Comment("开机注入策略：none=不重跑(标失败)/rerun=未完成重跑/always=默认开启单例"),
		field.String("title").Default("").Comment("展示标题"),
		field.String("user_id").Default("").Comment("发起用户id，空=系统任务"),
		// payload 存重建任务所需参数的 JSON；state 存断点续传 JSON；result 存 []TaskResult JSON。
		field.Text("payload").Default("").Comment("重建参数(JSON)"),
		field.Text("state").Default("").Comment("断点续传状态(JSON)"),
		field.Text("result").Default("").Comment("执行结果(JSON)"),
		field.Int64("progress_total").Default(0).Comment("总进度"),
		field.Int64("progress_finished").Default(0).Comment("已完成进度"),
		field.Text("message").Default("").Comment("当前消息"),
		field.Text("error").Default("").Comment("错误信息"),
		field.Int64("interval_ms").Default(0).Comment("定时任务触发周期(毫秒)"),
		field.Int64("timeout_ms").Default(0).Comment("一次性任务超时(毫秒，0=管理器默认)"),
		field.Time("end_time").Optional().Nillable().Comment("结束时间"),
	}
}

func (Task) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("type"),
		index.Fields("user_id"),
		// 开机注入扫描：always 全部 + rerun 的未完成项。
		index.Fields("resume_policy", "status"),
	}
}

func (Task) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
