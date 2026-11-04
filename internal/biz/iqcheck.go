package biz

import (
	"context"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/iqcheckprompt"
	"momoko/internal/data/ent/gen/iqcheckresult"
	"momoko/internal/data/ent/gen/iqchecksite"
	"momoko/internal/data/ent/schema"
	"momoko/pkg/iqcheck"
	"momoko/pkg/response"
	"momoko/pkg/task"
)

// ErrIQCheckPromptGrading 评分方式说明过长。
var ErrIQCheckPromptGrading = response.BadRequest(500, "评分方式最多 64 个字符")

// IQCheckRepo 智商检测数据访问：提示词、站点与检测结果。
type IQCheckRepo interface {
	ListPrompts(ctx context.Context, page, pageSize int64, keywords *string) ([]*gen.IQCheckPrompt, int64, error)
	ListEnabledPrompts(ctx context.Context) ([]*gen.IQCheckPrompt, error)
	GetPrompt(ctx context.Context, id string) (*gen.IQCheckPrompt, error)
	CreatePrompt(ctx context.Context, prompt *gen.IQCheckPrompt) (*gen.IQCheckPrompt, error)
	UpdatePrompt(ctx context.Context, prompt *gen.IQCheckPrompt) (*gen.IQCheckPrompt, error)
	DeletePrompt(ctx context.Context, id string) error

	ListSites(ctx context.Context, page, pageSize int64, keywords *string) ([]*gen.IQCheckSite, int64, error)
	ListEnabledSites(ctx context.Context) ([]*gen.IQCheckSite, error)
	GetSite(ctx context.Context, id string) (*gen.IQCheckSite, error)
	GetSiteBySlug(ctx context.Context, slug string) (*gen.IQCheckSite, error)
	CreateSite(ctx context.Context, site *gen.IQCheckSite) (*gen.IQCheckSite, error)
	UpdateSite(ctx context.Context, site *gen.IQCheckSite) (*gen.IQCheckSite, error)
	DeleteSite(ctx context.Context, id string) error
	// MarkSiteChecked 记录一轮检测的开始时间；countRun 为 true 时累计定时检测次数。
	MarkSiteChecked(ctx context.Context, id string, at time.Time, countRun bool) error

	CreateResults(ctx context.Context, results []*gen.IQCheckResult) error
	// ListResults 按时间倒序分页列出站点结果；model/resultID 为空表示不筛选。
	ListResults(ctx context.Context, siteID, model, resultID string, page, pageSize int64) ([]*gen.IQCheckResult, int64, error)

	ListAllSites(ctx context.Context, keywords string) ([]*gen.IQCheckSite, error)
	RecentResults(ctx context.Context, siteID string, limit int) ([]*gen.IQCheckResult, error)
	CountSiteResults(ctx context.Context, siteID string) (int64, error)
	// LastSuccessAt 返回站点最近一条成功结果的检测时间；从未成功时返回 nil。
	LastSuccessAt(ctx context.Context, siteID string) (*time.Time, error)
	CountResultsSince(ctx context.Context, since time.Time) (int64, error)
	CountPrompts(ctx context.Context) (all int64, enabled int64, err error)
	PromptUsage(ctx context.Context, promptID string) (IQCheckPromptUsage, error)
	SiteResultStats(ctx context.Context, siteID string, since time.Time) (IQCheckSiteResultStats, error)
}

// IQCheckPromptUsage 提示词使用情况。
type IQCheckPromptUsage struct {
	Count          int64
	SiteCount      int64
	LastUsedAt     *time.Time
	AvgInputTokens int64
}

// IQCheckModelCount 按模型统计的结果条数。
type IQCheckModelCount struct {
	Model string
	Count int64
}

// IQCheckSiteResultStats 站点检测结果汇总。
type IQCheckSiteResultStats struct {
	Total         int64
	Success       int64
	AvgDurationMs int64
	RecentTotal   int64
	RecentSuccess int64
	Models        []IQCheckModelCount
}

const (
	iqCheckTickType         = "iqcheck.tick"
	iqCheckTickInterval     = time.Minute
	iqCheckDefaultInterval  = 60
	iqCheckSlugLength       = 12
	iqCheckSlugAttempts     = 5
	iqCheckModelConcurrency = 4
	iqCheckStoreTimeout     = 30 * time.Second
	iqCheckMaxPageSize      = 100
	iqCheckRecentRounds     = 8
	iqCheckGradingMaxLen    = 64
)

// IQCheckUsecase 智商检测：提示词与站点配置、定时/手动检测、公开页数据。
// 上游协议细节在 pkg/iqcheck，这里只做校验、调度与持久化。
type IQCheckUsecase struct {
	repo  IQCheckRepo
	tasks *task.Manager

	mu      sync.Mutex
	running map[string]struct{} // 正在检测中的站点 id，避免同一站点并发跑两轮
}

func NewIQCheckUsecase(repo IQCheckRepo, tasks *task.Manager) *IQCheckUsecase {
	uc := &IQCheckUsecase{repo: repo, tasks: tasks, running: make(map[string]struct{})}
	// 定时单例：每分钟扫描一次到期站点并执行检测，开机随任务管理器启动。
	_ = tasks.EnsureSingleton(context.Background(), &iqCheckTickTask{uc: uc})
	return uc
}

// —— 提示词 ——

func iqCheckPage(page, pageSize, defaultSize int64) (int64, int64) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultSize
	}
	if pageSize > iqCheckMaxPageSize {
		pageSize = iqCheckMaxPageSize
	}
	return page, pageSize
}

func (u *IQCheckUsecase) ListPrompts(ctx context.Context, req *v1.ListIQCheckPromptsRequest) (*v1.ListIQCheckPromptsResponse, error) {
	page, pageSize := iqCheckPage(req.Page, req.PageSize, 10)
	rows, total, err := u.repo.ListPrompts(ctx, page, pageSize, req.Keywords)
	if err != nil {
		return nil, ErrSystem(err)
	}
	list := make([]*v1.IQCheckPromptInfo, 0, len(rows))
	for _, row := range rows {
		info := toIQCheckPromptInfo(row)
		usage, err := u.repo.PromptUsage(ctx, row.ID)
		if err != nil {
			return nil, ErrSystem(err)
		}
		info.UseCount = usage.Count
		info.SiteCount = usage.SiteCount
		info.AvgInputTokens = usage.AvgInputTokens
		if usage.LastUsedAt != nil {
			info.LastUsedAt = timestamppb.New(*usage.LastUsedAt)
		}
		list = append(list, info)
	}
	all, enabled, err := u.repo.CountPrompts(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.ListIQCheckPromptsResponse{
		Page:         page,
		PageSize:     pageSize,
		Total:        total,
		Prompts:      list,
		EnabledCount: enabled,
		AllCount:     all,
	}, nil
}

// iqCheckPromptInput 创建/更新提示词共用的可编辑字段。
type iqCheckPromptInput struct {
	Title    string
	Content  string
	Enabled  bool
	Order    int32
	Category v1.IQCheckPromptCategory
	Grading  string
}

// applyIQCheckPromptInput 校验并写入提示词可编辑字段。
func applyIQCheckPromptInput(row *gen.IQCheckPrompt, in iqCheckPromptInput) error {
	title, content, grading := strings.TrimSpace(in.Title), strings.TrimSpace(in.Content), strings.TrimSpace(in.Grading)
	if title == "" {
		return ErrIQCheckPromptTitle
	}
	if content == "" {
		return ErrIQCheckPromptContent
	}
	if len([]rune(grading)) > iqCheckGradingMaxLen {
		return ErrIQCheckPromptGrading
	}
	row.Title, row.Content, row.Enabled, row.Order = title, content, in.Enabled, int(in.Order)
	row.Category = iqCheckCategoryFromProto(in.Category)
	row.Grading = grading
	return nil
}

func (u *IQCheckUsecase) CreatePrompt(ctx context.Context, req *v1.CreateIQCheckPromptRequest) (*v1.IQCheckPromptInfo, error) {
	prompt := &gen.IQCheckPrompt{ID: uuid.NewString()}
	if err := applyIQCheckPromptInput(prompt, iqCheckPromptInput{
		Title: req.Title, Content: req.Content, Enabled: req.Enabled, Order: req.Order, Category: req.Category, Grading: req.Grading,
	}); err != nil {
		return nil, err
	}
	row, err := u.repo.CreatePrompt(ctx, prompt)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toIQCheckPromptInfo(row), nil
}

func (u *IQCheckUsecase) UpdatePrompt(ctx context.Context, req *v1.UpdateIQCheckPromptRequest) (*v1.IQCheckPromptInfo, error) {
	row, err := u.repo.GetPrompt(ctx, req.Id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrIQCheckPromptNotFound
		}
		return nil, ErrSystem(err)
	}
	if err := applyIQCheckPromptInput(row, iqCheckPromptInput{
		Title: req.Title, Content: req.Content, Enabled: req.Enabled, Order: req.Order, Category: req.Category, Grading: req.Grading,
	}); err != nil {
		return nil, err
	}
	updated, err := u.repo.UpdatePrompt(ctx, row)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toIQCheckPromptInfo(updated), nil
}

func (u *IQCheckUsecase) DeletePrompt(ctx context.Context, id string) error {
	if _, err := u.repo.GetPrompt(ctx, id); err != nil {
		if gen.IsNotFound(err) {
			return ErrIQCheckPromptNotFound
		}
		return ErrSystem(err)
	}
	if err := u.repo.DeletePrompt(ctx, id); err != nil {
		return ErrSystem(err)
	}
	return nil
}

// pickPrompt 从启用的提示词中随机选一条；没有启用的提示词时返回 ErrIQCheckNoPrompt。
func (u *IQCheckUsecase) pickPrompt(ctx context.Context) (*gen.IQCheckPrompt, error) {
	prompts, err := u.repo.ListEnabledPrompts(ctx)
	if err != nil {
		return nil, err
	}
	if len(prompts) == 0 {
		return nil, ErrIQCheckNoPrompt
	}
	return prompts[rand.IntN(len(prompts))], nil
}

// —— 站点 ——

// iqCheckSiteInput 创建/更新站点共用的可编辑字段。
type iqCheckSiteInput struct {
	Name            string
	Endpoint        string
	APIKey          string
	UserAgent       string
	Protocol        v1.IQCheckProtocol
	Models          []*v1.IQCheckModelConfig
	Enabled         bool
	IntervalMinutes int32
	MaxRuns         int32
}

// applyIQCheckSiteInput 校验并写入站点可编辑字段；APIKey 为空时保留原值。
func applyIQCheckSiteInput(site *gen.IQCheckSite, in iqCheckSiteInput) error {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return ErrIQCheckSiteName
	}
	endpoint := strings.TrimSpace(in.Endpoint)
	if _, err := iqcheck.NormalizeEndpoint(endpoint); err != nil {
		return ErrIQCheckEndpoint(err)
	}
	protocol, ok := iqCheckProtocolFromProto(in.Protocol)
	if !ok {
		return ErrIQCheckSiteProtocol
	}
	models, err := normalizeIQCheckModels(in.Models)
	if err != nil {
		return err
	}
	site.Name = name
	site.Endpoint = endpoint
	if key := strings.TrimSpace(in.APIKey); key != "" {
		site.APIKey = key
	}
	site.UserAgent = strings.TrimSpace(in.UserAgent)
	if site.UserAgent == "" {
		site.UserAgent = iqcheck.DefaultUserAgent
	}
	site.Protocol = iqchecksite.Protocol(protocol)
	site.Models = models
	site.Enabled = in.Enabled
	site.IntervalMinutes = int(in.IntervalMinutes)
	if site.IntervalMinutes <= 0 {
		site.IntervalMinutes = iqCheckDefaultInterval
	}
	site.MaxRuns = int(in.MaxRuns)
	if site.MaxRuns < 0 {
		site.MaxRuns = 0
	}
	return nil
}

// normalizeIQCheckModels 去掉空模型、按模型名去重并校验推理强度。
func normalizeIQCheckModels(models []*v1.IQCheckModelConfig) ([]schema.IQCheckModel, error) {
	out := make([]schema.IQCheckModel, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, m := range models {
		if m == nil {
			continue
		}
		name := strings.TrimSpace(m.Model)
		if name == "" {
			continue
		}
		effort := strings.TrimSpace(m.Effort)
		if !iqcheck.ValidEffort(effort) {
			return nil, ErrIQCheckEffort
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, schema.IQCheckModel{Model: name, Effort: effort})
	}
	return out, nil
}

// ListSites 站点数量有限：先取全部（按关键字过滤名称/端点/模型），补齐每个站点的最近轮次汇总，
// 再在内存中按状态筛选、统计并分页。
func (u *IQCheckUsecase) ListSites(ctx context.Context, req *v1.ListIQCheckSitesRequest) (*v1.ListIQCheckSitesResponse, error) {
	page, pageSize := iqCheckPage(req.Page, req.PageSize, 10)
	keywords := strings.TrimSpace(req.GetKeywords())
	rows, err := u.repo.ListAllSites(ctx, "")
	if err != nil {
		return nil, ErrSystem(err)
	}

	stats := &v1.IQCheckSiteStats{}
	modelSet := map[string]struct{}{}
	matched := make([]*v1.IQCheckSiteInfo, 0, len(rows))
	lowerKw := strings.ToLower(keywords)
	for _, row := range rows {
		info, err := u.siteInfoWithSummary(ctx, row)
		if err != nil {
			return nil, ErrSystem(err)
		}
		stats.Total++
		for _, m := range row.Models {
			modelSet[m.Model+"\x00"+m.Effort] = struct{}{}
		}
		if info.Checking {
			stats.Checking++
		}
		if !row.Enabled {
			stats.Disabled++
		}
		switch info.Health {
		case v1.IQCheckSiteHealth_IQ_CHECK_SITE_HEALTH_HEALTHY:
			stats.Healthy++
		case v1.IQCheckSiteHealth_IQ_CHECK_SITE_HEALTH_PARTIAL:
			stats.Partial++
		case v1.IQCheckSiteHealth_IQ_CHECK_SITE_HEALTH_FAILED:
			stats.Failed++
		default:
			stats.Never++
		}
		if lowerKw != "" && !iqCheckSiteMatches(row, lowerKw) {
			continue
		}
		if !iqCheckSiteFilterMatch(info, req.GetFilter()) {
			continue
		}
		matched = append(matched, info)
	}
	stats.ModelCount = int64(len(modelSet))
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if stats.ChecksToday, err = u.repo.CountResultsSince(ctx, today); err != nil {
		return nil, ErrSystem(err)
	}

	total := int64(len(matched))
	start := min((page-1)*pageSize, total)
	end := min(start+pageSize, total)
	return &v1.ListIQCheckSitesResponse{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
		Sites:    matched[start:end],
		Stats:    stats,
	}, nil
}

func iqCheckSiteMatches(site *gen.IQCheckSite, lowerKw string) bool {
	if strings.Contains(strings.ToLower(site.Name), lowerKw) || strings.Contains(strings.ToLower(site.Endpoint), lowerKw) {
		return true
	}
	for _, m := range site.Models {
		if strings.Contains(strings.ToLower(m.Model), lowerKw) {
			return true
		}
	}
	return false
}

func iqCheckSiteFilterMatch(info *v1.IQCheckSiteInfo, filter v1.IQCheckSiteFilter) bool {
	switch filter {
	case v1.IQCheckSiteFilter_IQ_CHECK_SITE_FILTER_HEALTHY:
		return info.Health == v1.IQCheckSiteHealth_IQ_CHECK_SITE_HEALTH_HEALTHY
	case v1.IQCheckSiteFilter_IQ_CHECK_SITE_FILTER_PARTIAL:
		return info.Health == v1.IQCheckSiteHealth_IQ_CHECK_SITE_HEALTH_PARTIAL
	case v1.IQCheckSiteFilter_IQ_CHECK_SITE_FILTER_FAILED:
		return info.Health == v1.IQCheckSiteHealth_IQ_CHECK_SITE_HEALTH_FAILED
	case v1.IQCheckSiteFilter_IQ_CHECK_SITE_FILTER_DISABLED:
		return !info.Enabled
	case v1.IQCheckSiteFilter_IQ_CHECK_SITE_FILTER_NEVER:
		return info.Health == v1.IQCheckSiteHealth_IQ_CHECK_SITE_HEALTH_UNSPECIFIED
	default:
		return true
	}
}

// iqCheckRoundResultLimit 估算覆盖最近 rounds 轮所需读取的结果条数（每轮 = 模型数，多留一轮余量）。
func iqCheckRoundResultLimit(site *gen.IQCheckSite, rounds int) int {
	return (max(len(site.Models), 1) + 1) * (rounds + 1)
}

// recentRounds 返回站点最近 rounds 轮检测汇总，旧的在前。
func (u *IQCheckUsecase) recentRounds(ctx context.Context, site *gen.IQCheckSite, rounds int) ([]*v1.IQCheckRoundSummary, error) {
	rows, err := u.repo.RecentResults(ctx, site.ID, iqCheckRoundResultLimit(site, rounds))
	if err != nil {
		return nil, err
	}
	return summarizeIQCheckRounds(rows, rounds), nil
}

// iqCheckRoundHealth 由一轮汇总得出健康状态。
func iqCheckRoundHealth(round *v1.IQCheckRoundSummary) v1.IQCheckSiteHealth {
	switch {
	case round == nil:
		return v1.IQCheckSiteHealth_IQ_CHECK_SITE_HEALTH_UNSPECIFIED
	case round.Failed == 0:
		return v1.IQCheckSiteHealth_IQ_CHECK_SITE_HEALTH_HEALTHY
	case round.Success == 0:
		return v1.IQCheckSiteHealth_IQ_CHECK_SITE_HEALTH_FAILED
	default:
		return v1.IQCheckSiteHealth_IQ_CHECK_SITE_HEALTH_PARTIAL
	}
}

// siteInfoWithSummary 在站点信息上补齐最近轮次汇总、结果总数、健康状态、最近成功时间与下次检测时间。
func (u *IQCheckUsecase) siteInfoWithSummary(ctx context.Context, site *gen.IQCheckSite) (*v1.IQCheckSiteInfo, error) {
	info := u.toIQCheckSiteInfo(site)
	rounds, err := u.recentRounds(ctx, site, iqCheckRecentRounds)
	if err != nil {
		return nil, err
	}
	info.RecentRounds = rounds
	if n := len(rounds); n > 0 {
		info.LastRound = rounds[n-1]
		info.Health = iqCheckRoundHealth(info.LastRound)
	}
	if info.ResultCount, err = u.repo.CountSiteResults(ctx, site.ID); err != nil {
		return nil, err
	}
	if info.ResultCount > 0 {
		at, err := u.repo.LastSuccessAt(ctx, site.ID)
		if err != nil {
			return nil, err
		}
		if at != nil {
			info.LastSuccessAt = timestamppb.New(*at)
		}
	}
	if next := iqCheckNextAt(site, time.Now()); next != nil {
		info.NextCheckAt = timestamppb.New(*next)
	}
	return info, nil
}

// summarizeIQCheckRounds 把最新在前的结果按轮次归并，返回最多 limit 轮，旧的在前；
// 同时统计每轮涉及的模型数（模型 + 推理强度），并带上本轮抽到的提示词标题。
func summarizeIQCheckRounds(rows []*gen.IQCheckResult, limit int) []*v1.IQCheckRoundSummary {
	var rounds []*v1.IQCheckRoundSummary
	index := map[string]*v1.IQCheckRoundSummary{}
	durations := map[string][2]int64{}
	models := map[string]map[string]struct{}{}
	for _, row := range rows {
		key := row.RoundID
		if key == "" {
			key = row.ID
		}
		round, ok := index[key]
		if !ok {
			if len(rounds) >= limit {
				continue
			}
			round = &v1.IQCheckRoundSummary{RoundId: key, RequestAt: timestamppb.New(row.RequestAt)}
			index[key] = round
			models[key] = map[string]struct{}{}
			rounds = append(rounds, round)
		}
		// 一轮内各结果的请求时间不同，取最早的作为本轮时间
		if row.RequestAt.Before(round.RequestAt.AsTime()) {
			round.RequestAt = timestamppb.New(row.RequestAt)
		}
		models[key][row.Model+"\x00"+row.Effort] = struct{}{}
		if round.PromptTitle == "" {
			round.PromptTitle = row.PromptTitle
		}
		if row.Status == iqcheckresult.StatusFailed {
			round.Failed++
			if round.Error == "" {
				round.Error = row.Error
			}
		} else {
			round.Success++
			d := durations[key]
			durations[key] = [2]int64{d[0] + row.DurationMs, d[1] + 1}
		}
	}
	for key, d := range durations {
		if d[1] > 0 {
			index[key].AvgDurationMs = d[0] / d[1]
		}
	}
	for key, round := range index {
		round.ModelCount = int32(len(models[key]))
	}
	// 最新在前 → 旧的在前
	for i, j := 0, len(rounds)-1; i < j; i, j = i+1, j-1 {
		rounds[i], rounds[j] = rounds[j], rounds[i]
	}
	return rounds
}

// iqCheckNextAt 估算下一次定时检测时间；未启用、无模型或已达次数上限时返回 nil。
func iqCheckNextAt(site *gen.IQCheckSite, now time.Time) *time.Time {
	if !site.Enabled || len(site.Models) == 0 {
		return nil
	}
	if site.MaxRuns > 0 && site.RunCount >= site.MaxRuns {
		return nil
	}
	if site.LastCheckAt == nil {
		next := now.Add(iqCheckTickInterval).Truncate(time.Minute)
		return &next
	}
	next := site.LastCheckAt.Add(time.Duration(max(site.IntervalMinutes, 1)) * time.Minute)
	if next.Before(now) {
		next = now.Add(iqCheckTickInterval).Truncate(time.Minute)
	}
	return &next
}

func (u *IQCheckUsecase) CreateSite(ctx context.Context, req *v1.CreateIQCheckSiteRequest) (*v1.IQCheckSiteInfo, error) {
	if strings.TrimSpace(req.ApiKey) == "" {
		return nil, ErrIQCheckSiteAPIKey
	}
	site := &gen.IQCheckSite{ID: uuid.NewString()}
	if err := applyIQCheckSiteInput(site, iqCheckSiteInput{
		Name:            req.Name,
		Endpoint:        req.Endpoint,
		APIKey:          req.ApiKey,
		UserAgent:       req.UserAgent,
		Protocol:        req.Protocol,
		Models:          req.Models,
		Enabled:         req.Enabled,
		IntervalMinutes: req.IntervalMinutes,
		MaxRuns:         req.MaxRuns,
	}); err != nil {
		return nil, err
	}
	// 公开页随机路径全局唯一，撞车时换一个重试。
	for attempt := 1; ; attempt++ {
		site.Slug = iqcheck.RandomSlug(iqCheckSlugLength)
		created, err := u.repo.CreateSite(ctx, site)
		if err == nil {
			return u.toIQCheckSiteInfo(created), nil
		}
		if !gen.IsConstraintError(err) || attempt >= iqCheckSlugAttempts {
			return nil, ErrSystem(err)
		}
	}
}

func (u *IQCheckUsecase) UpdateSite(ctx context.Context, req *v1.UpdateIQCheckSiteRequest) (*v1.IQCheckSiteInfo, error) {
	site, err := u.repo.GetSite(ctx, req.Id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrIQCheckSiteNotFound
		}
		return nil, ErrSystem(err)
	}
	if err := applyIQCheckSiteInput(site, iqCheckSiteInput{
		Name:            req.Name,
		Endpoint:        req.Endpoint,
		APIKey:          req.ApiKey,
		UserAgent:       req.UserAgent,
		Protocol:        req.Protocol,
		Models:          req.Models,
		Enabled:         req.Enabled,
		IntervalMinutes: req.IntervalMinutes,
		MaxRuns:         req.MaxRuns,
	}); err != nil {
		return nil, err
	}
	if req.ResetRunCount {
		site.RunCount = 0
	}
	updated, err := u.repo.UpdateSite(ctx, site)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return u.toIQCheckSiteInfo(updated), nil
}

func (u *IQCheckUsecase) DeleteSite(ctx context.Context, id string) error {
	if _, err := u.repo.GetSite(ctx, id); err != nil {
		if gen.IsNotFound(err) {
			return ErrIQCheckSiteNotFound
		}
		return ErrSystem(err)
	}
	if err := u.repo.DeleteSite(ctx, id); err != nil {
		return ErrSystem(err)
	}
	return nil
}

// FetchModels 拉取上游模型列表；编辑已有站点时未填 API Key 可复用已保存的值。
func (u *IQCheckUsecase) FetchModels(ctx context.Context, req *v1.FetchIQCheckModelsRequest) ([]string, error) {
	endpoint, apiKey, userAgent := strings.TrimSpace(req.Endpoint), strings.TrimSpace(req.ApiKey), strings.TrimSpace(req.UserAgent)
	protocol, ok := iqCheckProtocolFromProto(req.Protocol)
	if req.SiteId != "" && (apiKey == "" || endpoint == "" || !ok) {
		site, err := u.repo.GetSite(ctx, req.SiteId)
		if err != nil {
			if gen.IsNotFound(err) {
				return nil, ErrIQCheckSiteNotFound
			}
			return nil, ErrSystem(err)
		}
		if apiKey == "" {
			apiKey = site.APIKey
		}
		if endpoint == "" {
			endpoint = site.Endpoint
		}
		if userAgent == "" {
			userAgent = site.UserAgent
		}
		if !ok {
			protocol, ok = iqcheck.Protocol(site.Protocol), true
		}
	}
	if !ok {
		return nil, ErrIQCheckSiteProtocol
	}
	if apiKey == "" {
		return nil, ErrIQCheckSiteAPIKey
	}
	client, err := iqcheck.NewClient(iqcheck.Config{Endpoint: endpoint, APIKey: apiKey, Protocol: protocol, UserAgent: userAgent})
	if err != nil {
		return nil, ErrIQCheckEndpoint(err)
	}
	models, err := client.ListModels(ctx)
	if err != nil {
		return nil, ErrIQCheckUpstream(err)
	}
	return models, nil
}

// TriggerCheck 手动对站点执行一轮检测（后台异步）；不计入定时检测次数，但会刷新最近检测时间。
// 返回本轮的模型数。
func (u *IQCheckUsecase) TriggerCheck(ctx context.Context, id string) (int32, error) {
	site, err := u.repo.GetSite(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return 0, ErrIQCheckSiteNotFound
		}
		return 0, ErrSystem(err)
	}
	if len(site.Models) == 0 {
		return 0, ErrIQCheckSiteModels
	}
	if _, err := u.pickPrompt(ctx); err != nil {
		if err == ErrIQCheckNoPrompt {
			return 0, err
		}
		return 0, ErrSystem(err)
	}
	if !u.tryLock(site.ID) {
		return 0, ErrIQCheckRunning
	}
	go func() {
		defer u.unlock(site.ID)
		u.checkSite(context.Background(), site, false)
	}()
	return int32(len(site.Models)), nil
}

// —— 公开页 ——

func (u *IQCheckUsecase) publicSite(ctx context.Context, slug string) (*gen.IQCheckSite, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, ErrIQCheckPublicNotFound
	}
	site, err := u.repo.GetSiteBySlug(ctx, slug)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrIQCheckPublicNotFound
		}
		return nil, ErrSystem(err)
	}
	return site, nil
}

func (u *IQCheckUsecase) PublicSite(ctx context.Context, slug string) (*v1.PublicIQCheckSiteInfo, error) {
	site, err := u.publicSite(ctx, slug)
	if err != nil {
		return nil, err
	}
	// 公示页不返回 API Key 与接口地址
	info := &v1.PublicIQCheckSiteInfo{
		Name:            site.Name,
		Models:          iqCheckModelsToProto(site.Models),
		IntervalMinutes: int32(site.IntervalMinutes),
		Enabled:         site.Enabled,
	}
	if site.LastCheckAt != nil {
		info.LastCheckAt = timestamppb.New(*site.LastCheckAt)
	}
	rounds, err := u.recentRounds(ctx, site, 1)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if len(rounds) > 0 {
		info.LastRound = rounds[len(rounds)-1]
	}
	stats, err := u.repo.SiteResultStats(ctx, site.ID, time.Now().AddDate(0, 0, -7))
	if err != nil {
		return nil, ErrSystem(err)
	}
	info.Stats = &v1.PublicIQCheckStats{
		Total:         stats.Total,
		Success:       stats.Success,
		AvgDurationMs: stats.AvgDurationMs,
		Last7DTotal:   stats.RecentTotal,
		Last7DSuccess: stats.RecentSuccess,
	}
	for _, m := range stats.Models {
		info.Stats.Models = append(info.Stats.Models, &v1.PublicIQCheckModelCount{Model: m.Model, Count: m.Count})
	}
	return info, nil
}

func (u *IQCheckUsecase) PublicResults(ctx context.Context, req *v1.PublicIQCheckResultsRequest) ([]*v1.IQCheckResultInfo, int64, error) {
	site, err := u.publicSite(ctx, req.Slug)
	if err != nil {
		return nil, 0, err
	}
	page, pageSize := iqCheckPage(req.Page, req.PageSize, 20)
	model := ""
	if req.Model != nil {
		model = strings.TrimSpace(*req.Model)
	}
	resultID := ""
	if req.ResultId != nil {
		resultID = strings.TrimSpace(*req.ResultId)
	}
	rows, total, err := u.repo.ListResults(ctx, site.ID, model, resultID, page, pageSize)
	if err != nil {
		return nil, 0, ErrSystem(err)
	}
	list := make([]*v1.IQCheckResultInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, toIQCheckResultInfo(row))
	}
	return list, total, nil
}

// —— 检测执行 ——

// iqCheckTickTask 每分钟扫描一次到期站点的定时单例任务。
type iqCheckTickTask struct {
	uc *IQCheckUsecase
}

func (t *iqCheckTickTask) Meta() task.Meta {
	return task.Meta{
		ID:       iqCheckTickType,
		Type:     iqCheckTickType,
		Kind:     task.KindScheduled,
		Resume:   task.ResumeAlways,
		Interval: iqCheckTickInterval,
		Title:    "智商检测定时任务",
	}
}

func (t *iqCheckTickTask) Payload() any { return nil }

func (t *iqCheckTickTask) Run(ctx context.Context, _ task.Reporter) error {
	return t.uc.runDueSites(ctx)
}

// runDueSites 对到期的启用站点各起一个协程执行检测后立即返回，不阻塞定时任务；
// 同一站点由 running 锁保证不会并发跑两轮。
func (u *IQCheckUsecase) runDueSites(ctx context.Context) error {
	sites, err := u.repo.ListEnabledSites(ctx)
	if err != nil {
		log.Errorf("iqcheck: 读取站点列表失败: %v", err)
		return err
	}
	now := time.Now()
	for _, site := range sites {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !iqCheckSiteDue(site, now) || !u.tryLock(site.ID) {
			continue
		}
		go func(site *gen.IQCheckSite) {
			defer u.unlock(site.ID)
			u.checkSite(ctx, site, true)
		}(site)
	}
	return nil
}

// iqCheckSiteDue 报告站点是否到了下一次定时检测时间。
func iqCheckSiteDue(site *gen.IQCheckSite, now time.Time) bool {
	if len(site.Models) == 0 {
		return false
	}
	if site.MaxRuns > 0 && site.RunCount >= site.MaxRuns {
		return false
	}
	if site.LastCheckAt == nil {
		return true
	}
	interval := time.Duration(max(site.IntervalMinutes, 1)) * time.Minute
	return !now.Before(site.LastCheckAt.Add(interval))
}

// checkSite 从启用的提示词中随机抽一道题发给站点的每个模型；每次请求独立超时、完成即入库，
// 慢的或挂住的请求不影响其他结果的展示。
func (u *IQCheckUsecase) checkSite(ctx context.Context, site *gen.IQCheckSite, countRun bool) {
	setupCtx, cancelSetup := context.WithTimeout(ctx, iqCheckStoreTimeout)
	defer cancelSetup()
	prompt, err := u.pickPrompt(setupCtx)
	if err != nil {
		log.Warnf("iqcheck: 站点 %s 跳过检测: %v", site.Name, err)
		return
	}
	start := time.Now()
	client, err := iqcheck.NewClient(iqcheck.Config{
		Endpoint:  site.Endpoint,
		APIKey:    site.APIKey,
		Protocol:  iqcheck.Protocol(site.Protocol),
		UserAgent: site.UserAgent,
	})
	if err != nil {
		log.Errorf("iqcheck: 站点 %s 配置无效: %v", site.Name, err)
		return
	}
	if err := u.repo.MarkSiteChecked(setupCtx, site.ID, start, countRun); err != nil {
		log.Errorf("iqcheck: 站点 %s 记录检测时间失败: %v", site.Name, err)
		return
	}
	cancelSetup()
	roundID := uuid.NewString()
	// 固定数量的 worker，避免为每个排队请求启动一个等待信号量的协程。
	total := len(site.Models)
	jobs := make(chan schema.IQCheckModel, total)
	for _, model := range site.Models {
		jobs <- model
	}
	close(jobs)
	results := make(chan *gen.IQCheckResult, iqCheckModelConcurrency)
	var wg sync.WaitGroup
	for range min(iqCheckModelConcurrency, total) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for model := range jobs {
				// Complete 自带单次总超时；取消后排队请求也产生失败结果，不再发起请求。
				results <- u.checkModel(ctx, client, site, prompt, roundID, model)
			}
		}()
	}
	go func() { wg.Wait(); close(results) }()
	for row := range results {
		// 每条结果独立限时落库，服务取消时仍尝试保存已收到的内容。
		saveCtx, cancelSave := context.WithTimeout(context.WithoutCancel(ctx), iqCheckStoreTimeout)
		err := u.repo.CreateResults(saveCtx, []*gen.IQCheckResult{row})
		cancelSave()
		if err != nil {
			log.Errorf("iqcheck: 站点 %s 模型 %s 保存检测结果失败: %v", site.Name, row.Model, err)
		}
	}
	log.Infof("iqcheck: 站点 %s 完成一轮检测，%d 个模型，题目「%s」，耗时 %s", site.Name, len(site.Models), prompt.Title, time.Since(start).Round(time.Second))
}

// checkModel 对单个模型发送一道题并组装结果行；失败也产生一条 failed 结果。
func (u *IQCheckUsecase) checkModel(ctx context.Context, client *iqcheck.Client, site *gen.IQCheckSite, prompt *gen.IQCheckPrompt, roundID string, model schema.IQCheckModel) *gen.IQCheckResult {
	at := time.Now()
	row := &gen.IQCheckResult{
		ID:            uuid.NewString(),
		SiteID:        site.ID,
		RoundID:       roundID,
		Model:         model.Model,
		Effort:        model.Effort,
		PromptID:      prompt.ID,
		PromptTitle:   prompt.Title,
		PromptContent: prompt.Content,
		Status:        iqcheckresult.StatusSuccess,
		RequestAt:     at,
	}
	if err := ctx.Err(); err != nil {
		row.Status = iqcheckresult.StatusFailed
		row.Error = err.Error()
		return row
	}
	res, err := client.Complete(ctx, iqcheck.Request{Model: model.Model, Effort: model.Effort, Prompt: prompt.Content})
	row.DurationMs = time.Since(at).Milliseconds()
	if res != nil {
		row.ResponseText = res.Text
		row.InputTokens = res.InputTokens
		row.OutputTokens = res.OutputTokens
	}
	if err != nil {
		row.Status = iqcheckresult.StatusFailed
		row.Error = err.Error()
	}
	return row
}

func (u *IQCheckUsecase) tryLock(siteID string) bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	if _, ok := u.running[siteID]; ok {
		return false
	}
	u.running[siteID] = struct{}{}
	return true
}

func (u *IQCheckUsecase) unlock(siteID string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	delete(u.running, siteID)
}

func (u *IQCheckUsecase) isRunning(siteID string) bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	_, ok := u.running[siteID]
	return ok
}

// —— 转换 ——

func iqCheckProtocolFromProto(p v1.IQCheckProtocol) (iqcheck.Protocol, bool) {
	switch p {
	case v1.IQCheckProtocol_IQ_CHECK_PROTOCOL_CHAT:
		return iqcheck.ProtocolChat, true
	case v1.IQCheckProtocol_IQ_CHECK_PROTOCOL_RESPONSES:
		return iqcheck.ProtocolResponses, true
	case v1.IQCheckProtocol_IQ_CHECK_PROTOCOL_MESSAGES:
		return iqcheck.ProtocolMessages, true
	}
	return "", false
}

func iqCheckProtocolToProto(p iqchecksite.Protocol) v1.IQCheckProtocol {
	switch iqcheck.Protocol(p) {
	case iqcheck.ProtocolChat:
		return v1.IQCheckProtocol_IQ_CHECK_PROTOCOL_CHAT
	case iqcheck.ProtocolResponses:
		return v1.IQCheckProtocol_IQ_CHECK_PROTOCOL_RESPONSES
	case iqcheck.ProtocolMessages:
		return v1.IQCheckProtocol_IQ_CHECK_PROTOCOL_MESSAGES
	}
	return v1.IQCheckProtocol_IQ_CHECK_PROTOCOL_UNSPECIFIED
}

func iqCheckModelsToProto(models []schema.IQCheckModel) []*v1.IQCheckModelConfig {
	out := make([]*v1.IQCheckModelConfig, 0, len(models))
	for _, m := range models {
		out = append(out, &v1.IQCheckModelConfig{Model: m.Model, Effort: m.Effort})
	}
	return out
}

var iqCheckCategories = map[v1.IQCheckPromptCategory]iqcheckprompt.Category{
	v1.IQCheckPromptCategory_IQ_CHECK_PROMPT_CATEGORY_OTHER:   iqcheckprompt.CategoryOther,
	v1.IQCheckPromptCategory_IQ_CHECK_PROMPT_CATEGORY_LOGIC:   iqcheckprompt.CategoryLogic,
	v1.IQCheckPromptCategory_IQ_CHECK_PROMPT_CATEGORY_MATH:    iqcheckprompt.CategoryMath,
	v1.IQCheckPromptCategory_IQ_CHECK_PROMPT_CATEGORY_CODING:  iqcheckprompt.CategoryCoding,
	v1.IQCheckPromptCategory_IQ_CHECK_PROMPT_CATEGORY_DRAWING: iqcheckprompt.CategoryDrawing,
	v1.IQCheckPromptCategory_IQ_CHECK_PROMPT_CATEGORY_WRITING: iqcheckprompt.CategoryWriting,
}

// iqCheckCategoryFromProto 未知分类按「其他」处理。
func iqCheckCategoryFromProto(c v1.IQCheckPromptCategory) iqcheckprompt.Category {
	if v, ok := iqCheckCategories[c]; ok {
		return v
	}
	return iqcheckprompt.CategoryOther
}

func iqCheckCategoryToProto(c iqcheckprompt.Category) v1.IQCheckPromptCategory {
	for k, v := range iqCheckCategories {
		if v == c {
			return k
		}
	}
	return v1.IQCheckPromptCategory_IQ_CHECK_PROMPT_CATEGORY_OTHER
}

func toIQCheckPromptInfo(row *gen.IQCheckPrompt) *v1.IQCheckPromptInfo {
	return &v1.IQCheckPromptInfo{
		Id:         row.ID,
		Title:      row.Title,
		Content:    row.Content,
		Enabled:    row.Enabled,
		Order:      int32(row.Order),
		CreateTime: timestamppb.New(row.CreateTime),
		UpdateTime: timestamppb.New(row.UpdateTime),
		Category:   iqCheckCategoryToProto(row.Category),
		Grading:    row.Grading,
	}
}

func (u *IQCheckUsecase) toIQCheckSiteInfo(site *gen.IQCheckSite) *v1.IQCheckSiteInfo {
	info := &v1.IQCheckSiteInfo{
		Id:              site.ID,
		Name:            site.Name,
		Endpoint:        site.Endpoint,
		ApiKey:          iqcheck.MaskAPIKey(site.APIKey),
		Protocol:        iqCheckProtocolToProto(site.Protocol),
		Models:          iqCheckModelsToProto(site.Models),
		Enabled:         site.Enabled,
		IntervalMinutes: int32(site.IntervalMinutes),
		MaxRuns:         int32(site.MaxRuns),
		RunCount:        int32(site.RunCount),
		Slug:            site.Slug,
		Checking:        u.isRunning(site.ID),
		CreateTime:      timestamppb.New(site.CreateTime),
		UpdateTime:      timestamppb.New(site.UpdateTime),
		UserAgent:       site.UserAgent,
	}
	if site.LastCheckAt != nil {
		info.LastCheckAt = timestamppb.New(*site.LastCheckAt)
	}
	return info
}

func toIQCheckResultInfo(row *gen.IQCheckResult) *v1.IQCheckResultInfo {
	status := v1.IQCheckResultStatus_IQ_CHECK_RESULT_STATUS_SUCCESS
	if row.Status == iqcheckresult.StatusFailed {
		status = v1.IQCheckResultStatus_IQ_CHECK_RESULT_STATUS_FAILED
	}
	return &v1.IQCheckResultInfo{
		Id:            row.ID,
		SiteId:        row.SiteID,
		RoundId:       row.RoundID,
		Model:         row.Model,
		Effort:        row.Effort,
		PromptTitle:   row.PromptTitle,
		PromptContent: row.PromptContent,
		ResponseText:  row.ResponseText,
		Status:        status,
		Error:         row.Error,
		DurationMs:    row.DurationMs,
		InputTokens:   row.InputTokens,
		OutputTokens:  row.OutputTokens,
		RequestAt:     timestamppb.New(row.RequestAt),
	}
}
