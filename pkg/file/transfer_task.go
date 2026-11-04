package file

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/task"
)

// 文件传输任务类型键（注册到通用任务管理器）。
const (
	TaskTypeCopy = "file.copy"
	TaskTypeMove = "file.move"
)

// TransferPayload 是复制/移动任务的重建参数（落库，供开机/重试重建）。
type TransferPayload struct {
	SourceID string   `json:"source_id"`           // 系统来源 id，空=系统本地
	BasePath string   `json:"base_path,omitempty"` // 实例本地根；非空则目标为该根下的 LocalStore
	Paths    []string `json:"paths"`
	Target   string   `json:"target"`
}

// copyTask 在任意支持复制的来源上异步复制若干文件/目录。
type copyTask struct {
	meta    task.Meta
	copier  Copier
	payload TransferPayload
}

// moveTask 在任意支持移动的来源上异步移动若干文件/目录。
type moveTask struct {
	meta    task.Meta
	mover   Mover
	payload TransferPayload
}

// NewCopyTask 构造一个复制任务（copier 用于本次执行，payload 用于持久化/重建）。
func NewCopyTask(meta task.Meta, copier Copier, payload TransferPayload) task.Task {
	meta.Type = TaskTypeCopy
	meta.Kind = task.KindOneShot
	meta.Resume = task.ResumeNone
	meta.Total = int64(len(payload.Paths))
	meta.Target = payload.Target
	meta.Source = transferSource(payload)
	return &copyTask{meta: meta, copier: copier, payload: payload}
}

// NewMoveTask 构造一个移动任务。
func NewMoveTask(meta task.Meta, mover Mover, payload TransferPayload) task.Task {
	meta.Type = TaskTypeMove
	meta.Kind = task.KindOneShot
	meta.Resume = task.ResumeNone
	meta.Total = int64(len(payload.Paths))
	meta.Target = payload.Target
	meta.Source = transferSource(payload)
	return &moveTask{meta: meta, mover: mover, payload: payload}
}

// transferSource 返回传输任务的发起入口：实例文件（BasePath 非空）或文件管理。
func transferSource(payload TransferPayload) string {
	if payload.BasePath != "" {
		return "instance.file"
	}
	return "file"
}

func (t *copyTask) Meta() task.Meta { return t.meta }
func (t *copyTask) Payload() any    { return t.payload }
func (t *copyTask) Run(ctx context.Context, r task.Reporter) error {
	return runTransfer(ctx, r, t.payload, func(path string) []*v1.FileOperationResult {
		return t.copier.CopyToDir(ctx, []string{path}, t.payload.Target)
	})
}

func (t *moveTask) Meta() task.Meta { return t.meta }
func (t *moveTask) Payload() any    { return t.payload }
func (t *moveTask) Run(ctx context.Context, r task.Reporter) error {
	return runTransfer(ctx, r, t.payload, func(path string) []*v1.FileOperationResult {
		return t.mover.MoveToDir(ctx, []string{path}, t.payload.Target)
	})
}

// runTransfer 逐路径执行传输并上报结果，尊重 ctx 取消（尽力而为）。
// 有任意一项失败时任务记为失败（逐项结果仍保留），否则失败的复制/移动会显示为「成功」。
func runTransfer(ctx context.Context, r task.Reporter, payload TransferPayload, transfer func(string) []*v1.FileOperationResult) error {
	total := int64(len(payload.Paths))
	r.SetProgress(0, total, "执行中")
	failed := 0
	firstErr := ""
	for i, path := range payload.Paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		res := firstTransferResult(path, transfer(path))
		r.AppendResult(task.Result{Path: res.Path, Success: res.Success, Message: res.Message})
		if res.Success {
			r.Log(task.LogSuccess, res.Path)
		} else {
			r.Log(task.LogError, res.Path+"："+res.Message)
			failed++
			if firstErr == "" {
				firstErr = filepath.Base(res.Path) + "：" + res.Message
			}
		}
		r.SetProgress(int64(i+1), total, res.Path)
	}
	switch {
	case failed == 0:
		return nil
	case failed == 1:
		return errors.New(firstErr)
	default:
		return fmt.Errorf("%d 项失败，%s", failed, firstErr)
	}
}

func firstTransferResult(path string, results []*v1.FileOperationResult) *v1.FileOperationResult {
	if len(results) == 0 || results[0] == nil {
		return &v1.FileOperationResult{Path: path, Message: "文件操作未返回结果"}
	}
	return results[0]
}
