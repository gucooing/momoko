// Package sqlitesnap 为正在被其他进程使用的 SQLite 数据库做一致性快照（VACUUM INTO），
// WAL 模式下不阻塞写者；快照是一个独立的、不带 -wal / -shm 的完整数据库文件。
package sqlitesnap

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/glebarez/go-sqlite" // 注册 sqlite 驱动
)

// Snapshot 把 src 数据库快照到 dst（dst 已存在时先删除）。
func Snapshot(ctx context.Context, src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	_ = os.Remove(dst)
	abs, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	// 只读取不写入；忙时等待写者（与游戏服务端的 busy_timeout 同义）。
	// 不用 mode=ro：只读连接在 WAL 模式下可能因无法写 -shm 而打不开。
	dsn := filepath.ToSlash(abs) + "?_pragma=" + url.QueryEscape("busy_timeout(10000)")
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	target := strings.ReplaceAll(dst, "'", "''")
	if _, err := db.ExecContext(ctx, "VACUUM INTO '"+target+"'"); err != nil {
		_ = os.Remove(dst)
		return err
	}
	return nil
}

// Companions 返回 SQLite 数据库的伴随文件（-wal / -shm / -journal）。
func Companions(path string) []string {
	return []string{path + "-wal", path + "-shm", path + "-journal"}
}
