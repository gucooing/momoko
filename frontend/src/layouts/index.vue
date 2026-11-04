<template>
  <div class="layout-container">
    <AppShell />
  </div>
</template>

<script setup lang="ts">
import AppShell from '@/layouts/app/AppShell.vue'

defineOptions({ name: 'LayoutView' })
</script>

<style scoped lang="scss">
.layout-container {
  width: 100%;
  height: 100vh;
  height: 100dvh;
}
</style>
