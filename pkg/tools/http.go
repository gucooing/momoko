package tools

import (
	"net"
	"net/http"
	"strings"
)

func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		ip := strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
		if ip != "" {
			return ip
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func UserAgent(r *http.Request) string {
	userAgent := r.UserAgent()
	return userAgent
}
