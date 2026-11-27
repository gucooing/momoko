// Package shellcore 在本机伪终端里启动默认 shell，经 WebSocket 收发（协议见 pkg/termws，与 SSH 终端一致）。
// 连接断开即结束 shell（Unix 下连同它的进程组）；shell 自己退出则关闭连接。
package shellcore

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	pty "github.com/aymanbagabas/go-pty"
	"golang.org/x/net/websocket"

	"momoko/pkg/agentsandbox"
	"momoko/pkg/sysenv"
	"momoko/pkg/termws"
)

const (
	defaultCols    = 120
	defaultRows    = 32
	wsPingInterval = 20 * time.Second
	// killGrace 断开后给 shell 收尾的时间，过了还在就强制结束。
	killGrace = 2 * time.Second
)

// ServeWebSocket 在本机启动一个 shell 并把它接到 conn 上，直到任一端结束。
// dir 是起始目录：空的用主目录；给了就必须是存在的目录的绝对路径，否则在终端里说明原因、不启动 shell。
func ServeWebSocket(conn *websocket.Conn, dir string) error {
	cwd, err := resolveDir(dir)
	if err != nil {
		_ = websocket.Message.Send(conn, fmt.Sprintf("[system] %v\r\n", err))
		return err
	}
	name, args, env := shellCommand()
	// 用注册表现在的环境变量（Momoko 启动后装的工具也找得到），不是 Momoko 启动那一刻的
	return serve(conn, Spec{Path: name, Args: args, Dir: cwd, Env: append(sysenv.Environ(), env...)}, "")
}

// Spec 在伪终端里跑的一条命令。
type Spec struct {
	// Path 可执行文件的绝对路径（Windows 上的 .cmd / .bat 交给 cmd.exe）。
	Path string
	// Args 参数，不含命令本身。
	Args []string
	Dir  string
	// Env 完整的环境（KEY=VALUE）。
	Env []string
	// Sandbox 不为空时这条命令（连同它的子进程）在沙箱里跑：只有 Linux 做得到，见 pkg/agentsandbox。
	Sandbox *agentsandbox.Config
}

// TermEnv 终端里跑的程序都该有的环境（KEY=VALUE）：Unix 上是真彩色的 xterm、长输出不分页。
func TermEnv() []string {
	_, _, env := shellCommand()
	return env
}

// ServeCommand 在本机的伪终端里跑一条命令，接到 conn 上，直到命令退出或连接断开；命令退出时先把 farewell（可为空）写到终端里。
func ServeCommand(conn *websocket.Conn, spec Spec, farewell string) error {
	return serve(conn, spec, farewell)
}

// serve 起 spec 这条命令并把伪终端接到 conn 上。
func serve(conn *websocket.Conn, spec Spec, farewell string) error {
	ptmx, err := pty.New()
	if err != nil {
		_ = websocket.Message.Send(conn, fmt.Sprintf("[system] 创建伪终端失败: %v\r\n", err))
		return err
	}
	proc, wait, err := start(ptmx, spec)
	if err != nil {
		_ = ptmx.Close()
		_ = websocket.Message.Send(conn, fmt.Sprintf("[system] 启动 %s 失败: %v\r\n", filepath.Base(spec.Path), err))
		return err
	}
	_ = ptmx.Resize(defaultCols, defaultRows)

	exited := make(chan struct{})
	go func() {
		_ = wait()
		close(exited)
	}()

	var sendMu sync.Mutex
	sendText := func(text string) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		return websocket.Message.Send(conn, text)
	}
	sendPing := func() error {
		sendMu.Lock()
		defer sendMu.Unlock()
		payloadType := conn.PayloadType
		conn.PayloadType = websocket.PingFrame
		defer func() { conn.PayloadType = payloadType }()
		_, err := conn.Write(nil)
		return err
	}

	done := make(chan struct{})
	var doneOnce sync.Once
	closeDone := func() {
		doneOnce.Do(func() {
			close(done)
			_ = conn.Close()
			go terminate(proc, exited)
			_ = ptmx.Close()
		})
	}
	defer closeDone()

	// 终端输出 → 文本帧（被缓冲区截断的多字节字符留到下一次一起发送）
	go func() {
		buf := make([]byte, 8192)
		pending := 0
		for {
			n, err := ptmx.Read(buf[pending:])
			n += pending
			if n > 0 {
				cut := termws.UTF8Boundary(buf[:n])
				if err != nil {
					cut = n
				}
				if cut > 0 {
					if sendErr := sendText(string(buf[:cut])); sendErr != nil {
						closeDone()
						return
					}
				}
				pending = copy(buf, buf[cut:n])
			}
			if err != nil {
				closeDone()
				return
			}
		}
	}()

	go func() {
		select {
		case <-exited:
			if farewell != "" {
				// 让最后几行输出先到
				time.Sleep(200 * time.Millisecond)
				_ = sendText(farewell)
			}
			closeDone()
		case <-done:
		}
	}()

	go func() {
		ticker := time.NewTicker(wsPingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := sendPing(); err != nil {
					closeDone()
					return
				}
			}
		}
	}()

	for {
		var input string
		if err := websocket.Message.Receive(conn, &input); err != nil {
			return nil
		}
		if resize, ok := termws.ParseResize(input); ok {
			_ = ptmx.Resize(resize.Cols, resize.Rows)
			continue
		}
		if _, err := io.WriteString(ptmx, input); err != nil {
			return nil
		}
	}
}

// start 起命令：要进沙箱的交给 startSandboxed（只有 Linux），别的用 go-pty 起。
func start(ptmx pty.Pty, spec Spec) (*os.Process, func() error, error) {
	if spec.Sandbox != nil {
		return startSandboxed(ptmx, spec)
	}
	cmd := command(ptmx, spec.Path, spec.Args)
	cmd.Dir = spec.Dir
	cmd.Env = spec.Env
	configureCmd(cmd)
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	return cmd.Process, cmd.Wait, nil
}

// resolveDir 指定的起始目录：要绝对路径、要存在、要是目录；不满足就报错，不悄悄换成主目录。没指定就用 workDir。
func resolveDir(dir string) (string, error) {
	if dir == "" {
		return workDir(), nil
	}
	if !filepath.IsAbs(dir) {
		return "", fmt.Errorf("打不开目录 %s：不是绝对路径", dir)
	}
	dir = filepath.Clean(dir)
	info, err := os.Stat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", fmt.Errorf("打不开目录 %s：目录不存在", dir)
	case err != nil:
		return "", fmt.Errorf("打不开目录 %s：%v", dir, err)
	case !info.IsDir():
		return "", fmt.Errorf("打不开目录 %s：不是目录", dir)
	}
	return dir, nil
}

// workDir shell 的默认起始目录：运行 Momoko 的用户的主目录，取不到就用当前目录。
func workDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		if info, err := os.Stat(home); err == nil && info.IsDir() {
			return home
		}
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return ""
}
