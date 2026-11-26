//go:build windows

package sysservice

import (
	"testing"
	"time"

	"golang.org/x/sys/windows/svc"
)

func TestBinaryPath(t *testing.T) {
	cases := map[string]string{
		`"C:\Program Files\Momoko\momoko.exe"`:         `C:\Program Files\Momoko\momoko.exe`,
		`"C:\Program Files\Momoko\momoko.exe" -x "y"`:  `C:\Program Files\Momoko\momoko.exe`,
		`C:\Momoko\momoko.exe`:                         `C:\Momoko\momoko.exe`,
		`C:\Momoko\momoko.exe -conf C:\Momoko\configs`: `C:\Momoko\momoko.exe`,
	}
	for in, want := range cases {
		if got := binaryPath(in); got != want {
			t.Errorf("binaryPath(%q) = %q，应为 %q", in, got, want)
		}
	}
}

// 服务管理器要求停止时关闭 stop，serve 返回 0：服务正常停下，不触发故障恢复。
func TestHandlerStop(t *testing.T) {
	h := &handler{serve: func(stop <-chan struct{}) int { <-stop; return 0 }}
	r := make(chan svc.ChangeRequest, 1)
	s := make(chan svc.Status, 8)
	done := make(chan [2]any, 1)
	go func() { ssec, code := h.Execute(nil, r, s); done <- [2]any{ssec, code} }()
	waitStatus(t, s, svc.Running)
	r <- svc.ChangeRequest{Cmd: svc.Stop}
	waitStatus(t, s, svc.StopPending)
	got := <-done
	if got[0].(bool) || got[1].(uint32) != 0 {
		t.Fatalf("正常停止应返回 (false, 0)，得到 %v", got)
	}
}

// 在线更新后 serve 以非 0 返回：服务以失败结束，服务管理器按恢复设置把新程序拉起来。
func TestHandlerRestartAfterUpdate(t *testing.T) {
	h := &handler{serve: func(<-chan struct{}) int { return 75 }}
	r := make(chan svc.ChangeRequest)
	s := make(chan svc.Status, 8)
	ssec, code := h.Execute(nil, r, s)
	if !ssec || code != 75 || h.code != 75 {
		t.Fatalf("更新后应返回 (true, 75)，得到 (%v, %d)", ssec, code)
	}
}

func waitStatus(t *testing.T, s <-chan svc.Status, want svc.State) {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case st := <-s:
			if st.State == want {
				return
			}
		case <-timeout:
			t.Fatalf("等不到状态 %v", want)
		}
	}
}
