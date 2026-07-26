package avatar

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	m, err := NewManagerWithRoot(t.TempDir())
	if err != nil {
		t.Fatalf("创建头像管理器失败: %v", err)
	}
	return m
}

const tinyPNG = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO7Z0r8AAAAASUVORK5CYII="

func TestPrepareDataURLStoresByUserID(t *testing.T) {
	manager := newTestManager(t)

	value, commit, rollback, err := manager.Prepare("user:test", tinyPNG)
	if err != nil {
		t.Fatalf("准备头像失败: %v", err)
	}
	defer rollback()

	if value != PublicPath+"user_test" {
		t.Fatalf("头像路径应直接使用用户 ID，实际为 %s", value)
	}

	if err := commit(); err != nil {
		t.Fatalf("提交头像失败: %v", err)
	}

	name, ok := manager.requestedFileName(value)
	if !ok {
		t.Fatalf("头像路径无法转换为本地文件名: %s", value)
	}
	if !manager.view.Exists(name) {
		t.Fatalf("头像文件未保存成功: %s", name)
	}
}

func TestPrepareRejectsTraversalPath(t *testing.T) {
	manager := newTestManager(t)

	if _, _, _, err := manager.Prepare("user:test", PublicPath+"../secret.txt"); err == nil {
		t.Fatal("越权路径应被拒绝")
	}
}

func TestFilterRejectsTraversalPath(t *testing.T) {
	manager := newTestManager(t)
	handler := manager.Filter()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodGet, PublicPath+"../secret.txt", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("越权路径应返回 404，实际为 %d", resp.Code)
	}
}
