package server

import (
	"net/http"
	"strings"

	httpm "github.com/go-kratos/kratos/v2/transport/http"
)

func corsMiddleware() httpm.FilterFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				appendVary(w.Header(), "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

			requestHeaders := r.Header.Get("Access-Control-Request-Headers")
			if requestHeaders != "" {
				w.Header().Set("Access-Control-Allow-Headers", requestHeaders)
				appendVary(w.Header(), "Access-Control-Request-Headers")
			} else {
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Requested-With")
			}

			w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, Set-Cookie")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func appendVary(h http.Header, value string) {
	current := h.Get("Vary")
	if current == "" {
		h.Set("Vary", value)
		return
	}
	for _, part := range strings.Split(current, ",") {
		if strings.TrimSpace(part) == value {
			return
		}
	}
	h.Set("Vary", current+", "+value)
}
