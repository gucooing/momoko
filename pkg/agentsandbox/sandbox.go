// Package agentsandbox 让配了代理的 Agent 只能经代理联网：不是靠它自觉读代理环境变量，而是在系统层面断掉别的路。
//
// 只有 Linux 做得到，做法是：
//   - Agent 进程（连同它启动的 CLI、它执行的命令、它起的 MCP 服务器……整棵进程树）放进一个新的网络命名空间，
//     里面只有回环口，没有别的网卡、没有路由：直连任何外部地址、往外查 DNS 都直接失败；
//   - 回环口上唯一的出口是代理入口（ProxyAddr）：这个监听套接字由 Momoko 持有，每个连接由 Momoko 经上游代理转出去；
//   - 新的挂载命名空间里把本机解析服务的套接字（systemd-resolved、nscd、avahi）盖住，Agent 没法借它们在外面查 DNS；
//   - 去掉能离开这个网络的权限（CAP_SYS_ADMIN 等）：root 身份的 Agent 也不能 setns 回主机的网络。
//
// 进程先以「看门进程」（Momoko 自己，带 gateArg 参数）的身份在新的命名空间里起来，准备好上面这些、把代理入口交给 Momoko，
// 等 Momoko 接管后才换成真正的命令（exec，进程号不变）。哪一步不成都不会启动真正的命令。
//
// 刻意要绕的路不在防范之内：Agent 让系统里别的服务（Docker 守护进程、systemd 等）替它联网，那些服务按它们自己的网络配置走。
package agentsandbox

import "errors"

// ProxyAddr Agent 在自己的网络里看到的代理入口。网络是它独享的，端口不会和别的程序冲突。
const ProxyAddr = "127.0.0.1:41080"

// gateArg 看门进程的第一个参数。
const gateArg = "__momoko_agent_sandbox__"

// probeArg 只试一下能不能建隔离网络（不启动命令）。
const probeArg = "-probe"

// ErrUnsupported 这个系统给不了只走代理的隔离网络。
var ErrUnsupported = errors.New("只有 Linux 上能让 Agent 只走代理")
