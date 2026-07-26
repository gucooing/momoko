package file

import (
	"context"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/task"
)

// 文件相关的异步任务类型键（注册到通用任务管理器）。
const (
	TaskTypeCopy           = "file.copy"
	TaskTypeMove           = "file.move"
	TaskTypeUploadFinalize = "file.upload_finalize"
	TaskTypeUploadGC       = "file.upload_gc"
)

// TransferPayload 是复制/移动任务的重建参数（落库，供开机/重试重建）。
type TransferPayload struct {
	SourceID string   `json:"source_id"`           // 系统来源 id，空=系统本地
	BasePath string   `json:"base_path,omitempty"` // 受限本地根（实例工作目录）；非空则目标为该根下的受限来源
	Paths    []string `json:"paths"`
	Target   string   `json:"target"`
}

// FinalizePayload 是上传收尾任务的重建参数。
type FinalizePayload struct {
	UploadID string `json:"upload_id"`
}

// UploadCompleter 是收尾任务需要的最小持久化能力。
type UploadCompleter interface {
	SetUploadCompleted(ctx context.Context, id string) error
}

// transferTask 在任意支持复制/移动的来源上异步搬运若干文件/目录。
type transferTask struct {
	meta    task.Meta
	payload TransferPayload
	run     func(ctx context.Context, path string) []*v1.FileOperationResult
}

// NewCopyTask 构造一个复制任务（copier 用于本次执行，payload 用于持久化/重建）。
func NewCopyTask(meta task.Meta, copier Copier, payload TransferPayload) task.Task {
	return newTransferTask(TaskTypeCopy, meta, payload, func(ctx context.Context, path string) []*v1.FileOperationResult {
		return copier.CopyToDir(ctx, []string{path}, payload.Target)
	})
}

// NewMoveTask 构造一个移动任务。
func NewMoveTask(meta task.Meta, mover Mover, payload TransferPayload) task.Task {
	return newTransferTask(TaskTypeMove, meta, payload, func(ctx context.Context, path string) []*v1.FileOperationResult {
		return mover.MoveToDir(ctx, []string{path}, payload.Target)
	})
}

func newTransferTask(typ string, meta task.Meta, payload TransferPayload, run func(context.Context, string) []*v1.FileOperationResult) task.Task {
	meta.Type = typ
	meta.Kind = task.KindOneShot
	// 搬运不可重入：进程重启时无法确定中断点，重跑可能产生重复副本，故一律标失败由用户重试。
	meta.Resume = task.ResumeNone
	meta.Total = int64(len(payload.Paths))
	return &transferTask{meta: meta, payload: payload, run: run}
}

func (t *transferTask) Meta() task.Meta { return t.meta }
func (t *transferTask) Payload() any    { return t.payload }

func (t *transferTask) Run(ctx context.Context, r task.Reporter) error {
	r.SetProgress(0, int64(len(t.payload.Paths)), "执行中")
	for _, p := range t.payload.Paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		res := firstResult(p, t.run(ctx, p))
		r.AppendResult(task.Result{Path: res.Path, Success: res.Success, Message: res.Message})
	}
	return nil
}

func firstResult(path string, results []*v1.FileOperationResult) *v1.FileOperationResult {
	if len(results) == 0 || results[0] == nil {
		return &v1.FileOperationResult{Path: path, Message: "文件操作未返回结果"}
	}
	return results[0]
}

// finalizeTask 把一次上传的本地缓冲整流推送到远端来源并标记完成。
type finalizeTask struct {
	meta   task.Meta
	store  Store
	upload *Upload
	db     UploadCompleter
}

// NewFinalizeTask 构造一个上传收尾任务。
// 收尾是幂等的（重复推送同一份缓冲得到同一结果），故允许重启后续做。
func NewFinalizeTask(meta task.Meta, store Store, upload *Upload, db UploadCompleter) task.Task {
	meta.Type = TaskTypeUploadFinalize
	meta.Kind = task.KindOneShot
	meta.Resume = task.ResumeRerun
	meta.Total = 1
	return &finalizeTask{meta: meta, store: store, upload: upload, db: db}
}

func (t *finalizeTask) Meta() task.Meta { return t.meta }
func (t *finalizeTask) Payload() any    { return FinalizePayload{UploadID: t.upload.ID()} }

func (t *finalizeTask) Run(ctx context.Context, r task.Reporter) error {
	r.SetProgress(0, 1, "上传收尾中")
	if err := t.store.CompleteUpload(ctx, t.upload); err != nil {
		return err
	}
	if err := t.db.SetUploadCompleted(ctx, t.upload.ID()); err != nil {
		return err
	}
	r.SetProgress(1, 1, "完成")
	return nil
}
