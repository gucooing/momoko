//go:build !unix

package file

import (
	"io/fs"

	v1 "momoko/api/gen/v1"
)

// fillOwnerInfo 在非 unix 平台无属主/属组概念，保持空实现。
func fillOwnerInfo(_ *v1.FileEntryInfo, _ fs.FileInfo) {}
