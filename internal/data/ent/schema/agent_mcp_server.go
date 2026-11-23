package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentMcpServer MCP 服务器配置，属于创建它的用户；管理员在全员 Agent 页加的是共享的（主人 @shared），
// 共享给所有人或指定的人。新建会话时把选中的放进 mcpServers 交给 Agent。
type AgentMcpServer struct {
	ent.Schema
}

func (AgentMcpServer) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("MCP 服务器 id"),
		field.String("user_id").Default("").Comment("所属用户 id：每个人只看得到、只用得上自己的"),
		field.String("name").NotEmpty().Comment("名称，同一个人的不重名"),
		field.String("description").Default("").Comment("说明"),
		field.Enum("transport").Values("stdio", "http").Default("stdio").Comment("传输方式"),
		field.String("command").Default("").Comment("stdio：命令"),
		field.Strings("args").Optional().Comment("stdio：参数"),
		field.Text("env").Default("").Sensitive().Comment("stdio：环境变量（JSON，值用 secretbox 加密）"),
		field.String("url").Default("").Comment("HTTP：地址"),
		field.Text("headers").Default("").Sensitive().Comment("HTTP：请求头（JSON，值用 secretbox 加密）"),
		field.Bool("default_on").Default(false).Comment("新会话默认开启"),
		field.String("status").Default("untested").Comment("最近一次测试连接：ok / fail / untested"),
		field.String("status_text").Default("").Comment("测试结果说明"),
		field.Int("tool_count").Default(0).Comment("测试时读到的工具数"),
		field.Time("tested_at").Optional().Nillable().Comment("最近一次测试时间"),
		field.String("share").Default("").Comment("共享范围：空 = 只有主人自己用；all = 所有人；users = share_users 里的人。共享的主人是 @shared，在全员 Agent 页里管"),
		field.Strings("share_users").Optional().Comment("share 为 users 时能用的用户 id"),
	}
}

func (AgentMcpServer) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "name").Unique(),
	}
}

func (AgentMcpServer) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Annotation{Table: "agent_mcp_servers"},
	}
}

func (AgentMcpServer) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
