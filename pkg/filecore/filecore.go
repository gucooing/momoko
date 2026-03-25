package filecore

import (
	"errors"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io/fs"
	"momoko/api/gen/v1"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var ErrPathOutOfRoot = errors.New("路径超出限制范围")

// Instance 表示一个带根路径限制的文件操作实例。
// 根路径为空时表示不限制访问范围。
type Instance struct {
	rootPath string
}

// ListOptions 表示获取文件列表时的筛选与分页条件。
type ListOptions struct {
	Path          string
	Page          int64
	PageSize      int64
	Keywords      string
	IncludeSubDir bool
	SortField     v1.FileSortField
	Desc          bool
}

// ListResult 表示文件列表结果。
type ListResult struct {
	Directory *v1.FileDirectoryInfo
	Items     []*v1.FileEntryInfo
	Page      int64
	PageSize  int64
	Total     int64
}

// SizeResult 表示批量计算大小结果。
type SizeResult struct {
	Path    string
	Size    uint64
	Success bool
	Message string
}

// CreateItem 表示批量创建项。
type CreateItem struct {
	Path    string
	IsDir   bool
	Content []byte
}

// OperationResult 表示批量文件操作结果。
type OperationResult struct {
	Path    string
	Success bool
	Message string
}

// FileContent 表示打开文件后的内容与元信息。
type FileContent struct {
	Info    *v1.FileEntryInfo
	Content []byte
}

type fileIdentity struct {
	userName  string
	userID    uint32
	groupName string
	groupID   uint32
}

// New 创建一个文件操作实例。
func New(rootPath string) (*Instance, error) {
	normalizedRoot, err := normalizeRoot(rootPath)
	if err != nil {
		return nil, err
	}
	return &Instance{rootPath: normalizedRoot}, nil
}

// RootPath 返回当前实例的根路径限制。
func (i *Instance) RootPath() string {
	if i == nil {
		return ""
	}
	return i.rootPath
}

// Limited 返回当前实例是否受根路径限制。
func (i *Instance) Limited() bool {
	return i != nil && i.rootPath != ""
}

// List 获取指定目录下的文件列表。
func (i *Instance) List(opts ListOptions) (*ListResult, error) {
	dirPath, err := i.resolvePath(opts.Path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("目标路径不是目录")
	}

	directory, err := i.buildDirectoryInfo(dirPath)
	if err != nil {
		return nil, err
	}

	items, err := i.collectEntries(dirPath, strings.TrimSpace(opts.Keywords), opts.IncludeSubDir)
	if err != nil {
		return nil, err
	}
	sortEntries(items, opts.SortField, opts.Desc)

	page := opts.Page
	if page <= 0 {
		page = 1
	}

	total := int64(len(items))
	pageSize := opts.PageSize
	if pageSize <= 0 || pageSize > total && total > 0 {
		pageSize = total
	}
	if total == 0 {
		page = 1
		pageSize = 0
	}

	pagedItems := items
	if pageSize > 0 {
		start := (page - 1) * pageSize
		if start >= total {
			pagedItems = []*v1.FileEntryInfo{}
		} else {
			end := start + pageSize
			if end > total {
				end = total
			}
			pagedItems = items[start:end]
		}
	}

	return &ListResult{
		Directory: directory,
		Items:     pagedItems,
		Page:      page,
		PageSize:  pageSize,
		Total:     total,
	}, nil
}

// OpenFolder 打开指定文件夹，本质上返回该目录下的全部列表。
func (i *Instance) OpenFolder(path string) (*ListResult, error) {
	return i.List(ListOptions{Path: path})
}

// OpenFile 打开指定文件并返回内容。
func (i *Instance) OpenFile(path string) (*FileContent, error) {
	absPath, err := i.resolvePath(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, errors.New("目标路径不是文件")
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	entry, err := buildEntryInfo(absPath, info)
	if err != nil {
		return nil, err
	}
	return &FileContent{
		Info:    entry,
		Content: content,
	}, nil
}

// BatchCalcSize 批量计算指定路径的大小。
func (i *Instance) BatchCalcSize(paths []string) []SizeResult {
	results := make([]SizeResult, 0, len(paths))
	for _, path := range paths {
		result := SizeResult{Path: path}
		absPath, err := i.resolvePath(path)
		if err != nil {
			result.Message = err.Error()
			results = append(results, result)
			continue
		}

		size, err := calcPathSize(absPath)
		if err != nil {
			result.Message = err.Error()
			results = append(results, result)
			continue
		}

		result.Size = size
		result.Success = true
		results = append(results, result)
	}
	return results
}

// BatchCreate 批量创建文件或目录。
func (i *Instance) BatchCreate(items []CreateItem) []OperationResult {
	results := make([]OperationResult, 0, len(items))
	for _, item := range items {
		result := OperationResult{Path: item.Path}
		absPath, err := i.resolvePath(item.Path)
		if err != nil {
			result.Message = err.Error()
			results = append(results, result)
			continue
		}

		if item.IsDir {
			err = os.MkdirAll(absPath, 0o755)
		} else {
			if err = os.MkdirAll(filepath.Dir(absPath), 0o755); err == nil {
				err = os.WriteFile(absPath, item.Content, 0o644)
			}
		}
		if err != nil {
			result.Message = err.Error()
			results = append(results, result)
			continue
		}

		result.Success = true
		results = append(results, result)
	}
	return results
}

// BatchDelete 批量删除文件或目录。
func (i *Instance) BatchDelete(paths []string) []OperationResult {
	results := make([]OperationResult, 0, len(paths))
	for _, path := range paths {
		result := OperationResult{Path: path}
		absPath, err := i.resolvePath(path)
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

func normalizeRoot(rootPath string) (string, error) {
	rootPath = strings.TrimSpace(rootPath)
	if rootPath == "" {
		return "", nil
	}

	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absPath), nil
}

// 路径检查
func (i *Instance) resolvePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if i.rootPath == "" {
		return absPath, nil
	}

	relPath, err := filepath.Rel(i.rootPath, absPath)
	if err != nil {
		return "", err
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(os.PathSeparator)) {
		return "", ErrPathOutOfRoot
	}
	return absPath, nil
}

func (i *Instance) buildDirectoryInfo(dirPath string) (*v1.FileDirectoryInfo, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return &v1.FileDirectoryInfo{}, err
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

	size, err := calcPathSize(dirPath)
	if err != nil {
		return &v1.FileDirectoryInfo{}, err
	}

	parentPath := filepath.Dir(dirPath)
	if parentPath == dirPath {
		parentPath = ""
	}

	return &v1.FileDirectoryInfo{
		Name:       filepath.Base(dirPath),
		Path:       dirPath,
		ParentPath: parentPath,
		Size:       size,
		DirCount:   dirCount,
		FileCount:  fileCount,
		ItemCount:  int64(len(entries)),
	}, nil
}

func (i *Instance) collectEntries(dirPath, keywords string, includeSubDir bool) ([]*v1.FileEntryInfo, error) {
	if includeSubDir {
		return i.walkEntries(dirPath, keywords)
	}

	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	items := make([]*v1.FileEntryInfo, 0, len(dirEntries))
	for _, dirEntry := range dirEntries {
		fullPath := filepath.Join(dirPath, dirEntry.Name())
		if keywords != "" && !strings.Contains(strings.ToLower(dirEntry.Name()), strings.ToLower(keywords)) {
			continue
		}

		info, err := dirEntry.Info()
		if err != nil {
			return nil, err
		}

		item, err := buildEntryInfo(fullPath, info)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (i *Instance) walkEntries(rootPath, keywords string) ([]*v1.FileEntryInfo, error) {
	items := make([]*v1.FileEntryInfo, 0)
	lowerKeywords := strings.ToLower(keywords)

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == rootPath {
			return nil
		}
		if lowerKeywords != "" && !strings.Contains(strings.ToLower(d.Name()), lowerKeywords) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		item, err := buildEntryInfo(path, info)
		if err != nil {
			return err
		}
		items = append(items, item)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}

func buildEntryInfo(path string, info os.FileInfo) (*v1.FileEntryInfo, error) {
	identity := lookupFileIdentity(info)

	var size uint64
	if !info.IsDir() && info.Size() > 0 {
		size = uint64(info.Size())
	}

	return &v1.FileEntryInfo{
		Name:       info.Name(),
		Path:       path,
		IsDir:      info.IsDir(),
		Permission: fmt.Sprintf("%04o", info.Mode().Perm()),
		UserName:   identity.userName,
		UserId:     identity.userID,
		GroupName:  identity.groupName,
		GroupId:    identity.groupID,
		Size:       size,
		UpdateTime: timestamppb.New(info.ModTime()),
	}, nil
}

func calcPathSize(path string) (uint64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		if info.Size() <= 0 {
			return 0, nil
		}
		return uint64(info.Size()), nil
	}

	var total uint64
	err = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || info.Size() <= 0 {
			return nil
		}
		total += uint64(info.Size())
		return nil
	})
	return total, err
}

func sortEntries(items []*v1.FileEntryInfo, field v1.FileSortField, desc bool) {
	sort.Slice(items, func(i, j int) bool {
		less := compareEntry(items[i], items[j], field)
		if desc {
			return !less
		}
		return less
	})
}

func compareEntry(left, right *v1.FileEntryInfo, field v1.FileSortField) bool {
	switch field {
	case v1.FileSortField_FILE_SORT_FIELD_SIZE:
		if left.Size != right.Size {
			return left.Size < right.Size
		}
	case v1.FileSortField_FILE_SORT_FIELD_UPDATE_TIME:
		if !left.UpdateTime.AsTime().Equal(right.UpdateTime.AsTime()) {
			return left.UpdateTime.AsTime().Before(right.UpdateTime.AsTime())
		}
	default:
	}

	leftName := strings.ToLower(left.Name)
	rightName := strings.ToLower(right.Name)
	if leftName != rightName {
		return leftName < rightName
	}
	return left.Path < right.Path
}
