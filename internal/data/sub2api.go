package data

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/sub2apiannouncement"
	"momoko/internal/data/ent/gen/sub2apigroup"
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
	// 与 usage 同批：按上游 group_id 直接 upsert 分组，再写记录。
	if err := r.upsertGroupsFromRecords(ctx, records); err != nil {
		return err
	}

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
			groupID := strings.TrimSpace(record.GroupID)
			groupName := strings.TrimSpace(record.GroupName)
			b := r.data.db.Sub2APIUsageRecord.Create().
				SetID(record.ID).
				SetRequestTime(record.RequestTime).
				SetRequestDate(requestDate).
				SetModel(record.Model).
				SetEndpoint(record.Endpoint).
				SetGroupName(groupName).
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
				SetHTTPStatus(record.HTTPStatus)
			if groupID != "" {
				b = b.SetGroupID(groupID)
			}
			builders = append(builders, b)
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

// upsertGroupsFromRecords 用上游 group_id 作主键直接入库。
// 新分组 public_enabled=false；已存在则更新 name（保留公开勾选）。
func (r *sub2APIRepo) upsertGroupsFromRecords(ctx context.Context, records []*sub2apipkg.UsageRecord) error {
	type gkey struct{ id, name string }
	seen := make(map[string]string) // id -> name
	for _, rec := range records {
		id := strings.TrimSpace(rec.GroupID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(rec.GroupName)
		if name == "" {
			name = id
		}
		if prev, ok := seen[id]; !ok || (prev == id && name != id) {
			seen[id] = name
		}
	}
	if len(seen) == 0 {
		return nil
	}
	builders := make([]*gen.Sub2APIGroupCreate, 0, len(seen))
	for id, name := range seen {
		builders = append(builders, r.data.db.Sub2APIGroup.Create().
			SetID(id).
			SetName(name).
			SetPublicEnabled(false),
		)
	}
	return r.data.db.Sub2APIGroup.CreateBulk(builders...).
		OnConflictColumns(sub2apigroup.FieldID).
		Update(func(u *gen.Sub2APIGroupUpsert) {
			u.UpdateName()
			u.SetDeleted(false)
			// 不覆盖 PublicEnabled
		}).
		Exec(ctx)
}

func (r *sub2APIRepo) ClearUsageRecords(ctx context.Context) error {
	// 全量重同步：清 usage，保留分组（以便保留 public_enabled 勾选）。
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

// RecordsSince 按时间升序读取 start（含）之后的记录。
// publicOnly 时仅返回关联分组 public_enabled=true 的记录（数据库过滤）。
func (r *sub2APIRepo) RecordsSince(ctx context.Context, start *time.Time, publicOnly bool) ([]*sub2apipkg.UsageRecord, error) {
	query := r.data.db.Sub2APIUsageRecord.Query()
	if start != nil {
		query = query.Where(sub2apiusagerecord.RequestTimeGTE(*start))
	}
	if publicOnly {
		query = query.Where(sub2apiusagerecord.HasGroupWith(sub2apigroup.PublicEnabledEQ(true)))
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

// RecordsPage 按时间倒序分页读取 [start, end] 区间内的记录。
func (r *sub2APIRepo) RecordsPage(ctx context.Context, start, end *time.Time, offset, limit int, publicOnly bool) ([]*sub2apipkg.UsageRecord, int, error) {
	query := r.data.db.Sub2APIUsageRecord.Query()
	if start != nil {
		query = query.Where(sub2apiusagerecord.RequestTimeGTE(*start))
	}
	if end != nil {
		query = query.Where(sub2apiusagerecord.RequestTimeLTE(*end))
	}
	if publicOnly {
		query = query.Where(sub2apiusagerecord.HasGroupWith(sub2apigroup.PublicEnabledEQ(true)))
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
	groupID := ""
	if record.GroupID != nil {
		groupID = *record.GroupID
	}
	return &sub2apipkg.UsageRecord{
		ID:              record.ID,
		RequestTime:     record.RequestTime,
		RequestDate:     record.RequestDate,
		Model:           record.Model,
		Endpoint:        record.Endpoint,
		GroupID:         groupID,
		GroupName:       record.GroupName,
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

func (r *sub2APIRepo) ListGroups(ctx context.Context) ([]*sub2apipkg.Group, error) {
	rows, err := r.data.db.Sub2APIGroup.Query().
		Order(gen.Asc(sub2apigroup.FieldDeleted), gen.Asc(sub2apigroup.FieldName)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*sub2apipkg.Group, 0, len(rows))
	for _, g := range rows {
		out = append(out, &sub2apipkg.Group{
			ID:            g.ID,
			Name:          g.Name,
			PublicEnabled: g.PublicEnabled,
			Deleted:       g.Deleted,
		})
	}
	return out, nil
}

// SyncGroups 以 upstream 存活列表为准刷新本地分组。
// live 中的 ID 标记 deleted=false 并更新 name；本地有但 live 没有的标记 deleted=true。
func (r *sub2APIRepo) SyncGroups(ctx context.Context, live []*sub2apipkg.Group) error {
	liveIDs := make(map[string]string, len(live)) // id -> name
	for _, g := range live {
		if g == nil {
			continue
		}
		id := strings.TrimSpace(g.ID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(g.Name)
		if name == "" {
			name = id
		}
		liveIDs[id] = name
	}

	// upsert live
	if len(liveIDs) > 0 {
		builders := make([]*gen.Sub2APIGroupCreate, 0, len(liveIDs))
		for id, name := range liveIDs {
			builders = append(builders, r.data.db.Sub2APIGroup.Create().
				SetID(id).
				SetName(name).
				SetDeleted(false).
				SetPublicEnabled(false),
			)
		}
		if err := r.data.db.Sub2APIGroup.CreateBulk(builders...).
			OnConflictColumns(sub2apigroup.FieldID).
			Update(func(u *gen.Sub2APIGroupUpsert) {
				u.UpdateName()
				u.SetDeleted(false)
			}).
			Exec(ctx); err != nil {
			return err
		}
	}

	// mark missing as deleted
	existing, err := r.data.db.Sub2APIGroup.Query().Select(sub2apigroup.FieldID).All(ctx)
	if err != nil {
		return err
	}
	stale := make([]string, 0)
	for _, g := range existing {
		if _, ok := liveIDs[g.ID]; !ok {
			stale = append(stale, g.ID)
		}
	}
	if len(stale) == 0 {
		return nil
	}
	_, err = r.data.db.Sub2APIGroup.Update().
		Where(sub2apigroup.IDIn(stale...)).
		SetDeleted(true).
		Save(ctx)
	return err
}

// SetPublicGroups names 为活跃分组 name；含 DeletedGroupPublicKey 时启用全部已删除分组。
func (r *sub2APIRepo) SetPublicGroups(ctx context.Context, names []string) error {
	enableDeleted := false
	seen := make(map[string]struct{}, len(names))
	active := make([]string, 0, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if n == sub2apipkg.DeletedGroupPublicKey {
			enableDeleted = true
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		active = append(active, n)
	}

	if _, err := r.data.db.Sub2APIGroup.Update().
		SetPublicEnabled(false).
		Save(ctx); err != nil {
		return err
	}
	if len(active) > 0 {
		if _, err := r.data.db.Sub2APIGroup.Update().
			Where(
				sub2apigroup.DeletedEQ(false),
				sub2apigroup.NameIn(active...),
			).
			SetPublicEnabled(true).
			Save(ctx); err != nil {
			return err
		}
	}
	if enableDeleted {
		if _, err := r.data.db.Sub2APIGroup.Update().
			Where(sub2apigroup.DeletedEQ(true)).
			SetPublicEnabled(true).
			Save(ctx); err != nil {
			return err
		}
	}
	return nil
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
