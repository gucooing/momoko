package data

import (
	"context"
	"fmt"

	"momoko/internal/data/ent/gen"
	entuser "momoko/internal/data/ent/gen/user"
)

// 用户角色由「users.user_role 单角色外键」改为多对多 user_roles 表。
// 自动迁移（WithDropColumn）会直接删掉旧列，所以在 Schema.Create 之前读出旧的 用户 → 角色 对应关系，
// 建表后写回 user_roles；旧列不存在（新库或已经迁移过）时读取失败，直接跳过。

// captureLegacyUserRoles 读取旧列 users.user_role；读不到返回 nil。
func captureLegacyUserRoles(ctx context.Context, db *gen.Client) map[string]string {
	rows, err := db.QueryContext(ctx, "SELECT id, user_role FROM users WHERE user_role IS NOT NULL AND user_role <> ''")
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var userID, roleID string
		if err := rows.Scan(&userID, &roleID); err == nil && userID != "" && roleID != "" {
			out[userID] = roleID
		}
	}
	return out
}

// restoreLegacyUserRoles 把旧的单角色写入 user_roles（只处理还没有任何角色的用户）。
func restoreLegacyUserRoles(ctx context.Context, db *gen.Client, legacy map[string]string) error {
	for userID, roleID := range legacy {
		has, err := db.User.Query().Where(entuser.IDEQ(userID), entuser.HasRoles()).Exist(ctx)
		if err != nil {
			return fmt.Errorf("restore user roles: %w", err)
		}
		if has {
			continue
		}
		if err := db.User.UpdateOneID(userID).AddRoleIDs(roleID).Exec(ctx); err != nil &&
			!gen.IsNotFound(err) && !gen.IsConstraintError(err) {
			return fmt.Errorf("restore user roles: %w", err)
		}
	}
	return nil
}
