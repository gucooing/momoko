//go:build !windows && !linux && !darwin

package sysservice

import (
	"io"
	"os"
	"path/filepath"
)

func isWindowsService() bool { return false }

// IsAdmin 是不是 root。
func IsAdmin() bool { return os.Geteuid() == 0 }

// DefaultDir 默认安装目录。
func DefaultDir() string { return "/opt/momoko" }

// Supported 这个系统装不了。
func Supported() error { return ErrUnsupported }

// ValidateDir 要是绝对路径。
func ValidateDir(dir string) error {
	if !filepath.IsAbs(dir) {
		return ErrUnsupported
	}
	return nil
}

// ErrElevationCanceled 没拿到权限。
var ErrElevationCanceled = ErrUnsupported

// ErrAccessDenied 没有权限控制服务。
var ErrAccessDenied = ErrUnsupported

func Lookup() (Info, bool, error)                              { return Info{}, false, nil }
func Register(string, string) error                            { return ErrUnsupported }
func Unregister() error                                        { return ErrUnsupported }
func Start() error                                             { return ErrUnsupported }
func Stop() error                                              { return ErrUnsupported }
func LinkCommand(string, string) (string, error)               { return "", ErrUnsupported }
func UnlinkCommand(string, string) error                       { return nil }
func AllowFirewall(string, []int) (string, error)              { return "", nil }
func RemoveFirewall(string)                                    {}
func Run(serve func(stop <-chan struct{}) int) int             { return serve(nil) }
func PrepareServiceProcess(string)                             {}
func LogHint(string) string                                    { return "" }
func OwnConsole() bool                                         { return false }
func Elevate(string, []string, string, io.Writer) (int, error) { return 1, ErrUnsupported }
