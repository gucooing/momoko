//go:build windows

package sysservice

import (
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func isWindowsService() bool {
	ok, err := svc.IsWindowsService()
	return err == nil && ok
}

// IsAdmin 当前进程有没有管理员权限（UAC 提升过）。
func IsAdmin() bool { return windows.GetCurrentProcessToken().IsElevated() }

// DefaultDir 默认安装目录：C:\Program Files\Momoko。
func DefaultDir() string {
	base := os.Getenv("ProgramFiles")
	if base == "" {
		base = `C:\Program Files`
	}
	return filepath.Join(base, DisplayName)
}

// Supported Windows 都能装。
func Supported() error { return nil }

// ValidateDir 安装目录要是带盘符的绝对路径。
func ValidateDir(dir string) error {
	if !filepath.IsAbs(dir) || filepath.VolumeName(dir) == "" {
		return fmt.Errorf("安装目录要写完整路径（带盘符），比如 %s", DefaultDir())
	}
	return nil
}

// serviceSDDL 服务的权限：系统和管理员完全控制；本机登录的用户（交互式登录、远程桌面）除了查询还能启动、停止，
// 所以普通终端里 momoko start / stop / restart 不用 UAC 提权。改配置、删除服务仍然只有管理员能做。
const serviceSDDL = "D:" +
	"(A;;CCLCSWRPWPDTLOCRRC;;;SY)" +
	"(A;;CCDCLCSWRPWPDTLOCRSDRCWDWO;;;BA)" +
	"(A;;CCLCSWRPWPLOCRRC;;;IU)" +
	"(A;;CCLCSWLOCRRC;;;SU)"

// openService 用最小的权限打开服务（不用管理员也能查询；启动、停止靠上面给本机用户开的权限）。
func openService(access uint32) (*mgr.Service, func(), error) {
	h, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return nil, nil, fmt.Errorf("连接服务管理器失败：%w", err)
	}
	name, _ := windows.UTF16PtrFromString(Name)
	sh, err := windows.OpenService(h, name, access)
	if err != nil {
		_ = windows.CloseServiceHandle(h)
		return nil, nil, err
	}
	s := &mgr.Service{Name: Name, Handle: sh}
	return s, func() { _ = s.Close(); _ = windows.CloseServiceHandle(h) }, nil
}

func toState(st svc.State) State {
	switch st {
	case svc.Stopped:
		return StateStopped
	case svc.StartPending, svc.ContinuePending:
		return StateStarting
	case svc.Running:
		return StateRunning
	case svc.StopPending, svc.PausePending, svc.Paused:
		return StateStopping
	default:
		return StateUnknown
	}
}

// Lookup 查已安装的服务；没装时 ok 为 false。
func Lookup() (Info, bool, error) {
	s, done, err := openService(windows.SERVICE_QUERY_CONFIG | windows.SERVICE_QUERY_STATUS)
	if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
		return Info{}, false, nil
	}
	if err != nil {
		return Info{}, false, err
	}
	defer done()
	cfg, err := s.Config()
	if err != nil {
		return Info{}, false, err
	}
	exe := binaryPath(cfg.BinaryPathName)
	info := Info{Exe: exe, Dir: filepath.Dir(exe)}
	if st, err := s.Query(); err == nil {
		info.State = toState(st.State)
	}
	return info, true, nil
}

// binaryPath 从服务的命令行里取出程序路径（可能带引号和参数）。
func binaryPath(cmdline string) string {
	cmdline = strings.TrimSpace(cmdline)
	if rest, ok := strings.CutPrefix(cmdline, `"`); ok {
		p, _, _ := strings.Cut(rest, `"`)
		return p
	}
	p, _, _ := strings.Cut(cmdline, " ")
	return p
}

// Register 注册（或更新）服务：开机自动启动、以 LocalSystem 运行；进程异常退出、或在线更新后以非 0 退出码退出时，
// 2 秒后由服务管理器重新启动。要管理员权限。
func Register(dir, exe string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("连接服务管理器失败（要管理员权限）：%w", err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(Name)
	if err == nil {
		cfg, cerr := s.Config()
		if cerr == nil {
			cfg.BinaryPathName = syscall.EscapeArg(exe)
			cfg.StartType = mgr.StartAutomatic
			cfg.DisplayName = DisplayName
			cfg.Description = Description
			cfg.ServiceStartName = "LocalSystem"
			cfg.ErrorControl = mgr.ErrorNormal
			err = s.UpdateConfig(cfg)
		} else {
			err = cerr
		}
	} else {
		s, err = m.CreateService(Name, exe, mgr.Config{
			StartType:    mgr.StartAutomatic,
			ErrorControl: mgr.ErrorNormal,
			DisplayName:  DisplayName,
			Description:  Description,
		})
	}
	if err != nil {
		return fmt.Errorf("注册服务失败：%w", err)
	}
	defer s.Close()
	restart := mgr.RecoveryAction{Type: mgr.ServiceRestart, Delay: 2 * time.Second}
	if err := s.SetRecoveryActions([]mgr.RecoveryAction{restart, restart, restart}, 24*60*60); err != nil {
		return fmt.Errorf("设置服务自动重启失败：%w", err)
	}
	// 正常停下但退出码不是 0（在线更新换好程序后）也按失败处理、重新启动
	if err := s.SetRecoveryActionsOnNonCrashFailures(true); err != nil {
		return fmt.Errorf("设置服务自动重启失败：%w", err)
	}
	sd, err := windows.SecurityDescriptorFromString(serviceSDDL)
	if err != nil {
		return err
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return err
	}
	if err := windows.SetSecurityInfo(s.Handle, windows.SE_SERVICE, windows.DACL_SECURITY_INFORMATION, nil, nil, dacl, nil); err != nil {
		return fmt.Errorf("设置服务权限失败：%w", err)
	}
	return os.MkdirAll(filepath.Join(dir, "logs"), 0o755)
}

// Unregister 停止并删除服务。要管理员权限。
func Unregister() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("连接服务管理器失败（要管理员权限）：%w", err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(Name)
	if err != nil {
		return fmt.Errorf("打开服务失败：%w", err)
	}
	defer s.Close()
	if err := Stop(); err != nil {
		return err
	}
	if err := s.Delete(); err != nil {
		return fmt.Errorf("删除服务失败：%w", err)
	}
	return nil
}

// ErrAccessDenied 没有权限控制服务。
var ErrAccessDenied = errors.New("没有权限控制 Momoko 服务")

func queryState() (State, error) {
	s, done, err := openService(windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return StateUnknown, err
	}
	defer done()
	st, err := s.Query()
	if err != nil {
		return StateUnknown, err
	}
	return toState(st.State), nil
}

// Start 启动服务并等它跑起来。
func Start() error {
	s, done, err := openService(windows.SERVICE_START | windows.SERVICE_QUERY_STATUS)
	if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
		return ErrAccessDenied
	}
	if err != nil {
		return fmt.Errorf("打开服务失败：%w", err)
	}
	defer done()
	if err := s.Start(); err != nil && !errors.Is(err, windows.ERROR_SERVICE_ALREADY_RUNNING) {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return ErrAccessDenied
		}
		return fmt.Errorf("启动服务失败：%w", err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		st, err := s.Query()
		if err != nil {
			return err
		}
		switch st.State {
		case svc.Running:
			return nil
		case svc.Stopped:
			code := st.Win32ExitCode
			if st.ServiceSpecificExitCode != 0 {
				code = st.ServiceSpecificExitCode
			}
			return fmt.Errorf("服务启动后马上停了（退出码 %d），请看日志", code)
		}
		if time.Now().After(deadline) {
			return errors.New("等了 30 秒服务还没启动好")
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// Stop 停止服务并等它停下（没在运行时直接返回）。
func Stop() error {
	s, done, err := openService(windows.SERVICE_STOP | windows.SERVICE_QUERY_STATUS)
	if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
		return ErrAccessDenied
	}
	if err != nil {
		return fmt.Errorf("打开服务失败：%w", err)
	}
	defer done()
	if _, err := s.Control(svc.Stop); err != nil &&
		!errors.Is(err, windows.ERROR_SERVICE_NOT_ACTIVE) && !errors.Is(err, windows.ERROR_SERVICE_CANNOT_ACCEPT_CTRL) {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return ErrAccessDenied
		}
		return fmt.Errorf("停止服务失败：%w", err)
	}
	return waitState(queryState, StateStopped, 60*time.Second)
}

// environmentKey 系统环境变量在注册表里的位置。
const environmentKey = `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`

// LinkCommand 把安装目录加进系统 PATH（新开的终端里就能直接用 momoko）。
func LinkCommand(dir, _ string) (string, error) {
	err := editSystemPath(func(parts []string) []string {
		for _, p := range parts {
			if SamePath(strings.TrimRight(p, `\`), strings.TrimRight(dir, `\`)) {
				return nil
			}
		}
		return append(parts, dir)
	})
	if err != nil {
		return "", fmt.Errorf("加入系统 PATH 失败：%w", err)
	}
	return "已把 " + dir + " 加进系统 PATH，新开的终端里可以直接用 momoko 命令", nil
}

// UnlinkCommand 从系统 PATH 里去掉安装目录。
func UnlinkCommand(dir, _ string) error {
	return editSystemPath(func(parts []string) []string {
		out := parts[:0:0]
		changed := false
		for _, p := range parts {
			if SamePath(strings.TrimRight(p, `\`), strings.TrimRight(dir, `\`)) {
				changed = true
				continue
			}
			out = append(out, p)
		}
		if !changed {
			return nil
		}
		return out
	})
}

// editSystemPath 改系统 PATH：edit 返回 nil 表示不用改。改完通知其它程序环境变量变了。
func editSystemPath(edit func([]string) []string) error {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, environmentKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	cur, _, err := k.GetStringValue("Path")
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	var parts []string
	for _, p := range strings.Split(cur, ";") {
		if strings.TrimSpace(p) != "" {
			parts = append(parts, p)
		}
	}
	next := edit(parts)
	if next == nil {
		return nil
	}
	if err := k.SetExpandStringValue("Path", strings.Join(next, ";")); err != nil {
		return err
	}
	broadcastEnvChange()
	return nil
}

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	procSendMessageTimeoutW = user32.NewProc("SendMessageTimeoutW")
	shell32                 = windows.NewLazySystemDLL("shell32.dll")
	procShellExecuteExW     = shell32.NewProc("ShellExecuteExW")
	kernel32                = windows.NewLazySystemDLL("kernel32.dll")
	procGetConsoleProcList  = kernel32.NewProc("GetConsoleProcessList")
)

// broadcastEnvChange 通知资源管理器等程序重新读环境变量（之后新开的终端才看得到新的 PATH）。
func broadcastEnvChange() {
	const (
		hwndBroadcast   = 0xffff
		wmSettingChange = 0x001A
		smtoAbortIfHung = 0x0002
	)
	env, _ := windows.UTF16PtrFromString("Environment")
	var result uintptr
	_, _, _ = procSendMessageTimeoutW.Call(hwndBroadcast, wmSettingChange, 0, uintptr(unsafe.Pointer(env)),
		smtoAbortIfHung, 5000, uintptr(unsafe.Pointer(&result)))
}

// firewallRule 防火墙规则名。
const firewallRule = "Momoko"

// AllowFirewall 在 Windows 防火墙里放行 momoko.exe 的入站连接。服务在后台运行，系统不会弹「是否允许访问网络」，
// 不加规则别的机器就连不上面板。
func AllowFirewall(exe string, _ []int) (string, error) {
	RemoveFirewall(exe)
	out, err := exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
		"name="+firewallRule, "dir=in", "action=allow", "enable=yes", "profile=any", "program="+exe).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("添加防火墙规则失败（%v）：%s", err, strings.TrimSpace(string(out)))
	}
	return "已在 Windows 防火墙里放行 " + exe, nil
}

// RemoveFirewall 删掉安装时加的防火墙规则。
func RemoveFirewall(exe string) {
	_ = exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", "name="+firewallRule, "program="+exe).Run()
}

// Run 作为 Windows 服务运行 serve：服务管理器要求停止（或关机）时关闭 stop，serve 返回后服务停止。
// serve 返回非 0（在线更新换好了程序）时服务以失败结束，由服务管理器按恢复设置重新启动。
func Run(serve func(stop <-chan struct{}) int) int {
	h := &handler{serve: serve}
	if err := svc.Run(Name, h); err != nil {
		fmt.Fprintln(os.Stderr, "以 Windows 服务运行失败：", err)
		return 1
	}
	return h.code
}

type handler struct {
	serve func(stop <-chan struct{}) int
	code  int
}

func (h *handler) Execute(_ []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	s <- svc.Status{State: svc.StartPending}
	stop := make(chan struct{})
	done := make(chan int, 1)
	go func() { done <- h.serve(stop) }()
	s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	stopping := false
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				s <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				if !stopping {
					stopping = true
					s <- svc.Status{State: svc.StopPending, WaitHint: 30000}
					close(stop)
				}
			}
		case code := <-done:
			h.code = code
			if code != 0 {
				return true, uint32(code)
			}
			return false, 0
		}
	}
}

// PrepareServiceProcess 服务没有控制台：把输出（包括崩溃信息）写到 <安装目录>\logs\momoko.log。
func PrepareServiceProcess(dir string) {
	path := LogFile(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	rotateLog(path)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	os.Stdout, os.Stderr = f, f
	stdlog.SetOutput(f)
	_ = windows.SetStdHandle(windows.STD_OUTPUT_HANDLE, windows.Handle(f.Fd()))
	_ = windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(f.Fd()))
	_ = debug.SetCrashOutput(f, debug.CrashOptions{})
}

// LogHint 去哪看服务的日志。
func LogHint(dir string) string { return LogFile(dir) }

// shellExecuteInfo 是 SHELLEXECUTEINFOW。
type shellExecuteInfo struct {
	cbSize         uint32
	fMask          uint32
	hwnd           windows.Handle
	lpVerb         *uint16
	lpFile         *uint16
	lpParameters   *uint16
	lpDirectory    *uint16
	nShow          int32
	hInstApp       windows.Handle
	lpIDList       uintptr
	lpClass        *uint16
	hkeyClass      windows.Handle
	dwHotKey       uint32
	hIconOrMonitor windows.Handle
	hProcess       windows.Handle
}

// ErrElevationCanceled 没同意 UAC 提权。
var ErrElevationCanceled = errors.New("没有同意管理员权限，已取消")

// Elevate 以管理员身份（弹一次 UAC）在后台运行 exe args，把它的输出原样转到当前终端，返回它的退出码。
// relayFlag 是让子进程把输出写进文件的参数名（如 -relay），子进程窗口是隐藏的，不能有交互。
func Elevate(exe string, args []string, relayFlag string, out io.Writer) (int, error) {
	f, err := os.CreateTemp("", "momoko-elevated-*.log")
	if err != nil {
		return 1, err
	}
	relay := f.Name()
	_ = f.Close()
	defer os.Remove(relay)

	all := append(append([]string{}, args...), relayFlag, relay)
	quoted := make([]string, len(all))
	for i, a := range all {
		quoted[i] = syscall.EscapeArg(a)
	}
	verb, _ := windows.UTF16PtrFromString("runas")
	file, _ := windows.UTF16PtrFromString(exe)
	params, _ := windows.UTF16PtrFromString(strings.Join(quoted, " "))
	wd, _ := os.Getwd()
	cwd, _ := windows.UTF16PtrFromString(wd)
	const (
		seeMaskNoCloseProcess = 0x00000040
		seeMaskNoAsync        = 0x00000100
		seeMaskFlagNoUI       = 0x00000400
		swHide                = 0
	)
	info := shellExecuteInfo{
		fMask:        seeMaskNoCloseProcess | seeMaskNoAsync | seeMaskFlagNoUI,
		lpVerb:       verb,
		lpFile:       file,
		lpParameters: params,
		lpDirectory:  cwd,
		nShow:        swHide,
	}
	info.cbSize = uint32(unsafe.Sizeof(info))
	if r, _, e := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&info))); r == 0 {
		if errors.Is(e, windows.ERROR_CANCELLED) {
			return 1, ErrElevationCanceled
		}
		return 1, fmt.Errorf("以管理员身份运行失败：%w", e)
	}
	if info.hProcess == 0 {
		return 1, errors.New("以管理员身份运行失败：拿不到进程")
	}
	defer windows.CloseHandle(info.hProcess)

	rf, err := os.Open(relay)
	if err != nil {
		return 1, err
	}
	defer rf.Close()
	for {
		ev, _ := windows.WaitForSingleObject(info.hProcess, 150)
		_, _ = io.Copy(out, rf)
		if ev != uint32(windows.WAIT_TIMEOUT) {
			break
		}
	}
	_, _ = io.Copy(out, rf)
	var code uint32
	if err := windows.GetExitCodeProcess(info.hProcess, &code); err != nil {
		return 1, err
	}
	return int(code), nil
}

// OwnConsole 控制台窗口是不是为这个进程新开的（双击运行），是的话输出完要等用户按回车再关。
func OwnConsole() bool {
	var pids [4]uint32
	n, _, _ := procGetConsoleProcList.Call(uintptr(unsafe.Pointer(&pids[0])), uintptr(len(pids)))
	return n == 1
}
