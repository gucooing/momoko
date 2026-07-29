package file

import (
	"context"
	"time"

	"momoko/internal/data/ent/gen"
)

// UploadGCStore 是上传 GC 需要的数据访问能力（由数据层实现），使清理逻辑不依赖具体仓储。
type UploadGCStore interface {
	// ListStaleUploads 返回创建时间早于 before、仍未完成且未取消的上传会话。
	ListStaleUploads(ctx context.Context, before time.Time) ([]*gen.FileUpload, error)
	// DeleteUploadCascade 删除一个上传会话及其全部分片记录。
	DeleteUploadCascade(ctx context.Context, id string) error
}

// gcStaleUploads 清理「超过 retention 仍未收尾」的上传会话：先经 cancel 中止来源侧机制
// （删本地缓冲 / 中止 OSS 残留 multipart），再删除数据库记录，避免磁盘、对象存储分片与表项随中断上传无限堆积。
func gcStaleUploads(ctx context.Context, store UploadGCStore, retention time.Duration, cancel UploadCanceler) {
	stale, err := store.ListStaleUploads(ctx, time.Now().Add(-retention))
	if err != nil {
		return
	}
	for _, u := range stale {
		if cancel != nil {
			cancel(ctx, u)
		}
		_ = store.DeleteUploadCascade(ctx, u.ID)
	}
}
