<!-- 导航容器：读 menuStore.navMenuList（已按 active 与当前节点权限过滤），递归渲染 AppNavItem。 -->
<template>
  <nav class="app-nav">
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

defineProps<{ collapsed?: boolean }>()
const menuStore = useMenuStore()
const items = computed(() => menuStore.navMenuList)
</script>

<style scoped lang="scss">
.app-nav {
  display: flex;
  flex-direction: column;
  padding: 4px 12px 16px;
}
</style>
