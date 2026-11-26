// Package netprobe 做简单的网络探测：TCP 连通与延迟、TCP / UDP 监听地址是否空闲、进程是否绑定了 UDP 端口、HTTP GET 读取 JSON。
package netprobe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	gnet "github.com/shirou/gopsutil/v4/net"
)

// DialAddr 把监听地址转成可连接的地址：0.0.0.0 / :: / 空主机改为本机回环。
func DialAddr(listen string) string {
	host, port, err := net.SplitHostPort(strings.TrimSpace(listen))
	if err != nil {
		return listen
	}
	switch host {
	case "", "0.0.0.0":
		host = "127.0.0.1"
	case "::", "[::]":
		host = "::1"
	}
	return net.JoinHostPort(host, port)
}

// TCP 连接一次，返回耗时。
func TCP(ctx context.Context, addr string, timeout time.Duration) (time.Duration, error) {
	d := net.Dialer{Timeout: timeout}
	start := time.Now()
	conn, err := d.DialContext(ctx, "tcp", DialAddr(addr))
	if err != nil {
		return 0, err
	}
	_ = conn.Close()
	return time.Since(start), nil
}

// ErrInUse 监听地址已被占用。
var ErrInUse = errors.New("netprobe: address already in use")

// Free 判断监听地址当前是否可以绑定（先尝试连接，能连上即占用；再尝试监听）。
func Free(listen string) error {
	if _, _, err := net.SplitHostPort(listen); err != nil {
		return err
	}
	if _, err := TCP(context.Background(), listen, 300*time.Millisecond); err == nil {
		return ErrInUse
	}
	l, err := net.Listen("tcp", listen)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInUse, err)
	}
	return l.Close()
}

// FreeUDP 判断 UDP 监听地址当前是否可以绑定（绑定不了即占用）。
func FreeUDP(listen string) error {
	if _, _, err := net.SplitHostPort(listen); err != nil {
		return err
	}
	c, err := net.ListenPacket("udp", listen)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInUse, err)
	}
	return c.Close()
}

// UDPBound 判断这些进程里是否有一个绑定了这个 UDP 监听地址（读系统的套接字表，不占用端口）。
func UDPBound(ctx context.Context, listen string, pids []int32) (bool, error) {
	host, port, err := net.SplitHostPort(strings.TrimSpace(listen))
	if err != nil {
		return false, err
	}
	p, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return false, err
	}
	want := net.ParseIP(host)
	for _, pid := range pids {
		conns, err := gnet.ConnectionsPidWithContext(ctx, "udp", pid)
		if err != nil {
			continue
		}
		for _, c := range conns {
			if c.Laddr.Port != uint32(p) {
				continue
			}
			got := net.ParseIP(c.Laddr.IP)
			if want == nil || want.IsUnspecified() || got == nil || got.IsUnspecified() || got.Equal(want) {
				return true, nil
			}
		}
	}
	return false, nil
}

// Port 返回监听地址里的端口。
func Port(listen string) string {
	_, port, err := net.SplitHostPort(listen)
	if err != nil {
		return ""
	}
	return port
}

// GetJSON 发起 GET 请求并把 JSON 响应解析为任意值，返回耗时。
func GetJSON(ctx context.Context, url string, timeout time.Duration) (any, time.Duration, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	start := time.Now()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	elapsed := time.Since(start)
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, elapsed, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, elapsed, fmt.Errorf("HTTP %s", res.Status)
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, elapsed, err
	}
	return out, elapsed, nil
}
