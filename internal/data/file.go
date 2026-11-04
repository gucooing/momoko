package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/fileshare"
	"momoko/internal/data/ent/gen/filesharedownload"
	"momoko/internal/data/ent/gen/filesource"
	"momoko/internal/data/ent/gen/fileupload"
	"momoko/internal/data/ent/gen/fileuploadchunk"
	"momoko/internal/data/ent/gen/predicate"
	"momoko/internal/data/ent/schema/sharetype"
	"momoko/pkg/file"
	"momoko/pkg/utils"
)

type fileRepo struct {
	data *Data
}

func NewFileRepo(data *Data) biz.FileRepo {
	return &fileRepo{
		data: data,
	}
}

func (f *fileRepo) GetOrCreate(ctx context.Context, userId string, info *file.ChunkedUpload) (*gen.FileUpload, error) {
	var uInfo *gen.FileUpload
	err := utils.WithTx(ctx, f.data.db, func(tx *gen.Tx) error {
		existing, err := tx.FileUpload.
			Query().
			Where(
				fileupload.HashEQ(info.Hash),
				fileupload.PathEQ(info.Path),
				fileupload.SourceIDEQ(info.SourceID),
				fileupload.CompletedEQ(false),
				fileupload.CancelEQ(false),
			).
			WithChunks().
			Only(ctx)
		if err == nil {
			uInfo = existing
			return nil
		}
		uInfo, err = tx.FileUpload.Create().
			SetID(uuid.NewString()).
			SetHash(info.Hash).
			SetPath(info.Path).
			SetFileName(info.FileName).
			SetFileSize(info.FileSize).
			SetChunkSize(info.ChunkSize).
			SetTotalChunks(info.TotalChunks).
			SetSourceID(info.SourceID).
			SetBufferLocal(info.BufferLocal).
			SetUserID(userId).Save(ctx)
		if err != nil {
			return err
		}
		return nil
	})
	return uInfo, err
}

func (f *fileRepo) Query(ctx context.Context, id string) (*gen.FileUpload, error) {
	return f.data.db.FileUpload.Query().Where(
		fileupload.IDEQ(id),
	).WithChunks().Only(ctx)
}

func (f *fileRepo) QueryByUserID(ctx context.Context, userID, id string) (*gen.FileUpload, error) {
	return f.data.db.FileUpload.Query().Where(
		fileupload.IDEQ(id),
		fileupload.UserIDEQ(userID),
	).WithChunks().Only(ctx)
}

func (f *fileRepo) SaveChunkRecord(ctx context.Context, uploadID string, chunk uint64, hash string, size uint64) error {
	err := f.data.db.FileUploadChunk.
		Create().
		SetUploadID(uploadID).
		SetChunk(chunk).
		SetHash(hash).
		SetSize(size).
		OnConflictColumns(fileuploadchunk.FieldUploadID, fileuploadchunk.FieldChunk).
		UpdateNewValues().
		Exec(ctx)

	return err
}

// ListStaleUploads 返回创建时间早于 before、且仍未完成、未取消的上传会话（供后台 GC 清理）。
func (f *fileRepo) ListStaleUploads(ctx context.Context, before time.Time) ([]*gen.FileUpload, error) {
	return f.data.db.FileUpload.Query().
		Where(
			fileupload.CompletedEQ(false),
			fileupload.CancelEQ(false),
			fileupload.CreateTimeLT(before),
		).All(ctx)
}

// DeleteUploadCascade 删除一个上传会话及其全部分片记录。
func (f *fileRepo) DeleteUploadCascade(ctx context.Context, id string) error {
	return utils.WithTx(ctx, f.data.db, func(tx *gen.Tx) error {
		if _, err := tx.FileUploadChunk.Delete().
			Where(fileuploadchunk.UploadIDEQ(id)).
			Exec(ctx); err != nil {
			return err
		}
		return tx.FileUpload.DeleteOneID(id).Exec(ctx)
	})
}

// SetUploadCompleted 标记一个上传会话为已完成。
func (f *fileRepo) SetUploadCompleted(ctx context.Context, id string) error {
	return f.data.db.FileUpload.UpdateOneID(id).SetCompleted(true).Exec(ctx)
}

// SetUploadProviderRefIfEmpty 仅当 provider_ref 仍为空时写入，返回是否写入成功（OSS uploadID 初始化幂等/防竞态）。
func (f *fileRepo) SetUploadProviderRefIfEmpty(ctx context.Context, id, ref string) (bool, error) {
	n, err := f.data.db.FileUpload.Update().
		Where(fileupload.IDEQ(id), fileupload.ProviderRefEQ("")).
		SetProviderRef(ref).
		Save(ctx)
	return n > 0, err
}

// CreateShare 新建一条分享记录。items 为已探测/缓存好的被分享条目（可跨来源）。
func (f *fileRepo) CreateShare(ctx context.Context, userID, token, name string, items []sharetype.Item, req *v1.CreateShareRequest) (*gen.FileShare, error) {
	builder := f.data.db.FileShare.Create().
		SetID(uuid.NewString()).
		SetUserID(userID).
		SetName(name).
		SetItems(items).
		SetToken(token).
		SetCode(req.Code).
		SetMaxDownloads(req.MaxDownloads).
		SetEnabled(req.Enabled)
	if req.ExpiresAt != nil {
		builder.SetExpiresAt(req.ExpiresAt.AsTime())
	}
	return builder.Save(ctx)
}

// sharePredicates 返回分享状态筛选条件。状态优先级与前端一致：停用 > 过期 > 次数用完 > 有效。
func shareStatusPredicate(status string, now time.Time) predicate.FileShare {
	notExpired := fileshare.Or(fileshare.ExpiresAtIsNil(), fileshare.ExpiresAtGT(now))
	exhausted := predicate.FileShare(func(s *sql.Selector) {
		s.Where(sql.And(
			sql.GT(s.C(fileshare.FieldMaxDownloads), 0),
			sql.ColumnsGTE(s.C(fileshare.FieldDownloadCount), s.C(fileshare.FieldMaxDownloads)),
		))
	})
	switch status {
	case "disabled":
		return fileshare.EnabledEQ(false)
	case "expired":
		return fileshare.And(fileshare.EnabledEQ(true), fileshare.ExpiresAtNotNil(), fileshare.ExpiresAtLTE(now))
	case "exhausted":
		return fileshare.And(fileshare.EnabledEQ(true), notExpired, exhausted)
	case "active":
		return fileshare.And(fileshare.EnabledEQ(true), notExpired, fileshare.Not(exhausted))
	}
	return nil
}

// ListShares 分页返回某用户的分享（按创建时间倒序），支持名称/令牌/提取码关键字与状态筛选。
func (f *fileRepo) ListShares(ctx context.Context, userID string, page, pageSize int64, keywords, status string) ([]*gen.FileShare, int64, error) {
	query := f.data.db.FileShare.Query().Where(fileshare.UserIDEQ(userID))
	if keywords != "" {
		query = query.Where(fileshare.Or(
			fileshare.NameContainsFold(keywords),
			fileshare.TokenContains(keywords),
			fileshare.CodeEQ(keywords),
		))
	}
	if p := shareStatusPredicate(status, time.Now()); p != nil {
		query = query.Where(p)
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	items, err := query.
		Order(gen.Desc(fileshare.FieldCreateTime)).
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, int64(total), nil
}

// ShareStats 统计某用户全部分享的状态分布、本月下载次数与其中 302 直链下发次数（不受筛选/分页影响）。
func (f *fileRepo) ShareStats(ctx context.Context, userID string) (*v1.ShareStats, error) {
	rows, err := f.data.db.FileShare.Query().
		Where(fileshare.UserIDEQ(userID)).
		Select(
			fileshare.FieldEnabled,
			fileshare.FieldExpiresAt,
			fileshare.FieldMaxDownloads,
			fileshare.FieldDownloadCount,
		).All(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	soon := now.Add(72 * time.Hour)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthQuery := f.data.db.FileShareDownload.Query().Where(
		filesharedownload.UserIDEQ(userID),
		filesharedownload.CreateTimeGTE(monthStart),
	)
	month, err := monthQuery.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	redirect, err := monthQuery.Where(filesharedownload.RedirectEQ(true)).Count(ctx)
	if err != nil {
		return nil, err
	}
	stats := &v1.ShareStats{
		Total:                  int64(len(rows)),
		MonthDownloads:         int64(month),
		MonthRedirectDownloads: int64(redirect),
	}
	for _, r := range rows {
		switch {
		case !r.Enabled:
			stats.Disabled++
		case r.ExpiresAt != nil && !r.ExpiresAt.After(now):
			stats.Expired++
		case r.MaxDownloads > 0 && r.DownloadCount >= r.MaxDownloads:
			stats.Exhausted++
		default:
			stats.Active++
			if r.ExpiresAt != nil && r.ExpiresAt.Before(soon) {
				stats.ExpiringSoon++
			}
		}
	}
	return stats, nil
}

// GetShare 按 id 获取某用户自己的分享。
func (f *fileRepo) GetShare(ctx context.Context, userID, id string) (*gen.FileShare, error) {
	return f.data.db.FileShare.Query().Where(fileshare.IDEQ(id), fileshare.UserIDEQ(userID)).Only(ctx)
}

// GetShareByToken 按公开令牌获取分享（不限用户）。供公开访问使用；带出分享者用于公开展示。
func (f *fileRepo) GetShareByToken(ctx context.Context, token string) (*gen.FileShare, error) {
	return f.data.db.FileShare.Query().Where(fileshare.TokenEQ(token)).WithUser().Only(ctx)
}

// UpdateShare 全量覆盖可编辑字段（仅限本人）。expires_at 为空表示清除（永久）。
// items 非空时覆盖分享内容（已探测/缓存好，可跨来源）；为空表示不修改内容。
func (f *fileRepo) UpdateShare(ctx context.Context, userID, name string, items []sharetype.Item, req *v1.UpdateShareRequest) (*gen.FileShare, error) {
	builder := f.data.db.FileShare.UpdateOneID(req.Id).
		Where(fileshare.UserIDEQ(userID)).
		SetName(name).
		SetCode(req.Code).
		SetMaxDownloads(req.MaxDownloads).
		SetEnabled(req.Enabled)
	if len(items) > 0 {
		builder.SetItems(items)
	}
	if req.ExpiresAt != nil {
		builder.SetExpiresAt(req.ExpiresAt.AsTime())
	} else {
		builder.ClearExpiresAt()
	}
	return builder.Save(ctx)
}

// DeleteShare 删除一条分享（仅限本人）及其下载记录。
func (f *fileRepo) DeleteShare(ctx context.Context, userID, id string) error {
	return utils.WithTx(ctx, f.data.db, func(tx *gen.Tx) error {
		n, err := tx.FileShare.Delete().
			Where(fileshare.IDEQ(id), fileshare.UserIDEQ(userID)).
			Exec(ctx)
		if err != nil || n == 0 {
			return err
		}
		_, err = tx.FileShareDownload.Delete().Where(filesharedownload.ShareIDEQ(id)).Exec(ctx)
		return err
	})
}

// UpdateShareItems 回写分享条目缓存（旧记录补统计后使用），不改动其它字段与更新时间语义之外的内容。
func (f *fileRepo) UpdateShareItems(ctx context.Context, id string, items []sharetype.Item) error {
	return f.data.db.FileShare.UpdateOneID(id).SetItems(items).Exec(ctx)
}

// RecordShareDownload 记一次计数下载：下载次数 +1，并写一条下载记录（是否 302 直链）供统计。
func (f *fileRepo) RecordShareDownload(ctx context.Context, shareID, ownerID string, redirect bool) error {
	return utils.WithTx(ctx, f.data.db, func(tx *gen.Tx) error {
		if err := tx.FileShare.UpdateOneID(shareID).AddDownloadCount(1).Exec(ctx); err != nil {
			return err
		}
		return tx.FileShareDownload.Create().
			SetShareID(shareID).
			SetUserID(ownerID).
			SetRedirect(redirect).
			Exec(ctx)
	})
}

// DeleteSharesExpiredBefore 删除过期时间早于 before 的分享及其下载记录（后台定时清理）。
func (f *fileRepo) DeleteSharesExpiredBefore(ctx context.Context, before time.Time) (int, error) {
	var deleted int
	err := utils.WithTx(ctx, f.data.db, func(tx *gen.Tx) error {
		ids, err := tx.FileShare.Query().
			Where(fileshare.ExpiresAtNotNil(), fileshare.ExpiresAtLT(before)).
			IDs(ctx)
		if err != nil || len(ids) == 0 {
			return err
		}
		if _, err := tx.FileShareDownload.Delete().Where(filesharedownload.ShareIDIn(ids...)).Exec(ctx); err != nil {
			return err
		}
		deleted, err = tx.FileShare.Delete().Where(fileshare.IDIn(ids...)).Exec(ctx)
		return err
	})
	return deleted, err
}

// ---- 文件来源（OSS/FTP/WebDAV，全局/管理员维护）----

// CreateFileSource 新建一条文件来源；config 为已加密好的 JSON 串。
func (f *fileRepo) CreateFileSource(ctx context.Context, userID, name, typ, configJSON string, enabled, redirect302 bool) (*gen.FileSource, error) {
	return f.data.db.FileSource.Create().
		SetID(uuid.NewString()).
		SetUserID(userID).
		SetName(name).
		SetType(filesource.Type(typ)).
		SetEnabled(enabled).
		SetRedirect302(redirect302).
		SetConfig(configJSON).
		Save(ctx)
}

// ListFileSources 返回全部文件来源（全局可见），可按名称过滤、仅取启用项；带出创建者。
func (f *fileRepo) ListFileSources(ctx context.Context, keywords string, enabledOnly bool) ([]*gen.FileSource, error) {
	query := f.data.db.FileSource.Query()
	if keywords != "" {
		query = query.Where(filesource.NameContainsFold(keywords))
	}
	if enabledOnly {
		query = query.Where(filesource.EnabledEQ(true))
	}
	return query.Order(gen.Desc(filesource.FieldCreateTime)).WithUser().All(ctx)
}

// GetFileSource 按 id 获取一条文件来源（带出创建者）。
func (f *fileRepo) GetFileSource(ctx context.Context, id string) (*gen.FileSource, error) {
	return f.data.db.FileSource.Query().Where(filesource.IDEQ(id)).WithUser().Only(ctx)
}

// UpdateFileSource 更新一条文件来源（类型不可改）。
func (f *fileRepo) UpdateFileSource(ctx context.Context, id, name, configJSON string, enabled, redirect302 bool) (*gen.FileSource, error) {
	return f.data.db.FileSource.UpdateOneID(id).
		SetName(name).
		SetEnabled(enabled).
		SetRedirect302(redirect302).
		SetConfig(configJSON).
		Save(ctx)
}

// SaveFileSourceTest 回写一条文件来源最近一次连通性测试结果（含测得的已用空间 / 总容量，0=未知）。
func (f *fileRepo) SaveFileSourceTest(ctx context.Context, id string, res *v1.FileSourceTestResult) error {
	return f.data.db.FileSource.UpdateOneID(id).
		SetLastTestOk(res.Ok).
		SetLastTestLatencyMs(res.LatencyMs).
		SetLastTestMessage(res.Message).
		SetLastTestTime(res.TestedAt.AsTime()).
		SetLastTestUsedBytes(res.UsedBytes).
		SetLastTestTotalBytes(res.TotalBytes).
		Exec(ctx)
}

// DeleteFileSource 删除一条文件来源。
func (f *fileRepo) DeleteFileSource(ctx context.Context, id string) error {
	return f.data.db.FileSource.DeleteOneID(id).Exec(ctx)
}

func (f *fileRepo) WithTx(ctx context.Context, fn func(tx *gen.Tx) error) error {
	tx, err := f.data.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	err = fn(tx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
