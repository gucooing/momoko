package data

import (
	"context"

	"github.com/google/uuid"

	"momoko/internal/biz"
	"momoko/internal/data/ent"
	"momoko/internal/data/ent/fileupload"
	"momoko/internal/data/ent/fileuploadchunk"
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

func (f *fileRepo) GetOrCreate(ctx context.Context, userId string, info *file.ChunkedUpload) (*ent.FileUpload, error) {
	var uInfo *ent.FileUpload
	err := utils.WithTx(ctx, f.data.db, func(tx *ent.Tx) error {
		existing, err := tx.FileUpload.
			Query().
			Where(
				fileupload.HashEQ(info.Hash),
				fileupload.PathEQ(info.Path),
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
			SetUserID(userId).Save(ctx)
		if err != nil {
			return err
		}
		return nil
	})
	return uInfo, err
}

func (f *fileRepo) Query(ctx context.Context, id string) (*ent.FileUpload, error) {
	return f.data.db.FileUpload.Query().Where(
		fileupload.IDEQ(id),
	).WithChunks().Only(ctx)
}

func (f *fileRepo) QueryByUserID(ctx context.Context, userID, id string) (*ent.FileUpload, error) {
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

func (f *fileRepo) WithTx(ctx context.Context, fn func(tx *ent.Tx) error) error {
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
