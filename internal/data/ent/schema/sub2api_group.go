package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Sub2APIGroup 本地同步到的 Sub2API 分组，与使用记录关联。
// public_enabled 控制该分组是否参与公开页统计（公开页全局统一过滤）。
// deleted 表示上游已删除：设置页合并为「已删除分组」一项。
type Sub2APIGroup struct {
	ent.Schema
}

func (Sub2APIGroup) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("分组 ID（上游 group_id）"),
		field.String("name").NotEmpty().Comment("分组名称（与 Sub2API 侧一致）"),
		field.Bool("public_enabled").Default(false).Comment("是否参与公开页统计"),
		field.Bool("deleted").Default(false).Comment("上游是否已删除"),
	}
}

func (Sub2APIGroup) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("usage_records", Sub2APIUsageRecord.Type),
	}
}

func (Sub2APIGroup) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name"),
		index.Fields("public_enabled"),
		index.Fields("deleted"),
		// 活跃分组名称仍保持唯一；已删除可与活跃重名（上游软删后复用名）
		index.Fields("name", "deleted"),
	}
}

func (Sub2APIGroup) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
