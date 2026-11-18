package data

import (
	"context"
	"fmt"

	"momoko/internal/data/ent/gen"
)

// 服务器的启动器图标 / 背景改为应用图标 / 应用背景：game_servers 的 launcher_icon、launcher_icon_key、
// launcher_background、launcher_background_key 改名为 app_*。自动迁移（WithDropColumn）会直接删掉旧列，
// 所以在 Schema.Create 之前把旧列改名，已上传的链接与对象 key 保留；旧列不存在（新库或已经改过名）时跳过。

var appMediaColumns = [][2]string{
	{"launcher_icon", "app_icon"},
	{"launcher_icon_key", "app_icon_key"},
	{"launcher_background", "app_background"},
	{"launcher_background_key", "app_background_key"},
}

// renameAppMediaColumns 把 game_servers 的 launcher_* 列改名为 app_*。
func renameAppMediaColumns(ctx context.Context, db *gen.Client) error {
	for _, c := range appMediaColumns {
		rows, err := db.QueryContext(ctx, "SELECT "+c[0]+" FROM game_servers WHERE 1 = 0")
		if err != nil {
			continue
		}
		_ = rows.Close()
		if _, err := db.ExecContext(ctx, "ALTER TABLE game_servers RENAME COLUMN "+c[0]+" TO "+c[1]); err != nil {
			return fmt.Errorf("rename game_servers.%s: %w", c[0], err)
		}
	}
	return nil
}
