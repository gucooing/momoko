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

### 游戏服务器、GM 运营与用户通行证

- 游戏服务器：托管 / 仅 GM 两种接入，经 MGP v1（HMAC 签名）调用游戏服务端；第一个适配的游戏是学マス。支持客户端启动的游戏（学マス）在服务器的连接设置里有「客户端」：生成客户端应用凭据（应用 ID 与应用密钥，可重置密钥或重新生成应用 ID，都要先确认；应用密钥加密保存，只在生成时显示一次；托管服务器由 Momoko 把凭据和通行证对外地址（`passport.public_url`，必须已配置）写进服务端配置的 `momoko.app_id` / `momoko.app_secret` / `momoko.url`，运行中可选立即重启），并填写游戏客户端连接用的 SDK、游戏服务端与资源服务器地址（不带路径的 HTTP(S) origin；资源服务器地址可不填，不填时客户端使用官方资源）。
- GM 运营：工作台、玩家与玩家详情、运营管理（一张运营日历：卡池、活动、竞赛赛季、登录奖励、公告各一条泳道，点条目先看详情再编辑；草稿自动保存，发布前服务端校验，发布后热重载，有版本记录）、玩法、邮件补偿、兑换码、GM 操作（可同时作用于多名玩家）、GM 记录与玩家自助申请的审批。
- 客户端日志：游戏客户端凭游戏服务端签发的上传票据，把日志传到用户通行证 server（`POST /api/v1/game-client-logs`）；上传原因只有 `exception`、`crash`、`fatal`（客户端自动上传）与 `manual`（玩家手动上传）；每个游戏账号每小时最多 30 份、只保留最新 5 份。管理端「游戏 → 客户端日志」页列出全部服务器的日志（从新到旧分页，按服务器、玩家 ID、原因、上传时间筛选，关键字搜摘要、说明与设备，每行带服务器、玩家与绑定的通行证），GM 玩家详情的「客户端日志」分页是同一个列表按服务器与玩家筛选。两处共用查看器：点一行直接读全文（等宽、保留换行、按行虚拟滚动，几十 MB 也能看），可在日志里查找并在匹配之间跳转、复制全部、下载 `.log`；行内可复制摘要、删除（先确认）。
- 用户通行证：独立端口（默认 `:22634`）上的玩家端，只注册玩家路由；注册登录、两步验证、登录设备、绑定游戏账号、游戏空间、自助服务、兑换码和消息。
- 游戏客户端登录（PC 启动器、Android APK）：客户端不接触通行证的账号、密码或任何登录方式，只经浏览器授权取得会话。客户端 `POST /api/v1/auth/client`（`{client: launcher | android, deviceId, device}`，同一 IP 发起过于频繁时 400 `PASSPORT_TOO_MANY_REQUESTS`）拿到 `deviceCode`、`userCode`（`XXXX-XXXX`）与授权页地址 `{passport.public_url}/authorize?code=…`（没配置 `public_url` 时 400 `PASSPORT_PUBLIC_URL_UNSET`），用系统浏览器打开；玩家在通行证网页登录后核对代码并授权或拒绝（`POST /api/v1/auth/client/approve` / `deny`）；客户端每 2 秒 `POST /api/v1/auth/client/token {deviceCode}` 领取，授权后得到与登录相同的设备会话（登录方式记为 `client`），还没授权、轮询过快、被拒绝、过期或已领取分别是 `PASSPORT_AUTH_PENDING`、`PASSPORT_AUTH_SLOW_DOWN`、`PASSPORT_AUTH_DENIED`、`PASSPORT_AUTH_EXPIRED`。授权请求只存在进程内存里，600 秒后失效，`deviceCode` 只存哈希。
- 游戏客户端启动：客户端用客户端应用 ID 指明服务器。可启动的服务器要求游戏支持客户端启动、接入了 MGP、生成了客户端应用凭据、填写了 SDK 与游戏服务端地址；资源服务器地址可以不填，客户端这时使用官方资源。`GET /api/v1/launch/apps` 列出玩家有绑定或对他可见的启动目标（有绑定的按最近使用在前，其余按服务器名称），`GET /api/v1/launch/apps/{appId}` 查一个（单服务器客户端用）；没有游戏账号时绑定已有账号（`POST /api/v1/launch/apps/{appId}/bind`）或经 MGP 新建一个（`POST /api/v1/launch/apps/{appId}/account`，绑定方式为 `create`），再申请启动票据（`POST /api/v1/launch/apps/{appId}/ticket`）交给那台服务器的 SDK 换取游戏会话。票据为 `lt1.<payload>.<签名>`，payload 为 `{aid, sid, pid, uid, jti, iat, exp}`，用这台服务器的应用密钥做 HMAC-SHA256 签名，有效 120 秒；签发时记录绑定的最近使用时间。

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
- API Key 管理：创建、列表、复制、更新、刷新、删除、过期时间和节点权限范围。

### 多节点

- 主控经节点的 gRPC 端口接入其他 momoko 节点，在顶栏切换节点后统一管理其机器状态、应用（含控制台与实例文件）、文件、容器和工具（端口转发、内网穿透、SSH）。
- 签名鉴权：节点上创建的 API Key 只作为签名种子。主控每次调用携带由种子派生的 Key ID、时间戳、一次性随机数和绑定接口方法的 HMAC-SHA256 签名，节点校验 ±5 分钟时间窗并拒绝重放；种子不在网络上传输，主控侧加密存储。
- 权限范围：API Key 勾选可用功能，节点上的生效权限为“授予范围 ∩ Key 所属用户权限”，主控再与当前登录用户的权限取交集；切换到节点后，未授权的页面、菜单和操作按钮不显示。
- 转发方式：普通接口按 gRPC 操作名直接转发（复用各服务的 proto 定义，无需逐接口适配）；WebSocket 终端/日志与文件预签名上传/下载经 gRPC 双向流隧道转发。
- 节点管理：新增、编辑、删除、测试连接，后台定时探测在线状态、延迟、版本与授权范围；节点与主控的操作记录分别记下主控操作者和转发目标节点。
- 文件分享、SSH 分享依赖主控本机的公开地址与用户，远程节点上不提供。

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

### 智商检测

- 提示词配置：维护发送给上游模型的检测提示词，定时检测时从启用项中随机选一条。
- 检测配置：按站点配置端点、API Key、协议（chat / responses / messages）、User-Agent、勾选模型及各自的推理强度、检测间隔和次数上限；支持拉取上游模型列表、立即检测、列表/卡片视图。
- 定时检测：每分钟扫描到期站点，对勾选的全部模型各发一次同一条提示词（流式读取），一个模型一条结果。
- 公开公示页：`/public/iqcheck/{slug}`，无需登录，按时间倒序展示每次检测的渲染回复（Markdown / HTML / SVG，在 sandbox iframe 中渲染）、模型、推理强度和耗时，点击查看请求提示词与回复原文。

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
| 鉴权 | JWT, 刷新令牌, RBAC, bcrypt, API Key HMAC 时间戳签名 |
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
└── Makefile             # 生成、构建、测试命令
```

## 环境要求

- Go: `1.26.3`
- Node.js: `^20.19.0` 或 `>=22.12.0`
- pnpm: `10.30.3`
- protoc: 修改 proto 后生成代码时需要
- make: 使用 `Makefile` 命令时需要
- Docker: 使用 Docker 管理功能时需要

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

`pnpm dev` 同时启动管理后台 `http://localhost:3007`（`pnpm dev:admin`）和用户通行证玩家端 `http://localhost:3008`（`pnpm dev:passport`）。开发环境变量位于 [frontend/.env.development](./frontend/.env.development)：

```env
VITE_API_BASE_URL="http://localhost:22633/api/v1"
VITE_PASSPORT_API_BASE_URL="http://localhost:22634/api/v1"
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

### 游戏素材

游戏图片（`frontend/public/games/<游戏>/`）不打进二进制，Momoko 也不转发：`pnpm build` 会把所有游戏单独打包到 `frontend/dist-games/`（发布版本附带 `momoko_<版本>_games.zip`）。把里面的内容（每个游戏一个目录）上传到对象存储或 CDN 并允许公开读取，然后在 **系统设置 · 储存 · 游戏素材地址** 填上传到的那个目录（例如 `https://cdn.example.com/momoko-games`，图片地址为 `<地址>/gakumas/…`），浏览器直接从那里取图。没填时界面显示占位图。文件名带内容哈希，新版本覆盖上传即可。

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

gRPC 端口供主控接入本节点（多节点），默认明文。对公网开放时建议配置证书或置于 TLS 反向代理之后，主控添加节点时勾选「使用 TLS」：

```yaml
server:
  grpc:
    addr: 0.0.0.0:22733
    timeout: 15s
    tls_cert_file: ./configs/grpc.crt
    tls_key_file: ./configs/grpc.key
```

设置 `DEPLOY_ENV=dev` 后启动后端，可启用 Ent debug SQL 日志。

## 开发说明

- Proto 定义位于 [api/proto/v1](./api/proto/v1)，修改后执行 `make api`。
- Ent schema 位于 [internal/data/ent/schema](./internal/data/ent/schema)，修改后执行 `go generate ./internal/data/ent`。
- 增量 SQL 迁移脚本位于 [internal/data/migration/sql](./internal/data/migration/sql)，文件名为 `<序号>_<名称>.sql`（如 `00_remove_sub2api.sql`）。启动时在 ent 自动迁移之后按序号执行 `sql_migrations` 表中尚未记录的新脚本并写入记录；脚本以分号分隔语句，标识符用双引号，MySQL 下自动改写为反引号。
- HTTP 路由由 proto 注解生成，注册逻辑位于 [internal/server/http.go](./internal/server/http.go)。
- 功能权限校验位于 [internal/server/authz.go](./internal/server/authz.go)。
- 远程节点可调用的操作、所需节点权限和可走隧道的原始路径位于 [internal/server/node_policy.go](./internal/server/node_policy.go)；新增按节点区分的接口时在此登记，未登记的服务不对远程主控开放。前端对应的页面权限表位于 [frontend/src/config/nodePermission.ts](./frontend/src/config/nodePermission.ts)。
- 前端动态菜单转换位于 [frontend/src/router/menuToRoute.ts](./frontend/src/router/menuToRoute.ts)。
- 前端 API 类型由 proto 生成到 [frontend/src/types/v1](./frontend/src/types/v1)。
- OpenAPI 文件生成到 [frontend/src/types/openapi.yaml](./frontend/src/types/openapi.yaml)。

## 许可证

本项目使用 [MIT License](./LICENSE)。
