package localfs

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// copySuffix 是目标同名时追加的后缀。
const copySuffix = "-copy"

// CopyInto 把若干文件/目录复制进 targetDir。
// MoveInto 把若干文件/目录移动进 targetDir。
// 两者都逐项返回结果：成功时 Message 为目标真实路径，失败时为原因。
// 源与目标可位于不同挂载点（整机视图下的跨盘操作），跨盘移动自动降级为「复制后删源」。
func (f *FS) CopyInto(vpaths []string, targetDir string) []Result {
	return f.transfer(vpaths, targetDir, false)
}

// MoveInto 见 [FS.CopyInto]。
func (f *FS) MoveInto(vpaths []string, targetDir string) []Result {
	return f.transfer(vpaths, targetDir, true)
}

func (f *FS) transfer(vpaths []string, targetDir string, move bool) []Result {
	out := make([]Result, 0, len(vpaths))
	for _, p := range vpaths {
		res := Result{Path: p}
		target, err := f.transferOne(p, targetDir, move)
		if err != nil {
			res.Message = err.Error()
		} else {
			res.OK = true
			res.Message = target
		}
		out = append(out, res)
	}
	return out
}

func (f *FS) transferOne(vpath, targetDir string, move bool) (string, error) {
	if err := f.checkWritable(); err != nil {
		return "", err
	}
	if strings.TrimSpace(vpath) == "" {
		return "", fmt.Errorf("%w：路径不能为空", ErrInvalidPath)
	}

	src, err := f.resolve(vpath)
	if err != nil {
		return "", err
	}
	defer src.close()
	if src.isRoot() {
		return "", ErrRootScope
	}
	srcInfo, err := src.root.Lstat(src.rel)
	if err != nil {
		return "", sanitize(err)
	}

	dst, err := f.resolveDir(targetDir)
	if err != nil {
		return "", err
	}
	defer dst.close()

	// 搬运会带走整棵子树，保护清单必须连子孙一起判。
	if err := f.checkSubtree(src.real); err != nil {
		return "", err
	}
	if err := f.checkSubtree(dst.real); err != nil {
		return "", err
	}
	// 目录不能被复制/移动进自己或自己的子孙里（否则边写边长，永不终止）。
	// 两侧都先解引用再比较：只比词法路径的话，一个指向源内部的软链接目标就能骗过检查，
	// 而 WalkDir 是惰性读目录的，新写进去的条目会被继续遍历，直到写满磁盘。
	if srcInfo.IsDir() && overlaps(src.real, dst.real) {
		return "", errors.New("目标目录不能位于源目录内部")
	}

	name := baseName(src.rel)
	if err := ValidateName(name); err != nil {
		return "", err
	}
	sameDir := pathEqual(filepath.Dir(src.real), dst.real)
	// 同目录内复制必然撞名，直接进入 "-copy" 序列；移动到同目录则是空操作。
	dstRel, err := f.nextFreeName(dst, name, !move && sameDir)
	if err != nil {
		return "", err
	}
	dstReal := joinReal(dst.mount, dstRel)
	if f.policy.denied(dstReal) {
		return "", ErrDenied
	}
	if pathEqual(src.real, dstReal) {
		return dstReal, nil
	}

	if move {
		if err := f.movePath(src, dst, dstRel, srcInfo); err != nil {
			return "", err
		}
		return dstReal, nil
	}
	if err := f.copyPath(src, dst, dstRel, srcInfo); err != nil {
		return "", err
	}
	return dstReal, nil
}

// nextFreeName 在目标目录内挑一个未被占用的名字：
// name、name-copy、name-copy-2 …（forceSuffix 时跳过第一个候选）。
func (f *FS) nextFreeName(dst *loc, name string, forceSuffix bool) (string, error) {
	if !forceSuffix {
		rel := relJoin(dst.rel, name)
		free, err := notExists(dst.root, rel)
		if err != nil {
			return "", err
		}
		if free {
			return rel, nil
		}
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	if stem == "" { // 形如 ".gitignore"：整体当主名，不拆扩展名
		stem, ext = name, ""
	}
	for i := 1; i <= 1000; i++ {
		candidate := stem + copySuffix + ext
		if i > 1 {
			candidate = fmt.Sprintf("%s%s-%d%s", stem, copySuffix, i, ext)
		}
		if err := ValidateName(candidate); err != nil {
			return "", err
		}
		rel := relJoin(dst.rel, candidate)
		free, err := notExists(dst.root, rel)
		if err != nil {
			return "", err
		}
		if free {
			return rel, nil
		}
	}
	return "", errors.New("目标目录内同名文件过多，无法生成新名称")
}

func notExists(root *os.Root, rel string) (bool, error) {
	_, err := root.Lstat(rel)
	if err == nil {
		return false, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	return false, sanitize(err)
}

// movePath 优先用同挂载点内的 rename（原子且瞬时）；跨挂载点/跨设备时降级为复制后删源。
func (f *FS) movePath(src, dst *loc, dstRel string, srcInfo fs.FileInfo) error {
	if pathEqual(src.mount, dst.mount) {
		if err := src.root.Rename(src.rel, dstRel); err == nil {
			return nil
		} else if !crossDevice(err) {
			return sanitize(err)
		}
	}
	if err := f.copyPath(src, dst, dstRel, srcInfo); err != nil {
		return err
	}
	if err := src.root.RemoveAll(src.rel); err != nil {
		return fmt.Errorf("已复制但删除源失败: %w", sanitize(err))
	}
	return nil
}

func (f *FS) copyPath(src, dst *loc, dstRel string, srcInfo fs.FileInfo) error {
	if srcInfo.IsDir() {
		return f.copyDir(src, dst, dstRel)
	}
	return copyFileBetween(src.root, src.rel, dst.root, dstRel, srcInfo.Mode().Perm())
}

// copyDir 递归复制目录。遍历经由 os.Root 的 fs.FS，因此不会跟随符号链接、
// 也不可能沿链接爬出视图；指向视图外的链接会在读取时报错而非静默跳过。
func (f *FS) copyDir(src, dst *loc, dstRel string) error {
	if err := dst.root.MkdirAll(dstRel, 0o755); err != nil {
		return sanitize(err)
	}
	return fs.WalkDir(src.root.FS(), src.rel, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return sanitize(err)
		}
		if p == src.rel {
			return nil
		}
		rel, relErr := relTo(src.rel, p)
		if relErr != nil {
			return relErr
		}
		if err := ValidateName(d.Name()); err != nil {
			return fmt.Errorf("跳过非法名称 %q: %w", d.Name(), err)
		}
		if f.policy.deniedLexical(filepath.Join(src.mount, filepath.FromSlash(p))) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		target := relJoin(dstRel, rel)
		info, err := d.Info()
		if err != nil {
			return sanitize(err)
		}
		if d.IsDir() {
			return sanitize(dst.root.MkdirAll(target, info.Mode().Perm()))
		}
		return copyFileBetween(src.root, p, dst.root, target, info.Mode().Perm())
	})
}

// copyFileBetween 在两个受限根之间复制单个文件。目标已存在则失败（O_EXCL），
// 中途失败会删掉半成品，不留下截断的文件。
func copyFileBetween(srcRoot *os.Root, srcRel string, dstRoot *os.Root, dstRel string, mode fs.FileMode) error {
	if parent := relParent(dstRel); parent != "." {
		if err := dstRoot.MkdirAll(parent, 0o755); err != nil {
			return sanitize(err)
		}
	}
	in, err := srcRoot.Open(srcRel)
	if err != nil {
		return sanitize(err)
	}
	defer in.Close()

	if mode == 0 {
		mode = 0o644
	}
	out, err := dstRoot.OpenFile(dstRel, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return sanitize(err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = dstRoot.Remove(dstRel)
		return sanitize(err)
	}
	if err := out.Close(); err != nil {
		_ = dstRoot.Remove(dstRel)
		return sanitize(err)
	}
	return nil
}

// overlaps 报告 dst 是否落在 src 之内。两侧都尽力解引用符号链接后再比较——
// 纯词法比较挡不住「dst 是一个指向 src 内部的链接」这种构造。
func overlaps(src, dst string) bool {
	if withinPath(src, dst) {
		return true
	}
	realSrc, realDst := src, dst
	if r, err := filepath.EvalSymlinks(src); err == nil {
		realSrc = r
	}
	if r, err := filepath.EvalSymlinks(dst); err == nil {
		realDst = r
	}
	return withinPath(realSrc, realDst)
}

// relTo 计算 p 相对 base 的部分（两者都是同一个 fs.FS 内的 ValidPath）。
func relTo(base, p string) (string, error) {
	if base == "." {
		return p, nil
	}
	if !strings.HasPrefix(p, base+"/") {
		return "", fmt.Errorf("%w：遍历越出源目录", ErrTraversal)
	}
	return p[len(base)+1:], nil
}

// crossDevice 判断 rename 失败是否因跨设备/跨卷。
func crossDevice(err error) bool {
	if err == nil {
		return false
	}
	if le, ok := errors.AsType[*os.LinkError](err); ok {
		err = le.Err
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "cross-device") ||
		strings.Contains(msg, "cross device") ||
		strings.Contains(msg, "different disk drive")
}
