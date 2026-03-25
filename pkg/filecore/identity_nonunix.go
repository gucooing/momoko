//go:build !unix

package filecore

import "os"

func lookupFileIdentity(info os.FileInfo) fileIdentity {
	_ = info
	return fileIdentity{}
}
