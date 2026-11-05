package data

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/instance"
	"momoko/internal/data/ent/gen/instancetype"
	"momoko/internal/data/ent/gen/operationlog"
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

// DeleteType 删除实例类型：在用的应用改为未分类，返回受影响的应用数
func (i *instanceRepo) DeleteType(ctx context.Context, id string) (int, error) {
	tx, err := i.data.db.Tx(ctx)
	if err != nil {
		return 0, err
	}
	n, err := tx.Instance.Update().
		Where(instance.HasTypeWith(instancetype.IDEQ(id))).
		ClearType().
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if err := tx.InstanceType.DeleteOneID(id).
		Where(instancetype.IsSystem(false)).
		Exec(ctx); err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	return n, tx.Commit()
}

// GetInstances 获取该账号下管辖的、符合关键字/类型条件的全部实例(运行状态在内存里，分页由 biz 完成)
func (i *instanceRepo) GetInstances(ctx context.Context, userId string, keywords, types *string) ([]*gen.Instance, error) {
	query := i.data.db.Instance.Query().Where(
		instance.Or(
			instance.HasUserWith(user.IDEQ(userId)),  // 主人
			instance.HasUsersWith(user.IDEQ(userId)), // 分配给当前用户
		),
	)

	if keywords != nil {
		kw := strings.TrimSpace(*keywords)
		query.Where(
			instance.Or(
				instance.NameContainsFold(kw),
				instance.TagsContainsFold(kw),
				instance.RemarkContainsFold(kw),
				instance.PathContainsFold(kw),
			),
		)
	}
	if types != nil {
		query.Where(
			instance.HasTypeWith(instancetype.Or(instancetype.IDEQ(*types), instancetype.NameEQ(*types))),
		)
	}

	return query.
		WithUser().WithUsers().WithType().
		Order(gen.Asc(instance.FieldCreateTime)).
		All(ctx)
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

func (i *instanceRepo) CreateInstance(ctx context.Context, req *v1.CreateInstanceRequest, userId string, env biz.InstanceEnv) (*gen.Instance, error) {
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
		SetEnv(env.Lines).
		SetSecretEnvKeys(env.SecretKeys).
		SetStopTimeout(req.StopTimeout).
		SetAutoRestart(req.AutoRestart)
	if t := strings.TrimSpace(req.Type); t != "" {
		create.SetTypeID(t)
	}

	item, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	return i.GetInstanceByUserID(ctx, userId, item.ID)
}

func (i *instanceRepo) UpdateInstance(ctx context.Context, current *gen.Instance, req *v1.UpdateInstanceRequest, userId string, env *biz.InstanceEnv) (*gen.Instance, error) {
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
		if t := strings.TrimSpace(*req.Type); t != "" {
			update.SetTypeID(t)
		} else {
			update.ClearType()
		}
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
	if env != nil {
		update.SetEnv(env.Lines).SetSecretEnvKeys(env.SecretKeys)
	}
	if req.StopTimeout != nil {
		update.SetStopTimeout(*req.StopTimeout)
	}
	if req.AutoRestart != nil {
		update.SetAutoRestart(*req.AutoRestart)
	}
	if req.Maintenance != nil {
		if *req.Maintenance {
			update.SetMaintenance(true)
			if !current.Maintenance || current.MaintenanceTime == nil {
				update.SetMaintenanceTime(time.Now())
			}
		} else {
			update.SetMaintenance(false).ClearMaintenanceTime()
		}
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

// GetAllAutoStartInstances 获取全部自动启动的实例(维护中的除外)
func (i *instanceRepo) GetAllAutoStartInstances(ctx context.Context) ([]*gen.Instance, error) {
	return i.data.db.Instance.Query().
		Where(instance.AutoStartEQ(true), instance.MaintenanceEQ(false)).
		All(ctx)
}

// GetInstanceNames 按 id 批量查实例名称
func (i *instanceRepo) GetInstanceNames(ctx context.Context, ids []string) (map[string]string, error) {
	items, err := i.data.db.Instance.Query().
		Where(instance.IDIn(ids...)).
		Select(instance.FieldID, instance.FieldName).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(items))
	for _, item := range items {
		out[item.ID] = item.Name
	}
	return out, nil
}

// SetLastStopTime 记录最近一次进程退出时间(不改动「更新时间」)
func (i *instanceRepo) SetLastStopTime(ctx context.Context, id string, t time.Time) error {
	item, err := i.data.db.Instance.Get(ctx, id)
	if err != nil {
		return err
	}
	return i.data.db.Instance.UpdateOne(item).
		SetLastStopTime(t).
		SetUpdateTime(item.UpdateTime).
		Exec(ctx)
}

// ListInstanceOperationLogs 按实例 id 查最近的应用相关操作记录(请求路径或请求参数里带该 id)
func (i *instanceRepo) ListInstanceOperationLogs(ctx context.Context, instanceID string, limit int) ([]*gen.OperationLog, error) {
	return i.data.db.OperationLog.Query().
		Where(
			operationlog.OperationTypeHasPrefix("OperationTypeInstance"),
			operationlog.Not(operationlog.OperationTypeHasPrefix("OperationTypeInstanceType")),
			operationlog.Or(
				operationlog.PathContains(instanceID),
				operationlog.DetailContains(instanceID),
			),
		).
		WithUser().
		Order(gen.Desc(operationlog.FieldOperationTime)).
		Limit(limit).
		All(ctx)
}
