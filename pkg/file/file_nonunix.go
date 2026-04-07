//go:build !unix

package file

import (
	"io/fs"

	v1 "momoko/api/gen/v1"
)

func fillOwnerInfo(info *v1.FileEntryInfo, d fs.FileInfo) {
	return
}
