import { fileURLToPath, URL } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import tailwindcss from '@tailwindcss/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import AppLoading from 'vite-plugin-app-loading'
import ui from '@nuxt/ui/vite'

const resolveManualChunks = (id: string) => {
  if (!id.includes('node_modules')) return undefined

  if (
    id.includes('/node_modules/@element-plus/icons-vue/')
  ) {
    return 'vendor-element-icons'
  }

  if (id.includes('/node_modules/@heroicons/vue/24/outline/')) {
    return 'vendor-hero-outline'
  }

  if (id.includes('/node_modules/@heroicons/vue/24/solid/')) {
    return 'vendor-hero-solid'
  }

  if (
    id.includes('/node_modules/vue/') ||
    id.includes('/node_modules/vue-router/') ||
    id.includes('/node_modules/pinia/') ||
    id.includes('/node_modules/@vueuse/')
  ) {
    return 'vendor-vue'
  }

  if (
    id.includes('/node_modules/element-plus/') ||
    id.includes('/node_modules/@element-plus/')
  ) {
    return 'vendor-element'
  }

  if (
    id.includes('/node_modules/echarts/') ||
    id.includes('/node_modules/vue-echarts/')
  ) {
    return 'vendor-echarts'
  }

  if (id.includes('/node_modules/zrender/')) {
    return 'vendor-zrender'
  }

  if (id.includes('/node_modules/vxe-table/')) {
    return 'vendor-vxe-table'
  }

  if (id.includes('/node_modules/vxe-pc-ui/')) {
    return 'vendor-vxe-ui'
  }

  if (id.includes('/node_modules/@vxe-ui/')) {
    return 'vendor-vxe-core'
  }

  if (id.includes('/node_modules/xlsx/')) {
    return 'vendor-xlsx'
  }

  if (id.includes('/node_modules/lottie-web/')) {
    return 'vendor-lottie'
  }

  if (
    id.includes('/node_modules/axios/') ||
    id.includes('/node_modules/dayjs/') ||
    id.includes('/node_modules/@bufbuild/') ||
    id.includes('/node_modules/nprogress/') ||
    id.includes('/node_modules/platform/')
  ) {
    return 'vendor-utils'
  }

  return 'vendor-misc'
}

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const isProd = mode === 'production'

  return {
    base: env.VITE_STATIC_URL || '/',
    plugins: [
      vue(),
      tailwindcss(),
      // Nuxt UI v4（Reka UI + Tailwind v4）。colorMode:false —— 明暗仍由 themeStore 的 useDark() 独占 .dark。
      // 主色 primary=teal（teal-500 = #14b8a6 薄荷青），中性 neutral=slate。
      // 自建 auto-import/components 各写独立 dts，避免与项目现有 unplugin 实例互相覆盖。
      // Nuxt UI 只允许单个 unplugin-auto-import / unplugin-vue-components 实例，
      // 故把项目原有配置并入这里（defu 合并：数组与 Nuxt UI 默认项拼接，二者共存）。
      ui({
        colorMode: false,
        ui: { colors: { primary: 'teal', neutral: 'slate' } },
        autoImport: {
          imports: ['vue', 'vue-router', 'pinia'],
          dirs: ['src/stores', 'src/stores/**'],
          resolvers: [ElementPlusResolver()],
        },
        components: {
          resolvers: [ElementPlusResolver()],
          dirs: ['src/components'], // src/components 下组件自动全局注册（PascalCase）
        },
      }),
      !isProd && vueDevTools(),
      AppLoading(),
    ].filter(Boolean),
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    build: {
      rollupOptions: {
        output: {
          manualChunks: resolveManualChunks,
        },
      },
    },
    optimizeDeps: {
      // Pre-bundle Element Plus deep style imports to avoid runtime re-optimization reloads.
      include: ['element-plus/es/components/*/style/css'],
    },
    server: {
      host: '0.0.0.0',
      port: 3007,
      open: true,
    },
  }
})
