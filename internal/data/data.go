package data

import (
	"context"
	"momoko/internal/data/ent/gen"
	"os"

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
	// Run the auto migration tool.
	if err = db.Schema.Create(context.Background(), migrate.WithDropIndex(true)); err != nil {
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
