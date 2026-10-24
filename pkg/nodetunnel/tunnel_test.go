package nodetunnel

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	v1 "momoko/api/gen/v1"
)

type tunnelAgent struct {
	v1.UnimplementedNodeAgentServer
	listener *Listener
}

func (a *tunnelAgent) Tunnel(stream v1.NodeAgent_TunnelServer) error {
	conn := NewServerConn(stream, nil, "principal")
	if err := a.listener.Push(stream.Context(), conn); err != nil {
		return err
	}
	select {
	case <-conn.Done():
	case <-stream.Context().Done():
		_ = conn.Close()
	}
	return nil
}

// startTunnel 搭建“节点”：gRPC NodeAgent.Tunnel → Listener → http.Server；返回指向它的反向代理（模拟主控）。
func startTunnel(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	grpcServer := grpc.NewServer()
	agent := &tunnelAgent{listener: NewListener()}
	v1.RegisterNodeAgentServer(grpcServer, agent)
	go func() { _ = grpcServer.Serve(lis) }()

	var gotPrincipal any
	httpServer := &http.Server{
		Handler: handler,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			if sc, ok := c.(*ServerConn); ok {
				gotPrincipal = sc.Principal()
			}
			return ctx
		},
	}
	go func() { _ = httpServer.Serve(agent.listener) }()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	client := v1.NewNodeAgentClient(conn)
	transport := &http.Transport{
		DialContext: func(context.Context, string, string) (net.Conn, error) {
			ctx, cancel := context.WithCancel(context.Background())
			stream, err := client.Tunnel(ctx)
			if err != nil {
				cancel()
				return nil, err
			}
			return NewClientConn(stream, cancel, "node"), nil
		},
		DisableKeepAlives: true,
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL.Scheme = "http"
			pr.Out.URL.Host = "momoko-node"
		},
		Transport:     transport,
		FlushInterval: -1,
	}
	master := httptest.NewServer(proxy)
	t.Cleanup(func() {
		master.Close()
		_ = httpServer.Close()
		grpcServer.Stop()
		_ = conn.Close()
		if gotPrincipal != nil && gotPrincipal != "principal" {
			t.Errorf("principal not propagated: %v", gotPrincipal)
		}
	})
	return master
}

func TestTunnelHTTPRoundTrip(t *testing.T) {
	large := bytes.Repeat([]byte("momoko-"), 100_000) // 700KB，跨越多个隧道包
	mux := http.NewServeMux()
	mux.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(large)
	})
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_, _ = io.WriteString(w, r.Method+":"+r.URL.Query().Get("sign")+":"+string(rune('0'+len(body)%10)))
		if !bytes.Equal(body, large) {
			t.Errorf("upload body mismatch: %d bytes", len(body))
		}
	})
	master := startTunnel(t, mux)

	res, err := http.Get(master.URL + "/download")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if !bytes.Equal(body, large) {
		t.Fatalf("download mismatch: got %d bytes", len(body))
	}

	req, _ := http.NewRequest(http.MethodPut, master.URL+"/upload?sign=abc", bytes.NewReader(large))
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(res.Body)
	_ = res.Body.Close()
	if !strings.HasPrefix(string(body), "PUT:abc:") {
		t.Fatalf("unexpected upload reply %q", body)
	}
}

func TestTunnelWebSocket(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/ws", websocket.Handler(func(ws *websocket.Conn) {
		for {
			var msg string
			if err := websocket.Message.Receive(ws, &msg); err != nil {
				return
			}
			if err := websocket.Message.Send(ws, "echo:"+msg); err != nil {
				return
			}
		}
	}))
	master := startTunnel(t, mux)

	wsURL := "ws" + strings.TrimPrefix(master.URL, "http") + "/ws"
	ws, err := websocket.Dial(wsURL, "", master.URL)
	if err != nil {
		t.Fatalf("dial websocket through tunnel: %v", err)
	}
	defer ws.Close()
	_ = ws.SetDeadline(time.Now().Add(5 * time.Second))
	for _, msg := range []string{"hello", "节点", strings.Repeat("x", 100_000)} {
		if err := websocket.Message.Send(ws, msg); err != nil {
			t.Fatal(err)
		}
		var reply string
		if err := websocket.Message.Receive(ws, &reply); err != nil {
			t.Fatal(err)
		}
		if reply != "echo:"+msg {
			t.Fatalf("unexpected reply (len %d)", len(reply))
		}
	}
}
