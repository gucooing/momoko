# Momoko 前端改版 · 实现规范（本地版）

仓库根目录即本仓库，前端在 `frontend/`（Vue 3 `<script setup>` + TS + Pinia + vue-router + vue-i18n + Nuxt UI v4 + 自研 `src/components/ui/*`）。
主题令牌、外壳（侧边栏/顶栏/手机底部导航）、通用组件和所有页面都已经按新版做过一轮。现在的任务是**逐页对照原型找差异并改到一致**，PC 与手机都要好用。

## 设计稿（目标长相）

目录：本文件旁边的 `project/`
每个页面有 `XxxPC.dc.html` 和 `XxxMobile.dc.html`。它们是静态原型（自定义模板语法），是视觉/布局/信息架构的目标；不要搬它的 HTML 或 CSS 类，用项目里的组件和令牌实现同样的效果。真实页面已有的功能一个也不能丢。

**用户明确授权：如果后端不支持原型里展示的信息或功能（缺字段、缺统计数、缺筛选、缺接口），直接修改后端去实现它，且不需要写兼容分支代码**（直接改结构、改接口、改字段，不保留旧写法）。原型里纯装饰性的示例数据不用实现；但凡是用户能看到的、有意义的信息（例如列表里的计数、状态统计、最近使用时间、流量、大小、关联数量等），后端没有就补上。判断不了是否值得做时，选择做。

## 第一期的参考实现（先读，照这个风格写）

- `frontend/src/styles/design-tokens.css` — 令牌。颜色只用 `--m-*`（`--m-p --m-pOn --m-pInk --m-pSoft --m-a --m-aSoft --m-a2 --m-a2Soft --m-ok --m-okSoft --m-warn --m-warnSoft --m-bad --m-badSoft --m-ink --m-ink2 --m-bg --m-s --m-s2 --m-line`）以及已映射好的 `--el-text-color-primary/secondary/regular`、`--el-bg-color`、`--el-fill-color-light`、`--el-border-color-light/lighter`。圆角 `--app-radius-xs/sm/(默认)/lg/xl/pill`，阴影 `--app-shadow-brand`，字体 `--app-font-display`（Outfit，标题和大数字用）/`--app-font-mono`。工具类 `.m-num .m-mono .tnum .m-sticker .m-sticker--p/a/a2/ok/bad/warn/solid-p/solid-a/solid-a2/solid-ink`。**不要写死颜色**。
- `frontend/src/views/instance/list/index.vue` + `instanceCard.vue` — 卡片列表页：状态 chips + 搜索胶囊 + 视图切换、信息丰富的卡片（贴纸头像、`<dl>` 信息格放在 `--el-fill-color-light` 圆角块里）、卡片底部左标签右小图标按钮、底部悬浮深色批量操作条。
- `frontend/src/views/docker/container/index.vue` — 表格页：`ListToolbar`、行内常用操作是图标按钮 + `ActionMenu`「⋯」、手机端改为卡片列表（`menuStore.isMobile` 或 CSS 断点）。
- `frontend/src/views/docker/components/DockerSubNav.vue` — 同组页面的下划线子导航（基于 `SubNavTabs`）。
- `frontend/src/views/dashboard/home/index.vue` — 主色 hero（主色实底 + 黄色圆 + 紫色圆环装饰，不用渐变）、统计卡。
- `frontend/src/views/login/index.vue` — 无外壳的公开页布局。
- 通用组件（自动注册，直接用）：`PageHeader`、`AppPanel`、`DataTable`、`Pagination`、`StatusPill`、`EmptyState`、`AppIconButton`、`ActionMenu`、`ListToolbar`、`SubNavTabs`、`AppSwitch`、`AppSelect`、`AppCombobox`、`AppDropdown`、`DescriptionList`、`EntityCard`、`MetricStrip/MetricItem`、`SectionHeader`、`FormDialog`、`BaseDialog`、`UserPicker`、`PermissionTree`；Nuxt UI 的 `UButton/UInput/UBadge...` 也可用。先读组件源码确认 props。
- composables：`useAccessibleRoutes`（当前用户菜单里有的路径集合）、`usePaletteColors`（ECharts 要具体颜色时用）、`useButtonPermission`。store：`useMenuStore()`（`isMobile`、`iconComponents['HOutline:XxxIcon']` 渲染 heroicons）、`useNodeStore()`（`canAccessRoute`）、`useThemeStore()`。

## 规则

1. 改动尽量收敛在页面自己的目录；共享组件（`src/components/ui/*`、`src/components/terminal/*`、`src/layouts/*`、`src/styles/*`）可以改，但要确认所有使用方都不受影响。
2. **保留所有业务逻辑**：接口调用、store、权限判断（`v-permission`/`useButtonPermission`）、路由跳转、emits/props 契约、表单校验都不能丢功能。主要重写 `<template>` 和 `<style>`，script 只做为新布局服务的小改动（新增 computed、去掉不再用的变量）。
3. **新文案**：写在 `frontend/src/locales/extra/<模块>.ts`（已有的按模块拆分，优先往对应文件里加），默认导出 `{ 'zh-CN': {...}, 'zh-HK': {...}, 'en-US': {...} }`（嵌套对象，键路径与 `t('a.b.c')` 一致，三种语言都要写，繁体用香港用字），启动时会深合并进主文案表。优先复用已有键（在 messages.ts 里 grep）。英文里不要写撇号 `'`（用 do not 代替 don't）。
4. **视觉规则（用户明确反馈过）**：
   - 不要出现可见的滚动条横条：横向滚动的 chips/tab 行加 `scrollbar-width: none` 和 `::-webkit-scrollbar { display: none }`。
   - 卡片操作用小图标按钮（`AppIconButton` + `ActionMenu`），不要两个占半行的大按钮；启停用颜色区分（停 `--m-bad`、启 `--m-ok`）。
   - 卡片要信息丰富：关键字段放 `<dl>` 两列信息格。
   - 列表页：标题 + 计数说明 + 右上主按钮；状态 chips 带数量（数据里有就带）；**不要「序号」列**；行内常用操作小图标 + 「⋯」；手机端表格改卡片。
   - 同组页面用子导航串起来（用 `SubNavTabs`，见下面路由）。
   - 大圆角、粗体 Outfit 标题/数字、彩色贴纸图标块（`.m-sticker`，可轻微旋转），状态 pill 带圆点（`StatusPill`）。
   - 点击区 ≥ 40px（手机 ≥ 44px），手机端 390 宽不能横向溢出，表单在手机上单列。
5. **同一页面里的标签**：Docker、节点与 API Key、用户与权限、任务与日志、系统设置、智商检测都是「一个菜单 = 一个页面」，页内标签用 `?tab=` 切换，定义在 `frontend/src/config/pageGroups.ts`（`PageGroupHost` + `SubNavTabs`），切换标签不换路由、不重新挂载。应用详情 `/instance/app/:id?tab=console|files|config` 同理。不要再做「看起来是标签、实际跳转到另一个路由」的页面。后端默认菜单在 `internal/data/default_rbac.go`，已按原型导航排好。
6. **检查**：前端 `cd frontend && pnpm exec vue-tsc --build`（内存不够时加 `NODE_OPTIONS=--max-old-space-size=6144`），打包 `pnpm exec vite build`。

## 后端修改（Go Kratos，仓库根目录）

- 结构：`api/proto/v1/*.proto`（接口定义，HTTP 注解）→ 生成 `api/gen/v1/*`（Go）与 `frontend/src/types/v1/*.ts`（前端类型）；业务在 `internal/service`、`internal/biz`、`internal/data`（先读懂现有实现与数据库迁移方式再改）；依赖注入用 wire（`cmd/momoko/wire.go`）。前端接口调用在 `frontend/src/api/*`。
- 改完 proto 运行 `make api`（生成 Go 与前端 `src/types`）；改了 ent schema 运行 `cd internal/data/ent && go generate ./...`；改了 wire provider 运行 `cd cmd/momoko && wire`。不要手改生成文件。
- 后端检查：`go build ./...`、`go vet ./...`、`go test ./...`。
- 所有新增接口都要带上与同模块现有接口一致的鉴权/权限注解（照抄同文件里相邻接口的写法），多节点转发逻辑如果同模块已有也要照做。
- 最终报告里列出后端改动（新增/修改的接口、字段、表结构）。
7. **自查**：本地起前后端，每个改过的页面至少看一次 PC（1440 宽）和手机（390 宽），亮色、暗色各一次；空数据时的空状态也要好看。

## 新需求：持久性能监控（设计稿 `MonitorPC` / `MonitorMobile`，设置打开状态见 `MonitorSettingsPC` / `MonitorSettingsMobile`）

- 入口：PC 侧栏「工作台」下面新增「性能监控」（路由建议 `/monitor`，后端默认菜单同步加上，放在工作台之后）；手机端在「更多 → 应用与监控 → 性能监控」。其它设计稿的侧栏还没加这一项，以 `MonitorPC.dc.html` 为准。
- 页面：时间范围（1 小时 / 6 小时 / 24 小时 / 7 天 / 30 天 / 自定义）→ 五个概览块（CPU、GPU、内存、磁盘读写、网络，带迷你曲线）→ 曲线卡片（图例可点掉曲线、显示当前 / 平均 / 峰值，网络按网卡、磁盘按设备筛选，服务停止期间的空档要画成「无数据」而不是连成直线）→ 磁盘容量与写满预测 → 占用最高的应用（按 CPU / 内存排序）。没有显卡时 GPU 卡片显示空状态。手机端点概览块切换下面的详细曲线。
- 「监控设置」（PC 右侧抽屉 / 手机底部面板）：是否记录历史、采样间隔（5 / 10 / 30 / 60 秒）、**历史保留时长可配置**（7 天 / 30 天 / 90 天 / 180 天 / 1 年 / 自定义天数）、采集项目开关（CPU 与内存始终采集；GPU、磁盘、网络、按应用统计可关）、导出所选时段 CSV、清空历史。
- 后端：常驻采集协程按间隔采样写库；分级降采样（原始精度留 24 小时 → 1 分钟精度留 7 天 → 10 分钟精度留 90 天 → 1 小时精度留到保留期满），定时清理过期数据；查询接口按时间范围自动选精度，返回各序列与汇总（平均 / 峰值）。GPU 采集要能在没有显卡或驱动时安静降级。告警这期不做。
