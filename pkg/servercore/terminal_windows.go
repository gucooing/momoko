//go:build windows

package servercore

import (
	"fmt"
	"os"
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

// startTerminalProcess 在 Windows 下使用 ConPTY 启动 cmd，
// 让交互输入走真实控制台链路，而不是普通重定向管道。
func startTerminalProcess(s *Server) (*startResult, error) {
	commandPath, err := exec.LookPath(s.cfg.Command)
	if err != nil {
		commandPath = s.cfg.Command
	}

	commandLine, err := windows.UTF16PtrFromString(
		windows.ComposeCommandLine(append([]string{commandPath}, s.cfg.Args...)),
	)
	if err != nil {
		return nil, fmt.Errorf("构造命令行失败: %w", err)
	}

	var currentDir *uint16
	if s.cfg.Dir != "" {
		currentDir, err = windows.UTF16PtrFromString(s.cfg.Dir)
		if err != nil {
			return nil, fmt.Errorf("构造工作目录失败: %w", err)
		}
	}

	var (
		ptyInputRead   windows.Handle
		hostInputWrite windows.Handle
		hostOutputRead windows.Handle
		ptyOutputWrite windows.Handle
		pseudoConsole  windows.Handle
	)

	if err := windows.CreatePipe(&ptyInputRead, &hostInputWrite, nil, 0); err != nil {
		return nil, fmt.Errorf("创建终端输入管道失败: %w", err)
	}
	defer closeWindowsHandle(&ptyInputRead)
	defer closeWindowsHandle(&hostInputWrite)

	if err := windows.CreatePipe(&hostOutputRead, &ptyOutputWrite, nil, 0); err != nil {
		return nil, fmt.Errorf("创建终端输出管道失败: %w", err)
	}
	defer closeWindowsHandle(&hostOutputRead)
	defer closeWindowsHandle(&ptyOutputWrite)

	if err := windows.CreatePseudoConsole(
		windows.Coord{X: 120, Y: 30},
		ptyInputRead,
		ptyOutputWrite,
		0,
		&pseudoConsole,
	); err != nil {
		return nil, fmt.Errorf("创建伪终端失败: %w", err)
	}
	defer closePseudoConsoleHandle(&pseudoConsole)

	attributeList, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return nil, fmt.Errorf("创建进程属性列表失败: %w", err)
	}
	defer func() {
		if attributeList != nil {
			attributeList.Delete()
		}
	}()

	// 这里按官方 ConPTY 示例直接传入 HPCON 句柄值本身。
	if err := attributeList.Update(
		windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE,
		unsafe.Pointer(pseudoConsole),
		unsafe.Sizeof(pseudoConsole),
	); err != nil {
		return nil, fmt.Errorf("绑定伪终端属性失败: %w", err)
	}

	startupInfo := windows.StartupInfoEx{}
	startupInfo.Cb = uint32(unsafe.Sizeof(startupInfo))
	startupInfo.Flags = windows.STARTF_USESTDHANDLES
	startupInfo.StdInput = windows.InvalidHandle
	startupInfo.StdOutput = windows.InvalidHandle
	startupInfo.StdErr = windows.InvalidHandle
	startupInfo.ProcThreadAttributeList = attributeList.List()

	var procInfo windows.ProcessInformation
	if err := windows.CreateProcess(
		nil,
		commandLine,
		nil,
		nil,
		false,
		windows.EXTENDED_STARTUPINFO_PRESENT,
		nil,
		currentDir,
		&startupInfo.StartupInfo,
		&procInfo,
	); err != nil {
		return nil, fmt.Errorf("启动终端进程失败: %w", err)
	}

	// 伪终端对象已经持有了这两个句柄，创建子进程后父进程应当释放它们。
	closeWindowsHandle(&ptyInputRead)
	closeWindowsHandle(&ptyOutputWrite)

	inputFile := os.NewFile(uintptr(hostInputWrite), "conpty-input")
	outputFile := os.NewFile(uintptr(hostOutputRead), "conpty-output")
	pseudoConsoleHandle := pseudoConsole
	attributeListHandle := attributeList
	hostInputWrite = 0
	hostOutputRead = 0
	pseudoConsole = 0
	attributeList = nil

	return &startResult{
		stdin:  inputFile,
		stdout: outputFile,
		waitFn: func() error {
			_, err := windows.WaitForSingleObject(procInfo.Process, windows.INFINITE)
			return err
		},
		stopFn: func(force bool) error {
			if !force && pseudoConsoleHandle != 0 {
				windows.ClosePseudoConsole(pseudoConsoleHandle)
				pseudoConsoleHandle = 0
				return nil
			}
			if procInfo.Process == 0 {
				return nil
			}
			return windows.TerminateProcess(procInfo.Process, 1)
		},
		closeFn: func() {
			if inputFile != nil {
				_ = inputFile.Close()
			}
			if outputFile != nil {
				_ = outputFile.Close()
			}
			if procInfo.Thread != 0 {
				_ = windows.CloseHandle(procInfo.Thread)
			}
			if procInfo.Process != 0 {
				_ = windows.CloseHandle(procInfo.Process)
			}
			if attributeListHandle != nil {
				attributeListHandle.Delete()
				attributeListHandle = nil
			}
			closePseudoConsoleHandle(&pseudoConsoleHandle)
			closeWindowsHandle(&ptyInputRead)
			closeWindowsHandle(&ptyOutputWrite)
			closeWindowsHandle(&hostInputWrite)
			closeWindowsHandle(&hostOutputRead)
		},
		stdinLineEnd: "\r\n",
		rawOutput:    true,
	}, nil
}

// closeWindowsHandle 关闭普通 Windows 句柄，并把句柄值清零避免重复关闭。
func closeWindowsHandle(handle *windows.Handle) {
	if handle == nil || *handle == 0 {
		return
	}
	_ = windows.CloseHandle(*handle)
	*handle = 0
}

// closePseudoConsoleHandle 关闭 ConPTY 伪终端句柄，并把句柄值清零。
func closePseudoConsoleHandle(handle *windows.Handle) {
	if handle == nil || *handle == 0 {
		return
	}
	windows.ClosePseudoConsole(*handle)
	*handle = 0
}
