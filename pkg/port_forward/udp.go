package port_forward

import (
	"context"
	"net"
	"strconv"
	"sync"
	"time"
)

const (
	// udpSessionTimeout 单个 UDP 会话在目标无响应后的空闲超时。
	udpSessionTimeout = 2 * time.Minute
)

// UDPPortForward udp 端口转发。
// 以客户端地址为键维护会话，每个会话单独持有一条到目标的 UDP 连接。
type UDPPortForward struct {
	option     *Option
	listenConn *net.UDPConn
	targetAddr *net.UDPAddr
	ctx        context.Context
	cancel     context.CancelFunc
	stats      *Stats

	mu       sync.Mutex
	sessions map[string]*udpSession
}

type udpSession struct {
	targetConn *net.UDPConn
}

func NewUDPPortForward(option *Option, mctx context.Context) (*UDPPortForward, error) {
	listenAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(option.ListenAddress, strconv.Itoa(option.ListenPort)))
	if err != nil {
		return nil, NETListenError
	}
	targetAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(option.TargetAddress, strconv.Itoa(option.TargetPort)))
	if err != nil {
		return nil, NETListenError
	}
	listenConn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		return nil, NETListenError
	}
	ctx, cancel := context.WithCancel(mctx)
	return &UDPPortForward{
		option:     option,
		listenConn: listenConn,
		targetAddr: targetAddr,
		ctx:        ctx,
		cancel:     cancel,
		stats:      NewStats(),
		sessions:   make(map[string]*udpSession),
	}, nil
}

func (u *UDPPortForward) Option() *Option { return u.option }

func (u *UDPPortForward) Stats() *Stats { return u.stats }

func (u *UDPPortForward) Start() error {
	go func() {
		<-u.ctx.Done()
		_ = u.listenConn.Close()
	}()

	// 监听循环单缓冲，监听器生命周期内复用，无需入池。
	buf := make([]byte, relayBufferSize)
	for {
		n, clientAddr, err := u.listenConn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-u.ctx.Done():
				return nil
			default:
				return err
			}
		}

		session, err := u.session(clientAddr)
		if err != nil {
			continue
		}
		if nw, werr := session.targetConn.Write(buf[:n]); werr == nil {
			u.stats.addIn(int64(nw))
		}
	}
}

func (u *UDPPortForward) Stop() {
	u.cancel()
	_ = u.listenConn.Close()

	u.mu.Lock()
	for _, session := range u.sessions {
		_ = session.targetConn.Close()
	}
	u.sessions = make(map[string]*udpSession)
	u.mu.Unlock()
}

// session 返回客户端对应的会话，不存在则新建并启动回程读取协程。
func (u *UDPPortForward) session(clientAddr *net.UDPAddr) (*udpSession, error) {
	key := clientAddr.String()

	u.mu.Lock()
	if session := u.sessions[key]; session != nil {
		u.mu.Unlock()
		return session, nil
	}
	u.mu.Unlock()

	targetConn, err := net.DialUDP("udp", nil, u.targetAddr)
	if err != nil {
		return nil, err
	}
	session := &udpSession{targetConn: targetConn}

	u.mu.Lock()
	u.sessions[key] = session
	u.mu.Unlock()

	u.stats.connOpened()
	go u.readTarget(clientAddr, session)
	return session, nil
}

// readTarget 读取目标的回程数据并写回客户端，目标空闲超时后结束会话。
func (u *UDPPortForward) readTarget(clientAddr *net.UDPAddr, session *udpSession) {
	defer func() {
		_ = session.targetConn.Close()
		u.deleteSession(clientAddr.String(), session)
		u.stats.connClosed()
	}()

	buf := getRelayBuf()
	defer putRelayBuf(buf)
	for {
		_ = session.targetConn.SetReadDeadline(time.Now().Add(udpSessionTimeout))
		n, err := session.targetConn.Read(buf)
		if err != nil {
			return
		}
		if nw, werr := u.listenConn.WriteToUDP(buf[:n], clientAddr); werr == nil {
			u.stats.addOut(int64(nw))
		}
	}
}

func (u *UDPPortForward) deleteSession(key string, session *udpSession) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.sessions[key] == session {
		delete(u.sessions, key)
	}
}
