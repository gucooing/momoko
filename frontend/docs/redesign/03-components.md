# 03 · 通用组件目录(Component Catalog)

> 页面只允许用**这些**通用组件 + Nuxt UI 原子组件(`U*`)拼装,不在页面里重复造轮子。
> 全部放 `src/components/ui/`(自动全局注册,按文件名 PascalCase 使用)。
> 状态标注:`[建]` 需在 Phase 0 新建/重做;`[U]` 直接用 Nuxt UI 原子(`UButton/UInput/USelect/UModal/UTable/UBadge/UDropdownMenu/UTooltip/UPopover/UTabs/USwitch/UCheckbox/URadioGroup/UAvatar/UIcon`…)。
> 自建组件多为对 Nuxt UI/Reka 的**语义封装**以统一观感与移动端行为。组件名映射见 `03b`。
> 所有组件样式只用 `01` 的令牌;禁止在组件里写死颜色/阴影/圆角魔法值。

---

## A. 原子(Primitives)

### `StatusPill` `[建]`
状态指示:圆点 + 文本,语义单色淡底。
- props: `type: 'success'|'warning'|'danger'|'info'|'primary'|'default'`、`text?`、`dot=true`、`plain=false`
- slot: 默认(覆盖文本)
- 用于:列表/详情里的状态、启用/禁用、在线/离线。**替代**满色 `el-tag` 做状态。

### `Chip` `[建]`
轻量标签/筛选 chip(可关闭)。
- props: `label`、`closable=false`、`type?`(语义)
- emits: `close`
- 用于:已选筛选、标签集合。

### `AppAvatar` `[建]`(薄封装 `el-avatar`)
- props: `src?`、`name?`(无图时取首字生成)、`size=36`、`status?: 'online'|'offline'`(右下角点)
- 统一头像圆角、回退、状态点。

### `IconBox` `[建·克制使用]`
统一图标容器。**默认 `soft`、单色**;`gradient/solid` 仅限品牌化极少数场景(登录、单个空状态图标)。**禁止**在指标/列表里成排彩色使用。
- props: `icon`、`color?`(默认主色)、`variant:'soft'|'solid'|'gradient'|'outline'='soft'`、`size:'xs'|'sm'|'md'|'lg'|'xl'='md'`

### `Skeleton` / `SkeletonText` `[建]`
加载占位。基于 Nuxt UI `USkeleton`/Tailwind;列表/卡片加载用统一骨架(与卡片/行同尺寸)。封装 `TableSkeleton`(行占位)、`CardSkeleton`(卡片占位)。

---

## B. 页面结构(Page Structure)

### `PageHeader` `[建]`
每个业务页顶部第一块(全屏型页面除外)。
- props: `title`、`description?`、`icon?`(克制,默认不放彩色盒)、`breadcrumb?: {label,to?}[]`
- slots: `actions`(右侧主/次操作)、`badge`(标题旁徽标)、`description`
- 响应式:`< md` 标题与操作纵向堆叠;操作可收进"⋯"。

### `SectionHeader` `[建]`
区块小标题:overline(大写小字)/标题 + 右侧操作。
- props: `title`、`overline?`、`count?`
- slot: `actions`

### `AppPanel` `[建]`
带标题的内容卡(白面 + hairline)。
- props: `title?`、`subtitle?`、`icon?`、`flush?`(主体无内边距,用于内嵌表格)
- slots: 默认(主体)、`actions`(标题右)、`header`(自定义头)

### `Toolbar` `[建]`
操作/批量操作行(左操作组 + 右辅助)。用于列表页表格上方。
- slots: `left`、`right`;响应式换行。

---

## C. 数据呈现(Data Display)

### `MetricStrip` + `Metric` `[建]`
指标带:一块面板,竖向 hairline 分隔的等宽列。**这是仪表盘/概览的标准指标呈现,取代彩色 KPI 卡阵列。**
- `MetricStrip`:容器,`columns?`(默认自适应 2/4)
- `Metric` props:`label`、`value`、`unit?`、`caption?`、`percent?`(0–100→细单色进度线)、`trend?:{text,dir}`(可选,单色)
- slot(Metric):`value`(自定义,如网络上下行双行)、`caption`
- 响应式:4→2→1 列;分隔线方向自适应。

### `DataTable` `[建]`(核心)
精致表格封装,统一表头/行/hover/sticky/空/加载,并**内建"移动端转卡片"**。
- props:
  - `columns: Column[]`(`{ key,title,width?,minWidth?,align?,fixed?,sortable?,hideOnMobile?,primary?,slot? }`)
  - `data`、`loading?`、`rowKey`、`selection?`(多选)、`total?`、`page?`、`pageSize?`
  - `cardTitleKey?`、`cardMetaKeys?`:移动端卡片如何取主标题/元信息(不传则用 `primary` 列 + 前 N 列)
- slots:`cell-<key>`(单元格)、`card`(完全自定义移动卡)、`empty`、`toolbar`、`expand?`
- emits:`update:page`、`update:pageSize`、`selection-change`、`sort-change`、`row-click`
- 行为:`≥ lg` 渲染令牌 `DataTable`;`< lg` 自动渲染 `EntityCard` 列表(用 `cell-<key>`/`card` 复用渲染)。
- 用于:强列型/可比较型数据(操作日志、任务、菜单)。实体型数据优先 `EntityCard` 流(见下)。

### `EntityCard` `[建]`
人性化记录卡。列表页"卡片流"的单元。
- props:`title?`、`subtitle?`、`icon?`、`color?`、`selectable?`、`selected?`、`clickable?`
- slots:`media`(头像/图标/缩略)、`status`(状态胶囊)、默认(元信息主体)、`actions`(底部操作)、`corner`(右上)
- emits:`update:selected`
- 元信息推荐用 `DescriptionList` 或 `key·value` 行。

### `DescriptionList`(`DL` / `DLItem`)`[建]`
键值详情列表(详情页、卡片内元信息、系统信息)。
- `DL` props:`columns?`(1/2)、`inline?`
- `DLItem` props:`label`;slot 值。样式:label 次色定宽,value 常规色,可 hairline 行分隔。

### `EmptyState` `[建]`
空态:图标 + 标题 + 描述 + 操作。
- props:`title`、`description?`、`icon?`、`size:'sm'|'md'`
- slot:默认(操作按钮)、`description`

### `ErrorState` `[建]`
错误态:说明 + 重试。
- props:`title?`、`description?`、`retryText?`;emits:`retry`

### `Pagination` `[建]`(薄封装 `el-pagination` / 现 `TablePagination`)
桌面标准分页;`< lg` 退化为"上/下页 + x/y"。沿用现 `components/pagination/TablePagination.vue` 的移动降级思路。

---

## D. 表单与筛选(Forms & Filters)

### `FilterBar` `[建]`
桌面横向筛选条(白面 hairline)。`< lg` 自动折叠为"搜索 + [筛选]按钮 + 已选 chips",点开为 `FilterSheet`。
- slots:默认(字段:`el-input`/`el-select`/日期等)、`actions`(搜索/重置)
- props:`collapsibleOnMobile=true`

### `FilterSheet` `[建]`(移动端筛选底部 sheet)
- props:`modelValue`(开关);slot:字段;footer:重置/应用。由 `FilterBar` 内部驱动,页面一般不直接用。

### `FormDialog` `[建]`(标准表单弹窗)
统一弹窗:标题 + 主体(表单 slot) + 底部操作;`< lg` 自动全屏、`label-position=top`、操作吸底。
- props:`modelValue`、`title`、`width?`、`loading?`、`confirmText?`、`cancelText?`
- slots:默认(表单)、`footer`(自定义);emits:`confirm`、`cancel`、`update:modelValue`
- 现有 create.vue 弹窗统一改用它;沿用现 `BaseDialog`/`AdaptiveConfirm` 的自适应经验。

### `AdaptiveConfirm` `[已存在]`
就近确认(删除等)。保留,纳入 `02` 弹层规范。

---

## E. 操作(Actions)

### `ActionMenu` `[建]`
"⋯"更多操作菜单(`el-dropdown` 封装),统一图标/项样式;移动端触发底部菜单。
- props:`items: {label,icon?,danger?,disabled?,onClick}[]` 或用 slot。

### `FabButton` `[建]`
移动端右下悬浮主操作(新增等)。桌面隐藏。
- props:`icon`、`label?`;emits:`click`;避让底部安全区。

### `IconButton` `[已存在]`
图标按钮(顶栏/工具用)。保留,统一尺寸与 hover。

---

## F. 反馈(Feedback)
- 轻提示:`feedback` / `useFeedback()`(Nuxt UI toast,`z-index` 高于弹窗);确认:`Dialog.confirm` / `AdaptiveConfirm`;加载:骨架 / 按钮 loading / 轻 veil。
- 统一封装 `utils/feedback`(可选):`toast.success/error`、`confirmDanger()`。

---

## 组件-页型 映射速查

| 页型 | 用到的组件 |
|---|---|
| 列表/CRUD | PageHeader · FilterBar · Toolbar · (EntityCard 流 或 DataTable) · Pagination · FormDialog · ActionMenu · EmptyState |
| 详情 | PageHeader · AppPanel · DescriptionList · StatusPill · Toolbar |
| 仪表盘/概览 | PageHeader · MetricStrip/Metric · AppPanel · SectionHeader · EntityCard(运行项) |
| 表单/向导 | FormDialog 或整页表单 · AppPanel · EP 表单族 |
| 终端/文件 | 全屏自定义 + PageHeader(精简)+ Toolbar |
| 认证/独立 | 品牌区 + 表单卡 + EP 表单族 |
| 异常 | EmptyState/ErrorState 变体 |
