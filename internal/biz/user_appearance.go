package biz

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/themeasset"
)

// 个人外观偏好：只对自己生效，跟随账号同步到所有设备（存为用户表 appearance 列的 JSON）。

var (
	appearanceModes = []string{"light", "dark", "auto"}
	appearanceRadii = []string{"round", "standard"}
)

type appearanceJSON struct {
	// Theme 空表示跟随全站默认；内置配色键或「我的主题」id
	Theme        string `json:"theme"`
	Mode         string `json:"mode"`
	Compact      bool   `json:"compact"`
	ReduceMotion bool   `json:"reduce_motion"`
	Radius       string `json:"radius"`
	ShowTabs     bool   `json:"show_tabs"`
	// Background 外观页上改过、还没存进主题的背景（v1.ThemeBackground 的 protojson）
	Background json.RawMessage `json:"background,omitempty"`
}

func pickOr(v string, allowed []string, fallback string) string {
	if slices.Contains(allowed, v) {
		return v
	}
	return fallback
}

// normalizeAppearanceTheme 不查库的主题键校验：空 / 内置配色 / 形如主题 id 的值保留，其余落回 sakura。
func normalizeAppearanceTheme(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || IsBuiltinTheme(v) {
		return v
	}
	if len(v) == 36 {
		if _, err := uuid.Parse(v); err == nil {
			return v
		}
	}
	return "sakura"
}

// syntacticAssetResolver 只按格式校验资源引用（用于解析已存的偏好，存入时已按归属校验过）。
func syntacticAssetResolver(kind, ref string) (string, bool) {
	return ref, themeasset.KindOf(ref) == kind
}

// normalizeAppearance 把未知取值落回默认值，保证存下的偏好前端一定能套用（不查库）。
func normalizeAppearance(a *v1.UserAppearance) *v1.UserAppearance {
	out := &v1.UserAppearance{
		Theme:        normalizeAppearanceTheme(a.GetTheme()),
		Mode:         pickOr(a.GetMode(), appearanceModes, "auto"),
		Compact:      a.GetCompact(),
		ReduceMotion: a.GetReduceMotion(),
		Radius:       pickOr(a.GetRadius(), appearanceRadii, "round"),
		ShowTabs:     a.GetShowTabs(),
	}
	if bg := a.GetBackground(); bg != nil {
		out.Background = (&themeNormalizer{resolve: syntacticAssetResolver}).background(bg)
	}
	return out
}

// EncodeAppearance 规范化外观偏好并编码为存储用的 JSON。
func EncodeAppearance(a *v1.UserAppearance) string {
	n := normalizeAppearance(a)
	j := appearanceJSON{
		Theme:        n.Theme,
		Mode:         n.Mode,
		Compact:      n.Compact,
		ReduceMotion: n.ReduceMotion,
		Radius:       n.Radius,
		ShowTabs:     n.ShowTabs,
	}
	if n.Background != nil {
		if raw, err := protojson.Marshal(n.Background); err == nil {
			j.Background = raw
		}
	}
	b, _ := json.Marshal(j)
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
	a := &v1.UserAppearance{
		Theme:        j.Theme,
		Mode:         j.Mode,
		Compact:      j.Compact,
		ReduceMotion: j.ReduceMotion,
		Radius:       j.Radius,
		ShowTabs:     j.ShowTabs,
	}
	if len(j.Background) > 0 && string(j.Background) != "null" {
		bg := new(v1.ThemeBackground)
		if err := themeContentUnmarshal.Unmarshal(j.Background, bg); err == nil {
			a.Background = bg
		}
	}
	return normalizeAppearance(a)
}

// UpdateMyAppearance 规范化并保存当前用户的外观偏好，返回实际保存的值。
func (u *UserUsecase) UpdateMyAppearance(ctx context.Context, userId string, a *v1.UserAppearance) (*v1.UserAppearance, error) {
	n, err := u.theme.NormalizeAppearance(ctx, userId, a)
	if err != nil {
		return nil, err
	}
	if err := u.user.SaveAppearance(ctx, userId, EncodeAppearance(n)); err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrAdminNotFound
		}
		return nil, ErrSystem(err)
	}
	return n, nil
}

// MeAppearance 返回当前用户保存的外观偏好（从未保存过为 nil），
// 以及当前用的「我的主题」（appearance.theme 为主题 id 时）与全站默认主题。
func (u *UserUsecase) MeAppearance(ctx context.Context, userId string) (*v1.UserAppearance, *v1.Theme, *v1.SiteTheme, error) {
	row, err := u.user.FindByID(ctx, userId)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil, nil, ErrAdminNotFound
		}
		return nil, nil, nil, ErrSystem(err)
	}
	appearance := DecodeAppearance(row.Appearance)
	current, site := u.theme.AppearanceThemes(ctx, userId, appearance)
	return appearance, current, site, nil
}
