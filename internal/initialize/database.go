package initialize

import (
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	v1 "momoko/api/gen/v1"
	"momoko/internal/conf"

	"github.com/go-sql-driver/mysql"
)

const (
	DefaultSQLitePath = "./data/momoko.db"

	defaultMySQLPort      = 3306
	defaultPostgreSQLPort = 5432
)

func SupportedDatabaseTypes() []v1.DatabaseType {
	return []v1.DatabaseType{
		v1.DatabaseType_DatabaseType_SQLite,
		v1.DatabaseType_DatabaseType_MySQL,
		v1.DatabaseType_DatabaseType_PostgreSQL,
	}
}

func DatabaseDriver(databaseType v1.DatabaseType) (string, error) {
	switch databaseType {
	case v1.DatabaseType_DatabaseType_SQLite:
		return "sqlite3", nil
	case v1.DatabaseType_DatabaseType_MySQL:
		return "mysql", nil
	case v1.DatabaseType_DatabaseType_PostgreSQL:
		return "postgres", nil
	default:
		return "", fmt.Errorf("unsupported database type: %s", databaseType.String())
	}
}

func BuildDatabaseConfig(database *v1.InitializeDatabaseConfig) (*conf.Data_Database, error) {
	if database == nil {
		return nil, fmt.Errorf("数据库配置不能为空")
	}
	if database.Type == v1.DatabaseType_DatabaseType_Unspecified {
		return nil, fmt.Errorf("数据库类型不能为空")
	}

	driver, err := DatabaseDriver(database.Type)
	if err != nil {
		return nil, fmt.Errorf("数据库类型不支持")
	}

	source, err := DatabaseSource(database)
	if err != nil {
		return nil, err
	}

	return &conf.Data_Database{
		Driver: driver,
		Source: source,
	}, nil
}

func DatabaseSource(database *v1.InitializeDatabaseConfig) (string, error) {
	switch database.GetType() {
	case v1.DatabaseType_DatabaseType_SQLite:
		return sqliteSource(database.GetSqlitePath())
	case v1.DatabaseType_DatabaseType_MySQL:
		return mysqlSource(database)
	case v1.DatabaseType_DatabaseType_PostgreSQL:
		return postgreSQLSource(database)
	default:
		return "", fmt.Errorf("数据库类型不支持")
	}
}

func sqliteSource(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		path = DefaultSQLitePath
	}
	if strings.HasPrefix(strings.ToLower(path), "file:") || strings.Contains(path, "?") {
		return "", fmt.Errorf("SQLite 只需填写数据库文件路径，不要填写连接串")
	}

	cleanPath := filepath.ToSlash(filepath.Clean(path))
	if strings.HasPrefix(path, "./") && !strings.HasPrefix(cleanPath, "./") {
		cleanPath = "./" + cleanPath
	}
	return "file:" + cleanPath + "?_pragma=foreign_keys(1)", nil
}

func mysqlSource(database *v1.InitializeDatabaseConfig) (string, error) {
	address, username, databaseName, err := normalizeNetworkDatabaseFields(database, defaultMySQLPort)
	if err != nil {
		return "", err
	}

	config := mysql.NewConfig()
	config.User = username
	config.Passwd = database.GetPassword()
	config.Net = "tcp"
	config.Addr = address
	config.DBName = databaseName
	config.ParseTime = true
	config.Loc = time.Local

	return config.FormatDSN(), nil
}

func postgreSQLSource(database *v1.InitializeDatabaseConfig) (string, error) {
	address, username, databaseName, err := normalizeNetworkDatabaseFields(database, defaultPostgreSQLPort)
	if err != nil {
		return "", err
	}

	user := url.User(username)
	if database.GetPassword() != "" {
		user = url.UserPassword(username, database.GetPassword())
	}
	u := &url.URL{
		Scheme:   "postgres",
		User:     user,
		Host:     address,
		Path:     "/" + databaseName,
		RawQuery: "sslmode=disable",
	}
	return u.String(), nil
}

func normalizeNetworkDatabaseFields(database *v1.InitializeDatabaseConfig, defaultPort int) (string, string, string, error) {
	address, err := normalizeTCPAddress(database.GetAddress(), defaultPort)
	if err != nil {
		return "", "", "", err
	}

	username := database.GetUsername()
	if username == "" {
		return "", "", "", fmt.Errorf("数据库用户名不能为空")
	}

	databaseName := database.GetDatabaseName()
	if databaseName == "" {
		return "", "", "", fmt.Errorf("数据库名不能为空")
	}

	return address, username, databaseName, nil
}

func normalizeTCPAddress(address string, defaultPort int) (string, error) {
	if address == "" {
		return "", fmt.Errorf("数据库连接地址不能为空")
	}
	if _, _, err := net.SplitHostPort(address); err == nil {
		return address, nil
	}
	if strings.Contains(address, "://") || strings.ContainsAny(address, "/?") || strings.Count(address, ":") > 0 {
		return "", fmt.Errorf("数据库连接地址格式不正确")
	}
	return net.JoinHostPort(address, strconv.Itoa(defaultPort)), nil
}
