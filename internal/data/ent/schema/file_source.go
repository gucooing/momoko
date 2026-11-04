package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FileSource 记录一个文件来源：本机目录挂载（local，限定在根目录内）或网络来源（对象存储 OSS / FTP / WebDAV）。
// 整机本地磁盘是内置默认来源、不入库；全局可见、管理员维护。
type FileSource struct {
	ent.Schema
}

func (FileSource) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Comment("id"),
		field.String("name").NotEmpty().Comment("展示名称"),
		field.Enum("type").
			Values("local", "oss", "ftp", "webdav").
			Comment("来源类型"),
		field.Bool("enabled").Default(true).Comment("是否启用"),
		field.Bool("redirect_302").Default(false).Comment("预览/下载是否优先 302 跳转到来源直链（需来源支持预签名）"),
		// config 存类型相关配置的 JSON 串；其中密钥字段（OSS secret / FTP/WebDAV 密码）已用 secretbox 加密。
		field.Text("config").Default("").Comment("类型相关配置(JSON，含已加密密钥)"),
		field.String("user_id").Comment("创建者用户id"),
		// 最近一次连通性测试结果（测试已保存来源时回写），未测试过 last_test_time 为空。
		field.Bool("last_test_ok").Default(false).Comment("最近一次测试是否连通"),
		field.Int64("last_test_latency_ms").Default(0).Comment("最近一次测试耗时(毫秒)"),
		field.String("last_test_message").Default("").Comment("最近一次测试结果消息"),
		field.Time("last_test_time").Optional().Nillable().Comment("最近一次测试时间"),
		field.Uint64("last_test_used_bytes").Default(0).Comment("最近一次测试得到的已用空间(字节)，0=未知"),
		field.Uint64("last_test_total_bytes").Default(0).Comment("最近一次测试得到的总容量(字节)，0=未知或不限"),
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
