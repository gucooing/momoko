// Package agentproxy 是 Agent 的代理配置：一个节点（分享链接或 sing-box 的出站 JSON），以及沙箱里的时区和语言。
// 节点在沙箱里由 sing-box 用（见 pkg/agentsandbox），所以 sing-box 支持的协议都能用：
// vless、vmess、trojan、shadowsocks、hysteria2、hysteria、tuic、anytls、wireguard、ssh、socks（见 Protocols）。
// 存起来是一段 JSON：{"node": 原样的节点, "tz": 时区, "lang": 语言}。
package agentproxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

// Proxy 一份代理配置。
type Proxy struct {
	// Node 用户给的节点，原样：分享链接，或 sing-box 的出站 JSON。
	Node string
	// Outbound 由 Node 得出的 sing-box 出站（不带 tag）。
	Outbound map[string]any
	// Timezone 沙箱里的时区（IANA 名称，如 America/Los_Angeles）；空 = 跟服务器一样。
	Timezone string
	// Locale 沙箱里的语言区域（如 en_US.UTF-8）；空 = 跟服务器一样。
	Locale string
}

var (
	timezonePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_+\-]*(/[A-Za-z0-9_+\-]+)*$`)
	localePattern   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.\-]{0,39}$`)
)

// New 从节点、时区、语言得出一份配置：节点认不出、时区或语言写得不对时报错。
func New(node, timezone, locale string) (Proxy, error) {
	node = strings.TrimSpace(node)
	if node == "" {
		return Proxy{}, errors.New("先填节点")
	}
	outbound, err := parseNode(node)
	if err != nil {
		return Proxy{}, err
	}
	p := Proxy{Node: node, Outbound: outbound, Timezone: strings.TrimSpace(timezone), Locale: strings.TrimSpace(locale)}
	if p.Timezone != "" && (!timezonePattern.MatchString(p.Timezone) || strings.Contains(p.Timezone, "..")) {
		return Proxy{}, errors.New("时区要写成 America/Los_Angeles 这样的名称")
	}
	if p.Locale != "" && !localePattern.MatchString(p.Locale) {
		return Proxy{}, errors.New("语言要写成 en_US.UTF-8 这样的名称")
	}
	return p, nil
}

type stored struct {
	Node     string `json:"node"`
	Timezone string `json:"tz,omitempty"`
	Locale   string `json:"lang,omitempty"`
}

// Parse 读 Encode 存下的配置。
func Parse(raw string) (Proxy, error) {
	var s stored
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return Proxy{}, errors.New("代理配置读不出来")
	}
	return New(s.Node, s.Timezone, s.Locale)
}

// Encode 存起来的配置。
func (p Proxy) Encode() string {
	raw, _ := json.Marshal(stored{Node: p.Node, Timezone: p.Timezone, Locale: p.Locale})
	return string(raw)
}

// Protocol 出站的协议（sing-box 的 type）。
func (p Proxy) Protocol() string {
	s, _ := p.Outbound["type"].(string)
	return s
}

// Endpoint 节点在 sing-box 里是端点（wireguard），不是出站。
func (p Proxy) Endpoint() bool { return p.Protocol() == "wireguard" }

// Server 节点的「主机:端口」（wireguard 是第一个对端的）；出站里没有为空。
func (p Proxy) Server() string {
	if peers, ok := p.Outbound["peers"].([]any); ok && len(peers) > 0 {
		if peer, ok := peers[0].(map[string]any); ok {
			return hostPort(peer["address"], peer["port"])
		}
	}
	return hostPort(p.Outbound["server"], p.Outbound["server_port"])
}

func hostPort(h, port any) string {
	host, _ := h.(string)
	if host == "" {
		return ""
	}
	switch n := port.(type) {
	case int:
		return net.JoinHostPort(host, strconv.Itoa(n))
	case float64:
		return net.JoinHostPort(host, strconv.Itoa(int(n)))
	}
	return host
}

// String 给人看的节点：协议和地址，不带密钥。
func (p Proxy) String() string {
	if s := p.Server(); s != "" {
		return fmt.Sprintf("%s://%s", p.Protocol(), s)
	}
	return p.Protocol()
}
