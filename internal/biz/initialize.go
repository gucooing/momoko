package biz

import (
	"context"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	v1 "momoko/api/gen/v1"
	"momoko/internal/conf"
	"momoko/internal/initialize"
	"momoko/pkg/version"
)

type InitializeRepo interface {
	ConfirmInitialize(context.Context, v1.DatabaseType, *conf.Data_Database, *v1.InitializeAdminUser) error
	// TestDatabase 测试连接，返回数据库服务端版本与已有数据表数量。
	TestDatabase(context.Context, *conf.Data_Database) (*DatabaseProbe, error)
}

// DatabaseProbe 初始化向导「测试连接」的探测结果。
type DatabaseProbe struct {
	// ServerVersion 数据库服务端版本（带产品名前缀，如 "SQLite 3.46.0"）
	ServerVersion string
	// TableCount 目标库已有的数据表数量
	TableCount int64
}

type InitializeUsecase struct {
	repo InitializeRepo
}

func NewInitializeUsecase(repo InitializeRepo) *InitializeUsecase {
	return &InitializeUsecase{
		repo: repo,
	}
}

func (u *InitializeUsecase) Status() *v1.InitializeStatusResponse {
	hostname, _ := os.Hostname()
	return &v1.InitializeStatusResponse{
		Initialized:            initialize.IsInitialized(),
		SupportedDatabaseTypes: initialize.SupportedDatabaseTypes(),
		Version:                version.Current(),
		Hostname:               hostname,
		DefaultSqlitePath:      initialize.DefaultSQLitePath,
	}
}

func (u *InitializeUsecase) Confirm(ctx context.Context, req *v1.ConfirmInitializeRequest) (*v1.ConfirmInitializeResponse, error) {
	if initialize.IsInitialized() {
		return nil, ErrInitializeDone
	}
	if req == nil || req.Database == nil {
		return nil, ErrInitializeDatabaseEmpty
	}
	if req.Admin == nil {
		return nil, ErrInitializeAdminEmpty
	}

	database, err := initialize.BuildDatabaseConfig(req.Database)
	if err != nil {
		return nil, ErrSystem(err)
	}

	admin := req.Admin
	admin.Username = strings.TrimSpace(admin.Username)
	admin.Email = strings.TrimSpace(admin.Email)
	admin.Name = strings.TrimSpace(admin.Name)
	if admin.Name == "" {
		admin.Name = admin.Username
	}
	if admin.Username == "" {
		return nil, ErrUsernameEmpty
	}
	if err := ValidateNewPassword(admin.Password); err != nil {
		return nil, err
	}
	if admin.Email == "" {
		return nil, ErrEmailEmpty
	}
	if _, err := mail.ParseAddress(admin.Email); err != nil {
		return nil, ErrEmailInvalid
	}

	if err := ensureSQLiteDir(req.Database); err != nil {
		return nil, ErrSystem(err)
	}
	if err := u.repo.ConfirmInitialize(ctx, req.Database.Type, database, admin); err != nil {
		return nil, err
	}

	return &v1.ConfirmInitializeResponse{
		Initialized:     true,
		RestartRequired: true,
	}, nil
}

func (u *InitializeUsecase) TestDatabase(ctx context.Context, req *v1.TestInitializeDatabaseRequest) (*v1.TestInitializeDatabaseResponse, error) {
	if initialize.IsInitialized() {
		return nil, ErrInitializeDone
	}
	if req == nil || req.Database == nil {
		return nil, ErrInitializeDatabaseEmpty
	}

	database, err := initialize.BuildDatabaseConfig(req.Database)
	if err != nil {
		return nil, ErrSystem(err)
	}
	// SQLite：记录测试前文件是否存在；测试连接会创建空文件，若原本不存在则测完删掉，避免留下空库。
	resp := &v1.TestInitializeDatabaseResponse{DatabaseExists: true}
	if req.Database.GetType() == v1.DatabaseType_DatabaseType_SQLite {
		path, err := initialize.SQLitePath(req.Database.GetSqlitePath())
		if err != nil {
			return nil, ErrSystem(err)
		}
		resp.SqlitePath = path
		_, statErr := os.Stat(path)
		resp.DatabaseExists = statErr == nil
		if err := ensureSQLiteDir(req.Database); err != nil {
			return nil, ErrSystem(err)
		}
	}
	start := time.Now()
	probe, err := u.repo.TestDatabase(ctx, database)
	if !resp.DatabaseExists && resp.SqlitePath != "" {
		if info, statErr := os.Stat(resp.SqlitePath); statErr == nil && info.Size() == 0 {
			_ = os.Remove(resp.SqlitePath)
		}
	}
	if err != nil {
		return nil, ErrSystem(err)
	}

	resp.Success = true
	resp.ServerVersion = probe.ServerVersion
	resp.TableCount = probe.TableCount
	resp.LatencyMs = time.Since(start).Milliseconds()
	return resp, nil
}

// ensureSQLiteDir 为 SQLite 数据库文件创建所在目录（自定义路径不一定在 data/ 下）。
func ensureSQLiteDir(database *v1.InitializeDatabaseConfig) error {
	if database.GetType() != v1.DatabaseType_DatabaseType_SQLite {
		return nil
	}
	path, err := initialize.SQLitePath(database.GetSqlitePath())
	if err != nil {
		return err
	}
	return os.MkdirAll(filepath.Dir(path), 0o755)
}
