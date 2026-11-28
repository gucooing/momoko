package servercore

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"io"
	"net"
	"os"
	"sync"
	"time"

	pty "github.com/aymanbagabas/go-pty"
)

// 托管进程：面板为每个应用起一个（就是 momoko 自己，带 hostArg 参数），由它持有伪终端、启动应用并陪它到结束。
// 面板经本地套接字连上它收发控制台、发停止信号。面板重启（在线更新）时只断开连接，应用照常运行，
// 输出先存在托管进程里；新面板起来后连回来（Server.Attach），拿到进程号、启动时间和这段时间的输出。
// 应用退出后，托管进程把退出码交给面板再走；面板一直没来收，最多再等 hostLinger。

// hostArg 托管进程的命令行参数：momoko <hostArg>，启动参数从标准输入读（见 hostSpec）。
const hostArg = "__momoko-app-host"

// hostBacklog 托管进程存着的最近输出，面板连上时回放。
const hostBacklog = 256 << 10

// hostLinger 应用退出后等面板来收退出码的最长时间（面板正在重启时也能收到）。
var hostLinger = 10 * time.Minute

// hostOutputChunk 回放存着的输出时每帧的大小。
const hostOutputChunk = 32 << 10

// RunHost 当前进程是面板起的托管进程时，跑完托管逻辑后退出；不是就立即返回。
// main（以及测试的 TestMain）一开始就要调用。
func RunHost() {
	if len(os.Args) < 2 || os.Args[1] != hostArg {
		return
	}
	os.Exit(runHost(os.Stdin, os.Stdout))
}

func runHost(in io.Reader, out io.Writer) int {
	ready := func(r hostReady) {
		b, _ := json.Marshal(r)
		_, _ = out.Write(append(b, '\n'))
	}
	var spec hostSpec
	if err := json.NewDecoder(in).Decode(&spec); err != nil {
		ready(hostReady{Error: "读启动参数失败: " + err.Error(), Stage: hostStageSetup})
		return 2
	}
	h, stage, err := startHost(spec)
	if err != nil {
		ready(hostReady{Error: err.Error(), Stage: stage})
		return 2
	}
	ready(hostReady{PID: h.pid})
	// 面板只等这一行；之后不再碰标准输入输出（面板可能早已退出）
	detachStdio()
	return h.run()
}

type appHost struct {
	spec  hostSpec
	ptmx  pty.Pty
	cmd   *pty.Cmd
	pid   int
	start time.Time
	ln    net.Listener

	// ptyMu 保护 ptmx 的关闭：写入、调整尺寸拿读锁，关闭拿写锁
	ptyMu     sync.RWMutex
	ptyClosed bool

	mu      sync.Mutex
	backlog []byte
	client  *hostPeer
	waited  bool // 已经 Wait 到应用退出（之后不能再按进程号发信号：进程号可能已被复用）
	exited  bool // 退出码已经确定，输出也收完了
	exit    hostExitMsg

	readerDone chan struct{}
	quit       chan struct{}
	finishOnce sync.Once
}

func startHost(spec hostSpec) (*appHost, string, error) {
	_ = os.Remove(spec.Socket)
	ln, err := net.Listen("unix", spec.Socket)
	if err != nil {
		return nil, hostStageSetup, err
	}
	_ = os.Chmod(spec.Socket, 0o600)
	fail := func(stage string, err error) (*appHost, string, error) {
		_ = ln.Close()
		_ = os.Remove(spec.Socket)
		return nil, stage, err
	}

	ptmx, err := pty.New()
	if err != nil {
		return fail(hostStageSetup, err)
	}
	cmd := ptmx.Command(spec.Path, spec.Args...)
	cmd.Dir = spec.Dir
	if len(spec.Env) > 0 {
		cmd.Env = append(os.Environ(), spec.Env...)
	}
	configurePtyCmd(cmd)
	if err := cmd.Start(); err != nil {
		_ = ptmx.Close()
		return fail(hostStageStart, err)
	}
	if spec.Cols > 0 && spec.Rows > 0 {
		_ = ptmx.Resize(spec.Cols, spec.Rows)
	}
	h := &appHost{
		spec:       spec,
		ptmx:       ptmx,
		cmd:        cmd,
		start:      time.Now(),
		ln:         ln,
		readerDone: make(chan struct{}),
		quit:       make(chan struct{}),
	}
	if cmd.Process != nil {
		h.pid = cmd.Process.Pid
	}
	return h, "", nil
}

// run 陪应用到结束，交出退出码后返回托管进程的退出码。
func (h *appHost) run() int {
	go h.readOutput()
	go h.acceptLoop()
	stopSignals := hostSignals(func() { h.signal(false) })
	defer stopSignals()

	err := h.cmd.Wait()
	h.mu.Lock()
	h.waited = true
	h.mu.Unlock()

	// 留一点时间读完剩下的输出（Windows 的 ConPTY 不关就读不到结尾，只能等超时），再关伪终端
	select {
	case <-h.readerDone:
	case <-time.After(300 * time.Millisecond):
	}
	h.ptyMu.Lock()
	h.ptyClosed = true
	_ = h.ptmx.Close()
	h.ptyMu.Unlock()
	select {
	case <-h.readerDone:
	case <-time.After(time.Second):
	}

	h.mu.Lock()
	h.exited = true
	h.exit = hostExitMsg{Code: exitCode(err), At: time.Now().UnixMilli()}
	if h.client != nil {
		h.client.sendJSON(frameExit, h.exit)
	}
	h.mu.Unlock()

	select {
	case <-h.quit:
	case <-time.After(hostLinger):
		h.finish()
	}
	return 0
}

// finish 收尾：不再接受连接，删掉套接字和口令文件，再让 run 返回（进程退出时连接随之断开）。
// 面板收到断开时文件已经删了，紧接着重新启动不会又连上这个已经结束的托管进程。
func (h *appHost) finish() {
	h.finishOnce.Do(func() {
		_ = h.ln.Close()
		_ = os.Remove(h.spec.Socket)
		_ = os.Remove(h.spec.KeyFile)
		close(h.quit)
	})
}

func (h *appHost) readOutput() {
	defer close(h.readerDone)
	buf := make([]byte, hostOutputChunk)
	for {
		n, err := h.ptmx.Read(buf)
		if n > 0 {
			chunk := bytes.Clone(buf[:n])
			h.mu.Lock()
			h.appendBacklogLocked(chunk)
			if h.client != nil {
				h.client.send(frameOutput, chunk, false)
			}
			h.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

// appendBacklogLocked 存下输出，只留最后 hostBacklog 字节；截断时尽量从一行的开头留起。
func (h *appHost) appendBacklogLocked(chunk []byte) {
	h.backlog = append(h.backlog, chunk...)
	if len(h.backlog) <= hostBacklog {
		return
	}
	cut := len(h.backlog) - hostBacklog
	if i := bytes.IndexByte(h.backlog[cut:min(cut+4096, len(h.backlog))], '\n'); i >= 0 {
		cut += i + 1
	}
	h.backlog = append([]byte(nil), h.backlog[cut:]...)
}

func (h *appHost) acceptLoop() {
	for {
		conn, err := h.ln.Accept()
		if err != nil {
			return
		}
		go h.serve(conn)
	}
}

// serve 认口令，换下之前连着的面板，回放存着的输出，然后处理面板发来的帧直到断开。
func (h *appHost) serve(conn net.Conn) {
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	typ, payload, err := readFrame(conn)
	if err != nil || typ != frameHello {
		_ = conn.Close()
		return
	}
	var req hostHelloReq
	if json.Unmarshal(payload, &req) != nil || subtle.ConstantTimeCompare([]byte(req.Token), []byte(h.spec.Token)) != 1 {
		_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
		_ = writeFrame(conn, frameError, []byte("口令不对"))
		_ = conn.Close()
		return
	}
	_ = conn.SetReadDeadline(time.Time{})

	peer := newHostPeer(conn)
	h.mu.Lock()
	old := h.client
	h.client = peer
	peer.sendJSON(frameWelcome, hostWelcome{
		Version: hostProtoVersion,
		PID:     h.pid,
		Start:   h.start.UnixMilli(),
		Exited:  h.exited,
	})
	for rest := h.backlog; len(rest) > 0; {
		n := min(len(rest), hostOutputChunk)
		peer.send(frameOutput, bytes.Clone(rest[:n]), true)
		rest = rest[n:]
	}
	if h.exited {
		peer.sendJSON(frameExit, h.exit)
	}
	h.mu.Unlock()
	if old != nil {
		old.close()
	}

	h.readClient(peer)

	h.mu.Lock()
	if h.client == peer {
		h.client = nil
	}
	h.mu.Unlock()
	peer.close()
}

func (h *appHost) readClient(peer *hostPeer) {
	for {
		typ, payload, err := readFrame(peer.conn)
		if err != nil {
			return
		}
		switch typ {
		case frameInput:
			h.withPty(func() { _, _ = h.ptmx.Write(payload) })
		case frameResize:
			var rs hostResize
			if json.Unmarshal(payload, &rs) == nil && rs.Cols > 0 && rs.Rows > 0 {
				h.withPty(func() { _ = h.ptmx.Resize(rs.Cols, rs.Rows) })
			}
		case frameStop:
			h.signal(len(payload) > 0 && payload[0] == 1)
		case frameClear:
			h.mu.Lock()
			h.backlog = nil
			h.mu.Unlock()
		case frameQuit:
			h.mu.Lock()
			exited := h.exited
			h.mu.Unlock()
			if exited {
				h.finish()
			}
		}
	}
}

// withPty 伪终端还没关时执行 f。
func (h *appHost) withPty(f func()) {
	h.ptyMu.RLock()
	defer h.ptyMu.RUnlock()
	if !h.ptyClosed {
		f()
	}
}

// signal 结束应用（连同它的进程组）；已经退出的不再发。
func (h *appHost) signal(force bool) {
	h.mu.Lock()
	waited := h.waited
	h.mu.Unlock()
	if !waited {
		_ = stopPtyProcess(h.cmd.Process, force)
	}
}

// hostPeer 托管进程这边的一个面板连接：帧排队由单独的协程写出，读输出的协程不会被慢的面板卡住。
type hostPeer struct {
	conn net.Conn
	out  chan hostFrame
	done chan struct{}
	once sync.Once
}

type hostFrame struct {
	typ     byte
	payload []byte
}

func newHostPeer(conn net.Conn) *hostPeer {
	p := &hostPeer{conn: conn, out: make(chan hostFrame, 1024), done: make(chan struct{})}
	go p.writeLoop()
	return p
}

func (p *hostPeer) writeLoop() {
	for {
		select {
		case f := <-p.out:
			if err := writeFrame(p.conn, f.typ, f.payload); err != nil {
				p.close()
				return
			}
		case <-p.done:
			return
		}
	}
}

// send 排队一帧。面板落后太多（队列满）时，输出直接丢掉；must 的帧最多等几秒，等不到就断开这个面板。
func (p *hostPeer) send(typ byte, payload []byte, must bool) {
	f := hostFrame{typ: typ, payload: payload}
	select {
	case p.out <- f:
		return
	case <-p.done:
		return
	default:
	}
	if !must {
		return
	}
	select {
	case p.out <- f:
	case <-p.done:
	case <-time.After(5 * time.Second):
		p.close()
	}
}

func (p *hostPeer) sendJSON(typ byte, v any) {
	payload, err := json.Marshal(v)
	if err == nil {
		p.send(typ, payload, true)
	}
}

func (p *hostPeer) close() {
	p.once.Do(func() {
		close(p.done)
		_ = p.conn.Close()
	})
}
