package filecore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestInstanceRejectsPathOutOfRoot(t *testing.T) {
	rootDir := t.TempDir()
	operator, err := New(rootDir)
	if err != nil {
		t.Fatalf("创建文件实例失败: %v", err)
	}

	_, err = operator.List(ListOptions{Path: filepath.Join(rootDir, "..")})
	if !errors.Is(err, ErrPathOutOfRoot) {
		t.Fatalf("超出根路径限制时错误不正确: %v", err)
	}
}

func TestInstanceListAndOpenFile(t *testing.T) {
	rootDir := t.TempDir()
	operator, err := New(rootDir)
	if err != nil {
		t.Fatalf("创建文件实例失败: %v", err)
	}

	createResults := operator.BatchCreate([]CreateItem{
		{Path: filepath.Join(rootDir, "logs"), IsDir: true},
		{Path: filepath.Join(rootDir, "readme.txt"), Content: []byte("hello")},
	})
	for _, result := range createResults {
		if !result.Success {
			t.Fatalf("创建测试文件失败: %s", result.Message)
		}
	}

	listResult, err := operator.List(ListOptions{Path: rootDir, SortField: SortFieldName})
	if err != nil {
		t.Fatalf("获取文件列表失败: %v", err)
	}
	if listResult.Total != 2 {
		t.Fatalf("文件列表数量不正确，期望 2，实际 %d", listResult.Total)
	}
	t.Logf("temp info: %v", listResult)

	content, err := operator.OpenFile(filepath.Join(rootDir, "readme.txt"))
	if err != nil {
		t.Fatalf("打开文件失败: %v", err)
	}
	if string(content.Content) != "hello" {
		t.Fatalf("文件内容不正确，期望 hello，实际 %s", string(content.Content))
	}
}

func TestInstanceBatchCalcSizeAndDelete(t *testing.T) {
	rootDir := t.TempDir()
	operator, err := New(rootDir)
	if err != nil {
		t.Fatalf("创建文件实例失败: %v", err)
	}

	filePath := filepath.Join(rootDir, "data.txt")
	if err = os.WriteFile(filePath, []byte("abc"), 0o644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}

	sizeResults := operator.BatchCalcSize([]string{filePath})
	if len(sizeResults) != 1 || !sizeResults[0].Success || sizeResults[0].Size != 3 {
		t.Fatalf("批量计算大小结果不正确: %+v", sizeResults)
	}

	deleteResults := operator.BatchDelete([]string{filePath})
	if len(deleteResults) != 1 || !deleteResults[0].Success {
		t.Fatalf("批量删除结果不正确: %+v", deleteResults)
	}

	if _, err = os.Stat(filePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("文件删除后仍然存在: %v", err)
	}
}
