package localfs

import (
	"fmt"
	"strings"
)

// maxNameLen 单个路径分段的长度上限（字节）。多数文件系统的上限是 255。
const maxNameLen = 255

// winReservedNames 是 Windows 保留设备名。以这些名字（或以它们为主名，如 "NUL.txt"）
// 命名的文件在 Windows 上会被解释为设备而非文件，因此在所有平台一律拒绝：
// 否则一个 Linux 服务器上创建的 "NUL" 会让所有 Windows 客户端在下载/解压分享时失败。
var winReservedNames = map[string]struct{}{
	"CON": {}, "PRN": {}, "AUX": {}, "NUL": {},
	"COM0": {}, "COM1": {}, "COM2": {}, "COM3": {}, "COM4": {},
	"COM5": {}, "COM6": {}, "COM7": {}, "COM8": {}, "COM9": {},
	"LPT0": {}, "LPT1": {}, "LPT2": {}, "LPT3": {}, "LPT4": {},
	"LPT5": {}, "LPT6": {}, "LPT7": {}, "LPT8": {}, "LPT9": {},
	"CONIN$": {}, "CONOUT$": {},
}

// ValidateName 校验 name 是否可作为一个安全的单层文件/目录名使用。
// 只对「由我们新建的名字」把关（新建、重命名、上传落地、解压条目），
// 列目录与读取既有文件不受此约束——磁盘上已存在的怪名字仍应可见可读。
//
// 之所以必须独立于 os.Root 再做一层：os.Root 拦不住 Windows 的名称改写语义。
// 实测 Create("a.txt ") 会静默落在 "a.txt" 上，于是 O_EXCL 的存在性检查形同虚设，
// 攻击者可借此覆盖同目录下的任意既有文件。
func ValidateName(name string) error { return validateName(name, true) }

// validateArchiveName 是解压条目名的校验：比 ValidateName 放宽 `<>"|?*` 这一组
// 「仅 Windows 非法」的字符，好让来自 Linux 的正常压缩包能在 Linux 上原样解开；
// 真正有安全含义的部分（"..", ':' 引出的备用数据流、控制字符、结尾点空格、保留设备名）一律照旧拒绝。
func validateArchiveName(name string) error { return validateName(name, false) }

func validateName(name string, strictChars bool) error {
	switch {
	case name == "":
		return fmt.Errorf("%w：名称不能为空", ErrInvalidName)
	case name == "." || name == "..":
		return fmt.Errorf("%w：不能使用 %q", ErrInvalidName, name)
	case len(name) > maxNameLen:
		return fmt.Errorf("%w：名称过长（上限 %d 字节）", ErrInvalidName, maxNameLen)
	}

	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("%w：名称不能包含路径分隔符", ErrInvalidName)
	}
	// ':' 同时封堵盘符（"C:"）与 Windows 备用数据流（"a.txt:hidden"、"a.txt::$DATA"）。
	if strings.Contains(name, ":") {
		return fmt.Errorf("%w：名称不能包含 ':'", ErrInvalidName)
	}
	for _, r := range name {
		if r == 0 {
			return fmt.Errorf("%w：名称不能包含 NUL 字节", ErrInvalidName)
		}
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("%w：名称不能包含控制字符", ErrInvalidName)
		}
		// Windows 下这些字符非法；新建名称时统一拒绝以保证跨平台可搬运。
		if strictChars && strings.ContainsRune(`<>"|?*`, r) {
			return fmt.Errorf("%w：名称不能包含字符 %q", ErrInvalidName, r)
		}
	}
	// Windows 会静默剥掉结尾的 '.' 与空格，导致 "a.txt " 实际写入 "a.txt"。
	if last := name[len(name)-1]; last == '.' || last == ' ' {
		return fmt.Errorf("%w：名称不能以 '.' 或空格结尾", ErrInvalidName)
	}
	if name[0] == ' ' {
		return fmt.Errorf("%w：名称不能以空格开头", ErrInvalidName)
	}
	// 保留设备名判定取主名（"NUL.txt" 同样是设备）。
	stem := name
	if i := strings.IndexByte(stem, '.'); i >= 0 {
		stem = stem[:i]
	}
	if _, bad := winReservedNames[strings.ToUpper(stem)]; bad {
		return fmt.Errorf("%w：%q 是系统保留名称", ErrInvalidName, stem)
	}
	return nil
}

// ValidateToken 校验一个服务端生成的标识（上传会话 id 等）是否可安全嵌入文件名。
// 只允许 [A-Za-z0-9_-]，从而使由它拼出的缓冲文件名不可能带上任何路径语义。
func ValidateToken(s string) error {
	if s == "" {
		return fmt.Errorf("%w：标识不能为空", ErrInvalidName)
	}
	if len(s) > 64 {
		return fmt.Errorf("%w：标识过长", ErrInvalidName)
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return fmt.Errorf("%w：标识只能包含字母、数字、下划线与连字符", ErrInvalidName)
		}
	}
	return nil
}

// SafeName 把任意字符串压成一个安全的单层名称，用于「必须落地、不能报错」的场景
// （如以外部系统返回的 id 作为目录名）。不可打印字符、分隔符与保留名一律折叠为 '_'，
// 空结果回退为 fallback。注意：这是兜底手段，能报错的路径应优先用 ValidateName。
func SafeName(s, fallback string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '_', r == '-', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "._ ")
	if len(out) > maxNameLen {
		out = out[:maxNameLen]
		out = strings.Trim(out, "._ ")
	}
	if out == "" {
		return fallback
	}
	if err := ValidateName(out); err != nil {
		return fallback
	}
	return out
}
