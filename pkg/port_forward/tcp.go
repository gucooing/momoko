package port_forward

import (
	"context"
	"net"
	"strconv"
	"sync"
)

// TCPPortForward tcp 端口转发。
type TCPPortForward struct {
	option   *Option
	listener net.Listener
	ctx      context.Context
	cancel   context.CancelFunc
	stats    *Stats
}

func NewTCPPortForward(option *Option, mctx context.Context) (*TCPPortForward, error) {
	listener, err := net.Listen("tcp", net.JoinHostPort(option.ListenAddress, strconv.Itoa(option.ListenPort)))
	if err != nil {
		return nil, NETListenError
	}
	ctx, cancel := context.WithCancel(mctx)
	return &TCPPortForward{
		option:   option,
		listener: listener,
		ctx:      ctx,
		cancel:   cancel,
		stats:    NewStats(),
	}, nil
}

func (t *TCPPortForward) Option() *Option { return t.option }

func (t *TCPPortForward) Stats() *Stats { return t.stats }

func (t *TCPPortForward) Start() error {
	// ctx 取消时关闭监听，解除 Accept 阻塞。
	go func() {
		<-t.ctx.Done()
		_ = t.listener.Close()
	}()

	for {
		conn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.ctx.Done():
				return nil
			default:
				return err
			}
		}
		go t.handleConn(conn)
	}
}

func (t *TCPPortForward) Stop() {
	t.cancel()
	_ = t.listener.Close()
}

// handleConn 处理单条客户端连接：拨号目标并双向转发，期间统计连接数与流量。
func (t *TCPPortForward) handleConn(client net.Conn) {
	defer client.Close()

	target, err := net.Dial("tcp", net.JoinHostPort(t.option.TargetAddress, strconv.Itoa(t.option.TargetPort)))
	if err != nil {
		return
	}
	defer target.Close()

	t.stats.connOpened()
	defer t.stats.connClosed()

	// ctx 取消时强制关闭两端，解除 io 阻塞；连接自然结束时由 done 关闭该协程。
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-t.ctx.Done():
			_ = client.Close()
			_ = target.Close()
		case <-done:
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)
	// 客户端 → 目标：入站流量
	go func() {
		defer wg.Done()
		pipe(target, client, t.stats.addIn)
	}()
	// 目标 → 客户端：出站流量
	go func() {
		defer wg.Done()
		pipe(client, target, t.stats.addOut)
	}()
	wg.Wait()
}

// pipe 将 src 的数据搬运到 dst，按实际写入字节数实时累计，并在结束时半关闭两端。
// 缓冲取自 bufPool，函数返回时归还，避免每条连接各分配一份 64KB。
func pipe(dst, src net.Conn, count func(int64)) {
	buf := getRelayBuf()
	defer putRelayBuf(buf)
	for {
		nr, rerr := src.Read(buf)
		if nr > 0 {
			nw, werr := dst.Write(buf[:nr])
			if nw > 0 {
				count(int64(nw))
			}
			if werr != nil {
				break
			}
		}
		if rerr != nil {
			break
		}
	}
	closeWrite(dst)
	closeRead(src)
}

// closeRead 半关闭读方向（如支持），尽快通知对端数据已读完。
func closeRead(conn net.Conn) {
	if c, ok := conn.(*net.TCPConn); ok {
		_ = c.CloseRead()
	}
}

// closeWrite 半关闭写方向（如支持），向对端发送 FIN。
func closeWrite(conn net.Conn) {
	if c, ok := conn.(*net.TCPConn); ok {
		_ = c.CloseWrite()
	}
}
