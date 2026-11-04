package data

import "context"

// SaveAppearance 保存个人外观偏好（biz 已规范化并编码为 JSON）。
func (ur *userRepo) SaveAppearance(ctx context.Context, userId string, appearance string) error {
	return ur.data.db.User.UpdateOneID(userId).SetAppearance(appearance).Exec(ctx)
}
