package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GameServerProcess 托管服务器的一个进程（复用应用实例）。
type GameServerProcess struct {
	// Role 进程角色：main 游戏服务端 / resource 资源服务器 / 适配器定义的其他角色。
	Role string `json:"role"`
	// InstanceID 关联的应用实例 id。
	InstanceID string `json:"instanceId"`
}

// GameServer 接入的游戏服务器：一款游戏（适配器）可以接入任意多台，彼此完全独立，名称只是名字。
// 托管（hosted）：Momoko 在本机或节点上运行进程（复用应用实例）并做 GM；
// 仅 GM（remote）：服务端在别处，只对接 MGP。两种方式可随时切换。
type GameServer struct {
	ent.Schema
}

func (GameServer) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Unique().Comment("服务器 id"),
		field.String("game").NotEmpty().Comment("游戏（适配器 id）：gakumas / bedrock / generic …"),
		field.String("name").NotEmpty().Comment("名称（只是名字）"),
		field.Strings("tags").Optional().Comment("标签（自由输入，只用来筛选）"),
		field.Enum("mode").Values("hosted", "remote").Comment("接入方式：hosted 托管 / remote 仅 GM"),
		field.String("node_id").Default("").Comment("托管所在节点 id（空为本机）"),
		field.String("install_dir").Default("").Comment("托管：服务端目录"),
		field.JSON("processes", []GameServerProcess{}).Optional().Comment("托管：进程（应用实例）"),
		field.Text("options").Default("").Comment("托管：适配器安装选项（JSON，如端口、更新来源）"),
		field.String("cover").Default("").Comment("封面素材 key（空为适配器默认）"),
		field.String("mgp_endpoint").Default("").Comment("MGP 地址（不含 /mgp/v1）；空表示未接入 MGP"),
		field.String("key_id").Default("").Comment("MGP Key ID"),
		field.Text("secret").Default("").Sensitive().Comment("MGP 密钥（secretbox 加密）"),
		field.Time("key_rotated_at").Optional().Nillable().Comment("最近一次轮换密钥的时间"),
		field.Text("manifest").Default("").Comment("manifest 缓存（JSON）"),
		field.Time("manifest_at").Optional().Nillable().Comment("manifest 获取时间"),
		field.Text("status").Default("").Comment("status 缓存（JSON）"),
		field.Time("status_at").Optional().Nillable().Comment("status 获取时间"),
		field.Bool("probe_ok").Default(false).Comment("最近一次 MGP 探测是否成功"),
		field.Int("probe_latency_ms").Default(0).Comment("最近一次 MGP 探测延迟（毫秒）"),
		field.String("probe_error").Default("").Comment("最近一次 MGP 探测错误"),
		field.Time("probe_at").Optional().Nillable().Comment("最近一次 MGP 探测时间"),
		field.Time("last_online_at").Optional().Nillable().Comment("最近一次 MGP 可达的时间"),
		field.String("timezone").Default("").Comment("服务器时区（IANA，空为 Momoko 本机时区）：「今天」与每日次数按它的自然日计算"),
		field.Enum("passport_visibility").Values("public", "invite", "hidden").Default("public").Comment("通行证绑定可见性：public 公开 / invite 凭邀请码 / hidden 不可绑定"),
		field.String("passport_invite_code").Default("").Comment("通行证绑定邀请码（visibility=invite 时）"),
		field.String("remark").Default("").Comment("备注"),
	}
}

func (GameServer) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("game"),
	}
}

func (GameServer) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}
