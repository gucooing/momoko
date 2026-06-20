package sub2api

import (
	"context"
	"sort"
	"strconv"
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
// 整段只读取一次记录：全量统计、窗口趋势/排行、今日曲线与最近请求都在内存中算出，
// 把原先一个首页数十次的聚合查询降到一次。
func BuildSnapshot(ctx context.Context, store UsageStore, cfg *v1.Sub2APIConfig, state SyncState) (*v1.Sub2APIUsageSnapshot, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	windowStart := todayStart.AddDate(0, 0, -int(cfg.HistoryDays)+1)

	records, err := store.RecordsSince(ctx, nil)
	if err != nil {
		return nil, err
	}
	windowRecords := recordsFrom(records, windowStart)
	todayRecords := recordsFrom(records, todayStart)

	totals := totalsFromRecords(records)
	todayTotals := totalsFromRecords(todayRecords)

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
		Trend:                 buildTrend(windowRecords, windowStart, cfg.HistoryDays),
		ModelUsage:            buildTopItems(windowRecords, GroupByModel, topItemsLimit),
		EndpointUsage:         buildTopItems(windowRecords, GroupByEndpoint, topItemsLimit),
		RecentRequests:        buildRecentRequests(records),
		RecentTps:             tps,
		RecentQps:             0,
		RecentWindowSeconds:   0,
		TodaySeries:           buildTodaySeries(todayRecords),
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
// 区间记录只读取一次，汇总、趋势、模型/接口排行都在内存中算出。
func BuildStats(ctx context.Context, store UsageStore, cfg *v1.Sub2APIConfig, rangeDays int32) (*v1.Sub2APIStats, error) {
	days := normalizeRangeDays(rangeDays, cfg.HistoryDays)
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	start := todayStart.AddDate(0, 0, -(int(days) - 1))

	records, err := store.RecordsSince(ctx, &start)
	if err != nil {
		return nil, err
	}
	totals := totalsFromRecords(records)

	// 单日（今日）按时间分桶展示日内趋势，否则按天展示，避免单日只有一个数据点。
	var trend []*v1.Sub2APITrendPoint
	if days <= 1 {
		trend = buildIntradayTrend(records)
	} else {
		trend = buildTrend(records, start, days)
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
		Models:           buildTopItems(records, GroupByModel, 0),
		Endpoints:        buildTopItems(records, GroupByEndpoint, 0),
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

// recordsFrom 返回 records 中 RequestTime >= start 的尾部切片（records 已按时间升序）。
func recordsFrom(records []*UsageRecord, start time.Time) []*UsageRecord {
	i := sort.Search(len(records), func(i int) bool {
		return !records[i].RequestTime.Before(start)
	})
	return records[i:]
}

// totalsFromRecords 在内存中汇总一批记录的指标，口径与原先按区间聚合一致：
// 计费请求=成功或上游错误；平均延迟仅统计成功请求；平均生成速度仅统计 tps>0 的请求；Token 计入全部记录。
func totalsFromRecords(records []*UsageRecord) Totals {
	var (
		totals       Totals
		latencySum   float64
		latencyCount int64
		tpsSum       float64
		tpsCount     int64
	)
	for _, rec := range records {
		totals.TotalCount++
		totals.TokenCount += rec.TokenCount
		if isRateEligible(rec) {
			totals.RequestCount++
		}
		if rec.Success {
			totals.SuccessCount++
			latencySum += float64(rec.LatencyMS)
			latencyCount++
		}
		if rec.TPS > 0 {
			tpsSum += rec.TPS
			tpsCount++
		}
	}
	if latencyCount > 0 {
		totals.AverageLatencyMS = latencySum / float64(latencyCount)
	}
	if tpsCount > 0 {
		totals.AverageTPS = tpsSum / float64(tpsCount)
	}
	return totals
}

// buildTrend 按天聚合记录为趋势点，并补齐 [start, start+days) 区间内没有数据的日期。
func buildTrend(records []*UsageRecord, start time.Time, days int32) []*v1.Sub2APITrendPoint {
	type dayAgg struct {
		request, success, token int64
		latencySum              float64
		latencyCount            int64
	}
	daily := make(map[string]*dayAgg)
	for _, rec := range records {
		agg := daily[rec.RequestDate]
		if agg == nil {
			agg = &dayAgg{}
			daily[rec.RequestDate] = agg
		}
		if isRateEligible(rec) {
			agg.request++
			agg.token += rec.TokenCount
		}
		if rec.Success {
			agg.success++
			agg.latencySum += float64(rec.LatencyMS)
			agg.latencyCount++
		}
	}

	result := make([]*v1.Sub2APITrendPoint, 0, days)
	for i := int32(0); i < days; i++ {
		date := start.AddDate(0, 0, int(i)).Format("2006-01-02")
		agg := daily[date]
		if agg == nil {
			result = append(result, &v1.Sub2APITrendPoint{Date: date})
			continue
		}
		point := &v1.Sub2APITrendPoint{
			Date:         date,
			RequestCount: agg.request,
			SuccessCount: agg.success,
			SuccessRate:  percent(agg.success, agg.request),
			TokenCount:   agg.token,
		}
		if agg.latencyCount > 0 {
			point.AverageLatencyMs = agg.latencySum / float64(agg.latencyCount)
		}
		result = append(result, point)
	}
	return result
}

// buildTopItems 按模型/接口维度聚合记录，仅保留有计费请求的分组，按请求数、Token 排序后截断。
func buildTopItems(records []*UsageRecord, field GroupField, limit int) []*v1.Sub2APITopItem {
	type itemAgg struct {
		request, success, token int64
		tpsSum                  float64
		tpsCount                int64
	}
	groups := make(map[string]*itemAgg)
	for _, rec := range records {
		name := rec.Model
		if field == GroupByEndpoint {
			name = rec.Endpoint
		}
		if name == "" {
			continue // 排除分组维度为空的记录，避免上游错误等无模型记录污染统计
		}
		agg := groups[name]
		if agg == nil {
			agg = &itemAgg{}
			groups[name] = agg
		}
		if isRateEligible(rec) {
			agg.request++
			agg.token += rec.TokenCount
		}
		if rec.Success {
			agg.success++
		}
		if rec.TPS > 0 {
			agg.tpsSum += rec.TPS
			agg.tpsCount++
		}
	}

	items := make([]*v1.Sub2APITopItem, 0, len(groups))
	for name, agg := range groups {
		if agg.request == 0 {
			continue // 只保留有计费请求的分组
		}
		item := &v1.Sub2APITopItem{
			Name:         name,
			RequestCount: agg.request,
			TokenCount:   agg.token,
			SuccessRate:  percent(agg.success, agg.request),
		}
		if agg.tpsCount > 0 {
			item.AvgTps = agg.tpsSum / float64(agg.tpsCount)
		}
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].RequestCount != items[j].RequestCount {
			return items[i].RequestCount > items[j].RequestCount
		}
		if items[i].TokenCount != items[j].TokenCount {
			return items[i].TokenCount > items[j].TokenCount
		}
		return items[i].Name < items[j].Name
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

// buildRecentRequests 取最近的若干条请求（records 按时间升序，取末尾并倒序输出，最新在前）。
func buildRecentRequests(records []*UsageRecord) []*v1.Sub2APIRecentRequest {
	count := recentRecordsLimit
	if len(records) < count {
		count = len(records)
	}
	result := make([]*v1.Sub2APIRecentRequest, 0, count)
	for i := 0; i < count; i++ {
		record := records[len(records)-1-i]
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
	return result
}

// buildTodaySeries 将当日记录按固定时间桶聚合为“成功率 + 生成速度”时间序列，
// 供首页绘制随时间移动的曲线（前端使用 time 轴，短时段也能铺满）。
func buildTodaySeries(records []*UsageRecord) []*v1.Sub2APITimePoint {
	if len(records) == 0 {
		return nil
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
		if rec.TPS > 0 {
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
	return points
}

func isRateEligible(rec *UsageRecord) bool {
	return rec.Success || rec.Status == StatusUpstreamError
}

// buildIntradayTrend 将当日记录按时间桶聚合为趋势点（标签为 HH:MM），
// 用于详情页“今日”视图，避免按天聚合时单日只有一个数据点。
func buildIntradayTrend(records []*UsageRecord) []*v1.Sub2APITrendPoint {
	if len(records) == 0 {
		return nil
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
	return points
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
