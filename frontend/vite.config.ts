import { fileURLToPath, URL } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import tailwindcss from '@tailwindcss/vite'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import AppLoading from 'vite-plugin-app-loading'

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
      !isProd && vueDevTools(),
      AppLoading(),
      AutoImport({
        imports: ['vue', 'vue-router', 'pinia'],
        dirs: ['src/stores', 'src/stores/**'],
        resolvers: [ElementPlusResolver()],
      }),
      Components({
        resolvers: [ElementPlusResolver()],
        dirs: ['src/components'], // 指定组件目录,注册为全局组件
      }),
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
