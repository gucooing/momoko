package file

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/localfs"
)

// toEntryInfo 把 localfs 条目映射为对外的 api 结构。
func toEntryInfo(e *localfs.Entry) *v1.FileEntryInfo {
	if e == nil {
		return nil
	}
	return &v1.FileEntryInfo{
		Name:       e.Name,
		Path:       e.Path,
		IsDir:      e.IsDir,
		Permission: e.Perm(),
		UserName:   e.User,
		UserId:     e.UID,
		GroupName:  e.Group,
		GroupId:    e.GID,
		Size:       uint64(max(e.Size, 0)),
		UpdateTime: timestamppb.New(e.ModTime),
	}
}

func toEntryInfos(items []*localfs.Entry) []*v1.FileEntryInfo {
	out := make([]*v1.FileEntryInfo, 0, len(items))
	for _, e := range items {
		out = append(out, toEntryInfo(e))
	}
	return out
}

func toDirectoryInfo(st *localfs.DirStat) *v1.FileDirectoryInfo {
	if st == nil {
		return nil
	}
	return &v1.FileDirectoryInfo{
		Name:       st.Name,
		Path:       st.Path,
		ParentPath: st.ParentPath,
		DirCount:   st.Dirs,
		FileCount:  st.Files,
		ItemCount:  st.Items,
	}
}

// toOperationResults 把 localfs 批量结果映射为对外结构。
func toOperationResults(items []localfs.Result) []*v1.FileOperationResult {
	out := make([]*v1.FileOperationResult, 0, len(items))
	for _, r := range items {
		out = append(out, &v1.FileOperationResult{Path: r.Path, Success: r.OK, Message: r.Message})
	}
	return out
}

// toSortField 把 api 排序枚举映射为 localfs 排序字段。
func toSortField(f v1.FileSortField) localfs.SortField {
	switch f {
	case SortByModTime:
		return localfs.SortByModTime
	default:
		return localfs.SortByName
	}
}

// errResults 为每个路径生成一条相同错误的结果（用于整批前置校验失败）。
func errResults(paths []string, err error) []*v1.FileOperationResult {
	out := make([]*v1.FileOperationResult, 0, len(paths))
	for _, p := range paths {
		out = append(out, &v1.FileOperationResult{Path: p, Message: err.Error()})
	}
	return out
}
