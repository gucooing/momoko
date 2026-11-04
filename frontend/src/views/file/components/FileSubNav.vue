<!-- 文件页内标签：文件 / 分享 / 文件源（/file?tab=…）。只在手机端显示，PC 端由侧边栏的三个入口切换。 -->
<template>
  <SubNavTabs v-if="menuStore.isMobile" group="file" />
</template>

<script setup lang="ts">
const menuStore = useMenuStore()
</script>
