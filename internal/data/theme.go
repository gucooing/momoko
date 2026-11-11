package data

import (
	"context"
	"time"

	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	entthemeasset "momoko/internal/data/ent/gen/themeasset"
	entuser "momoko/internal/data/ent/gen/user"
	"momoko/internal/data/ent/gen/usertheme"
)

type themeRepo struct {
	data *Data
}

func NewThemeRepo(data *Data) biz.ThemeRepo {
	return &themeRepo{data: data}
}

func (r *themeRepo) ListThemes(ctx context.Context, userID string) ([]*gen.UserTheme, error) {
	return r.data.db.UserTheme.Query().
		Where(usertheme.UserIDEQ(userID)).
		Order(gen.Desc(usertheme.FieldCreateTime), gen.Desc(usertheme.FieldID)).
		All(ctx)
}

func (r *themeRepo) GetTheme(ctx context.Context, userID, id string) (*gen.UserTheme, error) {
	return r.data.db.UserTheme.Query().
		Where(usertheme.IDEQ(id), usertheme.UserIDEQ(userID)).
		Only(ctx)
}

func (r *themeRepo) FindThemeByName(ctx context.Context, userID, name string) (*gen.UserTheme, error) {
	row, err := r.data.db.UserTheme.Query().
		Where(usertheme.UserIDEQ(userID), usertheme.NameEQ(name)).
		Order(gen.Desc(usertheme.FieldUpdateTime)).
		First(ctx)
	if gen.IsNotFound(err) {
		return nil, nil
	}
	return row, err
}

func (r *themeRepo) CountThemes(ctx context.Context, userID string) (int, error) {
	return r.data.db.UserTheme.Query().Where(usertheme.UserIDEQ(userID)).Count(ctx)
}

func (r *themeRepo) CreateTheme(ctx context.Context, t *gen.UserTheme) (*gen.UserTheme, error) {
	return r.data.db.UserTheme.Create().
		SetID(t.ID).
		SetUserID(t.UserID).
		SetName(t.Name).
		SetAuthor(t.Author).
		SetVersion(t.Version).
		SetDescription(t.Description).
		SetSource(t.Source).
		SetContent(t.Content).
		Save(ctx)
}

func (r *themeRepo) UpdateTheme(ctx context.Context, t *gen.UserTheme) (*gen.UserTheme, error) {
	n, err := r.data.db.UserTheme.Update().
		Where(usertheme.IDEQ(t.ID), usertheme.UserIDEQ(t.UserID)).
		SetName(t.Name).
		SetAuthor(t.Author).
		SetVersion(t.Version).
		SetDescription(t.Description).
		SetSource(t.Source).
		SetContent(t.Content).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, &gen.NotFoundError{}
	}
	return r.GetTheme(ctx, t.UserID, t.ID)
}

func (r *themeRepo) DeleteTheme(ctx context.Context, userID, id string) error {
	n, err := r.data.db.UserTheme.Delete().
		Where(usertheme.IDEQ(id), usertheme.UserIDEQ(userID)).
		Exec(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return &gen.NotFoundError{}
	}
	return nil
}

func (r *themeRepo) CreateAsset(ctx context.Context, a *gen.ThemeAsset) (*gen.ThemeAsset, error) {
	return r.data.db.ThemeAsset.Create().
		SetID(a.ID).
		SetUserID(a.UserID).
		SetKind(a.Kind).
		SetName(a.Name).
		SetSize(a.Size).
		SetOriginalSize(a.OriginalSize).
		SetWidth(a.Width).
		SetHeight(a.Height).
		Save(ctx)
}

func (r *themeRepo) GetAssets(ctx context.Context, ids []string) ([]*gen.ThemeAsset, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	return r.data.db.ThemeAsset.Query().Where(entthemeasset.IDIn(ids...)).All(ctx)
}

func (r *themeRepo) ListStaleAssets(ctx context.Context, userID string, before time.Time, limit int) ([]*gen.ThemeAsset, error) {
	return r.data.db.ThemeAsset.Query().
		Where(entthemeasset.UserIDEQ(userID), entthemeasset.CreateTimeLT(before)).
		Order(gen.Asc(entthemeasset.FieldCreateTime)).
		Limit(limit).
		All(ctx)
}

// AssetReferenced 判断资源是否仍被任何用户的主题或个人外观引用（资源 id 是随机串，按子串匹配即可）。
func (r *themeRepo) AssetReferenced(ctx context.Context, id string) (bool, error) {
	ok, err := r.data.db.UserTheme.Query().Where(usertheme.ContentContains(id)).Exist(ctx)
	if err != nil || ok {
		return ok, err
	}
	return r.data.db.User.Query().Where(entuser.AppearanceContains(id)).Exist(ctx)
}

func (r *themeRepo) DeleteAsset(ctx context.Context, id string) error {
	return r.data.db.ThemeAsset.DeleteOneID(id).Exec(ctx)
}
