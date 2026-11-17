package biz

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/encoding/protojson"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/common"
)

// 启动器设置：PC 启动器的公告，以及各游戏在启动器里的名称、字标和游戏页背景。
// 后台在「用户通行证 → 启动器」里编辑（GET/PUT /api/v1/game/passport/launcher），存在系统配置 passport.launcher
// （v1.LauncherSettings 的 protojson）。背景从游戏素材里选，保存的是素材 key（见 game_asset.go）。
// 通行证 GET /api/v1/launcher 不用登录：启动器在登录页之前读取，背景换成通行证对外地址 + /games/... 的绝对地址。
// 服务器图标不在这里设置：跟随服务器封面（PassportLaunchApp.icon）。

const (
	launcherNoticeMax      = 500
	launcherNameMax        = 40
	launcherWordmarkMax    = 40
	launcherAssetKeyMax    = 200
	launcherBackgroundsMax = 3
)

// launcherGames 可以在启动器里设置展示的游戏：支持客户端启动的适配器（顺序即适配器顺序）。
func launcherGames() []string {
	out := []string{}
	for _, a := range GameAdapters() {
		if a.ClientLaunch {
			out = append(out, a.ID)
		}
	}
	return out
}

// launcherSettings 读取启动器设置；没有保存过（或无法解析）时为空设置。
func launcherSettings(ctx context.Context, config ConfigRepo, logger *log.Helper) *v1.LauncherSettings {
	out := &v1.LauncherSettings{Games: []*v1.LauncherGame{}}
	raw, err := config.Get(ctx, common.ConfigPassportLauncher)
	if err != nil || strings.TrimSpace(raw) == "" {
		return out
	}
	saved := &v1.LauncherSettings{}
	if err := protojson.Unmarshal([]byte(raw), saved); err != nil {
		logger.Warnf("parse launcher settings: %v", err)
		return out
	}
	if saved.Games == nil {
		saved.Games = []*v1.LauncherGame{}
	}
	for _, g := range saved.Games {
		if g.Backgrounds == nil {
			g.Backgrounds = []string{}
		}
	}
	return saved
}

// normalizeLauncherSettings 整理并校验启动器设置：去掉首尾空白；公告最多 500 字；每款游戏必须支持客户端启动且只出现一次，
// 名称必填（最多 40 字），字标最多 40 字；背景是这款游戏的素材 key，不重复，最多 3 张。
func normalizeLauncherSettings(s *v1.LauncherSettings) error {
	s.Notice = strings.TrimSpace(s.Notice)
	if utf8.RuneCountInString(s.Notice) > launcherNoticeMax {
		return ErrPassportLauncherInvalid
	}
	if s.Games == nil {
		s.Games = []*v1.LauncherGame{}
	}
	seen := map[string]bool{}
	for _, g := range s.Games {
		if g == nil {
			return ErrPassportLauncherInvalid
		}
		g.Game = strings.TrimSpace(g.Game)
		if a, ok := GameAdapterByID(g.Game); !ok || !a.ClientLaunch || seen[g.Game] {
			return ErrPassportLauncherInvalid
		}
		seen[g.Game] = true
		g.Name, g.Wordmark = strings.TrimSpace(g.Name), strings.TrimSpace(g.Wordmark)
		if g.Name == "" || utf8.RuneCountInString(g.Name) > launcherNameMax || utf8.RuneCountInString(g.Wordmark) > launcherWordmarkMax {
			return ErrPassportLauncherInvalid
		}
		if g.Backgrounds == nil {
			g.Backgrounds = []string{}
		}
		if len(g.Backgrounds) > launcherBackgroundsMax {
			return ErrPassportLauncherInvalid
		}
		picked := map[string]bool{}
		for i, key := range g.Backgrounds {
			key = strings.TrimSpace(key)
			if !validAssetKey(key) || picked[key] {
				return ErrPassportLauncherInvalid
			}
			picked[key] = true
			g.Backgrounds[i] = key
		}
	}
	return nil
}

// validAssetKey 素材 key 的格式：<分组>/<名称>，只含小写字母、数字、-、_ 和分隔的 /，不以 / 开头或结尾。
func validAssetKey(key string) bool {
	if len(key) == 0 || len(key) > launcherAssetKeyMax || !strings.Contains(key, "/") ||
		strings.HasPrefix(key, "/") || strings.HasSuffix(key, "/") || strings.Contains(key, "//") {
		return false
	}
	for _, c := range key {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '/') {
			return false
		}
	}
	return true
}

// LauncherConfig 公开的启动器配置（启动器在登录前读取）。publicURL 为通行证对外地址（不带结尾的 /）；
// 为空或素材不存在时，对应的背景不返回。
func (uc *PassportUsecase) LauncherConfig(ctx context.Context, publicURL string) *v1.PassportLauncherConfig {
	s := launcherSettings(ctx, uc.config, uc.log)
	out := &v1.PassportLauncherConfig{Notice: s.Notice, Games: make([]*v1.PassportLauncherGame, 0, len(s.Games))}
	for _, g := range s.Games {
		item := &v1.PassportLauncherGame{
			Game:        g.Game,
			Name:        g.Name,
			Wordmark:    g.Wordmark,
			Backgrounds: make([]string, 0, len(g.Backgrounds)),
		}
		for _, key := range g.Backgrounds {
			if p := gameAssetPath(g.Game, key); p != "" && publicURL != "" {
				item.Backgrounds = append(item.Backgrounds, publicURL+p)
			}
		}
		out.Games = append(out.Games, item)
	}
	return out
}

// AdminLauncherSettings 启动器设置、可以设置的游戏与通行证对外地址。
func (uc *PassportAdminUsecase) AdminLauncherSettings(ctx context.Context, info PassportServerInfo) *v1.GetLauncherSettingsResponse {
	return &v1.GetLauncherSettingsResponse{
		Settings:  launcherSettings(ctx, uc.config, uc.log),
		Games:     launcherGames(),
		PublicUrl: info.PublicURL,
	}
}

// SaveLauncherSettings 校验并保存启动器设置。
func (uc *PassportAdminUsecase) SaveLauncherSettings(ctx context.Context, s *v1.LauncherSettings) error {
	if err := normalizeLauncherSettings(s); err != nil {
		return err
	}
	raw, err := protojson.Marshal(s)
	if err != nil {
		return ErrSystem(err)
	}
	if err := uc.config.BatchUpdate(ctx, map[common.ConfigKey]string{common.ConfigPassportLauncher: string(raw)}); err != nil {
		return ErrSystem(err)
	}
	return nil
}
