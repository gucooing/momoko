package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentSession Agent 会话：Momoko 自己保存的一份，列表以它为准；Agent 侧的会话用 acp_session_id 对应。
type AgentSession struct {
	ent.Schema
}

func (AgentSession) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("会话 id"),
		field.String("user_id").NotEmpty().Comment("创建者用户 id"),
		field.String("title").Default("").Comment("标题：Agent 给的，或用户重命名后的"),
		field.Bool("title_custom").Default(false).Comment("用户重命名过，之后不再跟随 Agent 给的标题"),
		field.String("runtime_id").NotEmpty().Comment("Agent（运行时）id"),
		field.String("workspace_id").Default("").Comment("工作区 id，空 = 无工作区"),
		field.String("cwd").Default("").Comment("实际运行目录：工作区目录、工作树目录或临时目录"),
		field.Strings("extra_dirs").Optional().Comment("附加文件夹"),
		field.String("worktree").Default("").Comment("工作树名，不在工作树里为空"),
		field.String("provider_id").Default("").Comment("供应商 id，空 = 服务器上已有的登录"),
		field.String("model").Default("").Comment("模型"),
		field.String("effort").Default("").Comment("推理强度"),
		field.String("mode").Default("").Comment("权限模式"),
		field.Bool("unrestricted").Default(false).Comment("Turbo：起进程时就让 Agent 不再询问任何操作，只在建会话时定"),
		field.Enum("status").
			Values("running", "waiting", "idle", "error", "archived").
			Default("idle").
			Comment("状态：运行中 / 等待批准 / 空闲 / 出错 / 已归档"),
		field.Text("error").Default("").Comment("出错原因"),
		field.String("acp_session_id").Default("").Comment("Agent 侧的会话 id"),
		field.String("parent_id").Default("").Comment("复刻来源（父会话 id）"),
		field.Bool("detached").Default(false).Comment("已移到外层：不再缩进，删父会话时也不带上它"),
		field.Time("active_at").Default(time.Now).Comment("最近活动时间"),
		field.Int("turns").Default(0).Comment("轮数"),
		field.Int64("usage_total").Default(0).Comment("累计 token"),
		field.Int64("usage_input").Default(0).Comment("累计输入（不含缓存）"),
		field.Int64("usage_output").Default(0).Comment("累计输出"),
		field.Int64("usage_cache_read").Default(0).Comment("累计缓存命中"),
		field.Int64("usage_cache_write").Default(0).Comment("累计缓存写入"),
		field.Int64("usage_thought").Default(0).Comment("累计思考"),
		field.Text("last_usage").Default("").Comment("最近一轮的用量（JSON）"),
		field.Int64("context_used").Default(0).Comment("上下文已用"),
		field.Int64("context_size").Default(0).Comment("上下文上限"),
		field.Strings("mcp_ids").Optional().Comment("本会话开启的 MCP 服务器"),
		field.Text("state").Default("").Comment("会话状态快照（JSON）：Agent 给的命令和计划"),
		field.String("origin").Default("").Comment("用户访问 Momoko 用的地址，通知里的链接用它（没配站点地址时）"),
	}
}

func (AgentSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "status"),
		index.Fields("workspace_id"),
		index.Fields("parent_id"),
		index.Fields("active_at"),
	}
}

func (AgentSession) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Annotation{Table: "agent_sessions"},
	}
}

func (AgentSession) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
