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

## 新需求：自定义外观主题、主题包、在线主题编辑器（设计稿 `Theme*.dc.html`）

- 设计稿：`ThemePC` / `ThemeMobile`（外观页）、`ThemePreviewPC`（主题包 / 主题预览模式）、`ThemeEditorPC` / `ThemeEditorMobile`（在线编辑器）、`ThemeEditorLoginPC`（编辑器里「仅本页」的样子）、`ThemeImportPC` / `ThemeImportMobile` / `ThemeExportPC`（导入、导出弹窗；手机上是底部面板）。
- **主题 = 配色（浅色 + 深色两套 token）+ 背景 + 细节（圆角、字体、字号、阴影、按钮形状、侧栏样式）+ 页面单独设置**。内置 6 套不可改，「我的主题」可新建、编辑、复制、删除、导出；管理员可「设为全站默认」（没设置过外观的用户和登录页用它）。
- **背景**五种：跟随主题、纯色、图案（点阵 / 网格 / 斜纹 / 波点 / 波纹，颜色可跟随主题）、渐变（预设 + 色标 + 角度 + 线性 / 径向，可「缓慢流动」，开了「减少动效」就停）、图片（上传 JPG / PNG / WebP ≤ 10 MB，服务端转 WebP；模糊、遮罩、卡片不透明度、填充方式；深色模式自动加深遮罩）。背景铺在 `position: fixed` 的一层上；图片和渐变时卡片、侧栏、输入框改成 `color-mix(var(--s) 不透明度, transparent)` + `backdrop-filter` 毛玻璃。
- **在线编辑器**（独立全屏页，建议路由 `/profile/theme/editor/:id`）：
  - 左：「改动作用于 全局 / 仅当前页」+ 配色 / 背景 / 字体形状 / 组件 四个标签；配色列出所有 token（色块、名称、用途、色值，改过的有圆点，页面覆盖的有「本页」标签）；终端页多出终端专用 token（背景、文字、光标、ANSI 16 色）。
  - 中：按页面预览（工作台、应用列表、文件管理、应用控制台、登录页…，有单独设置的页面带圆点），可切电脑 / 手机、浅色 / 深色；「点选元素」开着时点预览里的元素就选中它用的 token，并用虚线框标出这个 token 用在哪。预览直接复用真实页面组件，套一层 CSS 变量即可，不要另画一份。
  - 右：取色器（饱和度方块、色相条、HEX、推荐色）、当前页的单独设置列表（可逐条改回跟随全局）、对比度检查（低于 4.5:1 提示，可一键修正）。
  - 撤销 / 重做、草稿自动保存；导出菜单：导出主题包、只导出当前页设置、复制主题代码（CSS 变量）。
- **主题包** `.momoko-theme`：zip，内含 `theme.json`（`format: 1`、name、author、version、`tokens.light` / `tokens.dark`、`background`、`shape`、`pages: { [路由名]: { tokens?, background?, cardOpacity? } }`）+ `assets/` 背景图（和字体）。导入时校验格式版本、只接受颜色、数值、图片和字体（不接受 CSS 文本或脚本）、图片转 WebP、检查对比度；同名主题可「另存为副本 / 覆盖旧版本」；导入后可「只导入」或「导入并使用」。
- **后端**：主题按用户存（新表，比如 `user_theme`：id、user_id、name、data JSON、assets、created_at / updated_at），当前选中的主题和外观偏好存在用户设置里；背景图和字体走现有文件存储，带大小限制；全站默认主题存系统配置。接口：主题增删改查、上传背景图、导入（解析 zip）、导出（打包 zip）、设为全站默认（需要管理员权限）。照现有接口加鉴权注解。
- **大图区域也能换图**：工作台欢迎横幅（以及登录页插画区、手机首页顶部卡片）的背景可选「主色 / 渐变 / 自定义图片」；图片可上传、调焦点、压暗，文字颜色可选自动 / 白字 / 黑字（自动按图片亮度判断）。在编辑器里点横幅即可编辑（`ThemeEditorPC` 默认就是这个状态），也可以按页面单独设置。主题包 `theme.json` 增加 `banner: { type, image?, gradient?, dim, ink, focus }`，图片放进 `assets/`。
- **主题包预览**：
  - 导入弹窗里的预览可以切换「工作台 / 登录页 / 终端 / 手机」和浅色 / 深色，直接看主题包在各页（包括它的页面单独设置）上的效果。
  - 「在整个界面里预览」（导入弹窗里，以及「我的主题」卡片上的眼睛按钮）：临时套用这个主题，顶部出现悬浮条「正在预览 · 浅色 / 深色 · 退出预览 · 导入并使用 / 使用这个主题」；期间可以随便切换页面，只存在当前会话（前端内存 / sessionStorage），不写入账号；退出或刷新后恢复原主题。

## 游戏服务端管理与 GM（新功能，另见 game/IMPL_GAME.md）

这是在改版之外新增的功能：游戏服务器（托管 / 仅 GM）、GM 运营（第一个适配学マス）、独立端口的「用户通行证」玩家端。原型在 `game/`，实现规范在 `game/IMPL_GAME.md`；本文件的前端通用规则同样适用。
