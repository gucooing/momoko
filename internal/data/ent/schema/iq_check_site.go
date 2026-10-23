package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// IQCheckModel 站点需要检测的一个模型及其推理强度。
type IQCheckModel struct {
	Model  string `json:"model"`
	Effort string `json:"effort"`
}

// IQCheckSite 智商检测站点：一个上游 AI 接口（端点 + API Key + 协议）及其定时检测配置。
type IQCheckSite struct {
	ent.Schema
}

func (IQCheckSite) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("站点 id"),
		field.String("name").NotEmpty().Comment("名称"),
		field.String("endpoint").NotEmpty().Comment("接口地址（端点）"),
		field.String("api_key").Default("").Sensitive().Comment("API Key"),
		field.String("user_agent").Default("pi").Comment("请求上游时使用的 User-Agent"),
		field.Enum("protocol").
			Values("chat", "responses", "messages").
			Default("chat").
			Comment("协议：chat=OpenAI Chat Completions，responses=OpenAI Responses，messages=Anthropic Messages"),
		field.JSON("models", []IQCheckModel{}).Comment("需要检测的模型及推理强度"),
		field.Bool("enabled").Default(true).Comment("是否启用定时检测"),
		field.Int("interval_minutes").Default(60).Comment("检测间隔（分钟）"),
		field.Int("max_runs").Default(0).Comment("检测次数上限，0=无限"),
		field.Int("run_count").Default(0).Comment("已执行的定时检测次数"),
		field.String("slug").NotEmpty().Unique().Comment("公开页随机路径"),
		field.Time("last_check_at").Optional().Nillable().Comment("最近检测时间"),
	}
}

func (IQCheckSite) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("results", IQCheckResult.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Comment("检测结果"),
	}
}

func (IQCheckSite) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("enabled"),
	}
}

func (IQCheckSite) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Annotation{Table: "iq_check_sites"},
	}
}

func (IQCheckSite) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
