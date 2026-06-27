package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FileSource 记录一个网络文件来源（对象存储 OSS / FTP / WebDAV 等）。
// 本地磁盘是默认来源、不入库；这里只保存需要凭证连接的远端来源，全局可见、管理员维护。
type FileSource struct {
	ent.Schema
}

func (FileSource) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Comment("id"),
		field.String("name").NotEmpty().Comment("展示名称"),
		field.Enum("type").
			Values("oss", "ftp", "webdav").
			Comment("来源类型"),
		field.Bool("enabled").Default(true).Comment("是否启用"),
		field.Bool("redirect_302").Default(false).Comment("预览/下载是否优先 302 跳转到来源直链（需来源支持预签名）"),
		// config 存类型相关配置的 JSON 串；其中密钥字段（OSS secret / FTP/WebDAV 密码）已用 secretbox 加密。
		field.Text("config").Default("").Comment("类型相关配置(JSON，含已加密密钥)"),
		field.String("user_id").Comment("创建者用户id"),
	}
}

func (FileSource) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Field("user_id").
			Unique().
			Required().
			Comment("创建者"),
	}
}

func (FileSource) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("type"),
		index.Fields("enabled"),
	}
}

func (FileSource) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
