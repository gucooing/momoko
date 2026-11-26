//go:build linux

package sysservice

import (
	"strings"
	"testing"
)

func TestUnitFile(t *testing.T) {
	u := unitFile("/opt/my momoko", "/opt/my momoko/momoko")
	for _, want := range []string{
		"WorkingDirectory=/opt/my momoko\n",
		`ExecStart="/opt/my momoko/momoko"` + "\n",
		"Environment=" + managedEnv + "=1\n",
		"Restart=always\n",
		"User=root\n",
		"WantedBy=multi-user.target\n",
	} {
		if !strings.Contains(u, want) {
			t.Errorf("单元文件里没有 %q：\n%s", want, u)
		}
	}
}

func TestParseUnitRoundTrip(t *testing.T) {
	for _, c := range [][2]string{
		{"/opt/momoko", "/opt/momoko/momoko"},
		{"/opt/my momoko", "/opt/my momoko/momoko"},
	} {
		info, err := parseUnit([]byte(unitFile(c[0], c[1])))
		if err != nil {
			t.Fatal(err)
		}
		if info.Dir != c[0] || info.Exe != c[1] {
			t.Errorf("解析出 %+v，应为 %v", info, c)
		}
	}
}
