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
		// Windows 上旧备份还被占着（比如有人在运行它）时删不掉但能改名，挪开等以后再删
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

// removeStaleBackups 删掉以前挪开的旧备份（还被占着的删不掉，下次再说）。
func removeStaleBackups(exe string) {
	files, _ := filepath.Glob(staleBackupPattern(exe))
	for _, f := range files {
		_ = os.Remove(f)
	}
}

// RestartExitCode 换好程序、正常停服后进程的退出码（EX_TEMPFAIL）。系统服务以非 0 退出时，
// 服务管理器（systemd Restart=always、launchd KeepAlive、Windows 服务的故障恢复）会马上把新程序拉起来。
const RestartExitCode = 75

// restart 是「程序已替换，请重启」的信号：main 收到后正常停掉服务，以 RestartExitCode 退出。
var restart = struct {
	once sync.Once
	ch   chan struct{}
}{ch: make(chan struct{})}

func requestRestart() {
	restart.once.Do(func() { close(restart.ch) })
}

// RestartRequested 程序已替换、等待重启时关闭。
func RestartRequested() <-chan struct{} { return restart.ch }

// RestartPending 是否在等重启（服务停下后由 main 判断要不要以 RestartExitCode 退出）。
func RestartPending() bool {
	select {
	case <-restart.ch:
		return true
	default:
		return false
	}
}
