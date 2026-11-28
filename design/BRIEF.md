# Momoko 前端改版 — 画板编写规范（所有协作者必读）

Momoko 是一个服务器 / 应用实例 / Docker / 网络工具的管理面板（Go 后端 + Vue 前端）。我们在一个 Design 画布上用 `.dc.html` 文件画新版界面。已经完成的画板在 `project/` 目录下，**它们就是你的模板，先读再写**：

- `project/AppsPC.dc.html` — PC 外壳（侧边栏 + 顶栏）、卡片列表、状态筛选 chips、批量操作条
- `project/ContainersPC.dc.html` — PC 表格页、页内下划线子分类 tab（utab）、分页
- `project/ThemePC.dc.html` — PC 设置类页面（分组卡片、开关 .tg、选项卡 .pc、seg）
- `project/AppsMobile.dc.html` — 手机长页面、信息丰富的卡片、卡片底部小图标操作、底部导航、悬浮新建按钮
- `project/ContainersMobile.dc.html` — 手机 390×844 视口、内部滚动、底部操作面板（sheet）
- `project/LoginPC.dc.html` / `project/LoginMobile.dc.html` — 公开页 / 无侧边栏页面

根目录（绝对路径）：`/tmp/claude-0/-home-claude/f072b29f-59cc-5df2-a68c-ac6824ba01c8/scratchpad/momoko/project/`

## 你只做这些

- 只用 Write 工具直接写你被分配的文件（每个文件完整 `.dc.html` 源码）。不要改其它文件，不要碰 `canvas.json`，不要发布，不要用浏览器 / Playwright 渲染检查，不要跑脚本生成文件。
- 写完后回报：每个文件的 文件名、建议画板尺寸 w×h、标题（中文，如「文件管理 · PC」）、是否可交互、PC 是否页面型（expand fill）。

## 文件格式（违反会静默出错）

- 骨架照抄模板：`<!doctype html>`、`<html lang="zh-CN">`、`<head>` 里**原样**保留 `<script src="./support.js"></script>`、`<body><x-dc><helmet>…</helmet> 页面 </x-dc><script type="text/x-dc" data-dc-script data-props='…'>…</script>`。
- `<helmet>`：Google Fonts Outfit 的两行 link + 一个 `<style>`。`<style>` 里**原样复制** ThemePC.dc.html 中的 12 个主题类（`.th-sakura-light` … `.th-classic-dark`），再加上你用到的组件类（从模板里复制 `.root .card .btn .btn-p .btn-s .btn-o .ib .ic .pill .ok .off .warn .bad .chip .nav .lbl .inp .kbd .utab .seg .av .mono .num .tg .cb` 等，只拷需要的）。所有颜色只用 `var(--p) var(--pOn) var(--pInk) var(--pSoft) var(--a) var(--aOn) var(--aSoft) var(--a2) var(--a2Soft) var(--bg) var(--s) var(--s2) var(--line) var(--ink) var(--ink2) var(--ok) var(--okSoft) var(--warn) var(--warnSoft) var(--bad) var(--badSoft)`，不要写死颜色（唯一例外：a2 实底上的白色图标/大字可用 #FFFFFF；终端黑底可用 #0F0B14 / #E8E3EE 这类固定终端色）。
- 再加一行隐藏滚动条：`.hs{scrollbar-width:none;-ms-overflow-style:none}.hs::-webkit-scrollbar{display:none}`，所有横向滚动行和内部纵向滚动区都加 `class="hs"`。**用户明确反感可见的滚动条横条。**
- 根节点：`<div class="root th-{{palette}}-{{mode}}" …>`。
- `data-props`（单引号包裹）：`{"palette":{"editor":"enum","options":["sakura","mint","volt","grape","sunset","classic"],"default":"sakura"},"dark":{"editor":"boolean","default":false},"$preview":{"width":W,"height":H}}`
- 脚本：经典 JS，`const IC = {...}` 图标路径表 + 其它常量，然后 `class Component extends DCLogic { renderVals() { … } }`。renderVals 开头照抄：
  ```js
  const st = this.state || {};
  const palette = this.props.palette ?? 'sakura';
  const dark = st.dark != null ? st.dark : (this.props.dark ?? false);
  ```
  返回 `palette, mode: dark ? 'dark' : 'light', i: IC, modeIcon: dark ? IC.sun : IC.moon, toggleDark: () => this.setState({ dark: !dark })` 以及页面数据。
- `{{hole}}` 只能是点路径查找（`{{a.name}}`），**不能写表达式**；需要计算的值在 renderVals 里算好再暴露。属性里 `x="{{path}}"` 传原值，`x="a {{p}} b"` 是字符串插值。
- 循环 `<sc-for list="{{items}}" as="it" hint-placeholder-count="3">`，条件 `<sc-if value="{{cond}}" hint-placeholder-val="{{true}}">`，hint 必填。
- 事件：`onClick="{{fn}}"`（fn 从 renderVals 返回）；循环里每项自带 `pick: () => this.setState(...)`。状态驱动的样式用 class 名或 style 洞（如 `style="color: {{r.fg}}"`）。
- 所有元素闭合、所有属性加引号、`<input>` 等空元素按模板写；不要 `innerHTML`、不要 `<iframe>`、不要 emoji、不要全局键盘监听、不要网络请求（字体 link 除外）。
- 图标：内联描边 SVG `<svg class="ic" viewBox="0 0 24 24" aria-hidden="true"><path d="{{i.xxx}}"></path></svg>`，路径放在 IC 里（模板里已有一大批可复用；新图标自己画简单的 24×24 描边路径）。
- 无障碍：真 `<button>` / `<a href>` / `<input>` + `<label>`；纯图标按钮必须 `aria-label`；文字对比度靠令牌已经保证，别用 opacity 把小字变淡。
- 文案全部中文，`<title>` 写画板名。示例数据要像真的（主机名 DESKTOP-K5K9BBC，用户「超级管理员」），不要 lorem；真的不知道的值用 [占位] 写法，比如 [版本号]。

## PC 画板（宽 1440，页面型）

- 外壳照抄 AppsPC：`root` 用 `min-height: Hpx; display:flex; flex-wrap:wrap`；左侧 aside 侧边栏（Logo、节点切换、导航、底部用户卡），右侧 `main` `flex: 999 1 560px`。顶栏：全局搜索 + 深浅色按钮 + 通知（+ 本页需要时的主按钮）。
- 侧边栏导航统一用下面这份（把当前页对应 key 的项标成 `nav-on`）：
  ```js
  const nav = [
    { title: '', items: [{ t: '工作台', d: IC.home, href: 'DashboardPC.dc.html', key: 'dash' }] },
    { title: '应用', items: [{ t: '应用列表', d: IC.apps, href: 'AppsPC.dc.html', key: 'apps' }, { t: '应用类型', d: IC.layers, href: 'AppTypesPC.dc.html', key: 'types' }] },
    { title: '文件', items: [{ t: '文件管理', d: IC.folder, href: 'FilesPC.dc.html', key: 'files' }, { t: '分享', d: IC.share, href: 'SharesPC.dc.html', key: 'shares' }, { t: '文件源', d: IC.db, href: 'FileSourcesPC.dc.html', key: 'sources' }] },
    { title: 'Docker', items: [{ t: '容器 · 镜像 · 网络', d: IC.box, href: 'ContainersPC.dc.html', key: 'docker' }] },
    { title: '工具', items: [{ t: '端口转发', d: IC.forward, href: 'PortForwardPC.dc.html', key: 'pf' }, { t: '内网穿透', d: IC.tunnel, href: 'TunnelPC.dc.html', key: 'tunnel' }, { t: 'SSH 终端', d: IC.terminal, href: 'SSHPC.dc.html', key: 'ssh' }, { t: '智商检测', d: IC.spark, href: 'IQCheckPC.dc.html', key: 'iq' }] },
    { title: '管理', items: [{ t: '节点与 API Key', d: IC.server, href: 'NodesPC.dc.html', key: 'nodes' }, { t: '用户与权限', d: IC.users, href: 'UsersPC.dc.html', key: 'users' }, { t: '任务与日志', d: IC.clock, href: 'TasksPC.dc.html', key: 'tasks' }, { t: '系统设置', d: IC.settings, href: 'SettingsPC.dc.html', key: 'settings' }] }
  ];
  ```
  需要的新图标：`users: 'M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM2 21a7 7 0 0 1 14 0M16 3.5a4 4 0 0 1 0 7.5M22 21a7 7 0 0 0-5-6.7'`，`clock: 'M12 3a9 9 0 1 1 0 18 9 9 0 0 1 0-18zM12 7v5l3 2'`。
- 页内子分类用 ContainersPC 那种下划线 `utab`，链接到同组画板：
  - Docker：容器 `ContainersPC` / 镜像 `ImagesPC` / 网络 `NetworksPC` / 连接配置 `DockerConfigPC`
  - 节点：节点 `NodesPC` / API Key `ApiKeysPC`
  - 用户与权限：用户 `UsersPC` / 角色 `RolesPC` / 菜单 `MenusPC`
  - 任务与日志：后台任务 `TasksPC` / 操作记录 `AuditPC`
  - 系统设置：常规 · 登录 · 邮件 `SettingsPC` / OIDC 身份提供方 `OIDCPC`
  - 个人中心：个人资料 `ProfilePC` / 外观 `ThemePC`（手机同理加 Mobile 后缀）
- 列表页规则：标题 + 一句计数说明 + 右上主按钮；状态 chips 带数量（可点击筛选，用 state）；搜索框；**不要「序号」列**；行内常用操作用小图标按钮（日志/终端/启停…）+「⋯」；勾选后底部出现深色批量操作条（参考 AppsPC）。表格放在 `overflow-x:auto` 的 card 里。
- 链接：页面之间能跳的都用 `<a href="XxxPC.dc.html">`（文件名见上表和下面的分配表）。

## 手机画板（宽 390）

- 两种形式选其一：长页面（`width:390px; min-height:Hpx`，底部 sticky 导航）或 390×844 视口（内部 `overflow-y:auto` 滚动区 + 绝对定位底部导航/面板），按页面需要选。
- 底部导航 5 项：首页 `DashboardMobile` / 应用 `AppsMobile` / 容器 `ContainersMobile` / 文件 `FilesMobile` / 更多 `MoreMobile`。工具、管理、个人中心类页面高亮「更多」。非一级页面（详情、表单、设置子页）可以不放底部导航，顶部放返回按钮 `<a class="ib" href="上级页">`。
- **用户反馈过的规则**：卡片操作不要用两个占半行的大按钮；卡片底部一行（`border-top: 1px solid var(--line)`）左边放标签/端口等，右边一排透明底的 `.ib` 图标按钮（44px 点击区，`aria-label`），启停这类用颜色区分（停止 `var(--bad)`、启动 `var(--ok)`）。卡片信息要充实：关键字段放在一块 `background: var(--s2)` 的 `<dl>` 两列网格里（参考 AppsMobile）。
- 点击区 ≥ 44px；横向滚动的 chips/tab 行加 `.hs`；新建用右下悬浮按钮。

## 视觉语言（照模板）

大圆角（卡片 24px、按钮全圆）、Outfit 粗数字、彩色「贴纸」图标块（`.av-p .av-a .av-a2 .av-ok`，可轻微旋转）、主色实底的 hero 区块加黄色圆和紫色圆环装饰（不要渐变）、状态 pill 带圆点。
