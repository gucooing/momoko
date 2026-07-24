package data

import (
	"context"
	"fmt"
	"os"

	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/migrate"

	_ "github.com/glebarez/go-sqlite/compat"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/wire"
	_ "github.com/lib/pq"

	"momoko/internal/conf"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewAuthRepo,
	NewUserRepo,
	NewSystemRepo,
	NewConfigRepo,
	NewInstanceRepo,
	NewOpenSSHRepo,
	NewNetworkRepo,
	NewTunnelRepo,
	NewUserAPIKeyRepo,
	NewAPIKeyRepo,
	NewFileRepo,
	NewTaskRepo,
	NewOperationLogRepo,
	NewInitializeRepo,
	NewSub2APIRepo,
	NewImageGenRepo,
	NewOIDCRepo,
)

// Data is a struct that contains the database client.
type Data struct {
	db *gen.Client
}

// NewData creates a new Data instance.
func NewData(c *conf.Data) (*Data, func(), error) {
	db, err := gen.Open(c.Database.Driver, c.Database.Source)
	if err != nil {
		return nil, nil, err
	}
	if os.Getenv("DEPLOY_ENV") == "dev" {
		// Enable debug mode for detailed logging.
		db = db.Debug()
	}

	// auth 会话模型破坏性变更：旧表结构（token/refresh 双行 + type 枚举）直接丢弃，
	// 不做任何兼容迁移；用户需重新登录。
	if err = dropLegacyAuthTable(context.Background(), db); err != nil {
		return nil, nil, err
	}

	// 自动迁移：允许删索引/列，直接对齐最新 schema。
	if err = db.Schema.Create(
		context.Background(),
		migrate.WithDropIndex(true),
		migrate.WithDropColumn(true),
	); err != nil {
		return nil, nil, err
	}
	if err = syncDefaultRBAC(context.Background(), db); err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		db.Close()
	}
	data := &Data{
		db: db,
	}
	return data, cleanup, nil
}

// dropLegacyAuthTable 强制 DROP 旧 auth 表，避免 type 双行结构残留。
// 破坏性：所有登录会话清空，用户需重新登录。
func dropLegacyAuthTable(ctx context.Context, db *gen.Client) error {
	if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS auths"); err != nil {
		return fmt.Errorf("drop legacy auths table: %w", err)
	}
	return nil
}
