package port_forward

import (
	"context"
	"io"
	"net"
	"strconv"
	"sync"
)

// TCPPortForward tcp端口转发
type TCPPortForward struct {
	option   *Option
	listener net.Listener
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewTCPPortForward(option *Option, mctx context.Context) (*TCPPortForward, error) {
	listener, err := net.Listen("tcp", net.JoinHostPort(option.ListenAddress, strconv.Itoa(option.ListenPort)))
	if err != nil {
		return nil, NETListenError
	}
	ctx, cancel := context.WithCancel(mctx)
	t := &TCPPortForward{
		listener: listener,
		option:   option,
		ctx:      ctx,
		cancel:   cancel,
	}

	return t, nil
}

func (t *TCPPortForward) Option() *Option {
	return t.option
}

func (t *TCPPortForward) Start() error {
	defer t.listener.Close()
	for {
		select {
		case <-t.ctx.Done():
			return nil
		default:
			conn, err := t.listener.Accept()
			if err != nil {
				return err
			}
			go t.forwardTCPConn(conn)
		}
	}
}

func (t *TCPPortForward) Stop() {
	t.cancel()
}

func (t *TCPPortForward) forwardTCPConn(client net.Conn) {
	defer client.Close()
	target, err := net.Dial("tcp", net.JoinHostPort(t.option.TargetAddress, strconv.Itoa(t.option.TargetPort)))
	if err != nil {
		// TODO
		return
	}
	defer target.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go copyTCP(&wg, target, client)
	go copyTCP(&wg, client, target)

	wg.Wait()
}

func copyTCP(wg *sync.WaitGroup, dst net.Conn, src net.Conn) {
	defer wg.Done()

	_, _ = io.Copy(dst, src)
}
