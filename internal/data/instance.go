package data

import (
	"context"

	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/instance"
	"momoko/internal/data/ent/gen/instancetype"
	"momoko/internal/data/ent/gen/user"
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
func (i *instanceRepo) GetTypes(ctx context.Context) ([]*gen.InstanceType, error) {
	return i.data.db.InstanceType.Query().
		Order(
			gen.Asc(instancetype.FieldCreateTime),
		).
		All(ctx)
}

// CreateType 创建实例类型
func (i *instanceRepo) CreateType(ctx context.Context, req *v1.CreateInstanceTypeRequest) (*gen.InstanceType, error) {
	create := i.data.db.InstanceType.Create().
		SetID(uuid.NewString()).
		SetName(req.Name).
		SetDescription(req.Description).
		SetStartCommand(req.StartCommand).
		SetStopCommand(req.StopCommand).
		SetQuickCommands(req.QuickCommands)
	if req.IsEnable != nil {
		create.SetIsEnable(*req.IsEnable)
	}

	return create.Save(ctx)
}

// UpdateType 更新实例类型
func (i *instanceRepo) UpdateType(ctx context.Context, req *v1.UpdateInstanceTypeRequest) (*gen.InstanceType, error) {
	update := i.data.db.InstanceType.
		UpdateOneID(req.Id).
		Where(instancetype.IsSystem(false))

	if req.Name != nil {
		update.SetName(*req.Name)
	}
	if req.IsEnable != nil {
		update.SetIsEnable(*req.IsEnable)
	}
	if req.Description != nil {
		update.SetDescription(*req.Description)
	}
	if req.StartCommand != nil {
		update.SetStartCommand(*req.StartCommand)
	}
	if req.StopCommand != nil {
		update.SetStopCommand(*req.StopCommand)
	}
	if req.QuickCommands != nil {
		update.SetQuickCommands(req.QuickCommands.GetItems())
	}

	return update.Save(ctx)
}

// CountTypeUsage 统计使用该类型的实例数(不区分用户，用于删除校验)
func (i *instanceRepo) CountTypeUsage(ctx context.Context, id string) (int, error) {
	return i.data.db.Instance.Query().
		Where(instance.HasTypeWith(instancetype.IDEQ(id))).
		Count(ctx)
}

// GetVisibleInstancesWithType 获取当前用户可见的全部实例(带类型)，用于类型使用统计
func (i *instanceRepo) GetVisibleInstancesWithType(ctx context.Context, userId string) ([]*gen.Instance, error) {
	return i.data.db.Instance.Query().
		Where(
			instance.Or(
				instance.HasUserWith(user.IDEQ(userId)),
				instance.HasUsersWith(user.IDEQ(userId)),
			),
		).
		WithType().
		Order(gen.Asc(instance.FieldCreateTime)).
		All(ctx)
}

// DeleteType 删除实例类型
func (i *instanceRepo) DeleteType(ctx context.Context, id string) error {
	return i.data.db.InstanceType.DeleteOneID(id).
		Where(instancetype.IsSystem(false)).
		Exec(ctx)
}

// GetInstances 获取该账号下管辖的实例列表
func (i *instanceRepo) GetInstances(ctx context.Context, page, pageSize int64, userId string, keywords, types *string) ([]*gen.Instance, int64, error) {
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
				instance.TagsContains(*keywords),
			),
		)
	}
	if types != nil {
		query.Where(
			instance.HasTypeWith(instancetype.Or(instancetype.IDEQ(*types), instancetype.NameEQ(*types))),
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
		Order(gen.Asc(instance.FieldCreateTime)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return instances, int64(total), nil
}

// GetInstanceByUserID 获取该账号下管辖的实例
func (i *instanceRepo) GetInstanceByUserID(ctx context.Context, userId, instanceId string) (*gen.Instance, error) {
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

func (i *instanceRepo) CreateInstance(ctx context.Context, req *v1.CreateInstanceRequest, userId string) (*gen.Instance, error) {
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
		SetEnv(req.Env).
		SetStopTimeout(req.StopTimeout).
		SetAutoRestart(req.AutoRestart)

	return create.Save(ctx)
}

func (i *instanceRepo) UpdateInstance(ctx context.Context, req *v1.UpdateInstanceRequest, userId string) (*gen.Instance, error) {
	update := i.data.db.Instance.UpdateOneID(req.Id).Where(
		instance.Or(
			instance.HasUserWith(user.IDEQ(userId)),  // 主人
			instance.HasUsersWith(user.IDEQ(userId)), // 分配给当前用户
		))

	if req.Name != nil {
		update.SetName(*req.Name)
	}
	if req.Remark != nil {
		update.SetRemark(*req.Remark)
	}
	if req.Tags != nil {
		update.SetTags(*req.Tags)
	}
	if req.Type != nil {
		update.SetTypeID(*req.Type)
	}
	if req.StartCommand != nil {
		update.SetStartCommand(*req.StartCommand)
	}
	if req.StopCommand != nil {
		update.SetStopCommand(*req.StopCommand)
	}
	if req.InstancePath != nil {
		update.SetPath(*req.InstancePath)
	}
	if req.AutoStart != nil {
		update.SetAutoStart(*req.AutoStart)
	}
	if req.Env != nil {
		update.SetEnv(req.Env)
	}
	if req.StopTimeout != nil {
		update.SetStopTimeout(*req.StopTimeout)
	}
	if req.AutoRestart != nil {
		update.SetAutoRestart(*req.AutoRestart)
	}
	if _, err := update.Save(ctx); err != nil {
		return nil, err
	}
	return i.GetInstanceByUserID(ctx, userId, req.Id)
}

func (i *instanceRepo) DeleteInstance(ctx context.Context, id, userId string) error {
	return i.data.db.Instance.DeleteOneID(id).Where(
		instance.Or(
			instance.HasUserWith(user.IDEQ(userId)), // 主人才能删
		)).Exec(ctx)
}

// GetAllAutoStartInstances 获取全部自动启动的实例
func (i *instanceRepo) GetAllAutoStartInstances(ctx context.Context) ([]*gen.Instance, error) {
	return i.data.db.Instance.Query().Where(instance.AutoStartEQ(true)).All(ctx)
}
