package sub2api

import "testing"

func TestMaskPublicDisplayName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"  alice  ", "alice"}, // 用户名不打码
		{"中文用户", "中文用户"},
		{"user_01", "user_01"},
		{"a@b.com", "a***@b.com"},
		{"ab@b.com", "a***@b.com"},
		{"abc@b.com", "ab***@b.com"},
		{"abcdef@x.y.z", "ab***@x.y.z"},
		{"no-at-here", "no-at-here"},
		{"@only.com", "@only.com"}, // @ 在开头当用户名
		{"local@", "local@"},       // 无域名
		{"local@nodot", "local@nodot"},
	}
	for _, c := range cases {
		if got := MaskPublicDisplayName(c.in); got != c.want {
			t.Errorf("MaskPublicDisplayName(%q)=%q want %q", c.in, got, c.want)
		}
	}
}
