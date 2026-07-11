import '@/styles/index.css' // 公共样式
import 'element-plus/theme-chalk/dark/css-vars.css' // Element Plus 深色模式样式
import '@/styles/design-tokens.css' // 设计令牌（在 EP 暗色变量之后导入，确保覆盖生效）
import 'nprogress/nprogress.css' // NProgress 样式
import '@/components/file/theme.css' // 文件模块自成体系的浅/暗设计令牌（全局，供 teleport 弹层共享）
import { APP_CONFIG } from '@/config/app.config' // 全局应用配置
import { loadingFadeOut } from 'virtual:app-loading' // 全局loading
import { permissionDirective } from '@/directives/permission' // 自定义权限指令
import { MotionPlugin } from '@vueuse/motion' // Motion 动画插件
import { createApp, nextTick } from 'vue'
import { createPinia } from 'pinia'
import App from '@/App.vue'
import router from '@/router/index'
import { i18n } from '@/locales'
import { setAppContext } from '@/utils/appContext'
import uiPlugin from '@nuxt/ui/vue-plugin' // Nuxt UI 运行时插件（虚拟模块，由 @nuxt/ui/vite 提供）

//  动态设置favicon和项目名称
const initAppConfig = () => {
  document.title = APP_CONFIG.name

  let faviconLink = document.querySelector("link[rel~='icon']") as HTMLLinkElement
  if (!faviconLink) {
    faviconLink = document.createElement('link')
    faviconLink.rel = 'icon'
    document.head.appendChild(faviconLink)
  }
  faviconLink.href = APP_CONFIG.faviconSrc
}

// 启动应用
const startApp = async () => {
  // 创建并挂载 Vue 应用
  const app = createApp(App)

  app.use(createPinia())
  app.use(i18n)
  app.use(router)
  app.use(uiPlugin)

  // 注册 Motion 动画插件
  app.use(MotionPlugin)

  // 注册自定义指令
  app.directive('permission', permissionDirective)

  // 保存应用上下文，供命令式渲染（如 utils/dialog.ts）复用插件（i18n 等）
  setAppContext(app._context)

  app.mount('#app')
  // 动态设置favicon和项目名称
  initAppConfig()

  // 等待路由完全准备好（包括动态路由加载）
  await router.isReady()

  // 再等待一个 tick，确保首次路由导航完成
  await nextTick()

  // 此时路由已完全加载，可以安全地隐藏 loading
  loadingFadeOut()
}

// 启动应用
startApp().catch((error) => {
  console.error('应用启动失败:', error)
})
