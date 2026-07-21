<template>
  <!-- toaster 必须高于 FormDialog/BaseDialog(2200)、ThemeConfig(2300)、AppSelect(2300) -->
  <UApp :toaster="{ position: 'top-right', ui: { viewport: 'z-[10000]' } }">
    <FeedbackBridge>
      <RouterView />
    </FeedbackBridge>
  </UApp>
</template>

<script setup lang="ts">
// EP el-config-provider 已下线；i18n 仍由 vue-i18n 接管，Nuxt UI 随 UApp。
</script>

<style>
/* 兜底：强制 toast 视口压过所有业务弹层（含 teleport 到 body 的对话框） */
[data-slot='viewport'] {
  z-index: 10000 !important;
}
</style>
