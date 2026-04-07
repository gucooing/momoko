//go:build unix

package file

import (
	"io/fs"
	"os/user"
	"strconv"
	"syscall"
)

func fillOwnerInfo(info *v1.FileEntryInfo, e fs.FileInfo) {
	stat, ok := e.Sys().(*syscall.Stat_t)
	if !ok {
		return
	}
	info.UserId = uint32(stat.Uid)
	info.GroupId = uint32(stat.Gid)
	uidStr := strconv.FormatUint(uint64(stat.Uid), 10)
	gidStr := strconv.FormatUint(uint64(stat.Gid), 10)
	info.UserName = uidStr
	info.GroupName = gidStr
	if u, err := user.LookupId(uidStr); err == nil {
		info.UserName = u.Username
	}
	if g, err := user.LookupGroupId(gidStr); err == nil {
		info.GroupName = g.Name
	}
}
