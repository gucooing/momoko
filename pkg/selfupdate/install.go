package selfupdate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// executablePath 当前运行的程序文件（解开符号链接，替换的是真正的文件）。
func executablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return filepath.Abs(exe)
}

// checkWritable 程序所在目录要能新建和改名文件，否则没法替换程序。
func checkWritable(dir string) error {
	f, err := os.CreateTemp(dir, ".momoko-update-check-*")
	if err != nil {
		return fmt.Errorf("程序所在目录 %s 不可写，没法在线更新：%w", dir, err)
	}
	name := f.Name()
	_ = f.Close()
	return os.Remove(name)
}

// backupPath 被替换下来的旧程序，更新后起不来时可以手动改回去。
func backupPath(exe string) string { return exe + ".old" }

// staleBackupPattern 删不掉的旧备份先改名成这个样子，以后启动时再删。
func staleBackupPattern(exe string) string { return backupPath(exe) + ".del-*" }

// replaceExecutable 用 staged 换掉 exe，旧程序改名为 exe.old。
// 运行中的程序文件在 Linux / macOS 上可以直接改名（进程用的是 inode），Windows 上也允许改名正在运行的 exe。
func replaceExecutable(exe, staged string) error {
	if st, err := os.Stat(exe); err == nil {
		_ = os.Chmod(staged, st.Mode().Perm()|0o100)
	}
	old := backupPath(exe)
	if err := os.Remove(old); err != nil && !errors.Is(err, os.ErrNotExist) {
		// Windows 上守护进程就是从这个文件跑起来的（第一次更新后它被改名成了 .old），删不掉但能改名，挪开等以后再删
		aside := fmt.Sprintf("%s.del-%d", old, time.Now().UnixNano())
		if rerr := os.Rename(old, aside); rerr != nil {
			return fmt.Errorf("删不掉上次留下的备份 %s：%w", old, err)
		}
	}
	if err := renameRetry(exe, old); err != nil {
		return fmt.Errorf("备份当前程序失败：%w", err)
	}
	if err := renameRetry(staged, exe); err != nil {
		if rerr := os.Rename(old, exe); rerr != nil {
			return fmt.Errorf("换上新程序失败（%v），还原旧程序也失败了，请手动把 %s 改回 %s：%w", err, old, exe, rerr)
		}
		return fmt.Errorf("换上新程序失败：%w", err)
	}
	return nil
}

// renameRetry 改名失败时再试几秒：Windows 上刚写好、刚运行过的程序文件可能正被杀毒软件扫着。
func renameRetry(from, to string) error {
	deadline := time.Now().Add(5 * time.Second)
	for {
		err := os.Rename(from, to)
		if err == nil || runtime.GOOS != "windows" || time.Now().After(deadline) {
			return err
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// removeStaleBackups 删掉以前挪开的旧备份（还被守护进程占着的删不掉，下次再说）。
func removeStaleBackups(exe string) {
	files, _ := filepath.Glob(staleBackupPattern(exe))
	for _, f := range files {
		_ = os.Remove(f)
	}
}

// restoreBackup 新程序启动不了时，把备份的旧程序换回来。
func restoreBackup(exe string) error {
	old := backupPath(exe)
	if _, err := os.Stat(old); err != nil {
		return err
	}
	_ = os.Remove(exe)
	return os.Rename(old, exe)
}

// restart 是「程序已替换，请重启」的信号：main 收到后正常停掉服务，再调用 Relaunch。
var restart = struct {
	once sync.Once
	ch   chan struct{}
	exe  string
}{ch: make(chan struct{})}

func requestRestart(exe string) {
	restart.once.Do(func() {
		restart.exe = exe
		close(restart.ch)
	})
}

// RestartRequested 程序已替换、等待重启时关闭。
func RestartRequested() <-chan struct{} { return restart.ch }

// RestartPending 是否在等重启（服务停下后由 main 判断要不要 Relaunch）。
func RestartPending() bool {
	select {
	case <-restart.ch:
		return true
	default:
		return false
	}
}

// Relaunch 服务停下后启动新程序：Linux / macOS 用 exec 原地换成新程序（进程号不变，systemd、Docker 照常管理），
// Windows 启动新进程后退出当前进程。启动失败时换回旧程序再试一次；成功时不返回。
func Relaunch() error {
	exe := restart.exe
	if exe == "" {
		return errors.New("没有待重启的更新")
	}
	err := relaunch(exe)
	if rerr := restoreBackup(exe); rerr != nil {
		return fmt.Errorf("启动新程序失败：%w；还原旧程序也失败：%v", err, rerr)
	}
	if err2 := relaunch(exe); err2 != nil {
		return fmt.Errorf("启动新程序失败：%w；换回旧程序后也启动失败：%v", err, err2)
	}
	return nil
}
