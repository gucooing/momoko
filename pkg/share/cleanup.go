package share

import (
	"context"
	"time"

	"momoko/pkg/task"
)

// TaskTypeCleanup 是过期分享清理任务的类型键。
const TaskTypeCleanup = "share.cleanup"

// ExpiredRetention 是分享过期后保留的时长：过期满 30 天的分享（连同其下载记录）由后台任务自动删除。
const ExpiredRetention = 30 * 24 * time.Hour

// CleanupStore 是过期分享清理需要的数据访问能力（由数据层实现）。
type CleanupStore interface {
	// DeleteSharesExpiredBefore 删除过期时间早于 before 的分享及其下载记录，返回删除条数。
	DeleteSharesExpiredBefore(ctx context.Context, before time.Time) (int, error)
}

// cleanupTask 周期性删除过期超过 retention 的分享。
type cleanupTask struct {
	store     CleanupStore
	interval  time.Duration
	retention time.Duration
}

// NewCleanupTask 构造过期分享清理的定时单例任务。
func NewCleanupTask(store CleanupStore, interval, retention time.Duration) task.Task {
	return &cleanupTask{store: store, interval: interval, retention: retention}
}

func (t *cleanupTask) Meta() task.Meta {
	return task.Meta{
		ID:       TaskTypeCleanup,
		Type:     TaskTypeCleanup,
		Kind:     task.KindScheduled,
		Resume:   task.ResumeAlways,
		Interval: t.interval,
		Title:    "过期分享清理",
	}
}

func (t *cleanupTask) Payload() any { return nil }

func (t *cleanupTask) Run(ctx context.Context, _ task.Reporter) error {
	_, err := t.store.DeleteSharesExpiredBefore(ctx, time.Now().Add(-t.retention))
	return err
}
