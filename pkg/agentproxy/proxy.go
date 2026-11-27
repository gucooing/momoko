// Package agentproxy 是 Agent 的代理配置：上游代理（HTTP 或 SOCKS5，可带账号密码），以及在沙箱里看到的时区和语言。
// 存起来是一个地址：socks5://用户名:密码@主机:端口?tz=America/Los_Angeles&lang=en_US.UTF-8。
// 怎么经它联网见 pkg/agentsandbox。
package agentproxy

import (
	"errors"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// 代理类型。
const (
	TypeHTTP   = "http"
	TypeSOCKS5 = "socks5"
)

// Proxy 一份代理配置。
type Proxy struct {
	Type     string
	Host     string
	Port     int
	Username string
	Password string
	// Timezone 沙箱里的时区（IANA 名称，如 America/Los_Angeles）；空 = 跟服务器一样。
	Timezone string
	// Locale 沙箱里的语言区域（如 en_US.UTF-8）；空 = 跟服务器一样。
	Locale string
}

var (
	timezonePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_+\-]*(/[A-Za-z0-9_+\-]+)*$`)
	localePattern   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.\-]{0,39}$`)
)

// Parse 读 Encode 存下的地址。
func Parse(raw string) (Proxy, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return Proxy{}, errors.New("代理地址不对")
	}
	p := Proxy{Type: strings.ToLower(u.Scheme), Host: u.Hostname()}
	if p.Port, err = strconv.Atoi(u.Port()); err != nil {
		return Proxy{}, errors.New("代理地址里没有端口")
	}
	if u.User != nil {
		p.Username = u.User.Username()
		p.Password, _ = u.User.Password()
	}
	q := u.Query()
	p.Timezone, p.Locale = q.Get("tz"), q.Get("lang")
	if err := p.Validate(); err != nil {
		return Proxy{}, err
	}
	return p, nil
}

// Encode 存起来的地址：带账号密码，时区和语言在查询参数 tz、lang 里。
func (p Proxy) Encode() string {
	u := url.URL{Scheme: p.Type, Host: net.JoinHostPort(p.Host, strconv.Itoa(p.Port))}
	switch {
	case p.Username != "" && p.Password != "":
		u.User = url.UserPassword(p.Username, p.Password)
	case p.Username != "":
		u.User = url.User(p.Username)
	}
	q := url.Values{}
	if p.Timezone != "" {
		q.Set("tz", p.Timezone)
	}
	if p.Locale != "" {
		q.Set("lang", p.Locale)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// Validate 类型、主机、端口、时区、语言都要对。
func (p Proxy) Validate() error {
	if p.Type != TypeHTTP && p.Type != TypeSOCKS5 {
		return errors.New("代理只能是 http:// 或 socks5://")
	}
	if p.Host == "" || strings.ContainsAny(p.Host, "/ @?#") {
		return errors.New("代理地址里没有主机")
	}
	if p.Port < 1 || p.Port > 65535 {
		return errors.New("代理端口要在 1 到 65535 之间")
	}
	if p.Password != "" && p.Username == "" {
		return errors.New("代理有密码就要有用户名")
	}
	if p.Timezone != "" && (!timezonePattern.MatchString(p.Timezone) || strings.Contains(p.Timezone, "..")) {
		return errors.New("时区要写成 America/Los_Angeles 这样的名称")
	}
	if p.Locale != "" && !localePattern.MatchString(p.Locale) {
		return errors.New("语言要写成 en_US.UTF-8 这样的名称")
	}
	return nil
}

// String 给人看的地址，不带账号密码。
func (p Proxy) String() string {
	return p.Type + "://" + net.JoinHostPort(p.Host, strconv.Itoa(p.Port))
}
