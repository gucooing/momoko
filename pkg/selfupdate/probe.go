package selfupdate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// probeTimeout 新程序跑 -version 最多等多久（Windows 上杀毒软件第一次扫描新程序会慢一些）。
const probeTimeout = 30 * time.Second

// VersionLine 是 `momoko -version` 输出的那一行：「momoko v0.2.0 linux/amd64」。
// 在线更新安装前会运行新程序的 -version，读到这一行才算新程序在本机能正常启动。
func VersionLine(name, ver string) string {
	if name == "" {
		name = "momoko"
	}
	return name + " " + ver + " " + Platform()
}

// ParseVersionLine 从 -version 的输出里找出版本号和系统/架构。
func ParseVersionLine(out string) (ver, platform string, ok bool) {
	for line := range strings.SplitSeq(out, "\n") {
		f := strings.Fields(line)
		if len(f) == 3 && f[0] == "momoko" && strings.Contains(f[2], "/") {
			return f[1], f[2], true
		}
	}
	return "", "", false
}

// probeBinary 在一个空的临时目录里运行 file -version：能在本机启动、报出版本号且系统架构对得上，才算能用。
// -version 在读配置之前就退出，不碰数据和端口。
func probeBinary(file string) (string, error) {
	dir, err := os.MkdirTemp("", "momoko-probe-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	var out limitedBuffer
	cmd := exec.CommandContext(ctx, file, "-version")
	cmd.Dir = dir
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.WaitDelay = 3 * time.Second
	runErr := cmd.Run()
	text := strings.TrimSpace(out.String())
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return "", fmt.Errorf("新程序运行 -version 超过 %d 秒没有结束，不像是正常的 Momoko 程序", int(probeTimeout/time.Second))
	case runErr != nil && strings.Contains(text, "flag provided but not defined"):
		return "", errors.New("新程序不认识 -version 参数（是还没有在线更新功能的老版本），确认不了它在本机能正常运行，请手动替换程序")
	case runErr != nil:
		return "", fmt.Errorf("新程序在本机运行失败（%v）：%s", runErr, firstLine(text))
	}
	ver, platform, ok := ParseVersionLine(text)
	if !ok {
		return "", fmt.Errorf("新程序运行 -version 的输出看不懂：%s", firstLine(text))
	}
	if platform != Platform() {
		return "", fmt.Errorf("新程序报告自己是 %s 的，本机是 %s", platform, Platform())
	}
	return ver, nil
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	if len(line) > 200 {
		line = line[:200] + "…"
	}
	if line == "" {
		return "没有输出"
	}
	return line
}

// limitedBuffer 只留前 64 KB 输出，防止新程序刷屏把内存撑满。
type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if room := 64<<10 - b.Len(); room > 0 {
		b.Buffer.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}
