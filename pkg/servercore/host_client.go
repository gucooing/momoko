package servercore

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// errHostDetached 面板主动断开了托管进程（面板要重启），应用还在运行。
var errHostDetached = errors.New("已断开托管进程，应用仍在运行")

// hostExitError 托管进程报来的非 0 退出码。
type hostExitError struct{ code int }

func (e hostExitError) Error() string { return fmt.Sprintf("exit status %d", e.code) }
func (e hostExitError) ExitCode() int { return e.code }

const (
	hostReadyTimeout = 10 * time.Second // 等托管进程起好的最长时间
	hostDialTimeout  = 2 * time.Second
	hostShakeTimeout = 5 * time.Second
	hostWriteTimeout = 5 * time.Second
	// maxSocketPath 本地套接字路径长度上限（Linux 108、macOS 104 字节，含结尾的 0），留些余量。
	maxSocketPath = 100
)

// hostOptions 面板这边起、找托管进程用的目录。
type hostOptions struct {
	dir     string // 口令文件所在目录（绝对路径）
	sockDir string // 套接字所在目录：一般同 dir，路径太长时换到临时目录
	exe     string // 起托管进程的程序；空时用当前程序
}

func newHostOptions(dir string) *hostOptions {
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	o := &hostOptions{dir: dir, sockDir: dir}
	if len(filepath.Join(dir, "0123456789abcdef.sock")) > maxSocketPath {
		sum := sha256.Sum256([]byte(dir))
		o.sockDir = filepath.Join(os.TempDir(), "momoko-apps-"+hex.EncodeToString(sum[:4]))
	}
	return o
}

// paths 应用 id 对应的套接字和口令文件（id 取哈希，文件名短且不受 id 里的字符影响）。
func (o *hostOptions) paths(id string) (sock, key string) {
	sum := sha256.Sum256([]byte(id))
	name := hex.EncodeToString(sum[:8])
	return filepath.Join(o.sockDir, name+".sock"), filepath.Join(o.dir, name+".key")
}

func (o *hostOptions) ensureDirs() error {
	for _, d := range []string{o.dir, o.sockDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
		_ = os.Chmod(d, 0o700)
	}
	return nil
}

// dial 连上应用 id 的托管进程并认证。没有托管进程（或已经不在了）时返回错误，并清掉留下的文件。
func (o *hostOptions) dial(id string) (net.Conn, hostWelcome, error) {
	sock, keyFile := o.paths(id)
	token, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, hostWelcome{}, err
	}
	conn, err := net.DialTimeout("unix", sock, hostDialTimeout)
	if err != nil {
		_ = os.Remove(sock)
		_ = os.Remove(keyFile)
		return nil, hostWelcome{}, err
	}
	welcome, err := hostHandshake(conn, string(token))
	if err != nil {
		_ = conn.Close()
		return nil, hostWelcome{}, err
	}
	return conn, welcome, nil
}

func hostHandshake(conn net.Conn, token string) (hostWelcome, error) {
	_ = conn.SetDeadline(time.Now().Add(hostShakeTimeout))
	defer func() { _ = conn.SetDeadline(time.Time{}) }()
	if err := writeJSONFrame(conn, frameHello, hostHelloReq{Version: hostProtoVersion, Token: token}); err != nil {
		return hostWelcome{}, err
	}
	typ, payload, err := readFrame(conn)
	if err != nil {
		return hostWelcome{}, err
	}
	switch typ {
	case frameWelcome:
		var w hostWelcome
		if err := json.Unmarshal(payload, &w); err != nil {
			return hostWelcome{}, err
		}
		return w, nil
	case frameError:
		return hostWelcome{}, fmt.Errorf("托管进程拒绝连接: %s", payload)
	default:
		return hostWelcome{}, fmt.Errorf("托管进程回了不认识的帧 %q", typ)
	}
}

// collectExit 收下已退出应用的输出和退出信息，告诉托管进程可以走了。
func collectExit(conn net.Conn) ([]byte, hostExitMsg) {
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(hostShakeTimeout))
	var out bytes.Buffer
	exit := hostExitMsg{Code: -1}
	for {
		typ, payload, err := readFrame(conn)
		if err != nil {
			return out.Bytes(), exit
		}
		switch typ {
		case frameOutput:
			out.Write(payload)
		case frameExit:
			_ = json.Unmarshal(payload, &exit)
			if writeFrame(conn, frameQuit, nil) == nil {
				waitHostClose(conn)
			}
			return out.Bytes(), exit
		}
	}
}

// waitHostClose 发了退出帧后等托管进程断开：它删掉套接字和口令文件后才退出。
func waitHostClose(conn net.Conn) {
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _ = io.Copy(io.Discard, conn)
}

// hostSetupError 托管进程自己没起来（不是应用的问题），调用方可以改为直接运行。
type hostSetupError struct{ err error }

func (e hostSetupError) Error() string { return "托管进程没起来: " + e.err.Error() }
func (e hostSetupError) Unwrap() error { return e.err }

// spawn 起一个托管进程运行 spec，连上并认证。
func (o *hostOptions) spawn(spec hostSpec) (net.Conn, hostWelcome, error) {
	if err := o.ensureDirs(); err != nil {
		return nil, hostWelcome{}, hostSetupError{err}
	}
	spec.Socket, spec.KeyFile = o.paths(spec.ID)
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return nil, hostWelcome{}, hostSetupError{err}
	}
	spec.Token = hex.EncodeToString(raw[:])
	if err := os.WriteFile(spec.KeyFile, []byte(spec.Token), 0o600); err != nil {
		return nil, hostWelcome{}, hostSetupError{err}
	}
	cleanup := func() {
		_ = os.Remove(spec.KeyFile)
	}

	exe := o.exe
	if exe == "" {
		var err error
		if exe, err = os.Executable(); err != nil {
			cleanup()
			return nil, hostWelcome{}, hostSetupError{err}
		}
	}
	input, err := json.Marshal(spec)
	if err != nil {
		cleanup()
		return nil, hostWelcome{}, hostSetupError{err}
	}

	pr, pw, err := os.Pipe()
	if err != nil {
		cleanup()
		return nil, hostWelcome{}, hostSetupError{err}
	}
	start := func(breakaway bool) (*exec.Cmd, error) {
		cmd := exec.Command(exe, hostArg)
		cmd.Dir = o.dir
		cmd.Stdin = bytes.NewReader(input)
		cmd.Stdout = pw
		setHostProcAttr(cmd, breakaway)
		return cmd, cmd.Start()
	}
	cmd, err := start(true)
	if err != nil {
		// Windows：面板所在的作业不允许脱离时 CREATE_BREAKAWAY_FROM_JOB 会失败，去掉再试
		cmd, err = start(false)
	}
	_ = pw.Close()
	if err != nil {
		_ = pr.Close()
		cleanup()
		return nil, hostWelcome{}, hostSetupError{err}
	}
	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait() // 收尸；面板退出后托管进程由系统接管
		close(exited)
	}()

	readyCh := make(chan hostReady, 1)
	go func() {
		defer func() { _ = pr.Close() }()
		line, err := bufio.NewReader(pr).ReadBytes('\n')
		var r hostReady
		if err != nil {
			r.Error, r.Stage = "托管进程没回话就退出了", hostStageSetup
		} else if jerr := json.Unmarshal(line, &r); jerr != nil {
			r.Error, r.Stage = "托管进程回话看不懂: "+jerr.Error(), hostStageSetup
		}
		readyCh <- r
	}()

	var ready hostReady
	select {
	case ready = <-readyCh:
	case <-time.After(hostReadyTimeout):
		_ = cmd.Process.Kill()
		cleanup()
		return nil, hostWelcome{}, hostSetupError{errors.New("等托管进程起来超时")}
	}
	if ready.Error != "" {
		<-exited
		cleanup()
		if ready.Stage == hostStageStart {
			return nil, hostWelcome{}, fmt.Errorf("启动子进程失败: %s", ready.Error)
		}
		return nil, hostWelcome{}, hostSetupError{errors.New(ready.Error)}
	}

	conn, err := net.DialTimeout("unix", spec.Socket, hostDialTimeout)
	if err != nil {
		_ = cmd.Process.Kill()
		cleanup()
		return nil, hostWelcome{}, hostSetupError{err}
	}
	welcome, err := hostHandshake(conn, spec.Token)
	if err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		cleanup()
		return nil, hostWelcome{}, hostSetupError{err}
	}
	return conn, welcome, nil
}

// hostClient 面板这边的托管进程连接，充当 Server 的终端：读到的是应用输出，写入的是键盘输入。
type hostClient struct {
	conn    net.Conn
	welcome hostWelcome

	wmu sync.Mutex
	pr  *io.PipeReader
	pw  *io.PipeWriter

	done     chan struct{}
	exited   bool
	exit     hostExitMsg
	err      error
	detached atomic.Bool
}

func newHostClient(conn net.Conn, welcome hostWelcome) *hostClient {
	pr, pw := io.Pipe()
	c := &hostClient{conn: conn, welcome: welcome, pr: pr, pw: pw, done: make(chan struct{})}
	go c.readLoop()
	return c
}

// readLoop 把输出帧原样写进管道（Server.readPipe 在另一头读），收到退出帧或连接断开时结束。
// 管道写入要等对面读完才返回，所以退出帧之前的输出都已经交给 Server 了。
func (c *hostClient) readLoop() {
	defer close(c.done)
	defer func() { _ = c.pw.Close() }()
	for {
		typ, payload, err := readFrame(c.conn)
		if err != nil {
			c.err = err
			return
		}
		switch typ {
		case frameOutput:
			_, _ = c.pw.Write(payload)
		case frameExit:
			_ = json.Unmarshal(payload, &c.exit)
			c.exited = true
			return
		}
	}
}

func (c *hostClient) Read(p []byte) (int, error) { return c.pr.Read(p) }

func (c *hostClient) Write(p []byte) (int, error) {
	for rest := p; len(rest) > 0; {
		n := min(len(rest), hostOutputChunk)
		if err := c.writeFrame(frameInput, rest[:n]); err != nil {
			return len(p) - len(rest), err
		}
		rest = rest[n:]
	}
	return len(p), nil
}

func (c *hostClient) Resize(cols, rows int) error {
	payload, err := json.Marshal(hostResize{Cols: cols, Rows: rows})
	if err != nil {
		return err
	}
	return c.writeFrame(frameResize, payload)
}

func (c *hostClient) writeFrame(typ byte, payload []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(hostWriteTimeout))
	return writeFrame(c.conn, typ, payload)
}

// stop 让托管进程结束应用；应用已经退出（连接也断了）时返回 os.ErrProcessDone。
func (c *hostClient) stop(force bool) error {
	select {
	case <-c.done:
		return os.ErrProcessDone
	default:
	}
	var b byte
	if force {
		b = 1
	}
	if err := c.writeFrame(frameStop, []byte{b}); err != nil {
		return os.ErrProcessDone
	}
	return nil
}

func (c *hostClient) clear() { _ = c.writeFrame(frameClear, nil) }

// wait 等应用退出（或连接断开），返回值的含义同 exec.Cmd.Wait。
func (c *hostClient) wait() error {
	<-c.done
	switch {
	case c.detached.Load():
		return errHostDetached
	case c.exited && c.exit.Code == 0:
		return nil
	case c.exited:
		return hostExitError{code: c.exit.Code}
	default:
		return fmt.Errorf("和托管进程的连接断了: %w", c.err)
	}
}

// release 应用退出后调用：告诉托管进程可以走了，等它收尾后关掉连接（readLoop 收到退出帧就不再读了）。
func (c *hostClient) release() {
	if c.exited && c.writeFrame(frameQuit, nil) == nil {
		waitHostClose(c.conn)
	}
	_ = c.conn.Close()
}

// detach 断开连接，应用继续在托管进程里运行。
func (c *hostClient) detach() {
	c.detached.Store(true)
	_ = c.conn.Close()
}

// hostedResult 把托管进程连接包装成 Server 的一次运行。
func hostedResult(c *hostClient) *startResult {
	r := &startResult{
		pid:     c.welcome.PID,
		term:    c,
		waitFn:  c.wait,
		stopFn:  c.stop,
		closeFn: c.release,
		detach:  c.detach,
		clear:   c.clear,
	}
	if c.welcome.Start > 0 {
		r.startTime = time.UnixMilli(c.welcome.Start)
	}
	return r
}
