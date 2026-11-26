package initialize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfigPortsAndSetHTTPPort(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "configs")
	if err := WriteDefaultConfig(dir, 8080); err != nil {
		t.Fatal(err)
	}
	p, err := ReadPorts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.HTTP != 8080 || p.GRPC != 22733 || p.Passport != 22634 || !p.PassportEnabled {
		t.Fatalf("端口不对：%+v", p)
	}
	content, _ := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if !strings.Contains(string(content), "source: file:./data/momoko.db") || strings.Contains(string(content), "secret: \n") {
		t.Fatalf("默认配置不对：\n%s", content)
	}

	if err := SetHTTPPort(dir, 9090); err != nil {
		t.Fatal(err)
	}
	p, err = ReadPorts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.HTTP != 9090 || p.GRPC != 22733 || p.Passport != 22634 {
		t.Fatalf("只该改后台端口：%+v", p)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if !strings.Contains(string(after), "addr: 0.0.0.0:9090") || !strings.Contains(string(after), "public_url") {
		t.Fatalf("其它配置应原样保留：\n%s", after)
	}
}

func TestSetHTTPPortKeepsHost(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(file, []byte("server:\n    http:\n        addr: 127.0.0.1:22633\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SetHTTPPort(file, 1234); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(file)
	if !strings.Contains(string(b), "addr: 127.0.0.1:1234") {
		t.Fatalf("监听地址应保留：\n%s", b)
	}
}
