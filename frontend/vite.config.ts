import { fileURLToPath, URL } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import tailwindcss from '@tailwindcss/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import AppLoading from 'vite-plugin-app-loading'
import ui from '@nuxt/ui/vite'

// Normalize to forward slashes so Windows + pnpm paths match reliably.
const nm = (id: string) => id.replace(/\\/g, '/')

// Match a package root under node_modules (handles pnpm: .../node_modules/<pkg>/...).
const isPkg = (id: string, pkg: string) => {
  const p = nm(id)
  return p.includes(`/node_modules/${pkg}/`)
}

/**
 * manualChunks 注意事项：
 * - Vue 3 本体拆成 vue + @vue/* 多个包；只把 `vue` 放进 vendor-vue、@vue/*
 *   落进其它 chunk 会形成 vendor-vue ↔ vendor-misc 环，浏览器 TDZ：
 *   `Cannot access 'Kt' before initialization`。
 * - vendor-vue 只放「不会再依赖外部 vendor」的核心：vue / @vue/* / vue-router /
 *   pinia / vue-demi / vue-i18n / @intlify。
 * - 不要把 @vueuse 塞进 vendor-vue：@vueuse/motion 依赖 popmotion 等，
 *   会再次形成 vendor-vue → vendor-misc → vendor-vue 环。
 * - 重型、边界清晰的库单独拆；其余进 vendor-misc。
 */
const resolveManualChunks = (id: string) => {
  if (!nm(id).includes('/node_modules/')) return undefined

  // —— Vue 运行时核心：必须同 chunk，且内部不得再依赖其它 vendor chunk ——
  if (
    isPkg(id, 'vue') ||
    isPkg(id, '@vue') || // @vue/runtime-core | reactivity | shared | ...
    isPkg(id, 'vue-router') ||
    isPkg(id, 'pinia') ||
    isPkg(id, 'vue-demi') ||
    isPkg(id, 'vue-i18n') ||
    isPkg(id, '@intlify')
  ) {
    return 'vendor-vue'
  }

  if (isPkg(id, '@element-plus/icons-vue')) {
    return 'vendor-element-icons'
  }

  if (isPkg(id, '@heroicons/vue')) {
    const p = nm(id)
    if (p.includes('/24/solid/')) return 'vendor-hero-solid'
    return 'vendor-hero-outline'
  }

  if (isPkg(id, 'element-plus') || isPkg(id, '@element-plus')) {
    return 'vendor-element'
  }

  if (isPkg(id, 'echarts') || isPkg(id, 'vue-echarts')) {
    return 'vendor-echarts'
  }

  if (isPkg(id, 'zrender')) {
    return 'vendor-zrender'
  }

  if (isPkg(id, 'vxe-table')) {
    return 'vendor-vxe-table'
  }

  if (isPkg(id, 'vxe-pc-ui')) {
    return 'vendor-vxe-ui'
  }

  if (isPkg(id, '@vxe-ui')) {
    return 'vendor-vxe-core'
  }

  if (isPkg(id, 'xlsx')) {
    return 'vendor-xlsx'
  }

  if (isPkg(id, 'lottie-web')) {
    return 'vendor-lottie'
  }

  // Monaco / xterm 较大且与入口解耦，单独拆，避免撑爆 vendor-misc
  if (isPkg(id, 'monaco-editor') || isPkg(id, '@guolao/vue-monaco-editor') || isPkg(id, '@monaco-editor')) {
    return 'vendor-monaco'
  }

  if (isPkg(id, '@xterm') || isPkg(id, 'xterm')) {
    return 'vendor-xterm'
  }

  if (
    isPkg(id, 'axios') ||
    isPkg(id, 'dayjs') ||
    isPkg(id, '@bufbuild') ||
    isPkg(id, 'nprogress') ||
    isPkg(id, 'platform')
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
          imports: ['vue', 'vue-router', 'pinia', { '@/utils/feedback': ['feedback', 'useFeedback'] }],
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
