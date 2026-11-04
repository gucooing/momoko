package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FileShareDownload 记录公开分享的一次计数下载（文件直传或打包下载各计 1 次），
// 用于分享管理页的「本月下载」「302 直链占比」统计；分享删除时一并清除。
type FileShareDownload struct {
	ent.Schema
}

func (FileShareDownload) Fields() []ent.Field {
	return []ent.Field{
		field.String("share_id").Comment("分享id"),
		field.String("user_id").Comment("分享创建者用户id（冗余，便于按用户统计）"),
		field.Bool("redirect").Default(false).Comment("是否经 302 直链由存储端直接下发"),
		field.Time("create_time").Default(time.Now).Immutable().Comment("下载时间"),
	}
}

func (FileShareDownload) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "create_time"),
		index.Fields("share_id"),
	}
}
