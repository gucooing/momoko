<!-- 导航容器：读 menuStore.sidebarMenuList（已按启用状态、当前节点权限与隐藏设置过滤），递归渲染 AppNavItem。 -->
<template>
  <nav class="app-nav" :class="{ 'is-collapsed': collapsed }" :aria-label="t('shell.nav.main')">
    <AppNavItem
      v-for="item in items"
      :key="item.id"
      :item="item"
      :depth="0"
      :collapsed="collapsed"
    />
  </nav>
</template>

<script setup lang="ts">
import AppNavItem from './AppNavItem.vue'
import { useI18n } from 'vue-i18n'

defineProps<{ collapsed?: boolean }>()
const menuStore = useMenuStore()
const { t } = useI18n()
const items = computed(() => menuStore.sidebarMenuList)
</script>

<style scoped lang="scss">
.app-nav {
  display: flex;
  flex-direction: column;
  padding: 0 16px 8px;
}
.app-nav.is-collapsed {
  padding: 0 10px 8px;
}
</style>
