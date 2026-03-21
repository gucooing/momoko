package data

import (
	"context"

	"github.com/google/uuid"

	"momoko/internal/data/ent"
	"momoko/internal/data/ent/instance"
	"momoko/internal/data/ent/user"
)

type InstanceRepo struct {
	data *Data
}

func NewInstanceRepo(data *Data) *InstanceRepo {
	return &InstanceRepo{
		data: data,
	}
}

// GetTypes 获取全部实例类型
func (i *InstanceRepo) GetTypes(ctx context.Context) ([]*ent.InstanceType, error) {
	return i.data.db.InstanceType.Query().All(ctx)
}

// CreateType 创建实例类型
func (i *InstanceRepo) CreateType(ctx context.Context, name string) (*ent.InstanceType, error) {
	create := i.data.db.InstanceType.Create().
		SetID(uuid.NewString()).
		SetName(name)

	return create.Save(ctx)
}

// UpdateType 更新实例类型
func (i *InstanceRepo) UpdateType(ctx context.Context, id string, name string) (*ent.InstanceType, error) {
	update := i.data.db.InstanceType.UpdateOneID(id).
		SetName(name)

	return update.Save(ctx)
}

// DeleteType 删除实例类型
func (i *InstanceRepo) DeleteType(ctx context.Context, id string) error {
	return i.data.db.InstanceType.DeleteOneID(id).Exec(ctx)
}

// GetInstances 获取该账号下管辖的实例
func (i *InstanceRepo) GetInstances(ctx context.Context, userId string) ([]*ent.Instance, error) {
	query := i.data.db.Instance.Query().Where(
		instance.Or(
			instance.UserIDEQ(userId),                // 主人
			instance.HasUsersWith(user.IDEQ(userId)), // 分配给当前用户
		),
	).WithUsers().WithType()

	return query.All(ctx)
}
