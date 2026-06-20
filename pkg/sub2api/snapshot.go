package sub2api

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	v1 "momoko/api/gen/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const todayBucketMinutes = 15

const (
	topItemsLimit      = 8
	recentRecordsLimit = 8
)

// BuildSnapshot 聚合本地使用记录，构造完整的用量快照（含派生指标）。
func BuildSnapshot(ctx context.Context, store UsageStore, cfg *v1.Sub2APIConfig, state SyncState) (*v1.Sub2APIUsageSnapshot, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	windowStart := todayStart.AddDate(0, 0, -int(cfg.HistoryDays)+1)

	totals, err := store.Totals(ctx, nil, true)
	if err != nil {
		return nil, err
	}
	todayTotals, err := store.Totals(ctx, &todayStart, true)
	if err != nil {
		return nil, err
	}
	trend, err := buildTrend(ctx, store, windowStart, cfg.HistoryDays)
	if err != nil {
		return nil, err
	}
	modelUsage, err := buildTopItems(ctx, store, windowStart, GroupByModel, topItemsLimit)
	if err != nil {
		return nil, err
	}
	endpointUsage, err := buildTopItems(ctx, store, windowStart, GroupByEndpoint, topItemsLimit)
	if err != nil {
		return nil, err
	}
	recent, err := buildRecentRequests(ctx, store)
	if err != nil {
		return nil, err
	}
	todaySeries, err := buildTodaySeries(ctx, store, todayStart)
	if err != nil {
		return nil, err
	}

	status := state.Status
	if status == "" {
		status = SyncStatusIdle
	}
	// Token 生成速度：按请求计算（输出token/秒，不含缓存），对 tps>0 的请求取平均，剔除 mini 测试模型。
	tps := todayTotals.AverageTPS
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
		Trend:                 trend,
		ModelUsage:            modelUsage,
		EndpointUsage:         endpointUsage,
		RecentRequests:        recent,
		RecentTps:             tps,
		RecentQps:             0,
		RecentWindowSeconds:   0,
		TodaySeries:           todaySeries,
	}, nil
}

// StripForPublic 移除公开首页不需要的明细，避免泄露逐条请求信息。
func StripForPublic(snapshot *v1.Sub2APIUsageSnapshot) {
	if snapshot == nil {
		return
	}
	snapshot.RecentRequests = nil
	snapshot.EndpointUsage = nil
	for _, item := range snapshot.Trend {
		item.AverageLatencyMs = 0
	}
}

// BuildStats 聚合指定时间区间（今日/最近 N 天）的用量统计，按模型/接口拆分。
func BuildStats(ctx context.Context, store UsageStore, cfg *v1.Sub2APIConfig, rangeDays int32) (*v1.Sub2APIStats, error) {
	days := normalizeRangeDays(rangeDays, cfg.HistoryDays)
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	start := todayStart.AddDate(0, 0, -(int(days) - 1))

	totals, err := store.Totals(ctx, &start, false)
	if err != nil {
		return nil, err
	}
	// 单日（今日）按时间分桶展示日内趋势，否则按天展示，避免单日只有一个数据点。
	var trend []*v1.Sub2APITrendPoint
	if days <= 1 {
		trend, err = buildIntradayTrend(ctx, store, todayStart)
	} else {
		trend, err = buildTrend(ctx, store, start, days)
	}
	if err != nil {
		return nil, err
	}
	models, err := buildTopItems(ctx, store, start, GroupByModel, 0)
	if err != nil {
		return nil, err
	}
	endpoints, err := buildTopItems(ctx, store, start, GroupByEndpoint, 0)
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
		Models:           models,
		Endpoints:        endpoints,
	}, nil
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

func buildTrend(ctx context.Context, store UsageStore, start time.Time, days int32) ([]*v1.Sub2APITrendPoint, error) {
	usageRows, err := store.DailyUsage(ctx, start)
	if err != nil {
		return nil, err
	}
	successRows, err := store.DailySuccess(ctx, start)
	if err != nil {
		return nil, err
	}
	latencyRows, err := store.DailyLatency(ctx, start)
	if err != nil {
		return nil, err
	}

	stats := make(map[string]*v1.Sub2APITrendPoint, len(usageRows))
	for _, row := range usageRows {
		stats[row.Date] = &v1.Sub2APITrendPoint{
			Date:         row.Date,
			RequestCount: row.RequestCount,
			TokenCount:   row.TokenCount,
		}
	}
	for _, row := range successRows {
		if stat := stats[row.Date]; stat != nil {
			stat.SuccessCount = row.Count
			stat.SuccessRate = percent(row.Count, stat.RequestCount)
		}
	}
	for _, row := range latencyRows {
		if stat := stats[row.Date]; stat != nil {
			stat.AverageLatencyMs = row.AverageLatencyMS
		}
	}

	result := make([]*v1.Sub2APITrendPoint, 0, days)
	for i := int32(0); i < days; i++ {
		date := start.AddDate(0, 0, int(i)).Format("2006-01-02")
		if stat := stats[date]; stat != nil {
			result = append(result, stat)
		} else {
			result = append(result, &v1.Sub2APITrendPoint{Date: date})
		}
	}
	return result, nil
}

func buildTopItems(ctx context.Context, store UsageStore, start time.Time, field GroupField, limit int) ([]*v1.Sub2APITopItem, error) {
	usageRows, err := store.TopUsage(ctx, start, field)
	if err != nil {
		return nil, err
	}
	successRows, err := store.TopSuccess(ctx, start, field)
	if err != nil {
		return nil, err
	}

	successByName := make(map[string]int64, len(successRows))
	for _, row := range successRows {
		successByName[row.Name] = row.Count
	}
	sort.SliceStable(usageRows, func(i, j int) bool {
		if usageRows[i].RequestCount == usageRows[j].RequestCount {
			return usageRows[i].TokenCount > usageRows[j].TokenCount
		}
		return usageRows[i].RequestCount > usageRows[j].RequestCount
	})
	if limit > 0 && len(usageRows) > limit {
		usageRows = usageRows[:limit]
	}

	result := make([]*v1.Sub2APITopItem, 0, len(usageRows))
	for _, row := range usageRows {
		name := row.Name
		if name == "" {
			name = "未标记"
		}
		result = append(result, &v1.Sub2APITopItem{
			Name:         name,
			RequestCount: row.RequestCount,
			TokenCount:   row.TokenCount,
			SuccessRate:  percent(successByName[row.Name], row.RequestCount),
			AvgTps:       row.AverageTPS,
		})
	}
	return result, nil
}

func buildRecentRequests(ctx context.Context, store UsageStore) ([]*v1.Sub2APIRecentRequest, error) {
	records, err := store.RecentRecords(ctx, recentRecordsLimit)
	if err != nil {
		return nil, err
	}
	result := make([]*v1.Sub2APIRecentRequest, 0, len(records))
	for _, record := range records {
		result = append(result, &v1.Sub2APIRecentRequest{
			RequestId:   record.ID,
			Model:       record.Model,
			Endpoint:    record.Endpoint,
			Status:      record.Status,
			Success:     record.Success,
			LatencyMs:   record.LatencyMS,
			TokenCount:  record.TokenCount,
			RequestTime: timestamppb.New(record.RequestTime),
		})
	}
	return result, nil
}

// buildTodaySeries 将当日记录按固定时间桶聚合为“成功率 + 生成速度”时间序列，
// 供首页绘制随时间移动的曲线（前端使用 time 轴，短时段也能铺满）。
func buildTodaySeries(ctx context.Context, store UsageStore, todayStart time.Time) ([]*v1.Sub2APITimePoint, error) {
	records, err := store.RecordsSince(ctx, todayStart)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	bucket := time.Duration(todayBucketMinutes) * time.Minute
	type bucketAgg struct {
		total    int64
		success  int64
		tpsSum   float64
		tpsCount int64
	}
	aggs := make(map[int64]*bucketAgg)
	order := make([]int64, 0)
	for _, rec := range records {
		key := rec.RequestTime.Truncate(bucket).Unix()
		agg := aggs[key]
		if agg == nil {
			agg = &bucketAgg{}
			aggs[key] = agg
			order = append(order, key)
		}
		agg.total++
		if rec.Success {
			agg.success++
		}
		if rec.TPS > 0 && !isTestModel(rec.Model) {
			agg.tpsSum += rec.TPS
			agg.tpsCount++
		}
	}
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })

	points := make([]*v1.Sub2APITimePoint, 0, len(order))
	for _, key := range order {
		agg := aggs[key]
		point := &v1.Sub2APITimePoint{
			Time:         timestamppb.New(time.Unix(key, 0)),
			RequestCount: agg.total,
			SuccessRate:  percent(agg.success, agg.total),
		}
		if agg.tpsCount > 0 {
			point.AvgTps = agg.tpsSum / float64(agg.tpsCount)
		}
		points = append(points, point)
	}
	return points, nil
}

// isTestModel 判定是否为后端测试模型（名称含 mini），用于在生成速度统计中剔除。
func isTestModel(model string) bool {
	return strings.Contains(strings.ToLower(model), "mini")
}

func isRateEligible(rec *UsageRecord) bool {
	return rec.Success || rec.Status == StatusUpstreamError
}

// buildIntradayTrend 将当日记录按时间桶聚合为趋势点（标签为 HH:MM），
// 用于详情页“今日”视图，避免按天聚合时单日只有一个数据点。
func buildIntradayTrend(ctx context.Context, store UsageStore, todayStart time.Time) ([]*v1.Sub2APITrendPoint, error) {
	records, err := store.RecordsSince(ctx, todayStart)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	bucket := time.Duration(todayBucketMinutes) * time.Minute
	type agg struct {
		request int64
		success int64
		token   int64
	}
	aggs := make(map[int64]*agg)
	order := make([]int64, 0)
	for _, rec := range records {
		key := rec.RequestTime.Truncate(bucket).Unix()
		a := aggs[key]
		if a == nil {
			a = &agg{}
			aggs[key] = a
			order = append(order, key)
		}
		if isRateEligible(rec) {
			a.request++
			a.token += rec.TokenCount
		}
		if rec.Success {
			a.success++
		}
	}
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })

	points := make([]*v1.Sub2APITrendPoint, 0, len(order))
	for _, key := range order {
		a := aggs[key]
		points = append(points, &v1.Sub2APITrendPoint{
			Date:         time.Unix(key, 0).Format("15:04"),
			RequestCount: a.request,
			SuccessCount: a.success,
			SuccessRate:  percent(a.success, a.request),
			TokenCount:   a.token,
		})
	}
	return points, nil
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
