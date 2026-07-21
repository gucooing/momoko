# 08 · 进度清单(Living Progress)

> **活文档**:每次会话收尾更新。状态图例:⬜ 未开始 · 🟡 进行中 · ✅ 完成(桌面+移动+明暗+验收过)。
> 验收标准见 `README` §5;每页需桌面(1440)+ 移动(390)截图核对。

---

## 里程碑

| 阶段 | 内容 | 状态 |
|---|---|---|
| 任务书 | `docs/redesign/*` 全套 | ✅ |
| 基线撤回 | 探索性改动还原 | ✅ |
| Phase 0 | Nuxt UI 接入 + 令牌 + 横切设施 + `components/ui/*` | ✅ 已跑通并浏览器验证 |
| Phase 1 | 外壳(侧栏/顶栏/标签/内容,桌面+移动) | ✅ 样板已建+验证(见下务实取舍) |
| Phase 2 | 样板页:工作台 + 用户管理(定方向) | ✅ 工作台 + 用户管理(卡/表/CRUD/移动/明暗)已完成并浏览器验证 |
| Phase 3 | 全量推广(见下逐页) | ✅ 业务页重写完成；假数据页 analysis/monitor + demo/extended **已删除**；终端/实例文件等见 Phase 3.5 |
| Phase 3.5 | **伪终端重写(实例控制台 + SSH 终端 → 真 xterm.js + 后端真 PTY)** | 🟡 前后端完成并实机验证:codex TUI 直连键入/回显/resize、SSH htop 鼠标滚轮(ws 抓包证实);Win10 宿主 ConPTY 无鼠标属 OS 限制(OpenConsole 集成经用户裁决**不做**);移动+浅色待验 |
| Phase 4 | 清理待决 + 暗色/可访问性/i18n 全量核对 + **EP/VXE 彻底卸载** | ✅ **EP/VXE 卸载完成**（`c289196`+后续）：死代码/demo/假仪表盘已删；BaseDialog 令牌化；ElMessage→`feedback` 单例；package 无 `element-plus`/`vxe-*`；模板 `<el-` / 运行时 `ElMessage` / node_modules 清零。仍保留自建 CSS 变量名 `--el-*`（非 EP 依赖，可选日后改名 `--app-*`）。**余量**：暗色/可访问性/i18n 全量核对；终端移动浅色；实例文件页 |

> **🔓 后端破坏性修改授权(2026-07-12,用户明确)**：若前端重写需要,**允许破坏性修改后端、不必考虑兼容性**(协议/RPC/表结构随意改)。后端从 GoLand 运行,改 Go 后需用户手动重跑。
>
> **🖥️ 终端(P6 伪终端)重写口径(2026-07-12,用户点名)**：实例控制台 + SSH 终端"都不好看、不符合新规范" → 一并重写为**真 xterm.js**。定稿:①**升级实例控制台为真 xterm**(旧为自建 `outputLines` 行渲染的"伪终端";SSH 本就是 xterm);②明暗**改为终端自身手动切换、不受全局主题影响、默认黑色**(自包含 `--term-*` 令牌 + `localStorage`,类似文件编辑器的独立主题);③外壳/工具条走**新令牌 + 薄荷强调 + 状态点**;④**终端页 fullBleed 全出血**(用户:"整个页面都属于终端,强行加框不伦不类");⑤**后端真 PTY**(用户:"一个是真实的终端,一个只是假装的终端" → 实例子进程跑在 `go-pty` 伪终端里,ws 协议与 SSH 完全同构:原始键盘流入 / `{"type":"resize"}` 控制帧 / 原始字节流出(二进制帧+前端流式 UTF-8 解码),后端**零加工输出**;实例控制台去掉命令输入条,xterm 直接键入)。共享 `components/terminal/` 的 **`TerminalConsole.vue`(两页唯一公共入口:外壳+xterm+主题+输入输出接线,页面只写传输层)** + `TerminalShell.vue`(外壳) + `useTerminalX.ts`(xterm 封装,含 onBinary 二进制鼠标通道) + `useTerminalTheme.ts`(手动主题)。**踩坑**:go-pty Windows 下把相对 argv0 拼到 Dir 上不查 PATH → `servercore.resolveCommandPath` 必须对裸命令名做 `exec.LookPath` 兜底(codex 等 PATH 安装的命令);Unix `findInDir` 要求可执行位防同名普通文件遮蔽;**Win10 inbox ConPTY 不透传 alt-buffer/鼠标开启序列且丢弃鼠标输入(探针实证)** → Windows 宿主上 TUI 滚轮无响应属 OS 限制,Linux 部署目标不受影响。

---

## ⭐ 交接手册（新会话从这里接手）

> **一句话现状（换设备从这里接）**：**前端从零重写 + EP/VXE 卸载已收口**。业务页已去 EP；`element-plus`/`@element-plus/icons-vue`/`vxe-*` 已卸包；demo/extended/假 analysis·monitor/旧布局壳已删；SSH 列表无持久 status（测试在编辑弹窗草稿探测，连接外侧按钮，`a8122d4`）；toast 压过弹窗 `z-index:10000`（`f2d30e9`）。关键提交：`7ae41db` share/oidc · `c6a5c29` 文件模块 · `6398d24`/`1dbebed` 死代码+菜单 · `f0376a1` BaseDialog · `2fbd154` feedback · `c289196` 卸包 · `a8122d4` SSH · `7b98516` AppSelect/CommandSearch · `f2d30e9` toast 层级。**下一步**：暗色/a11y/i18n 全量核对；终端移动浅色；实例文件页；可选 `--el-*`→`--app-*`。⚠️ 改 Go 后需用户在 GoLand 重跑后端。

### ✅ 已提交 `c8d08f2`：Sub2API 后端「统计聚合全下沉 ent + 接口按模块拆分 + 列表去闪」
> 铁律见记忆 [[sub2api-backend-rewrite-mandates]]。管理端：`GetSub2APIAdminTotals/Trend/Top` + recent `RecordFilter`；ent 聚合；tps≥20；token Top；`bucket15m`；DataTable 重载不换骨架。浏览器已验。

### ✅ 已提交 `ea7c1d4`：公开门户 home 拆接口 + stats 去 EP 全量重写（2026-07-20）
> 用户点名 home 包太大 + 禁止整页等请求；stats 为详情页可明确 loading。

**后端**
- `Sub2APIHome` **删除 `snapshot`**；新 RPC `GetPublicSub2APIOverview` → `GET /public/sub2api/overview`；`PublicHome` 仅元信息+公告/时间线。
- `make api` 已回。

**前端 home**
- 三请求并行 `loadPublicHome` / `loadPublicOverview` / `loadStats(1)`；乐观外壳；hero 未到显示 `—`。
- 去 EP：令牌 btn + Heroicons + StatusPill + EmptyState + FormDialog。
- 顶栏仅 **语言 / 主题 / 公告**（去掉与 hero 重复的用量详情/前往控制台）；hero 主 CTA 文案统一「用量详情」并 `goStats()`。

**前端 stats**
- 去 EP：`LanguageMenu`+`AppIconButton`+返回首页；`MetricStrip` 单色 + **`MetricItem.icon` Heroicons**；令牌 seg；桌面语义表 / 移动 rank；**首屏明确 loading「正在加载统计…」**（详情页，非门户渐进 `—`）；切区间 reload 条。
- 区块 stagger `.reveal`；`prefers-reduced-motion` 关闭。

**横切**
- `AppDropdown`：`left`+宽度 + visualViewport 钳制，防语言菜单飞出屏外。
- 公开页移动顶栏 **单行 nowrap**（home 窄屏隐藏顶栏文案 CTA）。

**验收**：桌面 1440 暗/浅 + 移动 360/390 + 语言菜单不越界 + 区间 7→1 + ep=0 + home/stats/overview 200；vue-tsc/eslint 绿。

### ✅ 已提交 `9c963f8`：`public/sub2api/imagine` 去 EP + `lottery` 顶栏对齐
> 规格见 `06e`。后续 public share/oidc、EP 卸包、SSH 等见会话日志与上方「一句话现状」。

### 如何启动 / 可视化验证
- 前端 dev：`cd frontend && pnpm dev`（`:3007`，占用则提示端口）。登录 `admin / admin`；后端 API `:22633`。
- 浏览器 MCP `chrome-devtools`（local，autoConnect）：`navigate_page` / `take_screenshot` / `resize_page` / `evaluate_script`。**每页验收：桌面 1440×900 + 移动 390×844，明+暗。** 截图落 `.browser-tmp/shots/`（gitignored）。
- ⚠️ `.app-content` 是**内部滚动容器**（非 window）：截"整页底部"要 `document.querySelector('.app-content').scrollTop = ...`，`fullPage` 只截到视口。

### 新增文件清单（读这些即懂样板架构）
- **地基**：`vite.config.ts`（`@nuxt/ui/vite` `ui()`，auto-import/components 并入其中）、`src/main.ts`（导入 `design-tokens.css` + `@nuxt/ui/vue-plugin`）、`src/App.vue`（`<UApp>` 包裹）、`src/styles/index.css`（`@import '@nuxt/ui'`）、`src/styles/design-tokens.css`（**令牌核心**）、`src/stores/theme/index.ts`（默认色 `#14B8A6`）
- **外壳**：`src/layouts/index.vue`（渲染 `AppShell`+`ThemeConfig`）、`src/layouts/app/*` = `AppShell/AppSidebar/AppNav/AppNavItem/AppTopbar/AppTabs/AppContent/CommandSearch/LanguageMenu/NotificationMenu/UserMenu`
- **基础组件**：`src/components/ui/*` = `AppPanel/SectionHeader/StatusPill/AppAvatar/AppIconButton/MetricStrip/MetricItem/EntityCard/EmptyState/DescriptionList/AppDropdown`（全令牌驱动、自动全局注册）
- **工作台**：`src/views/dashboard/home/*` = `index.vue` + `runningInstanceSection` + `shortcutSection` + `systemRealtimeCharts`（ECharts 逻辑保留）。`welcomePanel`/`systemOverviewCards` **已删除**。
- **反馈**：`src/utils/feedback.ts` + `components/FeedbackBridge.vue`；`App.vue` 上 toaster `z-[10000]`。

### 接手要点 / 已踩坑（务必先看）
1. **单实例 unplugin**：配置**并入 `ui({autoImport,components})`**，**别再加独立实例**。
2. **暗色**：`ui({colorMode:false})`，`.dark` 由 `themeStore` 独占；`design-tokens.css` 在 `main.ts` 导入（已无 EP 暗色 css）。teal-500 == 薄荷 `#14B8A6`。
3. **命名避让**：自建组件勿与全局撞名；图标按钮用 `AppIconButton`。
4. **多根 class 不透传**：`AppDropdown` 含 `Teleport` → 响应式隐藏包普通 div。
5. **内容区无 keep-alive**：`:key = tabsStore.getRouteRenderKey(fullPath)`；轮询 `onBeforeUnmount` 停。
6. **store 契约不改**：只换壳。图标 `menuStore.iconComponents['HOutline:XxxIcon']`；`Element:` 仅兼容映射。
7. **SSH 测试**：只测编辑弹窗草稿；列表无 status；连接外侧按钮。

### P1 列表/CRUD 范式（已定稿 · `system/user` 样板 2026-07-11）
**页结构**（`views/system/user/index.vue`）：`PageHeader`(标题+描述+`新增用户`) → `FilterBar`(用户名/状态+搜索/重置) → 批量条(选中出现，`PERM.USER_DELETE`) → 卡/表切换分段控件(桌面；移动强制卡) → `EntityCard` 卡片流 / `DataTable` 表视图 → `Pagination` → `FormDialog`(create.vue)。删除走 `Dialog.confirm`；toast 用 `feedback`/`useFeedback()`。
**新增 `components/ui/`**：`PageHeader`、`FilterBar`(移动折叠)、`Pagination`(移动简化)、`DataTable`(令牌驱动语义表格：选择/sticky 表头/hover/空/加载)、`ActionMenu`(基于 `AppDropdown` 的行内⋯)、`FormDialog`(令牌驱动模态外壳)。表单控件用 `design-tokens.css` 里的全局 `.app-input/.app-select/.app-textarea/.app-label` + `AppSelect`/`AppSwitch`。
**定稿取舍**：
- `DataTable` **自建令牌表格**（非 `UTable`/TanStack）——完全掌控观感；桌面表 / 移动卡。
- 表单字段用**原生令牌化控件** + `AppSelect`；`UButton` + `i-lucide-*`。
- 校验多为内联函数；toast 统一 `feedback`（`FeedbackBridge` 注入，层级高于弹窗）。

### 下一步（重写收口后）
- 暗色 / 可访问性 / i18n 全量核对；终端页移动+浅色；实例文件 `instance/files/:id` 若仍需。
- 可选：`--el-*` CSS 变量改名 `--app-*`；`FilterSheet`；valibot schemas。

---

## Phase 0 · 地基(逐项)
- ⬜ 接入 spike:装 `@nuxt/ui` + `@iconify-json/lucide`+`heroicons`;Vite 插件;CSS 入口;`<UApp>`;`UButton` 跑通(见 `03b` §2)
- ⬜ 移除 EP 引导:`main.ts` EP 导入、`ElementPlusResolver`、`optimizeDeps` EP、`el-config-provider`→`<UApp>`
- ⬜ `design-tokens.css`:按 `01` 数值覆盖 `--ui-*` + `@theme`;主色 `#14B8A6` 可切换(改造 `themeStore.setPrimaryColor`→`--ui-primary`)
- ⬜ 图标:`config/iconMap.ts`(旧名→iconify);`UIcon`;删 `iconRegistry.ts`
- ⬜ 横切:`utils/feedback.ts`(toast)、`confirmDanger`、`useOverlay` 弹窗、`schemas/*`(valibot)
- ⬜ `components/ui/*`(按 `03`):IconBox/StatusPill/Chip/AppAvatar/Skeleton系/PageHeader/SectionHeader/AppPanel/Toolbar/MetricStrip+Metric/DataTable/EntityCard/DescriptionList/EmptyState/ErrorState/Pagination/FilterBar/FilterSheet/FormDialog/ActionMenu/FabButton
- ⬜ 明暗两态令牌核对

## Phase 1 · 外壳(逐项,见 `04`)
- ⬜ `layouts/app/AppSidebar` + `AppNav` + `AppNavItem`(手写导航,分组/激活/折叠/多级/权限过滤/图标/i18n)
- ⬜ `AppTopbar`(标题+面包屑+搜索+操作+头像;`I18nDropdown/NotificationDropdown/UserDropdown` 基于 Nuxt UI 重写,保留数据逻辑)
- ⬜ `AppTabs`(极简,复用 `tabsStore`)
- ⬜ `AppContent`(过渡/内边距/限宽/`fullBleed`/keepAlive)
- ⬜ `AppShell` 组装 + `layouts/index.vue` 切换
- ⬜ 移动:抽屉侧栏、压缩顶栏、隐藏/横滑标签
- ⬜ 桌面+移动+明暗截图,替换旧 `leftMode/topMode/menu/menuItem/header/tabsView`

---

## 逐页清单(Phase 2–3)

> 列:页面 · 路由 · 页型 · 规格 · 桌面 · 移动 · 暗色 · 验收

### 仪表盘/个人中心(`06a`)
| 页面 | 路由 | 页型 | 桌面 | 移动 | 暗 | 验 |
|---|---|---|---|---|---|---|
| 工作台 | dashboard/home | P4 | ✅ | ✅ | ✅ | ✅ |
| 分析页 | dashboard/analysis | — | — | — | — | ✅ **已删除**（假数据 + 菜单种子裁剪 `6398d24`/`1dbebed`） |
| 监控页 | dashboard/monitor | — | — | — | — | ✅ **已删除**（假数据 + 菜单种子裁剪） |
| 个人中心 | profile | P2 | ✅ | ✅ | ✅ | ✅ **06a 真重写+完整浏览器验收**：桌面1440 双栏/操作链路；移动390 emulate 编辑弹窗/设备卡/日志卡/下线确认；768 单列；修设备网格 minmax(0,1fr) 横溢 + 面板头窄屏；已提交 |

### 实例/基建(`06b`)
| 页面 | 路由 | 页型 | 桌面 | 移动 | 暗 | 验 |
|---|---|---|---|---|---|---|
| 应用列表 | instance/list | P1 | ✅ | ✅ | ✅ | ✅ |
| 实例控制台 | instance/console/:id | P6 | 🟡 | ⬜ | 🟡 | 🟡 PTY 直连已实机验证(键入/回显/ws 抓包);移动+浅色待验 |
| 实例文件 | instance/files/:id | P6 | ⬜ | ⬜ | ⬜ | ⬜ |
| 实例类型 | instance/type | P1 | ✅ | ✅ | ✅ | ✅ |
| Docker 容器 | docker/container | P1 | ✅ | ✅ | ✅ | 🟡 全量重写(列表+创建/编辑结构化表单+详情+TerminalConsole 日志/终端+Stats FormDialog)；浏览器桌面/移动/暗色验证；后端 docker 未启用，数据流未跑 |
| Docker 镜像 | docker/image | P1 | ✅ | ✅ | ✅ | 🟡 结构+弹窗验证（后端无 Docker 守护，数据流未跑） |
| Docker 网络 | docker/network | P1 | ✅ | ✅ | ✅ | 🟡 结构+弹窗验证（后端无 Docker 守护，数据流未跑） |
| Docker 配置 | docker/config | P3 | ✅ | ✅ | ✅ | 🟡 P3 四 Tab(状态/连接/默认/仓库) + AppPanel；浏览器验证；docker 未启用时状态正确显示 |
| API Key | node/key | P1 | ✅ | ✅ | ✅ | ✅ |
| SSH 管理 | openssh/management | P1 | ✅ | ✅ | ✅ | ✅ 列表无持久 status；连接外侧按钮；测试在编辑弹窗草稿探测（`a8122d4`） |
| SSH 终端 | openssh/terminal | P6 | 🟡 | ⬜ | 🟡 | 🟡 重写完成,htop+鼠标滚轮+初始 resize 实测通过;移动+浅色待验 |

### 工具/文件/Sub2API(`06c`)
| 页面 | 路由 | 页型 | 桌面 | 移动 | 暗 | 验 |
|---|---|---|---|---|---|---|
| 内网穿透 | tools/tunnel | P1 | ✅ | ✅ | ✅ | ✅ 全量重写(列表+创建/frps/frpc/统计) |
| 端口转发 | tools/port-forward | P1 | ✅ | ✅ | ✅ | ✅ 全量重写(列表+创建/统计) |
| 文件管理 | file/index | P6 | ✅ | ✅ | ✅ | ✅ 真·从零重写；FileEditor/FormDialog/useFeedback；`c6a5c29` 等 |
| 文件分享 | file/share | P1 | ✅ | ✅ | ✅ | ✅ 桌面/移动/暗+创建/编辑弹窗全验;**有界内容**、**无序号** |
| 文件来源 | file/source | P1 | ✅ | ✅ | ✅ | ✅ 桌面/创建弹窗(动态 OSS/FTP/WebDAV 字段+AppSelect 浮层)验;移动暗沿用已验范式 |
| Sub2API 首页 | sub2api/home | P4 | ✅ | ✅ | ✅ | ✅ 管理端去 EP + 模块拆接口（`2fb1fe0`/`c8d08f2` 等） |
| Sub2API 配置 | sub2api/config | P3 | ✅ | ✅ | ✅ | ✅ P3 三 Tab 去 EP；`useFeedback` |
| Sub2API 活动 | sub2api/activity | P2/P4 | ✅ | ⬜ | ✅ | 🟡 去 EP + 历史按发放口径；**`24fa5e7`**；移动截图可再补 |

### 系统管理(`06d`)
| 页面 | 路由 | 页型 | 桌面 | 移动 | 暗 | 验 |
|---|---|---|---|---|---|---|
| 用户管理 | system/user | P1 | ✅ | ✅ | ✅ | ✅ |
| 角色管理 | system/role | P1 | ✅ | ✅ | ✅ | ✅ |
| 菜单管理 | system/menu | P1 | ✅ | ✅ | ✅ | ✅ |
| OIDC 客户端 | system/oidc | P1 | ✅ | ✅ | ✅ | ✅ |
| 系统设置 | system/settings | P3 | ✅ | ✅ | ✅ | ✅ |
| 定时任务 | system/task | P1 | ✅ | ✅ | ✅ | ✅ |
| 操作日志 | system/operation | P1 | ✅ | ✅ | ✅ | ✅ |

### 认证/独立(`06e`)
| 页面 | 路由 | 页型 | 桌面 | 移动 | 暗 | 验 |
|---|---|---|---|---|---|---|
| 登录/注册/找回 | login | P7 | ✅ | ✅ | ✅ | ✅ 居中卡重写 |
| 初始化向导 | initialize | P3 | ✅ | ✅ | ✅ | ✅ 居中卡三步向导去 EP |
| 分享落地 | public/share/:token | P7 | ✅ | ✅ | ✅ | ✅ 去 EP；**`7ae41db`** |
| OIDC 授权 | oidc/authorize | P7 | ✅ | ✅ | ✅ | ✅ 去 EP 全量重写；**`7ae41db`** |
| Sub2API 门户 | public/sub2api/home | P7 | ✅ | ✅ | ✅ | ✅ 拆接口+去 EP；**`ea7c1d4`** |
| Sub2API 统计 | public/sub2api/stats | P7 | ✅ | ✅ | ✅ | ✅ 去 EP；**`ea7c1d4`** |
| Sub2API 绘图 | public/sub2api/imagine | P7 | ✅ | ✅ | ✅ | ✅ 去 EP 全量；**`9c963f8`** |
| Sub2API 抽奖 | public/sub2api/lottery | P7 | ✅ | ✅ | ✅ | ✅ 去 EP + 顶栏 AppIconButton；**`9c963f8`** |
| 403 | exception/403 | P8 | ✅ | ✅ | ✅ | ✅ 克制居中 |
| 404 | exception/404 | P8 | ✅ | ✅ | ✅ | ✅ |
| redirect | redirect | 工具 | ✅ | — | ✅ | ✅ 令牌页底+logo 脉冲 |

---

## 已关闭的待决 / 清理项（代码与菜单已对齐）
- ✅ `dashboard/analysis` / `dashboard/monitor` 假数据页 **已删除**（`6398d24`），菜单种子裁剪（`1dbebed`）
- ✅ `views/demo/*` / `views/extended/*` **已删除**
- ✅ 旧布局壳 `layouts/{menu,leftMode,topMode,tabsView,header,...}` **已删除**（仅 `layouts/app/*`）
- ✅ EP/VXE 彻底卸载 + 全仓运行时引用清零（`c289196`）；历史操作日志文案键 `sshHostBatchTest` 可保留（仅展示旧日志）
- 可选：ThemeConfig 的 topMode 预览（新外壳下为空操作）；CSS `--el-*` 日后改名 `--app-*`；表格默认卡/表口径维持现状（桌面表+切换 / 移动强制卡）

---

## 定稿的架构取舍（已落地，非待确认）
- **图标**：`iconRegistry` + Heroicons Proxy；`Element:` 仅兼容映射到 Hero（菜单旧数据）
- **下拉**：自建 `AppDropdown` / `AppSelect` / `CommandSearch`（同框输入+结果面板）
- **移动抽屉**：令牌 Teleport 侧栏（非 EP drawer）
- **实时图表**：ECharts 逻辑保留；壳+控件令牌化
- **ThemeConfig**：令牌侧栏；toast `z-index:10000` 压过弹窗
- **主色**：默认薄荷 `#14B8A6`（Nuxt UI primary=teal）
- **命名**：`AppIconButton`（避开旧 IconButton，旧文件已删）

## 会话日志(每次追加一行)
| 日期 | 会话做了什么 | 下一步 |
|---|---|---|
| (首个会话) | 探索期尝试(令牌/组件/仪表盘,已撤回);定方向(精致侧边栏 + Nuxt UI + EP/VXE 下线);编写完整任务书;还原基线 | 下会话从 Phase 0 接入 spike 开始 |
| 2026-07-10 | **样板落地**：Phase 0(Nuxt UI 4.9 接入+`design-tokens.css`薄荷主色+明暗)跑通;Phase 1 外壳(手写侧栏/顶栏/命令搜索/三下拉/极简标签/内容区/移动抽屉,`layouts/app/*`)+`components/ui/*` 10 个基础组件;Phase 2 工作台重写(安静问候+单色 MetricStrip+实时面板+运行实例 EntityCard/空态+快捷入口+系统信息)。浏览器验证桌面/移动×明/暗、抽屉、下拉、徽标均 OK,零控制台报错,新增文件 lint 绿。见"务实取舍"。 | 等用户确认设计方向;确认后据反馈修订计划,再做 用户管理(列表页样板)与全量推广 |
| 2026-07-11 | **用户管理列表页样板（P1 定稿）**：新增 `components/ui/` 六件套（PageHeader/FilterBar/Pagination/DataTable/ActionMenu/FormDialog）+ 全局令牌化表单控件；重写 `system/user`（卡/表切换、批量删除、Dialog.info 删除确认、内联校验的 create/edit FormDialog），保留 userPage/deleteUser 接口与 PERM 权限。浏览器验证桌面卡/表、创建/编辑弹窗+校验、行内⋯菜单、批量条、分页、移动强制卡+筛选折叠、明/暗全 OK，零控制台报错，lint+tsc 绿。见"P1 范式"与务实取舍。 | Phase 3 全量推广（从 `system/role` 起） |
| 2026-07-11 | **Phase 3 首页-角色管理 `system/role`**：按 P1 范式重写 index（卡/表切换、批量删、Dialog.info 确认）+ create（FormDialog+校验）。新建自建令牌组件 **`components/ui/PermissionTree.vue`**（三态勾选树替代 el-tree，v-model=扁平 menuIds）；`DataTable` 加 `rowSelectable` 逐行禁选（内置角色锁图标、不可选/编辑/删除）。修复原表单 description 未必填导致的 500（后端 `NotEmpty`）→ 改必填。i18n 补 role.title/pageDesc/selectedCount/clearSelection/emptyDesc/descriptionRequired + 树控件 selectAll/clearAll/expandAll/collapseAll（三语）。浏览器验证：桌面卡/表、创建/编辑权限树全链路（父子联动+半选+后端往返恢复）、批量删、移动强制卡+折叠筛选、明/暗全 OK，lint+tsc 绿。 | 接 `system/menu`（菜单管理，表格主型）继续 Phase 3 |
| 2026-07-11 | **用户反馈-头像收敛**：原三处头像(顶栏右上角 `UserMenu`、工作台问候区、侧栏底部)统一收敛到**仅侧栏底部**。`UserMenu` 加 `variant='topbar'\|'sidebar'`——侧栏整行触发器、向上弹出,展开显示 用户信息/操作/版本号;`AppDropdown` 加 `side='top'`(向上弹)+`block`(填充 flex 父)。顶栏移除 `UserMenu`、工作台移除 `AppAvatar`。下拉底部常驻"版本号: dev";原"版本号"动作行改叫`layout.checkUpdate`(zh-CN/zh-TW/en 已补)避免重复。浏览器验证桌面(展开/折叠)×明/暗 + 移动抽屉均 OK,零控制台报错,lint+tsc 绿。 | 用户管理(列表页样板);顶栏右上角现仅剩 通知铃(用户菜单已下移) |
| 2026-07-11 | **菜单管理 `system/menu`（树表）**：`DataTable` 加 `tree` 树表模式（缩进+展开插入符+扁平化 displayRows）；index 用树表 + 客户端筛选（保留匹配祖先），create 用 FormDialog+令牌字段（类型/父级/条件路径/权限/图标/排序/状态+内联校验），父级选择用**扁平缩进下拉**（后接改为 AppSelect）规避 el-tree-select 嵌套层级；复用 IconSelectorDialog（spacious）。i18n 补 menu.title/pageDesc/topLevel/emptyDesc（三语）。 | 用户反馈"密度" → 全局紧凑化 |
| 2026-07-11 | **用户反馈-密度太松 + 原生 select 直角浮层（两次强调，所有已重写页通病）**：全局紧凑化（`design-tokens.css` 控件 36→32/字号 14→13/label 13→12；`DataTable` 行 padding 11→7px；`FilterBar` 16→12px 去阴影；段间距 16/14→12/10；`AppIconButton` 加 `box`、`ActionMenu` 用 28px → 表行 63→43px）。新建 **`components/ui/AppSelect.vue`**（令牌圆角下拉，teleport z-index 2300、泛型 v-model 类型安全、`fit`/`searchable` 本地过滤），**替换全部原生 `<select>`**（两筛选栏/菜单父级/用户角色/分页每页）。记忆 [[compact-density-appselect]]。**操作日志 `system/operation`**（P1 只读）：PageHeader+FilterBar（可搜索用户/类型 AppSelect+布尔结果+路径）+DataTable/移动卡+Pagination+详情 FormDialog(JSON+复制)；用户预加载一次做 userId→用户名 映射。i18n 补 operation.title/pageDesc/emptyDesc/allUsers/allTypes（三语）。浏览器验证 menu(树表/创建弹窗内 AppSelect 浮层高于弹窗/筛选) + role(卡/分页 AppSelect) + operation(搜索下拉/详情弹窗) + 明暗，lint+tsc 绿。 | 继续 Phase 3：`system/task` 或 `system/oidc` |
| 2026-07-11 | **用户反馈-总条数显示两次**：`Pagination` 组件自带的“共 N 条”与各页顶部 `X-page__bar-hint` 的总数重复（user/role/operation 都中招；menu 是树无分页故单次）。**统一从 `Pagination` 移除总数**（一处改动全解决）→ 全部页面总数只在顶部显示一次。**定时任务 `system/task`（P1 只读+行内操作）**：PageHeader(+刷新)+FilterBar(关键字+状态 AppSelect)+DataTable/移动卡+Pagination；令牌进度条（track+fill 按状态着色）、种类/状态 StatusPill、行内 `ActionMenu` 取消/重试/删除（按状态条件显示：active→取消、failed→重试、terminal→删除；删除走 Dialog.info）、2.5s 静默轮询。i18n 补 taskManager.pageTitle/pageDesc/allStatus/emptyDesc（三语）。浏览器验证桌面(表/进度/条件操作菜单)+移动卡+暗色，总数单次，lint+tsc 绿。 | 继续 Phase 3：`system/oidc`（系统管理最后一个列表页） |
| 2026-07-11 | **OIDC 客户端 `system/oidc`（P1，系统管理最后一个列表页）**：按 P1 范式重写 `index.vue`（PageHeader+两操作按钮 OIDC配置/生成客户端 → FilterBar → DataTable/移动卡 → Pagination）+ 三个子件：`configDialog.vue`（服务端配置：AppSwitch 启用 + Issuer/当前域名 + 3 列 TTL + 只读端点复制区）、`clientForm.vue`（创建/编辑：名称/回调必填内联校验 + Scopes + AppSwitch 状态；保存后若返回**完整** secret(无`*`)则 emit reveal）、内联 reveal `FormDialog`（完整 Client Secret 仅一次 + 13 端点字段 + 单项/全部复制）。行内 编辑/刷新密钥/删除（Dialog.info 确认）。新建自建令牌组件 **`AppSwitch.vue`**（替代 el-switch）。文案硬编码中文（唯一 i18n key=`system.common.total`；Phase 4 统一）。保留 list/create/update/delete/refreshSecret/getConfig/updateConfig 契约 + `PERM.OIDC_EDIT`。**浏览器全链路验证**：桌面暗(列表/配置弹窗/创建→reveal 完整secret/行菜单/校验/删除确认)+桌面浅+移动390(强制卡/折叠筛选/简化分页)，创建真实客户端走通后删除清理，零控制台报错，vue-tsc+eslint 绿。**⚠️ 本页及 role/menu/operation/task 全部 Phase 3 改动仍未提交。** | Phase 3：`system/settings`(P3 配置型) 或转 实例/文件/Docker 模块；提交 Phase 3 批次 |
| 2026-07-12 | **提交系统设置**（`f67e374`：`system/settings` P3 双 Tab + AppPanel 页脚 flex + i18n）→ 系统管理模块 100% 收口。**转 实例模块：应用列表 `instance/list`（P1，旗舰页）全量重写**：4 文件套 P1 范式——`index.vue`（PageHeader+FilterBar+批量条+卡/表切换+DataTable/卡片流+Pagination+受控 InstanceEditor，薄壳保留 `useInstanceListStore` 全部状态/动作契约与业务处理）、`instanceCard.vue`（EntityCard+StatusPill+ActionMenu+UButton，去 el-button/checkbox/dropdown/icon）、`instanceEditor.vue`（FormDialog+令牌字段+AppSelect+AppSwitch+内联校验，保留受控 props/emits）。表视图行操作收进 ActionMenu（控制台/配置/启动或停止/强制重启/文件管理/删除，删除按 owner 隐藏）。i18n 补 instance.title/pageDesc/allTypes/allStatus/emptyDesc/createTime（三语）。浏览器验证桌面暗（卡/表切换、行菜单全项、配置弹窗 fetch 回填 + 类型 AppSelect 浮层高于弹窗）+移动390（强制卡/折叠筛选/简化分页），零控制台报错，vue-tsc+eslint 绿。**⚠️ 用户第三次密度反馈**："总览占用大量空间 + 大组件大留白"——**删掉大号 MetricStrip 总览带**（P4 仪表盘才配），把 当前页运行/停止 计数**内联进列表工具条**（`共 N 条 · ●运行中 X · ●已停止 Y`，总数不再与「总实例」重复），删 `overviewStats.vue`。见 [[compact-density-appselect]] 新增「列表页禁大指标带」条。**未提交。** | 转 实例其余（`instance/type` 已存在待核）与 Docker/Node/工具/文件 P1 列表；套本页范式 + 密度铁律 |
| 2026-07-12 | **Docker 列表两页（image/network）**：按 P1 范式重写 `docker/image`（PageHeader[任务/拉取镜像]+FilterBar[关键字/悬空 AppSelect]+DataTable/移动卡+Pagination；拉取/打标签/编辑标签/详情[令牌 KV+chips]/历史[DataTable] 全部 FormDialog；行内 ActionMenu 详情/编辑标签/打标签/历史/删除，管理权限 + 容器数门控）与 `docker/network`（同范式+创建[名称/驱动 AppSelect/内部·IPv6·可附加 AppSwitch/子网/网关]/详情[hero+KV+IPAM 列表+标签+已连接容器带断开]/编辑[标签+重建开关→条件驱动/强制/子网/网关] FormDialog）。**沿用** `DockerTaskDialogs`（拉取/重建进度，迁移期 EP 保留）。i18n 三语补 docker.image/network title+pageDesc+emptyDesc。**后端无 Docker 守护** → 仅验证页面结构+空态+弹窗渲染（拉取/创建弹窗桌面暗已截图），行/详情数据流未跑；vue-tsc+eslint 绿。**未提交前项已提交 `c6e1a14`；本两页待提交。** 剩 `docker/container`（70KB 巨页：列表+创建表单+exec/logs/stats 终端弹窗，exec/logs 应复用 TerminalConsole）+ `docker/config`(P3)。 | Docker 容器(大)+配置；或转 工具/文件 | 
| 2026-07-12 | **P1 推广三页 + 新增 UserPicker**：①**实例类型 `instance/type`**（PageHeader+FilterBar[名称/状态]+内联计数[启用/禁用]+DataTable/移动卡+FormDialog[名称+AppSwitch 启用]，内置类型锁定，删除走 Dialog.info）。②**API Key `node/key`**（同范式+原生 `datetime-local` 过期时间+AppSwitch 永久有效[仅编辑]+复制 FormDialog[完整值一次+警示条]；行内 编辑/复制/刷新；**无删除接口**属原样保留）。③**SSH 管理 `openssh/management`**（可选表+批量测试/删除+行内 连接/测试/编辑/分享/删除[仅创建者]+分享 FormDialog；create 大表单 authType seg+条件密码/密钥[眼睛显隐]+patch-diff 更新；createTime 格式化修复宽度溢出）。新增自建 **`components/ui/UserPicker.vue`**（多选用户：预载全部用户 chips+可搜索 AppSelect 追加，替代 el-select multiple remote，SSH create/分享共用）。**踩坑**：`system.common.selectAll` 不存在（selectAll 在别的 `common` 命名空间）→ 移动端全选按钮显示原始 key，改用 `ssh.common.selectAll`（自加）。i18n 三语补 instance.type*/node.key.title+pageDesc/ssh.common.title+pageDesc+selectAll 等。**浏览器全验**：三页桌面暗/浅+移动，CRUD/弹窗/校验/批量/分享全链路，vue-tsc+eslint 绿。**未提交。** | 转 Docker(容器/镜像/网络+config) 或 工具(tunnel/port-forward) 或 文件(share/source) P1 列表 |
| 2026-07-11 | **提交 Phase 3 批次**（`902082e`：role/menu/operation/task/oidc + AppSelect/AppSwitch/PermissionTree + 密度化 + i18n，24 文件）。**系统设置 `system/settings`（P3 双 Tab 配置页，系统管理最后一页）**：EP 整页重写为 PageHeader + 令牌 seg Tab 条（安全与认证/邮件配置）+ `AppPanel`(flush) 分组 + `.set-row`（label+desc / 控件，分区保存页脚）。控件：`AppSwitch`(替 el-switch)、令牌 `.app-input`/number/`.app-textarea`、`AppSelect`(替 el-select，模板类型；`@change` 不存在→用 `watch` 承接重载)、密码眼睛显隐、占位符 chips 快捷插入。3 个 `FormDialog`：测试邮件/模板测试(动态占位符字段)/预览(iframe srcdoc + 占位符黄底高亮)。保留全部 login/email/template API + `PERM.SYSTEM_CONFIG_EDIT` + 占位符插入/渲染逻辑。i18n 补 `settings.pageTitle/pageDesc`(三语)。**踩坑**：模板测试字段标签 `{{ '{{.'+name+'}}' }}` 在 Vue 文本插值里字面 `{{` 触发 eslint 解析错误 → 抽 `fieldToken()` helper 返回该串。**浏览器全链路验证**：桌面暗/浅(双 Tab、开关/输入/密码眼睛、AppSelect 切换 watch 重载、预览 iframe 高亮、模板测试动态字段、空收件禁用发送)+移动390(行竖堆/Tab 撑满/控件全宽)，零控制台报错，vue-tsc+eslint 绿。**⚠️ 未提交。** | Phase 3 转其余模块：实例/Docker(多 P1 列表)、文件分享/来源、工具、仪表盘 profile/analysis/monitor |
| 2026-07-13 | **文件模块两页 P1 + 两条新铁律(用户点名)**：①**文件来源 `file/source`**（index + 新建 `create.vue`：FormDialog + 按类型动态字段 OSS/FTP/WebDAV + AppSelect/AppSwitch + 页脚测试连接；行内 测试/编辑/删除；内联启用/停用计数）。②**文件分享 `file/share`**（PageHeader+FilterBar+DataTable/移动卡+Pagination；有效/停用/过期/达上限 状态胶囊；复制链接/编辑/删除）+ **`ShareFormDialog` 从 EP BaseDialog 重写为 FormDialog**（令牌字段 + 原生 datetime-local + AppSwitch + 创建成功详情复制视图；保留 FilePicker 跨来源选择）。i18n 三语补 fileSource.title/pageDesc/keyword… + file.share.pageDesc/status*/content/itemsCount/selectedCount…。**用户中途两次点名的通病 → 提炼成 `01` 铁律 4「为使用者策展,不照搬 API」**：(a)**可变长内容有界化**——分享多文件不再逗号铺满/chips 无限堆叠,列表列改「`N 项 · 首名`(全清单走 title 悬浮)」、编辑弹窗改「已选 N 项 + 定高滚动列表 + 逐项移除」;(b)**去系统内置量**——`DataTable` 的 `seq` 序号列**全量下线**(14 个已重写列表页 sed 清除:docker×3/instance×2/node/ssh/system×4/tools×2/file×2)。踩坑:FormDialog 自定义 `#footer` 必须保留默认 `show-footer=true`(slot 在 `v-if="showFooter"` 内,设 false 会连按钮一起吞掉);`noUncheckedIndexedAccess` 下 `list[0]` 要先取局部变量再判空。浏览器全验(桌面/移动/暗+弹窗),vue-tsc+eslint 绿。**未提交。** | 主攻 `file/index` 文件管理大重写(用户点名);顺带 review 前两次提交(tunnel/port-forward + Docker) | 
| 2026-07-12 | **Docker 全模块收口**：①**容器 `docker/container` P1 全量**——PageHeader+FilterBar+DataTable/移动卡+Pagination；创建/编辑 FormDialog 结构化控件（`AppCombobox` 镜像、重启策略 seg、内存/CPU chips+数字、`ContainerPort/Mount/EnvEditor`）；详情令牌壳；日志/终端复用 `TerminalConsole` 浮层；统计 FormDialog+echarts。②**配置 `docker/config` P3**——四 Tab（状态/连接/默认/仓库）+ AppPanel+AppSwitch。③新组件 `AppCombobox` + 三个 spec 编辑器。**反人类修复**：列表失败置 `dockerUnavailable`，创建弹窗不再二次请求镜像/网络选项（拦截器 500 会连弹 toast）。浏览器：桌面/移动/暗色+创建表单结构化行 OK；后端 docker 未启用。vue-tsc 绿。**待提交本批。** | 提交 Docker 批次后停（用户指定）；下一会话 工具 tunnel/port-forward 或 文件 share/source |
| 2026-07-15 | **登录门面 `login` P7 重写（用户嫌半屏渐变难看 → 改居中卡）**：四文件去 EP——`index` 冷灰页底 + 居中 hairline 卡 + 角标(语言/主题配置/明暗)；`accountLogin/register/forgotPassword` 令牌 `.app-input` + 内联校验 + 密码眼睛内嵌 + `useFeedback` toast。**不做**半屏主色品牌区（空、廉价）。桌面 1440 / 移动 390 **均垂直居中**；暗色随 `themeStore`。登录 admin 链路 OK。i18n 补 slogan/copyright/toggleTheme/showPassword/hidePassword/passwordLabel（三语）。**未提交。** | 提交登录批次；下一页 `initialize` 或 个人中心/监控/Sub2API/public |
| 2026-07-15 | **初始化向导 `initialize` P3 重写（对齐登录门面）**：单文件去 EP——冷灰页底 + 居中 hairline 卡 + 角标(语言/主题/明暗)；自建紧凑步骤条(完成√/当前主色)；`AppSelect` 库类型 + 令牌表单(远程库条件字段+密码眼睛) + 内联校验；确认摘要 DescriptionList 风 + 结果态(成功/失败/轮询旋转)；`useFeedback` toast；版权复用 `login.copyright`。业务契约保留(status/test/confirm + 重启轮询)。i18n 补 `pageTitle/pageDesc`(三语)。浏览器：mock 拦截 `code:200` 全链路(SQLite→管理员→确认→完成)、桌面/移动/暗色，vue-tsc+eslint 绿。**未提交。** | 提交 initialize；下一页 个人中心 `profile` / 异常 403·404 / public/share 或 Sub2API |
| 2026-07-15 | **异常/工具页 P8**：`exception/403`/`404` 去掉 ~17KB 彩色插画 SVG，改为克制居中（大号码 + 标题 + 说明 + 回首页/上一页）；`redirect` 令牌页底 + logo 脉冲。i18n 补 forbiddenTitle/notFoundTitle/goHome/goBack（三语）。浏览器桌面/移动/暗色验，vue-tsc+eslint 绿。**已提交 `8f54c27`（与 initialize 同批）。** | 个人中心 `profile` |
| 2026-07-15 | **个人中心 `profile` 第一次换壳**（五 Tab 平铺，未对齐 06a）→ 用户点名返工。 | 按 06a 真重写 |
| 2026-07-15 | **个人中心 `profile` 按 06a 真·从零**：①克制浅主色条头部+编辑资料/改密 ②**左栏常驻**档案 DescriptionList + 桌面内嵌编辑；**右栏 Tabs** 仅 消息→权限→设备→日志（`personalInfo` 移出 store tabs） ③设备 **EntityCard 网格** + AdaptiveConfirm 下线 ④消息/设备危险操作用 AdaptiveConfirm ⑤移动：头压缩、Tab 横滑、列表转卡、**编辑 FormDialog 近全屏** ⑥改密 FormDialog+内联校验；去掉未实现危险区注销。i18n 补 editProfile/logoutDevice/thisSession。**完整浏览器验收**（用户要求后补做）：桌面1440 保存确认/一键已读/发消息/改密校验/下线确认/日志详情；移动390 编辑弹窗+设备卡+日志卡+移动下线 Dialog；768 单列无横溢；修 `dev-grid` `minmax(0,1fr)` 与 AppPanel 窄屏 head wrap。vue-tsc+eslint 绿。 | 下一页 监控/Sub2API/public |
| 2026-07-13 | **文件管理 `file/index` 真·从零重写 + 用户点名两条刚性标准**：全套文件组件除净老框架（EP-free）：`FileBrowser.vue`（外壳=来源 AppSelect/前进后退上级/面包屑/刷新 + 双栏 左树+右列表·网格 + 工具栏 UButton/AppDropdown更多/搜索/筛选/视图段 + 选择条 + 页脚 Pagination；全出血 `flex:1;min-height:0` 填满高度[修 278px 塌陷]；图标走 `menuStore.iconComponents['HOutline:*']`；删除确认=内联令牌 FormDialog）、`FileTree/FileTreeNode`（→ 可覆盖 `--ft-*` 局部变量+app 令牌兜底，heroicons）、`FilePromptDialog/FileMediaDialog/FileUploadDialog`（→ FormDialog+app 令牌+AppSelect）、**`FileEditor` 重写**（自包含 `--fe-*` 独立明暗主题[非旧 fm-*]、menuStore+本地保存 SVG、主题**分段**替代旧 FileMenu、`useFeedback`+Promise 门闸令牌确认）、`useFileUpload`（1 处 ElMessage→fb）。**新建 `src/utils/feedback.ts` = `useFeedback()` 封装 Nuxt UI `useToast`（全项目首个采用者，已浏览器实证渲染）**；`ElMessage`→`fb`、`showRequestError`→`fb.error(getRequestErrorMessage())`、`Dialog.confirm`→令牌 FormDialog。**删** orphaned `FileMenu.vue`；`icons.ts` 回退 HEAD（去 2 个没用的 view 图标）。**踩坑**：移动端表头 类型/时间 列 `.fb-thead .fb-col{display:flex}`(0,2,0) 盖过 `.fb-col--type{display:none}`(0,1,0)→ 表头列不隐藏挤压，改用 `.fb-thead .fb-col--type` 提特异性。**保留** `theme.css`(fm-*，薄荷)——仍服务未重写的 `FilePicker/FileViewerDialog/FileDialog/FilePager`(public/share 用，增量待迁)。**浏览器验证**：桌面浅/暗、网格、移动(tree 隐藏/单列/无横溢)、toast 渲染、全出血填高、新建文件夹 FormDialog；vue-tsc+eslint 绿。**未提交。** 记忆 [[rewritten-components-drop-all-legacy-framework]]。**浏览器 MCP 坑**：chrome-devtools 读默认 profile 的 DevToolsActivePort；用户 Chrome 无 debug 端口→ 另起隔离 debug Chrome `:9222`（`$TEMP\momoko-devtools-profile`）+ 把默认 profile 里陈旧 DevToolsActivePort 改指 9222。 | ①**用户重跑 Go 后端**以吃到 `pkg/file.Create` 吞错修复 ②提交本批文件管理重写 ③Monaco 自动化键入未通（保存 dirty 态未 E2E 到，手工点一下即可）④Docker 复查修复（容器网络列无界/Stats echarts 暗色/硬编码「连接」/engine-down 空态）⑤剩余页面 |
| 2026-07-20 | **Sub2API 配置 `sub2api/config`（P3，去 EP 全量重写）**：`el-tabs/el-card/el-form/el-switch/el-input(-number)/el-checkbox-group/el-tag/ElMessage/v-loading` → 令牌三 Tab 分段(连接/首页/生图·`menuStore` 图标) + `PageHeader`(`#actions` 挂同步状态 `StatusPill`，`store.statusType` 的 `danger`→`error` 映射) + `AppPanel`(flush)+`set-row`(label+desc) + `AppSwitch` + 令牌 `.app-input/.app-textarea` + adminApiKey 眼睛显隐(`AppIconButton`) + **展示分组 toggle-chip 多选**(选中 teal 描边+淡底+勾) + **允许站点可移除 chip + 行内追加输入**。toast 用 `useFeedback`。保留全部逻辑/接口(`useSub2APIStore.loadAdmin/testConfig/saveConfig`、`configForm`、`groups`、`PERM.SUB2API_EDIT`、`__deleted__` 分组合并)。`configPageTitle` 补三语(原页标题误用「连接配置」)。**浏览器全验**：桌面1440(连接/首页/生图) + 移动390 emulate(连接/首页) + 明+暗 + 真实后端(已连接、正常状态、真实分组回填) + 追加/移除站点交互；vue-tsc + eslint 绿。**⚠️ 核对 `stores/dashboard/monitor.ts`：监控页 100% `Math.random` 假数据 → 登记 Phase 4 待决(勿精修)**。**未提交。** | Sub2API 首页 `sub2api/home`(P4 大页) / public/*；或 Phase 4 清 demo + 卸 EP |
| 2026-07-20 | **Sub2API 管理首页 `home` 去 EP 重写并提交（`2fb1fe0`）**：PageHeader(#actions StatusPill+同步)+令牌三 Tab(概览/公告/时间线)；概览=range seg+datetime-local+**MetricStrip 单色(修原彩色边框违规)**+AppPanel 图表(保留 ECharts)+最近请求 DataTable/移动卡+Pagination+详情 FormDialog(KV)；公告/时间线=DataTable/卡+ActionMenu+FormDialog CRUD。桌面/移动/明暗+真实后端全验。**随后用户提 3 诉求→牵出后端大重构（见上 🚧 区块）**：Top 图表改 token 且按 token 排序、最近请求多维筛选、tps 剔除 output<20、**全部聚合下沉 ent(禁 SQL/禁内存)、页面按模块拆接口**。已落：前端 token-Top+recent 筛选前端侧、proto optional 筛选字段(regen)、ent `bucket15m` 列(regen)+写入、`pkg/types` 接口重设计、`internal/data` 全部 ent 聚合方法实现、snapshot.go 排序改 token。**⚠️ 未完成：snapshot.go 仍调已删的 `RecordsSince` → 构建断裂；proto 管理端接口未拆；前端未拆 loadAdmin* + 未加 FilterBar。记忆 [[sub2api-backend-rewrite-mandates]]。** | **先修构建**：按 🚧「剩余步骤」1–7 完成后端拆分+前端拆分+FilterBar，用户重跑 Go+全量重同步回填 bucket15m，再全验 |
| 2026-07-20 | **Sub2API 后端聚合重写全链路修好（构建断裂→全绿）**：①`snapshot.go` 去内存聚合、`BuildSnapshot/BuildStats` 改调 ent 聚合方法（`AggregateTotals/DailyTrend/TopItems/IntradaySeries/RecordsPage`）+映射助手（`mapTopItems/dailyTrendPoints/intradayTrendPoints/todaySeriesPoints/trendDayRange/dayStart`），删 `BuildRangeStats/totalsFromRecords/build*/records*/isRateEligible/tpsEligible`+`sort` import。②proto 删 admin `GetSub2APIStats`、加 `GetSub2APIAdminTotals/Trend/Top`（`/stats/totals\|trend\|top`），`make api`。③`pkg/service.go` 加 `adminWindow`+`AdminTotals/Trend/Top`、`RecentRequests(...,filter)`。④biz+service 3 handler + `RecordFilter{req.Model/GroupName/AccountName/Outcome(*string)}`。⑤前端 `api` 加 3 fn、`store` 拆 `adminTotals/adminTrend/adminModels/adminGroups`+`loadAdmin{Totals,Trend,Top}`（各自 loading）+`loadAdminRecent(...,filter)`+`buildStatsCards` 放宽 `StatsCardSource`、Top 图 `slice(0,10)`、下拉用全量枚举、`home` 三面板 `panel-loading`+最近请求 `FilterBar`(model/group/outcome 即时 + account 回车，改 filter 回第 1 页)+i18n `allModels/allGroups/allOutcomes/filterAccount` 三语。**`go build`/`vet`/`test`+`vue-tsc`/`eslint` 全绿；LSP "is not a type" 系 proto 重生后 gopls 陈旧误报（build/vet=0 证伪）。未提交。** | 用户重跑 Go + 全量重同步回填 `bucket15m` → 浏览器全验（三面板独立 loading/趋势桶/token Top/四维筛选链路/tps 口径）→ 提交本批 |
| 2026-07-20 | **Sub2API 后端重写浏览器实测通过 + 全局列表闪屏修复（用户点名）**：用户已重跑 Go+全量重同步；curl+浏览器实测新端点全对。**用户反馈整页闪** → 修共享 `DataTable`：骨架仅首屏，重载保留旧行+延迟 220ms veil；管理 home 概览改轻 spinner。已随 `c8d08f2` 提交。 | 公开门户 |
| 2026-07-20 | **公开门户 home+stats 提交 `ea7c1d4`**：home 拆 `snapshot`+`GetPublicSub2APIOverview`+三请求渐进；stats 去 EP（Metric 图标/明确 loading/stagger/LanguageMenu）；`AppDropdown` 视口钳制；移动顶栏单行；home 顶栏去重复 CTA、hero「用量详情」；完整浏览器验收后提交。 | `public/sub2api/imagine` → lottery → activity |
| 2026-07-20 | **Sub2API 活动 `activity` 按重写文档去 EP 重写布局**：用户点名历史勿按钮弹窗、设置进累计中。实现：`el-tabs`→令牌 `s2a-tabs`；累计中 `#actions` 设置 FormDialog(P3 set-row)；报名中 AppPanel；**页底历史 DataTable 行点详情**；`DataTable` 加 `row-click`/`rowClickable`；规格补入 `06c`。已提交 **`24fa5e7`**（含抽奖历史状态按发放口径）。 | public imagine/lottery |
| 2026-07-20 | **公开绘图 `public/sub2api/imagine` 去 EP 全量重写 + lottery 顶栏对齐**：①imagine：`el-*`/`BaseDialog`/`v-loading` → AppSelect(Key/Model)+AppIconButton 主题 + EmptyState/StatusPill + 令牌 mode-seg composer + FormDialog(参数 chip/详情/删除) + 自建 lightbox(替 el-image)；store 契约不动。②lottery 顶栏 `UIcon` icon-btn → AppIconButton。③规格补 `06e`。浏览器：imagine 1440 暗/浅+参数弹窗+390 移动(含图生图)；lottery 桌面暗真实奖池/历史/规则弹窗；vue-tsc/eslint 绿。**已提交 `9c963f8`。** | `public/share` / `oidc/authorize` 去 EP 或 Phase 4 |
| 2026-07-21 | **公开分享落地 `public/share` + OIDC 授权 `oidc/authorize` 去 EP**：①**share 链路**（`views/public/share/index.vue`+`components/file/FileViewerDialog.vue`+`FileDialog.vue`）移除 `el-icon`×9 → 裸 heroicon svg（`sp-*`/`fv-*`/`fd-*` 图标类 `font-size`→`width/height`；`theme.css .fm-btn svg` 已含尺寸，无需改）；登录按钮硬编码中文→`file.share.login` 三语。**保留 fm-* 文件模块自有主题**（自成体系薄荷主题，非 EP，随文件模块增量待迁）。②**oidc/authorize 全量重写**：EP（`el-avatar`/`el-icon`/`el-alert`/`v-loading`/`el-button`+`@element-plus/icons-vue`+`--el-*` 布局+硬编码中文）→ 居中令牌卡，**对齐 login 门面**（冷灰页底+hairline 卡+角标 `LanguageMenu`+明暗 `AppIconButton`）；shield 头 + `AppAvatar` 账户条 + **scope 友好列表**（`SCOPE_META` 映射 openid/profile/email/offline_access/phone/address→图标+三语文案，未知 scope 原样）+ redirect host 安全提示 + `UButton` 授权/拒绝（移动 column-reverse 主按钮置顶）+ `EmptyState` 无效请求态 + `useFeedback`+`getRequestErrorMessage`（**弃用 EP 的 `showRequestError`**）；保留全部授权流（authorize-info/authorize-code/deny access_denied/PKCE 透传）。`oidc.authorize.*` 顶层命名空间三语。**浏览器全验**（隔离 headless Chrome `:9222` + 手写 `DevToolsActivePort` bridge 到默认 profile 供 MCP autoConnect）：oidc 桌面1440/移动390(emulate 真 390，`resize_page` 触底 500px)×明暗 happy path（临时建真实 client 验后删）+ 无效请求错误态（真实后端「OIDC 客户端无效」）；share 桌面/移动×明暗 真实分享(Lolo) 根/子目录（面包屑 chevron 分隔）/README 预览(FileViewerDialog+Monaco markdown 高亮+关闭/下载图标)；零控制台报错，vue-tsc+eslint 绿。**已提交 `7ae41db`。** | Phase 4（卸 EP/VXE + 清 demo/extended + 暗色/可访问性/i18n 全量核对） |
| 2026-07-21 | **文件模块去 EP 收尾（FilePager + FilePicker，`c6a5c29`）**：FilePager 移除 `el-icon`×3→裸 heroicon svg；FilePicker `BaseDialog`→`FileDialog`（令牌模态）+ `el-tag(closable)`→令牌 chip(fpk-tag-close ×) + `el-button`→`fm-btn`，保留跨来源多选/祖先后代去重。至此 `components/file/*` 全 EP-free。浏览器：file/share 新建→选择文件 FilePicker 开于 FileDialog、来源切换、树勾选、chip 渲染+×移除、确定态随选择启停；vue-tsc+eslint 绿。 | Phase 4 Slice1 删旧代码 |
| 2026-07-21 | **Phase 4 Slice1：删 demo/extended/假分析监控/旧布局壳（`6398d24` 37 文件 8842 删行）+ 后端菜单裁剪（`1dbebed`）**。用户授权「所有旧内容旧代码旧框架全部删除」。删：`views/demo/*`、`views/extended/*`、`views/dashboard/{analysis,monitor}/*`+对应 store、旧布局壳 `layouts/{menu,menuItem,leftMode,topMode,tabsView,header,userDropdown,i18nDropdown,notificationDropdown,breadcrumb}.vue`（已被 `layouts/app/*` 取代、自成死岛）、`dashboard/home/systemOverviewCards.vue`、`plugins/vxeGrid.ts`+`styles/vxeGrid.css`（孤儿）；`stores/dashboard/home.ts` 去 analysis/monitor 快捷入口。`<el-` 47→18 文件、VXE 前端清零。后端 `default_rbac.go` 菜单种子删 menu_1_2/1_3(分析/监控)+整个「扩展组件」dev 目录（`syncDefaultRBAC` 每启动全量同步会 prune 库中旧菜单行，`go build ./...` 通过，**需重跑 Go 生效**）。浏览器验工作台/新外壳正常、零控制台报错。**踩坑**：gopls 对 `new("menu_X")`(局部 new 遮蔽内建) 报 `NotAType` 假阳性，`go build`=0 证伪。 | Slice2 BaseDialog |
| 2026-07-21 | **Phase 4 Slice2：BaseDialog 去 EP 重写为令牌模态（`f0376a1`）**。`el-dialog/el-button/IconButton`→Teleport 令牌模态（遮罩+头/体/脚，AppIconButton 关闭，UButton 页脚，与 FormDialog/FileDialog 同套）；**保留命令式契约**：`attrs.onConfirm`(异步+确认按钮 loading)、emit close/update:modelValue、`#header/#default/#footer`、showClose/Footer/Cancel/Confirm、width+移动端自适应；**去掉**模板遗留全屏/拖拽/缩放/before-close（无消费者用）。`utils/dialog.ts` 去 `ElIcon`，类型图标直接渲染 heroicon+style 定尺寸。**透传去 EP**：`Dialog.confirm/info`(24 文件) + 直接消费者 UpdatePassword/SelectAvatarDialog/IconSelectorDialog/DockerTaskDialogs 一并脱离 EP 模态。浏览器验：IconSelectorDialog(声明式 800px)+file/share 删除 Dialog.confirm(命令式,图标/取消/关闭)桌面正常；vue-tsc+eslint 绿。**既有噪声**：Dialog.confirm 取消时 `utils/dialog.ts` `reject('cancel')` 的未捕获 promise（本次未改该逻辑，属既有契约，后续统一调用方口径时处理）。 | Slice3 ElMessage→useFeedback 单例 |
| 2026-07-21 | **Phase 4 Slice3–5：EP/VXE 卸载收口**：①Slice3 `ElMessage`→`feedback` 单例（`FeedbackBridge`+`utils/feedback` 代理+`request` 拦截器+41 文件迁移，`2fbd154`）。②Slice4 删死组件（IconButton/LoadingButton/BadgeTabsMenu/TablePagination/BaseCard/RunningInstanceCard/TextEllipsis/BaseTag）+ 重写 live：ThemeConfig 令牌抽屉、AppShell 移动抽屉、AdaptiveConfirm 统一 Dialog.confirm、IconSelectorDialog/SelectAvatarDialog 去 EP 控件、DockerTaskDialogs 去 el-button/empty/v-loading、UpdateReleaseContent→EmptyState、App.vue 去 el-config-provider。③Slice5 卸包 `element-plus/@element-plus/icons-vue/vxe-*`；iconRegistry 仅 Heroicons（Element: 兼容映射）；vite 去 ElementPlusResolver；tsconfig 去 element-plus/global；main 去 EP dark css；菜单种子 Element:→Hero。**验证**：vue-tsc/eslint/go build 绿；浏览器工作台+主题抽屉；grep 模板 el-/运行时 ElMessage/node_modules EP·VXE 全 0。 | 暗色/a11y/i18n 全量核对；可选 --el-* 重命名 |
| 2026-07-21 | **文档/代码一致化收口**：08 里程碑/交接/逐页清单去掉陈旧「未提交」「待决」；analysis/monitor/demo/extended 标已删除；EP/VXE 卸载与 SSH 改造写清；06a/06e/03 同步。代码侧无新增业务变更。 | 暗色/a11y/i18n 全量核对；终端移动浅色；实例文件页 |
