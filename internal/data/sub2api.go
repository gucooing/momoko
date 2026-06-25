package data

import (
	"context"
	"time"

	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
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
				SetUserAgent(record.UserAgent).
				SetStatus(record.Status).
				SetSuccess(record.Success).
				SetLatencyMs(record.LatencyMS).
				SetTokenCount(record.TokenCount).
				SetOutputTokens(record.OutputTokens).
				SetTps(record.TPS).
				SetCost(record.Cost).
				SetFirstTokenMs(record.FirstTokenMS).
				SetReasoningEffort(record.ReasoningEffort).
				SetAccountName(record.AccountName).
				SetErrorMessage(record.ErrorMessage).
				SetHTTPStatus(record.HTTPStatus),
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

// RecordsSince 按时间升序读取 start（含）之后的记录；start 为 nil 时返回全部记录。
// 聚合统计统一改为读取记录后在内存中计算（见 pkg/sub2api），因此数据层只保留这一个读取入口，
// 把原先一个页面数十次的聚合查询降到一次。
func (r *sub2APIRepo) RecordsSince(ctx context.Context, start *time.Time) ([]*sub2apipkg.UsageRecord, error) {
	query := r.data.db.Sub2APIUsageRecord.Query()
	if start != nil {
		query = query.Where(sub2apiusagerecord.RequestTimeGTE(*start))
	}
	records, err := query.Order(gen.Asc(sub2apiusagerecord.FieldRequestTime)).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*sub2apipkg.UsageRecord, 0, len(records))
	for _, record := range records {
		result = append(result, toUsageRecord(record))
	}
	return result, nil
}

// RecordsPage 按时间倒序（最新在前）分页读取 [start, end] 区间内的记录，并返回区间总数。
func (r *sub2APIRepo) RecordsPage(ctx context.Context, start, end *time.Time, offset, limit int) ([]*sub2apipkg.UsageRecord, int, error) {
	query := r.data.db.Sub2APIUsageRecord.Query()
	if start != nil {
		query = query.Where(sub2apiusagerecord.RequestTimeGTE(*start))
	}
	if end != nil {
		query = query.Where(sub2apiusagerecord.RequestTimeLTE(*end))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	records, err := query.
		Order(gen.Desc(sub2apiusagerecord.FieldRequestTime)).
		Offset(offset).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*sub2apipkg.UsageRecord, 0, len(records))
	for _, record := range records {
		result = append(result, toUsageRecord(record))
	}
	return result, total, nil
}

func toUsageRecord(record *gen.Sub2APIUsageRecord) *sub2apipkg.UsageRecord {
	return &sub2apipkg.UsageRecord{
		ID:              record.ID,
		RequestTime:     record.RequestTime,
		RequestDate:     record.RequestDate,
		Model:           record.Model,
		Endpoint:        record.Endpoint,
		UserAgent:       record.UserAgent,
		Status:          record.Status,
		Success:         record.Success,
		LatencyMS:       record.LatencyMs,
		TokenCount:      record.TokenCount,
		OutputTokens:    record.OutputTokens,
		TPS:             record.Tps,
		Cost:            record.Cost,
		FirstTokenMS:    record.FirstTokenMs,
		ReasoningEffort: record.ReasoningEffort,
		AccountName:     record.AccountName,
		ErrorMessage:    record.ErrorMessage,
		HTTPStatus:      record.HTTPStatus,
	}
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
