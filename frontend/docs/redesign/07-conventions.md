# 07 · 代码约定(Conventions)

> 保证多会话、跨模块写出来的代码风格一致、可维护。写代码前读本文件 + `03b`(组件用法)。

---

## 1. 目录与文件

```
src/
  components/
    ui/            # 通用设计系统组件(自建,自动全局注册)——见 03
    <domain>/      # 业务专用组件(file/share/…)保留
  layouts/app/     # 新外壳——见 04
  views/<module>/<page>/index.vue   # 路由页(路由=文件夹路径,勿挪)
                       create.vue / *Dialog.vue  # 页内弹窗/子组件
  stores/<module>/ # pinia(保留现有,不改对外契约)
  api/             # axios 请求层(保留)
  schemas/         # 表单校验 schema(valibot),新目录
  config/          # permission(PERM)、app.config、iconMap(新)
  utils/           # feedback(toast/confirm)、dialog(useOverlay)、appContext…
  locales/         # vue-i18n(messages.ts / types)
  styles/          # index.css(Tailwind + @nuxt/ui + design-tokens)、design-tokens.css
```
- **路由页文件位置不可随意移动**(`menuToRoute` 用 `menu.path → /src/views/<path>/index.vue`)。
- 新自建组件放 `components/ui/`,PascalCase 文件名 = 组件名(自动注册)。

## 2. 组件与样式
- 只用 `03` 的通用组件 + Nuxt UI `U*` 原子 + `01` 令牌。**禁止**页面内写死颜色/阴影/圆角魔法值;用 token / Tailwind 语义类。
- 优先 Tailwind 原子类 + 少量 scoped SCSS;复杂/复用样式沉淀到 `components/ui`。
- 图标:`<UIcon name="i-lucide-…"/>`;菜单等存量图标经 `config/iconMap.ts` 解析(见 `03b` §3)。
- 响应式:优先 CSS/Tailwind 断点(`sm/md/lg/xl`);结构级切换(表↔卡、抽屉)才用 JS(`menuStore` 派生断点,见 `02`)。

## 3. 交互设施(Nuxt UI)
- 提示:`utils/feedback.ts` → `toast.success/error/info`(封装 `useToast`)。**禁用** `ElMessage`。
- 确认:`confirmDanger({title,description})`(基于 `UModal`/`AdaptiveConfirm`)。**禁用** `ElMessageBox`。
- 命令式弹窗:`useOverlay()` 打开 `UModal`;**禁用** EP 命令式渲染。
- 加载:骨架(`Skeleton/CardSkeleton/TableSkeleton`)或按钮 `loading`;**禁用** `v-loading`。
- 表单:`UForm :schema`(`schemas/*`,valibot)+ `UFormField`;错误字段级展示。

## 4. i18n
- 文案走 `vue-i18n`;新增键加 `locales/messages.ts`(zh + en 同步)。已知业务名用 `translateKnownText`。
- **禁止**模板里硬编码中文(除品牌名"Momoko")。Nuxt UI 组件内置文案由 `<UApp :locale>` 提供,勿混。

## 5. 权限
- 按钮级:`v-permission="[PERM.XXX]"` + `PERM`(`config/permission.ts`);组合判断用 `useButtonPermission`。
- 菜单级:后端下发 + `menuStore` 过滤,导航层不额外判断。
- 保留内置超管(`role_1`)/内置角色的受限规则。

## 6. 数据与状态
- 请求走 `api/*`(axios 封装 `utils/request`);类型用 `types/v1/*`(protobuf 生成,勿手改)。
- 组件状态就近;跨页/共享用现有 pinia store,**不改 store 对外契约**(只在其上换 UI)。
- 列表页:分页参数、筛选、选中态标准化(page/pageSize/total + query + selectedIds)。

## 7. keepAlive / 标签
- 沿用 `route.meta.keepAlive` 与 `tabsStore.getRouteRenderKey`;不破坏多标签缓存语义。
- 终端/文件/控制台等有状态页保持 keepAlive。

## 8. 可访问性 / 质量门槛
- `:focus-visible` 主色环;表单错误有文字;对比度达 `01`;`prefers-reduced-motion` 降级动画。
- 每页:桌面(1440)+ 移动(390)截图核对;明/暗两态核对。

## 9. Git / 提交
- 分支:重大重写按阶段推进;提交信息中文、聚焦一件事(如 `外壳:重写侧边栏(Nuxt UI)`)。
- 提交结尾附:
  `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`
- 不提交 `.browser-tmp/`、构建产物、二进制。

## 10. 验证工作流(每次改动)
- 用浏览器 MCP(`chrome-devtools`,下会话直接可用;本会话见 `.browser-tmp/b.mjs`)截图桌面 + 移动核对。
- 参考 `README` §5 验收标准逐项过;通过后更新 `08-progress.md`。
