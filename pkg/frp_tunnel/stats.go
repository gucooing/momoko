package frp_tunnel

import (
	"time"

	"github.com/fatedier/frp/pkg/metrics/mem"
)

// ProxyStat 是某条隧道（frp proxy）的实时统计快照。
// 流量为当日累计值（frp 内存统计按天计），上层按相邻采样点差值换算速率。
type ProxyStat struct {
	Online      bool  // frpc 是否已连接且 proxy 在线
	ActiveConns int64 // 当前活跃（最终用户）连接数
	BytesIn     int64 // 当日累计入站流量(字节)
	BytesOut    int64 // 当日累计出站流量(字节)

	LastStartTime time.Time // 最近一次上线时间（零值表示未知）
	LastCloseTime time.Time // 最近一次下线时间（零值表示未知）
}

// Stat 返回指定 proxy 名称的实时统计；该 proxy 从未连接过时 ok=false。
//
// 在线判定基于 frp 内存统计的最近上线/下线时刻：proxy 未关闭，或上线时刻不早于下线时刻
// （重连）即视为在线。frps 在 frpc 断开（含异常断开）时会更新该统计，故无需依赖插件回调。
func Stat(name string) (ProxyStat, bool) {
	ps := mem.StatsCollector.GetProxyByName(name)
	if ps == nil {
		return ProxyStat{}, false
	}
	online := ps.LastCloseTime == "" || ps.LastStartTime >= ps.LastCloseTime
	return ProxyStat{
		Online:      online,
		ActiveConns: ps.CurConns,
		BytesIn:     ps.TodayTrafficIn,
		BytesOut:    ps.TodayTrafficOut,

		LastStartTime: parseStatTime(ps.LastStartTime),
		LastCloseTime: parseStatTime(ps.LastCloseTime),
	}, true
}

// parseStatTime 解析 frp 内存统计的时间（格式 "01-02 15:04:05"，不含年份，按本地时区）。
// 补上当前年份；若得到的时间晚于现在则视为去年。
func parseStatTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	now := time.Now()
	t, err := time.ParseInLocation("01-02 15:04:05", s, now.Location())
	if err != nil {
		return time.Time{}
	}
	t = t.AddDate(now.Year(), 0, 0)
	if t.After(now.Add(time.Minute)) {
		t = t.AddDate(-1, 0, 0)
	}
	return t
}
