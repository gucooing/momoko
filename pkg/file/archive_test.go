package file

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestFileOper_BatchCompressAndUnzip(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := filepath.Join(baseDir, "source")
	nestedDir := filepath.Join(sourceDir, "nested")

	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "a.txt"), []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "b.txt"), []byte("bravo"), 0o644); err != nil {
		t.Fatal(err)
	}

	oper, err := NewFileOper(SystemPath)
	if err != nil {
		t.Fatal(err)
	}

	archivePath, err := oper.BatchCompress([]string{sourceDir}, "")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(archivePath) != ".zip" {
		t.Fatalf("unexpected archive path: %s", archivePath)
	}
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatal(err)
	}

	extractDir, err := oper.Unzip(archivePath, "")
	if err != nil {
		t.Fatal(err)
	}

	rootName := filepath.Base(sourceDir)
	contentA, err := os.ReadFile(filepath.Join(extractDir, rootName, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contentA) != "alpha" {
		t.Fatalf("unexpected content: %s", string(contentA))
	}

	contentB, err := os.ReadFile(filepath.Join(extractDir, rootName, "nested", "b.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contentB) != "bravo" {
		t.Fatalf("unexpected content: %s", string(contentB))
	}
}

func TestFileOper_UnzipRejectsTraversalEntry(t *testing.T) {
	baseDir := t.TempDir()
	archivePath := filepath.Join(baseDir, "bad.zip")

	archiveFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	writer := zip.NewWriter(archiveFile)
	header := &zip.FileHeader{Name: "../escape.txt", Method: zip.Deflate}
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("escape")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archiveFile.Close(); err != nil {
		t.Fatal(err)
	}

	oper, err := NewFileOper(SystemPath)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := oper.Unzip(archivePath, filepath.Join(baseDir, "out")); err == nil {
		t.Fatal("expected unzip traversal error")
	}
}
