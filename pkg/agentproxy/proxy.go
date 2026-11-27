// Package agentproxy 是 Agent 用的上游代理（HTTP 或 SOCKS5）：经它连到目标地址、给 Momoko 自己的请求用的 HTTP 客户端，
// 以及一个本地的 HTTP 代理入口（Serve）。
//
// Agent 的进程只看到本地入口（HTTP_PROXY 之类指向它），由 Momoko 把每个连接经上游转出去：
//   - SOCKS5 对所有工具都能用（Claude Code 这类只认 HTTP 代理的也行）；
//   - 目标的域名原样交给上游，由上游解析，本机不查 DNS；
//   - 上游的账号密码不出现在 Agent 的环境变量里。
//
// 在 Linux 上本地入口开在 Agent 自己的网络命名空间里，那是它唯一的出口（见 pkg/agentsandbox）。
package agentproxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	xproxy "golang.org/x/net/proxy"
)

// 代理类型。
const (
	TypeHTTP   = "http"
	TypeSOCKS5 = "socks5"
)

// dialTimeout 连上游、和上游握手最多等多久。
const dialTimeout = 20 * time.Second

// Proxy 一个上游代理。
type Proxy struct {
	Type     string
	Host     string
	Port     int
	Username string
	Password string
}

// Parse 读存起来的代理地址：http://[用户名[:密码]@]主机:端口 或 socks5://…。空字符串返回 nil。
func Parse(raw string) (*Proxy, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, errors.New("代理地址不对")
	}
	p := &Proxy{Type: strings.ToLower(u.Scheme), Host: u.Hostname()}
	if p.Port, err = strconv.Atoi(u.Port()); err != nil {
		return nil, errors.New("代理地址里没有端口")
	}
	if u.User != nil {
		p.Username = u.User.Username()
		p.Password, _ = u.User.Password()
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return p, nil
}

// Validate 类型、主机、端口都要对。
func (p Proxy) Validate() error {
	if p.Type != TypeHTTP && p.Type != TypeSOCKS5 {
		return errors.New("代理类型只能是 HTTP 或 SOCKS5")
	}
	host := strings.TrimSpace(p.Host)
	if host == "" || strings.ContainsAny(host, "/ @?#") {
		return errors.New("先填代理的主机地址")
	}
	if p.Port < 1 || p.Port > 65535 {
		return errors.New("代理端口要在 1 到 65535 之间")
	}
	if p.Password != "" && p.Username == "" {
		return errors.New("填了代理密码就要填用户名")
	}
	return nil
}

// URL 存起来用的地址，带着账号密码。
func (p Proxy) URL() string {
	u := url.URL{Scheme: p.Type, Host: p.Addr()}
	if p.Username != "" {
		if p.Password != "" {
			u.User = url.UserPassword(p.Username, p.Password)
		} else {
			u.User = url.User(p.Username)
		}
	}
	return u.String()
}

// Addr 上游的「主机:端口」。
func (p Proxy) Addr() string {
	return net.JoinHostPort(strings.TrimSpace(p.Host), strconv.Itoa(p.Port))
}

// String 给人看的地址，不带密码。
func (p Proxy) String() string {
	user := ""
	if p.Username != "" {
		user = p.Username + "@"
	}
	return p.Type + "://" + user + p.Addr()
}

// Dial 经上游连到 addr（主机:端口）。域名原样交给上游解析。
func (p Proxy) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	if network != "tcp" && network != "tcp4" && network != "tcp6" {
		return nil, fmt.Errorf("代理只能转发 TCP 连接，不支持 %s", network)
	}
	ctx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()
	var conn net.Conn
	var err error
	if p.Type == TypeSOCKS5 {
		conn, err = p.dialSOCKS5(ctx, addr)
	} else {
		conn, err = p.dialHTTP(ctx, addr)
	}
	if err != nil {
		return nil, p.wrap(ctx, err)
	}
	return conn, nil
}

// wrap 把连不上的原因说成人话。
func (p Proxy) wrap(ctx context.Context, err error) error {
	var reply *replyError
	switch {
	case errors.As(err, &reply):
		return reply
	case ctx.Err() != nil:
		return fmt.Errorf("代理 %s 一直没有响应", p.Addr())
	default:
		return fmt.Errorf("经代理 %s 连不上：%v", p.Addr(), err)
	}
}

// replyError 代理回话了，但不肯转发。
type replyError struct{ msg string }

func (e *replyError) Error() string { return e.msg }

func (p Proxy) dialSOCKS5(ctx context.Context, addr string) (net.Conn, error) {
	var auth *xproxy.Auth
	if p.Username != "" {
		auth = &xproxy.Auth{User: p.Username, Password: p.Password}
	}
	d, err := xproxy.SOCKS5("tcp", p.Addr(), auth, &net.Dialer{Timeout: dialTimeout, KeepAlive: 30 * time.Second})
	if err != nil {
		return nil, err
	}
	cd, ok := d.(xproxy.ContextDialer)
	if !ok {
		return d.Dial("tcp", addr)
	}
	return cd.DialContext(ctx, "tcp", addr)
}

func (p Proxy) dialHTTP(ctx context.Context, addr string) (net.Conn, error) {
	d := net.Dialer{Timeout: dialTimeout, KeepAlive: 30 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", p.Addr())
	if err != nil {
		return nil, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	req := &http.Request{Method: http.MethodConnect, URL: &url.URL{Opaque: addr}, Host: addr, Header: http.Header{}}
	if p.Username != "" {
		req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(p.Username+":"+p.Password)))
	}
	if err := req.Write(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, req)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_ = conn.Close()
		if resp.StatusCode == http.StatusProxyAuthRequired {
			return nil, &replyError{"代理要求登录：用户名或密码不对"}
		}
		return nil, &replyError{fmt.Sprintf("代理拒绝了到 %s 的连接（%s）", addr, resp.Status)}
	}
	_ = conn.SetDeadline(time.Time{})
	// 握手时多读进来的字节（目标先开口的协议）不能丢
	if br.Buffered() > 0 {
		return &bufConn{Conn: conn, r: br}, nil
	}
	return conn, nil
}

// bufConn 先读完缓冲区里剩下的，再读连接。
type bufConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *bufConn) Read(b []byte) (int, error) { return c.r.Read(b) }

// CloseWrite 底下的连接能半关就半关。
func (c *bufConn) CloseWrite() error {
	if cw, ok := c.Conn.(interface{ CloseWrite() error }); ok {
		return cw.CloseWrite()
	}
	return nil
}

// Transport Momoko 自己的请求经这个代理发出去用的传输：所有连接都经上游，不读代理环境变量。
func (p Proxy) Transport() *http.Transport {
	return &http.Transport{
		Proxy:                 nil,
		DialContext:           p.Dial,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   30 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
}

// Client Transport 上的 HTTP 客户端；timeout 为 0 是不限。
func (p Proxy) Client(timeout time.Duration) *http.Client {
	return &http.Client{Transport: p.Transport(), Timeout: timeout}
}

// Check 经代理连到 target（主机:端口）并做一次 TLS 握手，返回用了多久：代理通不通、能不能到那边。
func (p Proxy) Check(ctx context.Context, target string) (time.Duration, error) {
	started := time.Now()
	conn, err := p.Dial(ctx, "tcp", target)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	host, _, _ := net.SplitHostPort(target)
	tc := tls.Client(conn, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	hctx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()
	if err := tc.HandshakeContext(hctx); err != nil {
		return 0, fmt.Errorf("经代理连到了 %s，但 TLS 握手失败：%v", target, err)
	}
	return time.Since(started), nil
}

// LocalHosts 不经代理的地址：本机回环（流量不出这台机器；在隔离的网络里就是 Agent 自己）。
const LocalHosts = "localhost,127.0.0.1,::1"

// Env 让进程经 addr（本地入口，主机:端口）上网的环境变量：常见的代理变量大小写各一份，
// Node 自带的 fetch 也读代理变量，npm 的代理配置也设上。
func Env(addr string) map[string]string {
	u := "http://" + addr
	return map[string]string{
		"HTTP_PROXY": u, "HTTPS_PROXY": u, "ALL_PROXY": u,
		"http_proxy": u, "https_proxy": u, "all_proxy": u,
		"NO_PROXY": LocalHosts, "no_proxy": LocalHosts,
		"NODE_USE_ENV_PROXY":     "1",
		"npm_config_proxy":       u,
		"npm_config_https_proxy": u,
		"npm_config_noproxy":     LocalHosts,
	}
}

// Serve 在 ln 上当本地 HTTP 代理：CONNECT 建隧道，普通请求（绝对地址）转发，连接都经上游出去。ln 关闭后返回。
func Serve(ln net.Listener, p Proxy) {
	h := &handler{p: p, tr: p.Transport()}
	h.rp = &httputil.ReverseProxy{
		Rewrite:   func(pr *httputil.ProxyRequest) {},
		Transport: h.tr,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadGateway)
		},
	}
	srv := &http.Server{Handler: h, ReadHeaderTimeout: 30 * time.Second, ErrorLog: log.New(io.Discard, "", 0)}
	_ = srv.Serve(ln)
	_ = srv.Close()
	h.tr.CloseIdleConnections()
	h.closeTunnels()
}

type handler struct {
	p  Proxy
	tr *http.Transport
	rp *httputil.ReverseProxy

	mu      sync.Mutex
	tunnels map[net.Conn]struct{}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		h.connect(w, r)
		return
	}
	if !r.URL.IsAbs() || r.URL.Host == "" {
		http.Error(w, "这里是代理：请求要带完整地址", http.StatusBadRequest)
		return
	}
	h.rp.ServeHTTP(w, r)
}

func (h *handler) connect(w http.ResponseWriter, r *http.Request) {
	addr := r.Host
	if _, port, err := net.SplitHostPort(addr); err != nil || port == "" {
		http.Error(w, "CONNECT 要带端口", http.StatusBadRequest)
		return
	}
	upstream, err := h.p.Dial(r.Context(), "tcp", addr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		_ = upstream.Close()
		http.Error(w, "不支持隧道", http.StatusInternalServerError)
		return
	}
	client, buf, err := hj.Hijack()
	if err != nil {
		_ = upstream.Close()
		return
	}
	if _, err := client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		_ = client.Close()
		_ = upstream.Close()
		return
	}
	// 客户端可能已经把 TLS 握手跟着 CONNECT 一起发过来了
	if n := buf.Reader.Buffered(); n > 0 {
		pending, _ := buf.Reader.Peek(n)
		if _, err := upstream.Write(pending); err != nil {
			_ = client.Close()
			_ = upstream.Close()
			return
		}
	}
	h.track(client, upstream)
	pipe(client, upstream)
	h.untrack(client, upstream)
}

func (h *handler) track(conns ...net.Conn) {
	h.mu.Lock()
	if h.tunnels == nil {
		h.tunnels = map[net.Conn]struct{}{}
	}
	for _, c := range conns {
		h.tunnels[c] = struct{}{}
	}
	h.mu.Unlock()
}

func (h *handler) untrack(conns ...net.Conn) {
	h.mu.Lock()
	for _, c := range conns {
		delete(h.tunnels, c)
	}
	h.mu.Unlock()
}

// closeTunnels 入口关了（进程退出）：还开着的隧道一起断掉。
func (h *handler) closeTunnels() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.tunnels {
		_ = c.Close()
	}
	h.tunnels = nil
}

// pipe 双向转发：一个方向读完了就把另一边的写端半关（对方知道这边说完了），出错就两边都断；两个方向都结束后关掉连接。
func pipe(a, b net.Conn) {
	var wg sync.WaitGroup
	cp := func(dst, src net.Conn) {
		defer wg.Done()
		if _, err := io.Copy(dst, src); err != nil {
			_ = a.Close()
			_ = b.Close()
			return
		}
		if cw, ok := dst.(interface{ CloseWrite() error }); ok {
			_ = cw.CloseWrite()
		}
	}
	wg.Add(2)
	go cp(a, b)
	go cp(b, a)
	wg.Wait()
	_ = a.Close()
	_ = b.Close()
}
