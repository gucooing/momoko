package sub2api

import (
	"context"
	"strconv"
	"time"

	v1 "momoko/api/gen/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// minutesPerDay 统计窗口跨度与自然日的换算，也用于判定日内/按天趋势的分桶方式。
const minutesPerDay int32 = 24 * 60

const (
	topItemsLimit      = 8
	recentRecordsLimit = 8
)

// BuildSnapshot 构造完整的用量快照（含派生指标）。
// 全部聚合下沉数据层（ent Count/Sum/Mean/GroupBy），不在内存中遍历记录；
// publicOnly=true 时数据层仅统计公开启用分组的记录（公开页全局统一）。
func BuildSnapshot(ctx context.Context, store UsageStore, cfg *v1.Sub2APIConfig, state SyncState, publicOnly bool) (*v1.Sub2APIUsageSnapshot, error) {
	now := time.Now()
	todayStart := dayStart(now)
	windowStart := todayStart.AddDate(0, 0, -int(cfg.HistoryDays)+1)

	allWindow := StatsWindow{PublicOnly: publicOnly}
	todayWindow := StatsWindow{Start: &todayStart, PublicOnly: publicOnly}
	trendWindow := StatsWindow{Start: &windowStart, PublicOnly: publicOnly}

	totals, err := store.AggregateTotals(ctx, allWindow)
	if err != nil {
		return nil, err
	}
	todayTotals, err := store.AggregateTotals(ctx, todayWindow)
	if err != nil {
		return nil, err
	}
	daily, err := store.DailyTrend(ctx, trendWindow)
	if err != nil {
		return nil, err
	}
	modelTop, err := store.TopItems(ctx, trendWindow, GroupByModel, topItemsLimit)
	if err != nil {
		return nil, err
	}
	recent, _, err := store.RecordsPage(ctx, nil, nil, 0, recentRecordsLimit, publicOnly, RecordFilter{})
	if err != nil {
		return nil, err
	}
	todaySeries, err := store.IntradaySeries(ctx, todayWindow)
	if err != nil {
		return nil, err
	}

	status := state.Status
	if status == "" {
		status = SyncStatusIdle
	}
	return &v1.Sub2APIUsageSnapshot{
		Configured:            ClientConfigFromConfig(cfg).Configured(),
		Status:                status,
		Error:                 state.Error,
		LastSyncTime:          timestampOrNil(state.LastSyncTime),
		NextSyncTime:          timestampOrNil(state.NextSyncTime),
		LatestRecordTime:      timestampOrNil(state.LatestRecordTime),
		SyncedRecords:         state.SyncedRecords,
		TotalRecords:          totals.TotalCount,
		RequestCount:          totals.RequestCount,
		SuccessCount:          totals.SuccessCount,
		SuccessRate:           percent(totals.SuccessCount, totals.RequestCount),
		AverageLatencyMs:      totals.AverageLatencyMS,
		TokenCount:            totals.TokenCount,
		TodayRequestCount:     todayTotals.RequestCount,
		TodaySuccessCount:     todayTotals.SuccessCount,
		TodaySuccessRate:      percent(todayTotals.SuccessCount, todayTotals.RequestCount),
		TodayAverageLatencyMs: todayTotals.AverageLatencyMS,
		TodayTokenCount:       todayTotals.TokenCount,
		DataRange:             "最近 " + strconv.FormatInt(int64(cfg.HistoryDays), 10) + " 天",
		GeneratedAt:           timestamppb.New(now),
		Trend:                 dailyTrendPoints(daily, windowStart, cfg.HistoryDays),
		ModelUsage:            mapTopItems(modelTop),
		RecentRequests:        toRecentRequests(recent),
		RecentTps:             todayTotals.AverageTPS,
		RecentQps:             0,
		RecentWindowSeconds:   0,
		TodaySeries:           todaySeriesPoints(todaySeries),
	}, nil
}

// BuildPublicSnapshot 公开页快照：数据层按 public_enabled 分组过滤后聚合，并脱敏。
func BuildPublicSnapshot(ctx context.Context, store UsageStore, cfg *v1.Sub2APIConfig, state SyncState) (*v1.Sub2APIUsageSnapshot, error) {
	snapshot, err := BuildSnapshot(ctx, store, cfg, state, true)
	if err != nil {
		return nil, err
	}
	StripForPublic(snapshot)
	return snapshot, nil
}

// StripForPublic 移除公开首页不需要的明细，避免泄露逐条请求信息。
func StripForPublic(snapshot *v1.Sub2APIUsageSnapshot) {
	if snapshot == nil {
		return
	}
	snapshot.RecentRequests = nil
	for _, item := range snapshot.Trend {
		item.AverageLatencyMs = 0
	}
}

// BuildStats 聚合指定区间（今日/最近 N 天）的公开用量统计：标量 + 趋势 + 分组/UA。
// 模型排行拆到 BuildPublicModels，不在此接口返回。
// 全部聚合在数据层 DB 侧完成；单日按 15 分钟桶、多日按自然日展示趋势。
func BuildStats(ctx context.Context, store UsageStore, cfg *v1.Sub2APIConfig, rangeDays int32) (*v1.Sub2APIStats, error) {
	days := normalizeRangeDays(rangeDays, cfg.HistoryDays)
	now := time.Now()
	todayStart := dayStart(now)
	start := todayStart.AddDate(0, 0, -(int(days) - 1))
	window := StatsWindow{Start: &start, PublicOnly: true}

	totals, err := store.AggregateTotals(ctx, window)
	if err != nil {
		return nil, err
	}

	var trend []*v1.Sub2APITrendPoint
	if days <= 1 {
		series, err := store.IntradaySeries(ctx, window)
		if err != nil {
			return nil, err
		}
		trend = intradayTrendPoints(series)
	} else {
		daily, err := store.DailyTrend(ctx, window)
		if err != nil {
			return nil, err
		}
		trend = dailyTrendPoints(daily, start, days)
	}

	userAgents, err := store.TopItems(ctx, window, GroupByUserAgent, 0)
	if err != nil {
		return nil, err
	}
	groups, err := store.TopItems(ctx, window, GroupByGroup, 0)
	if err != nil {
		return nil, err
	}

	return &v1.Sub2APIStats{
		RangeDays:        days,
		RangeLabel:       rangeLabel(days),
		RequestCount:     totals.RequestCount,
		SuccessCount:     totals.SuccessCount,
		SuccessRate:      percent(totals.SuccessCount, totals.RequestCount),
		TokenCount:       totals.TokenCount,
		AverageLatencyMs: totals.AverageLatencyMS,
		AverageTps:       totals.AverageTPS,
		Trend:            trend,
		UserAgents:       mapTopItems(userAgents),
		Groups:           mapTopItems(groups),
	}, nil
}

// BuildPublicModels 公开热门模型：数据层 TopItems(GroupByModel) 直出。
// limit<=0 返回全部；首页传 8，详情页传 0。
func BuildPublicModels(ctx context.Context, store UsageStore, cfg *v1.Sub2APIConfig, rangeDays, limit int32) ([]*v1.Sub2APITopItem, error) {
	days := normalizeRangeDays(rangeDays, cfg.HistoryDays)
	todayStart := dayStart(time.Now())
	start := todayStart.AddDate(0, 0, -(int(days) - 1))
	window := StatsWindow{Start: &start, PublicOnly: true}
	models, err := store.TopItems(ctx, window, GroupByModel, int(limit))
	if err != nil {
		return nil, err
	}
	return mapTopItems(models), nil
}

// ---------- 管理端时间段归一化与标签 ----------

// normalizeRange 归一化管理端时间段：
// end 缺省/超前取“现在”；start 为零表示不限制起点（全部）；
// start 非零但 >= end 时回退到今日 0 点（必要时 end 前 1 小时）。
func normalizeRange(start, end time.Time) (time.Time, time.Time) {
	now := time.Now()
	if end.IsZero() || end.After(now) {
		end = now
	}
	if start.IsZero() {
		return start, end
	}
	if !start.Before(end) {
		start = dayStart(now)
		if !start.Before(end) {
			start = end.Add(-time.Hour)
		}
	}
	return start, end
}

// rangeStatsLabel 为时间段生成简洁中文标签：区间截止到“现在”时用相对描述（今日/近 N 小时/近 N 天），
// 否则展示具体的日期区间。start 为零表示全部。
func rangeStatsLabel(start, end, now time.Time) string {
	if start.IsZero() {
		return "全部"
	}
	if end.Before(now.Add(-2 * time.Minute)) {
		return start.Format("01-02 15:04") + " ~ " + end.Format("01-02 15:04")
	}
	if start.Equal(dayStart(now)) {
		return "今日"
	}
	mins := int64(now.Sub(start).Minutes())
	switch {
	case mins <= int64(minutesPerDay) && mins%60 == 0:
		return "近 " + strconv.FormatInt(mins/60, 10) + " 小时"
	case mins%int64(minutesPerDay) == 0:
		return "近 " + strconv.FormatInt(mins/int64(minutesPerDay), 10) + " 天"
	case mins%60 == 0:
		return "近 " + strconv.FormatInt(mins/60, 10) + " 小时"
	default:
		return "近 " + strconv.FormatInt(mins, 10) + " 分钟"
	}
}

func normalizeRangeDays(rangeDays, historyDays int32) int32 {
	if rangeDays <= 1 {
		return 1
	}
	if historyDays > 0 && rangeDays > historyDays {
		return historyDays
	}
	if rangeDays > MaxHistoryDays {
		return MaxHistoryDays
	}
	return rangeDays
}

func rangeLabel(days int32) string {
	if days <= 1 {
		return "今日"
	}
	return "最近 " + strconv.FormatInt(int64(days), 10) + " 天"
}

// ---------- 数据层聚合结果 → proto 映射 ----------

// dayStart 返回 t 当日 0 点（本地时区）。
func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// mapTopItems 将维度聚合排行映射为 proto（成功率按计费请求算；顺序沿用数据层 token 降序）。
func mapTopItems(stats []TopStat) []*v1.Sub2APITopItem {
	items := make([]*v1.Sub2APITopItem, 0, len(stats))
	for _, s := range stats {
		items = append(items, &v1.Sub2APITopItem{
			Name:         s.Name,
			RequestCount: s.Request,
			TokenCount:   s.Token,
			SuccessRate:  percent(s.Success, s.Request),
			AvgTps:       s.AvgTPS,
		})
	}
	return items
}

// dailyTrendPoints 将按日聚合结果补齐为 [startDay, startDay+days) 每一天的趋势点（缺口填空点）。
func dailyTrendPoints(stats []DayStat, startDay time.Time, days int32) []*v1.Sub2APITrendPoint {
	byDate := make(map[string]DayStat, len(stats))
	for _, s := range stats {
		byDate[s.Date] = s
	}
	result := make([]*v1.Sub2APITrendPoint, 0, days)
	for i := int32(0); i < days; i++ {
		date := startDay.AddDate(0, 0, int(i)).Format("2006-01-02")
		s, ok := byDate[date]
		if !ok {
			result = append(result, &v1.Sub2APITrendPoint{Date: date})
			continue
		}
		result = append(result, &v1.Sub2APITrendPoint{
			Date:             date,
			RequestCount:     s.Request,
			SuccessCount:     s.Success,
			SuccessRate:      percent(s.Success, s.Request),
			AverageLatencyMs: s.AvgLatencyMS,
			TokenCount:       s.Token,
		})
	}
	return result
}

// intradayTrendPoints 将 15 分钟桶聚合映射为日内趋势点（标签 HH:MM）。
func intradayTrendPoints(series []BucketStat) []*v1.Sub2APITrendPoint {
	points := make([]*v1.Sub2APITrendPoint, 0, len(series))
	for _, b := range series {
		points = append(points, &v1.Sub2APITrendPoint{
			Date:         b.Bucket.Format("15:04"),
			RequestCount: b.Request,
			SuccessCount: b.Success,
			SuccessRate:  percent(b.Success, b.Request),
			TokenCount:   b.Token,
		})
	}
	return points
}

// todaySeriesPoints 将 15 分钟桶聚合映射为当日“成功率 + 生成速度”时间序列（前端 time 轴）。
func todaySeriesPoints(series []BucketStat) []*v1.Sub2APITimePoint {
	if len(series) == 0 {
		return nil
	}
	points := make([]*v1.Sub2APITimePoint, 0, len(series))
	for _, b := range series {
		points = append(points, &v1.Sub2APITimePoint{
			Time:         timestamppb.New(b.Bucket),
			RequestCount: b.Total,
			SuccessRate:  percent(b.Success, b.Total),
			AvgTps:       b.AvgTPS,
		})
	}
	return points
}

// trendDayRange 计算按天补齐的起始日与天数。start 为零时以数据中最早日期为起点；无数据返回 (零值, 0)。
func trendDayRange(start, end time.Time, stats []DayStat) (time.Time, int32) {
	var startDay time.Time
	if start.IsZero() {
		if len(stats) == 0 {
			return time.Time{}, 0
		}
		first, err := time.ParseInLocation("2006-01-02", stats[0].Date, time.Local)
		if err != nil {
			return time.Time{}, 0
		}
		startDay = first
	} else {
		startDay = dayStart(start)
	}
	endDay := dayStart(end)
	days := int32(endDay.Sub(startDay).Hours()/24) + 1
	if days < 1 {
		days = 1
	}
	return startDay, days
}

// toRecentRequest 将领域记录映射为最近请求 DTO（含详情字段）。
func toRecentRequest(record *UsageRecord) *v1.Sub2APIRecentRequest {
	return &v1.Sub2APIRecentRequest{
		RequestId:       record.ID,
		Model:           record.Model,
		Endpoint:        record.Endpoint,
		Status:          record.Status,
		Success:         record.Success,
		LatencyMs:       record.LatencyMS,
		TokenCount:      record.TokenCount,
		RequestTime:     timestamppb.New(record.RequestTime),
		Cost:            record.Cost,
		FirstTokenMs:    record.FirstTokenMS,
		ReasoningEffort: record.ReasoningEffort,
		AccountName:     record.AccountName,
		ErrorMessage:    record.ErrorMessage,
		HttpStatus:      int32(record.HTTPStatus),
		GroupName:       record.GroupName,
	}
}

// toRecentRequests 映射一页记录（已按时间倒序）。
func toRecentRequests(records []*UsageRecord) []*v1.Sub2APIRecentRequest {
	result := make([]*v1.Sub2APIRecentRequest, 0, len(records))
	for _, record := range records {
		result = append(result, toRecentRequest(record))
	}
	return result
}

func percent(part, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) * 100 / float64(total)
}

func timestampOrNil(t *time.Time) *timestamppb.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return timestamppb.New(*t)
}
