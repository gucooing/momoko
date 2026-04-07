package biz

import (
	"context"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/filecore"
)

// FileUsecase 提供系统级文件操作能力。
type FileUsecase struct{}

// NewFileUsecase 创建文件操作用例。
func NewFileUsecase() *FileUsecase {
	return &FileUsecase{}
}

func (f *FileUsecase) newSystemInstance() (*filecore.Instance, error) {
	return filecore.New("")
}

// GetFileSystemList 获取系统文件列表。
func (f *FileUsecase) GetFileSystemList(ctx context.Context, req *v1.GetFileSystemListRequest) (*v1.GetFileSystemListResponse, error) {
	_ = ctx

	instance, err := f.newSystemInstance()
	if err != nil {
		return nil, ErrSystem(err)
	}

	result, err := instance.List(filecore.ListOptions{
		Path:          req.Path,
		Page:          req.Page,
		PageSize:      req.PageSize,
		Keywords:      req.GetKeywords(),
		IncludeSubDir: req.GetIncludeSubDir(),
		SortField:     req.SortField,
		Desc:          req.IsDesc,
	})
	if err != nil {
		return nil, ErrSystem(err)
	}

	return &v1.GetFileSystemListResponse{
		Directory: result.Directory,
		Items:     result.Items,
		Page:      result.Page,
		PageSize:  result.PageSize,
		Total:     result.Total,
	}, nil
}

// BatchCalcFileSystemSize 批量计算文件大小。
func (f *FileUsecase) BatchCalcFileSystemSize(ctx context.Context, req *v1.BatchCalcFileSystemSizeRequest) (*v1.BatchCalcFileSystemSizeResponse, error) {
	_ = ctx

	instance, err := f.newSystemInstance()
	if err != nil {
		return nil, ErrSystem(err)
	}

	items := instance.BatchCalcSize(req.Paths)
	out := make([]*v1.FileSizeResult, 0, len(items))
	for _, item := range items {
		out = append(out, &v1.FileSizeResult{
			Path:    item.Path,
			Size:    item.Size,
			Success: item.Success,
			Message: item.Message,
		})
	}
	return &v1.BatchCalcFileSystemSizeResponse{Items: out}, nil
}

// BatchDeleteFileSystem 批量删除文件。
func (f *FileUsecase) BatchDeleteFileSystem(ctx context.Context, req *v1.BatchDeleteFileSystemRequest) (*v1.BatchDeleteFileSystemResponse, error) {
	_ = ctx

	instance, err := f.newSystemInstance()
	if err != nil {
		return nil, ErrSystem(err)
	}

	items := instance.BatchDelete(req.Paths)
	out := make([]*v1.FileOperationResult, 0, len(items))
	for _, item := range items {
		out = append(out, &v1.FileOperationResult{
			Path:    item.Path,
			Success: item.Success,
			Message: item.Message,
		})
	}
	return &v1.BatchDeleteFileSystemResponse{Items: out}, nil
}

// BatchCreateFileSystem 批量创建文件。
func (f *FileUsecase) BatchCreateFileSystem(ctx context.Context, req *v1.BatchCreateFileSystemRequest) (*v1.BatchCreateFileSystemResponse, error) {
	_ = ctx

	instance, err := f.newSystemInstance()
	if err != nil {
		return nil, ErrSystem(err)
	}

	createItems := make([]filecore.CreateItem, 0, len(req.Items))
	for _, item := range req.Items {
		createItems = append(createItems, filecore.CreateItem{
			Path:    item.Path,
			IsDir:   item.IsDir,
			Content: []byte(item.Content),
		})
	}

	items := instance.BatchCreate(createItems)
	out := make([]*v1.FileOperationResult, 0, len(items))
	for _, item := range items {
		out = append(out, &v1.FileOperationResult{
			Path:    item.Path,
			Success: item.Success,
			Message: item.Message,
		})
	}
	return &v1.BatchCreateFileSystemResponse{Items: out}, nil
}

// OpenFileSystemFile 打开文件并返回内容。
func (f *FileUsecase) OpenFileSystemFile(ctx context.Context, req *v1.OpenFileSystemFileRequest) (*v1.OpenFileSystemFileResponse, error) {
	_ = ctx

	instance, err := f.newSystemInstance()
	if err != nil {
		return nil, ErrSystem(err)
	}

	content, err := instance.OpenFile(req.Path)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.OpenFileSystemFileResponse{Info: content.Content}, nil
}
