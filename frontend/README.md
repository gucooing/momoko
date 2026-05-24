<div align="center">
  <img src="./public/logo.svg" alt="Momoko Frontend Logo" width="120" />

# Momoko Frontend

</div>

Momoko Frontend 是 Momoko 管理面板的前端工程，基于 Vue 3、TypeScript、Element Plus 与 Vite 构建，负责提供登录、权限菜单、系统监控、实例管理、SSH 终端、文件管理和系统配置等后台管理界面。

> 原项目标注：本前端基于 [DFAN Admin](https://github.com/DFANNN/DFAN-Admin) 二次开发，原作者为 DFANNN，原项目使用 MIT License。Momoko 在此基础上接入真实后端 API，并扩展了游戏服务器/应用实例管理相关业务模块。

## 功能概览

- 登录、刷新令牌、登录设备管理与个人中心
- 后端动态菜单、角色权限与按钮级权限控制
- 用户、角色、菜单、系统配置与操作日志管理
- 系统概览、CPU/内存/网络/磁盘状态监控
- 实例列表、实例类型、实例控制台与实例文件管理
- SSH 主机管理与 Web SSH 终端
- 文件管理、分片上传、下载、预览和在线编辑
- 明暗主题、标签页、响应式布局和常用后台组件示例

## 技术栈

| 分类 | 技术 |
| --- | --- |
| 核心框架 | Vue 3, TypeScript |
| 构建工具 | Vite |
| 状态与路由 | Pinia, Vue Router |
| UI 组件 | Element Plus |
| 表格 | VxeTable, VxePC UI |
| 图表与终端 | ECharts, xterm.js, Monaco Editor |
| 请求层 | Axios |
| 工具库 | Day.js, VueUse, noble hashes |
| 样式 | Sass, Tailwind CSS |
| 类型来源 | 后端 proto 生成的 TypeScript 类型 |

## 环境要求

- Node.js: `^20.19.0` 或 `>=22.12.0`
- pnpm: `>=10.4.1`，当前锁定版本见 `packageManager`

## 快速开始

启动前请先在仓库根目录运行后端：

```bash
go run ./cmd/momoko -conf ./configs
```

然后启动前端：

```bash
cd frontend
pnpm install
pnpm dev
```

开发服务器默认地址为 `http://localhost:3007`。默认后端 API 地址来自 `.env.development`：

```env
VITE_API_BASE_URL="http://localhost:22633/api/v1"
VITE_STATIC_URL="/"
```

默认管理员账号：

```text
用户名: admin
密码: admin
```

## 环境变量

| 变量 | 说明 |
| --- | --- |
| `VITE_API_BASE_URL` | 后端 API 基础地址，开发环境默认为 `http://localhost:22633/api/v1` |
| `VITE_STATIC_URL` | 静态资源和路由 base，默认 `/` |

生产环境 `.env.production` 默认使用相对 API 地址 `/api/v1`，适合由后端二进制同时托管前端静态资源和 API。

## 常用命令

```bash
# 启动开发服务器
pnpm dev

# 类型检查并构建生产版本
pnpm build

# 仅预览 dist 产物
pnpm preview

# ESLint 自动修复
pnpm lint

# 格式化 src 目录
pnpm format
```

`pnpm build` 会执行类型检查、Vite 构建、复制 `dist/index.html` 为 `dist/404.html`，并通过 `scripts/compress-dist.mjs` 生成压缩资源，供后端静态资源服务优先返回 gzip/brotli 版本。

## 与后端集成

前端 API 类型和 OpenAPI 文件由根目录 proto 生成：

```bash
cd ..
make api
```

生成结果主要包括：

- `frontend/src/types/v1/*.ts`
- `frontend/src/types/openapi.yaml`

后端打包时会通过根目录 `dist_embed.go` 嵌入 `frontend/dist`。完整一体化构建顺序：

```bash
cd frontend
pnpm install
pnpm build
cd ..
go build -ldflags "-X main.Name=momoko -X main.Version=dev" -o ./bin/momoko ./cmd/momoko
```

## 目录结构

```text
frontend/
├── public/              # 静态资源
├── scripts/             # 构建辅助脚本
├── src/
│   ├── api/             # 后端 API 封装
│   ├── assets/          # 图片、Logo、动效等资源
│   ├── components/      # 公共组件
│   ├── composables/     # 组合式函数
│   ├── config/          # 应用、权限、组件配置
│   ├── directives/      # 自定义指令
│   ├── layouts/         # 后台布局
│   ├── router/          # 静态路由与动态路由转换
│   ├── stores/          # Pinia 状态
│   ├── styles/          # 全局样式
│   ├── types/           # proto/openapi 生成类型
│   ├── utils/           # 请求、资源、格式化等工具
│   └── views/           # 页面视图
├── vite.config.ts       # Vite 配置
└── package.json         # 脚本与依赖
```

## 开发约定

- 新增接口优先放在 `src/api`，并复用 `src/utils/request.ts` 的统一请求实例。
- 后端菜单会通过 `src/router/menuToRoute.ts` 转换为动态路由，页面路径需与 `src/views` 下的文件路径匹配。
- 按钮权限通过后端返回的权限码和 `src/directives/permission.ts` 控制。
- 应用名称、Logo、Favicon 和主题配置集中在 `src/config/app.config.ts`。
- Element Plus 全局默认行为集中在 `src/config/elementConfig.ts`。

## 原项目与许可证

- 原项目仓库：[DFANNN/DFAN-Admin](https://github.com/DFANNN/DFAN-Admin)
- 原项目作者：DFANNN
- 原项目许可证：MIT License，见 [frontend/LICENSE](./LICENSE)
- Momoko 仓库许可证：见根目录 [LICENSE](../LICENSE)
