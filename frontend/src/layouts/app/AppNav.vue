<!-- 导航容器：读 menuStore.menuList（已按 active 过滤），递归渲染 AppNavItem。 -->
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
const items = computed(() => menuStore.menuList)
</script>

<style scoped lang="scss">
.app-nav {
  display: flex;
  flex-direction: column;
  padding: 6px 10px 16px;
}
</style>
