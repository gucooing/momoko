package momoko

import "embed"

// EmbeddedDist contains the frontend build output from the module-root dist directory.
//
//go:embed all:dist
var EmbeddedDist embed.FS
