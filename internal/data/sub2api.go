package data

import (
	"context"
	"time"

	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/predicate"
	"momoko/internal/data/ent/gen/sub2apiannouncement"
	"momoko/internal/data/ent/gen/sub2apitimelineitem"
	"momoko/internal/data/ent/gen/sub2apiusagerecord"
	sub2apipkg "momoko/pkg/sub2api"
)

type sub2APIRepo struct {
	data *Data
}

func NewSub2APIRepo(data *Data) biz.Sub2APIRepo {
	return &sub2APIRepo{data: data}
}

// ---------- 使用记录：写入与查询（实现 pkg/sub2api.UsageStore）----------

func (r *sub2APIRepo) SaveUsageRecords(ctx context.Context, records []*sub2apipkg.UsageRecord) error {
	const batchSize = 500
	for start := 0; start < len(records); start += batchSize {
		end := start + batchSize
		if end > len(records) {
			end = len(records)
		}
		builders := make([]*gen.Sub2APIUsageRecordCreate, 0, end-start)
		for _, record := range records[start:end] {
			requestDate := record.RequestDate
			if requestDate == "" {
				requestDate = record.RequestTime.In(time.Local).Format("2006-01-02")
			}
			builders = append(builders, r.data.db.Sub2APIUsageRecord.Create().
				SetID(record.ID).
				SetRequestTime(record.RequestTime).
				SetRequestDate(requestDate).
				SetModel(record.Model).
				SetEndpoint(record.Endpoint).
				SetStatus(record.Status).
				SetSuccess(record.Success).
				SetLatencyMs(record.LatencyMS).
				SetTokenCount(record.TokenCount).
				SetOutputTokens(record.OutputTokens).
				SetTps(record.TPS),
			)
		}
		if err := r.data.db.Sub2APIUsageRecord.CreateBulk(builders...).
			OnConflictColumns(sub2apiusagerecord.FieldID).
			DoNothing().
			Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r *sub2APIRepo) ClearUsageRecords(ctx context.Context) error {
	_, err := r.data.db.Sub2APIUsageRecord.Delete().Exec(ctx)
	return err
}

func (r *sub2APIRepo) LatestUsageRecordTime(ctx context.Context) (*time.Time, error) {
	record, err := r.data.db.Sub2APIUsageRecord.Query().
		Order(gen.Desc(sub2apiusagerecord.FieldRequestTime)).
		First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &record.RequestTime, nil
}

func (r *sub2APIRepo) LatestUpstreamErrorRecordTime(ctx context.Context) (*time.Time, error) {
	record, err := r.data.db.Sub2APIUsageRecord.Query().
		Where(sub2apiusagerecord.StatusEQ(sub2apipkg.StatusUpstreamError)).
		Order(gen.Desc(sub2apiusagerecord.FieldRequestTime)).
		First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &record.RequestTime, nil
}

func (r *sub2APIRepo) Totals(ctx context.Context, start *time.Time, excludeTestModels bool) (sub2apipkg.Totals, error) {
	base := func() *gen.Sub2APIUsageRecordQuery {
		query := r.data.db.Sub2APIUsageRecord.Query()
		if start != nil {
			query = query.Where(sub2apiusagerecord.RequestTimeGTE(*start))
		}
		return query
	}

	totalCount, err := base().Count(ctx)
	if err != nil || totalCount == 0 {
		return sub2apipkg.Totals{}, err
	}
	requestCount, err := base().Where(rateEligiblePredicate()).Count(ctx)
	if err != nil {
		return sub2apipkg.Totals{}, err
	}
	successCount, err := base().Where(sub2apiusagerecord.SuccessEQ(true)).Count(ctx)
	if err != nil {
		return sub2apipkg.Totals{}, err
	}

	var tokenRows []struct {
		TokenCount int64 `json:"token_count"`
	}
	if err = base().Aggregate(
		gen.As(gen.Sum(sub2apiusagerecord.FieldTokenCount), "token_count"),
	).Scan(ctx, &tokenRows); err != nil {
		return sub2apipkg.Totals{}, err
	}

	var latencyRows []struct {
		AverageLatencyMS float64 `json:"average_latency_ms"`
	}
	if err = base().Where(sub2apiusagerecord.SuccessEQ(true)).Aggregate(
		gen.As(gen.Mean(sub2apiusagerecord.FieldLatencyMs), "average_latency_ms"),
	).Scan(ctx, &latencyRows); err != nil {
		return sub2apipkg.Totals{}, err
	}

	var tpsRows []struct {
		AverageTPS float64 `json:"average_tps"`
	}
	tpsQuery := base().Where(sub2apiusagerecord.TpsGT(0))
	if excludeTestModels {
		tpsQuery = tpsQuery.Where(excludeTestModelPredicate())
	}
	if err = tpsQuery.Aggregate(
		gen.As(gen.Mean(sub2apiusagerecord.FieldTps), "average_tps"),
	).Scan(ctx, &tpsRows); err != nil {
		return sub2apipkg.Totals{}, err
	}

	totals := sub2apipkg.Totals{
		TotalCount:   int64(totalCount),
		RequestCount: int64(requestCount),
		SuccessCount: int64(successCount),
	}
	if len(tokenRows) > 0 {
		totals.TokenCount = tokenRows[0].TokenCount
	}
	if len(latencyRows) > 0 {
		totals.AverageLatencyMS = latencyRows[0].AverageLatencyMS
	}
	if len(tpsRows) > 0 {
		totals.AverageTPS = tpsRows[0].AverageTPS
	}
	return totals, nil
}

func (r *sub2APIRepo) DailyUsage(ctx context.Context, start time.Time) ([]sub2apipkg.DailyUsageRow, error) {
	var rows []struct {
		RequestDate  string `json:"request_date"`
		RequestCount int64  `json:"count"`
		TokenCount   int64  `json:"token_count"`
	}
	if err := r.data.db.Sub2APIUsageRecord.Query().
		Where(sub2apiusagerecord.RequestTimeGTE(start), rateEligiblePredicate()).
		GroupBy(sub2apiusagerecord.FieldRequestDate).
		Aggregate(gen.Count(), gen.As(gen.Sum(sub2apiusagerecord.FieldTokenCount), "token_count")).
		Scan(ctx, &rows); err != nil {
		return nil, err
	}
	result := make([]sub2apipkg.DailyUsageRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, sub2apipkg.DailyUsageRow{
			Date:         row.RequestDate,
			RequestCount: row.RequestCount,
			TokenCount:   row.TokenCount,
		})
	}
	return result, nil
}

func (r *sub2APIRepo) DailySuccess(ctx context.Context, start time.Time) ([]sub2apipkg.DateCountRow, error) {
	var rows []struct {
		RequestDate string `json:"request_date"`
		Count       int64  `json:"count"`
	}
	if err := r.data.db.Sub2APIUsageRecord.Query().
		Where(sub2apiusagerecord.RequestTimeGTE(start), sub2apiusagerecord.SuccessEQ(true)).
		GroupBy(sub2apiusagerecord.FieldRequestDate).
		Aggregate(gen.Count()).
		Scan(ctx, &rows); err != nil {
		return nil, err
	}
	result := make([]sub2apipkg.DateCountRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, sub2apipkg.DateCountRow{Date: row.RequestDate, Count: row.Count})
	}
	return result, nil
}

func (r *sub2APIRepo) DailyLatency(ctx context.Context, start time.Time) ([]sub2apipkg.DateLatencyRow, error) {
	var rows []struct {
		RequestDate      string  `json:"request_date"`
		AverageLatencyMS float64 `json:"average_latency_ms"`
	}
	if err := r.data.db.Sub2APIUsageRecord.Query().
		Where(sub2apiusagerecord.RequestTimeGTE(start), sub2apiusagerecord.SuccessEQ(true)).
		GroupBy(sub2apiusagerecord.FieldRequestDate).
		Aggregate(gen.As(gen.Mean(sub2apiusagerecord.FieldLatencyMs), "average_latency_ms")).
		Scan(ctx, &rows); err != nil {
		return nil, err
	}
	result := make([]sub2apipkg.DateLatencyRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, sub2apipkg.DateLatencyRow{Date: row.RequestDate, AverageLatencyMS: row.AverageLatencyMS})
	}
	return result, nil
}

func (r *sub2APIRepo) TopUsage(ctx context.Context, start time.Time, field sub2apipkg.GroupField) ([]sub2apipkg.NamedUsageRow, error) {
	column := groupFieldColumn(field)
	var rows []struct {
		Model        string `json:"model"`
		Endpoint     string `json:"endpoint"`
		RequestCount int64  `json:"count"`
		TokenCount   int64  `json:"token_count"`
	}
	if err := r.data.db.Sub2APIUsageRecord.Query().
		Where(sub2apiusagerecord.RequestTimeGTE(start), rateEligiblePredicate(), nonEmptyGroupPredicate(field)).
		GroupBy(column).
		Aggregate(gen.Count(), gen.As(gen.Sum(sub2apiusagerecord.FieldTokenCount), "token_count")).
		Scan(ctx, &rows); err != nil {
		return nil, err
	}

	// 平均 token 生成速度：仅统计 tps>0 的请求，0 不参与计算以免拉低均值。
	// 此处按模型拆分，mini 等测试模型作为独立条目正常统计，不在明细中剔除。
	var tpsRows []struct {
		Model      string  `json:"model"`
		Endpoint   string  `json:"endpoint"`
		AverageTPS float64 `json:"average_tps"`
	}
	if err := r.data.db.Sub2APIUsageRecord.Query().
		Where(sub2apiusagerecord.RequestTimeGTE(start), sub2apiusagerecord.TpsGT(0), nonEmptyGroupPredicate(field)).
		GroupBy(column).
		Aggregate(gen.As(gen.Mean(sub2apiusagerecord.FieldTps), "average_tps")).
		Scan(ctx, &tpsRows); err != nil {
		return nil, err
	}
	tpsByName := make(map[string]float64, len(tpsRows))
	for _, row := range tpsRows {
		tpsByName[pickGroupName(field, row.Model, row.Endpoint)] = row.AverageTPS
	}

	result := make([]sub2apipkg.NamedUsageRow, 0, len(rows))
	for _, row := range rows {
		name := pickGroupName(field, row.Model, row.Endpoint)
		result = append(result, sub2apipkg.NamedUsageRow{
			Name:         name,
			RequestCount: row.RequestCount,
			TokenCount:   row.TokenCount,
			AverageTPS:   tpsByName[name],
		})
	}
	return result, nil
}

func (r *sub2APIRepo) TopSuccess(ctx context.Context, start time.Time, field sub2apipkg.GroupField) ([]sub2apipkg.NameCountRow, error) {
	column := groupFieldColumn(field)
	var rows []struct {
		Model    string `json:"model"`
		Endpoint string `json:"endpoint"`
		Count    int64  `json:"count"`
	}
	if err := r.data.db.Sub2APIUsageRecord.Query().
		Where(sub2apiusagerecord.RequestTimeGTE(start), sub2apiusagerecord.SuccessEQ(true), nonEmptyGroupPredicate(field)).
		GroupBy(column).
		Aggregate(gen.Count()).
		Scan(ctx, &rows); err != nil {
		return nil, err
	}
	result := make([]sub2apipkg.NameCountRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, sub2apipkg.NameCountRow{
			Name:  pickGroupName(field, row.Model, row.Endpoint),
			Count: row.Count,
		})
	}
	return result, nil
}

func (r *sub2APIRepo) RecentRecords(ctx context.Context, limit int) ([]*sub2apipkg.UsageRecord, error) {
	records, err := r.data.db.Sub2APIUsageRecord.Query().
		Order(gen.Desc(sub2apiusagerecord.FieldRequestTime)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*sub2apipkg.UsageRecord, 0, len(records))
	for _, record := range records {
		result = append(result, toUsageRecord(record))
	}
	return result, nil
}

func (r *sub2APIRepo) RecordsSince(ctx context.Context, start time.Time) ([]*sub2apipkg.UsageRecord, error) {
	records, err := r.data.db.Sub2APIUsageRecord.Query().
		Where(sub2apiusagerecord.RequestTimeGTE(start)).
		Order(gen.Asc(sub2apiusagerecord.FieldRequestTime)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*sub2apipkg.UsageRecord, 0, len(records))
	for _, record := range records {
		result = append(result, toUsageRecord(record))
	}
	return result, nil
}

func toUsageRecord(record *gen.Sub2APIUsageRecord) *sub2apipkg.UsageRecord {
	return &sub2apipkg.UsageRecord{
		ID:           record.ID,
		RequestTime:  record.RequestTime,
		RequestDate:  record.RequestDate,
		Model:        record.Model,
		Endpoint:     record.Endpoint,
		Status:       record.Status,
		Success:      record.Success,
		LatencyMS:    record.LatencyMs,
		TokenCount:   record.TokenCount,
		OutputTokens: record.OutputTokens,
		TPS:          record.Tps,
	}
}

func groupFieldColumn(field sub2apipkg.GroupField) string {
	if field == sub2apipkg.GroupByEndpoint {
		return sub2apiusagerecord.FieldEndpoint
	}
	return sub2apiusagerecord.FieldModel
}

// nonEmptyGroupPredicate 排除分组维度为空的记录，避免上游错误等无模型记录污染为“未标记”。
func nonEmptyGroupPredicate(field sub2apipkg.GroupField) predicate.Sub2APIUsageRecord {
	if field == sub2apipkg.GroupByEndpoint {
		return sub2apiusagerecord.EndpointNEQ("")
	}
	return sub2apiusagerecord.ModelNEQ("")
}

// excludeTestModelPredicate 在计算 token 生成速度时剔除名称含 "mini" 的后端测试模型（token 极少，会拉偏均值）。
func excludeTestModelPredicate() predicate.Sub2APIUsageRecord {
	return sub2apiusagerecord.Not(sub2apiusagerecord.ModelContainsFold("mini"))
}

func pickGroupName(field sub2apipkg.GroupField, model, endpoint string) string {
	if field == sub2apipkg.GroupByEndpoint {
		return endpoint
	}
	return model
}

func rateEligiblePredicate() predicate.Sub2APIUsageRecord {
	return sub2apiusagerecord.Or(
		sub2apiusagerecord.SuccessEQ(true),
		sub2apiusagerecord.StatusEQ(sub2apipkg.StatusUpstreamError),
	)
}

// ---------- 公告 CRUD ----------

func (r *sub2APIRepo) ListAnnouncements(ctx context.Context) ([]*gen.Sub2APIAnnouncement, error) {
	return r.data.db.Sub2APIAnnouncement.Query().
		Order(gen.Desc(sub2apiannouncement.FieldPinned), gen.Desc(sub2apiannouncement.FieldPublishedAt), gen.Desc(sub2apiannouncement.FieldCreateTime)).
		All(ctx)
}

func (r *sub2APIRepo) CreateAnnouncement(ctx context.Context, req *v1.CreateSub2APIAnnouncementRequest) (*gen.Sub2APIAnnouncement, error) {
	builder := r.data.db.Sub2APIAnnouncement.Create().
		SetID(uuid.NewString()).
		SetTitle(req.Title).
		SetContent(req.Content).
		SetLevel(announcementLevel(req.Level)).
		SetPinned(req.Pinned).
		SetPublishedAt(publishedAtOrNow(req.PublishedAt.AsTime(), req.PublishedAt != nil))
	return builder.Save(ctx)
}

func (r *sub2APIRepo) UpdateAnnouncement(ctx context.Context, req *v1.UpdateSub2APIAnnouncementRequest) (*gen.Sub2APIAnnouncement, error) {
	builder := r.data.db.Sub2APIAnnouncement.UpdateOneID(req.Id).
		SetTitle(req.Title).
		SetContent(req.Content).
		SetLevel(announcementLevel(req.Level)).
		SetPinned(req.Pinned)
	if req.PublishedAt != nil {
		builder = builder.SetPublishedAt(req.PublishedAt.AsTime())
	}
	return builder.Save(ctx)
}

func (r *sub2APIRepo) DeleteAnnouncement(ctx context.Context, id string) error {
	return r.data.db.Sub2APIAnnouncement.DeleteOneID(id).Exec(ctx)
}

// ---------- 时间线 CRUD ----------

func (r *sub2APIRepo) ListTimeline(ctx context.Context) ([]*gen.Sub2APITimelineItem, error) {
	return r.data.db.Sub2APITimelineItem.Query().
		Order(gen.Desc(sub2apitimelineitem.FieldPublishedAt), gen.Desc(sub2apitimelineitem.FieldCreateTime)).
		All(ctx)
}

func (r *sub2APIRepo) CreateTimelineItem(ctx context.Context, req *v1.CreateSub2APITimelineItemRequest) (*gen.Sub2APITimelineItem, error) {
	return r.data.db.Sub2APITimelineItem.Create().
		SetID(uuid.NewString()).
		SetTitle(req.Title).
		SetContent(req.Content).
		SetCategory(timelineCategory(req.Category)).
		SetPublishedAt(publishedAtOrNow(req.PublishedAt.AsTime(), req.PublishedAt != nil)).
		Save(ctx)
}

func (r *sub2APIRepo) UpdateTimelineItem(ctx context.Context, req *v1.UpdateSub2APITimelineItemRequest) (*gen.Sub2APITimelineItem, error) {
	builder := r.data.db.Sub2APITimelineItem.UpdateOneID(req.Id).
		SetTitle(req.Title).
		SetContent(req.Content).
		SetCategory(timelineCategory(req.Category))
	if req.PublishedAt != nil {
		builder = builder.SetPublishedAt(req.PublishedAt.AsTime())
	}
	return builder.Save(ctx)
}

func (r *sub2APIRepo) DeleteTimelineItem(ctx context.Context, id string) error {
	return r.data.db.Sub2APITimelineItem.DeleteOneID(id).Exec(ctx)
}

func announcementLevel(level string) string {
	switch level {
	case "success", "warning", "danger", "info":
		return level
	default:
		return "info"
	}
}

func timelineCategory(category string) string {
	if category == "" {
		return "更新"
	}
	return category
}

func publishedAtOrNow(t time.Time, provided bool) time.Time {
	if provided && !t.IsZero() {
		return t
	}
	return time.Now()
}
