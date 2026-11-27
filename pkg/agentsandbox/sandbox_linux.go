//go:build linux

package agentsandbox

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"momoko/pkg/agentproxy"
)

// ctlFD 看门进程里和 Momoko 通话的套接字（ExtraFiles 的第一个）。
const ctlFD = 3

// attachTimeout 等看门进程准备好的时间：它要先把 Momoko 这个程序加载起来。
const attachTimeout = 30 * time.Second

// selfExe 看门进程就是 Momoko 自己：/proc/self/exe 在 fork 之后指向的还是这个程序，程序文件被换掉了也能起。
const selfExe = "/proc/self/exe"

// hiddenPaths 盖住的本机解析服务：Agent 的网络里查不了 DNS，也不能借它们在外面查。
// 不存在的跳过；/var/run 一般是 /run 的链接，按真实路径去重。
var hiddenPaths = []string{
	"/run/systemd/resolve",
	"/var/run/systemd/resolve",
	"/run/nscd",
	"/var/run/nscd",
	"/run/avahi-daemon",
	"/var/run/avahi-daemon",
}

// droppedCaps 进了隔离网络就不再给的权限：CAP_SYS_ADMIN（setns 回主机网络、改挂载）、CAP_NET_ADMIN（改网卡）、
// CAP_SYS_PTRACE（附到外面的进程上替它联网）、CAP_SYS_MODULE、CAP_BPF、CAP_PERFMON。
var droppedCaps = []uintptr{unix.CAP_SYS_ADMIN, unix.CAP_NET_ADMIN, unix.CAP_SYS_PTRACE, unix.CAP_SYS_MODULE, unix.CAP_BPF, unix.CAP_PERFMON}

// Supported 这个系统能不能把 Agent 关进只走代理的网络（只有 Linux）。
func Supported() bool { return true }

var (
	checkOnce sync.Once
	checkErr  error
)

// Check 现在能不能建隔离网络：要 root（或者至少能建网络和挂载命名空间），在容器里还要给够权限。
// 真的起一次看门进程试试，结果记下来。
func Check() error {
	checkOnce.Do(func() { checkErr = probe() })
	return checkErr
}

func probe() error {
	if os.Geteuid() != 0 {
		return errors.New("Momoko 没有以 root 身份运行，建不了只走代理的隔离网络：用 momoko install 装成系统服务，或者用 root 运行")
	}
	ctx, cancel := context.WithTimeout(context.Background(), attachTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, selfExe, gateArg, probeArg)
	cmd.SysProcAttr = &unix.SysProcAttr{Cloneflags: unix.CLONE_NEWNET | unix.CLONE_NEWNS}
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	reason := strings.TrimSpace(string(out))
	if reason == "" {
		reason = err.Error()
	}
	return errors.New("建不了只走代理的隔离网络：" + reason)
}

// Sandbox 一个关在隔离网络里的进程：Wrap 改好启动命令，进程起来后 Attach，进程退出后 Close。
type Sandbox struct {
	parent *os.File
	child  *os.File

	mu  sync.Mutex
	ln  net.Listener
	ctl net.Conn
}

// Wrap 把 cmd 改成在隔离网络里启动：先起看门进程，Attach 之后它才换成 cmd 原来的命令。
// cmd 的 Path 要是可执行文件的路径（相对路径相对 cmd.Dir）；环境变量里应已带上 agentproxy.Env(ProxyAddr)。
func Wrap(cmd *exec.Cmd) (*Sandbox, error) {
	if err := Check(); err != nil {
		return nil, err
	}
	if len(cmd.ExtraFiles) > 0 {
		return nil, errors.New("隔离网络里的进程不能再带别的文件")
	}
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_SEQPACKET|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("建不了只走代理的隔离网络：%v", err)
	}
	// 这一端交给网络轮询器，等看门进程时才能设超时
	if err := unix.SetNonblock(fds[0], true); err != nil {
		_ = unix.Close(fds[0])
		_ = unix.Close(fds[1])
		return nil, fmt.Errorf("建不了只走代理的隔离网络：%v", err)
	}
	s := &Sandbox{
		parent: os.NewFile(uintptr(fds[0]), "agent-sandbox"),
		child:  os.NewFile(uintptr(fds[1]), "agent-sandbox-gate"),
	}
	argv := cmd.Args
	if len(argv) == 0 {
		argv = []string{cmd.Path}
	}
	cmd.Args = append([]string{"momoko-agent-sandbox", gateArg, cmd.Path}, argv...)
	cmd.Path = selfExe
	cmd.ExtraFiles = []*os.File{s.child}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &unix.SysProcAttr{}
	}
	cmd.SysProcAttr.Cloneflags |= unix.CLONE_NEWNET | unix.CLONE_NEWNS
	return s, nil
}

// Attach 进程起来之后调：等看门进程交来代理入口，在上面经 p 转发，再放它换成真正的命令。
// 失败时进程还停在看门这一步，调用方要把它结束掉。
func (s *Sandbox) Attach(p agentproxy.Proxy) error {
	s.closeChild()
	conn, err := net.FileConn(s.parent)
	_ = s.parent.Close()
	if err != nil {
		return fmt.Errorf("建不了只走代理的隔离网络：%v", err)
	}
	s.mu.Lock()
	s.ctl = conn
	s.mu.Unlock()
	_ = conn.SetDeadline(time.Now().Add(attachTimeout))

	msg, fd, err := recvListener(conn)
	if err != nil {
		return fmt.Errorf("建不了只走代理的隔离网络：%v", err)
	}
	if msg != "" {
		return errors.New("建不了只走代理的隔离网络：" + msg)
	}
	f := os.NewFile(uintptr(fd), "agent-proxy")
	ln, err := net.FileListener(f)
	_ = f.Close()
	if err != nil {
		return fmt.Errorf("建不了只走代理的隔离网络：%v", err)
	}
	s.mu.Lock()
	s.ln = ln
	s.mu.Unlock()
	go agentproxy.Serve(ln, p)
	if _, err := conn.Write([]byte{'G'}); err != nil {
		return fmt.Errorf("建不了只走代理的隔离网络：%v", err)
	}
	return nil
}

// recvListener 收看门进程的回话：成功时是代理入口的监听套接字，失败时是原因。
func recvListener(conn net.Conn) (msg string, fd int, err error) {
	uc, ok := conn.(*net.UnixConn)
	if !ok {
		return "", -1, errors.New("和看门进程的连接不对")
	}
	rc, err := uc.SyscallConn()
	if err != nil {
		return "", -1, err
	}
	buf := make([]byte, 1024)
	oob := make([]byte, unix.CmsgSpace(4))
	var n, oobn int
	var rerr error
	err = rc.Read(func(raw uintptr) bool {
		// 收到的描述符直接带上 CLOEXEC：同时起的别的进程不会继承它
		n, oobn, _, _, rerr = unix.Recvmsg(int(raw), buf, oob, unix.MSG_CMSG_CLOEXEC)
		return rerr != unix.EAGAIN
	})
	if err == nil {
		err = rerr
	}
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			return "", -1, errors.New("看门进程一直没有准备好")
		}
		return "", -1, err
	}
	if n == 0 {
		return "", -1, errors.New("看门进程没准备好就退出了")
	}
	if buf[0] == 'E' {
		return string(buf[1:n]), -1, nil
	}
	if buf[0] != 'L' {
		return "", -1, errors.New("看门进程的回话不对")
	}
	cmsgs, err := unix.ParseSocketControlMessage(oob[:oobn])
	if err != nil || len(cmsgs) != 1 {
		return "", -1, errors.New("没有收到代理入口")
	}
	fds, err := unix.ParseUnixRights(&cmsgs[0])
	if err != nil || len(fds) != 1 {
		for _, f := range fds {
			_ = unix.Close(f)
		}
		return "", -1, errors.New("没有收到代理入口")
	}
	return "", fds[0], nil
}

func (s *Sandbox) closeChild() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.child != nil {
		_ = s.child.Close()
		s.child = nil
	}
}

// Close 进程退出后（或没起来时）调：关掉代理入口，进程网络里还开着的连接一起断掉。
func (s *Sandbox) Close() {
	s.closeChild()
	_ = s.parent.Close()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ctl != nil {
		_ = s.ctl.Close()
		s.ctl = nil
	}
	if s.ln != nil {
		_ = s.ln.Close()
		s.ln = nil
	}
}

// ———————————————————————————— 看门进程 ————————————————————————————

// RunGate 放在 main 的最前面：这次是被当成看门进程启动的，就准备好隔离网络并换成真正的命令（不会返回）；不是就什么都不做。
func RunGate() {
	if len(os.Args) < 3 || os.Args[1] != gateArg {
		return
	}
	if os.Args[2] == probeArg {
		runtime.LockOSThread()
		err := prepare()
		if err == nil {
			err = dropCaps()
		}
		if err == nil {
			var ln net.Listener
			if ln, err = net.Listen("tcp4", ProxyAddr); err == nil {
				_ = ln.Close()
			}
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

// gate 准备好隔离网络，把代理入口交给 Momoko，等它放行后换成 target。只在出错时返回。
func gate(target string, argv []string) int {
	runtime.LockOSThread()
	fail := func(err error) int {
		_ = unix.Sendmsg(ctlFD, []byte("E"+err.Error()), nil, nil, 0)
		return 125
	}
	if err := prepare(); err != nil {
		return fail(err)
	}
	if err := dropCaps(); err != nil {
		return fail(err)
	}
	ln, err := net.Listen("tcp4", ProxyAddr)
	if err != nil {
		return fail(fmt.Errorf("开不了代理入口：%v", err))
	}
	f, err := ln.(*net.TCPListener).File()
	if err != nil {
		return fail(fmt.Errorf("开不了代理入口：%v", err))
	}
	if err := unix.Sendmsg(ctlFD, []byte("L"), unix.UnixRights(int(f.Fd())), nil, 0); err != nil {
		return 125
	}
	// 入口归 Momoko 了，这边的两份关掉（套接字本身还在 Momoko 那里）
	_ = f.Close()
	_ = ln.Close()

	buf := make([]byte, 1)
	if n, err := unix.Read(ctlFD, buf); err != nil || n != 1 || buf[0] != 'G' {
		return 125
	}
	_ = unix.Close(ctlFD)
	err = unix.Exec(target, argv, os.Environ())
	fmt.Fprintf(os.Stderr, "启动 %s 失败：%v\n", target, err)
	return 127
}

// prepare 在新的网络和挂载命名空间里：主机上之后的挂载照样传进来、这里的不传出去，盖住本机解析服务，打开回环口。
func prepare() error {
	if err := unix.Mount("", "/", "", unix.MS_REC|unix.MS_SLAVE, ""); err != nil {
		return fmt.Errorf("隔离不了挂载：%v", err)
	}
	seen := map[string]bool{}
	for _, p := range hiddenPaths {
		real, err := filepath.EvalSymlinks(p)
		if err != nil || seen[real] {
			continue
		}
		seen[real] = true
		if info, err := os.Stat(real); err != nil || !info.IsDir() {
			continue
		}
		if err := unix.Mount("tmpfs", real, "tmpfs", unix.MS_RDONLY|unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC, "size=4k,mode=755"); err != nil {
			return fmt.Errorf("盖不住本机的解析服务 %s：%v", real, err)
		}
	}
	if err := loopbackUp(); err != nil {
		return fmt.Errorf("打不开回环网卡：%v", err)
	}
	return nil
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

// dropCaps 从边界集里去掉 droppedCaps：换成真正的命令后它和它的子孙都拿不回这些权限（root 也一样）。
// 老内核不认识的权限（CAP_BPF、CAP_PERFMON）跳过；别的去不掉就不启动。
func dropCaps() error {
	for _, c := range droppedCaps {
		if err := unix.Prctl(unix.PR_CAPBSET_DROP, c, 0, 0, 0); err != nil && !errors.Is(err, unix.EINVAL) {
			return fmt.Errorf("去不掉权限 %d：%v", c, err)
		}
	}
	_ = unix.Prctl(unix.PR_CAP_AMBIENT, unix.PR_CAP_AMBIENT_CLEAR_ALL, 0, 0, 0)
	return nil
}
