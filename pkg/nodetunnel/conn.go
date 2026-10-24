// Package nodetunnel 把 NodeAgent.Tunnel 双向流适配为 net.Conn：
// 一条 gRPC 流承载一条 HTTP/1.1 连接的原始字节，使 WebSocket 终端、文件预签名上传/下载
// 等非一元调用的流量也能经节点 gRPC 端口转发，节点侧复用自身的 HTTP 处理逻辑。
package nodetunnel

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1 "momoko/api/gen/v1"
)

// maxPacketSize 单个隧道包的最大载荷，远低于 gRPC 默认 4MB 消息上限。
const maxPacketSize = 32 * 1024

type packetStream interface {
	Send(*v1.TunnelPacket) error
	Recv() (*v1.TunnelPacket, error)
}

// Addr 是隧道连接的地址描述。
type Addr string

func (a Addr) Network() string { return "momoko-tunnel" }
func (a Addr) String() string  { return string(a) }

type streamConn struct {
	stream  packetStream
	onClose func()

	// 接收协程把 Recv 结果逐包交给 Read，使读操作可被读超时打断（net/http 依赖这一点结束后台读）。
	recvCh   chan []byte
	recvDone chan struct{}
	recvErr  error

	readMu       sync.Mutex
	buf          []byte
	readDeadline deadline

	writeMu       sync.Mutex
	writeDeadline deadline

	closeOnce sync.Once
	closed    chan struct{}

	local, remote net.Addr
}

func newStreamConn(stream packetStream, local, remote net.Addr, onClose func()) *streamConn {
	c := &streamConn{
		stream:        stream,
		onClose:       onClose,
		recvCh:        make(chan []byte),
		recvDone:      make(chan struct{}),
		readDeadline:  newDeadline(),
		writeDeadline: newDeadline(),
		closed:        make(chan struct{}),
		local:         local,
		remote:        remote,
	}
	go c.recvLoop()
	return c
}

func (c *streamConn) recvLoop() {
	defer close(c.recvDone)
	for {
		packet, err := c.stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
				err = io.EOF
			}
			c.recvErr = err
			return
		}
		data := packet.GetData()
		if len(data) == 0 {
			continue
		}
		select {
		case c.recvCh <- data:
		case <-c.closed:
			return
		}
	}
}

func (c *streamConn) Read(p []byte) (int, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()
	if len(c.buf) == 0 {
		select {
		case data := <-c.recvCh:
			c.buf = data
		case <-c.recvDone:
			return 0, c.recvErr
		case <-c.closed:
			return 0, net.ErrClosed
		case <-c.readDeadline.wait():
			return 0, os.ErrDeadlineExceeded
		}
	}
	n := copy(p, c.buf)
	c.buf = c.buf[n:]
	return n, nil
}

func (c *streamConn) Write(p []byte) (int, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	written := 0
	for len(p) > 0 {
		select {
		case <-c.closed:
			return written, net.ErrClosed
		case <-c.writeDeadline.wait():
			return written, os.ErrDeadlineExceeded
		default:
		}
		n := min(len(p), maxPacketSize)
		// Send 在返回前完成序列化，调用方随后复用 p 是安全的。
		if err := c.stream.Send(&v1.TunnelPacket{Data: p[:n]}); err != nil {
			return written, err
		}
		written += n
		p = p[n:]
	}
	return written, nil
}

func (c *streamConn) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
		if c.onClose != nil {
			c.onClose()
		}
	})
	return nil
}

func (c *streamConn) LocalAddr() net.Addr  { return c.local }
func (c *streamConn) RemoteAddr() net.Addr { return c.remote }

func (c *streamConn) SetDeadline(t time.Time) error {
	c.readDeadline.set(t)
	c.writeDeadline.set(t)
	return nil
}

func (c *streamConn) SetReadDeadline(t time.Time) error {
	c.readDeadline.set(t)
	return nil
}

// SetWriteDeadline 只在发送下一个包前检查；已进入 Send 的包由流上下文与 Close 结束。
func (c *streamConn) SetWriteDeadline(t time.Time) error {
	c.writeDeadline.set(t)
	return nil
}

// NewClientConn 包装主控侧建立的隧道流；Close 时半关闭发送方向并取消流上下文。
func NewClientConn(stream v1.NodeAgent_TunnelClient, cancel context.CancelFunc, remote string) net.Conn {
	return newStreamConn(stream, Addr("master"), Addr(remote), func() {
		_ = stream.CloseSend()
		cancel()
	})
}

// ServerConn 是节点侧的一条隧道连接，携带建流时完成鉴权的主体信息。
type ServerConn struct {
	*streamConn
	principal any
}

// NewServerConn 包装节点侧收到的隧道流；remote 一般取 gRPC 对端地址，用于审计记录来源 IP。
func NewServerConn(stream v1.NodeAgent_TunnelServer, remote net.Addr, principal any) *ServerConn {
	if remote == nil {
		remote = Addr("master")
	}
	return &ServerConn{
		streamConn: newStreamConn(stream, Addr("node"), remote, nil),
		principal:  principal,
	}
}

// Principal 返回建流时鉴权得到的主体。
func (c *ServerConn) Principal() any {
	return c.principal
}

// Done 在连接关闭后关闭；隧道 handler 据此结束流。
func (c *ServerConn) Done() <-chan struct{} {
	return c.closed
}

// deadline 是可重复设置的超时信号：到期时关闭 wait() 返回的通道，设置为未来时间或零值可重新生效。
type deadline struct {
	mu     sync.Mutex
	timer  *time.Timer
	cancel chan struct{}
}

func newDeadline() deadline {
	return deadline{cancel: make(chan struct{})}
}

func (d *deadline) set(t time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil && !d.timer.Stop() {
		<-d.cancel // 定时器已触发：等它关闭通道后再重建
	}
	d.timer = nil

	expired := isClosed(d.cancel)
	if t.IsZero() {
		if expired {
			d.cancel = make(chan struct{})
		}
		return
	}
	if dur := time.Until(t); dur > 0 {
		if expired {
			d.cancel = make(chan struct{})
		}
		cancel := d.cancel
		d.timer = time.AfterFunc(dur, func() { close(cancel) })
		return
	}
	if !expired {
		close(d.cancel)
	}
}

func (d *deadline) wait() <-chan struct{} {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.cancel
}

func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}
