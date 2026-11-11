package biz

import (
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/themeasset"
)

// 主题内容规范化：新建 / 保存 / 导入 / 全站默认 / 个人外观里的背景都走同一套规则，
// 只保留颜色、数值、枚举和资源引用，其余一律丢弃（并计数，主题包导入时提示）。

var (
	ThemeBuiltinKeys    = []string{"sakura", "mint", "volt", "grape", "sunset", "classic"}
	themeBackgroundType = []string{"theme", "solid", "pattern", "gradient", "image"}
	themePatterns       = []string{"dots", "grid", "stripe", "polka", "wave"}
	themePatternColors  = []string{"ink", "p", "a2"}
	themeGradientKinds  = []string{"linear", "radial"}
	themeImageFits      = []string{"cover", "contain", "tile", "center"}
	themeFonts          = []string{"outfit", "system", "rounded", "serif", "mono", "custom"}
	themeFontSizes      = []string{"s", "m", "l"}
	themeShadows        = []string{"none", "soft", "strong"}
	themeButtons        = []string{"pill", "round", "square"}
	themeSidebars       = []string{"solid", "clear", "glass"}
	themePills          = []string{"dot", "fill"}
	themeBannerTypes    = []string{"color", "gradient", "image"}
	themeBannerInks     = []string{"auto", "light", "dark"}

	themeDefaultGradient = []string{"#FFD3E2", "#FFE9C7", "#D9E4FF"}

	themeTokenKeys = func() map[string]struct{} {
		keys := []string{
			"p", "pOn", "pInk", "pSoft", "a", "aOn", "aSoft", "a2", "a2Soft",
			"bg", "s", "s2", "line", "ink", "ink2",
			"ok", "okSoft", "warn", "warnSoft", "bad", "badSoft",
			"termBg", "termFg", "termCursor",
		}
		m := make(map[string]struct{}, len(keys)+16)
		for _, k := range keys {
			m[k] = struct{}{}
		}
		for i := 0; i < 16; i++ {
			m["ansi"+strconv.Itoa(i)] = struct{}{}
		}
		return m
	}()

	themeHexPattern     = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	themePageKeyPattern = regexp.MustCompile(`^/[A-Za-z0-9/_:.\-]{0,79}$`)
	themeSizeKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,47}$`)
)

const (
	themeMaxPages        = 40
	themeFontNameMaxRune = 40
	themeMaxSizes        = 200
	themeMaxSizeValue    = 1000.0
)

// IsBuiltinTheme 判断是否为内置配色键。
func IsBuiltinTheme(key string) bool {
	return slices.Contains(ThemeBuiltinKeys, key)
}

// themeAssetResolver 校验并改写资源引用：kind 为 image / font，返回改写后的引用与是否有效。
type themeAssetResolver func(kind, ref string) (string, bool)

// themeNormalizer 规范化主题内容；dropped 统计被丢弃的非法取值数量。
type themeNormalizer struct {
	resolve themeAssetResolver
	dropped int
}

func (n *themeNormalizer) enum(v string, allowed []string, def string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	if slices.Contains(allowed, v) {
		return v
	}
	n.dropped++
	return def
}

func (n *themeNormalizer) hex(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if themeHexPattern.MatchString(v) {
		return strings.ToUpper(v)
	}
	n.dropped++
	return ""
}

func (n *themeNormalizer) ref(kind, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if n.resolve != nil {
		if out, ok := n.resolve(kind, ref); ok {
			return out
		}
	}
	n.dropped++
	return ""
}

func clampInt32(v, lo, hi int32) int32 {
	return max(lo, min(hi, v))
}

func (n *themeNormalizer) tokens(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		if _, ok := themeTokenKeys[k]; !ok {
			n.dropped++
			continue
		}
		if c := n.hex(v); c != "" {
			out[k] = c
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// sizes 规范化组件尺寸覆盖（px）：键来自前端的开放登记表，只校验格式；值须为有限数，
// 限制在 0 - 1000 并取整到 0.5；每份最多 200 项（按键排序，多余的丢弃）。
func (n *themeNormalizer) sizes(in map[string]float64) map[string]float64 {
	if len(in) == 0 {
		return nil
	}
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(map[string]float64, min(len(in), themeMaxSizes))
	for _, k := range keys {
		v := in[k]
		if !themeSizeKeyPattern.MatchString(k) || math.IsNaN(v) || math.IsInf(v, 0) || len(out) >= themeMaxSizes {
			n.dropped++
			continue
		}
		out[k] = math.Round(max(0, min(themeMaxSizeValue, v))*2) / 2
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// content 规范化整份主题内容；nil 视为空内容。没给的背景 / 细节 / 横幅保持为空（前端按默认值处理）。
func (n *themeNormalizer) content(c *v1.ThemeContent) *v1.ThemeContent {
	out := &v1.ThemeContent{
		Base:        n.enum(c.GetBase(), ThemeBuiltinKeys, "sakura"),
		TokensLight: n.tokens(c.GetTokensLight()),
		TokensDark:  n.tokens(c.GetTokensDark()),
		Sizes:       n.sizes(c.GetSizes()),
	}
	if c.GetBackground() != nil {
		out.Background = n.background(c.GetBackground())
	}
	if c.GetShape() != nil {
		out.Shape = n.shape(c.GetShape())
	}
	if c.GetBanner() != nil {
		out.Banner = n.banner(c.GetBanner())
	}
	if pages := c.GetPages(); len(pages) > 0 {
		keys := make([]string, 0, len(pages))
		for k := range pages {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out.Pages = make(map[string]*v1.ThemePageOverride)
		for _, k := range keys {
			if !themePageKeyPattern.MatchString(k) || len(out.Pages) >= themeMaxPages {
				n.dropped++
				continue
			}
			out.Pages[k] = n.page(pages[k])
		}
		if len(out.Pages) == 0 {
			out.Pages = nil
		}
	}
	return out
}

func (n *themeNormalizer) page(p *v1.ThemePageOverride) *v1.ThemePageOverride {
	out := &v1.ThemePageOverride{
		TokensLight: n.tokens(p.GetTokensLight()),
		TokensDark:  n.tokens(p.GetTokensDark()),
		Sizes:       n.sizes(p.GetSizes()),
	}
	if p.GetBackground() != nil {
		out.Background = n.background(p.GetBackground())
	}
	if p != nil && p.CardOpacity != nil {
		v := clampInt32(p.GetCardOpacity(), 40, 100)
		out.CardOpacity = &v
	}
	if p.GetBanner() != nil {
		out.Banner = n.banner(p.GetBanner())
	}
	return out
}

// background 规范化背景。
func (n *themeNormalizer) background(b *v1.ThemeBackground) *v1.ThemeBackground {
	out := &v1.ThemeBackground{
		Type:          n.enum(b.GetType(), themeBackgroundType, "theme"),
		Color:         n.hex(b.GetColor()),
		Pattern:       n.enum(b.GetPattern(), themePatterns, "dots"),
		PatternColor:  n.enum(b.GetPatternColor(), themePatternColors, "ink"),
		GradientStops: n.gradientStops(b.GetGradientStops()),
		GradientAngle: clampInt32(b.GetGradientAngle(), 0, 360),
		GradientKind:  n.enum(b.GetGradientKind(), themeGradientKinds, "linear"),
		GradientFlow:  b.GetGradientFlow(),
		Image:         n.ref(themeasset.KindImage, b.GetImage()),
		Fit:           n.enum(b.GetFit(), themeImageFits, "cover"),
		Blur:          clampInt32(b.GetBlur(), 0, 40),
		Dim:           clampInt32(b.GetDim(), 0, 80),
		AutoDarkDim:   b.GetAutoDarkDim(),
	}
	if out.Type == "image" && out.Image == "" {
		out.Type = "theme"
	}
	switch {
	case b.GetCardOpacity() != 0:
		out.CardOpacity = clampInt32(b.GetCardOpacity(), 40, 100)
	case out.Type == "image" || out.Type == "gradient":
		out.CardOpacity = 76
	default:
		out.CardOpacity = 100
	}
	return out
}

func (n *themeNormalizer) gradientStops(stops []string) []string {
	if len(stops) == 0 {
		return slices.Clone(themeDefaultGradient)
	}
	out := make([]string, 0, len(stops))
	for _, s := range stops {
		if c := n.hex(s); c != "" {
			out = append(out, c)
		}
	}
	if len(out) < 2 || len(out) > 5 {
		return slices.Clone(themeDefaultGradient)
	}
	return out
}

// shape 规范化细节。
func (n *themeNormalizer) shape(s *v1.ThemeShape) *v1.ThemeShape {
	out := &v1.ThemeShape{
		Font:      n.enum(s.GetFont(), themeFonts, "outfit"),
		FontAsset: n.ref(themeasset.KindFont, s.GetFontAsset()),
		FontName:  themeText(s.GetFontName(), themeFontNameMaxRune),
		FontSize:  n.enum(s.GetFontSize(), themeFontSizes, "m"),
		Shadow:    n.enum(s.GetShadow(), themeShadows, "none"),
		Button:    n.enum(s.GetButton(), themeButtons, "pill"),
		Sidebar:   n.enum(s.GetSidebar(), themeSidebars, "glass"),
		Pill:      n.enum(s.GetPill(), themePills, "dot"),
	}
	if s != nil {
		if s.Radius != nil {
			r := clampInt32(s.GetRadius(), 0, 32)
			out.Radius = &r
		}
		if s.CardBorder != nil {
			v := s.GetCardBorder()
			out.CardBorder = &v
		}
		if s.StickerTilt != nil {
			v := s.GetStickerTilt()
			out.StickerTilt = &v
		}
	}
	if out.Font == "custom" && out.FontAsset == "" {
		out.Font = "outfit"
	}
	return out
}

// banner 规范化大图区域。
func (n *themeNormalizer) banner(b *v1.ThemeBanner) *v1.ThemeBanner {
	out := &v1.ThemeBanner{
		Type:     n.enum(b.GetType(), themeBannerTypes, "color"),
		Gradient: clampInt32(b.GetGradient(), 0, 2),
		Image:    n.ref(themeasset.KindImage, b.GetImage()),
		Dim:      clampInt32(b.GetDim(), 0, 80),
		Ink:      n.enum(b.GetInk(), themeBannerInks, "auto"),
		FocusX:   clampInt32(b.GetFocusX(), 0, 100),
		FocusY:   clampInt32(b.GetFocusY(), 0, 100),
	}
	if out.Type == "image" && out.Image == "" {
		out.Type = "color"
	}
	return out
}

// themeText 清理自由文本：去掉控制字符与首尾空白，按字数截断。
func themeText(s string, maxRunes int) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > maxRunes {
		s = strings.TrimSpace(string([]rune(s)[:maxRunes]))
	}
	return s
}

// walkThemeAssetRefs 依次访问主题内容里的全部资源引用（可就地改写）。
func walkThemeAssetRefs(c *v1.ThemeContent, fn func(kind string, ref *string)) {
	if c == nil {
		return
	}
	visitBg := func(b *v1.ThemeBackground) {
		if b != nil && b.Image != "" {
			fn(themeasset.KindImage, &b.Image)
		}
	}
	visitBanner := func(b *v1.ThemeBanner) {
		if b != nil && b.Image != "" {
			fn(themeasset.KindImage, &b.Image)
		}
	}
	visitBg(c.Background)
	visitBanner(c.Banner)
	if c.Shape != nil && c.Shape.FontAsset != "" {
		fn(themeasset.KindFont, &c.Shape.FontAsset)
	}
	keys := make([]string, 0, len(c.Pages))
	for k := range c.Pages {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if p := c.Pages[k]; p != nil {
			visitBg(p.Background)
			visitBanner(p.Banner)
		}
	}
}

// themeAssetRefs 收集主题内容引用的资源（去重，保持出现顺序）。
func themeAssetRefs(c *v1.ThemeContent) []string {
	var out []string
	seen := map[string]struct{}{}
	walkThemeAssetRefs(c, func(_ string, ref *string) {
		if _, ok := seen[*ref]; !ok {
			seen[*ref] = struct{}{}
			out = append(out, *ref)
		}
	})
	return out
}
