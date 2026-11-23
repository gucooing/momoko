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

	ptmx, err := pty.New()
	if err != nil {
		_ = websocket.Message.Send(conn, fmt.Sprintf("[system] 创建伪终端失败: %v\r\n", err))
		return err
	}

	name, args, env := shellCommand()
	cmd := ptmx.Command(name, args...)
	cmd.Dir = cwd
	// 用注册表现在的环境变量（Momoko 启动后装的工具也找得到），不是 Momoko 启动那一刻的
	cmd.Env = append(sysenv.Environ(), env...)
	configureCmd(cmd)
	if err := cmd.Start(); err != nil {
		_ = ptmx.Close()
		_ = websocket.Message.Send(conn, fmt.Sprintf("[system] 启动 %s 失败: %v\r\n", name, err))
		return err
	}
	_ = ptmx.Resize(defaultCols, defaultRows)

	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
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
			go terminate(cmd.Process, exited)
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
