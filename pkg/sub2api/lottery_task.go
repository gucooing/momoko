package sub2api

import (
	"context"
	"time"

	"momoko/pkg/task"
)

// 抽奖定时任务（单例）。仅 Interval 触发，启动不首跑。
const (
	TaskTypeLotteryTick = "sub2api.lottery.tick"
	lotteryTickInterval = time.Minute
)

type lotteryTickTask struct {
	svc *LotteryService
}

// NewLotteryTickTask 构造抽奖定时任务。
func NewLotteryTickTask(svc *LotteryService) task.Task {
	return &lotteryTickTask{svc: svc}
}

func (t *lotteryTickTask) Meta() task.Meta {
	return task.Meta{
		ID:       TaskTypeLotteryTick,
		Type:     TaskTypeLotteryTick,
		Kind:     task.KindScheduled,
		Resume:   task.ResumeAlways,
		Interval: lotteryTickInterval,
		Title:    "Sub2API 每日抽奖调度",
	}
}

func (t *lotteryTickTask) Payload() any { return nil }

// Run 仅定时器到点调用；内部 Tick 再判墙钟。重启路径不会进这里。
func (t *lotteryTickTask) Run(_ context.Context, _ task.Reporter) error {
	if t.svc == nil {
		return nil
	}
	t.svc.Tick()
	return nil
}
