package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// AgentWorkspace 工作区：服务器上的一个目录加一组新会话的默认配置。Git 信息和项目指令文件都实时读目录，不另存。
type AgentWorkspace struct {
	ent.Schema
}

func (AgentWorkspace) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("工作区 id"),
		field.String("user_id").NotEmpty().Comment("创建者用户 id"),
		field.String("name").NotEmpty().Comment("名称"),
		field.String("path").NotEmpty().Unique().Comment("目录"),
		field.String("git_url").Default("").Comment("从 Git 克隆时的仓库地址"),
		field.String("clone_status").Default("").Comment("克隆状态：空 / cloning / failed"),
		field.Text("clone_error").Default("").Comment("克隆失败原因"),
		field.String("runtime_id").Default("").Comment("默认模型所属的 Agent"),
		field.String("provider_id").Default("").Comment("默认模型所属的供应商"),
		field.String("model").Default("").Comment("默认模型"),
		field.String("effort").Default("").Comment("默认推理强度"),
		field.String("mode").Default("").Comment("默认权限模式"),
		field.Strings("mcp_ids").Optional().Comment("新会话默认开启的 MCP 服务器"),
	}
}

func (AgentWorkspace) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Annotation{Table: "agent_workspaces"},
	}
}

func (AgentWorkspace) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
