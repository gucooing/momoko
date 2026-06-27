package file

import (
	"context"
	"os"
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

// StartUploadJanitor 启动后台 GC：每 interval 清理一次「超过 retention 仍未收尾」的上传会话，
// 删除残留的 .part 临时文件与对应的数据库记录，避免磁盘与表项随中断上传无限堆积。
// 随 ctx 取消而退出。
func StartUploadJanitor(ctx context.Context, store UploadGCStore, interval, retention time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				gcStaleUploads(ctx, store, retention)
			}
		}
	}()
}

func gcStaleUploads(ctx context.Context, store UploadGCStore, retention time.Duration) {
	stale, err := store.ListStaleUploads(ctx, time.Now().Add(-retention))
	if err != nil {
		return
	}
	for _, u := range stale {
		_ = os.Remove(uploadBufferPath(u))
		_ = store.DeleteUploadCascade(ctx, u.ID)
	}
}
