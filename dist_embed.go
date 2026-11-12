package momoko

import "embed"

// EmbeddedDist contains the frontend build output from the module-root dist directory.
//
//go:embed all:frontend/dist
var EmbeddedDist embed.FS

// EmbeddedPassportDist 用户通行证（玩家端）前端的构建产物，只由通行证 server 提供。
//
//go:embed all:frontend/dist-passport
var EmbeddedPassportDist embed.FS
