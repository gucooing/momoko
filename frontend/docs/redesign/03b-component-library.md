# 03b · 组件库:Nuxt UI v4 接入 + Element Plus / VXE 全面下线

> **架构决定(已定稿)**:框架保留 **Vue 3 + Vite + TS + Tailwind v4**;组件库换为 **Nuxt UI v4(latest,≥4.9.0)**;
> **Element Plus 与 VXE Table 全部下线**。Nuxt UI 底层是 **Reka UI + Tailwind v4**,官方支持纯 Vue + Vite。
> 本文件是"换库"的**技术总纲 + 逐项映射**。任何页面迁移前必须读它。

---

## 1. 为什么可行 / 边界

- `@nuxt/ui@4.9.0` 导出了 `./vite`(Vite 插件)与 `./vue-plugin`(Vue 插件)+ `<UApp>` 根组件 → **非 Nuxt 的 Vue+Vite 可用**。
- 保留(与组件库无关,**不动**):**ECharts**(vue-echarts)、**Monaco**(编辑器)、**xterm**(终端)、`@bufbuild/protobuf` 类型、axios 请求层、pinia、vue-router、vue-i18n。
- 下线:`element-plus`、`@element-plus/icons-vue`、`vxe-table`、`vxe-pc-ui`、`@/plugins/vxeGrid`、`src/config/elementConfig.ts`(EP 专属)。
- 图标:`@heroicons/vue` 与自建 `iconRegistry` → 迁到 **Nuxt UI `UIcon`(Iconify)**;安装图标集
  `@iconify-json/lucide`(Nuxt UI 默认,主用)+ `@iconify-json/heroicons`(承接后端存量图标名)。

---

## 2. Phase 0 接入 spike(第一件事,必须先跑通再迁移)

> 目标:装包 + 配好 Vite/CSS/插件 + `<UApp>` + 渲染一个 `UButton`/`UModal`/`UInput` + 明暗 + 主色可切换。
> 跑通后再动任何页面。以下步骤以 **Nuxt UI 官方 "Vue (Vite)" 指南**为准,若 API 细节有出入以官方为准并回填本文件。

1. **装包**
   ```
   pnpm add @nuxt/ui
   pnpm add -D @iconify-json/lucide @iconify-json/heroicons
   # 之后按需:@internationalized/date(日期)已随 Nuxt UI;富文本 tiptap 仅用到再装
   ```
2. **Vite 插件**(`vite.config.ts`):
   ```ts
   import ui from '@nuxt/ui/vite'
   plugins: [ vue(), ui({ /* colors/theme options */ }), /* 其余保留 */ ]
   ```
   → Nuxt UI 自带组件/composable 自动导入。**移除** `ElementPlusResolver`;`unplugin-vue-components` 仅保留扫描 `src/components`(自建组件)。
3. **CSS 入口**(`src/styles/index.css` 顶部):
   ```css
   @import 'tailwindcss';
   @import '@nuxt/ui';
   /* 之后:@import './design-tokens.css'; 覆盖 --ui-* 令牌(见 §4) */
   ```
   删除 `element-plus/theme-chalk/dark/css-vars.css`、`@import 'element-plus'` 等。
4. **Vue 插件 + 根 `<UApp>`**:
   ```ts
   // main.ts
   import uiPlugin from '@nuxt/ui/vue-plugin'
   app.use(uiPlugin)
   ```
   ```vue
   <!-- App.vue -->
   <UApp :locale="uiLocale"><RouterView /></UApp>
   ```
   `<UApp>` 提供 toast/overlay/tooltip 容器与 locale,**必须包在最外层**。
5. **明暗**:Nuxt UI 用 `.dark` 类(与现 `useDark()` 一致)。保留 `useThemeStore` 的明/暗/自动逻辑,继续 toggle `<html>.dark`。
6. **验收**:一个演示页放 `UButton/UInput/USelect/UModal/UTable/UBadge`,切明暗 + 切主色正常 → spike 通过。

---

## 3. 图标迁移

- 组件内直接:`<UIcon name="i-lucide-server" />`。新代码一律用 `i-lucide-*`(主)或 `i-heroicons-*`。
- **后端存量菜单图标名**(如 `'HOutline:ServerStackIcon'`、`'Element:Monitor'`):建 `src/config/iconMap.ts`
  将旧名 → iconify 名(如 `HOutline:ServerStackIcon → i-heroicons-server-stack`,`Element:Monitor → i-lucide-monitor`)。
  外壳导航与 `IconSelectorDialog` 通过该 map 解析;`iconRegistry.ts` 删除。
- 图标选择器(`system/menu` 用)改为在 iconify 集合里选,存 iconify 名;兼容旧值经 `iconMap` 兜底。

---

## 4. 主题令牌与主色(可切换)

- Nuxt UI 语义色:`primary/secondary/success/info/warning/error/neutral`,由 CSS 变量 `--ui-*` 驱动,
  组件 class 走 Tailwind v4 `@theme`。
- **中性 / 语义 / 圆角 / 阴影** 取 `01-design-language.md` 的数值,写进 `design-tokens.css`,以
  `@theme` + `:root`/`.dark` 覆盖 Nuxt UI 的 `--ui-*`(如 `--ui-bg`、`--ui-text`、`--ui-border`、`--ui-radius`)。
  (旧 `--el-*` 覆盖作废;数值不变,改挂到 `--ui-*` / Tailwind token。)
- **主色可切换**:`themeStore.togglePrimaryColor(hex)` 由"注入 `--el-color-primary`"改为**注入 `--ui-primary` 全阶**
  (`--ui-color-primary-50..950`),用 `color-mix` 从 hex 生成明暗两套阶(沿用现 `setPrimaryColor` 的 mix 思路)。
  默认 **薄荷青绿 `#14B8A6`**。Phase 0 spike 里确认确切变量名并回填。

---

## 5. Element Plus → Nuxt UI 组件映射

| Element Plus | Nuxt UI | 备注 |
|---|---|---|
| `el-button` | `UButton` | `type=primary`→`color="primary"`;`link`→`variant="link"`;`plain`→`variant="soft/outline"` |
| `el-input` / `el-textarea` | `UInput` / `UTextarea` | |
| `el-select` / `el-option` | `USelect` / `USelectMenu` | 远程/可搜索用 `USelectMenu` |
| `el-form` / `el-form-item` | `UForm` / `UFormField` | 校验改 schema(见 §6) |
| `el-dialog` | `UModal` | 移动端全屏见 `02` |
| `el-drawer` | `USlideover` | 侧栏抽屉/筛选 sheet |
| `el-table` / `VxeGrid` | `UTable`(经 `DataTable` 封装) | TanStack 列定义;移动转卡片 |
| `el-pagination` | `UPagination` | 移动降级见 `03` |
| `el-tag` | `UBadge` / 自建 `StatusPill` | 状态一律 `StatusPill` |
| `el-tabs` | `UTabs` | 外壳标签见 `04` |
| `el-dropdown` | `UDropdownMenu` | 行内"⋯"= 自建 `ActionMenu` 封装它 |
| `el-tooltip` | `UTooltip` | |
| `el-popover` / `el-popconfirm` | `UPopover` / 自建 `AdaptiveConfirm`(基于 UPopover/UModal) | |
| `el-switch` | `USwitch` | |
| `el-checkbox` / `el-radio` | `UCheckbox` / `URadioGroup` | |
| `el-avatar` | `UAvatar`(经 `AppAvatar`) | |
| `el-card` | 自建 `AppPanel` / `UCard` | 卡片一律走自建以统一观感 |
| `el-date-picker` / `el-time-picker` | Nuxt UI 日期(基于 `@internationalized/date` 的 Calendar/`UPopover`) | 复杂区间用 `UCalendar` 组合 |
| `el-upload` | `UFileUpload`(或自建基于 input+dropzone) | 文件模块见 `06c` |
| `el-tree` / `el-tree-select` | `UTree`(若无则自建基于 Reka) | 菜单树、文件树 |
| `el-color-picker` | 自建(或第三方)小组件 | 仅 ThemeConfig 用 |
| `el-scrollbar` | 原生滚动 + `01` 滚动条样式 | 去掉 |
| `el-config-provider` | `<UApp :locale>` | |
| `el-icon` + `@element-plus/icons-vue` | `UIcon` + Iconify | 见 §3 |

## 6. 横切设施替换(逐项)

- **消息提示** `ElMessage.success/error` → `const toast = useToast(); toast.add({ title, color:'success'|'error' })`。
  统一封装 `src/utils/feedback.ts`:`toast.success()/error()/info()`。
- **确认框** `ElMessageBox.confirm` → 自建 `useConfirm()`(基于 `UModal`/`useOverlay`)或 `AdaptiveConfirm`;
  危险操作 `confirmDanger({ title, description })`。
- **加载** `v-loading` 指令 → 用 `Skeleton`/`CardSkeleton`/`TableSkeleton` 或区块内 `UProgress`/遮罩;
  按钮 `:loading` 用 `UButton` 原生 `loading`。删除 `v-loading` 用法。
- **通知** `ElNotification` → `useToast()`(带 description/action)。
- **表单校验**:`el-form` 的 `rules` → `UForm :schema`(标准 schema,建议 **valibot**;或 zod)。
  校验错误由 `UFormField` 展示。写 `src/schemas/*` 存放各表单 schema。
- **弹窗命令式**:`utils/dialog.ts`(现基于 EP 命令式渲染)→ 改为 `useOverlay()` 打开 `UModal`。
- **locale**:`<UApp :locale>` 用 `@nuxt/ui/locale` 的 `zh_cn`/`en`;与 `vue-i18n` 并存(我们自己的文案仍走 vue-i18n)。

## 7. 表格(VXE → UTable via DataTable)

- 列定义迁到 TanStack 风格(`accessorKey/header/cell`)。`DataTable`(见 `03`)封装:表头/行/hover/sticky/
  选择/排序/空/加载 + **移动端转卡片**。
- 现 `src/plugins/vxeGrid.ts`、`styles/vxeGrid.css`、`components/pagination/TablePagination.vue`(EP 分页)重写/删除。
- 强列型页面(操作日志、任务、菜单树)用 `DataTable`;实体型页面优先 `EntityCard` 卡片流。

## 8. 下线清单(package.json / 代码)

- 卸载依赖:**已完成** — `element-plus`、`@element-plus/icons-vue`、`vxe-table`、`vxe-pc-ui` 已从 package 移除;图标仍用 `@heroicons/vue`(非 iconify 全量迁移)。
- 删除:`plugins/vxeGrid.ts`、`styles/vxeGrid.css`、`config/elementConfig.ts`、`config/iconRegistry.ts`、`main.ts` 的 EP 导入、`vite.config` 的 `ElementPlusResolver`、`optimizeDeps` 里 EP 项。
- 全仓 grep 清零:`el-`、`El` 命名式组件、`ElMessage`、`ElMessageBox`、`v-loading`、`VxeGrid`、`@element-plus`、`vxe`。
- **迁移期策略**:按模块推进;某模块迁完即删该模块的 EP 引用。**允许迁移期短暂共存**(EP 仍在 deps),
  但**最后一个模块迁完必须彻底卸载 EP/VXE 并 grep 清零**(`08-progress.md` 有"EP 清零"验收项)。

## 9. 风险与注意

- **Tailwind v4 冲突**:项目已用 `@tailwindcss/vite`;Nuxt UI 也基于 Tailwind v4。统一一个 Tailwind 入口,`@import '@nuxt/ui'` 之后再 `@import 'design-tokens.css'` 覆盖。避免重复 preflight。
- **自动导入**:去掉 EP resolver;保留 vue/router/pinia 自动导入与 `src/components` 扫描。Nuxt UI 组件由其 Vite 插件自动导入(无需手动)。
- **SSR 无关**:纯 SPA,忽略 Nuxt UI 的 SSR 分支。
- **i18n 组件文案**:Nuxt UI 内置组件(分页、日历等)文案走 `<UApp :locale>`;别和 vue-i18n 混。
- **体积**:迁移完成删 EP/VXE 后体积应下降;迁移期共存会临时增大,可接受。
- **首屏令牌闪烁**:`design-tokens.css` 在 `@nuxt/ui` 之后导入;主色在 `main.ts` 早期注入,避免闪烁。
