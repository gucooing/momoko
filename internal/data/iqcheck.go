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
		Save(ctx)
}

func (r *iqCheckRepo) UpdatePrompt(ctx context.Context, prompt *gen.IQCheckPrompt) (*gen.IQCheckPrompt, error) {
	return r.data.db.IQCheckPrompt.UpdateOneID(prompt.ID).
		SetTitle(prompt.Title).
		SetContent(prompt.Content).
		SetEnabled(prompt.Enabled).
		SetOrder(prompt.Order).
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

func (r *iqCheckRepo) ListResults(ctx context.Context, siteID, model string, page, pageSize int64) ([]*gen.IQCheckResult, int64, error) {
	query := r.data.db.IQCheckResult.Query().Where(iqcheckresult.SiteIDEQ(siteID))
	if model != "" {
		query = query.Where(iqcheckresult.ModelEQ(model))
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
