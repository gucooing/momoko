package data

import (
	"context"
	"time"

	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/iqcheckprompt"
	"momoko/internal/data/ent/gen/iqcheckresult"
	"momoko/internal/data/ent/gen/iqchecksite"
	"momoko/internal/data/ent/schema"
)

type iqCheckRepo struct {
	data *Data
}

func NewIQCheckRepo(data *Data) biz.IQCheckRepo {
	return &iqCheckRepo{data: data}
}

// —— 提示词 ——

func (r *iqCheckRepo) ListPrompts(ctx context.Context, page, pageSize int64, keywords *string) ([]*gen.IQCheckPrompt, int64, error) {
	query := r.data.db.IQCheckPrompt.Query()
	if keywords != nil && *keywords != "" {
		query = query.Where(iqcheckprompt.TitleContains(*keywords))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := query.
		Order(gen.Asc(iqcheckprompt.FieldOrder), gen.Desc(iqcheckprompt.FieldCreateTime)).
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return rows, int64(total), nil
}

func (r *iqCheckRepo) ListEnabledPrompts(ctx context.Context) ([]*gen.IQCheckPrompt, error) {
	return r.data.db.IQCheckPrompt.Query().
		Where(iqcheckprompt.EnabledEQ(true)).
		Order(gen.Asc(iqcheckprompt.FieldOrder)).
		All(ctx)
}

func (r *iqCheckRepo) GetPrompt(ctx context.Context, id string) (*gen.IQCheckPrompt, error) {
	return r.data.db.IQCheckPrompt.Get(ctx, id)
}

func (r *iqCheckRepo) CreatePrompt(ctx context.Context, prompt *gen.IQCheckPrompt) (*gen.IQCheckPrompt, error) {
	return r.data.db.IQCheckPrompt.Create().
		SetID(prompt.ID).
		SetTitle(prompt.Title).
		SetContent(prompt.Content).
		SetEnabled(prompt.Enabled).
		SetOrder(prompt.Order).
		SetCategory(prompt.Category).
		SetGrading(prompt.Grading).
		Save(ctx)
}

func (r *iqCheckRepo) UpdatePrompt(ctx context.Context, prompt *gen.IQCheckPrompt) (*gen.IQCheckPrompt, error) {
	return r.data.db.IQCheckPrompt.UpdateOneID(prompt.ID).
		SetTitle(prompt.Title).
		SetContent(prompt.Content).
		SetEnabled(prompt.Enabled).
		SetOrder(prompt.Order).
		SetCategory(prompt.Category).
		SetGrading(prompt.Grading).
		Save(ctx)
}

func (r *iqCheckRepo) DeletePrompt(ctx context.Context, id string) error {
	return r.data.db.IQCheckPrompt.DeleteOneID(id).Exec(ctx)
}

// —— 站点 ——

func (r *iqCheckRepo) ListSites(ctx context.Context, page, pageSize int64, keywords *string) ([]*gen.IQCheckSite, int64, error) {
	query := r.data.db.IQCheckSite.Query()
	if keywords != nil && *keywords != "" {
		query = query.Where(iqchecksite.NameContains(*keywords))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := query.
		Order(gen.Desc(iqchecksite.FieldCreateTime)).
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return rows, int64(total), nil
}

func (r *iqCheckRepo) ListEnabledSites(ctx context.Context) ([]*gen.IQCheckSite, error) {
	return r.data.db.IQCheckSite.Query().
		Where(iqchecksite.EnabledEQ(true)).
		Order(gen.Asc(iqchecksite.FieldCreateTime)).
		All(ctx)
}

func (r *iqCheckRepo) GetSite(ctx context.Context, id string) (*gen.IQCheckSite, error) {
	return r.data.db.IQCheckSite.Get(ctx, id)
}

func (r *iqCheckRepo) GetSiteBySlug(ctx context.Context, slug string) (*gen.IQCheckSite, error) {
	return r.data.db.IQCheckSite.Query().
		Where(iqchecksite.SlugEQ(slug)).
		Only(ctx)
}

func siteModels(models []schema.IQCheckModel) []schema.IQCheckModel {
	if models == nil {
		return []schema.IQCheckModel{}
	}
	return models
}

func (r *iqCheckRepo) CreateSite(ctx context.Context, site *gen.IQCheckSite) (*gen.IQCheckSite, error) {
	return r.data.db.IQCheckSite.Create().
		SetID(site.ID).
		SetName(site.Name).
		SetEndpoint(site.Endpoint).
		SetAPIKey(site.APIKey).
		SetUserAgent(site.UserAgent).
		SetProtocol(site.Protocol).
		SetModels(siteModels(site.Models)).
		SetEnabled(site.Enabled).
		SetIntervalMinutes(site.IntervalMinutes).
		SetMaxRuns(site.MaxRuns).
		SetRunCount(0).
		SetSlug(site.Slug).
		Save(ctx)
}

func (r *iqCheckRepo) UpdateSite(ctx context.Context, site *gen.IQCheckSite) (*gen.IQCheckSite, error) {
	return r.data.db.IQCheckSite.UpdateOneID(site.ID).
		SetName(site.Name).
		SetEndpoint(site.Endpoint).
		SetAPIKey(site.APIKey).
		SetUserAgent(site.UserAgent).
		SetProtocol(site.Protocol).
		SetModels(siteModels(site.Models)).
		SetEnabled(site.Enabled).
		SetIntervalMinutes(site.IntervalMinutes).
		SetMaxRuns(site.MaxRuns).
		SetRunCount(site.RunCount).
		Save(ctx)
}

// DeleteSite 先删检测结果再删站点，不依赖数据库的外键级联。
func (r *iqCheckRepo) DeleteSite(ctx context.Context, id string) error {
	if _, err := r.data.db.IQCheckResult.Delete().Where(iqcheckresult.SiteIDEQ(id)).Exec(ctx); err != nil {
		return err
	}
	return r.data.db.IQCheckSite.DeleteOneID(id).Exec(ctx)
}

func (r *iqCheckRepo) MarkSiteChecked(ctx context.Context, id string, at time.Time, countRun bool) error {
	update := r.data.db.IQCheckSite.UpdateOneID(id).SetLastCheckAt(at)
	if countRun {
		update = update.AddRunCount(1)
	}
	return update.Exec(ctx)
}

// —— 检测结果 ——

func (r *iqCheckRepo) CreateResults(ctx context.Context, results []*gen.IQCheckResult) error {
	if len(results) == 0 {
		return nil
	}
	builders := make([]*gen.IQCheckResultCreate, 0, len(results))
	for _, row := range results {
		builders = append(builders, r.data.db.IQCheckResult.Create().
			SetID(row.ID).
			SetSiteID(row.SiteID).
			SetRoundID(row.RoundID).
			SetModel(row.Model).
			SetEffort(row.Effort).
			SetPromptID(row.PromptID).
			SetPromptTitle(row.PromptTitle).
			SetPromptContent(row.PromptContent).
			SetResponseText(row.ResponseText).
			SetStatus(row.Status).
			SetError(row.Error).
			SetDurationMs(row.DurationMs).
			SetInputTokens(row.InputTokens).
			SetOutputTokens(row.OutputTokens).
			SetRequestAt(row.RequestAt))
	}
	return r.data.db.IQCheckResult.CreateBulk(builders...).Exec(ctx)
}

func (r *iqCheckRepo) ListResults(ctx context.Context, siteID, model, resultID string, page, pageSize int64) ([]*gen.IQCheckResult, int64, error) {
	query := r.data.db.IQCheckResult.Query().Where(iqcheckresult.SiteIDEQ(siteID))
	if model != "" {
		query = query.Where(iqcheckresult.ModelEQ(model))
	}
	if resultID != "" {
		query = query.Where(iqcheckresult.IDEQ(resultID))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := query.
		Order(gen.Desc(iqcheckresult.FieldRequestAt), gen.Desc(iqcheckresult.FieldCreateTime)).
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return rows, int64(total), nil
}

// —— 统计 ——

// ListAllSites 返回全部站点（按名称/端点关键字过滤），用于在内存中按状态筛选与统计。
func (r *iqCheckRepo) ListAllSites(ctx context.Context, keywords string) ([]*gen.IQCheckSite, error) {
	query := r.data.db.IQCheckSite.Query()
	if keywords != "" {
		query = query.Where(iqchecksite.Or(
			iqchecksite.NameContains(keywords),
			iqchecksite.EndpointContains(keywords),
		))
	}
	return query.Order(gen.Desc(iqchecksite.FieldCreateTime)).All(ctx)
}

// RecentResults 返回站点最近的若干条检测结果（只取汇总需要的字段），最新的在前。
func (r *iqCheckRepo) RecentResults(ctx context.Context, siteID string, limit int) ([]*gen.IQCheckResult, error) {
	return r.data.db.IQCheckResult.Query().
		Where(iqcheckresult.SiteIDEQ(siteID)).
		Order(gen.Desc(iqcheckresult.FieldRequestAt), gen.Desc(iqcheckresult.FieldCreateTime)).
		Limit(limit).
		Select(
			iqcheckresult.FieldID,
			iqcheckresult.FieldRoundID,
			iqcheckresult.FieldStatus,
			iqcheckresult.FieldError,
			iqcheckresult.FieldDurationMs,
			iqcheckresult.FieldRequestAt,
			iqcheckresult.FieldModel,
			iqcheckresult.FieldEffort,
			iqcheckresult.FieldPromptID,
			iqcheckresult.FieldPromptTitle,
		).
		All(ctx)
}

// CountSiteResults 返回站点的检测结果总数。
func (r *iqCheckRepo) CountSiteResults(ctx context.Context, siteID string) (int64, error) {
	n, err := r.data.db.IQCheckResult.Query().Where(iqcheckresult.SiteIDEQ(siteID)).Count(ctx)
	return int64(n), err
}

// LastSuccessAt 返回站点最近一条成功结果的检测时间；从未成功时返回 nil。
func (r *iqCheckRepo) LastSuccessAt(ctx context.Context, siteID string) (*time.Time, error) {
	row, err := r.data.db.IQCheckResult.Query().
		Where(iqcheckresult.SiteIDEQ(siteID), iqcheckresult.StatusEQ(iqcheckresult.StatusSuccess)).
		Order(gen.Desc(iqcheckresult.FieldRequestAt)).
		Select(iqcheckresult.FieldID, iqcheckresult.FieldRequestAt).
		First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	at := row.RequestAt
	return &at, nil
}

// CountResultsSince 返回全部站点在 since 之后产生的检测结果条数。
func (r *iqCheckRepo) CountResultsSince(ctx context.Context, since time.Time) (int64, error) {
	n, err := r.data.db.IQCheckResult.Query().Where(iqcheckresult.RequestAtGTE(since)).Count(ctx)
	return int64(n), err
}

// CountPrompts 返回提示词总数与启用数。
func (r *iqCheckRepo) CountPrompts(ctx context.Context) (int64, int64, error) {
	all, err := r.data.db.IQCheckPrompt.Query().Count(ctx)
	if err != nil {
		return 0, 0, err
	}
	enabled, err := r.data.db.IQCheckPrompt.Query().Where(iqcheckprompt.EnabledEQ(true)).Count(ctx)
	if err != nil {
		return 0, 0, err
	}
	return int64(all), int64(enabled), nil
}

// PromptUsage 返回提示词被检测使用的次数、最近使用时间、涉及的站点数与平均输入 token 数。
func (r *iqCheckRepo) PromptUsage(ctx context.Context, promptID string) (biz.IQCheckPromptUsage, error) {
	var usage biz.IQCheckPromptUsage
	query := r.data.db.IQCheckResult.Query().Where(iqcheckresult.PromptIDEQ(promptID))
	count, err := query.Clone().Count(ctx)
	if err != nil || count == 0 {
		return usage, err
	}
	usage.Count = int64(count)
	latest, err := query.Clone().
		Order(gen.Desc(iqcheckresult.FieldRequestAt)).
		Select(iqcheckresult.FieldID, iqcheckresult.FieldRequestAt).
		First(ctx)
	if err != nil {
		return usage, err
	}
	at := latest.RequestAt
	usage.LastUsedAt = &at
	sites, err := query.Clone().Unique(true).Select(iqcheckresult.FieldSiteID).Strings(ctx)
	if err != nil {
		return usage, err
	}
	usage.SiteCount = int64(len(sites))
	// 没有用量数据时 AVG 为 NULL，用指针接收
	var tokens []struct {
		Avg *float64 `json:"avg"`
	}
	if err := query.Clone().
		Where(iqcheckresult.InputTokensGT(0)).
		Aggregate(gen.As(gen.Mean(iqcheckresult.FieldInputTokens), "avg")).
		Scan(ctx, &tokens); err != nil {
		return usage, err
	}
	if len(tokens) > 0 && tokens[0].Avg != nil {
		usage.AvgInputTokens = int64(*tokens[0].Avg + 0.5)
	}
	return usage, nil
}

// SiteResultStats 汇总站点检测结果：总数、成功数、成功平均耗时、since 之后的总数/成功数、按模型计数。
func (r *iqCheckRepo) SiteResultStats(ctx context.Context, siteID string, since time.Time) (biz.IQCheckSiteResultStats, error) {
	var stats biz.IQCheckSiteResultStats
	base := r.data.db.IQCheckResult.Query().Where(iqcheckresult.SiteIDEQ(siteID))
	total, err := base.Clone().Count(ctx)
	if err != nil || total == 0 {
		return stats, err
	}
	stats.Total = int64(total)
	success := base.Clone().Where(iqcheckresult.StatusEQ(iqcheckresult.StatusSuccess))
	okCount, err := success.Clone().Count(ctx)
	if err != nil {
		return stats, err
	}
	stats.Success = int64(okCount)
	if okCount > 0 {
		var durations []struct {
			Avg float64 `json:"avg"`
		}
		if err := success.Clone().Aggregate(gen.As(gen.Mean(iqcheckresult.FieldDurationMs), "avg")).Scan(ctx, &durations); err != nil {
			return stats, err
		}
		if len(durations) > 0 {
			stats.AvgDurationMs = int64(durations[0].Avg)
		}
	}
	recent := base.Clone().Where(iqcheckresult.RequestAtGTE(since))
	recentTotal, err := recent.Clone().Count(ctx)
	if err != nil {
		return stats, err
	}
	stats.RecentTotal = int64(recentTotal)
	recentOK, err := recent.Clone().Where(iqcheckresult.StatusEQ(iqcheckresult.StatusSuccess)).Count(ctx)
	if err != nil {
		return stats, err
	}
	stats.RecentSuccess = int64(recentOK)

	var models []struct {
		Model string `json:"model"`
		Count int64  `json:"count"`
	}
	if err := base.Clone().GroupBy(iqcheckresult.FieldModel).Aggregate(gen.Count()).Scan(ctx, &models); err != nil {
		return stats, err
	}
	for _, m := range models {
		stats.Models = append(stats.Models, biz.IQCheckModelCount{Model: m.Model, Count: m.Count})
	}
	return stats, nil
}
