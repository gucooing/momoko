package biz

import (
	"io/fs"
	"path"
	"strings"
	"sync"

	root "momoko"

	"momoko/internal/data/ent/gen"
	"momoko/pkg/mgp/gakumas"
)

// 游戏素材：前端产物 frontend/dist/games/<game>/<分组>/<名称>.<hash>.webp（scripts/games/<game>/ 导出，
// 前端按 src/games/<game>/assets/*.json 的同一份哈希引用），两个 server 都在 /games/ 下原样提供。
// 素材 key 为 <分组>/<名称>，例如 cards/idol/hski-3-000-land、ui/login-event2d-000。后端只在要把图片交给
// 游戏客户端（PC 启动器）时用这份索引；开发构建不嵌入前端产物，索引为空。

var (
	gameAssetsOnce sync.Once
	gameAssets     map[string]map[string]string // game → key → /games/... 路径
)

func loadGameAssets() {
	gameAssets = map[string]map[string]string{}
	games, err := fs.Sub(root.EmbeddedDist, "frontend/dist/games")
	if err != nil {
		return
	}
	_ = fs.WalkDir(games, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".webp") {
			return nil
		}
		game, rest, ok := strings.Cut(p, "/")
		dir, file := path.Split(rest)
		hashed := strings.TrimSuffix(file, ".webp")
		dot := strings.LastIndex(hashed, ".")
		if !ok || dir == "" || dot <= 0 {
			return nil
		}
		if gameAssets[game] == nil {
			gameAssets[game] = map[string]string{}
		}
		gameAssets[game][dir+hashed[:dot]] = "/games/" + p
		return nil
	})
}

// gameAssetPath 素材 key 的站内路径（/games/...）；素材不存在时为空串。
func gameAssetPath(game, key string) string {
	gameAssetsOnce.Do(loadGameAssets)
	return gameAssets[game][key]
}

// gakumasCoverIdols 学マス 默认封面：13 位偶像初始 SSR 卡（-3-000）的横图。
// 与前端 src/games/gakumas/assets.ts 的 COVER_IDOLS、coverKey 相同，两边必须一起改。
var gakumasCoverIdols = []string{"hski", "ttmr", "fktn", "hmsz", "kllj", "kcna", "ssmk", "shro", "hrnm", "hume", "jsna", "atbm", "amao"}

// serverCoverKey 服务器卡片上的图（素材 key）：设置了封面用封面；没有时学マス按服务器 ID 稳定地挑一张默认封面，
// 与前端 gameCover(game, 服务器 ID) 相同；其他游戏没有默认封面，返回空串。
func serverCoverKey(s *gen.GameServer) string {
	if s.Cover != "" {
		return s.Cover
	}
	if s.Game != gakumas.ID {
		return ""
	}
	var h uint32
	for _, c := range s.ID {
		h = h*31 + uint32(c)
	}
	return "cards/idol/" + gakumasCoverIdols[h%uint32(len(gakumasCoverIdols))] + "-3-000-land"
}

// serverIconPath 服务器图标的站内路径：服务器卡片上那张卡的方图（封面是偶像卡横图 …-land 时换成同一张卡的 …-sq），
// 没有方图时用封面本身；素材不存在时为空串。
func serverIconPath(s *gen.GameServer) string {
	key := serverCoverKey(s)
	if key == "" {
		return ""
	}
	if card, ok := strings.CutSuffix(key, "-land"); ok {
		if p := gameAssetPath(s.Game, card+"-sq"); p != "" {
			return p
		}
	}
	return gameAssetPath(s.Game, key)
}
