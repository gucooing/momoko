package port_forward

import (
	"context"
	"io"
	"net"
	"strconv"
	"testing"
	"time"
)

func freeTCPPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free tcp port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func freeUDPPort(t *testing.T) int {
	t.Helper()
	c, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("find free udp port: %v", err)
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).Port
}

func waitStats(fwd PortForward, ok func(Snapshot) bool) Snapshot {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s := fwd.Stats().Snapshot(); ok(s) {
			return s
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fwd.Stats().Snapshot()
}

func TestTCPPortForwardEchoAndStats(t *testing.T) {
	// 目标：TCP 回显服务器。
	target, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen target: %v", err)
	}
	defer target.Close()
	go func() {
		for {
			conn, err := target.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_, _ = io.Copy(c, c)
			}(conn)
		}
	}()

	listenPort := freeTCPPort(t)
	fwd, err := NewTCPPortForward(&Option{
		ID:            "test-tcp",
		Protocol:      TCP,
		ListenAddress: "127.0.0.1",
		ListenPort:    listenPort,
		TargetAddress: "127.0.0.1",
		TargetPort:    target.Addr().(*net.TCPAddr).Port,
	}, context.Background())
	if err != nil {
		t.Fatalf("new tcp forward: %v", err)
	}
	go func() { _ = fwd.Start() }()
	defer fwd.Stop()

	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(listenPort))
	var conn net.Conn
	for i := 0; i < 50; i++ {
		if conn, err = net.Dial("tcp", addr); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("dial forward: %v", err)
	}
	defer conn.Close()

	payload := []byte("hello momoko")
	if _, err := conn.Write(payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	buf := make([]byte, len(payload))
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if string(buf) != string(payload) {
		t.Fatalf("echo mismatch: got %q want %q", buf, payload)
	}

	s := waitStats(fwd, func(s Snapshot) bool {
		return s.ActiveConns == 1 && s.BytesIn >= int64(len(payload)) && s.BytesOut >= int64(len(payload))
	})
	if s.TotalConns != 1 {
		t.Fatalf("TotalConns = %d, want 1", s.TotalConns)
	}
	if s.ActiveConns != 1 {
		t.Fatalf("ActiveConns = %d, want 1", s.ActiveConns)
	}
	if s.BytesIn < int64(len(payload)) || s.BytesOut < int64(len(payload)) {
		t.Fatalf("bytes in/out = %d/%d, want >= %d", s.BytesIn, s.BytesOut, len(payload))
	}

	// 客户端关闭后活跃连接应归零（验证连接自然结束时不会泄漏）。
	_ = conn.Close()
	if got := waitStats(fwd, func(s Snapshot) bool { return s.ActiveConns == 0 }); got.ActiveConns != 0 {
		t.Fatalf("ActiveConns after close = %d, want 0", got.ActiveConns)
	}
}

func TestUDPPortForwardEchoAndStats(t *testing.T) {
	// 目标：UDP 回显服务器。
	target, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("listen udp target: %v", err)
	}
	defer target.Close()
	go func() {
		buf := make([]byte, 2048)
		for {
			n, addr, err := target.ReadFromUDP(buf)
			if err != nil {
				return
			}
			_, _ = target.WriteToUDP(buf[:n], addr)
		}
	}()

	listenPort := freeUDPPort(t)
	fwd, err := NewUDPPortForward(&Option{
		ID:            "test-udp",
		Protocol:      UDP,
		ListenAddress: "127.0.0.1",
		ListenPort:    listenPort,
		TargetAddress: "127.0.0.1",
		TargetPort:    target.LocalAddr().(*net.UDPAddr).Port,
	}, context.Background())
	if err != nil {
		t.Fatalf("new udp forward: %v", err)
	}
	go func() { _ = fwd.Start() }()
	defer fwd.Stop()

	serverAddr := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: listenPort}
	client, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("dial udp forward: %v", err)
	}
	defer client.Close()

	payload := []byte("hello udp")
	if _, err := client.Write(payload); err != nil {
		t.Fatalf("write udp: %v", err)
	}
	_ = client.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, len(payload))
	if _, err := io.ReadFull(client, buf); err != nil {
		t.Fatalf("read udp echo: %v", err)
	}
	if string(buf) != string(payload) {
		t.Fatalf("udp echo mismatch: got %q want %q", buf, payload)
	}

	s := waitStats(fwd, func(s Snapshot) bool {
		return s.TotalConns == 1 && s.BytesIn >= int64(len(payload)) && s.BytesOut >= int64(len(payload))
	})
	if s.TotalConns != 1 {
		t.Fatalf("TotalConns = %d, want 1", s.TotalConns)
	}
	if s.BytesIn < int64(len(payload)) || s.BytesOut < int64(len(payload)) {
		t.Fatalf("udp bytes in/out = %d/%d, want >= %d", s.BytesIn, s.BytesOut, len(payload))
	}
}
