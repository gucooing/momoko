package biz

import (
	"context"
	"time"

	"momoko/internal/data/ent/gen"
	"momoko/pkg/file"
	"momoko/pkg/task"
)

// uploadCanceler 在清理一个残留上传会话时，先经其来源中止机制（删本地缓冲 / 中止 OSS multipart）。
type uploadCanceler func(ctx context.Context, row *gen.FileUpload)

// uploadGCTask 是清理废弃上传会话的定时单例任务。
//
// 它放在 biz 而不是 pkg/file：清理要读写 ent 行，属于业务编排；
// 让 pkg 反向依赖 internal/data 会把分层拧成环，也使 pkg 无法独立测试。
type uploadGCTask struct {
	repo      FileRepo
	interval  time.Duration
	retention time.Duration
	cancel    uploadCanceler
}

func newUploadGCTask(repo FileRepo, interval, retention time.Duration, cancel uploadCanceler) task.Task {
	return &uploadGCTask{repo: repo, interval: interval, retention: retention, cancel: cancel}
}

func (t *uploadGCTask) Meta() task.Meta {
	return task.Meta{
		ID:       file.TaskTypeUploadGC,
		Type:     file.TaskTypeUploadGC,
		Kind:     task.KindScheduled,
		Resume:   task.ResumeAlways,
		Interval: t.interval,
		Title:    "上传会话清理",
	}
}

func (t *uploadGCTask) Payload() any { return nil }

func (t *uploadGCTask) Run(ctx context.Context, _ task.Reporter) error {
	rows, err := t.repo.ListStaleUploads(ctx, time.Now().Add(-t.retention))
	if err != nil {
		return nil // 列举失败不算任务失败，下个周期再来
	}
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		if t.cancel != nil {
			t.cancel(ctx, row)
		}
		_ = t.repo.DeleteUploadCascade(ctx, row.ID)
	}
	return nil
}
