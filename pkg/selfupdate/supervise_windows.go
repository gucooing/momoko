//go:build windows

package selfupdate

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ShouldSupervise 这次启动要不要当守护进程（不是被守护进程启动的子进程时）。
func ShouldSupervise() bool { return !supervised }

// Supervise 当守护进程：启动真正干活的子进程并等它退出；子进程因为更新退出（RestartExitCode）时启动新程序，
// 其它情况返回子进程的退出码。守护进程不加载配置、不开端口，只占几兆内存。
func Supervise() int {
	exe, err := executablePath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "找不到当前程序文件：", err)
		return 1
	}
	// Ctrl+C、关窗口这些信号子进程在同一个控制台里也会收到，由它正常停服；守护进程只管等它退出
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		for range sig {
		}
	}()
	// 守护进程被强行结束（任务管理器、服务包装器结束进程）时，子进程跟着结束，不留下没人管的服务
	job := newKillOnCloseJob()
	wd, _ := os.Getwd()

	afterUpdate := false
	for {
		cmd := exec.Command(exe, os.Args[1:]...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		cmd.Env = append(os.Environ(), supervisedEnv+"=1")
		cmd.Dir = wd
		if err := cmd.Start(); err != nil {
			// 刚换上的新程序启动不了：换回旧程序再试一次（安装前已经跑过 -version，一般不会走到这里）
			if afterUpdate {
				afterUpdate = false
				if rerr := restoreBackup(exe); rerr == nil {
					fmt.Fprintf(os.Stderr, "新程序启动失败（%v），已换回旧程序\n", err)
					continue
				}
			}
			fmt.Fprintf(os.Stderr, "启动 %s 失败：%v\n", exe, err)
			return 1
		}
		job.add(cmd.Process.Pid)
		err := cmd.Wait()
		code := 1
		var exitErr *exec.ExitError
		if err == nil || errors.As(err, &exitErr) {
			code = cmd.ProcessState.ExitCode()
		}
		if code != RestartExitCode {
			return code
		}
		afterUpdate = true
	}
}

// killOnCloseJob 是一个「句柄关闭时结束里面所有进程」的作业对象；创建失败时为 nil（不影响运行）。
type killOnCloseJob struct{ h windows.Handle }

func newKillOnCloseJob() *killOnCloseJob {
	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil
	}
	var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(h, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		_ = windows.CloseHandle(h)
		return nil
	}
	return &killOnCloseJob{h: h}
}

func (j *killOnCloseJob) add(pid int) {
	if j == nil {
		return
	}
	ph, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return
	}
	defer windows.CloseHandle(ph)
	_ = windows.AssignProcessToJobObject(j.h, ph)
}
