package file

import (
	"context"
	"errors"
	"io"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/localfs"
)

// localCaps 是本地来源的能力集合（唯一支持压缩/解压的来源）。
var localCaps = Caps{Presign: false, Copy: true, Move: true, Compress: true, ResumableUpload: true}

// LocalStore 把 localfs 视图适配成 Store 协议。
// 它只做协议转换：路径校验、越界防护、大小限流全部由内部的 localfs.FS 承担。
type LocalStore struct {
	fs *localfs.FS
}

// NewSystemStore 构造整机来源（管理员文件管理器用）。
// opts 通常带上 localfs.Deny(...) 以保护 momoko 自身的数据与配置目录。
func NewSystemStore(opts ...localfs.Option) (*LocalStore, error) {
	fsys, err := localfs.OpenSystem(opts...)
	if err != nil {
		return nil, err
	}
	return &LocalStore{fs: fsys}, nil
}

// NewScopedStore 构造锁死在 dir 内的本地来源（实例工作目录等）。
// 一切越出 dir 的访问都会失败，包括经符号链接/junction 的迂回。
func NewScopedStore(dir string, opts ...localfs.Option) (*LocalStore, error) {
	fsys, err := localfs.Open(dir, opts...)
	if err != nil {
		return nil, err
	}
	return &LocalStore{fs: fsys}, nil
}

// FS 返回底层视图，供需要 localfs 原生能力（分片上传缓冲、真实路径解析）的调用方使用。
func (s *LocalStore) FS() *localfs.FS { return s.fs }

// Sub 在当前来源内再收窄出一个以 vpath 为根的本地来源。
// 分享用它把每个被分享的目录条目锁成独立来源，公开访问者即便构造出畸形子路径也只能在该目录内打转。
func (s *LocalStore) Sub(vpath string, opts ...localfs.Option) (*LocalStore, error) {
	sub, err := s.fs.Sub(vpath, opts...)
	if err != nil {
		return nil, err
	}
	return &LocalStore{fs: sub}, nil
}

func (s *LocalStore) List(_ context.Context, dir string, field v1.FileSortField, desc bool) ([]*v1.FileEntryInfo, error) {
	items, err := s.fs.List(dir, localfs.ListOptions{Sort: toSortField(field), Desc: desc})
	if err != nil {
		return nil, err
	}
	return toEntryInfos(items), nil
}

func (s *LocalStore) Search(_ context.Context, dir, keywords string, recursive bool) ([]*v1.FileEntryInfo, error) {
	items, err := s.fs.Search(dir, localfs.SearchOptions{Keywords: keywords, Recursive: recursive})
	if err != nil {
		return nil, err
	}
	return toEntryInfos(items), nil
}

func (s *LocalStore) DirInfo(_ context.Context, dir string) (*v1.FileDirectoryInfo, error) {
	st, err := s.fs.DirStat(dir)
	if err != nil {
		return nil, err
	}
	return toDirectoryInfo(st), nil
}

func (s *LocalStore) Stat(_ context.Context, path string) (*v1.FileEntryInfo, error) {
	e, err := s.fs.Stat(path)
	if err != nil {
		return nil, err
	}
	return toEntryInfo(e), nil
}

func (s *LocalStore) Open(_ context.Context, path string) (io.ReadCloser, *v1.FileEntryInfo, error) {
	f, e, err := s.fs.OpenRead(path)
	if err != nil {
		return nil, nil, err
	}
	return f, toEntryInfo(e), nil
}

func (s *LocalStore) Write(_ context.Context, path string, r io.Reader) error {
	_, err := s.fs.WriteFile(path, r)
	return err
}

func (s *LocalStore) Create(_ context.Context, item *v1.FileCreateItem) error {
	if item == nil {
		return errors.New("缺少创建参数")
	}
	if item.IsDir {
		return s.fs.MkdirAll(item.Path)
	}
	return s.fs.CreateFile(item.Path, item.Content)
}

func (s *LocalStore) Delete(_ context.Context, paths []string) []*v1.FileOperationResult {
	return toOperationResults(s.fs.Remove(paths))
}

func (s *LocalStore) Rename(_ context.Context, path, newName string) (string, error) {
	return s.fs.Rename(path, newName)
}

func (s *LocalStore) Caps() Caps { return localCaps }

// ---- 可选能力 ----

func (s *LocalStore) CopyToDir(_ context.Context, paths []string, targetDir string) []*v1.FileOperationResult {
	return toOperationResults(s.fs.CopyInto(paths, targetDir))
}

func (s *LocalStore) MoveToDir(_ context.Context, paths []string, targetDir string) []*v1.FileOperationResult {
	return toOperationResults(s.fs.MoveInto(paths, targetDir))
}

func (s *LocalStore) Compress(_ context.Context, paths []string, targetPath string) (string, error) {
	return s.fs.Compress(paths, targetPath)
}

func (s *LocalStore) Unzip(_ context.Context, path, targetPath string) (string, error) {
	return s.fs.Extract(path, targetPath)
}

// ---- 分片上传：分片经 momoko 签名端点落到目标目录旁的缓冲文件，收尾时同卷原子 rename ----

func (s *LocalStore) PrepareUpload(_ context.Context, u *Upload) error {
	fillSignedParts(u)
	return nil
}

func (s *LocalStore) CompleteUpload(_ context.Context, u *Upload) error {
	_, err := s.fs.CommitUpload(u.Spec, u.Done())
	return err
}

func (s *LocalStore) CancelUpload(_ context.Context, u *Upload) error {
	return s.fs.DiscardUpload(u.Spec)
}

// 编译期断言：LocalStore 必须满足 Store 与全部可选能力接口。
var (
	_ Store    = (*LocalStore)(nil)
	_ Copier   = (*LocalStore)(nil)
	_ Mover    = (*LocalStore)(nil)
	_ Archiver = (*LocalStore)(nil)
)
