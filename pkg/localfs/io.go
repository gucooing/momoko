package localfs

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// tempPrefix 是原子写入所用临时文件的前缀（点开头，便于识别与清理）。
const tempPrefix = ".momoko-tmp-"

// OpenRead 打开文件用于读取/串流。返回的 *os.File 实现了 io.ReadSeekCloser，
// 因此调用方可以直接交给 http.ServeContent 以支持 Range 续传。
// 目标是目录时返回 ErrIsDir；调用方负责 Close。
func (f *FS) OpenRead(vpath string) (*os.File, *Entry, error) {
	l, err := f.resolve(vpath)
	if err != nil {
		return nil, nil, err
	}
	defer l.close()

	file, err := l.root.Open(l.rel)
	if err != nil {
		return nil, nil, sanitize(err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, sanitize(err)
	}
	if info.IsDir() {
		_ = file.Close()
		return nil, nil, ErrIsDir
	}
	return file, newEntry(info, l.real), nil
}

// ReadFile 读取整个文件内容，超过策略上限（默认 10 MiB）则拒绝。
// 供在线编辑器等「必须一次读完」的场景使用；大文件请走 OpenRead 串流。
func (f *FS) ReadFile(vpath string) ([]byte, error) {
	file, entry, err := f.OpenRead(vpath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if entry.Size > f.policy.MaxFileSize {
		return nil, fmt.Errorf("%w（上限 %d 字节）", ErrTooLarge, f.policy.MaxFileSize)
	}
	// 多读 1 字节：文件可能在 Stat 之后被写大，据此发现并拒绝，避免上限被竞态绕过。
	data, err := io.ReadAll(io.LimitReader(file, f.policy.MaxFileSize+1))
	if err != nil {
		return nil, sanitize(err)
	}
	if int64(len(data)) > f.policy.MaxFileSize {
		return nil, fmt.Errorf("%w（上限 %d 字节）", ErrTooLarge, f.policy.MaxFileSize)
	}
	return data, nil
}

// WriteFile 覆盖写入一个文件（不存在则创建），返回写入字节数。
//
// 刻意就地覆盖，而不是「写临时文件再 rename」：后者虽然原子，却会把目标换成一个新 inode，
// 于是指向共享配置的符号链接会被替换成普通文件、硬链接被断开、属主变成 momoko 进程用户。
// 面板用户拿它编辑的正是 server.properties 这类常被 symlink 出去的配置，
// 悄悄改变部署拓扑的代价远大于「崩溃时可能写了一半」的代价。
//
// 内容先整体读进内存再落盘：这样体积超限在截断目标之前就被判定出来，
// 不会出现「先把文件清空、再报错说太大」。上限默认 10 MiB，内存开销可控。
func (f *FS) WriteFile(vpath string, r io.Reader) (int64, error) {
	if err := f.checkWritable(); err != nil {
		return 0, err
	}
	l, err := f.resolveNew(vpath)
	if err != nil {
		return 0, err
	}
	defer l.close()

	// 多读 1 字节以识别超限。
	data, err := io.ReadAll(io.LimitReader(r, f.policy.MaxFileSize+1))
	if err != nil {
		return 0, sanitize(err)
	}
	if int64(len(data)) > f.policy.MaxFileSize {
		return 0, fmt.Errorf("%w（上限 %d 字节）", ErrTooLarge, f.policy.MaxFileSize)
	}

	// 保留既有权限位；目标是目录则拒绝。
	mode := fs.FileMode(0o644)
	if info, err := l.root.Lstat(l.rel); err == nil {
		if info.IsDir() {
			return 0, ErrIsDir
		}
		if info.Mode().IsRegular() {
			mode = info.Mode().Perm()
		}
	}
	// O_TRUNC 会跟随视图内的符号链接写到其目标（保住链接本身）；
	// 指向视图外的链接仍由 os.Root 拒绝。
	file, err := l.root.OpenFile(l.rel, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return 0, sanitize(err)
	}
	n, err := file.Write(data)
	if err != nil {
		_ = file.Close()
		return int64(n), sanitize(err)
	}
	if err := file.Close(); err != nil {
		return int64(n), sanitize(err)
	}
	return int64(n), nil
}

// CreateFile 新建文件。目标已存在时返回 ErrExist（而不是静默覆盖——
// 「新建」动作误伤既有文件是最容易被忽视的数据丢失路径）。content 可为空。
func (f *FS) CreateFile(vpath string, content []byte) error {
	if err := f.checkWritable(); err != nil {
		return err
	}
	if int64(len(content)) > f.policy.MaxFileSize {
		return fmt.Errorf("%w（上限 %d 字节）", ErrTooLarge, f.policy.MaxFileSize)
	}
	l, err := f.resolveNew(vpath)
	if err != nil {
		return err
	}
	defer l.close()

	// 建父目录前逐段校验：MkdirAll 会一次造出多级目录，每一级都得是合法名字。
	if parent := relParent(l.rel); parent != "." {
		if err := validateRelNames(parent); err != nil {
			return err
		}
		if err := l.root.MkdirAll(parent, 0o755); err != nil {
			return sanitize(err)
		}
	}
	file, err := l.root.OpenFile(l.rel, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return sanitize(err)
	}
	if len(content) > 0 {
		if _, err := file.Write(content); err != nil {
			_ = file.Close()
			_ = l.root.Remove(l.rel)
			return sanitize(err)
		}
	}
	return sanitize(file.Close())
}

// MkdirAll 创建目录（含各级父目录）。目录已存在视为成功。
func (f *FS) MkdirAll(vpath string) error {
	if err := f.checkWritable(); err != nil {
		return err
	}
	l, err := f.resolveNew(vpath)
	if err != nil {
		return err
	}
	defer l.close()
	// 逐段校验名称：MkdirAll 会一次造出多级目录，每一级都得是合法名字。
	if err := validateRelNames(l.rel); err != nil {
		return err
	}
	return sanitize(l.root.MkdirAll(l.rel, 0o755))
}

// Remove 批量删除文件或目录（目录递归删除）。
// 逐个返回结果而不是遇错即止，与文件管理器的批量语义一致。
// 视图根/盘符根自身永远不可删除。
func (f *FS) Remove(vpaths []string) []Result {
	out := make([]Result, 0, len(vpaths))
	for _, p := range vpaths {
		out = append(out, Result{Path: p, OK: true, Message: ""})
		res := &out[len(out)-1]
		if err := f.removeOne(p); err != nil {
			res.OK = false
			res.Message = err.Error()
		}
	}
	return out
}

func (f *FS) removeOne(vpath string) error {
	if err := f.checkWritable(); err != nil {
		return err
	}
	l, err := f.resolve(vpath)
	if err != nil {
		return err
	}
	defer l.close()
	if l.isRoot() {
		return ErrRootScope
	}
	// RemoveAll 会递归删光目标之下的一切，包括其中的保护目录。
	if err := f.checkSubtree(l.real); err != nil {
		return err
	}
	return sanitize(l.root.RemoveAll(l.rel))
}

// Rename 重命名文件或目录。newName 必须是单层合法名称（不含任何路径成分），
// 因此改名永远不可能把目标移出所在目录。返回新的真实绝对路径。
func (f *FS) Rename(vpath, newName string) (string, error) {
	if err := f.checkWritable(); err != nil {
		return "", err
	}
	if err := ValidateName(newName); err != nil {
		return "", err
	}
	l, err := f.resolve(vpath)
	if err != nil {
		return "", err
	}
	defer l.close()
	if l.isRoot() {
		return "", ErrRootScope
	}

	targetRel := relJoin(relParent(l.rel), newName)
	targetReal := joinReal(l.mount, targetRel)
	if targetRel == l.rel {
		return targetReal, nil
	}
	// 改名会把整棵子树挪到新名字下，保护清单里的绝对路径随即失配——等于永久解除保护。
	if err := f.checkSubtree(l.real); err != nil {
		return "", err
	}
	if err := f.checkSubtree(targetReal); err != nil {
		return "", err
	}
	if _, err := l.root.Lstat(targetRel); err == nil {
		return "", ErrExist
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", sanitize(err)
	}
	if err := l.root.Rename(l.rel, targetRel); err != nil {
		return "", sanitize(err)
	}
	return targetReal, nil
}

// newTemp 在目标同目录下创建一个空的临时文件，返回其相对路径与清理函数。
//
// 刻意放在目标同目录：后续 rename 才落在同一卷上，从而瞬时且原子完成。
// 集中到统一临时目录看似更整洁，但那与目标多半不同卷，rename 无从成立，
// 只能退化成一次全量复制——为了观感付出成倍 I/O 不划算。
// 「碍眼」的问题另由 [isInternalName] 在列目录/搜索时过滤掉解决。
func (f *FS) newTemp(l *loc) (string, func(), error) {
	dir := relParent(l.rel)
	for range 8 {
		var buf [12]byte
		if _, err := rand.Read(buf[:]); err != nil {
			return "", func() {}, fmt.Errorf("生成临时文件名失败: %w", err)
		}
		tmpRel := relJoin(dir, tempPrefix+hex.EncodeToString(buf[:]))
		file, err := l.root.OpenFile(tmpRel, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			if errors.Is(err, fs.ErrExist) {
				continue
			}
			return "", func() {}, sanitize(err)
		}
		_ = file.Close()
		return tmpRel, func() { _ = l.root.Remove(tmpRel) }, nil
	}
	return "", func() {}, errors.New("创建临时文件失败：重试次数已用尽")
}

// validateRelNames 逐段校验相对路径中每一层的名称合法性。
func validateRelNames(rel string) error {
	if rel == "." {
		return nil
	}
	for seg := range splitRel(rel) {
		if err := ValidateName(seg); err != nil {
			return err
		}
	}
	return nil
}

// splitRel 迭代相对路径的各个分段。
func splitRel(rel string) func(func(string) bool) {
	return func(yield func(string) bool) {
		for rel != "" && rel != "." {
			i := len(rel)
			for j := 0; j < len(rel); j++ {
				if rel[j] == '/' {
					i = j
					break
				}
			}
			if !yield(rel[:i]) {
				return
			}
			if i == len(rel) {
				return
			}
			rel = rel[i+1:]
		}
	}
}

// baseName 返回逻辑路径的末段名。
func baseName(vpath string) string {
	return filepath.Base(filepath.FromSlash(vpath))
}
