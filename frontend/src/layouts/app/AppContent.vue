<!-- 内容区：router-view + 轻 fade + 内边距/限宽 + renderKey（复刻原语义：无 keep-alive，导航即重挂载）。 -->
<template>
  <main class="app-content" :class="{ 'app-content--bleed': isBleed }">
    <RouterView v-slot="{ Component, route: r }">
      <Transition :name="r.meta?.disableTransition === true ? '' : 'fade-slide'" mode="out-in">
        <div :key="tabsStore.getRouteRenderKey(r.fullPath)" class="app-page">
          <component :is="Component" />
        </div>
      </Transition>
    </RouterView>
  </main>
</template>

<script setup lang="ts">
const route = useRoute()
const tabsStore = useTabsStore()
const isBleed = computed(() => !!route.meta?.fullBleed)
</script>

<style scoped lang="scss">
.app-content {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  background: var(--el-bg-color-page);
  padding: 16px;
}
@media (width >= 1024px) {
  .app-content {
    padding: 24px;
  }
}
.app-page {
  display: flex;
  flex-direction: column;
  min-height: 100%;
  min-width: 0;
  max-width: 1600px;
  margin: 0 auto;
  width: 100%;
}
.app-content--bleed {
  padding: 0;
}
.app-content--bleed .app-page {
  max-width: none;
}
</style>
