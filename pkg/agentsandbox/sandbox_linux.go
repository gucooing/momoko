//go:build linux

package agentsandbox

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// ctlFD 1 号进程里和 Momoko 通话的套接字（ExtraFiles 的第一个）。
const ctlFD = 3

// stepTimeout 等 1 号进程每一步的时间：它要先把 Momoko 这个程序加载起来，网卡要等 sing-box 建好。
const stepTimeout = 30 * time.Second

// zoneinfo 时区数据的目录。
const zoneinfo = "/usr/share/zoneinfo"

// cloneFlags 沙箱的命名空间：网络、挂载、进程号。
const cloneFlags = unix.CLONE_NEWNET | unix.CLONE_NEWNS | unix.CLONE_NEWPID

// hiddenDirs、hiddenFiles 沙箱里盖住的：本机的解析服务（不能借它们在外面查 DNS），和能替进程联网的服务的套接字
// （不能让 Docker 拉镜像、让 systemd 起个外面的进程）。不存在的跳过；/var/run 一般是 /run 的链接，按真实路径去重。
var (
	hiddenDirs = []string{
		"/run/systemd/resolve", "/run/nscd", "/run/avahi-daemon",
		"/run/dbus", "/run/containerd", "/run/podman", "/run/docker",
		"/var/run/systemd/resolve", "/var/run/nscd", "/var/run/avahi-daemon",
		"/var/run/dbus", "/var/run/containerd", "/var/run/podman", "/var/run/docker",
	}
	hiddenFiles = []string{
		"/run/docker.sock", "/run/snapd.socket", "/run/snapd-snap.socket", "/run/systemd/private",
		"/var/run/docker.sock", "/var/run/snapd.socket", "/var/run/snapd-snap.socket", "/var/run/systemd/private",
	}
)

// proxyVars 主机环境里可能有的代理变量：沙箱里一个都不留。
var proxyVars = []string{
	"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "FTP_PROXY", "NO_PROXY",
	"http_proxy", "https_proxy", "all_proxy", "ftp_proxy", "no_proxy",
	"npm_config_proxy", "npm_config_https_proxy", "npm_config_noproxy", "NODE_USE_ENV_PROXY",
}

// droppedCaps 沙箱里的命令拿不到的权限：CAP_SYS_ADMIN（setns 回主机网络、改挂载）、CAP_NET_ADMIN（改网卡和路由）、
// CAP_SYS_PTRACE（附到外面的进程上替它联网）、CAP_SYS_MODULE、CAP_BPF、CAP_PERFMON。
var droppedCaps = []uintptr{unix.CAP_SYS_ADMIN, unix.CAP_NET_ADMIN, unix.CAP_SYS_PTRACE, unix.CAP_SYS_MODULE, unix.CAP_BPF, unix.CAP_PERFMON}

// Supported 这个系统能不能建沙箱（只有 Linux）。
func Supported() bool { return true }

var (
	checkOnce sync.Once
	checkErr  error
)

// Check 现在能不能建沙箱：要 root、要有 /dev/net/tun，在容器里还要给够权限。真的起一次 1 号进程试试，结果记下来。
// 第一次调用时顺便清掉 Momoko 上次没来得及收的网络命名空间。
func Check() error {
	checkOnce.Do(func() {
		sweepNetns()
		checkErr = probe()
	})
	return checkErr
}

func probe() error {
	if os.Geteuid() != 0 {
		return errors.New("Momoko 没有以 root 身份运行，建不了沙箱：用 momoko install 装成系统服务，或者用 root 运行")
	}
	if _, err := os.Stat("/dev/net/tun"); err != nil {
		return errors.New("服务器上没有 /dev/net/tun，sing-box 建不了网卡")
	}
	ctx, cancel := context.WithTimeout(context.Background(), stepTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, selfExe, gateArg, probeArg)
	cmd.SysProcAttr = &unix.SysProcAttr{Cloneflags: cloneFlags}
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	reason := strings.TrimSpace(string(out))
	if reason == "" {
		reason = err.Error()
	}
	return errors.New("建不了沙箱：" + reason)
}

// Start 在沙箱里启动 cmd（Path、Args、Dir、Env、标准输入输出、SysProcAttr 都已设好；Env 为空时用当前进程的环境）。
// 成功时命令已经在沙箱里跑起来了；返回的 wait 等它结束并收掉 sing-box，不要再调 cmd.Wait。失败时沙箱里的进程都已结束。
func Start(cmd *exec.Cmd, cfg Config) (func() error, error) {
	if err := Check(); err != nil {
		return nil, err
	}
	if info, err := os.Stat(cfg.SingBox); err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("沙箱要用的 sing-box 还没下载")
	}
	if len(cmd.ExtraFiles) > 0 {
		return nil, errors.New("沙箱里的命令不能再带别的文件")
	}
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_SEQPACKET|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("建不了沙箱：%v", err)
	}
	// 这一端交给网络轮询器，等 1 号进程时才能设超时
	if err := unix.SetNonblock(fds[0], true); err != nil {
		_ = unix.Close(fds[0])
		_ = unix.Close(fds[1])
		return nil, fmt.Errorf("建不了沙箱：%v", err)
	}
	parent := os.NewFile(uintptr(fds[0]), "agent-sandbox")
	child := os.NewFile(uintptr(fds[1]), "agent-sandbox-gate")
	defer child.Close()
	ctlConn, err := net.FileConn(parent)
	_ = parent.Close()
	if err != nil {
		_ = child.Close()
		return nil, fmt.Errorf("建不了沙箱：%v", err)
	}
	ctl := ctlConn.(*net.UnixConn)

	base := cmd.Env
	if base == nil {
		base = os.Environ()
	}
	cmd.Env = environ(base, cfg)
	argv := cmd.Args
	if len(argv) == 0 {
		argv = []string{cmd.Path}
	}
	cmd.Args = append([]string{"momoko-agent-sandbox", gateArg, cmd.Path}, argv...)
	cmd.Path = selfExe
	cmd.ExtraFiles = []*os.File{child}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &unix.SysProcAttr{}
	}
	cmd.SysProcAttr.Cloneflags |= cloneFlags
	if err := cmd.Start(); err != nil {
		_ = ctl.Close()
		return nil, err
	}

	var sb *singBox
	netns := ""
	fail := func(err error) (func() error, error) {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_ = ctl.Close()
		if sb != nil {
			sb.stop()
		}
		if netns != "" {
			unbindNetns(netns)
		}
		return nil, fmt.Errorf("建不了沙箱：%w", err)
	}
	// 1 号进程准备好挂载、时区、DNS 和回环口
	if err := expect(ctl, 'R'); err != nil {
		return fail(err)
	}
	// 在它的网络里起 sing-box：网卡、路由、DNS 都由 sing-box 在里面建好
	if netns, err = bindNetns(cmd.Process.Pid); err != nil {
		return fail(err)
	}
	if sb, err = startSingBox(cfg, netns); err != nil {
		return fail(err)
	}
	// 放行：1 号进程确认网卡在了，去掉权限，起真正的命令
	if _, err := ctl.Write([]byte{'G'}); err != nil {
		return fail(err)
	}
	if err := expect(ctl, 'S'); err != nil {
		return fail(err)
	}
	_ = ctl.Close()
	// sing-box 意外退出：网卡已经跟着没了，沙箱也不留着
	go func() {
		<-sb.done
		_ = cmd.Process.Kill()
	}()
	wait := func() error {
		err := cmd.Wait()
		sb.stop()
		unbindNetns(netns)
		return err
	}
	return wait, nil
}

// expect 等 1 号进程说 want；它说的是 E 时后面是做不到的原因。
func expect(ctl *net.UnixConn, want byte) error {
	_ = ctl.SetDeadline(time.Now().Add(stepTimeout))
	defer ctl.SetDeadline(time.Time{})
	buf := make([]byte, 2048)
	n, err := ctl.Read(buf)
	switch {
	case errors.Is(err, os.ErrDeadlineExceeded):
		return errors.New("沙箱一直没有准备好")
	case err != nil || n == 0:
		return errors.New("沙箱没准备好就退出了")
	case buf[0] == 'E':
		return errors.New(string(buf[1:n]))
	case buf[0] != want:
		return errors.New("沙箱的回话不对")
	}
	return nil
}

// environ 沙箱里命令的环境：base 去掉代理变量（一个都不留），设了语言就去掉 LC_*，再加上时区和语言。
func environ(base []string, cfg Config) []string {
	set := map[string]string{}
	if cfg.Proxy.Timezone != "" {
		set["TZ"] = cfg.Proxy.Timezone
		set[timezoneEnv] = cfg.Proxy.Timezone
	}
	if cfg.Proxy.Locale != "" {
		set["LANG"] = cfg.Proxy.Locale
		set["LANGUAGE"] = language(cfg.Proxy.Locale)
	}
	drop := map[string]bool{timezoneEnv: true}
	for _, k := range proxyVars {
		drop[k] = true
	}
	out := make([]string, 0, len(base)+len(set))
	for _, kv := range base {
		k, _, _ := strings.Cut(kv, "=")
		if _, replaced := set[k]; replaced || drop[k] {
			continue
		}
		if cfg.Proxy.Locale != "" && strings.HasPrefix(k, "LC_") {
			continue
		}
		out = append(out, kv)
	}
	for k, v := range set {
		out = append(out, k+"="+v)
	}
	return out
}

// language LANGUAGE 的值：en_US.UTF-8 → en_US:en。
func language(locale string) string {
	tag, _, _ := strings.Cut(locale, ".")
	short, _, _ := strings.Cut(tag, "_")
	if short == tag {
		return tag
	}
	return tag + ":" + short
}

// ———————————————————————————— 网络命名空间的名字 ————————————————————————————

// sing-box 按名字找网络命名空间（/run/netns/<名字>）：把 1 号进程的网络挂到这个名字上，沙箱结束后摘掉。
const (
	netnsDir    = "/run/netns"
	netnsPrefix = "momoko-sandbox-"
)

func bindNetns(pid int) (string, error) {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	name := netnsPrefix + hex.EncodeToString(b[:])
	if err := os.MkdirAll(netnsDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(netnsDir, name)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDONLY, 0o444)
	if err != nil {
		return "", err
	}
	_ = f.Close()
	if err := unix.Mount(fmt.Sprintf("/proc/%d/ns/net", pid), path, "", unix.MS_BIND, ""); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("挂不上沙箱的网络：%v", err)
	}
	return name, nil
}

func unbindNetns(name string) {
	path := filepath.Join(netnsDir, name)
	_ = unix.Unmount(path, unix.MNT_DETACH)
	_ = os.Remove(path)
}

// sweepNetns 摘掉 Momoko 上次没来得及收的（异常退出时留下的）。
func sweepNetns() {
	entries, err := os.ReadDir(netnsDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), netnsPrefix) {
			unbindNetns(e.Name())
		}
	}
}

// ———————————————————————————— 1 号进程 ————————————————————————————

// RunGate 放在 main 的最前面：这次是被当成沙箱的 1 号进程启动的，就做好沙箱、起命令、陪它到结束；
// 是在沙箱里替 Momoko 发请求的，就发完退出。都不会返回；不是这两种就什么都不做。
func RunGate() {
	if len(os.Args) >= 2 && os.Args[1] == fetchArg {
		os.Exit(runFetch())
	}
	if len(os.Args) < 3 || os.Args[1] != gateArg {
		return
	}
	// 权限是按线程算的：设置、去掉权限、起命令都在这一个线程上
	runtime.LockOSThread()
	if os.Args[2] == probeArg {
		err := enter("")
		if err == nil {
			err = dropCaps()
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(125)
		}
		os.Exit(0)
	}
	if len(os.Args) < 4 {
		os.Exit(125)
	}
	os.Exit(gate(os.Args[2], os.Args[3:]))
}

// gate 做好沙箱，等 sing-box 建好网卡，起 target 并陪它到结束，返回它的退出码。
func gate(target string, argv []string) int {
	fail := func(err error) int {
		_, _ = unix.Write(ctlFD, []byte("E"+err.Error()))
		return 125
	}
	if err := enter(os.Getenv(timezoneEnv)); err != nil {
		return fail(err)
	}
	if _, err := unix.Write(ctlFD, []byte{'R'}); err != nil {
		return 125
	}
	buf := make([]byte, 1)
	if n, err := unix.Read(ctlFD, buf); err != nil || n != 1 || buf[0] != 'G' {
		return 125
	}
	if err := waitTun(); err != nil {
		return fail(err)
	}
	if err := dropCaps(); err != nil {
		return fail(err)
	}
	env := make([]string, 0, len(os.Environ()))
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, timezoneEnv+"=") {
			env = append(env, kv)
		}
	}
	signals := make(chan os.Signal, 16)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
	pid, err := syscall.ForkExec(target, argv, &syscall.ProcAttr{Env: env, Files: []uintptr{0, 1, 2}})
	if err != nil {
		return fail(fmt.Errorf("启动 %s 失败：%v", target, err))
	}
	if _, err := unix.Write(ctlFD, []byte{'S'}); err != nil {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		return 125
	}
	_ = unix.Close(ctlFD)
	go func() {
		for s := range signals {
			// 终端的 Ctrl+C、Ctrl+\ 已经直接发给前台进程组里的命令了
			if s == syscall.SIGINT || s == syscall.SIGQUIT {
				continue
			}
			_ = syscall.Kill(pid, s.(syscall.Signal))
		}
	}()
	// 1 号进程要给命名空间里所有的孤儿收尸；命令退出就跟着退出，剩下的进程由内核结束
	for {
		var ws syscall.WaitStatus
		wpid, err := syscall.Wait4(-1, &ws, 0, nil)
		if errors.Is(err, syscall.EINTR) {
			continue
		}
		if err != nil {
			return 125
		}
		if wpid != pid {
			continue
		}
		switch {
		case ws.Exited():
			return ws.ExitStatus()
		case ws.Signaled():
			return 128 + int(ws.Signal())
		default:
			return 125
		}
	}
}

// enter 在新的命名空间里做好沙箱：挂载只从主机传进来不传出去，挂上只看得到沙箱进程的 /proc，盖住 hiddenDirs、hiddenFiles，
// 换时区（tz 为空不换），DNS 指向 sing-box 网卡上的地址，打开回环口。
func enter(tz string) error {
	if err := unix.Mount("", "/", "", unix.MS_REC|unix.MS_SLAVE, ""); err != nil {
		return fmt.Errorf("隔离不了挂载：%v", err)
	}
	if err := unix.Mount("proc", "/proc", "proc", unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC, ""); err != nil {
		return fmt.Errorf("挂不上沙箱的 /proc：%v", err)
	}
	if err := hide(); err != nil {
		return err
	}
	if tz != "" {
		if err := setTimezone(tz); err != nil {
			return err
		}
	}
	if err := setResolver(); err != nil {
		return err
	}
	if err := loopbackUp(); err != nil {
		return fmt.Errorf("打不开回环网卡：%v", err)
	}
	return nil
}

func hide() error {
	seen := map[string]bool{}
	for _, list := range [][]string{hiddenDirs, hiddenFiles} {
		for _, p := range list {
			real, err := filepath.EvalSymlinks(p)
			if err != nil || seen[real] {
				continue
			}
			seen[real] = true
			info, err := os.Stat(real)
			if err != nil {
				continue
			}
			if info.IsDir() {
				err = unix.Mount("tmpfs", real, "tmpfs", unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC, "size=64k,mode=755")
			} else {
				err = unix.Mount("/dev/null", real, "", unix.MS_BIND, "")
			}
			if err != nil {
				return fmt.Errorf("盖不住 %s：%v", real, err)
			}
		}
	}
	return nil
}

// isHidden path 在 hide 盖上的空目录里（沙箱自己的，写进去不碰主机）。
func isHidden(path string) bool {
	for _, d := range hiddenDirs {
		if real, err := filepath.EvalSymlinks(d); err == nil && strings.HasPrefix(path, real+"/") {
			return true
		}
	}
	return false
}

// setTimezone /etc/localtime 换成 tz 的时区数据，/etc/timezone 换成 tz。
func setTimezone(tz string) error {
	zone := filepath.Join(zoneinfo, tz)
	if info, err := os.Stat(zone); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("服务器上没有时区 %s 的数据（装上 tzdata）", tz)
	}
	if real, err := filepath.EvalSymlinks("/etc/localtime"); err == nil {
		if err := unix.Mount(zone, real, "", unix.MS_BIND, ""); err != nil {
			return fmt.Errorf("换不了时区：%v", err)
		}
	}
	if real, err := filepath.EvalSymlinks("/etc/timezone"); err == nil {
		if err := bindText(real, tz+"\n"); err != nil {
			return fmt.Errorf("换不了时区：%v", err)
		}
	}
	return nil
}

// setResolver /etc/resolv.conf 指向 sing-box 网卡上的 DNS。它是链接、指向的文件在盖住的目录里时（systemd-resolved 的），
// 直接在那个空目录里写一份。
func setResolver() error {
	content := "nameserver " + tunDNS + "\n"
	const conf = "/etc/resolv.conf"
	if real, err := filepath.EvalSymlinks(conf); err == nil {
		return bindText(real, content)
	}
	link, err := os.Readlink(conf)
	if err != nil {
		return fmt.Errorf("服务器上没有 %s", conf)
	}
	if !filepath.IsAbs(link) {
		link = filepath.Join(filepath.Dir(conf), link)
	}
	if !isHidden(link) {
		return fmt.Errorf("%s 指向的 %s 不存在", conf, link)
	}
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return err
	}
	return os.WriteFile(link, []byte(content), 0o644)
}

// bindText 把一段文字挂到 target 上：先写进内存里的临时文件，挂上后删掉它的名字（挂载留着它的内容）。
func bindText(target, text string) error {
	dir := "/dev/shm"
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		dir = os.TempDir()
	}
	f, err := os.CreateTemp(dir, "momoko-sandbox-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err := f.WriteString(text); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return unix.Mount(name, target, "", unix.MS_BIND, "")
}

// loopbackUp 新的网络命名空间里回环口默认是关着的。
func loopbackUp() error {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	ifr, err := unix.NewIfreq("lo")
	if err != nil {
		return err
	}
	if err := unix.IoctlIfreq(fd, unix.SIOCGIFFLAGS, ifr); err != nil {
		return err
	}
	ifr.SetUint16(ifr.Uint16() | unix.IFF_UP)
	return unix.IoctlIfreq(fd, unix.SIOCSIFFLAGS, ifr)
}

// waitTun sing-box 说起好了之后，网卡应该已经在沙箱里了；再确认一次它在、开着。
func waitTun() error {
	deadline := time.Now().Add(5 * time.Second)
	for {
		if ifc, err := net.InterfaceByName(tunName); err == nil && ifc.Flags&net.FlagUp != 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("sing-box 没有在沙箱里建好网卡")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// dropCaps 从边界集里去掉 droppedCaps：之后起的命令和它的子孙都拿不回这些权限（root 也一样）。
// 老内核不认识的权限（CAP_BPF、CAP_PERFMON）跳过。
func dropCaps() error {
	for _, c := range droppedCaps {
		if err := unix.Prctl(unix.PR_CAPBSET_DROP, c, 0, 0, 0); err != nil && !errors.Is(err, unix.EINVAL) {
			return fmt.Errorf("去不掉权限 %d：%v", c, err)
		}
	}
	_ = unix.Prctl(unix.PR_CAP_AMBIENT, unix.PR_CAP_AMBIENT_CLEAR_ALL, 0, 0, 0)
	return nil
}
