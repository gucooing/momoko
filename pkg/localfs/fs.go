package localfs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FS 是一个受安全策略约束的本地文件系统视图。
// 值语义、并发安全、无需 Close：每次操作独立打开并关闭底层 os.Root 句柄，
// 因此可以随请求随手构造（业务层的 store 就是每请求新建一个），不会累积句柄。
type FS struct {
	// base 为受限视图的真实根目录绝对路径；整机视图下为空。
	base string
	// system 标记整机视图。
	system bool
	policy Policy
}

// Open 返回锁死在 dir 内的受限视图。dir 必须已存在且是目录。
// 越出 dir 的一切访问（"..", 绝对路径, 符号链接/junction 逃逸）都会失败。
func Open(dir string, opts ...Option) (*FS, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("%w：根目录不能为空", ErrInvalidPath)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("%w：无法解析根目录", ErrInvalidPath)
	}
	// 根目录自身按常规语义解引用符号链接（它来自配置/数据库，不是每请求的攻击面），
	// 之后的所有子路径访问才由 os.Root 承担穿越防护。
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		abs = real
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, sanitize(err)
	}
	if !info.IsDir() {
		return nil, ErrNotDir
	}
	return &FS{base: filepath.Clean(abs), policy: defaultPolicy(opts...)}, nil
}

// OpenDir 与 [Open] 相同，但目录不存在时先创建（含各级父目录）。
// 供头像、生图存储、上传缓冲等「开机即应就绪」的固定目录使用。
func OpenDir(dir string, opts ...Option) (*FS, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("%w：根目录不能为空", ErrInvalidPath)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, sanitize(err)
	}
	return Open(dir, opts...)
}

// OpenSystem 返回整机视图，供管理员文件管理器使用。
// Windows 下空路径是「此电脑」虚拟根（列出盘符），类 Unix 下空路径等价于 "/"。
// 整机视图内部仍是「每个挂载点一个 os.Root」，穿越防护照常生效；
// 对 momoko 自身敏感目录的保护请用 [Deny]。
func OpenSystem(opts ...Option) (*FS, error) {
	return &FS{system: true, policy: defaultPolicy(opts...)}, nil
}

// Sub 在当前视图内再收窄出一个以 vpath 为根的受限视图。
// 常用于把一次分享、一个实例目录限制成独立视图。策略默认继承，可用 opts 追加覆盖。
//
// 这里必须经 os.Root 实打实地 Stat 一次，而不能只做路径解析：
// [FS.RealPath] 给出的是词法路径，若 vpath 恰是一个指向视图外的符号链接，
// 用它直接开新视图就会把范围「收窄」成向外扩张。os.Root 的 Stat 会拒绝逃逸的链接，
// 因此 Stat 成功即证明目标确实落在当前视图之内。
func (f *FS) Sub(vpath string, opts ...Option) (*FS, error) {
	l, err := f.resolve(vpath)
	if err != nil {
		if virtual(err) {
			return nil, ErrSystemScope
		}
		return nil, err
	}
	defer l.close()

	info, err := l.root.Stat(l.rel)
	if err != nil {
		return nil, sanitize(err)
	}
	if !info.IsDir() {
		return nil, ErrNotDir
	}
	// 把子视图钉在「解引用之后」的真实目录上，而不是 l.real 这个可能含符号链接的词法路径。
	// 否则 Stat 只证明了「此刻」目标在视图内，之后每次操作都会重新按名字打开——
	// 攻击者只要在中间把那个链接改指到别处，子视图就跟着漂出去了。
	realBase := l.real
	if resolved, err := filepath.EvalSymlinks(realBase); err == nil {
		realBase = resolved
	}
	// 解引用后必须仍在父视图之内（整机视图 base 为空，其边界是挂载点，已由上面的 Stat 保证）。
	if f.base != "" && !withinPath(f.base, realBase) {
		return nil, ErrTraversal
	}
	sub := &FS{base: realBase, policy: f.policy}
	for _, o := range opts {
		if o != nil {
			o(&sub.policy)
		}
	}
	return sub, nil
}

// Base 返回受限视图的根目录真实路径；整机视图返回空串。
func (f *FS) Base() string { return f.base }

// IsSystem 报告这是否是整机视图。
func (f *FS) IsSystem() bool { return f.system }

// RealPath 校验 vpath 并返回其真实绝对路径。
//
// 仅供确实需要真实路径的场景使用（把工作目录交给外部进程、把路径写入数据库以便日后
// 重新校验）。拿到真实路径后再自行 os.Open 就绕开了本包的全部防护，因此除上述场景外
// 一律应直接调用本视图的操作方法。
func (f *FS) RealPath(vpath string) (string, error) {
	l, err := f.resolve(vpath)
	if err != nil {
		return "", err
	}
	defer l.close()
	return l.real, nil
}

// loc 是一次解析结果：一个已打开的受限根句柄 + 根内相对路径。
type loc struct {
	root *os.Root
	// rel 是 fs.ValidPath 形式的根内相对路径（"." 表示根自身）。
	rel string
	// real 是真实绝对路径，仅用于回传展示与保护清单判定，绝不用于直接打开文件。
	real string
	// mount 是本次解析所用的挂载点（受限视图下即 base）。
	mount string
}

func (l *loc) close() {
	if l != nil && l.root != nil {
		_ = l.root.Close()
	}
}

// isRoot 报告本次解析指向的是视图/挂载点的根自身。
func (l *loc) isRoot() bool { return l.rel == "." }

// resolve 是本包唯一的路径解析入口，所有操作都必须经过它。
// 返回的 loc 持有一个已打开的 os.Root，调用方负责 close。
//
// 解析顺序刻意如此：先拒绝含 ".." 的原始输入（可审计），再拆挂载点，再打开 os.Root
// （此后内核保证不可逃逸），最后查保护清单。
func (f *FS) resolve(vpath string) (*loc, error) {
	p := strings.TrimSpace(vpath)

	if f.system {
		if p == "" || p == "." {
			if systemHasVirtualRoot() {
				return nil, errVirtualRoot
			}
			p = string(filepath.Separator)
		}
		// 整机视图下相对路径按进程工作目录解释（历史行为：如 "./servers"）。
		if !isRooted(p) {
			// 先拒绝 ".."，否则 filepath.Abs 会静默折叠掉它。
			if _, err := cleanRel(p); err != nil {
				return nil, err
			}
			abs, err := filepath.Abs(p)
			if err != nil {
				return nil, fmt.Errorf("%w：无法解析路径", ErrInvalidPath)
			}
			p = abs
		} else if _, err := cleanRel(trimVolume(p)); err != nil {
			return nil, err
		}
		mount, rel, err := splitMount(p)
		if err != nil {
			return nil, err
		}
		return f.openLoc(mount, rel)
	}

	// 受限视图：客户端可能回传绝对路径（Entry.Path 就是真实绝对路径），也可能给相对路径。
	var rel string
	var err error
	if isRooted(p) {
		if _, err = cleanRel(trimVolume(p)); err != nil {
			return nil, err
		}
		if rel, err = stripBase(f.base, p); err != nil {
			return nil, err
		}
	} else if rel, err = cleanRel(p); err != nil {
		return nil, err
	}
	return f.openLoc(f.base, rel)
}

// trimVolume 去掉 Windows 卷名前缀，便于对余下部分做 ".." 检查。
func trimVolume(p string) string {
	if vol := filepath.VolumeName(filepath.FromSlash(p)); vol != "" {
		return p[len(vol):]
	}
	return p
}

// openLoc 打开挂载点对应的 os.Root 并完成保护清单判定。
func (f *FS) openLoc(mount, rel string) (*loc, error) {
	real := joinReal(mount, rel)
	if f.policy.denied(real) {
		return nil, ErrDenied
	}
	root, err := os.OpenRoot(mount)
	if err != nil {
		return nil, sanitize(err)
	}
	return &loc{root: root, rel: rel, real: real, mount: mount}, nil
}

// resolveDir 解析 vpath 并要求它是一个已存在的目录。
func (f *FS) resolveDir(vpath string) (*loc, error) {
	l, err := f.resolve(vpath)
	if err != nil {
		return nil, err
	}
	info, err := l.root.Stat(l.rel)
	if err != nil {
		l.close()
		return nil, sanitize(err)
	}
	if !info.IsDir() {
		l.close()
		return nil, ErrNotDir
	}
	return l, nil
}

// resolveNew 解析一个「待创建」目标：校验其名称合法性并返回定位。
// 目标可以尚不存在，但其名称必须通过 ValidateName，父级必须在视图内。
func (f *FS) resolveNew(vpath string) (*loc, error) {
	l, err := f.resolve(vpath)
	if err != nil {
		return nil, err
	}
	if l.isRoot() {
		l.close()
		return nil, ErrRootScope
	}
	if err := ValidateName(filepath.Base(filepath.FromSlash(l.rel))); err != nil {
		l.close()
		return nil, err
	}
	return l, nil
}

// checkWritable 在任何写操作前调用。
func (f *FS) checkWritable() error {
	if f.policy.ReadOnly {
		return ErrReadOnly
	}
	return nil
}

// checkSubtree 在「会波及整棵子树」的操作（删除/改名/移动/复制/打包/解压落地）前调用。
// 单点访问用 resolve 内的 denied 就够了，但这类操作必须连目标的子孙一起判，
// 否则对着保护目录的父级动手就能整棵搬走或删掉。
func (f *FS) checkSubtree(real string) error {
	if f.policy.affectsDenied(real) {
		return ErrDenied
	}
	return nil
}

// virtual 报告 err 是否为虚拟根哨兵。
func virtual(err error) bool { return errors.Is(err, errVirtualRoot) }
