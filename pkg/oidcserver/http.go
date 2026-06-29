package oidcserver

import (
	"encoding/json"
	"net/http"
	"strings"
)

// BearerToken 从 Authorization 头提取 OIDC access_token。
func BearerToken(r *http.Request) string {
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authorization, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
}

// WriteJSON 写出 OIDC 标准端点的原始 JSON 响应。
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// WriteError 写出 OAuth/OIDC 标准错误响应。
func WriteError(w http.ResponseWriter, status int, code string, err error) {
	WriteJSON(w, status, map[string]string{
		"error":             code,
		"error_description": err.Error(),
	})
}
