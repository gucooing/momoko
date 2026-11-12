// 用户通行证（玩家端）入口：独立的路由、store 与文案，和管理后台共用设计令牌与游戏素材（src/games）。
import '@fontsource-variable/outfit'
import '@/styles/index.css'
import '@/styles/design-tokens.css'
import '@/games/gakumas/theme.css'
import './styles.css'
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import uiPlugin from '@nuxt/ui/vue-plugin'
import App from './App.vue'
import router from './router'
import { i18n } from './locales'
import { useUiStore } from './stores/ui'

const app = createApp(App)
app.use(createPinia())
app.use(i18n)
app.use(router)
app.use(uiPlugin)
useUiStore().initScheme()
document.documentElement.lang = i18n.global.locale.value
app.mount('#app')
