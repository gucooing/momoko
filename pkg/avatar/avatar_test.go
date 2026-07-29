package avatar

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const tinyPNG = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO7Z0r8AAAAASUVORK5CYII="

func TestPrepareDataURLStoresByUserID(t *testing.T) {
	manager := NewManagerWithRoot(t.TempDir())

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

	filePath, ok := manager.requestedFilePath(value)
	if !ok {
		t.Fatalf("头像路径无法转换为本地文件路径: %s", value)
	}
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("头像文件未保存成功: %v", err)
	}
}

func TestPrepareRejectsTraversalPath(t *testing.T) {
	manager := NewManagerWithRoot(t.TempDir())

	if _, _, _, err := manager.Prepare("user:test", PublicPath+"../secret.txt"); err == nil {
		t.Fatal("越权路径应被拒绝")
	}
}

func TestFilterRejectsTraversalPath(t *testing.T) {
	manager := NewManagerWithRoot(t.TempDir())
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
