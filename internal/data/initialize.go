package data

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/conf"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/migrate"
	entuser "momoko/internal/data/ent/gen/user"
	"momoko/internal/initialize"
)

type initializeRepo struct {
	configPath string
}

var initializeConfirmMu sync.Mutex

func NewInitializeRepo(configPath string) biz.InitializeRepo {
	return &initializeRepo{
		configPath: configPath,
	}
}

func (r *initializeRepo) ConfirmInitialize(ctx context.Context, databaseType v1.DatabaseType, database *conf.Data_Database, admin *v1.InitializeAdminUser) error {
	initializeConfirmMu.Lock()
	defer initializeConfirmMu.Unlock()

	if initialize.IsInitialized() {
		return nil
	}

	if err := MigrateAndSeedInitialData(ctx, database, admin); err != nil {
		return err
	}

	if err := initialize.ApplyDatabaseConfig(r.configPath, database); err != nil {
		return err
	}
	return initialize.WriteMarker()
}

func (r *initializeRepo) TestDatabase(ctx context.Context, database *conf.Data_Database) (*biz.DatabaseProbe, error) {
	if err := TestDatabaseConnection(ctx, database); err != nil {
		return nil, err
	}
	db, err := sql.Open(database.Driver, database.Source)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return &biz.DatabaseProbe{
		ServerVersion: databaseServerVersion(ctx, db, database.Driver),
		TableCount:    databaseTableCount(ctx, db, database.Driver),
	}, nil
}

// databaseServerVersion 查询数据库服务端版本，用于初始化向导展示；查询失败返回空串。
func databaseServerVersion(ctx context.Context, db *sql.DB, driver string) string {
	var query, prefix string
	switch driver {
	case "sqlite3", "sqlite":
		query, prefix = "SELECT sqlite_version()", "SQLite "
	case "mysql":
		query, prefix = "SELECT VERSION()", "MySQL "
	case "postgres", "pgx":
		query, prefix = "SHOW server_version", "PostgreSQL "
	default:
		return ""
	}
	var v string
	if err := db.QueryRowContext(ctx, query).Scan(&v); err != nil {
		return ""
	}
	v = strings.TrimSpace(v)
	if driver == "mysql" && strings.Contains(strings.ToLower(v), "mariadb") {
		prefix = "MariaDB "
		v = strings.SplitN(v, "-MariaDB", 2)[0]
	}
	if i := strings.IndexAny(v, " ("); i > 0 && prefix == "PostgreSQL " {
		v = v[:i]
	}
	return prefix + v
}

// databaseTableCount 统计目标库里已有的数据表数量（用户表，不含系统表）；查询失败返回 0。
func databaseTableCount(ctx context.Context, db *sql.DB, driver string) int64 {
	var query string
	switch driver {
	case "sqlite3", "sqlite":
		query = "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'"
	case "mysql":
		query = "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE()"
	case "postgres", "pgx":
		query = "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema()"
	default:
		return 0
	}
	var n int64
	if err := db.QueryRowContext(ctx, query).Scan(&n); err != nil {
		return 0
	}
	return n
}

func TestDatabaseConnection(ctx context.Context, database *conf.Data_Database) error {
	if database == nil {
		return fmt.Errorf("database config is required")
	}
	if err := os.MkdirAll(initialize.DataDir, 0o755); err != nil {
		return err
	}

	db, err := sql.Open(database.Driver, database.Source)
	if err != nil {
		return err
	}
	defer db.Close()

	return db.PingContext(ctx)
}

func MigrateAndSeedInitialData(ctx context.Context, database *conf.Data_Database, admin *v1.InitializeAdminUser) error {
	if database == nil {
		return fmt.Errorf("database config is required")
	}
	if admin == nil {
		return fmt.Errorf("admin user is required")
	}
	if err := os.MkdirAll(initialize.DataDir, 0o755); err != nil {
		return err
	}

	db, err := gen.Open(database.Driver, database.Source)
	if err != nil {
		return err
	}
	if os.Getenv("DEPLOY_ENV") == "dev" {
		db = db.Debug()
	}
	defer db.Close()

	if err := db.Schema.Create(ctx, migrate.WithDropIndex(true)); err != nil {
		return err
	}

	return syncDefaultRBACWithUsers(ctx, db, []*defaultUser{
		{
			ID:       "admin_1",
			Username: strings.TrimSpace(admin.Username),
			Password: admin.Password,
			Email:    strings.TrimSpace(admin.Email),
			Status:   entuser.StatusActive,
			Avatar:   "",
			Bio:      "",
			Name:     strings.TrimSpace(admin.Name),
			Tags:     "",
			RoleID:   adminPermissionRoleID,
		},
	})
}
