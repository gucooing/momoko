package momoko

import "embed"

// EmbeddedDist contains the frontend build output from the module-root dist directory.
//
//go:embed all:frontend/dist
var EmbeddedDist embed.FS
