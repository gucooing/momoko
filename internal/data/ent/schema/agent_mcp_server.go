package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// AgentMcpServer MCP 服务器配置。新建会话时把选中的放进 mcpServers 交给 Agent。
type AgentMcpServer struct {
	ent.Schema
}

func (AgentMcpServer) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("MCP 服务器 id"),
		field.String("name").NotEmpty().Unique().Comment("名称"),
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
