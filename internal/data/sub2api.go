package data

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/predicate"
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
				SetBucket15m(record.RequestTime.Truncate(15 * time.Minute).Unix()).
				SetModel(record.Model).
				SetEndpoint(record.Endpoint).
				SetGroupName(groupName).
				SetUserName(strings.TrimSpace(record.UserName)).
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
			if record.UserID > 0 {
				b = b.SetUserID(record.UserID)
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

// ---------- DB 侧聚合：统计全部由 ent 计算，不落内存遍历 ----------

// windowQuery 构造带时间窗口 + 公开过滤的基础查询。
func (r *sub2APIRepo) windowQuery(w sub2apipkg.StatsWindow) *gen.Sub2APIUsageRecordQuery {
	q := r.data.db.Sub2APIUsageRecord.Query()
	if w.Start != nil {
		q = q.Where(sub2apiusagerecord.RequestTimeGTE(*w.Start))
	}
	if w.End != nil {
		q = q.Where(sub2apiusagerecord.RequestTimeLTE(*w.End))
	}
	if w.PublicOnly {
		q = q.Where(sub2apiusagerecord.HasGroupWith(sub2apigroup.PublicEnabledEQ(true)))
	}
	return q
}

// rateEligiblePred 计费请求谓词：成功 或 上游错误。
func rateEligiblePred() predicate.Sub2APIUsageRecord {
	return sub2apiusagerecord.Or(
		sub2apiusagerecord.Success(true),
		sub2apiusagerecord.StatusEQ(sub2apipkg.StatusUpstreamError),
	)
}

// tpsEligiblePred 生成速度达标谓词：tps>0 且 输出 token>=阈值（输出过短的速率无统计意义）。
func tpsEligiblePred() predicate.Sub2APIUsageRecord {
	return sub2apiusagerecord.And(
		sub2apiusagerecord.TpsGT(0),
		sub2apiusagerecord.OutputTokensGTE(sub2apipkg.MinTPSOutputTokens),
	)
}

// scalarFloat 取单个聚合标量（NULL/空结果按 0）。
func scalarFloat(ctx context.Context, sel *gen.Sub2APIUsageRecordSelect) (float64, error) {
	var rows []struct {
		V *float64 `json:"v"`
	}
	if err := sel.Scan(ctx, &rows); err != nil {
		return 0, err
	}
	if len(rows) == 0 || rows[0].V == nil {
		return 0, nil
	}
	return *rows[0].V, nil
}

// AggregateTotals 汇总窗口内标量指标（Count/Sum/Mean 全部 DB 侧计算）。
func (r *sub2APIRepo) AggregateTotals(ctx context.Context, w sub2apipkg.StatsWindow) (sub2apipkg.Totals, error) {
	var totals sub2apipkg.Totals

	total, err := r.windowQuery(w).Count(ctx)
	if err != nil {
		return totals, err
	}
	totals.TotalCount = int64(total)

	req, err := r.windowQuery(w).Where(rateEligiblePred()).Count(ctx)
	if err != nil {
		return totals, err
	}
	totals.RequestCount = int64(req)

	succ, err := r.windowQuery(w).Where(sub2apiusagerecord.Success(true)).Count(ctx)
	if err != nil {
		return totals, err
	}
	totals.SuccessCount = int64(succ)

	// Token 计入全部记录
	tokenSum, err := scalarFloat(ctx, r.windowQuery(w).Aggregate(gen.As(gen.Sum(sub2apiusagerecord.FieldTokenCount), "v")))
	if err != nil {
		return totals, err
	}
	totals.TokenCount = int64(tokenSum)

	// 平均延迟仅统计成功请求
	totals.AverageLatencyMS, err = scalarFloat(ctx,
		r.windowQuery(w).Where(sub2apiusagerecord.Success(true)).Aggregate(gen.As(gen.Mean(sub2apiusagerecord.FieldLatencyMs), "v")))
	if err != nil {
		return totals, err
	}

	// 平均生成速度仅统计达标请求
	totals.AverageTPS, err = scalarFloat(ctx,
		r.windowQuery(w).Where(tpsEligiblePred()).Aggregate(gen.As(gen.Mean(sub2apiusagerecord.FieldTps), "v")))
	if err != nil {
		return totals, err
	}
	return totals, nil
}

// groupColumn / groupNotEmptyPred 将维度映射到 ent 列与「非空」谓词。
func groupColumn(field sub2apipkg.GroupField) string {
	switch field {
	case sub2apipkg.GroupByGroup:
		return sub2apiusagerecord.FieldGroupName
	case sub2apipkg.GroupByUserAgent:
		return sub2apiusagerecord.FieldUserAgent
	default:
		return sub2apiusagerecord.FieldModel
	}
}

func groupNotEmptyPred(field sub2apipkg.GroupField) predicate.Sub2APIUsageRecord {
	switch field {
	case sub2apipkg.GroupByGroup:
		return sub2apiusagerecord.GroupNameNEQ("")
	case sub2apipkg.GroupByUserAgent:
		return sub2apiusagerecord.UserAgentNEQ("")
	default:
		return sub2apiusagerecord.ModelNEQ("")
	}
}

// groupRow 承载 GroupBy 结果：三种维度列各一（GroupBy 只选其一，其余留零），按维度取 key。
type groupRow struct {
	Model     string   `json:"model"`
	GroupName string   `json:"group_name"`
	UserAgent string   `json:"user_agent"`
	Cnt       int64    `json:"cnt"`
	Tok       *float64 `json:"tok"`
	Tps       *float64 `json:"tps"`
}

func (row groupRow) key(field sub2apipkg.GroupField) string {
	switch field {
	case sub2apipkg.GroupByGroup:
		return row.GroupName
	case sub2apipkg.GroupByUserAgent:
		return row.UserAgent
	default:
		return row.Model
	}
}

// TopItems 按维度分组的用量排行（DB 侧 GroupBy），按 token 降序；不同 WHERE 的分组结果按 key 合并。
func (r *sub2APIRepo) TopItems(ctx context.Context, w sub2apipkg.StatsWindow, field sub2apipkg.GroupField, limit int) ([]sub2apipkg.TopStat, error) {
	col := groupColumn(field)
	stats := map[string]*sub2apipkg.TopStat{}
	order := make([]string, 0)
	get := func(key string) *sub2apipkg.TopStat {
		if s := stats[key]; s != nil {
			return s
		}
		s := &sub2apipkg.TopStat{Name: key}
		stats[key] = s
		order = append(order, key)
		return s
	}

	// 计费请求：请求数 + token 之和
	var reqRows []groupRow
	if err := r.windowQuery(w).Where(rateEligiblePred(), groupNotEmptyPred(field)).
		GroupBy(col).
		Aggregate(gen.As(gen.Count(), "cnt"), gen.As(gen.Sum(sub2apiusagerecord.FieldTokenCount), "tok")).
		Scan(ctx, &reqRows); err != nil {
		return nil, err
	}
	for _, row := range reqRows {
		s := get(row.key(field))
		s.Request = row.Cnt
		if row.Tok != nil {
			s.Token = int64(*row.Tok)
		}
	}

	// 成功数
	var succRows []groupRow
	if err := r.windowQuery(w).Where(sub2apiusagerecord.Success(true), groupNotEmptyPred(field)).
		GroupBy(col).
		Aggregate(gen.As(gen.Count(), "cnt")).
		Scan(ctx, &succRows); err != nil {
		return nil, err
	}
	for _, row := range succRows {
		if s := stats[row.key(field)]; s != nil {
			s.Success = row.Cnt
		}
	}

	// 达标请求平均生成速度
	var tpsRows []groupRow
	if err := r.windowQuery(w).Where(tpsEligiblePred(), groupNotEmptyPred(field)).
		GroupBy(col).
		Aggregate(gen.As(gen.Mean(sub2apiusagerecord.FieldTps), "tps")).
		Scan(ctx, &tpsRows); err != nil {
		return nil, err
	}
	for _, row := range tpsRows {
		if s := stats[row.key(field)]; s != nil && row.Tps != nil {
			s.AvgTPS = *row.Tps
		}
	}

	// 仅保留有计费请求的分组，按 token 降序（并列比请求数、名称）
	result := make([]sub2apipkg.TopStat, 0, len(order))
	for _, key := range order {
		if s := stats[key]; s != nil && s.Request > 0 {
			result = append(result, *s)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Token != result[j].Token {
			return result[i].Token > result[j].Token
		}
		if result[i].Request != result[j].Request {
			return result[i].Request > result[j].Request
		}
		return result[i].Name < result[j].Name
	})
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// dayRow 承载按 request_date 分组的结果。
type dayRow struct {
	Date string   `json:"request_date"`
	Cnt  int64    `json:"cnt"`
	Tok  *float64 `json:"tok"`
	Lat  *float64 `json:"lat"`
}

// DailyTrend 按自然日聚合（DB 侧 GroupBy(request_date)），仅返回有数据的日期（缺口由上层补齐）。
func (r *sub2APIRepo) DailyTrend(ctx context.Context, w sub2apipkg.StatsWindow) ([]sub2apipkg.DayStat, error) {
	stats := map[string]*sub2apipkg.DayStat{}
	order := make([]string, 0)
	get := func(date string) *sub2apipkg.DayStat {
		if s := stats[date]; s != nil {
			return s
		}
		s := &sub2apipkg.DayStat{Date: date}
		stats[date] = s
		order = append(order, date)
		return s
	}

	var reqRows []dayRow
	if err := r.windowQuery(w).Where(rateEligiblePred()).
		GroupBy(sub2apiusagerecord.FieldRequestDate).
		Aggregate(gen.As(gen.Count(), "cnt"), gen.As(gen.Sum(sub2apiusagerecord.FieldTokenCount), "tok")).
		Scan(ctx, &reqRows); err != nil {
		return nil, err
	}
	for _, row := range reqRows {
		s := get(row.Date)
		s.Request = row.Cnt
		if row.Tok != nil {
			s.Token = int64(*row.Tok)
		}
	}

	var succRows []dayRow
	if err := r.windowQuery(w).Where(sub2apiusagerecord.Success(true)).
		GroupBy(sub2apiusagerecord.FieldRequestDate).
		Aggregate(gen.As(gen.Count(), "cnt"), gen.As(gen.Mean(sub2apiusagerecord.FieldLatencyMs), "lat")).
		Scan(ctx, &succRows); err != nil {
		return nil, err
	}
	for _, row := range succRows {
		s := get(row.Date)
		s.Success = row.Cnt
		if row.Lat != nil {
			s.AvgLatencyMS = *row.Lat
		}
	}

	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	result := make([]sub2apipkg.DayStat, 0, len(order))
	for _, date := range order {
		result = append(result, *stats[date])
	}
	return result, nil
}

// bucketRow 承载按 bucket15m 分组的结果。
type bucketRow struct {
	Bucket int64    `json:"bucket15m"`
	Cnt    int64    `json:"cnt"`
	Tok    *float64 `json:"tok"`
	Lat    *float64 `json:"lat"`
	Tps    *float64 `json:"tps"`
}

// IntradaySeries 按 15 分钟桶聚合（DB 侧 GroupBy(bucket15m)），按桶升序。
func (r *sub2APIRepo) IntradaySeries(ctx context.Context, w sub2apipkg.StatsWindow) ([]sub2apipkg.BucketStat, error) {
	stats := map[int64]*sub2apipkg.BucketStat{}
	order := make([]int64, 0)
	get := func(bucket int64) *sub2apipkg.BucketStat {
		if s := stats[bucket]; s != nil {
			return s
		}
		s := &sub2apipkg.BucketStat{Bucket: time.Unix(bucket, 0)}
		stats[bucket] = s
		order = append(order, bucket)
		return s
	}

	// 全部记录数
	var totalRows []bucketRow
	if err := r.windowQuery(w).
		GroupBy(sub2apiusagerecord.FieldBucket15m).
		Aggregate(gen.As(gen.Count(), "cnt")).
		Scan(ctx, &totalRows); err != nil {
		return nil, err
	}
	for _, row := range totalRows {
		get(row.Bucket).Total = row.Cnt
	}

	// 计费请求 + token
	var reqRows []bucketRow
	if err := r.windowQuery(w).Where(rateEligiblePred()).
		GroupBy(sub2apiusagerecord.FieldBucket15m).
		Aggregate(gen.As(gen.Count(), "cnt"), gen.As(gen.Sum(sub2apiusagerecord.FieldTokenCount), "tok")).
		Scan(ctx, &reqRows); err != nil {
		return nil, err
	}
	for _, row := range reqRows {
		s := get(row.Bucket)
		s.Request = row.Cnt
		if row.Tok != nil {
			s.Token = int64(*row.Tok)
		}
	}

	// 成功数 + 平均延迟
	var succRows []bucketRow
	if err := r.windowQuery(w).Where(sub2apiusagerecord.Success(true)).
		GroupBy(sub2apiusagerecord.FieldBucket15m).
		Aggregate(gen.As(gen.Count(), "cnt"), gen.As(gen.Mean(sub2apiusagerecord.FieldLatencyMs), "lat")).
		Scan(ctx, &succRows); err != nil {
		return nil, err
	}
	for _, row := range succRows {
		s := get(row.Bucket)
		s.Success = row.Cnt
		if row.Lat != nil {
			s.AvgLatencyMS = *row.Lat
		}
	}

	// 达标请求平均生成速度
	var tpsRows []bucketRow
	if err := r.windowQuery(w).Where(tpsEligiblePred()).
		GroupBy(sub2apiusagerecord.FieldBucket15m).
		Aggregate(gen.As(gen.Mean(sub2apiusagerecord.FieldTps), "tps")).
		Scan(ctx, &tpsRows); err != nil {
		return nil, err
	}
	for _, row := range tpsRows {
		s := get(row.Bucket)
		if row.Tps != nil {
			s.AvgTPS = *row.Tps
		}
	}

	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	result := make([]sub2apipkg.BucketStat, 0, len(order))
	for _, bucket := range order {
		result = append(result, *stats[bucket])
	}
	return result, nil
}

// RecordsPage 按时间倒序分页读取 [start, end] 区间内的记录，支持多维度筛选。
func (r *sub2APIRepo) RecordsPage(ctx context.Context, start, end *time.Time, offset, limit int, publicOnly bool, filter sub2apipkg.RecordFilter) ([]*sub2apipkg.UsageRecord, int, error) {
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
	// 多维度筛选：指针非 nil 才参与过滤（区分「不筛选」与「筛选空值」）
	if filter.Model != nil {
		query = query.Where(sub2apiusagerecord.ModelEQ(*filter.Model))
	}
	if filter.GroupName != nil {
		query = query.Where(sub2apiusagerecord.GroupNameEQ(*filter.GroupName))
	}
	if filter.AccountName != nil {
		query = query.Where(sub2apiusagerecord.AccountNameContainsFold(*filter.AccountName))
	}
	if filter.Outcome != nil {
		switch *filter.Outcome {
		case "success":
			query = query.Where(sub2apiusagerecord.Success(true))
		case "failed":
			query = query.Where(sub2apiusagerecord.Success(false))
		}
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
	var userID int64
	if record.UserID != nil {
		userID = *record.UserID
	}
	return &sub2apipkg.UsageRecord{
		ID:              record.ID,
		RequestTime:     record.RequestTime,
		RequestDate:     record.RequestDate,
		Model:           record.Model,
		Endpoint:        record.Endpoint,
		GroupID:         groupID,
		GroupName:       record.GroupName,
		UserID:          userID,
		UserName:        record.UserName,
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
