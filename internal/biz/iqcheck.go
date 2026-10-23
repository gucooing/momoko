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
	"momoko/internal/data/ent/gen/iqcheckresult"
	"momoko/internal/data/ent/gen/iqchecksite"
	"momoko/internal/data/ent/schema"
	"momoko/pkg/iqcheck"
	"momoko/pkg/task"
)

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
	ListResults(ctx context.Context, siteID, model string, page, pageSize int64) ([]*gen.IQCheckResult, int64, error)
}

const (
	iqCheckTickType         = "iqcheck.tick"
	iqCheckTickInterval     = time.Minute
	iqCheckDefaultInterval  = 60
	iqCheckSlugLength       = 12
	iqCheckSlugAttempts     = 5
	iqCheckModelConcurrency = 4
	iqCheckMaxPageSize      = 100
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

func (u *IQCheckUsecase) ListPrompts(ctx context.Context, req *v1.ListIQCheckPromptsRequest) ([]*v1.IQCheckPromptInfo, int64, error) {
	page, pageSize := iqCheckPage(req.Page, req.PageSize, 10)
	rows, total, err := u.repo.ListPrompts(ctx, page, pageSize, req.Keywords)
	if err != nil {
		return nil, 0, ErrSystem(err)
	}
	list := make([]*v1.IQCheckPromptInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, toIQCheckPromptInfo(row))
	}
	return list, total, nil
}

func (u *IQCheckUsecase) CreatePrompt(ctx context.Context, req *v1.CreateIQCheckPromptRequest) (*v1.IQCheckPromptInfo, error) {
	title, content := strings.TrimSpace(req.Title), strings.TrimSpace(req.Content)
	if title == "" {
		return nil, ErrIQCheckPromptTitle
	}
	if content == "" {
		return nil, ErrIQCheckPromptContent
	}
	row, err := u.repo.CreatePrompt(ctx, &gen.IQCheckPrompt{
		ID:      uuid.NewString(),
		Title:   title,
		Content: content,
		Enabled: req.Enabled,
		Order:   int(req.Order),
	})
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
	title, content := strings.TrimSpace(req.Title), strings.TrimSpace(req.Content)
	if title == "" {
		return nil, ErrIQCheckPromptTitle
	}
	if content == "" {
		return nil, ErrIQCheckPromptContent
	}
	row.Title, row.Content, row.Enabled, row.Order = title, content, req.Enabled, int(req.Order)
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

// pickPrompt 从启用的提示词中随机选一条。
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

func (u *IQCheckUsecase) ListSites(ctx context.Context, req *v1.ListIQCheckSitesRequest) ([]*v1.IQCheckSiteInfo, int64, error) {
	page, pageSize := iqCheckPage(req.Page, req.PageSize, 10)
	rows, total, err := u.repo.ListSites(ctx, page, pageSize, req.Keywords)
	if err != nil {
		return nil, 0, ErrSystem(err)
	}
	list := make([]*v1.IQCheckSiteInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, u.toIQCheckSiteInfo(row))
	}
	return list, total, nil
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
	info := &v1.PublicIQCheckSiteInfo{
		Name:            site.Name,
		Endpoint:        site.Endpoint,
		Protocol:        iqCheckProtocolToProto(site.Protocol),
		Models:          iqCheckModelsToProto(site.Models),
		IntervalMinutes: int32(site.IntervalMinutes),
	}
	if site.LastCheckAt != nil {
		info.LastCheckAt = timestamppb.New(*site.LastCheckAt)
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
	rows, total, err := u.repo.ListResults(ctx, site.ID, model, page, pageSize)
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

// runDueSites 对所有到期的启用站点并发执行一轮检测，等全部结束后返回。
func (u *IQCheckUsecase) runDueSites(ctx context.Context) error {
	sites, err := u.repo.ListEnabledSites(ctx)
	if err != nil {
		log.Errorf("iqcheck: 读取站点列表失败: %v", err)
		return err
	}
	now := time.Now()
	var wg sync.WaitGroup
	for _, site := range sites {
		if !iqCheckSiteDue(site, now) || !u.tryLock(site.ID) {
			continue
		}
		wg.Add(1)
		go func(site *gen.IQCheckSite) {
			defer wg.Done()
			defer u.unlock(site.ID)
			u.checkSite(ctx, site, true)
		}(site)
	}
	wg.Wait()
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

// checkSite 对站点的全部模型各发一次同一条随机提示词，并把每个模型的结果入库。
func (u *IQCheckUsecase) checkSite(ctx context.Context, site *gen.IQCheckSite, countRun bool) {
	prompt, err := u.pickPrompt(ctx)
	if err != nil {
		log.Warnf("iqcheck: 站点 %s 跳过检测: %v", site.Name, err)
		return
	}
	start := time.Now()
	if err := u.repo.MarkSiteChecked(ctx, site.ID, start, countRun); err != nil {
		log.Errorf("iqcheck: 站点 %s 记录检测时间失败: %v", site.Name, err)
	}
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
	roundID := uuid.NewString()
	results := make([]*gen.IQCheckResult, len(site.Models))
	sem := make(chan struct{}, iqCheckModelConcurrency)
	var wg sync.WaitGroup
	for i, model := range site.Models {
		wg.Add(1)
		go func(i int, model schema.IQCheckModel) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = u.checkModel(ctx, client, site, prompt, roundID, model)
		}(i, model)
	}
	wg.Wait()
	if err := u.repo.CreateResults(ctx, results); err != nil {
		log.Errorf("iqcheck: 站点 %s 保存检测结果失败: %v", site.Name, err)
		return
	}
	log.Infof("iqcheck: 站点 %s 完成一轮检测，%d 个模型，耗时 %s", site.Name, len(results), time.Since(start).Round(time.Second))
}

// checkModel 对单个模型发送提示词并组装结果行；失败也产生一条 failed 结果。
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
	res, err := client.Complete(ctx, iqcheck.Request{Model: model.Model, Effort: model.Effort, Prompt: prompt.Content})
	row.DurationMs = time.Since(at).Milliseconds()
	if err != nil {
		row.Status = iqcheckresult.StatusFailed
		row.Error = err.Error()
		return row
	}
	row.ResponseText = res.Text
	row.InputTokens = res.InputTokens
	row.OutputTokens = res.OutputTokens
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

func toIQCheckPromptInfo(row *gen.IQCheckPrompt) *v1.IQCheckPromptInfo {
	return &v1.IQCheckPromptInfo{
		Id:         row.ID,
		Title:      row.Title,
		Content:    row.Content,
		Enabled:    row.Enabled,
		Order:      int32(row.Order),
		CreateTime: timestamppb.New(row.CreateTime),
		UpdateTime: timestamppb.New(row.UpdateTime),
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
