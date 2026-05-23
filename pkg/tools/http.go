package tools

import (
	"context"
	"net"
	"net/http"
	"strings"

	httptransport "github.com/go-kratos/kratos/v2/transport/http"
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

func ClientIPFromContext(ctx context.Context) string {
	httpReq, ok := httptransport.RequestFromServerContext(ctx)
	if !ok {
		return ""
	}
	return ClientIP(httpReq)
}

func UserAgent(r *http.Request) string {
	userAgent := r.UserAgent()
	return userAgent
}
