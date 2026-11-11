package biz

import (
	"context"
	"encoding/base64"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	entthemeasset "momoko/internal/data/ent/gen/themeasset"
	"momoko/internal/data/ent/gen/usertheme"
	"momoko/pkg/themeasset"
)

// 自定义外观主题：「我的主题」、主题资源（背景图 / 字体）、主题包导入导出与全站默认主题。

type ThemeRepo interface {
	// ListThemes 用户的全部主题（新的在前）。
	ListThemes(ctx context.Context, userID string) ([]*gen.UserTheme, error)
	// GetTheme 用户的某个主题；不存在或不属于该用户返回 NotFound。
	GetTheme(ctx context.Context, userID, id string) (*gen.UserTheme, error)
	// FindThemeByName 按名称找用户的主题；没有返回 nil, nil。
	FindThemeByName(ctx context.Context, userID, name string) (*gen.UserTheme, error)
	CountThemes(ctx context.Context, userID string) (int, error)
	CreateTheme(ctx context.Context, t *gen.UserTheme) (*gen.UserTheme, error)
	// UpdateTheme 按 id + user_id 整体更新名称、作者、版本、说明、来源与内容。
	UpdateTheme(ctx context.Context, t *gen.UserTheme) (*gen.UserTheme, error)
	DeleteTheme(ctx context.Context, userID, id string) error

	CreateAsset(ctx context.Context, a *gen.ThemeAsset) (*gen.ThemeAsset, error)
	GetAssets(ctx context.Context, ids []string) ([]*gen.ThemeAsset, error)
	// ListStaleAssets 用户在 before 之前上传的资源（最多 limit 条，旧的在前）。
	ListStaleAssets(ctx context.Context, userID string, before time.Time, limit int) ([]*gen.ThemeAsset, error)
	// AssetReferenced 资源是否仍被任何用户的主题或个人外观引用。
	AssetReferenced(ctx context.Context, id string) (bool, error)
	DeleteAsset(ctx context.Context, id string) error
}

const (
	themeMaxPerUser      = 50
	themeNameMaxRunes    = 40
	themeDescMaxRunes    = 200
	themeAuthorMaxRunes  = 40
	themeVersionMaxRunes = 20
	themeAssetNameMax    = 100
	themeCopySuffix      = " 副本"
	themeDefaultVersion  = "1.0"
	// 资源上传后 24 小时内即使没被引用也保留（编辑器里刚上传、还没保存的背景图）。
	themeAssetGCGrace = 24 * time.Hour
	themeAssetGCBatch = 200
)

type ThemeUsecase struct {
	repo   ThemeRepo
	config ConfigRepo
	users  UserRepo
	assets *themeasset.Manager
}

func NewThemeUsecase(repo ThemeRepo, config ConfigRepo, users UserRepo, assets *themeasset.Manager) *ThemeUsecase {
	return &ThemeUsecase{repo: repo, config: config, users: users, assets: assets}
}

// ---------------------------------------------------------------------------
// 编解码与转换
// ---------------------------------------------------------------------------

var themeContentUnmarshal = protojson.UnmarshalOptions{DiscardUnknown: true}

// decodeThemeContent 解析存储的主题内容；损坏时返回默认内容。
func decodeThemeContent(raw string) *v1.ThemeContent {
	c := new(v1.ThemeContent)
	if err := themeContentUnmarshal.Unmarshal([]byte(raw), c); err != nil {
		return (&themeNormalizer{}).content(nil)
	}
	return c
}

func encodeThemeContent(c *v1.ThemeContent) (string, error) {
	b, err := protojson.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func toThemeAssetInfo(a *gen.ThemeAsset) *v1.ThemeAsset {
	return &v1.ThemeAsset{
		Id:           a.ID,
		Kind:         string(a.Kind),
		Name:         a.Name,
		Url:          themeasset.URL(a.ID),
		Size:         a.Size,
		OriginalSize: a.OriginalSize,
		Width:        a.Width,
		Height:       a.Height,
	}
}

// toThemes 批量转换主题（一次查出全部引用到的资源）。
func (t *ThemeUsecase) toThemes(ctx context.Context, rows []*gen.UserTheme) ([]*v1.Theme, error) {
	siteID := ""
	if site, err := t.config.SiteTheme(ctx); err == nil && site != nil {
		siteID = site.GetThemeId()
	}
	contents := make([]*v1.ThemeContent, len(rows))
	var ids []string
	seen := map[string]struct{}{}
	for i, row := range rows {
		contents[i] = decodeThemeContent(row.Content)
		for _, ref := range themeAssetRefs(contents[i]) {
			if _, ok := seen[ref]; !ok && themeasset.ValidID(ref) {
				seen[ref] = struct{}{}
				ids = append(ids, ref)
			}
		}
	}
	assets, err := t.repo.GetAssets(ctx, ids)
	if err != nil {
		return nil, ErrSystem(err)
	}
	byID := make(map[string]*gen.ThemeAsset, len(assets))
	for _, a := range assets {
		byID[a.ID] = a
	}
	out := make([]*v1.Theme, 0, len(rows))
	for i, row := range rows {
		theme := &v1.Theme{
			Id:          row.ID,
			Name:        row.Name,
			Author:      row.Author,
			Version:     row.Version,
			Description: row.Description,
			Source:      string(row.Source),
			Content:     contents[i],
			Assets:      []*v1.ThemeAsset{},
			SiteDefault: siteID != "" && siteID == row.ID,
			CreatedAt:   timestamppb.New(row.CreateTime),
			UpdatedAt:   timestamppb.New(row.UpdateTime),
		}
		for _, ref := range themeAssetRefs(contents[i]) {
			if a, ok := byID[ref]; ok {
				theme.Assets = append(theme.Assets, toThemeAssetInfo(a))
			}
		}
		out = append(out, theme)
	}
	return out, nil
}

func (t *ThemeUsecase) toTheme(ctx context.Context, row *gen.UserTheme) (*v1.Theme, error) {
	themes, err := t.toThemes(ctx, []*gen.UserTheme{row})
	if err != nil {
		return nil, err
	}
	return themes[0], nil
}

// userDisplayName 用户展示名（昵称，否则用户名）；查不到返回空串。
func (t *ThemeUsecase) userDisplayName(ctx context.Context, userID string) string {
	if userID == "" {
		return ""
	}
	u, err := t.users.FindByID(ctx, userID)
	if err != nil || u == nil {
		return ""
	}
	return displayUserName(u)
}

// ---------------------------------------------------------------------------
// 资源引用校验
// ---------------------------------------------------------------------------

// siteAssetRefs 全站默认主题引用的资源（任何人的主题都可以引用）。
func (t *ThemeUsecase) siteAssetRefs(ctx context.Context) (map[string]struct{}, error) {
	site, err := t.config.SiteTheme(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string]struct{}{}
	for _, ref := range themeAssetRefs(site.GetContent()) {
		out[ref] = struct{}{}
	}
	return out, nil
}

// assetResolver 查出 raw 里引用的资源，按 allow 过滤后返回校验函数（引用值原样保留）。
func (t *ThemeUsecase) assetResolver(ctx context.Context, raw *v1.ThemeContent, allow func(a *gen.ThemeAsset) bool) (themeAssetResolver, error) {
	var ids []string
	for _, ref := range themeAssetRefs(raw) {
		if themeasset.ValidID(ref) {
			ids = append(ids, ref)
		}
	}
	valid := map[string]string{}
	if len(ids) > 0 {
		rows, err := t.repo.GetAssets(ctx, ids)
		if err != nil {
			return nil, err
		}
		for _, a := range rows {
			if allow == nil || allow(a) {
				valid[a.ID] = string(a.Kind)
			}
		}
	}
	return func(kind, ref string) (string, bool) {
		if k, ok := valid[ref]; ok && k == kind {
			return ref, true
		}
		return "", false
	}, nil
}

// userAssetResolver 用户自己的资源，或当前全站默认主题引用的资源。
func (t *ThemeUsecase) userAssetResolver(ctx context.Context, userID string, raw *v1.ThemeContent) (themeAssetResolver, error) {
	siteRefs, err := t.siteAssetRefs(ctx)
	if err != nil {
		return nil, err
	}
	return t.assetResolver(ctx, raw, func(a *gen.ThemeAsset) bool {
		if a.UserID == userID {
			return true
		}
		_, ok := siteRefs[a.ID]
		return ok
	})
}

// normalizeUserContent 按「用户自己的资源 + 全站默认引用的资源」规范化主题内容。
func (t *ThemeUsecase) normalizeUserContent(ctx context.Context, userID string, raw *v1.ThemeContent) (*v1.ThemeContent, error) {
	resolver, err := t.userAssetResolver(ctx, userID, raw)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return (&themeNormalizer{resolve: resolver}).content(raw), nil
}

// ---------------------------------------------------------------------------
// 名称 / 说明
// ---------------------------------------------------------------------------

func validateThemeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrThemeNameEmpty
	}
	if utf8.RuneCountInString(name) > themeNameMaxRunes {
		return "", ErrThemeNameTooLong
	}
	return themeText(name, themeNameMaxRunes), nil
}

func validateThemeDescription(desc string) (string, error) {
	desc = strings.TrimSpace(desc)
	if utf8.RuneCountInString(desc) > themeDescMaxRunes {
		return "", ErrThemeDescriptionLong
	}
	return themeText(desc, themeDescMaxRunes), nil
}

// copyThemeName 「原名 副本」，原名过长时截断。
func copyThemeName(name string) string {
	base := themeText(name, themeNameMaxRunes-utf8.RuneCountInString(themeCopySuffix))
	return base + themeCopySuffix
}

func (t *ThemeUsecase) checkThemeLimit(ctx context.Context, userID string) error {
	n, err := t.repo.CountThemes(ctx, userID)
	if err != nil {
		return ErrSystem(err)
	}
	if n >= themeMaxPerUser {
		return ErrThemeLimit
	}
	return nil
}

func (t *ThemeUsecase) getOwnTheme(ctx context.Context, userID, id string) (*gen.UserTheme, error) {
	row, err := t.repo.GetTheme(ctx, userID, strings.TrimSpace(id))
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrThemeNotFound
		}
		return nil, ErrSystem(err)
	}
	return row, nil
}

// ---------------------------------------------------------------------------
// 我的主题
// ---------------------------------------------------------------------------

func (t *ThemeUsecase) ListThemes(ctx context.Context, userID string) ([]*v1.Theme, error) {
	rows, err := t.repo.ListThemes(ctx, userID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return t.toThemes(ctx, rows)
}

func (t *ThemeUsecase) GetTheme(ctx context.Context, userID, id string) (*v1.Theme, error) {
	row, err := t.getOwnTheme(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return t.toTheme(ctx, row)
}

func (t *ThemeUsecase) CreateTheme(ctx context.Context, userID string, req *v1.CreateThemeRequest) (*v1.Theme, error) {
	name, err := validateThemeName(req.GetName())
	if err != nil {
		return nil, err
	}
	desc, err := validateThemeDescription(req.GetDescription())
	if err != nil {
		return nil, err
	}
	if err := t.checkThemeLimit(ctx, userID); err != nil {
		return nil, err
	}
	content, err := t.normalizeUserContent(ctx, userID, req.GetContent())
	if err != nil {
		return nil, err
	}
	return t.createTheme(ctx, &gen.UserTheme{
		UserID:      userID,
		Name:        name,
		Author:      themeText(t.userDisplayName(ctx, userID), themeAuthorMaxRunes),
		Version:     themeDefaultVersion,
		Description: desc,
		Source:      usertheme.SourceCustom,
	}, content)
}

func (t *ThemeUsecase) createTheme(ctx context.Context, row *gen.UserTheme, content *v1.ThemeContent) (*v1.Theme, error) {
	raw, err := encodeThemeContent(content)
	if err != nil {
		return nil, ErrSystem(err)
	}
	row.ID = uuid.NewString()
	row.Content = raw
	saved, err := t.repo.CreateTheme(ctx, row)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return t.toTheme(ctx, saved)
}

func (t *ThemeUsecase) UpdateTheme(ctx context.Context, userID string, req *v1.UpdateThemeRequest) (*v1.Theme, error) {
	row, err := t.getOwnTheme(ctx, userID, req.GetId())
	if err != nil {
		return nil, err
	}
	name, err := validateThemeName(req.GetName())
	if err != nil {
		return nil, err
	}
	desc, err := validateThemeDescription(req.GetDescription())
	if err != nil {
		return nil, err
	}
	content, err := t.normalizeUserContent(ctx, userID, req.GetContent())
	if err != nil {
		return nil, err
	}
	raw, err := encodeThemeContent(content)
	if err != nil {
		return nil, ErrSystem(err)
	}
	row.Name, row.Description, row.Content = name, desc, raw
	saved, err := t.repo.UpdateTheme(ctx, row)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrThemeNotFound
		}
		return nil, ErrSystem(err)
	}
	t.collectAssetGarbage(ctx, userID)
	return t.toTheme(ctx, saved)
}

func (t *ThemeUsecase) DeleteTheme(ctx context.Context, userID, id string) error {
	if err := t.repo.DeleteTheme(ctx, userID, strings.TrimSpace(id)); err != nil {
		if gen.IsNotFound(err) {
			return ErrThemeNotFound
		}
		return ErrSystem(err)
	}
	t.collectAssetGarbage(ctx, userID)
	return nil
}

func (t *ThemeUsecase) DuplicateTheme(ctx context.Context, userID string, req *v1.DuplicateThemeRequest) (*v1.Theme, error) {
	src, err := t.getOwnTheme(ctx, userID, req.GetId())
	if err != nil {
		return nil, err
	}
	name := copyThemeName(src.Name)
	if strings.TrimSpace(req.GetName()) != "" {
		if name, err = validateThemeName(req.GetName()); err != nil {
			return nil, err
		}
	}
	if err := t.checkThemeLimit(ctx, userID); err != nil {
		return nil, err
	}
	return t.createTheme(ctx, &gen.UserTheme{
		UserID:      userID,
		Name:        name,
		Author:      themeText(t.userDisplayName(ctx, userID), themeAuthorMaxRunes),
		Version:     themeDefaultVersion,
		Description: src.Description,
		Source:      usertheme.SourceCustom,
	}, decodeThemeContent(src.Content))
}

// ---------------------------------------------------------------------------
// 资源上传
// ---------------------------------------------------------------------------

// decodeThemeBase64 解码 base64（容忍 data:…;base64, 前缀与换行）；超过 limit 直接拒绝。
func decodeThemeBase64(data string, limit int, tooLarge error) ([]byte, bool, error) {
	data = strings.TrimSpace(data)
	if strings.HasPrefix(data, "data:") {
		if i := strings.IndexByte(data, ','); i >= 0 {
			data = data[i+1:]
		}
	}
	if strings.ContainsAny(data, "\r\n") {
		data = strings.NewReplacer("\r", "", "\n", "").Replace(data)
	}
	if data == "" {
		return nil, false, nil
	}
	if base64.StdEncoding.DecodedLen(len(data)) > limit+3 {
		return nil, true, tooLarge
	}
	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		if raw, err = base64.RawStdEncoding.DecodeString(strings.TrimRight(data, "=")); err != nil {
			return nil, false, nil
		}
	}
	if len(raw) > limit {
		return nil, true, tooLarge
	}
	return raw, true, nil
}

// assetFileName 清理上传时的原文件名（只留文件名部分）。
func assetFileName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	return themeText(name, themeAssetNameMax)
}

func (t *ThemeUsecase) UploadAsset(ctx context.Context, userID string, req *v1.UploadThemeAssetRequest) (*v1.ThemeAsset, error) {
	var (
		processed *themeasset.Processed
		raw       []byte
		ok        bool
		err       error
	)
	switch strings.TrimSpace(req.GetKind()) {
	case themeasset.KindImage:
		raw, ok, err = decodeThemeBase64(req.GetData(), themeasset.MaxImageSize, ErrThemeAsset(themeasset.ErrImageTooLarge))
		if err != nil {
			return nil, err
		}
		if !ok || len(raw) == 0 {
			return nil, ErrThemeAssetData
		}
		processed, err = themeasset.ProcessImage(raw)
	case themeasset.KindFont:
		raw, ok, err = decodeThemeBase64(req.GetData(), themeasset.MaxFontSize, ErrThemeAsset(themeasset.ErrFontTooLarge))
		if err != nil {
			return nil, err
		}
		if !ok || len(raw) == 0 {
			return nil, ErrThemeAssetData
		}
		processed, err = themeasset.ProcessFont(raw)
	default:
		return nil, ErrThemeAssetKind
	}
	if err != nil {
		return nil, ErrThemeAsset(err)
	}
	row, err := t.saveAsset(ctx, userID, assetFileName(req.GetFileName()), processed)
	if err != nil {
		return nil, err
	}
	return toThemeAssetInfo(row), nil
}

// saveAsset 写入资源文件并登记；登记失败时删除文件。
func (t *ThemeUsecase) saveAsset(ctx context.Context, userID, name string, p *themeasset.Processed) (*gen.ThemeAsset, error) {
	id := themeasset.NewID(p.Ext)
	if err := t.assets.Save(id, p.Data); err != nil {
		return nil, ErrSystem(err)
	}
	row, err := t.repo.CreateAsset(ctx, &gen.ThemeAsset{
		ID:           id,
		UserID:       userID,
		Kind:         entthemeasset.Kind(p.Kind),
		Name:         name,
		Size:         int64(len(p.Data)),
		OriginalSize: p.OriginalSize,
		Width:        int32(p.Width),
		Height:       int32(p.Height),
	})
	if err != nil {
		_ = t.assets.Delete(id)
		return nil, ErrSystem(err)
	}
	return row, nil
}

// removeAssets 尽力删除资源（导入失败时回滚已保存的文件）。
func (t *ThemeUsecase) removeAssets(ctx context.Context, ids []string) {
	ctx = context.WithoutCancel(ctx)
	for _, id := range ids {
		_ = t.repo.DeleteAsset(ctx, id)
		_ = t.assets.Delete(id)
	}
}

// collectAssetGarbage 清理用户 24 小时前上传、且已不被任何主题 / 个人外观 / 全站默认引用的资源。
// 尽力而为，失败不影响请求。
func (t *ThemeUsecase) collectAssetGarbage(ctx context.Context, userID string) {
	ctx = context.WithoutCancel(ctx)
	stale, err := t.repo.ListStaleAssets(ctx, userID, time.Now().Add(-themeAssetGCGrace), themeAssetGCBatch)
	if err != nil || len(stale) == 0 {
		return
	}
	siteRefs, err := t.siteAssetRefs(ctx)
	if err != nil {
		return
	}
	for _, a := range stale {
		if _, ok := siteRefs[a.ID]; ok {
			continue
		}
		referenced, err := t.repo.AssetReferenced(ctx, a.ID)
		if err != nil || referenced {
			continue
		}
		if err := t.repo.DeleteAsset(ctx, a.ID); err != nil {
			continue
		}
		_ = t.assets.Delete(a.ID)
	}
}

// ---------------------------------------------------------------------------
// 全站默认主题
// ---------------------------------------------------------------------------

// SiteTheme 全站默认主题（没设置过为 nil）。
func (t *ThemeUsecase) SiteTheme(ctx context.Context) (*v1.SiteTheme, error) {
	site, err := t.config.SiteTheme(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return site, nil
}

func (t *ThemeUsecase) SetSiteTheme(ctx context.Context, userID string, req *v1.SetSiteThemeRequest) (*v1.SiteTheme, error) {
	if !req.GetEnabled() {
		if err := t.config.SaveSiteTheme(ctx, nil); err != nil {
			return nil, ErrSystem(err)
		}
		return nil, nil
	}
	// 全站默认对所有人生效：引用的资源只需存在（任何人上传的都可以）。
	resolver, err := t.assetResolver(ctx, req.GetContent(), nil)
	if err != nil {
		return nil, ErrSystem(err)
	}
	content := (&themeNormalizer{resolve: resolver}).content(req.GetContent())
	themeID := themeText(req.GetThemeId(), 64)
	name := themeText(req.GetName(), themeNameMaxRunes)
	if name == "" && themeID != "" {
		if row, err := t.repo.GetTheme(ctx, userID, themeID); err == nil {
			name = row.Name
		} else {
			name = themeID
		}
	}
	site := &v1.SiteTheme{
		ThemeId:   themeID,
		Name:      name,
		Content:   content,
		UpdatedAt: timestamppb.Now(),
		UpdatedBy: t.userDisplayName(ctx, userID),
	}
	if err := t.config.SaveSiteTheme(ctx, site); err != nil {
		return nil, ErrSystem(err)
	}
	return site, nil
}

// ---------------------------------------------------------------------------
// 个人外观
// ---------------------------------------------------------------------------

// NormalizeAppearance 规范化个人外观：theme 为空表示跟随全站默认；
// 非空时必须是内置配色或该用户自己的主题 id，否则落回 sakura。背景按用户可用的资源校验（nil 保持 nil）。
func (t *ThemeUsecase) NormalizeAppearance(ctx context.Context, userID string, a *v1.UserAppearance) (*v1.UserAppearance, error) {
	n := normalizeAppearance(a)
	if n.Theme != "" && !IsBuiltinTheme(n.Theme) {
		if _, err := t.repo.GetTheme(ctx, userID, n.Theme); err != nil {
			if !gen.IsNotFound(err) {
				return nil, ErrSystem(err)
			}
			n.Theme = "sakura"
		}
	}
	if bg := a.GetBackground(); bg != nil {
		resolver, err := t.userAssetResolver(ctx, userID, &v1.ThemeContent{Background: bg})
		if err != nil {
			return nil, ErrSystem(err)
		}
		n.Background = (&themeNormalizer{resolve: resolver}).background(bg)
	}
	return n, nil
}

// AppearanceThemes 个人信息附带的主题：当前用的「我的主题」（appearance.theme 为主题 id 时）与全站默认主题。
// 当前主题已被删除时把 appearance.theme 落回 sakura。查询失败只是不附带，不影响个人信息。
func (t *ThemeUsecase) AppearanceThemes(ctx context.Context, userID string, a *v1.UserAppearance) (current *v1.Theme, site *v1.SiteTheme) {
	site, _ = t.config.SiteTheme(ctx)
	if a == nil || a.Theme == "" || IsBuiltinTheme(a.Theme) {
		return nil, site
	}
	row, err := t.repo.GetTheme(ctx, userID, a.Theme)
	if err != nil {
		if gen.IsNotFound(err) {
			a.Theme = "sakura"
		}
		return nil, site
	}
	current, _ = t.toTheme(ctx, row)
	return current, site
}

// cloneThemeContent 深拷贝主题内容。
func cloneThemeContent(c *v1.ThemeContent) *v1.ThemeContent {
	if c == nil {
		return nil
	}
	return proto.Clone(c).(*v1.ThemeContent)
}
