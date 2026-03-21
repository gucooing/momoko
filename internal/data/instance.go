package data

import (
	"context"

	"github.com/google/uuid"

	"momoko/internal/biz"
	"momoko/internal/data/ent"
	"momoko/internal/data/ent/instance"
	"momoko/internal/data/ent/instancetype"
	"momoko/internal/data/ent/user"
)

type instanceRepo struct {
	data *Data
}

func NewInstanceRepo(data *Data) biz.InstanceRepo {
	return &instanceRepo{
		data: data,
	}
}

// GetTypes 获取全部实例类型
func (i *instanceRepo) GetTypes(ctx context.Context) ([]*ent.InstanceType, error) {
	return i.data.db.InstanceType.Query().
		Order(
			ent.Asc(instancetype.FieldCreateTime),
		).
		All(ctx)
}

// CreateType 创建实例类型
func (i *instanceRepo) CreateType(ctx context.Context, name string) (*ent.InstanceType, error) {
	create := i.data.db.InstanceType.Create().
		SetID(uuid.NewString()).
		SetName(name)

	return create.Save(ctx)
}

// UpdateType 更新实例类型
func (i *instanceRepo) UpdateType(ctx context.Context, id string, name *string) (*ent.InstanceType, error) {
	update := i.data.db.InstanceType.UpdateOneID(id)

	if name != nil {
		update.SetName(*name)
	}

	return update.Save(ctx)
}

// DeleteType 删除实例类型
func (i *instanceRepo) DeleteType(ctx context.Context, id string) error {
	return i.data.db.InstanceType.DeleteOneID(id).Exec(ctx)
}

// GetInstances 获取该账号下管辖的实例
func (i *instanceRepo) GetInstances(ctx context.Context, userId string) ([]*ent.Instance, error) {
	query := i.data.db.Instance.Query().Where(
		instance.Or(
			instance.UserIDEQ(userId),                // 主人
			instance.HasUsersWith(user.IDEQ(userId)), // 分配给当前用户
		),
	).WithUsers().WithType()

	return query.All(ctx)
}
