package data

import (
	"context"

	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/menu"
	"momoko/internal/data/ent/gen/role"
	"momoko/internal/data/ent/gen/user"
)

type systemRepo struct {
	data *Data
}

func NewSystemRepo(data *Data) biz.SystemRepo {
	return &systemRepo{
		data: data,
	}
}

// GetMenusByRoleIds 返回这些角色关联菜单的并集（去重）。
func (s *systemRepo) GetMenusByRoleIds(ctx context.Context, roleIds []string) ([]*gen.Menu, error) {
	if len(roleIds) == 0 {
		return []*gen.Menu{}, nil
	}
	return s.data.db.Menu.Query().
		Where(menu.HasRolesWith(role.IDIn(roleIds...))).
		All(ctx)
}

func (s *systemRepo) GetMenus(ctx context.Context) ([]*gen.Menu, error) {
	return s.data.db.Menu.Query().
		All(ctx)
}

func (s *systemRepo) GetMenu(ctx context.Context, menuId string) (*gen.Menu, error) {
	return s.data.db.Menu.Query().
		Where(menu.IDEQ(menuId)).
		First(ctx)
}

func (s *systemRepo) CreateMenu(ctx context.Context, menuInfo *gen.Menu) (*gen.Menu, error) {
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
		SetHidden(menuInfo.Hidden).
		SetParentID(menuInfo.ParentID).
		Save(ctx)
}

func (s *systemRepo) UpdateMenu(ctx context.Context, menuInfo *gen.Menu) (*gen.Menu, error) {
	return s.data.db.Menu.UpdateOneID(menuInfo.ID).
		SetPath(menuInfo.Path).
		SetTitle(menuInfo.Title).
		SetPermission(menuInfo.Permission).
		SetOrder(menuInfo.Order).
		SetIcon(menuInfo.Icon).
		SetStatus(menuInfo.Status).
		SetHidden(menuInfo.Hidden).
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

func (s *systemRepo) GetRoles(ctx context.Context, page, pageSize int64, status *role.Status, name *string) ([]*gen.Role, int64, error) {
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
		Order(gen.Asc(role.FieldID)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return roles, int64(total), nil
}

// CountUsersByRoles 统计各角色下的用户数（角色数量有限，逐个计数）。
func (s *systemRepo) CountUsersByRoles(ctx context.Context, roleIds []string) (map[string]int64, error) {
	counts := make(map[string]int64, len(roleIds))
	for _, id := range roleIds {
		n, err := s.data.db.User.Query().Where(user.HasRolesWith(role.IDEQ(id))).Count(ctx)
		if err != nil {
			return nil, err
		}
		counts[id] = int64(n)
	}
	return counts, nil
}

// RoleMembers 返回各角色的前 limit 个成员（按用户 id 升序），用于角色列表的成员头像预览。
func (s *systemRepo) RoleMembers(ctx context.Context, roleIds []string, limit int) (map[string][]*gen.User, error) {
	out := make(map[string][]*gen.User, len(roleIds))
	for _, id := range roleIds {
		users, err := s.data.db.User.Query().
			Where(user.HasRolesWith(role.IDEQ(id))).
			Select(user.FieldID, user.FieldName, user.FieldUsername, user.FieldAvatar).
			Order(gen.Asc(user.FieldID)).
			Limit(limit).
			All(ctx)
		if err != nil {
			return nil, err
		}
		out[id] = users
	}
	return out, nil
}

func (s *systemRepo) GetRole(ctx context.Context, roleId string) (*gen.Role, error) {
	return s.data.db.Role.Query().
		Where(role.IDEQ(roleId)).
		WithMenus().
		First(ctx)
}

func (s *systemRepo) CreateRole(ctx context.Context, roleInfo *gen.Role, menuIds []string) (*gen.Role, error) {
	return s.data.db.Role.Create().
		SetID(roleInfo.ID).
		SetName(roleInfo.Name).
		SetDescription(roleInfo.Description).
		SetIsBuiltin(roleInfo.IsBuiltin).
		SetStatus(roleInfo.Status).
		AddMenuIDs(menuIds...).
		Save(ctx)
}

func (s *systemRepo) UpdateRole(ctx context.Context, roleInfo *gen.Role, menuIds []string) (*gen.Role, error) {
	return s.data.db.Role.UpdateOneID(roleInfo.ID).
		Where(role.IsBuiltinEQ(false)).
		SetName(roleInfo.Name).
		SetDescription(roleInfo.Description).
		SetStatus(roleInfo.Status).
		ClearMenus().
		AddMenuIDs(menuIds...).
		Save(ctx)
}

// DeleteRole 删除非内置角色；删除后没有任何角色的用户改为内置「无权限」角色。
func (s *systemRepo) DeleteRole(ctx context.Context, roleIds []string) error {
	tx, err := s.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	if _, err := tx.Role.Delete().
		Where(
			role.IDIn(roleIds...),
			role.IsBuiltinEQ(false),
		).
		Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.User.Update().
		Where(user.Not(user.HasRoles())).
		AddRoleIDs(noPermissionRoleID).
		Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
