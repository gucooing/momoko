//go:build unix

package filecore

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
)

func lookupFileIdentity(info os.FileInfo) fileIdentity {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return fileIdentity{}
	}

	identity := fileIdentity{
		userID:  stat.Uid,
		groupID: stat.Gid,
	}

	if u, err := user.LookupId(strconv.FormatUint(uint64(stat.Uid), 10)); err == nil {
		identity.userName = u.Username
	}
	if g, err := user.LookupGroupId(strconv.FormatUint(uint64(stat.Gid), 10)); err == nil {
		identity.groupName = g.Name
	}
	return identity
}
