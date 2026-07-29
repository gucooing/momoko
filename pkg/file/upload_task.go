package file

import (
	"context"
	"time"

	"momoko/internal/data/ent/gen"
	"momoko/pkg/task"
)

// 上传相关任务类型键。
const (
	TaskTypeUploadFinalize = "file.upload_finalize"
	TaskTypeUploadGC       = "file.upload_gc"
)

// UploadStore 是上传收尾/清理任务所需的数据访问能力（由数据层实现）。
type UploadStore interface {
	UploadGCStore
	// SetUploadCompleted 标记一个上传会话为已完成。
	SetUploadCompleted(ctx context.Context, id string) error
}

// FinalizePayload 是远端上传收尾任务的重建参数。
type FinalizePayload struct {
	UploadID string `json:"upload_id"`
}

// finalizeTask 把已缓冲到本地的整文件整流推送到远端来源（FTP/WebDAV），成功后标记完成并删缓冲。
// 这是「长任务」：用不超时 ctx 在任务管理器内执行，重启可按 rerun 续做。
type finalizeTask struct {
	meta   task.Meta
	store  Store
	upload *Upload
	db     UploadStore
}

// NewFinalizeTask 构造一个远端上传收尾任务（store 自行处理收尾机制，db 标记完成态）。
func NewFinalizeTask(meta task.Meta, store Store, upload *Upload, db UploadStore) task.Task {
	meta.Type = TaskTypeUploadFinalize
	meta.Kind = task.KindOneShot
	meta.Resume = task.ResumeRerun
	meta.Total = 1
	return &finalizeTask{meta: meta, store: store, upload: upload, db: db}
}

func (t *finalizeTask) Meta() task.Meta { return t.meta }
func (t *finalizeTask) Payload() any    { return FinalizePayload{UploadID: t.upload.ID} }
func (t *finalizeTask) Run(ctx context.Context, r task.Reporter) error {
	r.SetProgress(0, 1, "上传收尾中")
	if err := t.store.CompleteUpload(ctx, t.upload); err != nil {
		return err
	}
	if err := t.db.SetUploadCompleted(ctx, t.upload.ID); err != nil {
		return err
	}
	r.SetProgress(1, 1, "完成")
	return nil
}

// UploadCanceler 在清理某个残留上传会话时中止其来源侧机制（删缓冲/中止远端句柄）；由业务层注入。
type UploadCanceler func(ctx context.Context, u *gen.FileUpload)

// gcTask 周期性清理超期未收尾的上传会话：中止来源侧机制 + 删库记录。
type gcTask struct {
	store     UploadStore
	retention time.Duration
	interval  time.Duration
	cancel    UploadCanceler
}

// NewUploadGCTask 构造上传清理的定时单例任务。
func NewUploadGCTask(store UploadStore, interval, retention time.Duration, cancel UploadCanceler) task.Task {
	return &gcTask{store: store, retention: retention, interval: interval, cancel: cancel}
}

func (t *gcTask) Meta() task.Meta {
	return task.Meta{
		ID:       TaskTypeUploadGC,
		Type:     TaskTypeUploadGC,
		Kind:     task.KindScheduled,
		Resume:   task.ResumeAlways,
		Interval: t.interval,
		Title:    "上传会话清理",
	}
}
func (t *gcTask) Payload() any { return nil }
func (t *gcTask) Run(ctx context.Context, _ task.Reporter) error {
	gcStaleUploads(ctx, t.store, t.retention, t.cancel)
	return nil
}
