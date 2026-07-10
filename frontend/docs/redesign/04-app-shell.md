# 04 · 应用外壳(App Shell)· 从布局重写

> 用户选定方向:**精致侧边栏**。Phase 1 的目标是**从骨架重写外壳**——手写导航(不用 el-menu 的模板默认样式),
> 复用后端菜单/路由/权限数据。外壳决定全站观感,必须最先做到位,桌面 + 移动都过关。
>
> 组件基座 = **Nuxt UI**(见 `03b`)。文中 `el-drawer` 等 = 语义示意,实现用 `USlideover`/`UModal`/`UDropdownMenu` 等。
> 现有 `I18nDropdown / NotificationDropdown / UserDropdown` 保留其**数据逻辑**,触发器与弹层**基于 Nuxt UI 重写**。

---

## 1. 文件规划

新建 `src/layouts/app/` 承载新外壳(旧 `leftMode.vue/topMode.vue/menu.vue/menuItem.vue/header.vue/tabsView.vue` 逐步废弃):

```
src/layouts/
  index.vue            // 改为渲染 AppShell(+ 保留 ThemeConfig、移动抽屉挂载点)
  app/
    AppShell.vue       // 组装:Sidebar + 主列(Topbar + [Tabs] + Content)
    AppSidebar.vue     // 品牌 + 导航(手写)+ 底部用户/折叠
    AppNav.vue         // 递归导航容器(读 menuStore.menuList)
    AppNavItem.vue     // 单个导航项/分组(目录=分组标签,菜单=链接)
    AppTopbar.vue      // 标题 + 面包屑 + 搜索 + 操作 + 头像
    AppTabs.vue        // 极简标签(可选,复用 tabsStore 逻辑,重写样式)
    AppContent.vue     // <router-view> + 过渡 + 内边距/限宽(可并入 AppShell)
```

> 数据全部复用:`menuStore`(menuList/isCollapse/isMobile/toggle…)、`tabsStore`、`themeStore`、
> `useUserStore`、`router`、`translateKnownText`、`iconComponents`。**不改这些 store 的对外契约**。

---

## 2. AppSidebar(桌面 ≥ lg)

- **宽度**:展开 `248px`,折叠 `72px`。表面 `--el-bg-color`,右侧 `1px` hairline(`--el-border-color-light`)。整体不投影。
- **结构(纵向)**:
  1. **品牌区**(高 60,底部 hairline):logo(32,圆角 8)+ 文字标 `Momoko`(700,1.15rem)。折叠时只留 logo 居中。
  2. **导航区**(flex:1,可滚动,细滚动条):见 §4。
  3. **底部区**(顶部 hairline):用户迷你块(头像 32 + 名字/角色两行小字)+ 折叠按钮(`«`)。移动端此区改由抽屉承载。
- **激活态**:项背景 `color-mix(primary 12%, transparent)` + 主色文字 + 500 字重;**不要**大填充块、不要重投影。可加 `3px` 左侧主色短条(圆角)。父级目录含激活子项时,分组标签/父项文字转主色。
- **hover**:`--el-fill-color-light` 底 + 文字转 primary 文本色(非主色)。
- **折叠**:仅图标,label 用 tooltip;分组标签隐藏,用一条极短 hairline 代替。

---

## 3. AppTopbar(桌面)

- **高度** 60,底部 hairline,表面 `--el-bg-color`,不投影。
- **左区**:折叠按钮(桌面切展开/折叠)+ **页面标题**(取 `route.meta.title`,`translateKnownText`,1.05rem/600)+ 其下或其右**面包屑**(小字次色,可选)。
- **右区**(从左到右,`gap 8`):
  - **全局搜索/命令**:一个"⌘K"风格的搜索触发(占位"搜索…",点开命令面板/页面搜索;首版可先做**菜单快速跳转**)。
  - **主题**(齿轮→打开 ThemeConfig 抽屉)、**语言**(`I18nDropdown`)、**全屏**、**通知**(`NotificationDropdown`)——统一 `IconButton`,尺寸一致,quiet。
  - **分隔** hairline。
  - **用户头像下拉**(`UserDropdown`):头像 + 名 + 角色,下拉含个人中心/退出等。
- 保留现有 `I18nDropdown / NotificationDropdown / UserDropdown` 的**逻辑**,重写其**触发器与弹层样式**以统一。

---

## 4. AppNav / AppNavItem(手写导航,核心)

读 `menuStore.menuList`(已过滤仅 `MenuStatus_Active`;`MenuType_Button` 跳过)。

- `MenuType_Directory`(目录):
  - 有子项 → 渲染为**分组**:分组标签(overline:大写小字次色)+ 其下子项列表。**默认展开**(顶层)。
  - 也可做成可折叠分组(点标签展开/收起);首版可"顶层目录=分组标签常展开",二级目录=可展开子菜单。
- `MenuType_Menu`(菜单):渲染为**导航链接**(`router.push(path)`);`route.path === path` 判激活。
- 图标:`iconComponents[item.icon]`,18px,继承色。
- 文案:`translateKnownText(item.title)`。
- 深层嵌套:递归 `AppNavItem`;二级用缩进 + 更小字号;超过两级尽量避免(菜单设计层面),但组件需支持递归。
- 权限:菜单本身已按后端过滤;按钮权限不在导航层处理。

> 交互细节:激活项滚动进可视区;移动抽屉内点选后关闭抽屉;折叠态子菜单用 hover 弹出(桌面)或禁用展开(仅顶层图标)。

---

## 5. AppTabs(多开标签,极简 · 可选)

- 受 `themeStore.showTabs` 控制,默认**开**但样式极简;移动端(`< lg`)默认隐藏或横滑小条。
- 复用 `tabsStore` 的全部逻辑(增删、拖拽排序、右键菜单、刷新、滚动定位)。**只重写视觉**:
  - 由现"浏览器 chrome 标签"改为**下划线/胶囊轻量标签**:激活项主色文字 + 2px 主色下划线(或极浅主色底胶囊),非激活次色;关闭 `×` hover 显现。
  - 高度 ~40,底部 hairline 与内容分隔;不投影、不做圆角大耳朵。
- 提供关闭其他/右侧/左侧/全部(复用现逻辑),入口收进标签右侧"⋯"。

---

## 6. AppContent

- `<router-view v-slot>` + `<Transition>`(轻 fade ≤180ms,尊重 `meta.disableTransition`)。
- 内边距:`24px`(桌面)/ `16px`(移动);背景 `--el-bg-color-page`。
- **最大宽度**:常规不限;`≥ 2xl(1600)` 时内容居中限宽(如 1600)两侧留白,避免超宽屏一行拉太长。全屏型页面(终端/文件/控制台)`meta.fullBleed` 时取消内边距与限宽。
- `keepAlive`:沿用 `route.meta.keepAlive` 与 `tabsStore.getRouteRenderKey`。

---

## 7. 移动端外壳(< lg)

- **AppShell** 检测 `isCompact(<1024)`:侧栏不占位,改为 `el-drawer`(左,宽 ~82vw ≤300)承载 `AppSidebar` 内容(含底部用户区)。
- **AppTopbar** 压缩:`[☰] [标题(省略)] [⋯ 溢出] [头像]`。搜索折叠为图标;主题/语言/全屏/通知收进"⋯"或用户抽屉。
- 汉堡 → `menuStore.toggleMobileMenu()`;抽屉 `menuStore.isMobileMenuOpen`;选中项/遮罩/滑动关闭。
- Tabs 隐藏或横滑;内容单列;安全区适配。

---

## 8. 主题/布局配置(ThemeConfig)

- 保留 `ThemeConfig` 抽屉(明/暗/自动、主色、显示 logo/tabs)。
- **布局模式**:用户已选侧边栏方向。可先**移除或隐藏 topMode 选项**(统一侧栏),减少维护面;若保留,则 topMode 也走 AppShell 的顶部导航变体(后置,非 Phase 1 必须)。
- 主色切换保持:`themeStore.togglePrimaryColor` 内联注入 `--el-color-primary`。

---

## 9. Phase 1 验收
- [ ] 桌面:侧栏(展开/折叠)、顶栏、标签、内容留白/限宽 均达 `01` 观感;一个强调色、hairline、无重阴影。
- [ ] 导航:分组/激活/多级/权限过滤/图标/i18n 正确;路由跳转与现有一致。
- [ ] 移动:抽屉侧栏、压缩顶栏、隐藏/横滑标签、单列内容,390/768 无破版。
- [ ] 明/暗两态正确;`keepAlive`/多标签逻辑不回归。
- [ ] 截图(1440 + 390)留档于会话,并在 `08` 勾选。
