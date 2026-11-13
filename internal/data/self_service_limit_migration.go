package data

import (
	"context"
	"fmt"

	"momoko/internal/data/ent/gen"
)

// 玩家自助的次数由「每天次数 daily_limit + 间隔 cooldown_sec」改为「每 period_days 天 use_limit 次」（0 点刷新），
// 间隔不再使用。自动迁移（WithDropColumn）会直接删掉旧列，所以在 Schema.Create 之前读出旧的次数，建表后写回 use_limit
// （刷新天数默认 1，与原来的每天次数相同）；旧列不存在（新库或已经迁移过）时读取失败，直接跳过。

// captureLegacySelfServiceLimits 读取旧列 gm_self_services.daily_limit；读不到返回 nil。
func captureLegacySelfServiceLimits(ctx context.Context, db *gen.Client) map[string]int {
	rows, err := db.QueryContext(ctx, "SELECT id, daily_limit FROM gm_self_services WHERE daily_limit > 0")
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make(map[string]int)
	for rows.Next() {
		var id string
		var limit int
		if err := rows.Scan(&id, &limit); err == nil && id != "" {
			out[id] = limit
		}
	}
	return out
}

// restoreLegacySelfServiceLimits 把旧的每天次数写入 use_limit。
func restoreLegacySelfServiceLimits(ctx context.Context, db *gen.Client, legacy map[string]int) error {
	for id, limit := range legacy {
		err := db.GmSelfService.UpdateOneID(id).
			SetUseLimit(limit).
			SetPeriodDays(1).
			Exec(ctx)
		if err != nil && !gen.IsNotFound(err) {
			return fmt.Errorf("restore self-service limits: %w", err)
		}
	}
	return nil
}
