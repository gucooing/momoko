//go:build !linux

package sysmon

import (
	"context"

	gnet "github.com/shirou/gopsutil/v4/net"
)

// isPhysicalDisk 非 Linux 平台（Windows 按盘符）直接统计全部条目。
func isPhysicalDisk(string) bool { return true }

func tcpEstablished(ctx context.Context) (int, bool) {
	conns, err := gnet.ConnectionsWithContext(ctx, "tcp")
	if err != nil {
		return 0, false
	}
	n := 0
	for _, c := range conns {
		if c.Status == "ESTABLISHED" {
			n++
		}
	}
	return n, true
}
