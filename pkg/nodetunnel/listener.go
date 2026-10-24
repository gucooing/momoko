package nodetunnel

import (
	"context"
	"net"
	"sync"
)

// Listener 把隧道连接投递给进程内的 http.Server。
type Listener struct {
	conns chan net.Conn
	done  chan struct{}
	once  sync.Once
}

func NewListener() *Listener {
	return &Listener{
		conns: make(chan net.Conn),
		done:  make(chan struct{}),
	}
}

func (l *Listener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.conns:
		return conn, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *Listener) Close() error {
	l.once.Do(func() { close(l.done) })
	return nil
}

func (l *Listener) Addr() net.Addr {
	return Addr("node-tunnel")
}

// Push 投递一条连接，阻塞到被 Accept、监听器关闭或 ctx 结束。
func (l *Listener) Push(ctx context.Context, conn net.Conn) error {
	select {
	case l.conns <- conn:
		return nil
	case <-l.done:
		return net.ErrClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}
