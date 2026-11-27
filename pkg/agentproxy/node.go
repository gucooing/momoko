package agentproxy

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// Protocols 能用的协议（sing-box 的 type）。直连、拦截、选择器这类不是代理的出站不行；
// http 不行；shadowtls 要跟着另一个出站、tor 要另装 tor，一个节点表达不了，也不行。
var Protocols = []string{"vless", "vmess", "trojan", "shadowsocks", "hysteria2", "hysteria", "tuic", "anytls", "wireguard", "ssh", "socks"}

// parseNode 把节点换成 sing-box 的出站（wireguard 是端点；都不带 tag）：JSON（以 { 开头）原样用；分享链接按协议拆。
func parseNode(node string) (map[string]any, error) {
	if strings.HasPrefix(node, "{") {
		return parseJSON(node)
	}
	scheme, _, ok := strings.Cut(node, "://")
	if !ok {
		return nil, errors.New("认不出这个节点：要填分享链接（vless://、vmess://、trojan://、ss://、hysteria2://……）或 sing-box 的出站 JSON")
	}
	switch strings.ToLower(scheme) {
	case "vless":
		return parseVLESS(node)
	case "vmess":
		return parseVMess(node)
	case "trojan":
		return parseTrojan(node)
	case "ss":
		return parseShadowsocks(node)
	case "hysteria2", "hy2":
		return parseHysteria2(node)
	case "hysteria":
		return parseHysteria(node)
	case "tuic":
		return parseTUIC(node)
	case "anytls":
		return parseAnyTLS(node)
	case "wireguard", "wg":
		return parseWireGuard(node)
	case "socks", "socks5", "socks5h":
		return parseSocks(node)
	}
	return nil, fmt.Errorf("不支持 %s:// 的节点", scheme)
}

// parseJSON sing-box 的出站（或 wireguard 端点）JSON：协议要在 Protocols 里；不能有 detour（只有这一个出站）。
func parseJSON(node string) (map[string]any, error) {
	var out map[string]any
	if err := json.Unmarshal([]byte(node), &out); err != nil {
		return nil, errors.New("节点的 JSON 写得不对")
	}
	kind, _ := out["type"].(string)
	if kind == "" {
		return nil, errors.New("节点的 JSON 里没有 type")
	}
	if !supported(kind) {
		return nil, fmt.Errorf("不支持 %s 类型的节点", kind)
	}
	if _, ok := out["detour"]; ok {
		return nil, errors.New("节点的 JSON 里不能有 detour")
	}
	delete(out, "tag")
	return out, nil
}

func supported(kind string) bool {
	for _, p := range Protocols {
		if p == kind {
			return true
		}
	}
	return false
}

// endpoint 链接里的主机和端口。
func endpoint(u *url.URL) (string, int, error) {
	host := u.Hostname()
	port, err := strconv.Atoi(u.Port())
	if host == "" || err != nil || port < 1 || port > 65535 {
		return "", 0, errors.New("节点里没有主机或端口")
	}
	return host, port, nil
}

// base 出站的公共部分。
func base(kind, host string, port int) map[string]any {
	return map[string]any{"type": kind, "server": host, "server_port": port}
}

// tlsOf 按链接参数拼 TLS：security=tls / reality（或 enabled 为 true 时默认开），sni、fp、alpn、allowInsecure、pbk、sid。
func tlsOf(q url.Values, enabled bool, defaultSNI string) map[string]any {
	security := strings.ToLower(q.Get("security"))
	if !enabled && security != "tls" && security != "reality" && security != "xtls" {
		return nil
	}
	t := map[string]any{"enabled": true}
	sni := first(q, "sni", "peer", "servername")
	if sni == "" {
		sni = defaultSNI
	}
	if sni != "" {
		t["server_name"] = sni
	}
	if truthy(first(q, "allowInsecure", "insecure", "allow_insecure", "skip-cert-verify")) {
		t["insecure"] = true
	}
	if alpn := q.Get("alpn"); alpn != "" {
		t["alpn"] = strings.Split(alpn, ",")
	}
	if fp := q.Get("fp"); fp != "" && fp != "none" {
		t["utls"] = map[string]any{"enabled": true, "fingerprint": fp}
	}
	if security == "reality" {
		r := map[string]any{"enabled": true, "public_key": q.Get("pbk")}
		if sid := q.Get("sid"); sid != "" {
			r["short_id"] = sid
		}
		t["reality"] = r
		// Reality 要 uTLS
		if _, ok := t["utls"]; !ok {
			t["utls"] = map[string]any{"enabled": true, "fingerprint": "chrome"}
		}
	}
	return t
}

// transportOf 按链接参数拼传输层：ws、grpc、http（h2）、httpupgrade、quic；tcp（或不写）不用传输层。
func transportOf(kind, host, path, serviceName string) (map[string]any, error) {
	switch strings.ToLower(kind) {
	case "", "tcp", "raw", "none":
		return nil, nil
	case "ws", "websocket":
		t := map[string]any{"type": "ws", "path": orDefault(path, "/")}
		if host != "" {
			t["headers"] = map[string]any{"Host": host}
		}
		// Xray 的早期数据写在路径里：/路径?ed=2048
		if p, query, ok := strings.Cut(path, "?"); ok {
			if q, err := url.ParseQuery(query); err == nil {
				if ed, err := strconv.Atoi(q.Get("ed")); err == nil && ed > 0 {
					q.Del("ed")
					t["path"] = orDefault(p, "/")
					if rest := q.Encode(); rest != "" {
						t["path"] = orDefault(p, "/") + "?" + rest
					}
					t["max_early_data"] = ed
					t["early_data_header_name"] = "Sec-WebSocket-Protocol"
				}
			}
		}
		return t, nil
	case "grpc", "gun":
		return map[string]any{"type": "grpc", "service_name": serviceName}, nil
	case "http", "h2":
		t := map[string]any{"type": "http", "path": orDefault(path, "/")}
		if host != "" {
			t["host"] = strings.Split(host, ",")
		}
		return t, nil
	case "httpupgrade":
		t := map[string]any{"type": "httpupgrade", "path": orDefault(path, "/")}
		if host != "" {
			t["host"] = host
		}
		return t, nil
	case "quic":
		return map[string]any{"type": "quic"}, nil
	}
	return nil, fmt.Errorf("sing-box 不支持 %s 传输", kind)
}

// withTransport 链接里的 type、host、path、serviceName 拼成传输层放进出站。
func withTransport(out map[string]any, q url.Values) error {
	t, err := transportOf(q.Get("type"), q.Get("host"), q.Get("path"), q.Get("serviceName"))
	if err != nil {
		return err
	}
	setIf(out, "transport", t)
	return nil
}

func setIf(m map[string]any, key string, v map[string]any) {
	if v != nil {
		m[key] = v
	}
}

func first(q url.Values, keys ...string) string {
	for _, k := range keys {
		if v := q.Get(k); v != "" {
			return v
		}
	}
	return ""
}

func truthy(s string) bool { return s == "1" || strings.EqualFold(s, "true") }

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

// decodeBase64 分享链接里的 base64：标准或 URL 安全，带不带填充都认。
func decodeBase64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if b, err := enc.DecodeString(s); err == nil {
			return b, nil
		}
	}
	return nil, errors.New("base64 解不开")
}

// parseLink 拆分享链接；# 后面的名字不要（里面常有没转义的字符）。
func parseLink(node string) (*url.URL, error) {
	link, _, _ := strings.Cut(node, "#")
	return url.Parse(link)
}

// userOf 链接里 @ 前面的部分（用户名）；没有为空。
func userOf(u *url.URL) string {
	if u.User == nil {
		return ""
	}
	return u.User.Username()
}

// vless://uuid@host:port?security=reality&pbk=&sid=&sni=&fp=&flow=&type=tcp|ws|grpc&path=&host=&serviceName=#名字
func parseVLESS(node string) (map[string]any, error) {
	u, err := parseLink(node)
	if err != nil || userOf(u) == "" {
		return nil, errors.New("vless 链接写得不对")
	}
	host, port, err := endpoint(u)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	out := base("vless", host, port)
	out["uuid"] = userOf(u)
	if flow := q.Get("flow"); flow != "" {
		out["flow"] = flow
	}
	if pe := q.Get("packetEncoding"); pe != "" {
		out["packet_encoding"] = pe
	}
	setIf(out, "tls", tlsOf(q, false, ""))
	return out, withTransport(out, q)
}

// vmess://base64({"add","port","id","aid","scy","net","host","path","tls","sni","alpn","fp"})，
// 或者跟 vless 一样的写法：vmess://uuid@host:port?encryption=auto&security=tls&type=ws…
func parseVMess(node string) (map[string]any, error) {
	body, _, _ := strings.Cut(node[len("vmess://"):], "#")
	raw, err := decodeBase64(body)
	if err != nil {
		return parseVMessURL(node)
	}
	var v map[string]any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, errors.New("vmess 链接解不开")
	}
	str := func(k string) string {
		switch x := v[k].(type) {
		case string:
			return x
		case float64:
			return strconv.Itoa(int(x))
		}
		return ""
	}
	port, err := strconv.Atoi(str("port"))
	if str("add") == "" || str("id") == "" || err != nil {
		return nil, errors.New("vmess 链接里没有地址、端口或 id")
	}
	out := base("vmess", str("add"), port)
	out["uuid"] = str("id")
	out["security"] = orDefault(str("scy"), "auto")
	if aid, _ := strconv.Atoi(str("aid")); aid > 0 {
		out["alter_id"] = aid
	}
	q := url.Values{}
	if str("tls") == "tls" {
		q.Set("security", "tls")
	}
	q.Set("sni", str("sni"))
	q.Set("alpn", str("alpn"))
	q.Set("fp", str("fp"))
	setIf(out, "tls", tlsOf(q, false, str("host")))
	t, err := transportOf(str("net"), str("host"), str("path"), str("path"))
	if err != nil {
		return nil, err
	}
	setIf(out, "transport", t)
	return out, nil
}

func parseVMessURL(node string) (map[string]any, error) {
	u, err := parseLink(node)
	if err != nil || userOf(u) == "" {
		return nil, errors.New("vmess 链接解不开")
	}
	host, port, err := endpoint(u)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	out := base("vmess", host, port)
	out["uuid"] = userOf(u)
	out["security"] = orDefault(q.Get("encryption"), "auto")
	setIf(out, "tls", tlsOf(q, false, ""))
	return out, withTransport(out, q)
}

// trojan://password@host:port?sni=&type=&path=&host=&allowInsecure=#名字（默认 TLS）
func parseTrojan(node string) (map[string]any, error) {
	u, err := parseLink(node)
	if err != nil || userOf(u) == "" {
		return nil, errors.New("trojan 链接写得不对")
	}
	host, port, err := endpoint(u)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	out := base("trojan", host, port)
	out["password"] = userOf(u)
	if !strings.EqualFold(q.Get("security"), "none") {
		setIf(out, "tls", tlsOf(q, true, host))
	}
	return out, withTransport(out, q)
}

// ss://base64(method:password)@host:port?plugin=#名字，或 ss://base64(method:password@host:port)#名字
func parseShadowsocks(node string) (map[string]any, error) {
	rest, _, _ := strings.Cut(node[len("ss://"):], "#")
	rest, query, _ := strings.Cut(rest, "?")
	var userinfo, hostport string
	if at := strings.LastIndex(rest, "@"); at >= 0 {
		userinfo, hostport = rest[:at], rest[at+1:]
		if decoded, err := decodeBase64(userinfo); err == nil && strings.Contains(string(decoded), ":") {
			userinfo = string(decoded)
		} else if unescaped, err := url.PathUnescape(userinfo); err == nil {
			userinfo = unescaped
		}
	} else {
		decoded, err := decodeBase64(rest)
		if err != nil {
			return nil, errors.New("ss 链接解不开")
		}
		at := strings.LastIndex(string(decoded), "@")
		if at < 0 {
			return nil, errors.New("ss 链接写得不对")
		}
		userinfo, hostport = string(decoded[:at]), string(decoded[at+1:])
	}
	hostport = strings.TrimSuffix(hostport, "/")
	method, password, ok := strings.Cut(userinfo, ":")
	if !ok || method == "" {
		return nil, errors.New("ss 链接里没有加密方式和密码")
	}
	h, p, err := net.SplitHostPort(hostport)
	port, perr := strconv.Atoi(p)
	if err != nil || perr != nil || h == "" {
		return nil, errors.New("ss 链接里没有主机或端口")
	}
	out := base("shadowsocks", h, port)
	out["method"], out["password"] = method, password
	// 插件：plugin=obfs-local;obfs=http;obfs-host=…，或 v2ray-plugin;…
	if q, err := url.ParseQuery(query); err == nil && q.Get("plugin") != "" {
		name, opts, _ := strings.Cut(q.Get("plugin"), ";")
		if name == "simple-obfs" {
			name = "obfs-local"
		}
		out["plugin"], out["plugin_opts"] = name, opts
	}
	return out, nil
}

// hysteria2://password@host:port?sni=&insecure=&obfs=salamander&obfs-password=&mport=#名字
func parseHysteria2(node string) (map[string]any, error) {
	u, err := parseLink(node)
	if err != nil {
		return nil, errors.New("hysteria2 链接写得不对")
	}
	host, port, err := endpoint(u)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	out := base("hysteria2", host, port)
	if u.User != nil {
		password := u.User.Username()
		if p, ok := u.User.Password(); ok {
			password += ":" + p
		}
		out["password"] = password
	}
	if obfs := q.Get("obfs"); obfs != "" && obfs != "none" {
		out["obfs"] = map[string]any{"type": obfs, "password": q.Get("obfs-password")}
	}
	// 端口跳跃：mport=20000-30000,40000
	if mport := q.Get("mport"); mport != "" {
		var ports []string
		for _, r := range strings.Split(mport, ",") {
			if lo, hi, ok := strings.Cut(r, "-"); ok {
				ports = append(ports, lo+":"+hi)
			} else {
				ports = append(ports, r+":"+r)
			}
		}
		out["server_ports"] = ports
	}
	out["tls"] = tlsOf(q, true, host)
	return out, nil
}

// hysteria://host:port?auth=&peer=&insecure=&upmbps=&downmbps=&alpn=&obfsParam=#名字
func parseHysteria(node string) (map[string]any, error) {
	u, err := parseLink(node)
	if err != nil {
		return nil, errors.New("hysteria 链接写得不对")
	}
	host, port, err := endpoint(u)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	out := base("hysteria", host, port)
	if auth := q.Get("auth"); auth != "" {
		out["auth_str"] = auth
	}
	up, _ := strconv.Atoi(q.Get("upmbps"))
	down, _ := strconv.Atoi(q.Get("downmbps"))
	out["up_mbps"], out["down_mbps"] = max(up, 10), max(down, 50)
	if obfs := q.Get("obfsParam"); obfs != "" {
		out["obfs"] = obfs
	}
	out["tls"] = tlsOf(q, true, host)
	return out, nil
}

// tuic://uuid:password@host:port?sni=&alpn=&congestion_control=&udp_relay_mode=&allow_insecure=#名字
func parseTUIC(node string) (map[string]any, error) {
	u, err := parseLink(node)
	if err != nil || userOf(u) == "" {
		return nil, errors.New("tuic 链接写得不对")
	}
	host, port, err := endpoint(u)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	out := base("tuic", host, port)
	out["uuid"] = userOf(u)
	if p, ok := u.User.Password(); ok {
		out["password"] = p
	}
	if cc := q.Get("congestion_control"); cc != "" {
		out["congestion_control"] = cc
	}
	if mode := q.Get("udp_relay_mode"); mode != "" {
		out["udp_relay_mode"] = mode
	}
	t := tlsOf(q, true, host)
	if truthy(q.Get("disable_sni")) {
		t["disable_sni"] = true
	}
	out["tls"] = t
	return out, nil
}

// anytls://password@host:port?sni=&insecure=#名字
func parseAnyTLS(node string) (map[string]any, error) {
	u, err := parseLink(node)
	if err != nil || userOf(u) == "" {
		return nil, errors.New("anytls 链接写得不对")
	}
	host, port, err := endpoint(u)
	if err != nil {
		return nil, err
	}
	out := base("anytls", host, port)
	out["password"] = userOf(u)
	out["tls"] = tlsOf(u.Query(), true, host)
	return out, nil
}

// wireguard://私钥@host:port?publickey=&presharedkey=&address=10.0.0.2/32,fd00::2/128&mtu=&reserved=1,2,3#名字
// sing-box 里 wireguard 是端点（不是出站）。
func parseWireGuard(node string) (map[string]any, error) {
	u, err := parseLink(node)
	if err != nil || userOf(u) == "" {
		return nil, errors.New("wireguard 链接写得不对")
	}
	host, port, err := endpoint(u)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	// 密钥是标准 base64，链接里没转义的 + 会被当成空格
	publicKey := strings.ReplaceAll(first(q, "publickey", "publicKey", "public_key", "peer_public_key"), " ", "+")
	if publicKey == "" {
		return nil, errors.New("wireguard 链接里没有对端公钥（publickey）")
	}
	var addresses []string
	for _, a := range strings.Split(first(q, "address", "ip", "local_address"), ",") {
		if a = strings.TrimSpace(a); a == "" {
			continue
		}
		if !strings.Contains(a, "/") {
			if strings.Contains(a, ":") {
				a += "/128"
			} else {
				a += "/32"
			}
		}
		addresses = append(addresses, a)
	}
	if len(addresses) == 0 {
		return nil, errors.New("wireguard 链接里没有本机地址（address）")
	}
	peer := map[string]any{
		"address":     host,
		"port":        port,
		"public_key":  publicKey,
		"allowed_ips": []string{"0.0.0.0/0", "::/0"},
	}
	if psk := strings.ReplaceAll(first(q, "presharedkey", "preSharedKey", "pre_shared_key"), " ", "+"); psk != "" {
		peer["pre_shared_key"] = psk
	}
	if reserved := q.Get("reserved"); reserved != "" {
		var bytes []int
		for _, s := range strings.Split(reserved, ",") {
			b, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil || b < 0 || b > 255 {
				return nil, errors.New("wireguard 链接里的 reserved 要写成 1,2,3")
			}
			bytes = append(bytes, b)
		}
		peer["reserved"] = bytes
	}
	out := map[string]any{
		"type":        "wireguard",
		"address":     addresses,
		"private_key": userOf(u),
		"peers":       []any{peer},
	}
	if mtu, err := strconv.Atoi(q.Get("mtu")); err == nil && mtu > 0 {
		out["mtu"] = mtu
	}
	return out, nil
}

// socks5://[用户名:密码@]host:port
func parseSocks(node string) (map[string]any, error) {
	u, err := parseLink(node)
	if err != nil {
		return nil, errors.New("socks 链接写得不对")
	}
	host, port, err := endpoint(u)
	if err != nil {
		return nil, err
	}
	out := base("socks", host, port)
	out["version"] = "5"
	if u.User != nil {
		out["username"] = u.User.Username()
		if p, ok := u.User.Password(); ok {
			out["password"] = p
		}
	}
	return out, nil
}
