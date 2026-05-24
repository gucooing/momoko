# Momoko

Momoko 是一个面向游戏服务器与应用实例运维的管理面板。后端基于 Go + Kratos 构建，提供 HTTP/gRPC API、RBAC 权限、实例管理、SSH 管理、文件管理、系统监控与操作日志等能力；前端位于 `frontend/`，基于 Vue 3 + Element Plus + Vite 开发。

> 前端说明见 [frontend/README.md](./frontend/README.md)。

## 功能概览

- 用户、角色、菜单与按钮级权限管理
- 登录、刷新令牌、登录设备、登录日志与个人资料管理
- 系统概览、资源监控、系统配置、邮件配置与邮件模板
- 应用实例类型、实例列表、实例控制台和实例文件管理
- SSH 主机管理与 Web SSH 终端
- 文件管理、分片上传、下载预签名、在线预览与编辑
- 操作日志记录与前端静态资源内嵌服务

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 后端框架 | Go, Kratos, gRPC, HTTP |
| 数据访问 | Ent, SQLite, MySQL driver |
| 鉴权 | JWT, 访问令牌/刷新令牌, RBAC |
| 协议生成 | Protocol Buffers, OpenAPI, ts-proto |
| 前端 | Vue 3, TypeScript, Vite, Pinia, Vue Router |
| UI | Element Plus, VxeTable, ECharts, xterm.js |

## 目录结构

```text
momoko/
├── api/                 # proto 定义与生成后的 API 代码
├── cmd/momoko/          # 后端启动入口
├── configs/             # 后端配置文件
├── frontend/            # 前端项目
├── internal/            # 后端业务、服务、数据层与服务器实现
├── pkg/                 # 通用工具包
├── third_party/         # proto 依赖
├── dist_embed.go        # 将 frontend/dist 嵌入后端二进制
└── Makefile             # 代码生成与构建命令
```

## 环境要求

- Go: 以 `go.mod` 中声明版本为准
- Node.js: `^20.19.0` 或 `>=22.12.0`
- pnpm: `>=10.4.1`
- protoc: 修改 proto 后生成代码时需要
- make: 使用 `Makefile` 中的生成命令时需要

## 后端快速启动

1. 安装 Go 依赖：

```bash
go mod download
```

2. 启动后端：

```bash
go run ./cmd/momoko -conf ./configs
```

默认配置来自 `configs/config.yaml`：

- HTTP: `http://localhost:22633`
- gRPC: `localhost:22733`
- API 前缀: `/api/v1`
- 数据库: `./data/momoko.db`

首次启动会自动迁移数据库并写入内置 RBAC 数据。默认管理员账号：

```text
用户名: admin
密码: admin
```

部署或公开访问前请尽快修改默认密码，并替换 `configs/config.yaml` 中的 `auth.secret`。

## 前后端联调

先启动后端，再启动前端开发服务器：

```bash
cd frontend
pnpm install
pnpm dev
```

前端默认运行在 `http://localhost:3007`，开发环境 API 地址配置在 `frontend/.env.development`：

```env
VITE_API_BASE_URL="http://localhost:22633/api/v1"
VITE_STATIC_URL="/"
```

## 一体化构建

后端通过 `dist_embed.go` 嵌入 `frontend/dist`，因此打包后端二进制前需要先构建前端：

```bash
cd frontend
pnpm install
pnpm build
cd ..
go build -ldflags "-X main.Name=momoko -X main.Version=dev" -o ./bin/momoko ./cmd/momoko
```

Windows PowerShell 下可运行：

```powershell
.\bin\momoko.exe -conf .\configs
```

Linux/macOS 下可运行：

```bash
./bin/momoko -conf ./configs
```

构建后访问后端 HTTP 地址即可同时访问前端页面与 `/api/v1` 接口。

## Docker 部署

项目提供 `docker-compose.yml`，默认使用最新镜像 `gucooing/momoko:latest` 并同时启动 MySQL：

```bash
docker compose up -d
```

Compose 会读取 `configs/config.yaml`，默认端口：

- Momoko HTTP: `22633`
- Momoko gRPC: `22733`
- MySQL: `3306`

可通过环境变量覆盖镜像和端口，例如：

```bash
MOMOKO_IMAGE=gucooing/momoko:v0.0.1-dev MOMOKO_HTTP_PORT=8080 docker compose up -d
```

镜像发布目标：

- Docker Hub: `gucooing/momoko`
- GitHub Container Registry: `ghcr.io/gucooing/momoko`

## 常用命令

```bash
# 构建后端
make build

# 安装 proto / Kratos / Wire 相关生成工具
make init

# 生成 API Go 代码、OpenAPI 与前端 TypeScript 类型
make api

# 生成内部配置 proto 代码
make config

# 执行 go generate 并整理依赖
make gen

# 执行全部生成任务
make all

# 运行测试
go test ./...
```

## 配置说明

主要配置文件为 `configs/config.yaml`：

```yaml
server:
  http:
    addr: 0.0.0.0:22633
  grpc:
    addr: 0.0.0.0:22733
data:
  database:
    driver: sqlite3
    source: file:./data/momoko.db?_pragma=foreign_keys(1)
auth:
  secret: gucooing.auth
```

`data.database.driver` 当前可使用项目已引入的 SQLite 或 MySQL driver。切换数据库时请同步调整连接串，并确保运行环境具备对应数据库服务。

## 开发说明

- Proto 定义位于 `api/proto/v1`，修改后执行 `make api`。
- Ent schema 位于 `internal/data/ent/schema`，相关生成入口在 `internal/data/ent/generate.go`。
- 后端 HTTP 路由由 proto 注解生成，注册逻辑位于 `internal/server/http.go`。
- 前端 API 类型由后端 proto 生成到 `frontend/src/types`。
- 设置 `DEPLOY_ENV=dev` 后启动后端，可启用 Ent debug SQL 日志。

## 许可证

本项目使用 [MIT License](./LICENSE)。
