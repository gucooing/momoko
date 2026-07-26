//go:build unix

package localfs

import (
	"io/fs"
	"os/user"
	"strconv"
	"syscall"
)

// fillOwner 在类 Unix 上补齐属主信息。查不到名字时保留数字 uid/gid 作为展示值。
func fillOwner(e *Entry, info fs.FileInfo) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return
	}
	e.UID = uint32(stat.Uid)
	e.GID = uint32(stat.Gid)
	e.User = strconv.FormatUint(uint64(e.UID), 10)
	e.Group = strconv.FormatUint(uint64(e.GID), 10)
	if u, err := user.LookupId(e.User); err == nil {
		e.User = u.Username
	}
	if g, err := user.LookupGroupId(e.Group); err == nil {
		e.Group = g.Name
	}
}
