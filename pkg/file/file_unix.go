//go:build unix

package file

import (
	"io/fs"
	"os"
	"os/user"
	"strconv"
	"syscall"

	v1 "momoko/api/gen/v1"
)

// copyOwner 把 ref 的属主/属组套到 path 上；两者已一致时跳过（非特权进程只能 chown 给自己，避免无谓失败）。
func copyOwner(path string, ref fs.FileInfo) error {
	want, ok := ref.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	cur, err := os.Stat(path)
	if err != nil {
		return err
	}
	if st, ok := cur.Sys().(*syscall.Stat_t); ok && st.Uid == want.Uid && st.Gid == want.Gid {
		return nil
	}
	return os.Chown(path, int(want.Uid), int(want.Gid))
}

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
