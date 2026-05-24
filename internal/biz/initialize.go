package biz

import (
	"context"
	"net/mail"

	v1 "momoko/api/gen/v1"
	"momoko/internal/conf"
	"momoko/internal/initialize"
)

type InitializeRepo interface {
	ConfirmInitialize(context.Context, v1.DatabaseType, *conf.Data_Database, *v1.InitializeAdminUser) error
	TestDatabase(context.Context, *conf.Data_Database) error
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
	return &v1.InitializeStatusResponse{
		Initialized:            initialize.IsInitialized(),
		SupportedDatabaseTypes: initialize.SupportedDatabaseTypes(),
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
	if admin.Name == "" {
		admin.Name = admin.Username
	}
	if admin.Username == "" {
		return nil, ErrUsernameEmpty
	}
	if admin.Password == "" {
		return nil, ErrPasswordEmpty
	}
	if admin.Email == "" {
		return nil, ErrEmailEmpty
	}
	if _, err := mail.ParseAddress(admin.Email); err != nil {
		return nil, ErrEmailInvalid
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
	if err := u.repo.TestDatabase(ctx, database); err != nil {
		return nil, ErrSystem(err)
	}

	return &v1.TestInitializeDatabaseResponse{
		Success: true,
	}, nil
}
