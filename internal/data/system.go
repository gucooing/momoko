package data

import (
	"context"

	"momoko/internal/biz"
	"momoko/internal/data/ent"
	"momoko/internal/data/ent/menu"
	"momoko/internal/data/ent/role"
)

type systemRepo struct {
	data *Data
}

func NewSystemRepo(data *Data) biz.SystemRepo {
	return &systemRepo{
		data: data,
	}
}

func (s *systemRepo) GetMenusByRoleId(ctx context.Context, roleId string) ([]*ent.Menu, error) {
	return s.data.db.Role.Query().
		Where(role.IDEQ(roleId)).
		QueryMenus().
		All(ctx)
}

func (s *systemRepo) GetMenus(ctx context.Context) ([]*ent.Menu, error) {
	return s.data.db.Menu.Query().
		All(ctx)
}

func (s *systemRepo) GetMenu(ctx context.Context, menuId string) (*ent.Menu, error) {
	return s.data.db.Menu.Query().
		Where(menu.IDEQ(menuId)).
		First(ctx)
}

func (s *systemRepo) CreateMenu(ctx context.Context, menuInfo *ent.Menu) (*ent.Menu, error) {
	return s.data.db.Menu.Create().
		SetID(menuInfo.ID).
		SetType(menuInfo.Type).
		SetPath(menuInfo.Path).
		SetTitle(menuInfo.Title).
		SetPermission(menuInfo.Permission).
		SetOrder(menuInfo.Order).
		SetIcon(menuInfo.Icon).
		SetIsSystem(menuInfo.IsSystem).
		SetStatus(menuInfo.Status).
		SetParentID(menuInfo.ParentID).
		Save(ctx)
}

func (s *systemRepo) UpdateMenu(ctx context.Context, menuInfo *ent.Menu) (*ent.Menu, error) {
	return s.data.db.Menu.UpdateOneID(menuInfo.ID).
		SetPath(menuInfo.Path).
		SetTitle(menuInfo.Title).
		SetPermission(menuInfo.Permission).
		SetOrder(menuInfo.Order).
		SetIcon(menuInfo.Icon).
		SetStatus(menuInfo.Status).
		Save(ctx)
}

func (s *systemRepo) DeleteMenu(ctx context.Context, menuId string) error {
	menus, err := s.data.db.Menu.Query().
		Select(menu.FieldID, menu.FieldParentID).
		All(ctx)
	if err != nil {
		return err
	}
	parentChildren := make(map[string][]string, len(menus))
	for _, m := range menus {
		if m.ID == menuId || m.ParentID == "" || m.IsSystem {
			continue
		}
		parentChildren[m.ParentID] = append(parentChildren[m.ParentID], m.ID)
	}
	visited := map[string]struct{}{menuId: {}}
	queue := []string{menuId}
	deleteIDs := make([]string, 0, len(menus))
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		deleteIDs = append(deleteIDs, current)
		for _, childID := range parentChildren[current] {
			if _, ok := visited[childID]; ok {
				continue
			}
			visited[childID] = struct{}{}
			queue = append(queue, childID)
		}
	}
	_, err = s.data.db.Menu.Delete().
		Where(
			menu.IDIn(deleteIDs...),
			menu.IsSystemEQ(false),
		).
		Exec(ctx)
	return err
}

func (s *systemRepo) GetRoles(ctx context.Context, page, pageSize int64, status *role.Status, name *string) ([]*ent.Role, int64, error) {
	query := s.data.db.Role.Query()
	if status != nil {
		query = query.Where(role.StatusEQ(*status))
	}
	if name != nil {
		query = query.Where(role.NameContains(*name))
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	roles, err := query.
		WithMenus().
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		Order(ent.Asc(role.FieldID)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return roles, int64(total), nil
}

func (s *systemRepo) GetRole(ctx context.Context, roleId string) (*ent.Role, error) {
	return s.data.db.Role.Query().
		Where(role.IDEQ(roleId)).
		WithMenus().
		First(ctx)
}

func (s *systemRepo) CreateRole(ctx context.Context, roleInfo *ent.Role, menuIds []string) (*ent.Role, error) {
	return s.data.db.Role.Create().
		SetID(roleInfo.ID).
		SetName(roleInfo.Name).
		SetDescription(roleInfo.Description).
		SetIsBuiltin(roleInfo.IsBuiltin).
		SetStatus(roleInfo.Status).
		AddMenuIDs(menuIds...).
		Save(ctx)
}

func (s *systemRepo) UpdateRole(ctx context.Context, roleInfo *ent.Role, menuIds []string) (*ent.Role, error) {
	return s.data.db.Role.UpdateOneID(roleInfo.ID).
		Where(role.IsBuiltinEQ(false)).
		SetName(roleInfo.Name).
		SetDescription(roleInfo.Description).
		SetStatus(roleInfo.Status).
		ClearMenus().
		AddMenuIDs(menuIds...).
		Save(ctx)
}

func (s *systemRepo) DeleteRole(ctx context.Context, roleIds []string) error {
	_, err := s.data.db.Role.Delete().
		Where(
			role.IDIn(roleIds...),
			role.IsBuiltinEQ(false),
		).
		Exec(ctx)
	return err
}
