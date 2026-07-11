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
| Phase 3 | 全量推广(见下逐页) | ⬜ |
| Phase 4 | 清理待决 + 暗色/可访问性/i18n 全量核对 + **EP/VXE 彻底卸载** | ⬜ |

---

## ⭐ 交接手册（新会话从这里接手）

> **一句话现状**：Phase 0 地基（Nuxt UI 接入+令牌）+ Phase 1 外壳 + Phase 2 **工作台 + 用户管理列表页样板均已完成并浏览器验证**（设计方向经用户确认 2026-07-10）。**P1 列表/CRUD 范式已定稿**（PageHeader+FilterBar+批量条+卡/表切换+DataTable+Pagination+FormDialog），**下一步 = Phase 3 全量推广**（建议从 角色管理 `system/role` 起，复用这套范式）。旧栈 EP/VXE 迁移期共存，未卸载。

### 如何启动 / 可视化验证
- 前端 dev：`cd frontend && pnpm dev`（本次会话 :3007 被占用→实际跑在 **:3008**，新会话按提示端口）。登录 `admin / admin`；后端 API `:22633`。
- 浏览器 MCP `chrome-devtools`（local，autoConnect）：`navigate_page` / `take_screenshot` / `resize_page` / `evaluate_script`。**每页验收：桌面 1440×900 + 移动 390×844，明+暗。** 截图落 `.browser-tmp/shots/`（gitignored）。
- ⚠️ `.app-content` 是**内部滚动容器**（非 window）：截"整页底部"要 `document.querySelector('.app-content').scrollTop = ...`，`fullPage` 只截到视口。

### 新增文件清单（读这些即懂样板架构）
- **地基**：`vite.config.ts`（`@nuxt/ui/vite` `ui()`，auto-import/components 并入其中）、`src/main.ts`（导入 `design-tokens.css` + `@nuxt/ui/vue-plugin`）、`src/App.vue`（`<UApp>` 包裹）、`src/styles/index.css`（`@import '@nuxt/ui'`）、`src/styles/design-tokens.css`（**令牌核心**）、`src/stores/theme/index.ts`（默认色 `#14B8A6`）
- **外壳**：`src/layouts/index.vue`（渲染 `AppShell`+`ThemeConfig`）、`src/layouts/app/*` = `AppShell/AppSidebar/AppNav/AppNavItem/AppTopbar/AppTabs/AppContent/CommandSearch/LanguageMenu/NotificationMenu/UserMenu`
- **基础组件**：`src/components/ui/*` = `AppPanel/SectionHeader/StatusPill/AppAvatar/AppIconButton/MetricStrip/MetricItem/EntityCard/EmptyState/DescriptionList/AppDropdown`（全令牌驱动、自动全局注册）
- **工作台**：`src/views/dashboard/home/*` = `index.vue`（编排+问候+MetricStrip）、`runningInstanceSection.vue`、`shortcutSection.vue`、`systemRealtimeCharts.vue`（仅重写壳、保留 ECharts）。`welcomePanel.vue`/`systemOverviewCards.vue` 已弃用可删。

### 接手要点 / 已踩坑（务必先看）
1. **单实例 unplugin**：Nuxt UI 只允许一个 `unplugin-auto-import`/`unplugin-vue-components`，配置**并入 `ui({autoImport,components})`**（defu 合并、数组拼接），**别再加独立实例**（否则启动报 "Multiple instances"）。
2. **暗色**：`ui({colorMode:false})`，`.dark` 由 `themeStore`（`useDark()`）独占。`design-tokens.css` 必须在 `main.ts` 里**于 EP 暗色 css 之后**导入才能覆盖；暗色块用 `html.dark` 提特异性。teal-500 == 薄荷 `#14B8A6`。
3. **命名避让**：自建组件勿与既有全局组件撞名（已踩 `IconButton`→改 `AppIconButton`）。新建前先 grep `components/**`。
4. **多根 class 不透传**：`AppDropdown` 含 `Teleport` 属多根，外部 `class` 不落到触发器 → 响应式隐藏要包**普通 div**（见 `AppTopbar .app-topbar__desktop`）。
5. **内容区无 keep-alive**：复刻原语义——导航即重挂载，`:key = tabsStore.getRouteRenderKey(fullPath)`；页面轮询靠 `onBeforeUnmount` 停（勿包 `<keep-alive>`）。
6. **store 契约不改**：`menuStore/tabsStore/themeStore/userStore/userProfileStore/useDashboardHomeStore` 全部复用，只换壳。图标经 `menuStore.iconComponents['HOutline:XxxIcon']`（Heroicons，Proxy 按需）。

### P1 列表/CRUD 范式（已定稿 · `system/user` 样板 2026-07-11）
**页结构**（`views/system/user/index.vue`）：`PageHeader`(标题+描述+`新增用户`) → `FilterBar`(用户名/状态+搜索/重置) → 批量条(选中出现，`PERM.USER_DELETE`) → 卡/表切换分段控件(桌面；移动强制卡) → `EntityCard` 卡片流 / `DataTable` 表视图 → `Pagination` → `FormDialog`(create.vue)。删除走 `Dialog.info` 确认，toast 用 `ElMessage`(迁移期)。
**新增 `components/ui/`**：`PageHeader`、`FilterBar`(移动折叠)、`Pagination`(移动简化)、`DataTable`(令牌驱动语义表格：选择/sticky 表头/hover/空/加载/序号)、`ActionMenu`(基于 `AppDropdown` 的行内⋯)、`FormDialog`(令牌驱动模态外壳)。表单控件用 `design-tokens.css` 里的全局 `.app-input/.app-select/.app-textarea/.app-label`。
**务实取舍**（延续 Phase 1 自建令牌驱动路线，见下表）：
- `DataTable` **自建令牌表格**（非 `UTable`/TanStack 封装）——完全掌控观感、免 UTable API 试错、与卡片流一致。桌面渲染，移动由页面切卡片。
- 表单字段用**原生令牌化控件**（非 `UInput`/`USelect`）——免 U* 表单主题不齐；`UButton` 仍用于按钮，`i-lucide-*` 图标(iconify)。
- 校验**内联函数**（非 valibot schema）——移植原 rules；`schemas/*`+valibot 留待后续。
- 删除确认复用 `Dialog.info`、toast 复用 `ElMessage`（迁移期共存，全局横切后续统一换）。

### 下一步：Phase 3 全量推广（建议从 `system/role` 起）
- 复用上面这套 P1 范式推平系统管理其余页（角色/菜单/OIDC/任务/操作日志）与实例/文件等模块。
- 角色管理需 **权限勾选树**（`UTree` 或自建）；菜单/任务/操作日志是**表格主型**，直接用 `DataTable`（操作日志详情走弹窗，勿塞单元格）。
- 待补：`FilterSheet`(移动底部 sheet，当前用内联折叠代替)、`schemas/*`+valibot、`utils/feedback.ts`(toast 收口)。

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
| 工作台 | dashboard/home | P4 | ✅ | ✅ | ✅ | 🟡 待用户确认方向 |
| 分析页(待决) | dashboard/analysis | P5 | ⬜ | ⬜ | ⬜ | ⬜ |
| 监控页 | dashboard/monitor | P5 | ⬜ | ⬜ | ⬜ | ⬜ |
| 个人中心 | profile | P2 | ⬜ | ⬜ | ⬜ | ⬜ |

### 实例/基建(`06b`)
| 页面 | 路由 | 页型 | 桌面 | 移动 | 暗 | 验 |
|---|---|---|---|---|---|---|
| 应用列表 | instance/list | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| 实例控制台 | instance/console/:id | P6 | ⬜ | ⬜ | ⬜ | ⬜ |
| 实例文件 | instance/files/:id | P6 | ⬜ | ⬜ | ⬜ | ⬜ |
| 实例类型 | instance/type | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| Docker 容器 | docker/container | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| Docker 镜像 | docker/image | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| Docker 网络 | docker/network | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| Docker 配置 | docker/config | P3 | ⬜ | ⬜ | ⬜ | ⬜ |
| API Key | node/key | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| SSH 管理 | openssh/management | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| SSH 终端 | openssh/terminal | P6 | ⬜ | ⬜ | ⬜ | ⬜ |

### 工具/文件/Sub2API(`06c`)
| 页面 | 路由 | 页型 | 桌面 | 移动 | 暗 | 验 |
|---|---|---|---|---|---|---|
| 内网穿透 | tools/tunnel | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| 端口转发 | tools/port-forward | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| 文件管理 | file/index | P6 | ⬜ | ⬜ | ⬜ | ⬜ |
| 文件分享 | file/share | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| 文件来源 | file/source | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| Sub2API 首页 | sub2api/home | P4 | ⬜ | ⬜ | ⬜ | ⬜ |
| Sub2API 配置 | sub2api/config | P3 | ⬜ | ⬜ | ⬜ | ⬜ |

### 系统管理(`06d`)
| 页面 | 路由 | 页型 | 桌面 | 移动 | 暗 | 验 |
|---|---|---|---|---|---|---|
| 用户管理 | system/user | P1 | ✅ | ✅ | ✅ | ✅ |
| 角色管理 | system/role | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| 菜单管理 | system/menu | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| OIDC 客户端 | system/oidc | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| 系统设置 | system/settings | P3 | ⬜ | ⬜ | ⬜ | ⬜ |
| 定时任务 | system/task | P1 | ⬜ | ⬜ | ⬜ | ⬜ |
| 操作日志 | system/operation | P1 | ⬜ | ⬜ | ⬜ | ⬜ |

### 认证/独立(`06e`)
| 页面 | 路由 | 页型 | 桌面 | 移动 | 暗 | 验 |
|---|---|---|---|---|---|---|
| 登录/注册/找回 | login | P7 | ⬜ | ⬜ | ⬜ | ⬜ |
| 初始化向导 | initialize | P3 | ⬜ | ⬜ | ⬜ | ⬜ |
| 分享落地 | public/share/:token | P7 | ⬜ | ⬜ | ⬜ | ⬜ |
| OIDC 授权 | oidc/authorize | P7 | ⬜ | ⬜ | ⬜ | ⬜ |
| Sub2API 门户 | public/sub2api/home | P7 | ⬜ | ⬜ | ⬜ | ⬜ |
| Sub2API 统计 | public/sub2api/stats | P7 | ⬜ | ⬜ | ⬜ | ⬜ |
| Sub2API 绘图 | public/sub2api/imagine | P7 | ⬜ | ⬜ | ⬜ | ⬜ |
| 403 | exception/403 | P8 | ⬜ | ⬜ | ⬜ | ⬜ |
| 404 | exception/404 | P8 | ⬜ | ⬜ | ⬜ | ⬜ |
| redirect | redirect | 工具 | ⬜ | ⬜ | ⬜ | ⬜ |

---

## 待决(与用户确认后处理,Phase 4)
- ⬜ `dashboard/analysis` 假数据页:删除 or 改造真实分析?
- ⬜ `dashboard/monitor` 是否含假数据面板(核对 `stores/dashboard/monitor`)
- ⬜ `views/demo/*`(VXE 演示)——随 VXE 下线删除
- ⬜ `views/extended/*`(组件演示)——删除
- ⬜ 顶部导航布局(topMode)是否保留;ThemeConfig 是否精简
- ⬜ 表格默认视图(卡/表)与是否提供切换的最终口径
- ⬜ EP/VXE 彻底卸载 + 全仓 grep(`el-`/`El*`/`ElMessage`/`v-loading`/`vxe`/`@element-plus`)清零

---

## 本次样板的务实取舍(待与用户确认，确认后回填/修订计划)
> 目标是"最快看到真实设计方向"，故对少数非视觉项做了务实取舍。方向确认后可按需回正：
- **图标**：暂**复用现有 `iconComponents`(Heroicons，Proxy 按需加载)**，未做 `iconMap.ts` + `UIcon` 迁移。设计主图标本就是 Heroicons outline，视觉一致；iconify 迁移留待正式 Phase 0 收尾。
- **下拉弹层**：自建**令牌驱动 `AppDropdown`**(onClickOutside+Teleport)承载 语言/通知/用户/命令搜索，未用 `UDropdownMenu`/`UPopover`(避免库 API 试错、完全掌控观感)。可后续替换为 U* 或保留。
- **移动抽屉**：复用现有 **EP `el-drawer`** 承载侧栏(可靠、迁移期共存)；正式可换 `USlideover`。
- **实时图表**：`systemRealtimeCharts` **仅重写 template(AppPanel+令牌化分段/原生 select)+样式，保留全部 ECharts 逻辑**；控件由 EP 换为原生+令牌。
- **ThemeConfig**：暂**原样复用**(EP 抽屉，齿轮打开)；`layout` 的 topMode 选项在新外壳下暂为空操作(任务书允许移除 topMode)。
- **i18n**：`搜索菜单…` 等 2~3 处样板文案用中文字面量(zh-CN)，未入 `messages.ts`；`theme.colors.teal` 选项键未加。正式需补 `messages.ts`。
- **主色**：默认改薄荷青 `#14B8A6`(Nuxt UI `colors.primary:'teal'`=teal-500 同值)；`themeStore` 仅改默认值，未把主色切换同步给 Nuxt UI 全阶(切色时 U* 组件仍为 teal)。
- **命名冲突**：`components/ui/IconButton.vue` 与既有 `components/button/IconButton.vue` 撞名，已改名 `AppIconButton`。后续新自建组件注意避让既有全局组件名。

## 会话日志(每次追加一行)
| 日期 | 会话做了什么 | 下一步 |
|---|---|---|
| (首个会话) | 探索期尝试(令牌/组件/仪表盘,已撤回);定方向(精致侧边栏 + Nuxt UI + EP/VXE 下线);编写完整任务书;还原基线 | 下会话从 Phase 0 接入 spike 开始 |
| 2026-07-10 | **样板落地**：Phase 0(Nuxt UI 4.9 接入+`design-tokens.css`薄荷主色+明暗)跑通;Phase 1 外壳(手写侧栏/顶栏/命令搜索/三下拉/极简标签/内容区/移动抽屉,`layouts/app/*`)+`components/ui/*` 10 个基础组件;Phase 2 工作台重写(安静问候+单色 MetricStrip+实时面板+运行实例 EntityCard/空态+快捷入口+系统信息)。浏览器验证桌面/移动×明/暗、抽屉、下拉、徽标均 OK,零控制台报错,新增文件 lint 绿。见"务实取舍"。 | 等用户确认设计方向;确认后据反馈修订计划,再做 用户管理(列表页样板)与全量推广 |
| 2026-07-11 | **用户管理列表页样板（P1 定稿）**：新增 `components/ui/` 六件套（PageHeader/FilterBar/Pagination/DataTable/ActionMenu/FormDialog）+ 全局令牌化表单控件；重写 `system/user`（卡/表切换、批量删除、Dialog.info 删除确认、内联校验的 create/edit FormDialog），保留 userPage/deleteUser 接口与 PERM 权限。浏览器验证桌面卡/表、创建/编辑弹窗+校验、行内⋯菜单、批量条、分页、移动强制卡+筛选折叠、明/暗全 OK，零控制台报错，lint+tsc 绿。见"P1 范式"与务实取舍。 | Phase 3 全量推广（从 `system/role` 起） |
| 2026-07-11 | **用户反馈-头像收敛**：原三处头像(顶栏右上角 `UserMenu`、工作台问候区、侧栏底部)统一收敛到**仅侧栏底部**。`UserMenu` 加 `variant='topbar'\|'sidebar'`——侧栏整行触发器、向上弹出,展开显示 用户信息/操作/版本号;`AppDropdown` 加 `side='top'`(向上弹)+`block`(填充 flex 父)。顶栏移除 `UserMenu`、工作台移除 `AppAvatar`。下拉底部常驻"版本号: dev";原"版本号"动作行改叫`layout.checkUpdate`(zh-CN/zh-TW/en 已补)避免重复。浏览器验证桌面(展开/折叠)×明/暗 + 移动抽屉均 OK,零控制台报错,lint+tsc 绿。 | 用户管理(列表页样板);顶栏右上角现仅剩 通知铃(用户菜单已下移) |
