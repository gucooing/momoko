<!-- 面板：白卡 + hairline + lg 圆角 + 极轻阴影。静态表面不重投影（01 §5）。 -->
<template>
  <section class="app-panel" :class="{ 'app-panel--flush': props.padded === false }">
    <header v-if="title || $slots.header || $slots.actions" class="app-panel__head">
      <slot name="header">
        <div class="app-panel__title">
          <component
            :is="menuStore.iconComponents[titleIcon]"
            v-if="titleIcon"
            class="app-panel__title-icon"
          />
          <h2>{{ title }}</h2>
          <span v-if="caption" class="app-panel__caption">{{ caption }}</span>
        </div>
      </slot>
      <div v-if="$slots.actions" class="app-panel__actions"><slot name="actions" /></div>
    </header>
    <div class="app-panel__body" :class="bodyClass"><slot /></div>
    <footer v-if="$slots.footer" class="app-panel__foot"><slot name="footer" /></footer>
  </section>
</template>

<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    title?: string
    caption?: string
    titleIcon?: string
    padded?: boolean
    bodyClass?: string
  }>(),
  { padded: true },
)
const menuStore = useMenuStore()
</script>

<style scoped lang="scss">
.app-panel {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  box-shadow: var(--app-shadow-card);
  overflow: hidden;
}
.app-panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 15px 20px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  min-width: 0;
}
@media (width < 640px) {
  .app-panel__head {
    flex-wrap: wrap;
    align-items: flex-start;
  }
  .app-panel__actions {
    max-width: 100%;
    flex-wrap: wrap;
    justify-content: flex-start;
  }
}
.app-panel__title {
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
  min-width: 0;
}
.app-panel__title h2 {
  font-size: 1rem;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--el-text-color-primary);
}
.app-panel__title-icon {
  width: 18px;
  height: 18px;
  align-self: center;
  color: var(--el-text-color-secondary);
}
.app-panel__caption {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
.app-panel__actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 1;
  min-width: 0;
}
.app-panel__body {
  padding: 20px;
}
.app-panel--flush .app-panel__body {
  padding: 0;
}
.app-panel__foot {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 13px 20px;
  border-top: 1px solid var(--el-border-color-lighter);
}
</style>
