package biz

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/mgp"
	"momoko/pkg/mgp/gakumas"
)

// 学マス 的配置文档：Momoko 按 id 保存草稿与版本，卡池与公告可以发布，其余只读（运营日历）。
const (
	GakumasConfigGasha      = "gasha"
	GakumasConfigNotice     = "notice"
	GakumasConfigPvpRate    = "pvpRate"
	GakumasConfigHome       = "home"
	GakumasConfigResearch   = "research"
	GakumasConfigStoryEvent = "storyEvent"
	GakumasConfigLoginBonus = "loginBonus"
)

type gakumasConfig struct {
	title    string
	file     gakumas.ConfigFile
	editable bool
}

var gakumasConfigs = map[string]gakumasConfig{
	GakumasConfigGasha:      {title: "卡池", file: gakumas.ConfigGasha, editable: true},
	GakumasConfigNotice:     {title: "公告", file: gakumas.ConfigNotice, editable: true},
	GakumasConfigPvpRate:    {title: "竞赛", file: gakumas.ConfigPvpRate},
	GakumasConfigHome:       {title: "主页", file: gakumas.ConfigHome},
	GakumasConfigResearch:   {title: "研究活动", file: gakumas.ConfigResearch},
	GakumasConfigStoryEvent: {title: "活动", file: gakumas.ConfigStoryEvent},
	GakumasConfigLoginBonus: {title: "登录奖励", file: gakumas.ConfigLoginBonus},
}

// editable 可发布的配置文档。
func editableConfig(id string) (gakumasConfig, error) {
	cfg := gakumasConfigs[id]
	if !cfg.editable {
		return cfg, ErrGmConfigReadOnly
	}
	return cfg, nil
}

// Config 读取配置文档：服务端内容与 revision、草稿与最近发布。
func (uc *GakumasUsecase) Config(ctx context.Context, serverID, id string) (*v1.GetGmConfigResponse, error) {
	g, err := uc.client(ctx, serverID)
	if err != nil {
		return nil, err
	}
	cfg := gakumasConfigs[id]
	doc, err := g.Config(ctx, cfg.file)
	if err != nil {
		return nil, GameMGPError(err)
	}
	d := &v1.GmConfigDomain{
		Id:       id,
		Title:    cfg.title,
		File:     doc.File,
		Revision: doc.Revision,
	}
	repo := uc.gm.Repo()
	if draft, err := repo.GetConfigDraft(ctx, serverID, id); err == nil && draft != nil {
		d.Draft = configDraftProto(draft)
	}
	if last, err := repo.LastPublished(ctx, serverID); err == nil {
		if r, ok := last[id]; ok {
			d.LastSeq, d.LastPublishedBy = int32(r.Seq), r.AuthorName
			if r.PublishedAt != nil {
				d.LastPublishedAt = timestamppb.New(*r.PublishedAt)
			}
		}
	}
	return &v1.GetGmConfigResponse{
		Domain:  d,
		Content: string(doc.Content),
	}, nil
}

// SaveConfigDraft 保存草稿（每份配置文档一份）。
func (uc *GakumasUsecase) SaveConfigDraft(ctx context.Context, actor GmActor, serverID, id, content, baseRevision, comment string) (*v1.GmConfigDraft, error) {
	if _, err := uc.client(ctx, serverID); err != nil {
		return nil, err
	}
	if _, err := editableConfig(id); err != nil {
		return nil, err
	}
	if !validJSON(content) {
		return nil, ErrGmConfigInvalidJSON
	}
	saved, err := uc.gm.Repo().SaveConfigDraft(ctx, &gen.GameConfigRevision{
		ServerID:     serverID,
		Domain:       id,
		BaseRevision: baseRevision,
		Content:      content,
		AuthorID:     actor.UserID,
		AuthorName:   actor.Name,
		Comment:      strings.TrimSpace(comment),
	})
	if err != nil {
		return nil, ErrSystem(err)
	}
	return configDraftProto(saved), nil
}

// DiscardConfigDraft 放弃草稿。
func (uc *GakumasUsecase) DiscardConfigDraft(ctx context.Context, serverID, id string) error {
	if _, err := uc.client(ctx, serverID); err != nil {
		return err
	}
	if err := uc.gm.Repo().DeleteConfigDraft(ctx, serverID, id); err != nil {
		return ErrSystem(err)
	}
	return nil
}

// ValidateConfig 用服务端的加载逻辑校验内容（试运行发布，不改变服务端）。
func (uc *GakumasUsecase) ValidateConfig(ctx context.Context, serverID, id, content string) (*v1.ValidateGmConfigResponse, error) {
	g, err := uc.client(ctx, serverID)
	if err != nil {
		return nil, err
	}
	cfg, err := editableConfig(id)
	if err != nil {
		return nil, err
	}
	if !validJSON(content) {
		return &v1.ValidateGmConfigResponse{
			Valid: false,
			Error: "invalid JSON",
		}, nil
	}
	r, err := g.PublishConfig(ctx, cfg.file, json.RawMessage(content), "", "", mgp.OperationOptions{
		DryRun: true,
	})
	if err != nil {
		return nil, GameMGPError(err)
	}
	return &v1.ValidateGmConfigResponse{
		Valid: r.Valid,
		Error: r.Error,
	}, nil
}

// PublishConfig 发布配置：写回服务端并热重载，记一个版本与一条 config.publish GM 记录。content 为空时发布草稿。
func (uc *GakumasUsecase) PublishConfig(ctx context.Context, actor GmActor, serverID, id, content, baseRevision, comment string) (*v1.PublishGmConfigResponse, error) {
	s, g, err := uc.server(ctx, serverID)
	if err != nil {
		return nil, err
	}
	cfg, err := editableConfig(id)
	if err != nil {
		return nil, err
	}
	repo := uc.gm.Repo()
	base := baseRevision
	comment = strings.TrimSpace(comment)
	if content == "" {
		draft, _ := repo.GetConfigDraft(ctx, serverID, id)
		if draft == nil {
			return nil, ErrGmConfigNoDraft
		}
		content = draft.Content
		if base == "" {
			base = draft.BaseRevision
		}
	}
	if !validJSON(content) {
		return nil, ErrGmConfigInvalidJSON
	}
	last, _ := repo.LastPublished(ctx, serverID)
	fromSeq := 0
	if r, ok := last[id]; ok {
		fromSeq = r.Seq
	}
	e := &gen.GmExecution{
		ID:           uuid.NewString(),
		ServerID:     s.ID,
		Command:      "config.publish",
		CommandTitle: cfg.title,
		Risk:         RiskHigh,
		TargetKind:   TargetServer,
		ActorID:      actor.UserID,
		ActorName:    actor.Name,
		ActorKind:    gmActorKind(actor.Kind),
		Channel:      gmChannel(actor.Channel),
		ClientIP:     actor.IP,
		Platform:     actor.Platform,
		Params:       "{}",
		Summary: map[string]string{
			"domain":  id,
			"title":   cfg.title,
			"fromSeq": strconv.Itoa(fromSeq),
			"toSeq":   strconv.Itoa(fromSeq + 1),
			"comment": comment,
		},
		Status:    "running",
		RequestID: "momoko-" + uuid.NewString(),
		Source:    "config:" + id,
	}
	if e, err = repo.CreateExecution(ctx, e); err != nil {
		return nil, ErrSystem(err)
	}
	start := time.Now()
	r, err := g.PublishConfig(ctx, cfg.file, json.RawMessage(content), base, comment, mgp.OperationOptions{
		IdempotencyKey: e.RequestID,
		Operator:       actor.Name,
	})
	now := time.Now()
	e.DurationMs, e.FinishedAt = int(now.Sub(start).Milliseconds()), &now
	if err != nil {
		e.Status, e.FailedCount, e.ErrorCode, e.ErrorMessage = "failed", 1, mgp.CodeOf(err), err.Error()
		_, _ = repo.SaveExecution(context.WithoutCancel(ctx), e)
		if mgp.CodeOf(err) == "REVISION_MISMATCH" {
			return nil, ErrGmConfigConflict
		}
		return nil, GameMGPError(err)
	}
	e.Status, e.SuccessCount = "success", 1
	e.Summary["revision"] = r.Revision
	rev, err := repo.PublishConfig(ctx, &gen.GameConfigRevision{
		ServerID:     s.ID,
		Domain:       id,
		Revision:     r.Revision,
		BaseRevision: base,
		Content:      content,
		AuthorID:     actor.UserID,
		AuthorName:   actor.Name,
		Comment:      comment,
		PublishedAt:  &now,
		ExecutionID:  e.ID,
	})
	if err != nil {
		uc.log.Errorf("record config revision: %v", err)
	}
	seq := fromSeq + 1
	if rev != nil {
		seq = rev.Seq
		e.Summary["toSeq"] = strconv.Itoa(seq)
	}
	_, _ = repo.SaveExecution(context.WithoutCancel(ctx), e)
	return &v1.PublishGmConfigResponse{
		Revision:    r.Revision,
		Seq:         int32(seq),
		ExecutionId: e.ID,
	}, nil
}

// ConfigRevisions 版本记录。
func (uc *GakumasUsecase) ConfigRevisions(ctx context.Context, serverID, id string, page, size int32) (*v1.ListGmConfigRevisionsResponse, error) {
	if _, err := uc.client(ctx, serverID); err != nil {
		return nil, err
	}
	items, total, err := uc.gm.Repo().ListConfigRevisions(ctx, serverID, id, int(page), int(size))
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := &v1.ListGmConfigRevisionsResponse{
		Total: int32(total),
	}
	for _, r := range items {
		out.Items = append(out.Items, ConfigRevisionProto(r, false))
	}
	return out, nil
}

// ConfigRevision 某个版本的内容。
func (uc *GakumasUsecase) ConfigRevision(ctx context.Context, serverID, domain, id string) (*v1.GmConfigRevision, error) {
	if _, err := uc.client(ctx, serverID); err != nil {
		return nil, err
	}
	r, err := uc.gm.Repo().GetConfigRevision(ctx, id)
	if gen.IsNotFound(err) || (err == nil && (r.ServerID != serverID || r.Domain != domain)) {
		return nil, ErrGmConfigRevisionNotFound
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	return ConfigRevisionProto(r, true), nil
}
