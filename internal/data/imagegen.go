package data

import (
	"context"
	"time"

	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/imagegengeneration"
	"momoko/internal/data/ent/gen/imagegenimage"
	sub2apipkg "momoko/pkg/sub2api"
	"momoko/pkg/utils"
)

type imageGenRepo struct {
	data *Data
}

func NewImageGenRepo(data *Data) biz.ImageGenRepo {
	return &imageGenRepo{data: data}
}

// ---------- 任务 ----------

func (r *imageGenRepo) ListGenerations(ctx context.Context, userID, srcHost string, limit int) ([]*gen.ImageGenGeneration, error) {
	if limit <= 0 {
		limit = 50
	}
	return r.data.db.ImageGenGeneration.Query().
		Where(imagegengeneration.UserIDEQ(userID), imagegengeneration.SrcHostEQ(srcHost)).
		Order(gen.Desc(imagegengeneration.FieldCreateTime)).
		Limit(limit).
		All(ctx)
}

func (r *imageGenRepo) CreateGeneration(ctx context.Context, req *v1.CreateImageGenGenerationRequest, userID, srcHost, mode string) (*gen.ImageGenGeneration, error) {
	now := time.Now()
	builder := r.data.db.ImageGenGeneration.Create().
		SetID(uuid.NewString()).
		SetUserID(userID).
		SetMode(mode).
		SetAPIKeyID(req.GetApiKeyId()).
		SetModel(req.GetModel()).
		SetPrompt(req.GetPrompt()).
		SetSize(req.GetSize()).
		SetN(int(req.GetN())).
		SetStatus("pending").
		SetStartedAt(now)
	if srcHost != "" {
		builder = builder.SetSrcHost(srcHost)
	}
	if q := req.GetQuality(); q != "" {
		builder = builder.SetQuality(q)
	}
	if of := req.GetOutputFormat(); of != "" {
		builder = builder.SetOutputFormat(of)
	}
	return builder.Save(ctx)
}

func (r *imageGenRepo) UpdateGenerationResult(ctx context.Context, id, userID, status, errMsg string, latencyMs int64, resultCount int, finishedAt *time.Time) error {
	builder := r.data.db.ImageGenGeneration.Update().
		Where(imagegengeneration.IDEQ(id), imagegengeneration.UserIDEQ(userID)).
		SetStatus(status).
		SetLatencyMs(latencyMs).
		SetResultCount(resultCount).
		SetErrorMessage(errMsg)
	if finishedAt != nil {
		builder = builder.SetFinishedAt(*finishedAt)
	}
	_, err := builder.Save(ctx)
	return err
}

func (r *imageGenRepo) GetGeneration(ctx context.Context, id, userID, srcHost string) (*gen.ImageGenGeneration, error) {
	g, err := r.data.db.ImageGenGeneration.Query().
		Where(imagegengeneration.IDEQ(id), imagegengeneration.UserIDEQ(userID), imagegengeneration.SrcHostEQ(srcHost)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return g, nil
}

func (r *imageGenRepo) DeleteGeneration(ctx context.Context, id, userID, srcHost string) error {
	// 校验归属：未归属则不删除
	g, err := r.GetGeneration(ctx, id, userID, srcHost)
	if err != nil || g == nil {
		return err
	}
	return utils.WithTx(ctx, r.data.db, func(tx *gen.Tx) error {
		if _, err := tx.ImageGenImage.Delete().Where(imagegenimage.GenerationIDEQ(id)).Exec(ctx); err != nil {
			return err
		}
		return tx.ImageGenGeneration.DeleteOneID(id).Exec(ctx)
	})
}

func (r *imageGenRepo) SweepStalePending(ctx context.Context, userID, srcHost string, before time.Time) error {
	_, err := r.data.db.ImageGenGeneration.Update().
		Where(imagegengeneration.UserIDEQ(userID), imagegengeneration.SrcHostEQ(srcHost), imagegengeneration.StatusEQ("pending"), imagegengeneration.StartedAtLTE(before)).
		SetStatus("failed").
		SetErrorMessage("生成超时或被中断").
		Save(ctx)
	return err
}

// ---------- 图片 ----------

func (r *imageGenRepo) CreateImage(ctx context.Context, generationID, userID string, img *sub2apipkg.ImageResult) (*gen.ImageGenImage, error) {
	builder := r.data.db.ImageGenImage.Create().
		SetID(uuid.NewString()).
		SetGenerationID(generationID).
		SetUserID(userID).
		SetPath(img.Path).
		SetFilename(img.Filename).
		SetSizeBytes(img.SizeBytes).
		SetWidth(img.Width).
		SetHeight(img.Height).
		SetMimeType(img.MimeType).
		SetIndex(img.Index)
	if img.RevisedPrompt != "" {
		builder = builder.SetRevisedPrompt(img.RevisedPrompt)
	}
	return builder.Save(ctx)
}

func (r *imageGenRepo) ListImagesByGeneration(ctx context.Context, generationID string) ([]*gen.ImageGenImage, error) {
	return r.data.db.ImageGenImage.Query().
		Where(imagegenimage.GenerationIDEQ(generationID)).
		Order(gen.Asc(imagegenimage.FieldIndex)).
		All(ctx)
}

func (r *imageGenRepo) ListImagesByGenerationIDs(ctx context.Context, ids []string) ([]*gen.ImageGenImage, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	return r.data.db.ImageGenImage.Query().
		Where(imagegenimage.GenerationIDIn(ids...)).
		Order(gen.Asc(imagegenimage.FieldIndex)).
		All(ctx)
}

func (r *imageGenRepo) GetImage(ctx context.Context, id string) (*gen.ImageGenImage, error) {
	img, err := r.data.db.ImageGenImage.Query().
		Where(imagegenimage.IDEQ(id)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return img, nil
}
