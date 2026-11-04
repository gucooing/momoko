package sshcore

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/ssh"
	"golang.org/x/net/websocket"
)

const (
	AuthPassword       = "password"
	AuthKey            = "key"
	defaultDialTimeout = 10 * time.Second
	defaultCols        = 120
	defaultRows        = 32
	wsPingInterval     = 20 * time.Second
	// latencyInterval 终端会话中测量 momoko → SSH 服务端往返延迟的间隔。
	latencyInterval = 5 * time.Second
)

type Config struct {
	Host        string
	Port        int
	Username    string
	AuthType    string
	Credential  string
	Passphrase  string
	Fingerprint string
	Timeout     time.Duration
}

type TestResult struct {
	Fingerprint string
}

type wsControl struct {
	Type string `json:"type"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

// ServerControl 是服务端发往浏览器的控制帧（以 WebSocket 二进制帧发送，终端输出始终是文本帧）。
//   - {"type":"latency","ms":42}：momoko 到 SSH 服务端的往返延迟（keepalive 请求测得）
type ServerControl struct {
	Type string `json:"type"`
	Ms   int64  `json:"ms,omitempty"`
}

// KeyType 解析私钥并返回简短的算法描述（ed25519、rsa-4096、ecdsa-256 …）；解析失败返回空串。
func KeyType(privateKey, passphrase string) string {
	if strings.TrimSpace(privateKey) == "" {
		return ""
	}
	var (
		raw any
		err error
	)
	if passphrase == "" {
		raw, err = ssh.ParseRawPrivateKey([]byte(privateKey))
	} else {
		raw, err = ssh.ParseRawPrivateKeyWithPassphrase([]byte(privateKey), []byte(passphrase))
	}
	if err != nil {
		return ""
	}
	switch k := raw.(type) {
	case *rsa.PrivateKey:
		return fmt.Sprintf("rsa-%d", k.N.BitLen())
	case *ecdsa.PrivateKey:
		return fmt.Sprintf("ecdsa-%d", k.Curve.Params().BitSize)
	}
	signer, err := ssh.NewSignerFromKey(raw)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(signer.PublicKey().Type(), "ssh-")
}

func Test(ctx context.Context, cfg Config) (*TestResult, error) {
	var fingerprint string
	client, err := dial(ctx, cfg, func(value string) {
		fingerprint = value
	})
	if err != nil {
		return nil, err
	}
	_ = client.Close()
	return &TestResult{Fingerprint: fingerprint}, nil
}

func ServeWebSocket(conn *websocket.Conn, cfg Config) error {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultDialTimeout
	}
	dialCtx, cancelDial := context.WithTimeout(context.Background(), timeout)
	defer cancelDial()

	client, err := dial(dialCtx, cfg, nil)
	if err != nil {
		_ = websocket.Message.Send(conn, fmt.Sprintf("[system] SSH 连接失败: %v\r\n", err))
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		_ = client.Close()
		_ = websocket.Message.Send(conn, fmt.Sprintf("[system] SSH 会话创建失败: %v\r\n", err))
		return err
	}

	var stopSSHOnce sync.Once
	stopSSH := func() {
		stopSSHOnce.Do(func() {
			_ = session.Close()
			_ = client.Close()
		})
	}
	defer stopSSH()

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return err
	}

	if err := session.RequestPty("xterm-256color", defaultRows, defaultCols, ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}); err != nil {
		return err
	}
	if err := session.Shell(); err != nil {
		return err
	}

	var sendMu sync.Mutex
	sendText := func(text string) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		return websocket.Message.Send(conn, text)
	}
	sendControl := func(control ServerControl) error {
		payload, err := json.Marshal(control)
		if err != nil {
			return err
		}
		sendMu.Lock()
		defer sendMu.Unlock()
		return websocket.Message.Send(conn, payload)
	}
	sendPing := func() error {
		sendMu.Lock()
		defer sendMu.Unlock()

		payloadType := conn.PayloadType
		conn.PayloadType = websocket.PingFrame
		defer func() {
			conn.PayloadType = payloadType
		}()

		_, err := conn.Write(nil)
		return err
	}

	done := make(chan struct{})
	var doneOnce sync.Once
	closeDone := func() {
		doneOnce.Do(func() {
			close(done)
			_ = conn.Close()
			stopSSH()
		})
	}

	copyOutput := func(r io.Reader) {
		buf := make([]byte, 4096)
		pending := 0 // 上一次读到末尾、尚不完整的 UTF-8 字节数（已移到 buf 开头）
		for {
			n, err := r.Read(buf[pending:])
			n += pending
			if n > 0 {
				// 文本帧必须是合法 UTF-8：被缓冲区截断的多字节字符留到下一次一起发送
				cut := utf8Boundary(buf[:n])
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
				return
			}
		}
	}

	go copyOutput(stdout)
	go copyOutput(stderr)

	go func() {
		err := session.Wait()
		if err != nil {
			_ = sendText(fmt.Sprintf("\r\n[SSH] 会话已结束: %v\r\n", err))
		} else {
			_ = sendText("\r\n[SSH] 会话已结束\r\n")
		}
		closeDone()
	}()

	go func() {
		measure := func() {
			started := time.Now()
			// keepalive 请求：OpenSSH 等服务端会回复（成功或失败均可），往返即为延迟
			if _, _, err := client.SendRequest("keepalive@openssh.com", true, nil); err != nil {
				return
			}
			_ = sendControl(ServerControl{Type: "latency", Ms: max(time.Since(started).Milliseconds(), 1)})
		}
		measure()
		ticker := time.NewTicker(latencyInterval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				measure()
			}
		}
	}()

	go func() {
		pingTicker := time.NewTicker(wsPingInterval)
		defer pingTicker.Stop()

		for {
			select {
			case <-done:
				return
			case <-pingTicker.C:
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
			closeDone()
			return nil
		}
		if resize, ok := parseResize(input); ok {
			_ = session.WindowChange(resize.Rows, resize.Cols)
			continue
		}
		if _, err := io.WriteString(stdin, input); err != nil {
			_ = sendText(fmt.Sprintf("\r\n[SSH] 输入写入失败: %v\r\n", err))
			closeDone()
			return nil
		}
	}
}

func dial(ctx context.Context, cfg Config, captureFingerprint func(string)) (*ssh.Client, error) {
	auth, err := authMethod(cfg)
	if err != nil {
		return nil, err
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultDialTimeout
	}
	clientConfig := &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: hostKeyCallback(cfg.Fingerprint, captureFingerprint),
		Timeout:         timeout,
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, clientConfig)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return ssh.NewClient(sshConn, chans, reqs), nil
}

func authMethod(cfg Config) (ssh.AuthMethod, error) {
	switch cfg.AuthType {
	case AuthPassword:
		if cfg.Credential == "" {
			return nil, errors.New("password is empty")
		}
		return ssh.Password(cfg.Credential), nil
	case AuthKey:
		if cfg.Credential == "" {
			return nil, errors.New("private key is empty")
		}
		var (
			signer ssh.Signer
			err    error
		)
		if cfg.Passphrase == "" {
			signer, err = ssh.ParsePrivateKey([]byte(cfg.Credential))
		} else {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(cfg.Credential), []byte(cfg.Passphrase))
		}
		if err != nil {
			return nil, err
		}
		return ssh.PublicKeys(signer), nil
	default:
		return nil, errors.New("unsupported ssh auth type")
	}
}

func hostKeyCallback(expected string, capture func(string)) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		fingerprint := ssh.FingerprintSHA256(key)
		if capture != nil {
			capture(fingerprint)
		}
		expected = strings.TrimSpace(expected)
		if expected == "" || fingerprint == expected {
			return nil
		}
		return fmt.Errorf("ssh host fingerprint mismatch")
	}
}

// utf8Boundary 返回 b 中最后一个完整 UTF-8 字符之后的位置（末尾最多 3 个字节可能是被截断的字符）。
func utf8Boundary(b []byte) int {
	n := len(b)
	for i := 1; i <= utf8.UTFMax-1 && i <= n; i++ {
		c := b[n-i]
		if c < 0x80 {
			return n // ASCII：此前都完整
		}
		if utf8.RuneStart(c) {
			if utf8.FullRune(b[n-i:]) {
				return n
			}
			return n - i
		}
	}
	return n
}

func parseResize(input string) (wsControl, bool) {
	if !strings.HasPrefix(strings.TrimSpace(input), "{") {
		return wsControl{}, false
	}
	var control wsControl
	if err := json.Unmarshal([]byte(input), &control); err != nil {
		return wsControl{}, false
	}
	if control.Type != "resize" || control.Cols <= 0 || control.Rows <= 0 {
		return wsControl{}, false
	}
	return control, true
}
