<template>
  <!-- toaster 必须高于 FormDialog/BaseDialog(2200)、ThemeConfig(2300)、AppSelect(2300) -->
  <UApp :toaster="{ position: 'top-right', ui: { viewport: 'z-[10000]' } }">
    <FeedbackBridge>
      <!-- 自定义背景层（主题背景铺在整个界面下面） -->
      <ThemeBackdrop />
      <RouterView />
      <!-- 主题预览模式的悬浮条 -->
      <ThemePreviewBar />
    </FeedbackBridge>
  </UApp>
</template>

<script setup lang="ts">
// EP el-config-provider 已下线；i18n 仍由 vue-i18n 接管，Nuxt UI 随 UApp。
import ThemeBackdrop from '@/components/theme/ThemeBackdrop.vue'
import ThemePreviewBar from '@/components/theme/ThemePreviewBar.vue'
import { pageKeyOf } from '@/theme/pages'
import { installThemeFrameBridge } from '@/theme/frameBridge'
import { useAppearanceSync } from '@/stores/theme/sync'

const route = useRoute()
const router = useRouter()
const themeStore = useThemeStore()

// 页面单独设置按「匹配到的路由模板」取，例如 /instance/app/:instanceId；
// 路由 meta.siteTheme 的页面（登录页）是访客看到的样子，套用全站默认主题。
watch(
  () => [route.matched[route.matched.length - 1]?.path, !!route.meta.siteTheme] as const,
  ([path, siteTheme]) => {
    themeStore.setRoute(path ? pageKeyOf(path) : null, siteTheme)
    if (siteTheme) void themeStore.loadSiteTheme()
  },
  { immediate: true },
)

installThemeFrameBridge(router)
// 外观改动自动保存到账号（挂在根组件上：外壳之外的全屏页，如主题编辑器，也要能保存）
useAppearanceSync()
</script>

<style>
/* 兜底：强制 toast 视口压过所有业务弹层（含 teleport 到 body 的对话框） */
[data-slot='viewport'] {
  z-index: 10000 !important;
}
</style>
