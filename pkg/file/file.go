package file

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"momoko/internal/data/ent/gen"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
)

const (
	SortByName    = v1.FileSortField_FILE_SORT_FIELD_NAME
	SortByModTime = v1.FileSortField_FILE_SORT_FIELD_UPDATE_TIME

	SystemPath  = ""
	ServersPath = "./servers"

	// MaxLoadFileSize 最大在线读取大小
	MaxLoadFileSize = 10 * 1024 * 1024
)

type FileOper struct {
	basePath string // 底线路径
}

func NewFileOper(basePath string) (*FileOper, error) {
	f := &FileOper{}
	if basePath == SystemPath {
		f.basePath = basePath
	} else {
		baseAbs, err := filepath.Abs(basePath)
		if err != nil {
			return nil, err
		}
		f.basePath = baseAbs
	}

	return f, nil
}

// isSystemRoot 系统级且未指定路径时，代表“虚拟根”：
// Windows 下为盘符列表（此电脑），类 Unix 下为根目录 "/"。
// 这样前端无需手动输入盘符即可跨盘浏览整个文件系统。
func (f *FileOper) isSystemRoot(dir string) bool {
	return f.basePath == SystemPath && dir == ""
}

// listDrives 枚举当前存在的 Windows 盘符为目录项；非 Windows 返回 nil。
func listDrives() []*v1.FileEntryInfo {
	if runtime.GOOS != "windows" {
		return nil
	}
	drives := make([]*v1.FileEntryInfo, 0, 26)
	for c := 'A'; c <= 'Z'; c++ {
		p := fmt.Sprintf("%c:\\", c)
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		drives = append(drives, &v1.FileEntryInfo{
			Name:       fmt.Sprintf("%c:", c),
			Path:       p,
			IsDir:      true,
			Permission: "drwxrwxrwx",
			Size:       0,
			UpdateTime: timestamppb.New(info.ModTime()),
		})
	}
	return drives
}

// ResolveRealPath 返回 targetPath 对应的真实路径。
// 如果目标越过 basePath，返回错误。
func (f *FileOper) ResolveRealPath(targetPath string) (string, error) {
	if f.basePath == SystemPath {
		if filepath.IsAbs(targetPath) {
			return targetPath, nil
		}
		targetAbs, err := filepath.Abs(targetPath)
		if err != nil {
			return "", errors.New("路径不存在")
		}
		return targetAbs, nil
	}
	var candidate string
	if filepath.IsAbs(targetPath) {
		candidate = targetPath
	} else {
		candidate = filepath.Join(f.basePath, targetPath)
	}

	targetAbs, err := filepath.Abs(candidate)
	if err != nil {
		return "", errors.New("路径不存在")
	}

	targetPath, err = filepath.EvalSymlinks(targetAbs)
	if err != nil {
		return "", errors.New("路径不存在")
	}
	rel, err := filepath.Rel(f.basePath, targetPath)
	if err != nil {
		return "", errors.New("路径不存在")
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("路径穿越！")
	}

	return targetPath, nil
}

// ListDir 获取文件列表
func (f *FileOper) ListDir(dir string, field v1.FileSortField, order bool) ([]*v1.FileEntryInfo, error) {
	if f.isSystemRoot(dir) {
		if runtime.GOOS == "windows" {
			return listDrives(), nil
		}
		dir = "/"
	}
	var results []*v1.FileEntryInfo
	path, err := f.ResolveRealPath(dir)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() { // 文件夹和文件分开
			return entries[i].IsDir()
		}
		switch field {
		case SortByModTime: // 时间排序
			t1Info, err := entries[i].Info()
			if err != nil {
				return false
			}
			t2Info, err := entries[j].Info()
			if err != nil {
				return false
			}
			t1 := t1Info.ModTime()
			t2 := t2Info.ModTime()

			if !t1.Equal(t2) {
				if order {
					return t1.After(t2)
				}
				return t1.Before(t2)
			}

		case SortByName: // 名称排序
			fallthrough
		default:
		}

		n1 := entries[i].Name()
		n2 := entries[j].Name()

		if order {
			return n1 > n2
		}
		return n1 < n2
	})

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		fullPath := filepath.Join(path, entry.Name())
		results = append(results, toFileEntryInfo(info, fullPath))
	}
	return results, nil
}

func (f *FileOper) QueryDir(dir, keywords string, recursive bool) ([]*v1.FileEntryInfo, error) {
	if f.isSystemRoot(dir) {
		if runtime.GOOS == "windows" {
			// 盘符层级仅按盘符名过滤，不做跨盘递归（代价过大）
			res := make([]*v1.FileEntryInfo, 0)
			for _, d := range listDrives() {
				if keywords == "" || strings.Contains(d.Name, keywords) {
					res = append(res, d)
				}
			}
			return res, nil
		}
		dir = "/"
	}
	var results []*v1.FileEntryInfo

	root, err := f.ResolveRealPath(dir)
	if err != nil {
		return nil, err
	}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != root {
			name := d.Name()
			if strings.Contains(name, keywords) {
				info, err := d.Info()
				if err != nil {
					return err
				}
				results = append(results, toFileEntryInfo(info, path))
			}
		}

		// 如果不递归，跳过 root 下的所有子目录
		if !recursive && d.IsDir() && path != root {
			return filepath.SkipDir
		}

		return nil
	})
	if err != nil {
		return nil, errors.New("查询失败")
	}

	return results, err
}

func toFileEntryInfo(d fs.FileInfo, fullPath string) *v1.FileEntryInfo {
	info := &v1.FileEntryInfo{
		Name:       d.Name(),
		Path:       fullPath,
		IsDir:      d.IsDir(),
		Permission: d.Mode().Perm().String(),
		UserName:   "",
		UserId:     0,
		GroupName:  "",
		GroupId:    0,
		Size:       uint64(d.Size()),
		UpdateTime: timestamppb.New(d.ModTime()),
	}
	fillOwnerInfo(info, d)
	return info
}

func (f *FileOper) DirInfo(dir string) (*v1.FileDirectoryInfo, error) {
	if f.isSystemRoot(dir) {
		if runtime.GOOS == "windows" {
			drives := listDrives()
			return &v1.FileDirectoryInfo{
				Name:       "此电脑",
				Path:       "",
				ParentPath: "",
				DirCount:   int64(len(drives)),
				FileCount:  0,
				ItemCount:  int64(len(drives)),
			}, nil
		}
		dir = "/"
	}
	path, err := f.ResolveRealPath(dir)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var dirCount int64
	var fileCount int64
	for _, entry := range entries {
		if entry.IsDir() {
			dirCount++
			continue
		}
		fileCount++
	}

	if f.basePath != SystemPath {
		path, err = filepath.Rel(f.basePath, path)
		if err != nil {
			return nil, errors.New("获取相对路径失败")
		}
		if path != "." && !strings.HasPrefix(path, "..") {
			path = "." + string(filepath.Separator) + path
		}
	}

	return &v1.FileDirectoryInfo{
		Name:       filepath.Base(path),
		Path:       path,
		ParentPath: path,
		DirCount:   dirCount,
		FileCount:  fileCount,
		ItemCount:  int64(len(entries)),
	}, nil
}

// LoadFile 读取文件
func (f *FileOper) LoadFile(path string) ([]byte, error) {
	path, err := f.ResolveRealPath(path)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New("文件不存在")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > MaxLoadFileSize {
		return nil, errors.New("文件太大")
	}

	return io.ReadAll(file)
}

// SaveFile 覆盖保存已有文件内容。
func (f *FileOper) SaveFile(path string, content []byte) error {
	if len(content) > MaxLoadFileSize {
		return errors.New("文件太大")
	}

	absPath, err := f.ResolveRealPath(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return errors.New("文件不存在")
	}
	if info.IsDir() {
		return errors.New("目标路径不能是目录")
	}

	return os.WriteFile(absPath, content, info.Mode().Perm())
}

// BatchDelete 批量删除文件或文件夹
func (f *FileOper) BatchDelete(paths []string) []*v1.FileOperationResult {
	results := make([]*v1.FileOperationResult, 0, len(paths))
	for _, path := range paths {
		result := &v1.FileOperationResult{Path: path}
		absPath, err := f.ResolveRealPath(path)
		if err != nil {
			result.Message = err.Error()
			results = append(results, result)
			continue
		}
		if err = os.RemoveAll(absPath); err != nil {
			result.Message = err.Error()
			results = append(results, result)
			continue
		}
		result.Success = true
		results = append(results, result)
	}
	return results
}

// Create 创建文件或目录。
func (f *FileOper) Create(item *v1.FileCreateItem) error {
	absPath, err := f.ResolveRealPath(item.Path)
	if err != nil {
		return err
	}
	if item.IsDir {
		err = os.MkdirAll(absPath, 0o755)
	} else {
		if err = os.MkdirAll(filepath.Dir(absPath), 0o755); err == nil {
			err = os.WriteFile(absPath, item.Content, 0o644)
		}
	}
	return nil
}

// Rename 重命名文件或目录，目标名称不能包含路径分隔符。
func (f *FileOper) Rename(path, newName string) (string, error) {
	newName = strings.TrimSpace(newName)
	if newName == "" || newName == "." || newName == ".." {
		return "", errors.New("新名称无效")
	}
	if filepath.VolumeName(newName) != "" || strings.ContainsAny(newName, `/\`) {
		return "", errors.New("新名称不能包含路径")
	}

	sourcePath, err := f.ResolveRealPath(path)
	if err != nil {
		return "", err
	}
	targetDir, err := f.ResolveRealPath(filepath.Dir(sourcePath))
	if err != nil {
		return "", err
	}
	targetPath := filepath.Join(targetDir, newName)
	if sourcePath == targetPath {
		return targetPath, nil
	}
	if _, err = os.Stat(targetPath); err == nil {
		return "", errors.New("目标名称已存在")
	} else if !os.IsNotExist(err) {
		return "", err
	}

	if err = os.Rename(sourcePath, targetPath); err != nil {
		return "", err
	}
	return targetPath, nil
}

type ChunkedUpload struct {
	*gen.FileUpload
	fileSync sync.RWMutex
}

const (
	minChunkSize = 4 * 1024 * 1024  // 4MB
	maxChunkSize = 64 * 1024 * 1024 // 64MB
	targetChunks = 100

	tempName = "%s-momoko-upload-%v.part"
)

// TempPartPath 返回某个上传会话在目标目录下的临时分片文件路径。
// 导出以便业务层（如后台 GC）按上传记录重建该路径来清理残留 .part 文件。
func TempPartPath(dir, fileName, id string) string {
	return filepath.Join(dir, fmt.Sprintf(tempName, fileName, id))
}

func NewChunkedUpload(hash, path, name string, fileSize uint64) *ChunkedUpload {
	chunkSize := calcChunkSize(fileSize)
	return &ChunkedUpload{
		FileUpload: &gen.FileUpload{
			Hash:        hash,
			Path:        path,
			FileName:    name,
			FileSize:    fileSize,
			ChunkSize:   chunkSize,
			TotalChunks: (chunkSize + fileSize - 1) / chunkSize,
		},
		fileSync: sync.RWMutex{},
	}
}

func calcChunkSize(fileSize uint64) uint64 {
	if fileSize == 0 {
		return minChunkSize
	}
	chunkSize := fileSize / targetChunks
	if fileSize%targetChunks != 0 {
		chunkSize++
	}
	if chunkSize < minChunkSize {
		chunkSize = minChunkSize
	}
	if chunkSize > maxChunkSize {
		chunkSize = maxChunkSize
	}
	return chunkSize
}

// 上传分片
func (u *ChunkedUpload) UploadFilePart(r io.Reader, chunk uint64) (uint64, string, error) {
	u.fileSync.RLock()
	defer u.fileSync.RUnlock()

	if u.Completed {
		return 0, "", errors.New("上传已完成")
	}
	if u.Cancel {
		return 0, "", errors.New("上传已取消")
	}
	if chunk == 0 || chunk > u.TotalChunks {
		return 0, "", errors.New("异常的分片")
	}

	offset := (chunk - 1) * u.ChunkSize
	partSize := u.ChunkSize
	if chunk == u.TotalChunks {
		partSize = u.FileSize - offset
	}

	if offset > math.MaxInt64 || partSize > math.MaxInt64 {
		return 0, "", errors.New("文件过大")
	}

	tempFile, err := os.OpenFile(
		TempPartPath(u.Path, u.FileName, u.ID),
		os.O_CREATE|os.O_WRONLY,
		0o644,
	)
	if err != nil {
		return 0, "", errors.New("创建临时文件失败")
	}
	defer tempFile.Close()

	hasher := sha256.New()
	writer := io.NewOffsetWriter(tempFile, int64(offset))
	mw := io.MultiWriter(writer, hasher)

	limited := io.LimitReader(r, int64(partSize))
	written, err := io.CopyBuffer(mw, limited, make([]byte, 64*1024))
	if err != nil {
		return 0, "", errors.New("写入分片文件失败")
	}
	if uint64(written) != partSize {
		return 0, "", errors.New("文件大小异常")
	}

	var extra [1]byte
	n, err := r.Read(extra[:])
	if err != nil && err != io.EOF {
		return 0, "", err
	}
	if n > 0 {
		return 0, "", errors.New("文件大小异常")
	}

	partHash := hex.EncodeToString(hasher.Sum(nil))
	return partSize, partHash, nil
}

// Complete 校验分片完整性后将临时文件原子地落为最终文件。
// 校验项：分片齐全且编号无重复、每片大小正确、合计与临时文件实际大小均等于声明大小，
// 以避免 offset 空洞 / 短写 / 漏片导致合并出损坏文件。
func (u *ChunkedUpload) Complete() error {
	u.fileSync.Lock()
	defer u.fileSync.Unlock()
	if u.Completed {
		return nil
	}
	if u.Cancel {
		return errors.New("上传已取消")
	}

	partPath := TempPartPath(u.Path, u.FileName, u.ID)
	finalPath := filepath.Join(u.Path, u.FileName)

	// 空文件没有分片，直接落一个空文件。
	if u.TotalChunks == 0 {
		f, err := os.OpenFile(finalPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("创建文件失败: %w", err)
		}
		return f.Close()
	}

	if err := u.verifyChunks(); err != nil {
		return err
	}
	fi, err := os.Stat(partPath)
	if err != nil {
		return fmt.Errorf("临时文件缺失: %w", err)
	}
	if uint64(fi.Size()) != u.FileSize {
		return fmt.Errorf("文件大小不一致：实际 %d，声明 %d", fi.Size(), u.FileSize)
	}

	if err := os.Rename(partPath, finalPath); err != nil {
		return fmt.Errorf("重命名文件失败: %w", err)
	}
	return nil
}

// verifyChunks 校验已记录的分片：齐全、编号合法、每片大小符合预期、合计等于文件大小。
// 不校验「重复」——分片表对 (upload_id, chunk) 有唯一约束、写入用 upsert，重复上传是
// 幂等的（覆盖同偏移同内容），不应在此（或上传分片时）报错而打断客户端续传/重试。
func (u *ChunkedUpload) verifyChunks() error {
	chunks := u.Edges.Chunks
	if uint64(len(chunks)) != u.TotalChunks {
		return fmt.Errorf("分片未上传完成：%d/%d", len(chunks), u.TotalChunks)
	}
	var total uint64
	for _, c := range chunks {
		if c.Chunk < 1 || c.Chunk > u.TotalChunks {
			return errors.New("存在异常分片编号")
		}
		expected := u.ChunkSize
		if c.Chunk == u.TotalChunks {
			expected = u.FileSize - (u.TotalChunks-1)*u.ChunkSize
		}
		if c.Size != expected {
			return fmt.Errorf("分片 %d 大小异常：%d，应为 %d", c.Chunk, c.Size, expected)
		}
		total += c.Size
	}
	if total != u.FileSize {
		return fmt.Errorf("分片合计 %d 与文件大小 %d 不一致", total, u.FileSize)
	}
	return nil
}

// Canceld 取消上传并删除临时分片文件。
func (u *ChunkedUpload) Canceld() error {
	u.fileSync.Lock()
	defer u.fileSync.Unlock()
	if u.Completed {
		return errors.New("上传已完成")
	}
	if u.Cancel {
		return nil
	}
	if err := os.Remove(TempPartPath(u.Path, u.FileName, u.ID)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
