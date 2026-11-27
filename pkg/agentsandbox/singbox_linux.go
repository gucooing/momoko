//go:build linux

package agentsandbox

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"momoko/pkg/agentinstall"
	"momoko/pkg/agentproxy"
)

// SingBoxRelease sing-box 在 GitHub 上给这个系统的发行包：静态编译的那个，包里有一个带版本号的顶层目录。
func SingBoxRelease() (agentinstall.Release, bool) {
	arch, ok := map[string]string{"amd64": "amd64", "arm64": "arm64", "386": "386", "arm": "armv7"}[runtime.GOARCH]
	if !ok {
		return agentinstall.Release{}, false
	}
	return agentinstall.Release{
		Name:  "sing-box",
		Repo:  "SagerNet/sing-box",
		Asset: "sing-box-{version}-linux-" + arch + ".tar.gz",
		Files: []string{"sing-box"},
	}, true
}

// singBoxReadyLine sing-box 起好后在日志里说的话（info 级别）。
const singBoxReadyLine = "sing-box started"

// singBoxLogKeep 留着的日志行数：起不来时拿来说明原因。
const singBoxLogKeep = 20

// singBox 一个沙箱的 sing-box：在主机的网络里跑，网卡建在沙箱的网络命名空间里。
type singBox struct {
	cmd  *exec.Cmd
	done chan struct{}

	mu  sync.Mutex
	log []string
}

// singBoxConfig 沙箱的 sing-box 配置：网卡 eth0 在 netns 里接管默认路由和 DNS；DNS 一律给假 IP，
// 连接按假 IP 还原成域名后经代理出去，域名由代理那边解析；代理服务器自己的地址用主机的解析（local）。
func singBoxConfig(p agentproxy.Proxy, netns string) map[string]any {
	outbound := map[string]any{
		"tag":             "proxy",
		"server":          p.Host,
		"server_port":     p.Port,
		"domain_resolver": "local",
	}
	if p.Type == agentproxy.TypeSOCKS5 {
		outbound["type"], outbound["version"] = "socks", "5"
	} else {
		outbound["type"] = "http"
	}
	if p.Username != "" {
		outbound["username"], outbound["password"] = p.Username, p.Password
	}
	return map[string]any{
		"log": map[string]any{"level": "info", "timestamp": false},
		"dns": map[string]any{
			"servers": []any{
				map[string]any{"type": "local", "tag": "local"},
				map[string]any{"type": "fakeip", "tag": "fakeip", "inet4_range": "198.18.0.0/15", "inet6_range": "fc00::/18"},
			},
			"final": "fakeip",
		},
		"inbounds": []any{map[string]any{
			"type":           "tun",
			"tag":            "tun",
			"interface_name": tunName,
			"address":        []string{tunAddress, tunAddress6},
			"mtu":            1500,
			"auto_route":     true,
			"strict_route":   true,
			"netns":          netns,
		}},
		"outbounds": []any{outbound},
		"route": map[string]any{
			"rules": []any{
				map[string]any{"action": "sniff"},
				map[string]any{"protocol": "dns", "action": "hijack-dns"},
			},
			"final": "proxy",
		},
	}
}

// startSingBox 起 sing-box（配置从标准输入给，不落盘），等它说起好了。它跟着 Momoko 走：Momoko 没了它也结束。
func startSingBox(cfg Config, netns string) (*singBox, error) {
	conf, err := json.Marshal(singBoxConfig(cfg.Proxy, netns))
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(cfg.SingBox, "run", "-c", "stdin", "-D", os.TempDir())
	cmd.Stdin = bytes.NewReader(conf)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = cmd.Stdout
	// 自成一个进程组：沙箱里命令的终端信号碰不到它
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动 sing-box 失败：%v", err)
	}
	sb := &singBox{cmd: cmd, done: make(chan struct{})}
	ready := make(chan struct{})
	go func() {
		var once sync.Once
		scanner := bufio.NewScanner(out)
		for scanner.Scan() {
			line := scanner.Text()
			sb.keep(line)
			if strings.Contains(line, singBoxReadyLine) {
				once.Do(func() { close(ready) })
			}
		}
		_, _ = io.Copy(io.Discard, out)
	}()
	go func() {
		_ = cmd.Wait()
		close(sb.done)
	}()
	select {
	case <-ready:
		return sb, nil
	case <-sb.done:
		return nil, fmt.Errorf("sing-box 起不来：%s", sb.reason())
	case <-time.After(stepTimeout):
		sb.stop()
		return nil, errors.New("sing-box 一直没有起好：" + sb.reason())
	}
}

func (sb *singBox) keep(line string) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.log = append(sb.log, line)
	if len(sb.log) > singBoxLogKeep {
		sb.log = sb.log[len(sb.log)-singBoxLogKeep:]
	}
}

// reason sing-box 最后说的几句。
func (sb *singBox) reason() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	if len(sb.log) == 0 {
		return "没有输出"
	}
	tail := sb.log
	if len(tail) > 3 {
		tail = tail[len(tail)-3:]
	}
	return strings.Join(tail, "；")
}

// stop 结束 sing-box：先请它退出，两秒后还在就强制结束。
func (sb *singBox) stop() {
	select {
	case <-sb.done:
		return
	default:
	}
	_ = sb.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-sb.done:
	case <-time.After(2 * time.Second):
		_ = sb.cmd.Process.Kill()
		<-sb.done
	}
}
