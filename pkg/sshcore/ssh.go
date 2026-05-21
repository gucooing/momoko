package sshcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

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
		for {
			n, err := r.Read(buf)
			if n > 0 {
				if sendErr := sendText(string(buf[:n])); sendErr != nil {
					closeDone()
					return
				}
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
