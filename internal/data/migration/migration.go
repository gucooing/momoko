// Package migration 在启动时按序号执行内置的增量 SQL 脚本。
//
// 脚本放在 sql 目录下，文件名为 <序号>_<名称>.sql（如 00_remove_sub2api.sql），
// 序号为十进制整数，按数值升序执行；已执行的脚本记录在 sql_migrations 表中，
// 下次启动只执行序号大于上次执行位置的新脚本，序号落在已执行位置之前的脚本会被跳过并告警。
//
// 脚本约定：
//   - 语句以分号分隔，支持 -- 行注释和 /* */ 块注释；
//   - 标识符用双引号（标准 SQL），MySQL 下由执行器改写为反引号；
//   - 每个脚本在一个事务内执行并写入记录，失败即回滚并中止启动
//     （MySQL 的 DDL 会隐式提交，无法回滚，脚本应尽量幂等，如 DROP TABLE IF EXISTS）。
package migration

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"entgo.io/ent/dialect"
	"github.com/go-kratos/kratos/v2/log"

	"momoko/internal/data/ent/gen"
)

//go:embed sql/*.sql
var embedded embed.FS

// Script 一个迁移脚本文件。
type Script struct {
	Version int64  // 文件名序号
	File    string // 文件名
}

var fileNameRE = regexp.MustCompile(`^(\d+)_[A-Za-z0-9_-]+\.sql$`)

// Run 执行内置 sql 目录中尚未执行的迁移脚本。driver 为 ent 方言名（sqlite3/mysql/postgres）。
func Run(ctx context.Context, client *gen.Client, driver string) error {
	sub, err := fs.Sub(embedded, "sql")
	if err != nil {
		return fmt.Errorf("migration: 读取内置脚本目录失败: %w", err)
	}
	return RunFS(ctx, client, driver, sub)
}

// RunFS 执行 fsys 根目录下尚未执行的迁移脚本。
func RunFS(ctx context.Context, client *gen.Client, driver string, fsys fs.FS) error {
	scripts, err := LoadScripts(fsys)
	if err != nil {
		return err
	}
	applied, position, err := appliedVersions(ctx, client)
	if err != nil {
		return fmt.Errorf("migration: 读取已执行记录失败: %w", err)
	}
	for _, s := range scripts {
		if _, ok := applied[s.Version]; ok {
			continue
		}
		if s.Version < position {
			log.Warnf("migration: 跳过 %s：序号小于上次执行位置 %d，请改用更大的序号", s.File, position)
			continue
		}
		content, err := fs.ReadFile(fsys, s.File)
		if err != nil {
			return fmt.Errorf("migration: 读取 %s 失败: %w", s.File, err)
		}
		stmts := SplitStatements(string(content), driver == dialect.MySQL)
		if err := apply(ctx, client, s, stmts); err != nil {
			return fmt.Errorf("migration: 执行 %s 失败: %w", s.File, err)
		}
		position = s.Version
		log.Infof("migration: 已执行 %s（%d 条语句）", s.File, len(stmts))
	}
	return nil
}

// LoadScripts 读取并校验 fsys 根目录下的脚本文件名，按序号升序返回。
func LoadScripts(fsys fs.FS) ([]Script, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("migration: 读取脚本目录失败: %w", err)
	}
	seen := make(map[int64]string, len(entries))
	scripts := make([]Script, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := fileNameRE.FindStringSubmatch(e.Name())
		if m == nil {
			return nil, fmt.Errorf("migration: 文件名 %q 不合法，应为 <序号>_<名称>.sql", e.Name())
		}
		version, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("migration: 文件名 %q 的序号无法解析: %w", e.Name(), err)
		}
		if prev, ok := seen[version]; ok {
			return nil, fmt.Errorf("migration: 序号 %d 重复：%s 与 %s", version, prev, e.Name())
		}
		seen[version] = e.Name()
		scripts = append(scripts, Script{Version: version, File: e.Name()})
	}
	sort.Slice(scripts, func(i, j int) bool { return scripts[i].Version < scripts[j].Version })
	return scripts, nil
}

// appliedVersions 返回已执行的序号集合与其中的最大值（无记录时为 -1）。
func appliedVersions(ctx context.Context, client *gen.Client) (map[int64]struct{}, int64, error) {
	records, err := client.SQLMigration.Query().All(ctx)
	if err != nil {
		return nil, 0, err
	}
	applied := make(map[int64]struct{}, len(records))
	position := int64(-1)
	for _, r := range records {
		applied[r.Version] = struct{}{}
		if r.Version > position {
			position = r.Version
		}
	}
	return applied, position, nil
}

// apply 在一个事务内执行脚本的全部语句并写入执行记录。
func apply(ctx context.Context, client *gen.Client, s Script, stmts []string) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("语句 %q: %w", abbreviate(stmt, 120), err)
		}
	}
	if _, err := tx.SQLMigration.Create().SetVersion(s.Version).SetName(s.File).Save(ctx); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("写入执行记录: %w", err)
	}
	return tx.Commit()
}

// SplitStatements 去掉注释并按分号拆分语句；mysql 为 true 时把双引号标识符改写为反引号。
// 单引号、双引号、反引号内的内容原样保留（同类引号连写两次视为转义）。
func SplitStatements(src string, mysql bool) []string {
	var (
		stmts []string
		cur   strings.Builder
	)
	flush := func() {
		if s := strings.TrimSpace(cur.String()); s != "" {
			stmts = append(stmts, s)
		}
		cur.Reset()
	}
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '-' && i+1 < len(src) && src[i+1] == '-':
			// 行注释：跳到行尾，换行符保留。
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				i = len(src)
			} else {
				i += 2 + end + 2
			}
		case c == '\'' || c == '"' || c == '`':
			j := closingQuote(src, i)
			if j < 0 {
				// 未闭合：余下内容原样写入。
				cur.WriteString(src[i:])
				i = len(src)
				break
			}
			segment := src[i : j+1]
			if c == '"' && mysql {
				inner := strings.ReplaceAll(segment[1:len(segment)-1], `""`, `"`)
				segment = "`" + strings.ReplaceAll(inner, "`", "``") + "`"
			}
			cur.WriteString(segment)
			i = j + 1
		case c == ';':
			flush()
			i++
		default:
			cur.WriteByte(c)
			i++
		}
	}
	flush()
	return stmts
}

// closingQuote 返回从 start 开始的引号段的闭合引号下标，未闭合返回 -1。
func closingQuote(src string, start int) int {
	q := src[start]
	for j := start + 1; j < len(src); j++ {
		if src[j] != q {
			continue
		}
		if j+1 < len(src) && src[j+1] == q {
			j++ // 连写两次为转义
			continue
		}
		return j
	}
	return -1
}

// abbreviate 把语句压成单行并截断到 n 个字符，用于错误信息。
func abbreviate(s string, n int) string {
	r := []rune(strings.Join(strings.Fields(s), " "))
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return string(r)
}
