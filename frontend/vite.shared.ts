// 管理后台（vite.config.ts）与用户通行证（vite.passport.config.ts）共用的构建设置。

// Nuxt UI 把自己排除在依赖预构建之外（它要经 Vite 解析 #imports），它引用的库因此不会被自动预构建：
// 开发时 reka-ui 等会按几百个源文件逐个请求。这里显式预构建（nested 写法从 @nuxt/ui 的位置解析）。
export const NUXT_UI_PREBUNDLE = [
  '@nuxt/ui > reka-ui',
  '@nuxt/ui > defu',
  '@nuxt/ui > ufo',
  '@nuxt/ui > ohash',
  '@nuxt/ui > tailwind-variants',
  '@nuxt/ui > scule',
  '@nuxt/ui > hookable',
  '@nuxt/ui > @iconify/vue',
  '@nuxt/ui > @internationalized/date',
  '@nuxt/ui > @floating-ui/dom',
  '@nuxt/ui > @tanstack/vue-virtual',
  '@nuxt/ui > vaul-vue',
  '@nuxt/ui > motion-v',
  '@nuxt/ui > embla-carousel-vue',
  '@nuxt/ui > colortranslator',
]
