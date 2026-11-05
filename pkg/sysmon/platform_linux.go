//go:build linux

package sysmon

import (
	"bufio"
	"context"
	"os"
	"strings"
)

// isPhysicalDisk 只统计 /sys/block 下带 device 的整块磁盘：
// 分区（sda1）、loop、device-mapper（dm-*）、md 阵列等逻辑设备的流量已经算在底层磁盘上。
func isPhysicalDisk(name string) bool {
	if strings.HasPrefix(name, "sr") || strings.HasPrefix(name, "fd") || strings.HasPrefix(name, "zram") {
		return false
	}
	_, err := os.Stat("/sys/block/" + name + "/device")
	return err == nil
}

// tcpEstablished 直接数 /proc/net/tcp{,6} 里 ESTABLISHED（01）的行，避免遍历全部进程的 fd。
func tcpEstablished(_ context.Context) (int, bool) {
	total, ok := 0, false
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		ok = true
		sc := bufio.NewScanner(f)
		first := true
		for sc.Scan() {
			if first {
				first = false
				continue
			}
			fields := strings.Fields(sc.Text())
			if len(fields) > 3 && fields[3] == "01" {
				total++
			}
		}
		_ = f.Close()
	}
	return total, ok
}
