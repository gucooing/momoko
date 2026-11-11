package biz

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/usertheme"
	"momoko/pkg/themeasset"
)

// 主题包（.momoko-theme）= zip：theme.json（UTF-8）+ assets/…（图片、字体）。
// 读取时只接受颜色、数值、枚举与包内图片 / 字体引用，其余内容一律丢弃并计数；绝不接受 CSS 或脚本。

const (
	themePackageFormat     = 1
	themePackageExt        = ".momoko-theme"
	themePackageMaxSize    = 40 << 20
	themePackageMaxEntries = 64
	themePackageMaxImages  = 24
	themeManifestName      = "theme.json"
	themeManifestMaxSize   = 1 << 20
	themePackageAssetsDir  = "assets/"
	themePackageWorkers    = 3
	themeFileNameMaxRunes  = 100
)

// ---------------------------------------------------------------------------
// theme.json 结构（camelCase，与前端主题包格式一致）
// ---------------------------------------------------------------------------

type pkgTheme struct {
	Format      int            `json:"format"`
	Name        string         `json:"name"`
	Author      string         `json:"author,omitempty"`
	Version     string         `json:"version,omitempty"`
	Description string         `json:"description,omitempty"`
	Base        string         `json:"base"`
	Tokens      *pkgTokens     `json:"tokens,omitempty"`
	Background  *pkgBackground `json:"background,omitempty"`
	Shape       *pkgShape      `json:"shape,omitempty"`
	Banner      *pkgBanner     `json:"banner,omitempty"`
	// Sizes 组件尺寸覆盖（px），键见前端尺寸登记表
	Sizes map[string]float64  `json:"sizes,omitempty"`
	Pages map[string]*pkgPage `json:"pages,omitempty"`
}

type pkgTokens struct {
	Light map[string]string `json:"light,omitempty"`
	Dark  map[string]string `json:"dark,omitempty"`
}

type pkgGradient struct {
	Stops []string `json:"stops,omitempty"`
	Angle int      `json:"angle"`
	Kind  string   `json:"kind,omitempty"`
	Flow  bool     `json:"flow,omitempty"`
}

type pkgBackground struct {
	Type         string       `json:"type"`
	Color        string       `json:"color,omitempty"`
	Pattern      string       `json:"pattern,omitempty"`
	PatternColor string       `json:"patternColor,omitempty"`
	Gradient     *pkgGradient `json:"gradient,omitempty"`
	Image        string       `json:"image,omitempty"`
	Fit          string       `json:"fit,omitempty"`
	Blur         int          `json:"blur,omitempty"`
	Dim          int          `json:"dim,omitempty"`
	CardOpacity  *int         `json:"cardOpacity,omitempty"`
	AutoDarkDim  bool         `json:"autoDarkDim,omitempty"`
}

type pkgShape struct {
	Radius      *int   `json:"radius,omitempty"`
	Font        string `json:"font,omitempty"`
	FontFile    string `json:"fontFile,omitempty"`
	FontName    string `json:"fontName,omitempty"`
	FontSize    string `json:"fontSize,omitempty"`
	Shadow      string `json:"shadow,omitempty"`
	Button      string `json:"button,omitempty"`
	Sidebar     string `json:"sidebar,omitempty"`
	Pill        string `json:"pill,omitempty"`
	CardBorder  *bool  `json:"cardBorder,omitempty"`
	StickerTilt *bool  `json:"stickerTilt,omitempty"`
}

type pkgBanner struct {
	Type     string `json:"type"`
	Gradient int    `json:"gradient,omitempty"`
	Image    string `json:"image,omitempty"`
	Dim      int    `json:"dim,omitempty"`
	Ink      string `json:"ink,omitempty"`
	Focus    []int  `json:"focus,omitempty"`
}

type pkgPage struct {
	Tokens      *pkgTokens         `json:"tokens,omitempty"`
	Background  *pkgBackground     `json:"background,omitempty"`
	CardOpacity *int               `json:"cardOpacity,omitempty"`
	Banner      *pkgBanner         `json:"banner,omitempty"`
	Sizes       map[string]float64 `json:"sizes,omitempty"`
}

func clampToInt32(v int) int32 {
	return int32(max(math.MinInt32, min(math.MaxInt32, v)))
}

func pkgTokensToProto(t *pkgTokens) (light, dark map[string]string) {
	if t == nil {
		return nil, nil
	}
	return t.Light, t.Dark
}

func protoTokensToPkg(light, dark map[string]string) *pkgTokens {
	if len(light) == 0 && len(dark) == 0 {
		return nil
	}
	return &pkgTokens{Light: light, Dark: dark}
}

func pkgBackgroundToProto(b *pkgBackground) *v1.ThemeBackground {
	if b == nil {
		return nil
	}
	out := &v1.ThemeBackground{
		Type:         b.Type,
		Color:        b.Color,
		Pattern:      b.Pattern,
		PatternColor: b.PatternColor,
		Image:        pkgAssetPath(b.Image),
		Fit:          b.Fit,
		Blur:         clampToInt32(b.Blur),
		Dim:          clampToInt32(b.Dim),
		AutoDarkDim:  b.AutoDarkDim,
	}
	if b.CardOpacity != nil {
		// 显式给出的 0 也按下限处理（0 在 proto 里表示「未设置」）
		out.CardOpacity = clampInt32(clampToInt32(*b.CardOpacity), 40, 100)
	}
	if g := b.Gradient; g != nil {
		out.GradientStops = g.Stops
		out.GradientAngle = clampToInt32(g.Angle)
		out.GradientKind = g.Kind
		out.GradientFlow = g.Flow
	}
	return out
}

func protoBackgroundToPkg(b *v1.ThemeBackground) *pkgBackground {
	if b == nil {
		return nil
	}
	opacity := int(b.GetCardOpacity())
	return &pkgBackground{
		Type:         b.GetType(),
		Color:        b.GetColor(),
		Pattern:      b.GetPattern(),
		PatternColor: b.GetPatternColor(),
		Gradient: &pkgGradient{
			Stops: b.GetGradientStops(),
			Angle: int(b.GetGradientAngle()),
			Kind:  b.GetGradientKind(),
			Flow:  b.GetGradientFlow(),
		},
		Image:       b.GetImage(),
		Fit:         b.GetFit(),
		Blur:        int(b.GetBlur()),
		Dim:         int(b.GetDim()),
		CardOpacity: &opacity,
		AutoDarkDim: b.GetAutoDarkDim(),
	}
}

func pkgShapeToProto(s *pkgShape) *v1.ThemeShape {
	if s == nil {
		return nil
	}
	out := &v1.ThemeShape{
		Font:        s.Font,
		FontAsset:   pkgAssetPath(s.FontFile),
		FontName:    s.FontName,
		FontSize:    s.FontSize,
		Shadow:      s.Shadow,
		Button:      s.Button,
		Sidebar:     s.Sidebar,
		Pill:        s.Pill,
		CardBorder:  s.CardBorder,
		StickerTilt: s.StickerTilt,
	}
	if s.Radius != nil {
		r := clampToInt32(*s.Radius)
		out.Radius = &r
	}
	return out
}

func protoShapeToPkg(s *v1.ThemeShape) *pkgShape {
	if s == nil {
		return nil
	}
	out := &pkgShape{
		Font:        s.GetFont(),
		FontName:    s.GetFontName(),
		FontSize:    s.GetFontSize(),
		Shadow:      s.GetShadow(),
		Button:      s.GetButton(),
		Sidebar:     s.GetSidebar(),
		Pill:        s.GetPill(),
		CardBorder:  s.CardBorder,
		StickerTilt: s.StickerTilt,
	}
	if s.GetFont() == "custom" {
		out.FontFile = s.GetFontAsset()
	}
	if s.Radius != nil {
		r := int(s.GetRadius())
		out.Radius = &r
	}
	return out
}

func pkgBannerToProto(b *pkgBanner) *v1.ThemeBanner {
	if b == nil {
		return nil
	}
	out := &v1.ThemeBanner{
		Type:     b.Type,
		Gradient: clampToInt32(b.Gradient),
		Image:    pkgAssetPath(b.Image),
		Dim:      clampToInt32(b.Dim),
		Ink:      b.Ink,
		FocusX:   50,
		FocusY:   50,
	}
	if len(b.Focus) >= 2 {
		out.FocusX, out.FocusY = clampToInt32(b.Focus[0]), clampToInt32(b.Focus[1])
	}
	return out
}

func protoBannerToPkg(b *v1.ThemeBanner) *pkgBanner {
	if b == nil {
		return nil
	}
	return &pkgBanner{
		Type:     b.GetType(),
		Gradient: int(b.GetGradient()),
		Image:    b.GetImage(),
		Dim:      int(b.GetDim()),
		Ink:      b.GetInk(),
		Focus:    []int{int(b.GetFocusX()), int(b.GetFocusY())},
	}
}

// pkgToContent 把 theme.json 转成主题内容（资源引用仍为包内路径，未规范化）。
func pkgToContent(p *pkgTheme) *v1.ThemeContent {
	c := &v1.ThemeContent{
		Base:       p.Base,
		Background: pkgBackgroundToProto(p.Background),
		Shape:      pkgShapeToProto(p.Shape),
		Banner:     pkgBannerToProto(p.Banner),
		Sizes:      p.Sizes,
	}
	c.TokensLight, c.TokensDark = pkgTokensToProto(p.Tokens)
	if len(p.Pages) > 0 {
		c.Pages = make(map[string]*v1.ThemePageOverride, len(p.Pages))
		for k, page := range p.Pages {
			o := &v1.ThemePageOverride{}
			if page != nil {
				o.TokensLight, o.TokensDark = pkgTokensToProto(page.Tokens)
				o.Background = pkgBackgroundToProto(page.Background)
				o.Banner = pkgBannerToProto(page.Banner)
				o.Sizes = page.Sizes
				if page.CardOpacity != nil {
					v := clampToInt32(*page.CardOpacity)
					o.CardOpacity = &v
				}
			}
			c.Pages[k] = o
		}
	}
	return c
}

// contentToPkg 把（已规范化、资源引用已换成包内路径的）主题内容转成 theme.json。
func contentToPkg(c *v1.ThemeContent) *pkgTheme {
	p := &pkgTheme{
		Format:     themePackageFormat,
		Base:       c.GetBase(),
		Tokens:     protoTokensToPkg(c.GetTokensLight(), c.GetTokensDark()),
		Background: protoBackgroundToPkg(c.GetBackground()),
		Shape:      protoShapeToPkg(c.GetShape()),
		Banner:     protoBannerToPkg(c.GetBanner()),
		Sizes:      c.GetSizes(),
	}
	if len(c.GetPages()) > 0 {
		p.Pages = make(map[string]*pkgPage, len(c.GetPages()))
		for k, o := range c.GetPages() {
			page := &pkgPage{
				Tokens:     protoTokensToPkg(o.GetTokensLight(), o.GetTokensDark()),
				Background: protoBackgroundToPkg(o.GetBackground()),
				Banner:     protoBannerToPkg(o.GetBanner()),
				Sizes:      o.GetSizes(),
			}
			if o.CardOpacity != nil {
				v := int(o.GetCardOpacity())
				page.CardOpacity = &v
			}
			p.Pages[k] = page
		}
	}
	return p
}

// pkgAssetPath 规范化 theme.json 里的资源路径；不像包内路径的值原样返回（随后会被当作无效引用丢弃）。
func pkgAssetPath(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	p := strings.TrimPrefix(strings.ReplaceAll(ref, "\\", "/"), "./")
	if clean, ok := safeZipPath(p); ok && strings.HasPrefix(clean, themePackageAssetsDir) {
		return clean
	}
	return ref
}

// safeZipPath 校验 zip 内路径：拒绝绝对路径、盘符、.. 与 NUL（防 zip-slip）。
func safeZipPath(name string) (string, bool) {
	name = strings.ReplaceAll(name, "\\", "/")
	if name == "" || strings.HasPrefix(name, "/") || strings.ContainsAny(name, ":\x00") {
		return "", false
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return "", false
		}
	}
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false
	}
	return clean, true
}

// pruneJSON 按目标结构清理 theme.json：去掉无法识别的键、类型不符的值与 null，返回清理后的值、丢弃数量与是否保留。
// 字段名按 encoding/json 的规则大小写不敏感匹配。
func pruneJSON(v any, t reflect.Type) (any, int, bool) {
	if v == nil {
		return nil, 0, false
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct:
		m, ok := v.(map[string]any)
		if !ok {
			return nil, 1, false
		}
		out := make(map[string]any, len(m))
		n := 0
		for k, child := range m {
			f, ok := jsonFieldByName(t, k)
			if !ok {
				n++
				continue
			}
			c, d, keep := pruneJSON(child, f.Type)
			n += d
			if keep {
				out[k] = c
			}
		}
		return out, n, true
	case reflect.Map:
		m, ok := v.(map[string]any)
		if !ok {
			return nil, 1, false
		}
		out := make(map[string]any, len(m))
		n := 0
		for k, child := range m {
			c, d, keep := pruneJSON(child, t.Elem())
			n += d
			if keep {
				out[k] = c
			}
		}
		return out, n, true
	case reflect.Slice:
		s, ok := v.([]any)
		if !ok {
			return nil, 1, false
		}
		out := make([]any, 0, len(s))
		n := 0
		for _, child := range s {
			c, d, keep := pruneJSON(child, t.Elem())
			n += d
			if keep {
				out = append(out, c)
			}
		}
		return out, n, true
	case reflect.String:
		if _, ok := v.(string); !ok {
			return nil, 1, false
		}
	case reflect.Bool:
		if _, ok := v.(bool); !ok {
			return nil, 1, false
		}
	case reflect.Int, reflect.Int32, reflect.Int64:
		f, ok := v.(float64)
		if !ok || f != math.Trunc(f) || math.Abs(f) > 1e9 {
			return nil, 1, false
		}
	case reflect.Float64:
		if _, ok := v.(float64); !ok {
			return nil, 1, false
		}
	default:
		return nil, 1, false
	}
	return v, 0, true
}

func jsonFieldByName(t reflect.Type, key string) (reflect.StructField, bool) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		if strings.EqualFold(name, key) {
			return f, true
		}
	}
	return reflect.StructField{}, false
}

// ---------------------------------------------------------------------------
// 读取主题包
// ---------------------------------------------------------------------------

type parsedThemePackage struct {
	size        int64
	format      int
	name        string
	author      string
	version     string
	description string
	// 规范化后的内容，资源引用为包内路径
	content *v1.ThemeContent
	dropped int
	// 被引用且有效的图片 / 字体（包内路径 → 处理结果），paths 保持出现顺序
	images     map[string]*themeasset.Processed
	fonts      map[string]*themeasset.Processed
	imagePaths []string
	fontPaths  []string
}

// readZipEntry 读取 zip 条目，超过 limit 视为过大（不信任头里声明的大小）。
func readZipEntry(f *zip.File, limit int64) ([]byte, error) {
	if f.UncompressedSize64 > uint64(limit) {
		return nil, ErrThemePackageEntryLarge
	}
	rc, err := f.Open()
	if err != nil {
		return nil, ErrThemePackageInvalid
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, limit+1))
	if err != nil {
		return nil, ErrThemePackageInvalid
	}
	if int64(len(data)) > limit {
		return nil, ErrThemePackageEntryLarge
	}
	return data, nil
}

// packageNameFromFile 从文件名推导主题名（去掉扩展名）。
func packageNameFromFile(fileName string) string {
	name := assetFileName(fileName)
	if ext := path.Ext(name); ext != "" {
		name = strings.TrimSuffix(name, ext)
	}
	return themeText(name, themeNameMaxRunes)
}

func parseThemePackage(fileName, data string) (*parsedThemePackage, error) {
	raw, ok, err := decodeThemeBase64(data, themePackageMaxSize, ErrThemePackageTooLarge)
	if err != nil {
		return nil, err
	}
	if !ok || len(raw) == 0 {
		return nil, ErrThemePackageData
	}
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, ErrThemePackageInvalid
	}
	if len(zr.File) > themePackageMaxEntries {
		return nil, ErrThemePackageEntries
	}

	var manifest *zip.File
	assets := map[string]*zip.File{}
	for _, f := range zr.File {
		name, ok := safeZipPath(f.Name)
		if !ok {
			return nil, ErrThemePackageUnsafePath
		}
		if f.FileInfo().IsDir() || strings.HasSuffix(f.Name, "/") {
			continue
		}
		switch {
		case name == themeManifestName:
			manifest = f
		case strings.HasPrefix(name, themePackageAssetsDir):
			assets[name] = f
		}
	}
	if manifest == nil {
		return nil, ErrThemePackageNoManifest
	}
	manifestRaw, err := readZipEntry(manifest, themeManifestMaxSize)
	if err != nil {
		return nil, err
	}
	manifestRaw = bytes.TrimPrefix(manifestRaw, []byte("\xEF\xBB\xBF"))

	var generic any
	if err := json.Unmarshal(manifestRaw, &generic); err != nil {
		return nil, ErrThemePackageManifest
	}
	if _, ok := generic.(map[string]any); !ok {
		return nil, ErrThemePackageManifest
	}
	// 先按结构清理掉无法识别 / 类型不符的内容，再解码，避免类型不符的值以零值混进来
	var p pkgTheme
	cleaned, dropped, _ := pruneJSON(generic, reflect.TypeOf(p))
	cleanedRaw, err := json.Marshal(cleaned)
	if err != nil {
		return nil, ErrThemePackageManifest
	}
	if err := json.Unmarshal(cleanedRaw, &p); err != nil {
		return nil, ErrThemePackageManifest
	}
	if p.Format > themePackageFormat {
		return nil, ErrThemePackageNewer
	}
	if p.Format < 1 {
		return nil, ErrThemePackageInvalid
	}

	rawContent := pkgToContent(&p)
	out := &parsedThemePackage{
		size:        int64(len(raw)),
		format:      p.Format,
		name:        themeText(p.Name, themeNameMaxRunes),
		author:      themeText(p.Author, themeAuthorMaxRunes),
		version:     themeText(p.Version, themeVersionMaxRunes),
		description: themeText(p.Description, themeDescMaxRunes),
		images:      map[string]*themeasset.Processed{},
		fonts:       map[string]*themeasset.Processed{},
	}
	if out.name == "" {
		out.name = packageNameFromFile(fileName)
	}
	if out.name == "" {
		out.name = "导入的主题"
	}
	if out.version == "" {
		out.version = themeDefaultVersion
	}

	// 只处理被引用、且确实在包里的资源
	type job struct {
		kind, path string
		file       *zip.File
	}
	var jobs []job
	seen := map[string]struct{}{}
	walkThemeAssetRefs(rawContent, func(kind string, ref *string) {
		key := kind + "|" + *ref
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		if f, ok := assets[*ref]; ok {
			jobs = append(jobs, job{kind: kind, path: *ref, file: f})
		}
	})
	imageCount := 0
	for _, j := range jobs {
		if j.kind == themeasset.KindImage {
			imageCount++
		}
	}
	if imageCount > themePackageMaxImages {
		return nil, response500(fmt.Sprintf("主题包里的图片太多（最多 %d 张）", themePackageMaxImages))
	}

	results := make([]*themeasset.Processed, len(jobs))
	errs := make([]error, len(jobs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, themePackageWorkers)
	for i, j := range jobs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			limit := int64(themeasset.MaxImageSize)
			if j.kind == themeasset.KindFont {
				limit = themeasset.MaxFontSize
			}
			data, err := readZipEntry(j.file, limit)
			if err != nil {
				if errors.Is(err, ErrThemePackageEntryLarge) {
					if j.kind == themeasset.KindFont {
						err = themeasset.ErrFontTooLarge
					} else {
						err = themeasset.ErrImageTooLarge
					}
				}
				errs[i] = packageAssetErr(j.path, err)
				return
			}
			if j.kind == themeasset.KindFont {
				results[i], err = themeasset.ProcessFont(data)
			} else {
				results[i], err = themeasset.ProcessImage(data)
			}
			if err != nil {
				errs[i] = packageAssetErr(j.path, err)
			}
		}()
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	for i, j := range jobs {
		if j.kind == themeasset.KindFont {
			out.fonts[j.path] = results[i]
		} else {
			out.images[j.path] = results[i]
		}
	}

	n := &themeNormalizer{resolve: func(kind, ref string) (string, bool) {
		if kind == themeasset.KindFont {
			_, ok := out.fonts[ref]
			return ref, ok
		}
		_, ok := out.images[ref]
		return ref, ok
	}}
	out.content = n.content(rawContent)
	out.dropped = dropped + n.dropped

	// 只保留规范化后仍被引用的资源
	used := map[string]struct{}{}
	walkThemeAssetRefs(out.content, func(kind string, ref *string) {
		key := kind + "|" + *ref
		if _, ok := used[key]; ok {
			return
		}
		used[key] = struct{}{}
		if kind == themeasset.KindFont {
			out.fontPaths = append(out.fontPaths, *ref)
		} else {
			out.imagePaths = append(out.imagePaths, *ref)
		}
	})
	for p := range out.images {
		if _, ok := used[themeasset.KindImage+"|"+p]; !ok {
			delete(out.images, p)
		}
	}
	for p := range out.fonts {
		if _, ok := used[themeasset.KindFont+"|"+p]; !ok {
			delete(out.fonts, p)
		}
	}
	return out, nil
}

func response500(msg string) error {
	return ErrThemeAsset(errors.New(msg))
}

func packageAssetErr(p string, err error) error {
	return response500(fmt.Sprintf("主题包里的 %s：%v", p, err))
}

func (t *ThemeUsecase) InspectPackage(ctx context.Context, userID string, req *v1.InspectThemePackageRequest) (*v1.ThemePackageInfo, error) {
	pkg, err := parseThemePackage(req.GetFileName(), req.GetData())
	if err != nil {
		return nil, err
	}
	info := &v1.ThemePackageInfo{
		Format:      int32(pkg.format),
		Name:        pkg.name,
		Author:      pkg.author,
		Version:     pkg.version,
		Description: pkg.description,
		Content:     pkg.content,
		Images:      []*v1.ThemePackageImage{},
		Checks: []*v1.ThemePackageCheck{
			{Level: "ok", Code: "format_ok", Params: map[string]string{"format": strconv.Itoa(pkg.format)}},
		},
		PackageSize: pkg.size,
		Fonts:       map[string]int64{},
	}
	for _, p := range pkg.imagePaths {
		img := pkg.images[p]
		info.Images = append(info.Images, &v1.ThemePackageImage{
			Path:         p,
			DataUrl:      "data:image/webp;base64," + base64.StdEncoding.EncodeToString(img.Data),
			Width:        int32(img.Width),
			Height:       int32(img.Height),
			OriginalSize: img.OriginalSize,
			Size:         int64(len(img.Data)),
		})
		info.Checks = append(info.Checks, &v1.ThemePackageCheck{Level: "ok", Code: "image_converted", Params: map[string]string{
			"path":   p,
			"width":  strconv.Itoa(img.Width),
			"height": strconv.Itoa(img.Height),
			"from":   strconv.FormatInt(img.OriginalSize, 10),
			"to":     strconv.Itoa(len(img.Data)),
		}})
	}
	for _, p := range pkg.fontPaths {
		size := int64(len(pkg.fonts[p].Data))
		info.Fonts[p] = size
		info.Checks = append(info.Checks, &v1.ThemePackageCheck{Level: "ok", Code: "font_ok", Params: map[string]string{
			"name": p,
			"size": strconv.FormatInt(size, 10),
		}})
	}
	info.Checks = append(info.Checks, &v1.ThemePackageCheck{Level: "ok", Code: "safe_content", Params: map[string]string{}})
	if pkg.dropped > 0 {
		info.Checks = append(info.Checks, &v1.ThemePackageCheck{Level: "warn", Code: "dropped_fields", Params: map[string]string{
			"count": strconv.Itoa(pkg.dropped),
		}})
	}
	conflict, err := t.repo.FindThemeByName(ctx, userID, pkg.name)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if conflict != nil {
		info.ConflictThemeId = conflict.ID
		info.ConflictVersion = conflict.Version
	}
	return info, nil
}

func (t *ThemeUsecase) ImportPackage(ctx context.Context, userID string, req *v1.ImportThemePackageRequest) (*v1.Theme, error) {
	pkg, err := parseThemePackage(req.GetFileName(), req.GetData())
	if err != nil {
		return nil, err
	}
	conflict, err := t.repo.FindThemeByName(ctx, userID, pkg.name)
	if err != nil {
		return nil, ErrSystem(err)
	}
	overwrite := conflict != nil && strings.TrimSpace(req.GetConflict()) == "overwrite"
	if !overwrite {
		if err := t.checkThemeLimit(ctx, userID); err != nil {
			return nil, err
		}
	}

	// 保存资源并把包内路径换成资源 id
	ids := map[string]string{}
	var saved []string
	save := func(kind, p string, processed *themeasset.Processed) error {
		row, err := t.saveAsset(ctx, userID, assetFileName(p), processed)
		if err != nil {
			return err
		}
		ids[kind+"|"+p] = row.ID
		saved = append(saved, row.ID)
		return nil
	}
	for _, p := range pkg.imagePaths {
		if err := save(themeasset.KindImage, p, pkg.images[p]); err != nil {
			t.removeAssets(ctx, saved)
			return nil, err
		}
	}
	for _, p := range pkg.fontPaths {
		if err := save(themeasset.KindFont, p, pkg.fonts[p]); err != nil {
			t.removeAssets(ctx, saved)
			return nil, err
		}
	}
	content := cloneThemeContent(pkg.content)
	walkThemeAssetRefs(content, func(kind string, ref *string) {
		*ref = ids[kind+"|"+*ref]
	})
	raw, err := encodeThemeContent(content)
	if err != nil {
		t.removeAssets(ctx, saved)
		return nil, ErrSystem(err)
	}

	if overwrite {
		conflict.Name = pkg.name
		conflict.Author = pkg.author
		conflict.Version = pkg.version
		conflict.Description = pkg.description
		conflict.Source = usertheme.SourceImport
		conflict.Content = raw
		row, err := t.repo.UpdateTheme(ctx, conflict)
		if err != nil {
			t.removeAssets(ctx, saved)
			if gen.IsNotFound(err) {
				return nil, ErrThemeNotFound
			}
			return nil, ErrSystem(err)
		}
		t.collectAssetGarbage(ctx, userID)
		return t.toTheme(ctx, row)
	}

	name := pkg.name
	if conflict != nil {
		name = copyThemeName(name)
	}
	theme, err := t.createTheme(ctx, &gen.UserTheme{
		UserID:      userID,
		Name:        name,
		Author:      pkg.author,
		Version:     pkg.version,
		Description: pkg.description,
		Source:      usertheme.SourceImport,
	}, content)
	if err != nil {
		t.removeAssets(ctx, saved)
		return nil, err
	}
	return theme, nil
}

// ---------------------------------------------------------------------------
// 导出主题包
// ---------------------------------------------------------------------------

// stripThemeImages 不含图片：图片背景落回跟随主题，图片横幅落回主色。
func stripThemeImages(c *v1.ThemeContent) {
	stripBg := func(b *v1.ThemeBackground) {
		if b == nil {
			return
		}
		if b.Type == "image" {
			b.Type = "theme"
		}
		b.Image = ""
	}
	stripBanner := func(b *v1.ThemeBanner) {
		if b == nil {
			return
		}
		if b.Type == "image" {
			b.Type = "color"
		}
		b.Image = ""
	}
	stripBg(c.Background)
	stripBanner(c.Banner)
	for _, p := range c.Pages {
		stripBg(p.GetBackground())
		stripBanner(p.GetBanner())
	}
}

// sanitizePackageFileName 清理导出文件名并补上 .momoko-theme 扩展名。
func sanitizePackageFileName(name, themeName string) string {
	clean := func(s string) string {
		s = strings.Map(func(r rune) rune {
			if unicode.IsControl(r) || strings.ContainsRune(`<>:"/\|?*`, r) {
				return -1
			}
			return r
		}, s)
		return strings.Trim(strings.TrimSpace(s), ". ")
	}
	name = clean(name)
	if name == "" {
		name = clean(themeName)
	}
	if name == "" {
		name = "theme"
	}
	if !strings.HasSuffix(strings.ToLower(name), themePackageExt) {
		name = themeText(name, themeFileNameMaxRunes-len([]rune(themePackageExt))) + themePackageExt
	}
	return name
}

func (t *ThemeUsecase) ExportPackage(ctx context.Context, userID string, req *v1.ExportThemePackageRequest) (*v1.ExportThemePackageResponse, error) {
	var row *gen.UserTheme
	if strings.TrimSpace(req.GetThemeId()) != "" {
		r, err := t.getOwnTheme(ctx, userID, req.GetThemeId())
		if err != nil {
			return nil, err
		}
		row = r
	}
	var raw *v1.ThemeContent
	switch {
	case req.GetContent() != nil:
		raw = req.GetContent()
	case row != nil:
		raw = decodeThemeContent(row.Content)
	}
	// 引用的资源只需存在且文件还在（任何人上传的都可以，资源本身就是公开地址）
	resolver, err := t.assetResolver(ctx, raw, func(a *gen.ThemeAsset) bool { return t.assets.Exists(a.ID) })
	if err != nil {
		return nil, ErrSystem(err)
	}
	content := (&themeNormalizer{resolve: resolver}).content(raw)

	name := themeText(req.GetName(), themeNameMaxRunes)
	description := themeText(req.GetDescription(), themeDescMaxRunes)
	author, version := "", themeDefaultVersion
	if row != nil {
		if name == "" {
			name = row.Name
		}
		if description == "" {
			description = row.Description
		}
		author = row.Author
		if row.Version != "" {
			version = row.Version
		}
	}
	if author == "" {
		author = themeText(t.userDisplayName(ctx, userID), themeAuthorMaxRunes)
	}
	if name == "" {
		name = "未命名主题"
	}

	if page := strings.TrimSpace(req.GetOnlyPage()); page != "" {
		override, ok := content.GetPages()[page]
		if !ok {
			return nil, ErrThemeExportPage
		}
		content = &v1.ThemeContent{Base: content.GetBase(), Pages: map[string]*v1.ThemePageOverride{page: override}}
		name = themeText(name+" · "+page, themeNameMaxRunes)
	} else {
		// 不含细节：圆角 / 字体等与全局组件尺寸都不导出（页面里的尺寸跟随页面设置）
		if !req.GetIncludeShape() {
			content.Shape = nil
			content.Sizes = nil
		}
		if !req.GetIncludePages() {
			content.Pages = nil
		}
	}
	if !req.GetIncludeImages() {
		stripThemeImages(content)
	}
	if content.Shape != nil && content.Shape.Font != "custom" {
		content.Shape.FontAsset = ""
	}

	// 资源 id → 包内路径（同一资源只打包一次）
	type packFile struct{ id, path string }
	var files []packFile
	paths := map[string]string{}
	assign := func(id *string, base string) {
		if *id == "" {
			return
		}
		if p, ok := paths[*id]; ok {
			*id = p
			return
		}
		ext := themeasset.ExtWebP
		if themeasset.KindOf(*id) == themeasset.KindFont {
			ext = themeasset.ExtWOFF2
		}
		p := themePackageAssetsDir + base + "." + ext
		paths[*id] = p
		files = append(files, packFile{id: *id, path: p})
		*id = p
	}
	if b := content.Background; b != nil {
		assign(&b.Image, "background")
	}
	if b := content.Banner; b != nil {
		assign(&b.Image, "banner")
	}
	if s := content.Shape; s != nil {
		assign(&s.FontAsset, "font")
	}
	pageKeys := make([]string, 0, len(content.Pages))
	for k := range content.Pages {
		pageKeys = append(pageKeys, k)
	}
	sort.Strings(pageKeys)
	for i, k := range pageKeys {
		o := content.Pages[k]
		if b := o.GetBackground(); b != nil {
			assign(&b.Image, fmt.Sprintf("page-%d-background", i+1))
		}
		if b := o.GetBanner(); b != nil {
			assign(&b.Image, fmt.Sprintf("page-%d-banner", i+1))
		}
	}

	manifest := contentToPkg(content)
	manifest.Name, manifest.Author, manifest.Version, manifest.Description = name, author, version, description
	manifestRaw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, ErrSystem(err)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	now := time.Now()
	write := func(p string, data []byte, method uint16) error {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: p, Method: method, Modified: now})
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
	if err := write(themeManifestName, manifestRaw, zip.Deflate); err != nil {
		return nil, ErrSystem(err)
	}
	for _, f := range files {
		data, err := t.assets.Read(f.id)
		if err != nil {
			return nil, ErrThemeAssetMissing
		}
		// WebP / woff2 本身已压缩，直接存储
		if err := write(f.path, data, zip.Store); err != nil {
			return nil, ErrSystem(err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.ExportThemePackageResponse{
		FileName: sanitizePackageFileName(req.GetFileName(), name),
		Data:     base64.StdEncoding.EncodeToString(buf.Bytes()),
		Size:     int64(buf.Len()),
	}, nil
}
