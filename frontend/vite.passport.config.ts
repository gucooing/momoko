import { fileURLToPath, URL } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import ui from '@nuxt/ui/vite'
import { NUXT_UI_PREBUNDLE } from './vite.shared'

/**
 * 用户通行证（玩家端）前端：和管理后台同一个工程，源码在 passport/，产物单独输出到 dist-passport/，
 * 由通行证 server（默认 :22634）单独提供。与管理后台共用设计令牌、src/games（游戏素材与类型）与
 * src/components/game；不引用管理后台的路由、store、文案与页面（src/components/ui 里依赖这些的组件不能用）。
 *
 * 游戏图片是 public/games/ 下的静态文件：开发时由这个 dev server 直接提供，打包时不复制
 * （publicDir: false），通行证 server 从管理端产物里提供 /games/，两端只有一份。
 */
const here = (p: string) => fileURLToPath(new URL(p, import.meta.url))

export default defineConfig(({ command, mode }) => {
  const env = loadEnv(mode, here('.'), '')
  return {
    root: here('./passport'),
    envDir: here('.'),
    base: env.VITE_STATIC_URL || '/',
    publicDir: command === 'serve' ? here('./public') : false,
    plugins: [
      vue(),
      tailwindcss(),
      ui({
        colorMode: false,
        ui: {
          colors: { primary: 'pink', neutral: 'slate' },
          button: { slots: { base: 'app-ui-btn rounded-(--app-radius-btn) font-semibold' } },
          badge: { slots: { base: 'rounded-full font-semibold' } },
        },
        autoImport: {
          imports: ['vue', 'vue-router', 'pinia'],
          dirs: [here('./passport/src/stores')],
          dts: here('./passport/auto-imports.d.ts'),
        },
        components: {
          dirs: [here('./src/components/game'), here('./passport/src/components')],
          dts: here('./passport/components.d.ts'),
        },
      }),
    ],
    optimizeDeps: {
      include: NUXT_UI_PREBUNDLE,
    },
    resolve: {
      alias: {
        '@': here('./src'),
        '@passport': here('./passport/src'),
      },
    },
    build: {
      outDir: here('./dist-passport'),
      emptyOutDir: true,
    },
    server: {
      host: '0.0.0.0',
      port: 3008,
    },
  }
})
