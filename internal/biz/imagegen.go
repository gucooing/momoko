package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/cache"
	"momoko/pkg/common"
	"momoko/pkg/imagestore"
	sub2apipkg "momoko/pkg/sub2api"
)

const (
	// PreImageGenImage 公开图片服务路径前缀（manual handler），用于对外暴露生成结果图。
	PreImageGenImage = "/api/v1/public/sub2api/imagine/images"
	// imageGenStaleAge pending 任务超过该时长视为僵死，读列表时惰性回收。
	imageGenStaleAge = 5 * time.Minute
	// tokenCacheTTL sub2api token 校验结果缓存时长。
	tokenCacheTTL = time.Minute
	// listGenerationsLimit 列表默认条数。
	listGenerationsLimit = 50

	// 生成模式。
	modeText2Image  = "text2image"
	modeImage2Image = "image2image"
)

// ImageGenRepo 数据层需实现的持久化接口。任务按 (user_id, src_host) 复合归属。
type ImageGenRepo interface {
	ListGenerations(ctx context.Context, userID, srcHost string, limit int) ([]*gen.ImageGenGeneration, error)
	CreateGeneration(ctx context.Context, req *v1.CreateImageGenGenerationRequest, userID, srcHost, mode string) (*gen.ImageGenGeneration, error)
	UpdateGenerationResult(ctx context.Context, id, userID, status, errMsg string, latencyMs int64, resultCount int, finishedAt *time.Time) error
	GetGeneration(ctx context.Context, id, userID, srcHost string) (*gen.ImageGenGeneration, error)
	DeleteGeneration(ctx context.Context, id, userID, srcHost string) error
	SweepStalePending(ctx context.Context, userID, srcHost string, before time.Time) error

	CreateImage(ctx context.Context, generationID, userID string, img *sub2apipkg.ImageResult) (*gen.ImageGenImage, error)
	ListImagesByGeneration(ctx context.Context, generationID string) ([]*gen.ImageGenImage, error)
	ListImagesByGenerationIDs(ctx context.Context, ids []string) ([]*gen.ImageGenImage, error)
	GetImage(ctx context.Context, id string) (*gen.ImageGenImage, error)
}

// imagineTokenContext 校验后的 token 上下文（服务端缓存）。
type imagineTokenContext struct {
	UserID  string // 可信，来自 /keys 响应
	SrcHost string
	Keys    []sub2apipkg.ImagineKey
}

// ImageGenUsecase 生图业务编排：token 校验缓存、站点白名单、apikey 代理、异步生图 worker。
type ImageGenUsecase struct {
	repo   ImageGenRepo
	config ConfigRepo
	client *sub2apipkg.ImagineClient
	cache  *cache.Cache[string, *imagineTokenContext]

	allowedMu  sync.RWMutex
	allowedRaw string              // 最近一次解析的配置原文，用于判断是否需要重解析
	allowedSet map[string]struct{} // 允许站点哈希集合，O(1) 查询
}

func NewImageGenUsecase(repo ImageGenRepo, config ConfigRepo) (*ImageGenUsecase, func(), error) {
	_ = imagestore.EnsureDir()
	uc := &ImageGenUsecase{
		repo:       repo,
		config:     config,
		client:     sub2apipkg.NewImagineClient(),
		cache:      cache.New[string, *imagineTokenContext](tokenCacheTTL),
		allowedSet: map[string]struct{}{},
	}
	return uc, func() {}, nil
}

// ---------- token 校验 ----------

func (u *ImageGenUsecase) resolveToken(ctx context.Context, token, srcHost string) (*imagineTokenContext, error) {
	token = strings.TrimSpace(token)
	srcHost = strings.TrimSpace(srcHost)
	if token == "" {
		return nil, ErrImageGenTokenInvalid
	}
	// 生图功能总开关
	enabled, err := getBoolConfig(ctx, u.config, common.ConfigSub2APIImageEnabled)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if !enabled {
		return nil, ErrImageGenDisabled
	}
	// 站点白名单校验（先于缓存，保证配置变更立即生效）
	if err := u.checkSrcHostAllowed(ctx, srcHost); err != nil {
		return nil, err
	}
	key := tokenHash(token)
	if tctx, ok := u.cache.Get(key); ok {
		return tctx, nil
	}
	keys, userID, err := u.client.ListKeys(ctx, srcHost, token)
	if err != nil {
		if errors.Is(err, sub2apipkg.ErrImagineTokenInvalid) {
			return nil, ErrImageGenTokenInvalid
		}
		if re, ok := errors.AsType[*sub2apipkg.ImagineRequestError](err); ok && (re.HTTPStatus == 401 || re.HTTPStatus == 403) {
			return nil, ErrImageGenTokenInvalid
		}
		return nil, ErrSystem(err)
	}
	tctx := &imagineTokenContext{UserID: userID, SrcHost: srcHost, Keys: keys}
	u.cache.Set(key, tctx)
	return tctx, nil
}

// checkSrcHostAllowed 校验 src_host 是否在配置的允许站点哈希集合内。
// 仅当 src_host_whitelist_enabled=true 时才校验；集合为空时拒绝所有（避免误开白名单却漏配）。
// 仅当配置原文变化时才重新解析为哈希集合，请求期为 O(1) 查询。
func (u *ImageGenUsecase) checkSrcHostAllowed(ctx context.Context, srcHost string) error {
	whitelistOn, err := getBoolConfig(ctx, u.config, common.ConfigSub2APISrcHostWhitelistEnabled)
	if err != nil {
		return ErrSystem(err)
	}
	if !whitelistOn {
		return nil // 未开启白名单校验，放行
	}
	raw, err := u.config.Get(ctx, common.ConfigSub2APIAllowedSrcHosts)
	if err != nil {
		return ErrSystem(err)
	}
	// 配置原文变化时重新解析为哈希集合
	u.allowedMu.RLock()
	if raw != u.allowedRaw {
		u.allowedMu.RUnlock()
		set, perr := sub2apipkg.ParseAllowedSrcHostsSet(raw)
		if perr != nil {
			return ErrSystem(perr)
		}
		u.allowedMu.Lock()
		u.allowedRaw = raw
		u.allowedSet = set
		u.allowedMu.Unlock()
	} else {
		u.allowedMu.RUnlock()
	}

	u.allowedMu.RLock()
	set := u.allowedSet
	u.allowedMu.RUnlock()
	host := strings.TrimRight(srcHost, "/")
	if _, ok := set[host]; ok {
		return nil
	}
	return ErrImageGenSrcHostForbidden
}

// getBoolConfig 读取布尔配置。
func getBoolConfig(ctx context.Context, config ConfigRepo, key common.ConfigKey) (bool, error) {
	raw, err := config.Get(ctx, key)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes", "on":
		return true, nil
	default:
		return false, nil
	}
}

// ---------- 代理：apikey / models ----------

func (u *ImageGenUsecase) ListApiKeys(ctx context.Context, token, srcHost string) ([]*v1.ImageGenApiKeySummary, error) {
	tctx, err := u.resolveToken(ctx, token, srcHost)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.ImageGenApiKeySummary, 0, len(tctx.Keys))
	for _, k := range tctx.Keys {
		out = append(out, &v1.ImageGenApiKeySummary{
			Id:        k.ID,
			Name:      k.Name,
			GroupId:   k.GroupID,
			GroupName: k.GroupName,
			Platform:  k.Platform,
		})
	}
	return out, nil
}

func (u *ImageGenUsecase) ListModels(ctx context.Context, token, srcHost, apiKeyID string) ([]*v1.ImageGenModel, error) {
	tctx, err := u.resolveToken(ctx, token, srcHost)
	if err != nil {
		return nil, err
	}
	key, ok := sub2apipkg.FindKey(tctx.Keys, apiKeyID)
	if !ok {
		return nil, ErrImageGenApiKeyInvalid
	}
	models, err := u.client.ListModels(ctx, tctx.SrcHost, key.RawKey)
	if err != nil {
		return nil, sub2apipkg.ClassifyImagineError(err)
	}
	out := make([]*v1.ImageGenModel, 0, len(models))
	for _, m := range models {
		// 模型是否可用于生图由上游 sub2api 决定，后端不做判定。
		out = append(out, &v1.ImageGenModel{
			Id:          m.ID,
			DisplayName: m.DisplayName,
		})
	}
	return out, nil
}

// ---------- 任务 ----------

func (u *ImageGenUsecase) ListGenerations(ctx context.Context, token, srcHost string, limit int) ([]*v1.ImageGenGeneration, error) {
	tctx, err := u.resolveToken(ctx, token, srcHost)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = listGenerationsLimit
	}
	// 惰性回收僵死 pending 任务
	_ = u.repo.SweepStalePending(ctx, tctx.UserID, tctx.SrcHost, time.Now().Add(-imageGenStaleAge))
	list, err := u.repo.ListGenerations(ctx, tctx.UserID, tctx.SrcHost, limit)
	if err != nil {
		return nil, ErrSystem(err)
	}
	// 批量取这些任务的图片并按 generation_id 分组，挂到列表项上（供缩略图预览）
	imagesByGen := map[string][]*gen.ImageGenImage{}
	if len(list) > 0 {
		ids := make([]string, 0, len(list))
		for _, g := range list {
			ids = append(ids, g.ID)
		}
		imgs, err := u.repo.ListImagesByGenerationIDs(ctx, ids)
		if err != nil {
			return nil, ErrSystem(err)
		}
		for _, im := range imgs {
			imagesByGen[im.GenerationID] = append(imagesByGen[im.GenerationID], im)
		}
	}
	out := make([]*v1.ImageGenGeneration, 0, len(list))
	for _, g := range list {
		out = append(out, toV1Generation(g, imagesByGen[g.ID]))
	}
	return out, nil
}

func (u *ImageGenUsecase) GetGeneration(ctx context.Context, token, srcHost, id string) (*v1.ImageGenGeneration, error) {
	tctx, err := u.resolveToken(ctx, token, srcHost)
	if err != nil {
		return nil, err
	}
	_ = u.repo.SweepStalePending(ctx, tctx.UserID, tctx.SrcHost, time.Now().Add(-imageGenStaleAge))
	g, err := u.repo.GetGeneration(ctx, id, tctx.UserID, tctx.SrcHost)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if g == nil {
		return nil, ErrImageGenGenerationNotFound
	}
	imgs, err := u.repo.ListImagesByGeneration(ctx, g.ID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toV1Generation(g, imgs), nil
}

func (u *ImageGenUsecase) DeleteGeneration(ctx context.Context, token, srcHost, id string) error {
	tctx, err := u.resolveToken(ctx, token, srcHost)
	if err != nil {
		return err
	}
	// 先取该任务图片路径用于清理磁盘
	imgs, err := u.repo.ListImagesByGeneration(ctx, id)
	if err != nil {
		return ErrSystem(err)
	}
	for _, im := range imgs {
		_ = imagestore.Remove(im.Path)
	}
	if err := u.repo.DeleteGeneration(ctx, id, tctx.UserID, tctx.SrcHost); err != nil {
		return ErrSystem(err)
	}
	return nil
}

// CreateImageGenGeneration 提交生成（异步）：校验后建 pending 任务并启动 worker，立即返回。
// momoko 仅做鉴权与透传，不校验/补默认生图参数——是否支持某模型、尺寸、张数由上游 API 端点决定。
func (u *ImageGenUsecase) CreateImageGenGeneration(ctx context.Context, token, srcHost string, req *v1.CreateImageGenGenerationRequest) (*v1.ImageGenGeneration, error) {
	tctx, err := u.resolveToken(ctx, token, srcHost)
	if err != nil {
		return nil, err
	}

	key, ok := sub2apipkg.FindKey(tctx.Keys, req.GetApiKeyId())
	if !ok {
		return nil, ErrImageGenApiKeyInvalid
	}

	// 文生图 / 图生图是两个独立端点；带源图即图生图，由此可靠判定，不接受客户端指定 mode。
	sourceImage := strings.TrimSpace(req.GetSourceImage())
	mode := modeText2Image
	if sourceImage != "" {
		mode = modeImage2Image
	}

	genRec, err := u.repo.CreateGeneration(ctx, req, tctx.UserID, tctx.SrcHost, mode)
	if err != nil {
		return nil, ErrSystem(err)
	}

	params := sub2apipkg.ImageGenParams{
		Model:        req.GetModel(),
		Prompt:       req.GetPrompt(),
		N:            int(req.GetN()),
		Size:         req.GetSize(),
		Quality:      req.GetQuality(),
		OutputFormat: req.GetOutputFormat(),
	}
	go u.runGeneration(genRec.ID, tctx.UserID, key.RawKey, tctx.SrcHost, sourceImage, params)

	return toV1Generation(genRec, nil), nil
}

// ServeImage 供 raw handler 读取图片记录（公开，无 user 鉴权；id 为 UUID 不可枚举）。
func (u *ImageGenUsecase) ServeImage(ctx context.Context, id string) (*gen.ImageGenImage, error) {
	img, err := u.repo.GetImage(ctx, id)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if img == nil {
		return nil, ErrImageGenImageNotFound
	}
	return img, nil
}

// ---------- 异步 worker ----------

func (u *ImageGenUsecase) runGeneration(genID, userID, apiKey, srcHost, sourceImage string, params sub2apipkg.ImageGenParams) {
	ctx := context.Background()
	started := time.Now()

	// sub2api 交互（线格式拼装、HTTP、文件落盘）全部在 pkg 中完成；biz 只在两个独立端点之间选择。
	// 带源图走图生图端点，否则走文生图端点。
	var results []sub2apipkg.ImageResult
	var runErr error
	if sourceImage != "" {
		results, runErr = u.client.EditAndSave(ctx, srcHost, apiKey, userID, genID, sourceImage, params)
	} else {
		results, runErr = u.client.GenerateAndSave(ctx, srcHost, apiKey, userID, genID, params)
	}

	finishedAt := time.Now()
	latency := finishedAt.Sub(started).Milliseconds()

	if runErr != nil {
		_ = u.repo.UpdateGenerationResult(ctx, genID, userID, "failed", truncateGenMsg(runErr.Error()), latency, 0, &finishedAt)
		return
	}

	count := 0
	for i := range results {
		if _, e := u.repo.CreateImage(ctx, genID, userID, &results[i]); e != nil {
			_ = u.repo.UpdateGenerationResult(ctx, genID, userID, "failed", truncateGenMsg(e.Error()), latency, count, &finishedAt)
			return
		}
		count++
	}

	status := "completed"
	if count == 0 {
		status = "failed"
	}
	_ = u.repo.UpdateGenerationResult(ctx, genID, userID, status, "", latency, count, &finishedAt)
}

// ---------- 辅助 ----------

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func truncateGenMsg(msg string) string {
	const max = 500
	if len(msg) <= max {
		return msg
	}
	return msg[:max] + "…"
}

// ---------- 映射 ----------

func toV1Generation(g *gen.ImageGenGeneration, imgs []*gen.ImageGenImage) *v1.ImageGenGeneration {
	if g == nil {
		return nil
	}
	out := &v1.ImageGenGeneration{
		Id:           g.ID,
		UserId:       g.UserID,
		Mode:         g.Mode,
		Model:        g.Model,
		Prompt:       g.Prompt,
		Size:         g.Size,
		N:            int32(g.N),
		Quality:      g.Quality,
		OutputFormat: g.OutputFormat,
		Status:       g.Status,
		ErrorMessage: g.ErrorMessage,
		LatencyMs:    g.LatencyMs,
		ResultCount:  int32(g.ResultCount),
		ApiKeyId:     g.APIKeyID,
		CreatedAt:    timestamppb.New(g.CreateTime),
	}
	if g.StartedAt != nil {
		out.StartedAt = timestamppb.New(*g.StartedAt)
	}
	if g.FinishedAt != nil {
		out.FinishedAt = timestamppb.New(*g.FinishedAt)
	}
	for _, im := range imgs {
		out.Images = append(out.Images, toV1Image(im))
	}
	return out
}

func toV1Image(im *gen.ImageGenImage) *v1.ImageGenImage {
	if im == nil {
		return nil
	}
	return &v1.ImageGenImage{
		Id:            im.ID,
		GenerationId:  im.GenerationID,
		Filename:      im.Filename,
		Index:         int32(im.Index),
		Width:         int32(im.Width),
		Height:        int32(im.Height),
		MimeType:      im.MimeType,
		SizeBytes:     im.SizeBytes,
		RevisedPrompt: im.RevisedPrompt,
		CreatedAt:     timestamppb.New(im.CreateTime),
	}
}
