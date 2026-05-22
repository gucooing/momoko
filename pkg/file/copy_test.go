package file

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileOper_CopyToDirCopiesFileAndDirectory(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := filepath.Join(baseDir, "source")
	targetDir := filepath.Join(baseDir, "target")
	nestedDir := filepath.Join(sourceDir, "nested")

	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "a.txt"), []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "b.txt"), []byte("bravo"), 0o644); err != nil {
		t.Fatal(err)
	}

	oper, err := NewFileOper(baseDir)
	if err != nil {
		t.Fatal(err)
	}

	results := oper.CopyToDir([]string{"a.txt", "source"}, "target")
	if len(results) != 2 {
		t.Fatalf("unexpected result length: %d", len(results))
	}
	for _, result := range results {
		if !result.Success {
			t.Fatalf("copy failed for %s: %s", result.Path, result.Message)
		}
	}

	contentA, err := os.ReadFile(filepath.Join(targetDir, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contentA) != "alpha" {
		t.Fatalf("unexpected content: %s", string(contentA))
	}

	contentB, err := os.ReadFile(filepath.Join(targetDir, "source", "nested", "b.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contentB) != "bravo" {
		t.Fatalf("unexpected content: %s", string(contentB))
	}
}

func TestFileOper_CopyToDirSameDirectoryUsesCopySuffix(t *testing.T) {
	baseDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(baseDir, "a.txt"), []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}

	oper, err := NewFileOper(baseDir)
	if err != nil {
		t.Fatal(err)
	}

	results := oper.CopyToDir([]string{"a.txt"}, ".")
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("copy failed: %+v", results)
	}

	content, err := os.ReadFile(filepath.Join(baseDir, "a-copy.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "alpha" {
		t.Fatalf("unexpected content: %s", string(content))
	}
}

func TestFileOper_CopyToDirRejectsCopyIntoOwnChild(t *testing.T) {
	baseDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(baseDir, "source", "child"), 0o755); err != nil {
		t.Fatal(err)
	}

	oper, err := NewFileOper(baseDir)
	if err != nil {
		t.Fatal(err)
	}

	results := oper.CopyToDir([]string{"source"}, filepath.Join("source", "child"))
	if len(results) != 1 {
		t.Fatalf("unexpected result length: %d", len(results))
	}
	if results[0].Success {
		t.Fatal("expected copy into own child to fail")
	}
}

func TestFileOper_MoveToDirMovesFile(t *testing.T) {
	baseDir := t.TempDir()
	targetDir := filepath.Join(baseDir, "target")
	sourcePath := filepath.Join(baseDir, "a.txt")

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}

	oper, err := NewFileOper(baseDir)
	if err != nil {
		t.Fatal(err)
	}

	results := oper.MoveToDir([]string{"a.txt"}, "target")
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("move failed: %+v", results)
	}
	if _, err := os.Stat(sourcePath); !os.IsNotExist(err) {
		t.Fatalf("expected source to be removed, got err=%v", err)
	}

	content, err := os.ReadFile(filepath.Join(targetDir, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "alpha" {
		t.Fatalf("unexpected content: %s", string(content))
	}
}
