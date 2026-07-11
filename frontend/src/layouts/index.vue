<template>
  <div class="layout-container">
    <AppShell />
    <ThemeConfig />
  </div>
</template>

<script setup lang="ts">
import AppShell from '@/layouts/app/AppShell.vue'
import ThemeConfig from '@/components/ThemeConfig.vue'

defineOptions({ name: 'LayoutView' })
</script>

<style scoped lang="scss">
.layout-container {
  width: 100%;
  height: 100vh;
}
</style>
