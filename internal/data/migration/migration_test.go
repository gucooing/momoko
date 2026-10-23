package migration

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	_ "github.com/glebarez/go-sqlite/compat"

	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/enttest"
	"momoko/internal/data/ent/gen/sqlmigration"
	"momoko/internal/data/ent/gen/systemconfig"
	"momoko/internal/data/ent/gen/task"
)

// newClient 用临时文件 SQLite 建库并执行 ent 自动迁移，与生产启动路径一致。
func newClient(t *testing.T) *gen.Client {
	t.Helper()
	path := filepath.ToSlash(filepath.Join(t.TempDir(), "test.db"))
	client := enttest.Open(t, "sqlite3", "file:"+path+"?_pragma=foreign_keys(1)")
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func sqlFS(files map[string]string) fstest.MapFS {
	m := fstest.MapFS{}
	for name, content := range files {
		m[name] = &fstest.MapFile{Data: []byte(content)}
	}
	return m
}

func tableExists(t *testing.T, client *gen.Client, name string) bool {
	t.Helper()
	rows, err := client.QueryContext(context.Background(), "SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?", name)
	if err != nil {
		t.Fatalf("查询 sqlite_master 失败: %v", err)
	}
	defer rows.Close()
	return rows.Next()
}

func countRows(t *testing.T, client *gen.Client, table string) int {
	t.Helper()
	rows, err := client.QueryContext(context.Background(), `SELECT COUNT(*) FROM "`+table+`"`)
	if err != nil {
		t.Fatalf("统计 %s 失败: %v", table, err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatalf("统计 %s 无结果", table)
	}
	var n int
	if err := rows.Scan(&n); err != nil {
		t.Fatalf("读取 %s 计数失败: %v", table, err)
	}
	return n
}

func assertApplied(t *testing.T, client *gen.Client, want []int64) {
	t.Helper()
	records := client.SQLMigration.Query().Order(gen.Asc(sqlmigration.FieldVersion)).AllX(context.Background())
	got := make([]int64, 0, len(records))
	for _, r := range records {
		got = append(got, r.Version)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("已执行记录 = %v, 期望 %v", got, want)
	}
}

func TestLoadScripts(t *testing.T) {
	scripts, err := LoadScripts(sqlFS(map[string]string{"10_c.sql": "", "00_a.sql": "", "1_b.sql": ""}))
	if err != nil {
		t.Fatal(err)
	}
	got := make([]int64, 0, len(scripts))
	for _, s := range scripts {
		got = append(got, s.Version)
	}
	if want := []int64{0, 1, 10}; !slices.Equal(got, want) {
		t.Fatalf("序号顺序 = %v, 期望 %v", got, want)
	}
	if _, err := LoadScripts(sqlFS(map[string]string{"00-a.sql": ""})); err == nil {
		t.Fatal("文件名不合法应报错")
	}
	if _, err := LoadScripts(sqlFS(map[string]string{"01_a.sql": "", "1_b.sql": ""})); err == nil {
		t.Fatal("序号重复应报错")
	}
}

func TestSplitStatements(t *testing.T) {
	src := "-- 行注释; 不是语句\n" +
		"CREATE TABLE \"t\" (id INTEGER, \"key\" TEXT); /* 块注释; */\n" +
		"INSERT INTO \"t\" VALUES (1, 'a;b'); INSERT INTO \"t\" VALUES (2, 'it''s \"quoted\"');\n" +
		"SELECT \"a\"\"b\" FROM \"t\"\n"
	want := []string{
		`CREATE TABLE "t" (id INTEGER, "key" TEXT)`,
		`INSERT INTO "t" VALUES (1, 'a;b')`,
		`INSERT INTO "t" VALUES (2, 'it''s "quoted"')`,
		`SELECT "a""b" FROM "t"`,
	}
	if got := SplitStatements(src, false); !slices.Equal(got, want) {
		t.Fatalf("通用拆分 = %q, 期望 %q", got, want)
	}
	wantMySQL := []string{
		"CREATE TABLE `t` (id INTEGER, `key` TEXT)",
		"INSERT INTO `t` VALUES (1, 'a;b')",
		"INSERT INTO `t` VALUES (2, 'it''s \"quoted\"')",
		"SELECT `a\"b` FROM `t`",
	}
	if got := SplitStatements(src, true); !slices.Equal(got, wantMySQL) {
		t.Fatalf("MySQL 拆分 = %q, 期望 %q", got, wantMySQL)
	}
}

func TestRunFS(t *testing.T) {
	ctx := context.Background()
	client := newClient(t)
	files := sqlFS(map[string]string{
		"00_create.sql": `CREATE TABLE "demo" ("id" INTEGER PRIMARY KEY, "name" TEXT);`,
		"02_insert.sql": `INSERT INTO "demo" ("name") VALUES ('a'); INSERT INTO "demo" ("name") VALUES ('b');`,
	})
	if err := RunFS(ctx, client, "sqlite3", files); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, client, "demo"); n != 2 {
		t.Fatalf("demo 行数 = %d, 期望 2", n)
	}
	assertApplied(t, client, []int64{0, 2})

	// 再次执行：已记录的脚本不会重跑。
	if err := RunFS(ctx, client, "sqlite3", files); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, client, "demo"); n != 2 {
		t.Fatalf("重复执行后 demo 行数 = %d, 期望 2", n)
	}

	// 新增序号更大的脚本只执行它；序号落在已执行位置之前的脚本被跳过。
	files["01_late.sql"] = &fstest.MapFile{Data: []byte(`INSERT INTO "demo" ("name") VALUES ('late');`)}
	files["03_more.sql"] = &fstest.MapFile{Data: []byte(`INSERT INTO "demo" ("name") VALUES ('c');`)}
	if err := RunFS(ctx, client, "sqlite3", files); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, client, "demo"); n != 3 {
		t.Fatalf("增量执行后 demo 行数 = %d, 期望 3", n)
	}
	assertApplied(t, client, []int64{0, 2, 3})
}

func TestRunFSRollbackOnFailure(t *testing.T) {
	ctx := context.Background()
	client := newClient(t)
	files := sqlFS(map[string]string{
		"00_bad.sql": `CREATE TABLE "demo" ("id" INTEGER); INSERT INTO "missing" VALUES (1);`,
	})
	err := RunFS(ctx, client, "sqlite3", files)
	if err == nil {
		t.Fatal("语句失败应返回错误")
	}
	if !strings.Contains(err.Error(), "00_bad.sql") {
		t.Fatalf("错误信息应包含脚本名: %v", err)
	}
	assertApplied(t, client, []int64{})
	// SQLite 的 DDL 可回滚：失败脚本里的建表也应被撤销。
	if tableExists(t, client, "demo") {
		t.Fatal("失败脚本的建表应被回滚")
	}
}

func TestEmbeddedRemoveSub2API(t *testing.T) {
	ctx := context.Background()
	client := newClient(t)
	// 模拟移除 sub2api 前的旧库：残留表（含外键）、配置键和任务记录。
	legacy := []string{
		`CREATE TABLE sub2api_groups (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE sub2api_usage_records (id INTEGER PRIMARY KEY, group_id INTEGER REFERENCES sub2api_groups(id) ON DELETE SET NULL)`,
		`CREATE TABLE sub2api_lottery_rounds (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE sub2api_lottery_participants (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE sub2api_announcements (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE sub2api_timeline_items (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE image_gen_generations (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE image_gen_images (id INTEGER PRIMARY KEY)`,
		`INSERT INTO sub2api_groups (id, name) VALUES (1, 'g')`,
		`INSERT INTO sub2api_usage_records (id, group_id) VALUES (1, 1)`,
	}
	for _, stmt := range legacy {
		if _, err := client.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("准备旧表失败 %q: %v", stmt, err)
		}
	}
	client.SystemConfig.Create().SetKey("sub2api.admin_api_key").SetValue("secret").SaveX(ctx)
	client.SystemConfig.Create().SetKey("sub2api.title").SetValue("Sub2API").SaveX(ctx)
	client.SystemConfig.Create().SetKey("system.title").SetValue("momoko").SaveX(ctx)
	client.Task.Create().SetID("sub2api.lottery.tick").SetType("sub2api.lottery.tick").SetKind(task.KindScheduled).SaveX(ctx)
	client.Task.Create().SetID("file.gc").SetType("file.gc").SetKind(task.KindScheduled).SaveX(ctx)

	if err := Run(ctx, client, "sqlite3"); err != nil {
		t.Fatal(err)
	}

	for _, table := range []string{
		"sub2api_usage_records", "sub2api_groups", "sub2api_lottery_participants", "sub2api_lottery_rounds",
		"sub2api_announcements", "sub2api_timeline_items", "image_gen_images", "image_gen_generations",
	} {
		if tableExists(t, client, table) {
			t.Errorf("表 %s 应已删除", table)
		}
	}
	if n := client.SystemConfig.Query().Where(systemconfig.KeyHasPrefix("sub2api.")).CountX(ctx); n != 0 {
		t.Errorf("sub2api.* 配置键剩余 %d 条, 期望 0", n)
	}
	if n := client.SystemConfig.Query().CountX(ctx); n != 1 {
		t.Errorf("其他配置键剩余 %d 条, 期望 1", n)
	}
	if n := client.Task.Query().Where(task.TypeHasPrefix("sub2api.")).CountX(ctx); n != 0 {
		t.Errorf("sub2api.* 任务记录剩余 %d 条, 期望 0", n)
	}
	if n := client.Task.Query().CountX(ctx); n != 1 {
		t.Errorf("其他任务记录剩余 %d 条, 期望 1", n)
	}
	assertApplied(t, client, []int64{0})

	// 再次执行为空操作。
	if err := Run(ctx, client, "sqlite3"); err != nil {
		t.Fatal(err)
	}
	assertApplied(t, client, []int64{0})
}
