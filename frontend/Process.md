## 项目规范（必须遵守）

### 编码与文件基础规则

1. 所有代码文件必须使用 `UTF-8` 编码查看、编辑和保存。
2. 注释、假数据、占位文案优先使用中文，表达必须直接清晰。
3. 编码必须遵循现有项目风格，优先最简实现，不添加无意义结构、无意义封装、无意义中间层。
4. 所有实现必须同时满足“代码最简”和“运行高效”，不能只追求写法简短而忽略运行效率，也不能为了过度设计牺牲代码简洁性。
5. 禁止复制粘贴式重复代码；能通过已有结构复用、归并、抽取解决的，不能重复实现同一套逻辑。
6. 禁止保留不必要代码，包括无实际用途的变量、类型、注释、兼容层、假数据、临时结构、废弃分支和占位实现。
7. 能复用现有规范结构时，禁止额外新增平行结构。

### 数据层规范

1. 临时数据、页面状态、筛选条件、展示模型、业务枚举、业务映射、业务派生数据，统一放到 `src/stores` 对应业务域中。
2. 页面文件只保留视图绑定、页面交互、路由跳转、提示信息，不承担数据层职责。
3. `stores` 必须按业务域或职责拆分文件夹，禁止继续把业务 store 平铺在 `src/stores` 根目录。
4. 业务类型如果是给某个 store 域服务的，也应放到该 `stores` 域中，而不是放到 `src/types`、`src/views`、`src/components`。
5. 临时枚举、临时状态值、临时结构体，只要属于业务数据层，就必须归属到对应 `stores` 域。

### `types` 目录规范

1. `src/types` 只保留真实后端接口契约、生成代码、协议定义，以及确实属于全局外部契约的类型。
2. 纯前端业务模型、假数据结构、页面展示结构、临时枚举，禁止继续放在 `src/types` 根目录。
3. 当后端真实 API 已提供对应结构、枚举、状态值后：
   - 必须完全移除前端临时枚举和临时结构。
   - 必须改用后端提供的真实类型与真实枚举。
   - 禁止前后端两套结构长期并存。
4. “完全移除”指删除旧临时类型、旧映射、旧假枚举，而不是保留兼容层继续混用。

### 目录归位规范

1. 路由定义、路由转换、路由相关逻辑统一放到 `src/router`。
2. 数据层相关代码统一放到 `src/stores`。
3. `ts` 文件与 `vue` 页面/组件文件不要为了图方便混放在同一个页面组件目录中。
4. 如果某个类型非常轻量、只服务单个页面或单个组件，优先直接内联；如果属于业务数据层，优先放入对应 `stores` 域。
5. 组件目录放组件本身，页面目录放页面本身，不要让它们承担数据模型仓库的职责。

### 实施原则

1. 做规范化时优先做减法，能删除就删除，能归并就归并，避免结构越来越多。
2. 发现现有结构可被更规范结构替代时，应直接迁移并删除旧结构，而不是保留旧结构继续累积。
3. 新增接口落地后，要主动回查对应页面和 store，确认是否还残留旧假数据、旧临时字段、旧前端枚举；如有，必须一并清理。
4. 任何重构和新增代码都要顺手检查是否产生重复逻辑；如果已经出现重复，必须在本次修改中一起收敛。
5. 禁止为了兼容旧假数据、旧页面模型、旧组件入参而新增冗余转换；能直接使用后端返回结构的，必须直接使用，不能在页面、store、组件之间来回做无意义映射。
6. 如果某个展示结构只是对后端字段的轻量投影，也必须优先内联或集中在单一数据层出口生成，禁止同一份数据在多个页面/组件各自重复转换。

### 请求响应处理规范（新增）

1. 后端通用响应结构统一在 `src/utils/request.ts` 处理，禁止在页面、组件、store 中重复解析 `code`、`message`、`data`。
2. 请求失败提示必须统一走请求层；如果页面层确实需要兜底提示，必须使用统一错误辅助方法，禁止直接在 `catch` 中再次 `ElMessage.error`，避免同一错误弹出两次。
3. 页面层只负责成功提示、页面状态更新和少量本地校验提示，不负责重复处理通用请求失败文案。
4. 任何新增接口接入时，都必须优先复用现有请求封装和统一错误处理，禁止平行新增一套响应处理方式。
5. 只要后端已生成对应 `Request` 结构，前端 API 封装必须显式使用该结构；即使该请求当前没有字段，也必须保留对应 `Request` 类型和默认空结构（如 `params: XxxRequest = {}`），禁止擅自改成无参函数。
6. 如果后端已为路径参数、查询参数或请求体定义了统一 `Request` 结构，前端调用层也必须优先传递该结构对象，禁止为了图省事改成裸值参数，避免后续接口字段扩展时再次返工。

### 静态资源与首屏性能规范（新增）

1. 首屏入口 `src/main.ts` 只允许保留真正全局且高频必需的依赖；`echarts`、`vue-echarts`、`vxe-table`、`vxe-pc-ui`、`lottie-web`、`xlsx` 等重库禁止再次通过 `main.ts` 全局引入。
2. 页面级或少量页面使用的重库必须按页面或组件懒加载；新增图表默认复用 `src/components/chart/VChart.vue`，禁止在普通页面直接把 `vue-echarts` 重新变回全局入口依赖。
3. Lottie 动画默认复用 `src/components/animation/LottieAnimation.vue`，并优先通过 `path` 或 `?url` 加载大体积 JSON；除非确有必要，禁止把大动画 JSON 作为对象直接打进首屏 JS。
4. `xlsx` 必须保持动态导入，只能在真实导出场景触发加载；禁止恢复为顶层静态 `import 'xlsx'`。
5. VXE 相关能力必须按页面接入，默认复用 `src/plugins/vxeGrid.ts` 和 `src/styles/vxeGrid.css`；禁止重新恢复 `src/plugins/vxeTable.ts` 一类全局注册方案。
6. 普通分页默认优先使用轻量方案，禁止为了分页把整套 VXE 能力带入非表格页面首屏链路。
7. Element Plus 样式必须保持按需加载，禁止重新引入 `element-plus/dist/index.css`；新增组件时优先沿用 `vite.config.ts` 中的 `ElementPlusResolver({ importStyle: 'css' })`。
8. 图标运行时默认复用 `src/config/iconRegistry.ts`；禁止在运行时代码中再次使用 `import * as ElIcons`、`import * as HeroOutlineIcons`、`import * as HeroSolidIcons` 这类整包导入。完整图标集合只允许在图标选择器等明确场景中异步加载。
9. 新增第三方库时，必须先判断是否属于首屏必需；如果不是首屏必需，必须在首次使用点延迟加载，并同步检查是否需要在 `vite.config.ts` 的 `manualChunks` 中单独拆包。
10. 禁止通过调大 `chunkSizeWarningLimit`、关闭 `cssCodeSplit`、合并整包样式等方式掩盖体积问题；必须优先做真实拆分和按需加载。
11. 如果继续产出 `.gz` / `.br` 预压缩文件，必须始终保留原始 `.js` / `.css` 文件，确保普通静态托管和后端嵌入分发都可正常工作。
12. 任何影响打包体积的改动提交前，都应至少检查一次 `dist` 的大文件分布，确认首屏入口没有重新引入 `xlsx`、`lottie`、`echarts`、`vxe` 等非首屏必需库。

## 安装 Element Plus

```shell
pnpm install element-plus
```

## 安装自动导入

```shell
pnpm install -D unplugin-vue-components unplugin-auto-import
```

```typescript
import { defineConfig } from 'vite'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

export default defineConfig({
  plugins: [
    AutoImport({
      imports: ['vue', 'vue-router', 'pinia'], // 指定导入模块
      dirs: ['src/stores'], // 指定导入目录
      dts: 'src/auto-imports.d.ts', // 指定生成文件路径
      resolvers: [ElementPlusResolver()],
    }),
    Components({
      resolvers: [ElementPlusResolver()],
    }),
  ],
})
```

在 `tsconfig.app.json` 中添加：

```json
"include": ["env.d.ts", "src/**/*", "src/**/*.vue", "components.d.ts", "auto-imports.d.ts"]
```

## 浅色模式 / 深色模式

### 实现原理

使用 `@vueuse/core` 的 `useDark` 和 `useToggle` 实现主题模式切换。

### 核心代码

```typescript
import { useDark, useToggle } from '@vueuse/core'

// 在 store 中
const isDark = useDark()
const toggleDark = useToggle(isDark)

// 主题模式状态
const themeMode = ref<'light' | 'dark'>(
  (localStorage.getItem('themeMode') as 'light' | 'dark') || 'light',
)

// 切换主题模式
const toggleThemeMode = (newVal: 'light' | 'dark') => {
  themeMode.value = newVal
  toggleDark(newVal === 'dark')
  localStorage.setItem('themeMode', newVal)
}
```

### 工作原理

1. `useDark()` 会自动检测系统主题偏好，并在 `<html>` 标签上添加或移除 `dark` 类。
2. 用户选择的主题模式会保存到 `localStorage`，页面刷新后自动恢复。
3. 如果使用 Element Plus 深色模式，需要在 `main.ts` 中引入：

```typescript
import 'element-plus/theme-chalk/dark/css-vars.css'
```

### 使用方式

```typescript
const themeStore = useThemeStore()

themeStore.toggleThemeMode('dark')
themeStore.toggleThemeMode('light')
```

## 主题色变更

### 实现原理

通过动态设置 CSS 变量来修改 Element Plus 的主题色，并自动计算相关的浅色和深色变体。

### 核心代码

```typescript
// 设置 Element Plus 主题色变量
const setPrimaryColor = (color: string) => {
  const root = document.documentElement
  root.style.setProperty('--el-color-primary', color)
  root.style.setProperty('--el-color-primary-light-3', `color-mix(in srgb, ${color} 70%, white)`)
  root.style.setProperty('--el-color-primary-light-5', `color-mix(in srgb, ${color} 50%, white)`)
  root.style.setProperty('--el-color-primary-light-7', `color-mix(in srgb, ${color} 30%, white)`)
  root.style.setProperty('--el-color-primary-light-8', `color-mix(in srgb, ${color} 20%, white)`)
  root.style.setProperty('--el-color-primary-light-9', `color-mix(in srgb, ${color} 10%, white)`)
  root.style.setProperty('--el-color-primary-dark-2', `color-mix(in srgb, ${color} 80%, black)`)
}

// 主题颜色状态
const primaryColor = ref(localStorage.getItem('theme-color-primary') || '#8B5CF6')
setPrimaryColor(primaryColor.value)

// 切换主题颜色
const togglePrimaryColor = (colorValue: string) => {
  primaryColor.value = colorValue
  localStorage.setItem('theme-color-primary', colorValue)
  setPrimaryColor(colorValue)
}
```

### 工作原理

1. 直接操作 `document.documentElement.style` 设置 CSS 变量。
2. 使用 CSS `color-mix()` 自动计算主题色的浅色和深色变体。
3. 主题色保存在 `localStorage` 中，页面刷新后自动恢复。
4. Element Plus 组件会自动使用这些 CSS 变量，无需额外配置。

### 使用方式

```typescript
const themeStore = useThemeStore()
themeStore.togglePrimaryColor('#10B981')
```

### 注意事项

- `color-mix()` 需要现代浏览器支持，例如 Chrome 111+、Safari 16.4+。
- 如果浏览器不支持，可以考虑使用 JavaScript 颜色库（如 `tinycolor2`）计算颜色变体。

## 全局 Loading

### 实现原理

使用 `vite-plugin-app-loading` 在应用启动前显示全局 Loading 动画，避免页面刷新时出现白屏。插件会在 HTML 中自动注入 Loading 元素，覆盖应用初始化、Vue 挂载和路由初始化阶段。

### 安装插件

```shell
pnpm add -D vite-plugin-app-loading
```

### 配置 Vite

在 `vite.config.ts` 中添加插件：

```typescript
import { defineConfig } from 'vite'
import AppLoading from 'vite-plugin-app-loading'

export default defineConfig({
  plugins: [
    // ... 其他插件
    AppLoading(),
  ],
})
```

### 创建 `loading.html`

在项目根目录（与 `index.html` 同级）创建 `loading.html`，插件会自动读取并注入该文件内容。

注意：`loading.html` 中必须包含 `id="__app-loading__"` 的元素。

示例：

```html
<div id="__app-loading__" class="app-loading">
  <div class="app-loading-content">
    <img src="/logo.png" alt="logo" class="loading-logo" />
    <div class="loading-text">正在加载...</div>
  </div>
</div>
```

### 在 `main.ts` 中使用

```typescript
import { loadingFadeOut } from 'virtual:app-loading'
import { createApp, nextTick } from 'vue'

const app = createApp(App)
app.mount('#app')

// 等待路由完全准备好（包括动态路由加载）
await router.isReady()
// 再等待一个 tick，确保首次路由导航完成
await nextTick()
// 此时路由已完全加载，可以安全隐藏 loading
loadingFadeOut()
```

如果要让 TypeScript 识别虚拟导入类型，在 `tsconfig.app.json` 中加入：

```json
{
  "compilerOptions": {
    "types": ["vite-plugin-app-loading/client"]
  }
}
```

### 工作原理

1. `vite-plugin-app-loading` 会在 HTML 中自动注入 `loading.html` 的内容。
2. Loading 覆盖从页面刷新到 Vue 应用挂载完成的整个过程。
3. 通过 `router.isReady()` 确保动态路由加载完成。
4. 在路由完全准备好后调用 `loadingFadeOut()` 隐藏 Loading。

## Heroicons

### 安装

```shell
npm install @heroicons/vue
# 或 pnpm install @heroicons/vue
# 或 yarn add @heroicons/vue
```

### 图标样式和尺寸

Heroicons 提供多种样式和尺寸：

- `@heroicons/vue/24/outline`
- `@heroicons/vue/24/solid`
- `@heroicons/vue/20/solid`
- `@heroicons/vue/16/solid`

### 基本使用

```vue
<template>
  <div>
    <BeakerIcon class="h-6 w-6 text-blue-500" />
    <HomeIcon class="h-6 w-6 text-gray-600" />
  </div>
</template>

<script setup>
import { BeakerIcon } from '@heroicons/vue/24/solid'
import { HomeIcon } from '@heroicons/vue/24/outline'
</script>
```

### 图标尺寸设置

重要提示：Heroicons 是 SVG 图标，不是字体图标，因此：

- 不能使用 `font-size` 直接设置图标大小。
- 应通过 `width` 和 `height` 设置尺寸。

#### 正确方式

```vue
<template>
  <HomeIcon style="width: 24px; height: 24px;" />
  <HomeIcon class="w-6 h-6" />
  <HomeIcon class="icon-size" />
</template>

<style scoped>
.icon-size {
  width: 24px;
  height: 24px;
}
</style>
```

#### 如果需要使用 `font-size`

如果希望像字体图标一样通过 `font-size` 控制大小，可以采用以下两种方案。

方案 1：使用 `el-icon` 包裹

```vue
<template>
  <el-icon :size="20">
    <Cog6ToothIcon />
  </el-icon>
</template>

<script setup>
import { Cog6ToothIcon } from '@heroicons/vue/24/outline'
</script>
```

`el-icon` 会把 `size` 自动映射为 SVG 的 `width` 与 `height`。

方案 2：自行封装组件

```vue
<!-- IconWrapper.vue -->
<template>
  <component :is="icon" :style="{ width: size, height: size }" />
</template>

<script setup>
defineProps({
  icon: {
    type: Object,
    required: true,
  },
  size: {
    type: [String, Number],
    default: '1em',
  },
})
</script>
```

使用方式：

```vue
<template>
  <IconWrapper :icon="Cog6ToothIcon" size="20px" />
  <div style="font-size: 20px;">
    <IconWrapper :icon="Cog6ToothIcon" size="1em" />
  </div>
</template>

<script setup>
import { Cog6ToothIcon } from '@heroicons/vue/24/outline'
import IconWrapper from './IconWrapper.vue'
</script>
```

### 页面拆分规范（新增）

1. 页面实现必须按“页面入口 + 子组件”拆分，禁止把筛选区、工具栏、弹窗表单、复杂卡片等长期堆在单一 `.vue` 文件中。
2. 原则上每个业务页面至少包含一个入口页（如 `index.vue`）和按职责拆分的子文件（如 `filters.vue`、`toolbar.vue`、`editor.vue`）。
3. 页面入口文件只负责路由、页面级交互编排和组件拼装；可复用或复杂片段必须下沉为同目录独立组件文件。
4. 新增或整改页面时，若单文件已超过可维护范围（逻辑与模板明显混杂），必须在本次改动中完成拆分，不得继续堆积。

### 复用优先规范（新增）

1. 功能、交互、数据模型只要已存在可复用实现，默认先复用，禁止在页面内平行再造一套逻辑。
2. 当同一实现预期会在两个及以上业务页面使用时，应抽离为公共组件或公共模块，避免后续多点修改。
3. 抽离公共能力时，必须保持单一职责和稳定接口，页面仅做组装，不承载可复用逻辑副本。

### 非侧边栏独立标签页适配规范（新增）

1. 侧边栏不展示但允许打开独立标签页的路由，必须保证可稳定参与布局级 `<Transition mode="out-in">` 切换。
2. 布局层 `RouterView` 过渡节点必须使用单一元素壳（例如 `route-page-shell`），禁止直接让多根节点页面作为过渡根节点。
3. 该类路由默认以 `fullPath` 作为标签唯一标识和页面渲染 key，确保参数上下文不丢失。
4. 此规范属于通用能力，新增同类页面时必须复用现有适配，不允许页面内临时补丁式修复。
