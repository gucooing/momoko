// Package agentsandbox 让配了代理的 Agent 在沙箱里跑：只经代理联网，环境里看不到代理，时区和语言是配置的。
//
// 只有 Linux 做得到。Start 是启动这种进程的唯一入口（Agent 进程、检查登录、登录终端、会话里的 Git 推拉都经它）：
//   - 进程在新的网络、挂载、进程号命名空间里。网络里只有回环口和 sing-box 建的网卡 eth0：默认路由和 DNS 都指向它，
//     sing-box 在沙箱外面（主机的网络里）把每个连接经代理转出去，域名换成假 IP，真正的解析交给代理那边；
//     sing-box 停了网卡就没了，沙箱里没有网，不会改成直连；
//   - 环境里没有任何代理变量，/etc/resolv.conf 指向 eth0 上的 DNS，时区（TZ、/etc/localtime、/etc/timezone）
//     和语言（LANG、LANGUAGE）是配置的；进程号命名空间里只看得到自己这一棵进程树；
//   - 盖住本机解析服务（systemd-resolved、nscd、avahi）和能替它联网的服务（Docker、containerd、Podman、snapd、
//     D-Bus 系统总线、systemd）的套接字；去掉 CAP_SYS_ADMIN、CAP_NET_ADMIN、CAP_SYS_PTRACE 等权限，root 也回不到主机的网络。
//
// 沙箱里最先起来的是 Momoko 自己（带 gateArg 参数，进程号命名空间里的 1 号进程）：做好上面这些，等 Momoko 在外面起好 sing-box、
// 网卡出现后，再起真正的命令并一直陪着它（转发信号、收尸），命令退出它就退出，命名空间里剩下的进程跟着被内核结束。
package agentsandbox

import (
	"errors"

	"momoko/pkg/agentproxy"
)

// Config 一个沙箱：经哪个代理联网（连同时区、语言），用哪个 sing-box。
type Config struct {
	Proxy agentproxy.Proxy
	// SingBox sing-box 可执行文件的路径（由 Momoko 下载并更新，见 SingBoxRelease）。
	SingBox string
}

// selfExe 沙箱里的 Momoko（1 号进程、替 Momoko 发请求的）就是这个程序：/proc/self/exe 在 fork 之后指向的还是它，
// 程序文件被换掉了也能起。
const selfExe = "/proc/self/exe"

const (
	// gateArg 沙箱里的 1 号进程（Momoko 自己）的第一个参数。
	gateArg = "__momoko_agent_sandbox__"
	// probeArg 只试一下能不能建沙箱（不启动命令）。
	probeArg = "-probe"
	// timezoneEnv 交给 1 号进程的时区（它换好 /etc/localtime 后从环境里去掉）。
	timezoneEnv = "MOMOKO_SANDBOX_TZ"
	// tunName 沙箱里 sing-box 建的网卡。
	tunName = "eth0"
	// tunAddress 网卡的地址；DNS 在下一个地址上（sing-box 的默认）。
	tunAddress  = "172.19.0.1/30"
	tunAddress6 = "fdfe:dcba:9876::1/126"
	tunDNS      = "172.19.0.2"
)

// ErrUnsupported 这个系统建不了沙箱。
var ErrUnsupported = errors.New("只有 Linux 上能让 Agent 在只走代理的沙箱里跑")
