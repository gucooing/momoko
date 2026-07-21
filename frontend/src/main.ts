import '@/styles/index.css' // 公共样式
import '@/styles/design-tokens.css' // 设计令牌（含 --el-* 语义色，已自给自足，不再依赖 EP 暗色 css）
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

  // public 页常被 HTTPS 父页 iframe 嵌入；开发态 favicon 是 http://localhost/...，会触发 Mixed Content
  if (typeof window !== 'undefined' && window.location.pathname.startsWith('/public/')) {
    return
  }

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
