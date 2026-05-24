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

func (r *initializeRepo) TestDatabase(ctx context.Context, database *conf.Data_Database) error {
	return TestDatabaseConnection(ctx, database)
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
