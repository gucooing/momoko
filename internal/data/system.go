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
		if m.ID == menuId || m.ParentID == "" {
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
		Where(menu.IDIn(deleteIDs...)).
		Exec(ctx)
	return err
}
