//go:build !unix

package localfs

import "io/fs"

// fillOwner 在非 Unix 平台无属主概念，留空。
func fillOwner(_ *Entry, _ fs.FileInfo) {}
