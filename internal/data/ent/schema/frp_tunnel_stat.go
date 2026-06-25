package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FrpTunnelStat 内网穿透隧道的时间序列统计采样点（仅追加），用于详情页趋势图。
// 流量字段记录采样时刻的累计值（当日累计），前端按相邻采样点差值换算为速率。
type FrpTunnelStat struct {
	ent.Schema
}

func (FrpTunnelStat) Fields() []ent.Field {
	return []ent.Field{
		field.String("frp_tunnel_id").NotEmpty().Comment("所属隧道id"),
		field.Time("sample_time").Comment("采样时间"),
		field.Int64("active_connections").Default(0).Comment("活跃连接数"),
		field.Int64("bytes_in").Default(0).Comment("累计入站流量(字节)"),
		field.Int64("bytes_out").Default(0).Comment("累计出站流量(字节)"),
	}
}

func (FrpTunnelStat) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("frp_tunnel_id", "sample_time"),
		index.Fields("sample_time"),
	}
}
