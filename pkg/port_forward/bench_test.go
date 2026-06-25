package port_forward

import (
	"context"
	"io"
	"net"
	"strconv"
	"testing"
)

// 性能基准。运行：
//
//	go test ./pkg/port_forward/ -bench=. -benchmem -run=^$
//
// -benchmem 额外打印 B/op 与 allocs/op；-run=^$ 跳过功能测试只跑基准。
//
// 三个基准：
//  1. BenchmarkTCPPortForwardThroughput —— 本项目 TCPPortForward 单向（客户端→目标）最大吞吐
//  2. BenchmarkTCPIOCopyBaseline        —— 同拓扑但用裸 io.Copy 中继，作吞吐上界参照
//     （Linux 上两端均为 *net.TCPConn 时 io.Copy 走 splice，用户态拷贝无法企及）
//  3. BenchmarkTCPPortForwardEcho       —— 双向（回显）吞吐，两个方向各计 benchPayloadSize

// benchPayloadSize 每轮迭代单向搬运的数据量。
// 64MiB 足以摊薄建连/拆除开销，又不至于让单轮在慢机器上耗时过长。
const benchPayloadSize = 64 * 1024 * 1024

// freePortTB 申请一个空闲 TCP 端口（testing.TB 版，兼容基准测试）。
func freePortTB(t testing.TB) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// hostPortOf 把 "host:port" 拆成 host 与 int port。
func hostPortOf(addr string) (string, int) {
	h, p, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(p)
	return h, port
}

// sinkTarget 启动一个把入站数据全部丢弃的目标。每条连接收到 EOF 后回写一个哨兵字节，
// 供客户端确认目标侧已接收完毕，避免内核 socket 缓冲把吞吐测虚高。
func sinkTarget(t testing.TB) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("sink listen: %v", err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				_, _ = io.Copy(io.Discard, c)
				_, _ = c.Write([]byte{0}) // 哨兵：目标侧接收完毕
				_ = c.Close()
			}(c)
		}
	}()
	return ln.Addr().String(), func() { _ = ln.Close() }
}

// echoTarget 启动一个回显目标，原样返回入站数据，用于双向吞吐基准。
func echoTarget(t testing.TB) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("echo listen: %v", err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				_, _ = io.Copy(c, c)
				_ = c.Close()
			}(c)
		}
	}()
	return ln.Addr().String(), func() { _ = ln.Close() }
}

// ioCopyRelay 启动一个用裸 io.Copy 中继的转发器，作为 stdlib 基线。
// 与本项目的 pipe 一样在单向结束后半关闭对端，使 sink 能看到 EOF。
func ioCopyRelay(t testing.TB, targetAddr string) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("relay listen: %v", err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				tgt, err := net.Dial("tcp", targetAddr)
				if err != nil {
					_ = c.Close()
					return
				}
				go func() {
					_, _ = io.Copy(tgt, c) // 客户端→目标
					if tc, ok := tgt.(*net.TCPConn); ok {
						_ = tc.CloseWrite() // 通知目标：本方向数据结束
					}
				}()
				_, _ = io.Copy(c, tgt) // 目标→客户端
				_ = c.Close()
				_ = tgt.Close()
			}(c)
		}
	}()
	return ln.Addr().String(), func() { _ = ln.Close() }
}

// pumpOnce 经 addr 向中继推送 benchPayloadSize 字节，半关闭写方向，
// 并等待目标侧哨兵确认接收完毕。返回前连接已关闭。
func pumpOnce(b *testing.B, addr string, payload []byte) {
	b.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		b.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()
	var remaining int64 = benchPayloadSize
	for remaining > 0 {
		n, err := conn.Write(payload)
		if err != nil {
			b.Fatalf("write: %v", err)
		}
		remaining -= int64(n)
	}
	if tc, ok := conn.(*net.TCPConn); ok {
		_ = tc.CloseWrite()
	}
	var sentinel [1]byte
	if _, err := io.ReadFull(conn, sentinel[:]); err != nil {
		b.Fatalf("read sentinel: %v", err)
	}
}

// BenchmarkTCPPortForwardThroughput 测量本项目 TCPPortForward 单向最大吞吐。
// b.SetBytes 让 testing 自动报告 MB/s；ReportAllocs 报告每轮分配。
// 附加指标 B_in/iter 应≈benchPayloadSize、B_out/iter 应≈1（仅哨兵），验证满载下逐包计数仍准确。
func BenchmarkTCPPortForwardThroughput(b *testing.B) {
	sinkAddr, stopSink := sinkTarget(b)
	defer stopSink()

	host, port := hostPortOf(sinkAddr)
	listenPort := freePortTB(b)
	fwd, err := NewTCPPortForward(&Option{
		ID:            "bench-tcp",
		Protocol:      TCP,
		ListenAddress: "127.0.0.1",
		ListenPort:    listenPort,
		TargetAddress: host,
		TargetPort:    port,
	}, context.Background())
	if err != nil {
		b.Fatalf("new fwd: %v", err)
	}
	go func() { _ = fwd.Start() }()
	defer fwd.Stop()

	relayAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(listenPort))
	payload := make([]byte, relayBufferSize) // 复用同一写缓冲，长度匹配中继读缓冲

	b.SetBytes(benchPayloadSize)
	b.ReportAllocs()
	for b.Loop() {
		pumpOnce(b, relayAddr, payload)
	}
	s := fwd.Stats().Snapshot()
	b.ReportMetric(float64(s.BytesIn)/float64(b.N), "B_in/iter")
	b.ReportMetric(float64(s.BytesOut)/float64(b.N), "B_out/iter")
}

// BenchmarkTCPIOCopyBaseline 同拓扑但用裸 io.Copy 中继，作为吞吐上界参照。
func BenchmarkTCPIOCopyBaseline(b *testing.B) {
	sinkAddr, stopSink := sinkTarget(b)
	defer stopSink()
	relayAddr, stopRelay := ioCopyRelay(b, sinkAddr)
	defer stopRelay()

	payload := make([]byte, relayBufferSize)
	b.SetBytes(benchPayloadSize)
	b.ReportAllocs()
	for b.Loop() {
		pumpOnce(b, relayAddr, payload)
	}
}

// BenchmarkTCPPortForwardEcho 测量双向（回显）吞吐：客户端写 benchPayloadSize 并同时读回等量数据。
// b.SetBytes 计入两个方向共 2*benchPayloadSize。
func BenchmarkTCPPortForwardEcho(b *testing.B) {
	echoAddr, stopEcho := echoTarget(b)
	defer stopEcho()

	host, port := hostPortOf(echoAddr)
	listenPort := freePortTB(b)
	fwd, err := NewTCPPortForward(&Option{
		ID:            "bench-tcp-echo",
		Protocol:      TCP,
		ListenAddress: "127.0.0.1",
		ListenPort:    listenPort,
		TargetAddress: host,
		TargetPort:    port,
	}, context.Background())
	if err != nil {
		b.Fatalf("new fwd: %v", err)
	}
	go func() { _ = fwd.Start() }()
	defer fwd.Stop()

	relayAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(listenPort))
	payload := make([]byte, relayBufferSize)
	rbuf := make([]byte, relayBufferSize)

	b.SetBytes(2 * benchPayloadSize)
	b.ReportAllocs()
	for b.Loop() {
		conn, err := net.Dial("tcp", relayAddr)
		if err != nil {
			b.Fatalf("dial: %v", err)
		}
		writeErr := make(chan error, 1)
		go func() {
			var remaining int64 = benchPayloadSize
			for remaining > 0 {
				n, err := conn.Write(payload)
				if err != nil {
					writeErr <- err
					return
				}
				remaining -= int64(n)
			}
			writeErr <- nil
		}()
		var got int64
		for got < benchPayloadSize {
			n, err := conn.Read(rbuf)
			if err != nil {
				b.Fatalf("read: %v", err)
			}
			got += int64(n)
		}
		if err := <-writeErr; err != nil {
			b.Fatalf("write: %v", err)
		}
		_ = conn.Close()
	}
	s := fwd.Stats().Snapshot()
	b.ReportMetric(float64(s.BytesIn)/float64(b.N), "B_in/iter")
	b.ReportMetric(float64(s.BytesOut)/float64(b.N), "B_out/iter")
}
