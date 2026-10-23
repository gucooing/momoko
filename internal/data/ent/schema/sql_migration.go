package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	entschema "entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// SQLMigration 记录 internal/data/migration/sql 目录下已执行的增量 SQL 脚本，
// 启动时据此从上次执行的位置继续执行序号更大的新脚本。
type SQLMigration struct {
	ent.Schema
}

func (SQLMigration) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("version").Unique().Comment("脚本序号（文件名前缀）"),
		field.String("name").Comment("脚本文件名"),
		field.Time("applied_at").Default(time.Now).Immutable().Comment("执行时间"),
	}
}

func (SQLMigration) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Annotation{Table: "sql_migrations"},
	}
}
