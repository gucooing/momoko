package task

import (
	"context"
	"fmt"
	"time"
)

// RetentionTaskType 是「任务记录清理」定时单例的类型键。
const RetentionTaskType = "task.cleanup"

// DefaultRetention 是已结束任务记录的保留时长：结束超过 30 天的记录由清理任务自动删除。
const DefaultRetention = 30 * 24 * time.Hour

// retentionTask 周期性删除结束时间超过保留期的任务记录。
type retentionTask struct {
	store     Store
	retention time.Duration
	interval  time.Duration
}

// NewRetentionTask 构造任务记录清理的定时单例（由业务层 EnsureSingleton）。
func (m *Manager) NewRetentionTask(retention, interval time.Duration) Task {
	return &retentionTask{store: m.store, retention: retention, interval: interval}
}

func (t *retentionTask) Meta() Meta {
	return Meta{
		ID:       RetentionTaskType,
		Type:     RetentionTaskType,
		Kind:     KindScheduled,
		Resume:   ResumeAlways,
		Interval: t.interval,
		Title:    "任务记录清理",
		Source:   "task",
	}
}

func (t *retentionTask) Payload() any { return nil }

func (t *retentionTask) Run(ctx context.Context, r Reporter) error {
	n, err := t.store.DeleteFinishedBefore(ctx, time.Now().Add(-t.retention))
	if err != nil {
		return err
	}
	r.Log(LogInfo, fmt.Sprintf("已清理 %d 条结束超过 %d 天的任务记录", n, int(t.retention.Hours()/24)))
	return nil
}
