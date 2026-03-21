package data

import (
	"context"

	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
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
func (i *instanceRepo) UpdateType(ctx context.Context, id string, name *string, isEnable *bool) (*ent.InstanceType, error) {
	update := i.data.db.InstanceType.
		UpdateOneID(id).
		Where(instancetype.IsSystem(false))

	if name != nil {
		update.SetName(*name)
	}
	if isEnable != nil {
		update.SetIsEnable(*isEnable)
	}

	return update.Save(ctx)
}

// DeleteType 删除实例类型
func (i *instanceRepo) DeleteType(ctx context.Context, id string) error {
	return i.data.db.InstanceType.DeleteOneID(id).
		Where(instancetype.IsSystem(false)).
		Exec(ctx)
}

// GetInstances 获取该账号下管辖的实例列表
func (i *instanceRepo) GetInstances(ctx context.Context, page, pageSize int64, userId string, keywords, types *string) ([]*ent.Instance, int64, error) {
	query := i.data.db.Instance.Query().Where(
		instance.Or(
			instance.HasUserWith(user.IDEQ(userId)),  // 主人
			instance.HasUsersWith(user.IDEQ(userId)), // 分配给当前用户
		),
	)

	if keywords != nil {
		query.Where(
			instance.Or(
				instance.NameContains(*keywords),
				instance.Tags(*keywords),
			),
		)
	}
	if types != nil {
		query.Where(
			instance.HasTypeWith(instancetype.NameEQ(*types)),
		)
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	instances, err := query.
		WithUser().WithUsers().WithType().
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		Order(ent.Asc(instance.FieldCreateTime)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return instances, int64(total), nil
}

// GetInstanceByUserID 获取该账号下管辖的实例
func (i *instanceRepo) GetInstanceByUserID(ctx context.Context, userId, instanceId string) (*ent.Instance, error) {
	query := i.data.db.Instance.Query().Where(
		instance.Or(
			instance.HasUserWith(user.IDEQ(userId)),  // 主人
			instance.HasUsersWith(user.IDEQ(userId)), // 分配给当前用户
		),
		instance.IDEQ(instanceId),
	)
	info, err := query.
		WithUser().
		WithUsers().
		WithType().
		Only(ctx)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func (i *instanceRepo) CreateInstance(ctx context.Context, req *v1.CreateInstanceRequest, userId string) (*ent.Instance, error) {
	create := i.data.db.Instance.Create().
		SetUserID(userId).
		SetID(uuid.NewString()).
		SetName(req.Name).
		SetRemark(req.Remark).
		SetTags(req.Tags).
		SetPath(req.InstancePath).
		SetStartCommand(req.StartCommand).
		SetStopCommand(req.StopCommand).
		SetAutoStart(req.AutoStart).
		SetTypeID(req.Type).
		SetEnv(req.Env)

	return create.Save(ctx)
}
