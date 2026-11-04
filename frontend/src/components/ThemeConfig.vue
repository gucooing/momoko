<!-- 外观抽屉：右侧滑出，承载 AppearancePanel（配色 / 深浅色 / 细节）。
     入口：顶栏主题菜单「更多外观设置」、登录/初始化页的外观按钮。 -->
<template>
  <Teleport to="body">
    <Transition name="theme-drawer">
      <div
        v-if="themeStore.themeConfigDrawerOpen"
        class="theme-drawer-overlay"
        @mousedown.self="close"
      >
        <aside class="theme-drawer" role="dialog" aria-modal="true" :aria-label="t('theme.title')">
          <header class="theme-drawer__head">
            <h3 class="theme-drawer__title">{{ t('theme.title') }}</h3>
            <AppIconButton
              icon="HOutline:XMarkIcon"
              :label="t('common.close')"
              :size="20"
              :box="40"
              @click="close"
            />
          </header>
          <div class="theme-drawer__body">
            <AppearancePanel variant="drawer" />
          </div>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'ThemeConfig' })

const themeStore = useThemeStore()
const { t } = useI18n()

const close = () => {
  themeStore.themeConfigDrawerOpen = false
}

const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && themeStore.themeConfigDrawerOpen) close()
}

watch(
  () => themeStore.themeConfigDrawerOpen,
  (open) => {
    document.documentElement.style.overflow = open ? 'hidden' : ''
  },
)

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  document.documentElement.style.overflow = ''
})
</script>

<style scoped lang="scss">
.theme-drawer-overlay {
  position: fixed;
  inset: 0;
  z-index: 2300;
  display: flex;
  justify-content: flex-end;
  background: color-mix(in srgb, var(--m-ink) 38%, transparent);
  backdrop-filter: blur(3px);
}
.theme-drawer {
  width: min(440px, 100vw);
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color-page);
  border-left: 1px solid var(--el-border-color-light);
  box-shadow: var(--app-shadow-lg);
}
.theme-drawer__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 14px 16px 14px 22px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.theme-drawer__title {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.theme-drawer__body {
  flex: 1;
  overflow: auto;
  padding: 20px 22px 28px;
}

.theme-drawer-enter-active,
.theme-drawer-leave-active {
  transition: opacity 0.2s ease;
}
.theme-drawer-enter-active .theme-drawer,
.theme-drawer-leave-active .theme-drawer {
  transition: transform 0.22s cubic-bezier(0.4, 0, 0.2, 1);
}
.theme-drawer-enter-from,
.theme-drawer-leave-to {
  opacity: 0;
}
.theme-drawer-enter-from .theme-drawer,
.theme-drawer-leave-to .theme-drawer {
  transform: translateX(24px);
}
</style>
