package biz

import (
	"context"
	"encoding/json"
	"slices"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
)

// 个人外观偏好：只对自己生效，跟随账号同步到所有设备（存为用户表 appearance 列的 JSON）。

var (
	appearancePalettes = []string{"sakura", "mint", "volt", "grape", "sunset", "classic"}
	appearanceModes    = []string{"light", "dark", "auto"}
	appearanceRadii    = []string{"round", "standard"}
)

type appearanceJSON struct {
	Palette      string `json:"palette"`
	Mode         string `json:"mode"`
	Compact      bool   `json:"compact"`
	ReduceMotion bool   `json:"reduce_motion"`
	Radius       string `json:"radius"`
	ShowTabs     bool   `json:"show_tabs"`
}

func pickOr(v string, allowed []string, fallback string) string {
	if slices.Contains(allowed, v) {
		return v
	}
	return fallback
}

// normalizeAppearance 把未知取值落回默认值，保证存下的偏好前端一定能套用。
func normalizeAppearance(a *v1.UserAppearance) *v1.UserAppearance {
	return &v1.UserAppearance{
		Palette:      pickOr(a.GetPalette(), appearancePalettes, "sakura"),
		Mode:         pickOr(a.GetMode(), appearanceModes, "auto"),
		Compact:      a.GetCompact(),
		ReduceMotion: a.GetReduceMotion(),
		Radius:       pickOr(a.GetRadius(), appearanceRadii, "round"),
		ShowTabs:     a.GetShowTabs(),
	}
}

// EncodeAppearance 规范化外观偏好并编码为存储用的 JSON。
func EncodeAppearance(a *v1.UserAppearance) string {
	n := normalizeAppearance(a)
	b, _ := json.Marshal(appearanceJSON{
		Palette:      n.Palette,
		Mode:         n.Mode,
		Compact:      n.Compact,
		ReduceMotion: n.ReduceMotion,
		Radius:       n.Radius,
		ShowTabs:     n.ShowTabs,
	})
	return string(b)
}

// DecodeAppearance 解析存储的外观偏好；从未保存过（或内容损坏）返回 nil。
func DecodeAppearance(raw string) *v1.UserAppearance {
	if raw == "" {
		return nil
	}
	var j appearanceJSON
	if err := json.Unmarshal([]byte(raw), &j); err != nil {
		return nil
	}
	return normalizeAppearance(&v1.UserAppearance{
		Palette:      j.Palette,
		Mode:         j.Mode,
		Compact:      j.Compact,
		ReduceMotion: j.ReduceMotion,
		Radius:       j.Radius,
		ShowTabs:     j.ShowTabs,
	})
}

// UpdateMyAppearance 规范化并保存当前用户的外观偏好，返回实际保存的值。
func (u *UserUsecase) UpdateMyAppearance(ctx context.Context, userId string, a *v1.UserAppearance) (*v1.UserAppearance, error) {
	n := normalizeAppearance(a)
	if err := u.user.SaveAppearance(ctx, userId, EncodeAppearance(n)); err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrAdminNotFound
		}
		return nil, ErrSystem(err)
	}
	return n, nil
}

// MeAppearance 返回当前用户保存的外观偏好（从未保存过为 nil）。
func (u *UserUsecase) MeAppearance(ctx context.Context, userId string) (*v1.UserAppearance, error) {
	row, err := u.user.FindByID(ctx, userId)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrAdminNotFound
		}
		return nil, ErrSystem(err)
	}
	return DecodeAppearance(row.Appearance), nil
}
