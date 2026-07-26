package localfs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Stat 返回单个条目的信息。不跟随符号链接（按 lstat 语义），
// 因此指向视图外的链接会被如实报告为链接，而不会泄露链接目标的内容。
func (f *FS) Stat(vpath string) (*Entry, error) {
	l, err := f.resolve(vpath)
	if err != nil {
		if virtual(err) {
			return &Entry{Name: systemRootLabel, Path: "", IsDir: true, Mode: fs.ModeDir | 0o555}, nil
		}
		return nil, err
	}
	defer l.close()
	info, err := l.root.Lstat(l.rel)
	if err != nil {
		return nil, sanitize(err)
	}
	return newEntry(info, l.real), nil
}

// Exists 报告目标是否存在。解析失败（越界/受保护）一并按不存在处理，
// 避免调用方把「拒绝访问」和「不存在」当成两种可区分的状态而形成探测信道。
func (f *FS) Exists(vpath string) bool {
	_, err := f.Stat(vpath)
	return err == nil
}

// List 列出目录下的直接子项。目录恒排在文件之前，其余按 opt 排序。
// 整机视图的虚拟根（Windows「此电脑」）返回全部盘符。
func (f *FS) List(vpath string, opt ListOptions) ([]*Entry, error) {
	l, err := f.resolve(vpath)
	if err != nil {
		if virtual(err) {
			return f.listMounts(), nil
		}
		return nil, err
	}
	defer l.close()

	dirents, err := fs.ReadDir(l.root.FS(), l.rel)
	if err != nil {
		return nil, sanitize(err)
	}
	out := make([]*Entry, 0, len(dirents))
	for _, d := range dirents {
		if isInternalName(d.Name()) {
			continue // momoko 自己的中间产物，对用户不可见
		}
		info, err := d.Info()
		if err != nil {
			// 条目在列举过程中消失（或无权 stat）：跳过而不是让整次列目录失败。
			continue
		}
		e := newEntry(info, filepath.Join(l.real, d.Name()))
		if f.policy.deniedLexical(e.Path) {
			continue // 保护清单内的条目对文件管理器完全不可见
		}
		out = append(out, e)
	}
	sortEntries(out, opt)
	return out, nil
}

// listMounts 返回「此电脑」虚拟根下的挂载点条目。
func (f *FS) listMounts() []*Entry {
	mounts := systemMounts()
	out := make([]*Entry, 0, len(mounts))
	for _, m := range mounts {
		info, err := os.Stat(m)
		if err != nil {
			continue
		}
		out = append(out, mountEntry(m, info))
	}
	return out
}

// DirStat 返回目录概览（名称、父路径、子项计数）。
func (f *FS) DirStat(vpath string) (*DirStat, error) {
	l, err := f.resolve(vpath)
	if err != nil {
		if virtual(err) {
			n := int64(len(f.listMounts()))
			return &DirStat{Name: systemRootLabel, Dirs: n, Items: n}, nil
		}
		return nil, err
	}
	defer l.close()

	dirents, err := fs.ReadDir(l.root.FS(), l.rel)
	if err != nil {
		return nil, sanitize(err)
	}
	st := &DirStat{}
	for _, d := range dirents {
		if isInternalName(d.Name()) {
			continue // 与 List 保持一致，中间产物不计入
		}
		if d.IsDir() {
			st.Dirs++
			continue
		}
		st.Files++
	}
	st.Items = st.Dirs + st.Files
	st.Path, st.ParentPath, st.Name = f.displayPaths(l)
	return st, nil
}

// displayPaths 给出目录概览用的展示路径三元组。
// 受限视图回传相对形态（"." / "./sub"），不把服务器真实目录结构暴露给前端；
// 整机视图回传真实绝对路径，且盘符根的父级是虚拟根（空串）。
func (f *FS) displayPaths(l *loc) (self, parent, name string) {
	if f.system {
		self = l.real
		if l.isRoot() {
			return self, "", mountLabel(l.mount)
		}
		return self, filepath.Dir(self), filepath.Base(self)
	}
	if l.isRoot() {
		return ".", "", filepath.Base(f.base)
	}
	self = "." + string(filepath.Separator) + filepath.FromSlash(l.rel)
	parentRel := relParent(l.rel)
	if parentRel == "." {
		parent = "."
	} else {
		parent = "." + string(filepath.Separator) + filepath.FromSlash(parentRel)
	}
	return self, parent, filepath.Base(self)
}

// Search 在 vpath 下按名称子串搜索。
// 受策略双重限流：MaxSearchResults 限输出、MaxSearchVisits 限遍历量，
// 因此对整机视图执行递归搜索也不会把服务器拖死。
//
// 不跟随符号链接（fs.WalkDir 把链接视作叶子），所以既不会绕出视图，也不会陷入链接环。
func (f *FS) Search(vpath string, opt SearchOptions) ([]*Entry, error) {
	limit := opt.Limit
	if limit <= 0 || limit > f.policy.MaxSearchResults {
		limit = f.policy.MaxSearchResults
	}

	l, err := f.resolve(vpath)
	if err != nil {
		if virtual(err) {
			// 盘符层级只按盘符名过滤：跨盘递归代价过大且几乎必然超限。
			out := make([]*Entry, 0, 26)
			for _, m := range f.listMounts() {
				if matches(m.Name, opt.Keywords) {
					out = append(out, m)
				}
			}
			return out, nil
		}
		return nil, err
	}
	defer l.close()

	out := make([]*Entry, 0, 64)
	visits := 0
	walkErr := fs.WalkDir(l.root.FS(), l.rel, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// 单个子目录不可读（权限等）不应中断整次搜索。
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if p == l.rel {
			return nil
		}
		if isInternalName(d.Name()) {
			return nil // momoko 自己的中间产物，不出现在搜索结果里
		}
		// 保护子树整棵跳过，而不是遍历完再过滤结果。
		if f.policy.deniedLexical(filepath.Join(l.mount, filepath.FromSlash(p))) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		visits++
		if visits > f.policy.MaxSearchVisits {
			return errSearchBudget
		}

		if matches(d.Name(), opt.Keywords) {
			if info, err := d.Info(); err == nil {
				real := filepath.Join(l.mount, filepath.FromSlash(p))
				if !f.policy.deniedLexical(real) {
					out = append(out, newEntry(info, real))
					if len(out) >= limit {
						return errSearchBudget
					}
				}
			}
		}
		if !opt.Recursive && d.IsDir() {
			return fs.SkipDir
		}
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, errSearchBudget) {
		return nil, sanitize(walkErr)
	}
	sortEntries(out, ListOptions{})
	return out, nil
}

// errSearchBudget 是提前结束遍历的内部哨兵（命中上限，不是错误）。
var errSearchBudget = errors.New("localfs: search budget reached")

func matches(name, keywords string) bool {
	if keywords == "" {
		return true
	}
	return strings.Contains(strings.ToLower(name), strings.ToLower(keywords))
}

// sortEntries 目录优先，随后按指定字段排序；同值回落到名称升序以保证结果稳定。
func sortEntries(items []*Entry, opt ListOptions) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		switch opt.Sort {
		case SortByModTime:
			if !a.ModTime.Equal(b.ModTime) {
				if opt.Desc {
					return a.ModTime.After(b.ModTime)
				}
				return a.ModTime.Before(b.ModTime)
			}
		case SortBySize:
			if a.Size != b.Size {
				if opt.Desc {
					return a.Size > b.Size
				}
				return a.Size < b.Size
			}
		case SortByName:
		}
		if opt.Desc {
			return a.Name > b.Name
		}
		return a.Name < b.Name
	})
}
