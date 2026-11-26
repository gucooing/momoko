package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"momoko/internal/initialize"
	"momoko/pkg/selfupdate"
	"momoko/pkg/sysservice"
	"momoko/pkg/version"
)

// 命令行子命令：把 Momoko 装成系统服务（像 1Panel 那样，只问安装目录和端口；管理员和数据库在第一次打开网页时初始化），
// 以及装好后的启动、停止、重启、查看状态、卸载。要管理员 / root 权限的命令会自己申请：
// Windows 弹一次 UAC（在隐藏的管理员进程里执行，输出转回当前终端），Linux / macOS 用 sudo 在同一个终端里重新运行。

// out 命令的输出；以管理员身份在后台执行时（-relay）写进文件，由发起的那个进程转到终端。
var out io.Writer = os.Stdout

// elevatedChild 这个进程就是提权后在后台执行的那个（带了 -relay）：还没有权限时不能再提权，免得一直弹 UAC。
var elevatedChild bool

var commands = map[string]func(args []string) int{
	"install":   cmdInstall,
	"uninstall": cmdUninstall,
	"start":     func(args []string) int { return cmdControl("start", args) },
	"stop":      func(args []string) int { return cmdControl("stop", args) },
	"restart":   func(args []string) int { return cmdControl("restart", args) },
	"status":    cmdStatus,
	"run":       cmdRun,
	"help":      func([]string) int { usage(os.Stdout); return 0 },
}

const usageText = `用法：momoko [命令] [参数]

不带命令：和普通程序一样在当前目录运行（配置在 ./configs，数据在 ./data）。

命令：
  install     安装成系统服务：复制到安装目录、开机自动启动、在后台运行，并加进系统 PATH
                -dir <目录>   安装目录（默认 %s）
                -port <端口>  后台端口（默认 %d）
                -y            不提问，没给的都用默认值
              已经装过时，用当前这个程序替换已安装的程序并重启服务（升级）
  uninstall   卸载系统服务（程序、配置和数据留在安装目录）  -y 不确认
  start       启动服务
  stop        停止服务
  restart     重启服务
  status      查看服务状态和面板地址
  run         在前台运行已安装的 Momoko（排查问题用，先 momoko stop）
  help        显示这个说明

参数：
  -conf <路径>  配置目录或文件（默认 ./configs）
  -version      只显示版本号
`

func usage(w io.Writer) {
	fmt.Fprintf(w, usageText, sysservice.DefaultDir(), initialize.DefaultHTTPPort)
}

// isCommand 第一个参数是不是子命令（不是 -flag）。
func isCommand(arg string) bool { return arg != "" && !strings.HasPrefix(arg, "-") }

func runCommand(name string, args []string) int {
	cmd, ok := commands[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "不认识的命令「%s」\n\n", name)
		usage(os.Stderr)
		return 2
	}
	return cmd(args)
}

// newFlags 子命令的参数；-relay 是内部用的（以管理员身份在后台执行时，输出写到哪个文件）。
func newFlags(name string) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet("momoko "+name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	relay := fs.String("relay", "", "")
	fs.Usage = func() { usage(os.Stderr) }
	return fs, relay
}

// useRelay 有 -relay 时把输出写进那个文件。
func useRelay(path string) func() {
	if path == "" {
		return func() {}
	}
	elevatedChild = true
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return func() {}
	}
	out = f
	return func() { _ = f.Close() }
}

func fail(err error) int {
	fmt.Fprintln(out, "错误：", err)
	return 1
}

// interactive 标准输入是不是终端（能提问）。
func interactive() bool {
	st, err := os.Stdin.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

var stdin = bufio.NewReader(os.Stdin)

// stdinEOF 输入已经读完（管道、Ctrl+Z / Ctrl+D）：不能再重新提问，答错了就直接报错。
var stdinEOF bool

// ask 提问，直接回车用默认值。
func ask(question, def string) string {
	fmt.Fprintf(out, "%s [%s]: ", question, def)
	line, err := stdin.ReadString('\n')
	if err != nil {
		stdinEOF = true
		fmt.Fprintln(out)
	}
	if line = strings.TrimSpace(line); line == "" {
		return def
	}
	return line
}

// confirm 问是 / 否。
func confirm(question string, def bool) bool {
	hint := "y/N"
	if def {
		hint = "Y/n"
	}
	fmt.Fprintf(out, "%s [%s]: ", question, hint)
	line, _ := stdin.ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "":
		return def
	case "y", "yes", "是":
		return true
	}
	return false
}

// asAdmin 没有管理员 / root 权限时，以管理员身份重新运行 momoko <name> <args>，返回它的退出码；
// 已经有权限时 elevated 为 false，由调用方接着做。
func asAdmin(name string, args []string) (code int, elevated bool) {
	if sysservice.IsAdmin() {
		return 0, false
	}
	if elevatedChild {
		return fail(errors.New("以管理员身份运行了，但还是没有管理员权限（当前用户不是管理员？）")), true
	}
	exe, err := sysservice.Executable()
	if err != nil {
		return fail(err), true
	}
	if runtime.GOOS == "windows" {
		fmt.Fprintln(out, "需要管理员权限，请在弹出的窗口里点「是」…")
	} else {
		fmt.Fprintln(out, "需要 root 权限，用 sudo 继续…")
	}
	code, err = sysservice.Elevate(exe, append([]string{name}, args...), "-relay", out)
	if err != nil {
		return fail(err), true
	}
	return code, true
}

// ———— install ————

func cmdInstall(args []string) int {
	fs, relay := newFlags("install")
	dirFlag := fs.String("dir", "", "安装目录")
	portFlag := fs.Int("port", 0, "后台端口")
	yes := fs.Bool("y", false, "不提问")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	defer useRelay(*relay)()
	if err := sysservice.Supported(); err != nil {
		return fail(err)
	}
	exe, err := sysservice.Executable()
	if err != nil {
		return fail(fmt.Errorf("找不到当前程序文件：%w", err))
	}
	prompt := !*yes && interactive()

	info, installed, err := sysservice.Lookup()
	if err != nil {
		return fail(err)
	}
	if installed {
		return upgrade(info, exe, prompt, *dirFlag, *portFlag)
	}

	// 安装目录
	dir := *dirFlag
	if dir == "" {
		dir = sysservice.DefaultDir()
		if prompt {
			if here := filepath.Dir(exe); hasData(here) && !sysservice.SamePath(here, dir) {
				fmt.Fprintf(out, "提示：%s 里有正在用的 Momoko 数据；要接着用这些数据，安装目录填 %s（就地装成服务）。\n", here, here)
			}
			for {
				d := ask1("安装目录", dir)
				if _, _, err := checkDir(d); err != nil && !stdinEOF {
					fmt.Fprintln(out, "  ", err)
					continue
				}
				dir = d
				break
			}
		}
	}
	dir, existing, err := checkDir(dir)
	if err != nil {
		return fail(err)
	}

	// 端口：目录里已有配置时默认沿用它的
	ports := initialize.Ports{HTTP: initialize.DefaultHTTPPort, GRPC: 22733, Passport: 22634, PassportEnabled: true}
	if existing {
		if p, err := initialize.ReadPorts(filepath.Join(dir, "configs")); err == nil {
			ports = p
		}
	}
	port := *portFlag
	if port == 0 {
		port = ports.HTTP
		if port == 0 {
			port = initialize.DefaultHTTPPort
		}
		if prompt {
			for {
				p, err := strconv.Atoi(ask1("后台端口", strconv.Itoa(port)))
				if err != nil {
					err = errors.New("端口要填数字")
				} else {
					err = checkPort(p, ports, true)
				}
				if err != nil {
					if stdinEOF {
						return fail(err)
					}
					fmt.Fprintln(out, "  ", err)
					continue
				}
				port = p
				break
			}
		}
	}
	if err := checkPort(port, ports, true); err != nil {
		return fail(err)
	}

	// 问完再提权：Windows 上管理员进程在后台跑，不能再提问
	if code, elevated := asAdmin("install", []string{"-dir", dir, "-port", strconv.Itoa(port), "-y"}); elevated {
		return code
	}
	return doInstall(exe, dir, port, existing)
}

// ask1 提问并去掉两边的引号（Windows 上复制的路径常带引号）。
func ask1(question, def string) string {
	return strings.Trim(ask(question, def), `"'`)
}

// hasData 目录里有没有初始化过的 Momoko 数据。
func hasData(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, initialize.DataDir, initialize.MarkerFile))
	return err == nil
}

// checkDir 检查安装目录：返回绝对路径、里面是不是已经有 Momoko 的配置（有就沿用）。
// 目录里有别的东西时拒绝，免得把文件散在 /opt、C:\ 这种目录里。
func checkDir(dir string) (string, bool, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", false, err
	}
	if err := sysservice.ValidateDir(abs); err != nil {
		return "", false, err
	}
	st, err := os.Stat(abs)
	if errors.Is(err, os.ErrNotExist) {
		return abs, false, nil
	}
	if err != nil {
		return "", false, err
	}
	if !st.IsDir() {
		return "", false, fmt.Errorf("%s 是一个文件，不是目录", abs)
	}
	if _, err := os.Stat(filepath.Join(abs, "configs", "config.yaml")); err == nil {
		return abs, true, nil
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return "", false, err
	}
	for _, e := range entries {
		name := strings.ToLower(e.Name())
		switch {
		case name == "configs", name == "data", name == "logs", name == "servers",
			strings.HasPrefix(name, "momoko"), strings.HasPrefix(name, ".momoko"):
		default:
			return "", false, fmt.Errorf("%s 里已经有别的文件了，请换一个空目录（比如 %s）", abs, filepath.Join(abs, "momoko"))
		}
	}
	return abs, false, nil
}

// checkPort 端口要合法、不和 gRPC / 通行证端口撞、现在没被占用（free 为 true 时检查）。
func checkPort(p int, ports initialize.Ports, free bool) error {
	if p < 1 || p > 65535 {
		return errors.New("端口要在 1 到 65535 之间")
	}
	if p == ports.GRPC {
		return fmt.Errorf("端口 %d 是 gRPC（多节点）用的，换一个", p)
	}
	if ports.PassportEnabled && p == ports.Passport {
		return fmt.Errorf("端口 %d 是用户通行证用的，换一个", p)
	}
	if free {
		if err := portFree(p); err != nil {
			return err
		}
	}
	return nil
}

// portFree 端口现在有没有被占用：本机连得上就是有程序在监听；再试着在 127.0.0.1 上监听一下
// （只监听本机，Windows 防火墙不会因此弹窗）。
func portFree(p int) error {
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(p))
	busy := fmt.Errorf("端口 %d 已经被占用（是不是已经有 Momoko 在运行？先把它关掉）", p)
	if c, err := net.DialTimeout("tcp", addr, 500*time.Millisecond); err == nil {
		_ = c.Close()
		return busy
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		if errors.Is(err, syscall.EACCES) || errors.Is(err, os.ErrPermission) {
			return nil // 没有 root 绑不了 1024 以下的端口，提权后再说
		}
		return busy
	}
	_ = ln.Close()
	return nil
}

func doInstall(exe, dir string, port int, existing bool) int {
	fmt.Fprintf(out, "安装到 %s …\n", dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fail(err)
	}
	target := filepath.Join(dir, sysservice.ExeName())
	if !sysservice.SamePath(exe, target) {
		if err := copyFile(exe, target); err != nil {
			return fail(fmt.Errorf("复制程序失败：%w", err))
		}
	}
	confDir := filepath.Join(dir, "configs")
	if existing {
		if p, err := initialize.ReadPorts(confDir); err != nil || p.HTTP != port {
			if err := initialize.SetHTTPPort(confDir, port); err != nil {
				return fail(fmt.Errorf("修改配置里的端口失败：%w", err))
			}
		}
		fmt.Fprintln(out, "  沿用目录里已有的配置和数据")
	} else if err := initialize.WriteDefaultConfig(confDir, port); err != nil {
		return fail(fmt.Errorf("写入配置失败：%w", err))
	}
	ports, err := initialize.ReadPorts(confDir)
	if err != nil {
		return fail(err)
	}
	for _, p := range []struct {
		port int
		name string
		on   bool
	}{{ports.GRPC, "gRPC", true}, {ports.Passport, "用户通行证", ports.PassportEnabled}} {
		if p.on && p.port > 0 && portFree(p.port) != nil {
			fmt.Fprintf(out, "  注意：%s端口 %d 已经被占用，服务可能起不来；可以改 %s 里的端口后 momoko restart\n",
				p.name, p.port, filepath.Join(confDir, "config.yaml"))
		}
	}

	if err := sysservice.Register(dir, target); err != nil {
		return fail(err)
	}
	fmt.Fprintln(out, "  已注册成系统服务，开机自动启动")
	if msg, err := sysservice.LinkCommand(dir, target); err != nil {
		fmt.Fprintln(out, "  注意：", err)
	} else {
		fmt.Fprintln(out, " ", msg)
	}
	if msg, err := sysservice.AllowFirewall(target, publicPorts(ports)); err != nil {
		fmt.Fprintln(out, "  注意：", err)
	} else if msg != "" {
		fmt.Fprintln(out, " ", msg)
	}
	if err := sysservice.Start(); err != nil {
		return fail(fmt.Errorf("启动服务失败：%w（日志：%s）", err, sysservice.LogHint(dir)))
	}
	started := sysservice.WaitPort(port, 30*time.Second)

	fmt.Fprintln(out)
	if started {
		fmt.Fprintln(out, "Momoko 已安装并启动。")
	} else {
		fmt.Fprintf(out, "Momoko 已安装，服务已启动，但 30 秒内还没打开端口 %d，请看日志：%s\n", port, sysservice.LogHint(dir))
	}
	printInfo(dir, port, !hasData(dir))
	return 0
}

// publicPorts 要在防火墙放行的端口：后台和用户通行证（gRPC 默认明文，不自动放行）。
func publicPorts(p initialize.Ports) []int {
	ports := []int{p.HTTP}
	if p.PassportEnabled && p.Passport > 0 {
		ports = append(ports, p.Passport)
	}
	return ports
}

func printInfo(dir string, port int, fresh bool) {
	fmt.Fprintf(out, "  安装目录：%s（配置在 configs，数据在 data）\n", dir)
	for i, u := range panelURLs(port) {
		label := "  面板地址："
		if i > 0 {
			label = "            "
		}
		fmt.Fprintln(out, label+u)
	}
	if fresh {
		fmt.Fprintln(out, "  第一次打开面板时完成初始化：选数据库、创建管理员。")
	}
	if hint := sysservice.LogHint(dir); hint != "" {
		fmt.Fprintf(out, "  日志：%s\n", hint)
	}
	fmt.Fprintln(out, "  管理：momoko status | start | stop | restart | uninstall")
}

// panelURLs 本机各网卡的面板地址。
func panelURLs(port int) []string {
	var urls []string
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		ipn, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipn.IP.To4()
		if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		urls = append(urls, fmt.Sprintf("http://%s:%d", ip, port))
		if len(urls) == 4 {
			break
		}
	}
	return append(urls, fmt.Sprintf("http://127.0.0.1:%d", port))
}

// copyFile 把程序复制到 dst（先写临时文件再改名，不会留下半个文件）。
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".momoko-install-*")
	if err != nil {
		return err
	}
	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	_ = os.Chmod(tmp.Name(), 0o755)
	if err := os.Rename(tmp.Name(), dst); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return nil
}

// ———— 升级：已经装过时，用当前程序替换已安装的程序 ————

func upgrade(info sysservice.Info, exe string, prompt bool, dirFlag string, portFlag int) int {
	if sysservice.SamePath(exe, info.Exe) {
		fmt.Fprintf(out, "Momoko 已经安装在 %s（%s）。\n", info.Dir, info.State)
		fmt.Fprintln(out, "要更新：在面板的系统设置里在线更新，或者下载新版本后用新程序运行 momoko install。")
		return 0
	}
	installedVer := versionOf(info.Exe)
	fmt.Fprintf(out, "Momoko %s 已经安装在 %s（%s）。\n", installedVer, info.Dir, info.State)
	if (dirFlag != "" && !sysservice.SamePath(dirFlag, info.Dir)) || portFlag != 0 {
		fmt.Fprintln(out, "已安装时不改安装目录和端口（端口在 configs/config.yaml 里改；换目录先 momoko uninstall 再装）。")
	}
	if prompt && !confirm(fmt.Sprintf("用当前程序（%s）替换已安装的程序并重启服务？", version.Current()), true) {
		fmt.Fprintln(out, "已取消。")
		return 1
	}
	if code, elevated := asAdmin("install", []string{"-y"}); elevated {
		return code
	}

	fmt.Fprintf(out, "更新 %s …\n", info.Exe)
	if err := sysservice.Stop(); err != nil {
		return fail(fmt.Errorf("停止服务失败：%w", err))
	}
	backup := info.Exe + ".old"
	_ = os.Remove(backup)
	if err := os.Rename(info.Exe, backup); err != nil {
		_ = sysservice.Start()
		return fail(fmt.Errorf("备份旧程序失败：%w", err))
	}
	if err := copyFile(exe, info.Exe); err != nil {
		_ = os.Rename(backup, info.Exe)
		_ = sysservice.Start()
		return fail(fmt.Errorf("复制新程序失败：%w", err))
	}
	if err := sysservice.Register(info.Dir, info.Exe); err != nil {
		return fail(err)
	}
	_, _ = sysservice.LinkCommand(info.Dir, info.Exe)
	port := 0
	if p, err := initialize.ReadPorts(filepath.Join(info.Dir, "configs")); err == nil {
		port = p.HTTP
	}
	if err := sysservice.Start(); err != nil || (port > 0 && !sysservice.WaitPort(port, 60*time.Second)) {
		if err == nil {
			err = fmt.Errorf("60 秒内没有打开端口 %d", port)
		}
		fmt.Fprintf(out, "新程序没能正常启动（%v），换回旧程序…\n", err)
		_ = sysservice.Stop()
		_ = os.Remove(info.Exe)
		if rerr := os.Rename(backup, info.Exe); rerr != nil {
			return fail(fmt.Errorf("换回旧程序也失败了，请手动把 %s 改回 %s：%w", backup, info.Exe, rerr))
		}
		_ = sysservice.Start()
		return fail(fmt.Errorf("已换回 %s，日志：%s", installedVer, sysservice.LogHint(info.Dir)))
	}
	fmt.Fprintf(out, "已更新到 %s，服务已重启。旧程序备份在 %s。\n", version.Current(), backup)
	return 0
}

// versionOf 运行 exe -version 读版本号。
func versionOf(exe string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	b, err := exec.CommandContext(ctx, exe, "-version").Output()
	if err != nil {
		return "（版本未知）"
	}
	if v, _, ok := selfupdate.ParseVersionLine(string(b)); ok {
		return v
	}
	return "（版本未知）"
}

// ———— uninstall ————

func cmdUninstall(args []string) int {
	fs, relay := newFlags("uninstall")
	yes := fs.Bool("y", false, "不确认")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	defer useRelay(*relay)()
	info, ok, err := sysservice.Lookup()
	if err != nil {
		return fail(err)
	}
	if !ok {
		fmt.Fprintln(out, "Momoko 没有安装成系统服务。")
		return 0
	}
	if !*yes {
		if !interactive() {
			return fail(errors.New("要卸载请加 -y 确认"))
		}
		if !confirm(fmt.Sprintf("卸载 Momoko 系统服务？程序、配置和数据会留在 %s", info.Dir), false) {
			fmt.Fprintln(out, "已取消。")
			return 1
		}
	}
	if code, elevated := asAdmin("uninstall", []string{"-y"}); elevated {
		return code
	}
	if err := sysservice.Unregister(); err != nil {
		return fail(err)
	}
	if err := sysservice.UnlinkCommand(info.Dir, info.Exe); err != nil {
		fmt.Fprintln(out, "注意：", err)
	}
	sysservice.RemoveFirewall(info.Exe)
	fmt.Fprintf(out, "已卸载 Momoko 系统服务。程序、配置和数据还在 %s，不需要了可以直接删掉这个目录。\n", info.Dir)
	return 0
}

// ———— start / stop / restart ————

func cmdControl(name string, args []string) int {
	fs, relay := newFlags(name)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	defer useRelay(*relay)()
	info, ok, err := sysservice.Lookup()
	if err != nil {
		return fail(err)
	}
	if !ok {
		return fail(errors.New("Momoko 没有安装成系统服务，先运行 momoko install"))
	}
	// Windows 上本机用户不用提权就能启停（安装时给服务设了权限）；Linux / macOS 用 sudo
	if runtime.GOOS != "windows" {
		if code, elevated := asAdmin(name, nil); elevated {
			return code
		}
	}
	err = control(name)
	if errors.Is(err, sysservice.ErrAccessDenied) {
		if code, elevated := asAdmin(name, nil); elevated {
			return code
		}
	}
	if err != nil {
		return fail(fmt.Errorf("%w（日志：%s）", err, sysservice.LogHint(info.Dir)))
	}
	switch name {
	case "stop":
		fmt.Fprintln(out, "Momoko 已停止。")
	default:
		port := 0
		if p, err := initialize.ReadPorts(filepath.Join(info.Dir, "configs")); err == nil {
			port = p.HTTP
		}
		if port > 0 && !sysservice.WaitPort(port, 30*time.Second) {
			fmt.Fprintf(out, "服务已启动，但 30 秒内还没打开端口 %d，请看日志：%s\n", port, sysservice.LogHint(info.Dir))
			return 1
		}
		if name == "restart" {
			fmt.Fprintln(out, "Momoko 已重启。")
		} else {
			fmt.Fprintln(out, "Momoko 已启动。")
		}
		if port > 0 {
			fmt.Fprintf(out, "面板地址：%s\n", panelURLs(port)[0])
		}
	}
	return 0
}

func control(name string) error {
	switch name {
	case "start":
		return sysservice.Start()
	case "stop":
		return sysservice.Stop()
	default:
		if err := sysservice.Stop(); err != nil {
			return err
		}
		return sysservice.Start()
	}
}

// ———— status ————

func cmdStatus(args []string) int {
	fs, relay := newFlags("status")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	defer useRelay(*relay)()
	return printStatus()
}

// printStatus 显示服务状态。退出码：运行中 0，装了没运行 3，没装 4（和 systemctl status 一样）。
func printStatus() int {
	info, ok, err := sysservice.Lookup()
	if err != nil {
		return fail(err)
	}
	if !ok {
		fmt.Fprintln(out, "Momoko 没有安装成系统服务。")
		fmt.Fprintln(out, "安装：momoko install（会问安装目录和端口，需要管理员 / root 权限）")
		return 4
	}
	fmt.Fprintf(out, "Momoko 系统服务：%s\n", info.State)
	fmt.Fprintf(out, "  版本：%s\n", versionOf(info.Exe))
	port := 0
	if p, err := initialize.ReadPorts(filepath.Join(info.Dir, "configs")); err == nil {
		port = p.HTTP
	}
	printInfo(info.Dir, port, false)
	if info.State != sysservice.StateRunning {
		return 3
	}
	return 0
}

// installedHint 直接运行已安装的那份程序（双击、终端里敲 momoko）时：它已经在后台作为服务运行，不再起第二份，显示状态和用法。
func installedHint() int {
	code := printStatus()
	fmt.Fprintln(out)
	fmt.Fprintln(out, "这是已安装的 Momoko，在后台作为系统服务运行。用 momoko help 查看所有命令。")
	if sysservice.OwnConsole() {
		fmt.Fprint(out, "\n按回车键关闭…")
		_, _ = stdin.ReadString('\n')
	}
	if code == 3 {
		return 0
	}
	return code
}

// ———— run ————

// cmdRun 在前台运行：已安装的那份在安装目录里跑（排查服务起不来时用，先 momoko stop），没安装的和直接运行一样。
func cmdRun(args []string) int {
	fs, _ := newFlags("run")
	fs.StringVar(&flagconf, "conf", "./configs", "配置目录或文件")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if info, ok := sysservice.Installed(); ok {
		if err := os.Chdir(info.Dir); err != nil {
			return fail(err)
		}
		initialize.Re()
	}
	return serve(nil)
}
