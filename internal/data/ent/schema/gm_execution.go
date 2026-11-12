package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GmTarget 指令目标快照。
type GmTarget struct {
	PID       string `json:"pid"`
	Name      string `json:"name,omitempty"`
	DisplayID string `json:"displayId,omitempty"`
}

// GmExecution GM 执行与审计（含玩家自助）。params 只存不展示；result 为 MGP 返回的逐对象结果。
type GmExecution struct {
	ent.Schema
}

func (GmExecution) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("记录 id"),
		field.String("server_id").NotEmpty().Comment("服务器 id"),
		field.String("command").NotEmpty().Comment("指令 id"),
		field.String("command_title").Default("").Comment("指令标题快照"),
		field.String("command_group").Default("").Comment("指令分组快照"),
		field.String("risk").Default("").Comment("风险等级快照：low / medium / high / danger"),
		field.String("target_kind").Default("").Comment("目标类型快照：player / players / server"),
		field.String("actor_id").Default("").Comment("执行者 id：actor_kind=admin 为 Momoko 管理员，player 为通行证用户（passport_users）"),
		field.String("actor_name").Default("").Comment("执行者名称快照"),
		field.Enum("actor_kind").Values("admin", "player").Comment("执行者类型：管理员 / 玩家自助"),
		field.Enum("channel").Values("admin", "passport", "game").Default("admin").Comment("来源：Momoko 后台 / 通行证 / 游戏内"),
		field.String("client_ip").Default("").Comment("发起请求的 IP"),
		field.String("platform").Default("").Comment("发起请求的设备（User-Agent 解析）"),
		field.JSON("targets", []GmTarget{}).Optional().Comment("目标快照"),
		field.Strings("target_pids").Optional().Comment("目标 pid（筛选用）"),
		field.Text("params").Default("").Sensitive().Comment("请求参数（只存不展示）"),
		field.JSON("summary", map[string]string{}).Optional().Comment("可展示的摘要参数（礼物文案、邮件标题、配置 revision 等，不含参数原文；前端本地化）"),
		field.String("reason").Default("").Comment("玩家自助申请理由"),
		field.Bool("confirmed").Default(false).Comment("执行前输入了玩家昵称确认（危险指令）"),
		field.Bool("dry_run").Default(false).Comment("是否试运行"),
		field.Enum("status").Values("success", "partial", "failed", "pending", "running", "rejected").Comment("状态"),
		field.Text("result").Default("").Comment("MGP 返回的结果（results / summary 的 JSON）"),
		field.Int("success_count").Default(0).Comment("成功的目标数"),
		field.Int("failed_count").Default(0).Comment("失败的目标数"),
		field.String("backup_ref").Default("").Comment("服务端自动备份的 id（危险指令）"),
		field.String("error_code").Default("").Comment("错误码"),
		field.String("error_message").Default("").Comment("错误信息"),
		field.String("request_id").NotEmpty().Comment("MGP requestId（幂等）"),
		field.String("source").Default("").Comment("来源对象：mail:<id> / code:<id> / config:<domain> / self-service"),
		field.String("reviewed_by").Default("").Comment("审批人用户 id"),
		field.String("reviewer_name").Default("").Comment("审批人名称快照"),
		field.Time("reviewed_at").Optional().Nillable().Comment("审批时间"),
		field.String("review_note").Default("").Comment("审批意见 / 驳回原因"),
		field.Time("finished_at").Optional().Nillable().Comment("执行完成时间"),
		field.Int("duration_ms").Default(0).Comment("MGP 调用耗时（毫秒）"),
	}
}

func (GmExecution) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("server_id", "create_time"),
		index.Fields("server_id", "status"),
		index.Fields("actor_id", "create_time"),
		index.Fields("request_id").Unique(),
	}
}

func (GmExecution) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
