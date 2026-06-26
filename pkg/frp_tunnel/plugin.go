package frp_tunnel

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	pserver "github.com/fatedier/frp/pkg/plugin/server"
)

// LoginInfo 是 frpc 登录时传给插件的信息。
type LoginInfo struct {
	User          string
	Metas         map[string]string // frpc common metadatas（含 credential）
	RunID         string
	ClientAddress string
}

// NewProxyInfo 是 frpc 声明一个 proxy 时传给插件的信息。
type NewProxyInfo struct {
	User          string
	LoginMetas    map[string]string // 来自登录的 common metadatas（含 credential）
	ProxyMetas    map[string]string // proxy 级 metadatas
	ProxyName     string
	ProxyType     string
	RemotePort    int
	CustomDomains []string
	SubDomain     string

	UseEncryption  bool   // frpc 该 proxy 是否声明 useEncryption
	UseCompression bool   // frpc 该 proxy 是否声明 useCompression
	BandwidthLimit string // frpc 该 proxy 声明的带宽限制（如 "1MB"，空=未限制）
}

// PluginHooks 由业务层提供具体校验逻辑。
// OnLogin/OnNewProxy 返回非 nil error 表示拒绝，error 文本作为拒绝原因回给 frpc。
type PluginHooks struct {
	OnLogin    func(ctx context.Context, info LoginInfo) error
	OnNewProxy func(ctx context.Context, info NewProxyInfo) error
}

// PluginHandler 返回实现 frp 服务端插件协议的 HTTP handler。
//
// frps 以 POST 调用 `?op=Login|NewProxy`，请求体为 {version, op, content}，
// content 为对应操作的内容；handler 回 {reject, reject_reason, unchange}。
// 注：path token 的校验由注册时的固定路径承担（仅匹配该路径的请求才会到达此 handler）。
func PluginHandler(hooks PluginHooks) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := r.URL.Query().Get("op")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeResponse(w, reject("read body: "+err.Error()))
			return
		}

		var req struct {
			Op      string          `json:"op"`
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			writeResponse(w, reject("decode request: "+err.Error()))
			return
		}
		if op == "" {
			op = req.Op
		}

		switch op {
		case pserver.OpLogin:
			writeResponse(w, handleLogin(r.Context(), hooks, req.Content))
		case pserver.OpNewProxy:
			writeResponse(w, handleNewProxy(r.Context(), hooks, req.Content))
		default:
			// 未关注的操作一律放行（不改动内容）。
			writeResponse(w, pserver.Response{Unchange: true})
		}
	}
}

func handleLogin(ctx context.Context, hooks PluginHooks, content json.RawMessage) pserver.Response {
	var c pserver.LoginContent
	if err := json.Unmarshal(content, &c); err != nil {
		return reject("decode login content: " + err.Error())
	}
	if hooks.OnLogin != nil {
		if err := hooks.OnLogin(ctx, LoginInfo{
			User:          c.User,
			Metas:         c.Metas,
			RunID:         c.RunID,
			ClientAddress: c.ClientAddress,
		}); err != nil {
			return reject(err.Error())
		}
	}
	return pserver.Response{Unchange: true}
}

func handleNewProxy(ctx context.Context, hooks PluginHooks, content json.RawMessage) pserver.Response {
	var c pserver.NewProxyContent
	if err := json.Unmarshal(content, &c); err != nil {
		return reject("decode newproxy content: " + err.Error())
	}
	if hooks.OnNewProxy != nil {
		if err := hooks.OnNewProxy(ctx, NewProxyInfo{
			User:           c.User.User,
			LoginMetas:     c.User.Metas,
			ProxyMetas:     c.Metas,
			ProxyName:      c.ProxyName,
			ProxyType:      c.ProxyType,
			RemotePort:     c.RemotePort,
			CustomDomains:  c.CustomDomains,
			SubDomain:      c.SubDomain,
			UseEncryption:  c.UseEncryption,
			UseCompression: c.UseCompression,
			BandwidthLimit: c.BandwidthLimit,
		}); err != nil {
			return reject(err.Error())
		}
	}
	return pserver.Response{Unchange: true}
}

func reject(reason string) pserver.Response {
	return pserver.Response{Reject: true, RejectReason: reason}
}

// writeResponse 始终以 200 写出 frp 插件响应（拒绝信息位于响应体）。
func writeResponse(w http.ResponseWriter, resp pserver.Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
