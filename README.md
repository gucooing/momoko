# Momoko

Momoko 是一个面向服务器、应用实例和开发工具的管理面板。后端使用 Go + Kratos 提供 HTTP/gRPC API，前端使用 Vue 3 + Element Plus + Vite 构建，并可随前端产物一起打进单个后端二进制。

前端工程说明见 [frontend/README.md](./frontend/README.md)。

## 功能清单

### 初始化与账号

- 首次启动初始化向导：支持 SQLite、MySQL、PostgreSQL，初始化时创建超级管理员。
- 自动数据库迁移、内置菜单/角色写入、初始化标记写入和配置文件更新。
- 登录、注册、邮箱验证码登录/注册、访问令牌、刷新令牌、退出登录。
- 登录设备管理、登录日志、密码修改、个人资料、头像上传和个人权限查看。
- 用户、角色、菜单、按钮权限管理，内置超级管理员和无权限角色。

### 系统与运维

- 工作台、分析页、监控页、CPU、内存、磁盘、网络状态和系统概览。
- 登录配置、邮件配置、邮件模板、邮件发送测试和版本更新检查。
- 内置超级管理员进入工作台时自动检查新版本，发现更新后弹窗展示 Markdown 更新内容并提供下载入口。
- 操作记录审计，记录用户、模块、动作、请求详情和结果。
- 通用任务管理：任务列表、详情、取消、重试、删除和进度状态。
- 前端静态资源内嵌服务，支持 SPA 路由回退、gzip/brotli 压缩资源和头像静态服务。

### OIDC 身份提供方

- 内置 OIDC Provider，提供 Discovery、JWKS、Authorize、Token 和 UserInfo 标准端点。
- OIDC Provider 配置：启用状态、Issuer、Access Token 有效期、ID Token 有效期和授权码有效期。
- OIDC 客户端管理：创建、列表、编辑、删除、刷新 Client Secret、回调地址、授权范围和启用状态。
- 授权码登录流程：公开授权页、当前登录账号确认授权、授权码交换 Token、ID Token 和 Access Token 签发。
- OIDC 管理入口接入系统菜单、按钮权限和操作记录审计。

### 应用实例

- 应用类型管理和应用实例列表。
- 实例创建、编辑、启动、停止、强制停止、重启、强制重启、删除。
- 实例启动命令、停止命令、工作目录、环境变量、标签、自启动配置。
- WebSocket 实例控制台、实时输出、命令输入、日志清理和断线重连。
- 实例文件管理：目录树、列表、搜索、排序、创建、重命名、复制、移动、删除、压缩、解压、编辑、下载和分片上传。

### 文件与分享

- 系统文件管理：本地文件列表、目录树、搜索、排序、在线预览、在线编辑。
- 文件操作：创建、重命名、复制、移动、批量删除、压缩、解压、下载预签名、上传预签名、分片上传、上传状态查询、完成上传和取消上传。
- 文件来源管理：本地磁盘、OSS/S3、FTP、WebDAV，支持启用状态、302 直链、连通性测试和来源能力标识。
- 文件分享：分享创建、列表、更新、删除、公开访问 token、提取码、过期时间、下载次数限制、公开目录浏览和分享会话签名。

### SSH 与节点

- SSH 主机列表、创建、详情、更新、删除、分享、单机测试和批量测试。
- Web SSH 终端，支持 PTY、窗口尺寸同步和实时输入输出。
- API Key 管理：创建、列表、复制、更新、刷新、过期时间和启用状态。

### Docker

- Docker 连接配置：启用状态、连接地址、TLS 证书、API 版本、请求超时、默认平台、任务超时和仓库认证。
- Docker 状态：引擎信息、版本信息、容器数量、镜像数量、CPU、内存、存储驱动和错误信息。
- 容器管理：列表、详情、创建、更新、重建、启动、停止、重启、强制停止、暂停、恢复、重命名、删除、资源统计、日志 WebSocket 和 Exec 终端。
- 镜像管理：列表、详情、拉取、标签更新、打标签、删除和历史记录。
- 网络管理：列表、详情、创建、更新、重建、删除、连接容器和断开容器。
- Docker 异步任务：镜像拉取、容器重建、网络重建等任务状态和任务日志。

### 网络工具

- TCP/UDP 端口转发：创建、列表、详情、更新、删除和实时统计。
- 内网穿透：基于 frp 的隧道列表、创建、详情、更新、删除、实时统计。
- frps 配置管理：绑定地址、端口、HTTP/HTTPS 虚拟主机端口、KCP、QUIC、子域名、对外地址和统计采样间隔。
- 隧道凭据和 frpc 配置信息生成。

### 前端体验

- Vue 3 单页后台、动态菜单、标签页、多布局、明暗主题和多语言文案。
- Element Plus、VxeTable、ECharts、xterm.js、Monaco Editor 文件编辑器。
- 403/404 页面、公开分享页面、初始化页面、登录/注册/找回密码页面。
- 扩展组件示例：按钮、对话框、图标选择器、文本省略、Hover 动画、Transition 动画和 VXE Table。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 后端 | Go 1.26.3, Kratos, gRPC, HTTP |
| 数据 | Ent, SQLite, MySQL, PostgreSQL |
| 鉴权 | JWT, 刷新令牌, RBAC, bcrypt |
| 协议 | Protocol Buffers, OpenAPI, ts-proto |
| 文件 | 本地文件, OSS/S3, FTP, WebDAV, 分片上传 |
| 网络 | TCP/UDP 转发, frp, WebSocket |
| Docker | Docker Engine API |
| 前端 | Vue 3, TypeScript, Vite, Pinia, Vue Router |
| UI | Element Plus, VxeTable, ECharts, xterm.js, Monaco Editor |

## 目录结构

```text
momoko/
├── api/                 # proto 定义与生成后的 Go API 代码
├── cmd/momoko/          # 后端启动入口
├── configs/             # 后端配置文件
├── frontend/            # 前端工程
├── internal/            # 后端业务、服务、数据层与服务器实现
├── pkg/                 # 通用能力包
├── third_party/         # proto 依赖
├── dist_embed.go        # 嵌入 frontend/dist
├── Dockerfile           # 容器镜像构建
├── docker-compose.yml   # Compose 部署示例
└── Makefile             # 生成、构建、测试命令
```

## 环境要求

- Go: `1.26.3`
- Node.js: `^20.19.0` 或 `>=22.12.0`
- pnpm: `10.30.3`
- protoc: 修改 proto 后生成代码时需要
- make: 使用 `Makefile` 命令时需要
- Docker: 使用 Docker 管理功能或容器部署时需要

## 快速启动

安装后端依赖：

```bash
go mod download
```

启动后端：

```bash
go run ./cmd/momoko -conf ./configs
```

默认服务地址来自 [configs/config.yaml](./configs/config.yaml)：

- HTTP: `http://localhost:22633`
- gRPC: `localhost:22733`
- API 前缀: `/api/v1`
- SQLite 数据库: `./data/momoko.db`

首次访问会进入初始化流程。初始化时选择数据库并创建超级管理员；项目不再提供固定默认管理员账号。

## 前后端开发

先启动后端，再启动前端：

```bash
cd frontend
pnpm install
pnpm dev
```

前端开发服务器默认运行在 `http://localhost:3007`。开发环境变量位于 [frontend/.env.development](./frontend/.env.development)：

```env
VITE_API_BASE_URL="http://localhost:22633/api/v1"
VITE_STATIC_URL="/"
```

## 一体化构建

先构建前端，再构建后端二进制：

```bash
cd frontend
pnpm install
pnpm build
cd ..
go build -trimpath -ldflags "-X main.Name=momoko -X main.Version=dev" -o ./bin/momoko ./cmd/momoko
```

启动构建产物：

```bash
./bin/momoko -conf ./configs
```

Windows PowerShell：

```powershell
.\bin\momoko.exe -conf .\configs
```

构建后访问 HTTP 地址即可使用前端页面和 `/api/v1` 接口。

## Docker 部署

项目提供 [docker-compose.yml](./docker-compose.yml)：

```bash
docker compose up -d
```

默认镜像为 `gucooing/momoko:latest`，可通过环境变量指定镜像：

```bash
MOMOKO_IMAGE=gucooing/momoko:v0.0.1-dev docker compose up -d
```

Compose 当前使用 host 网络，并挂载：

- `./configs` -> `/app/configs`
- `./data` -> `/app/data`
- `./servers` -> `/app/servers`
- `/usr/bin` -> `/app/bin`

镜像发布位置：

- Docker Hub: `gucooing/momoko`
- GitHub Container Registry: `ghcr.io/gucooing/momoko`

## 常用命令

```bash
# 安装 proto / Kratos / Wire 相关生成工具
make init

# 生成 API Go 代码、HTTP 代码、gRPC 代码、OpenAPI 和前端 TypeScript 类型
make api

# 生成内部配置 proto 代码
make config

# 执行 go generate、整理 Go 依赖并构建前端
make gen

# 执行全部生成任务
make all

# 构建后端
make build

# 构建前端
make build-pnpm

# 后端测试、前端 ESLint 和前端构建
make test
```

## 配置

主配置文件为 [configs/config.yaml](./configs/config.yaml)：

```yaml
server:
  http:
    addr: 0.0.0.0:22633
    timeout: 15s
  grpc:
    addr: 0.0.0.0:22733
    timeout: 15s
data:
  database:
    driver: sqlite3
    source: file:./data/momoko.db?_pragma=foreign_keys(1)
auth:
  secret: "momoko:jwt:hs256:v1"
```

初始化完成后会写入数据库配置、生成新的 `auth.secret`，并创建 `data/initialized.json`。

设置 `DEPLOY_ENV=dev` 后启动后端，可启用 Ent debug SQL 日志。

## 开发说明

- Proto 定义位于 [api/proto/v1](./api/proto/v1)，修改后执行 `make api`。
- Ent schema 位于 [internal/data/ent/schema](./internal/data/ent/schema)，修改后执行 `go generate ./internal/data/ent`。
- HTTP 路由由 proto 注解生成，注册逻辑位于 [internal/server/http.go](./internal/server/http.go)。
- 功能权限校验位于 [internal/server/authz.go](./internal/server/authz.go)。
- 前端动态菜单转换位于 [frontend/src/router/menuToRoute.ts](./frontend/src/router/menuToRoute.ts)。
- 前端 API 类型由 proto 生成到 [frontend/src/types/v1](./frontend/src/types/v1)。
- OpenAPI 文件生成到 [frontend/src/types/openapi.yaml](./frontend/src/types/openapi.yaml)。

## 许可证

本项目使用 [MIT License](./LICENSE)。
