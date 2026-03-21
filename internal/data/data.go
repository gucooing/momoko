package data

import (
	"context"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/wire"

	"momoko/internal/conf"
	"momoko/internal/data/ent"
	"momoko/internal/data/ent/migrate"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewAuthRepo,
	NewUserRepo,
	NewSystemRepo,
	NewConfigRepo,
	NewInstanceRepo,
)

// Data is a struct that contains the database client.
type Data struct {
	db *ent.Client
}

// NewData creates a new Data instance.
func NewData(c *conf.Data) (*Data, func(), error) {
	db, err := ent.Open(c.Database.Driver, c.Database.Source)
	if err != nil {
		log.Fatalf("failed opening connection to database: %v", err)
	}
	if os.Getenv("DEPLOY_ENV") == "dev" {
		// Enable debug mode for detailed logging.
		db = db.Debug()
		// Run the auto migration tool.
		if err = db.Schema.Create(context.Background(), migrate.WithDropIndex(true)); err != nil {
			return nil, nil, err
		}
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
