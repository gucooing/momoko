<!-- 主题配置抽屉（去 EP）：Teleport 令牌侧栏 + AppSwitch + 原生 color input。
     topMode 布局选项在新外壳下为空操作（任务书允许移除），保留 UI 以免丢功能入口。 -->
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
              :size="18"
              @click="close"
            />
          </header>

          <div class="theme-drawer__body">
            <!-- 主题模式 -->
            <div class="config-section">
              <div class="section-title">
                <component :is="ic('HOutline:SunIcon')" class="section-title__icon" />
                <span>{{ t('theme.mode') }}</span>
              </div>
              <div class="section-content">
                <div class="mode-chip-group">
                  <button
                    type="button"
                    class="mode-chip"
                    :class="{ active: themeStore.themeMode === 'light' }"
                    @click="themeStore.toggleThemeMode('light')"
                  >
                    <component :is="ic('HOutline:SunIcon')" class="mode-chip__icon" />
                    <span>{{ t('theme.light') }}</span>
                  </button>
                  <button
                    type="button"
                    class="mode-chip"
                    :class="{ active: themeStore.themeMode === 'dark' }"
                    @click="themeStore.toggleThemeMode('dark')"
                  >
                    <component :is="ic('HOutline:MoonIcon')" class="mode-chip__icon" />
                    <span>{{ t('theme.dark') }}</span>
                  </button>
                  <button
                    type="button"
                    class="mode-chip"
                    :class="{ active: themeStore.themeMode === 'auto' }"
                    @click="themeStore.toggleThemeMode('auto')"
                  >
                    <component :is="ic('HOutline:ComputerDesktopIcon')" class="mode-chip__icon" />
                    <span>{{ t('theme.auto') }}</span>
                  </button>
                </div>
              </div>
            </div>

            <!-- 布局模式（topMode 在新外壳下暂为空操作，保留入口） -->
            <div class="config-section">
              <div class="section-title">
                <component :is="ic('HOutline:Squares2X2Icon')" class="section-title__icon" />
                <span>{{ t('theme.layout') }}</span>
              </div>
              <div class="section-content">
                <div class="layout-preview-group">
                  <button
                    type="button"
                    class="layout-preview-item"
                    :class="{ active: themeStore.layout === 'leftMode' }"
                    @click="themeStore.toggleLayout('leftMode')"
                  >
                    <div class="layout-preview left-layout">
                      <div class="preview-sidebar" />
                      <div class="preview-content">
                        <div class="preview-header" />
                        <div class="preview-main" />
                      </div>
                    </div>
                    <div class="layout-label">{{ t('theme.leftLayout') }}</div>
                  </button>
                  <button
                    type="button"
                    class="layout-preview-item"
                    :class="{ active: themeStore.layout === 'topMode' }"
                    @click="themeStore.toggleLayout('topMode'); menuStore.isCollapse = false"
                  >
                    <div class="layout-preview top-layout">
                      <div class="preview-header" />
                      <div class="preview-main" />
                    </div>
                    <div class="layout-label">{{ t('theme.topLayout') }}</div>
                  </button>
                </div>
              </div>
            </div>

            <!-- 主题颜色 -->
            <div class="config-section">
              <div class="section-title">
                <component :is="ic('HOutline:SwatchIcon')" class="section-title__icon" />
                <span>{{ t('theme.color') }}</span>
              </div>
              <div class="section-content theme-color-content">
                <div class="color-chip-group">
                  <button
                    v-for="color in themeStore.primaryColorOptions"
                    :key="color.value"
                    type="button"
                    class="color-chip"
                    :class="{ active: themeStore.primaryColor === color.value }"
                    @click="themeStore.togglePrimaryColor(color.value)"
                  >
                    <span class="chip-dot" :style="{ backgroundColor: color.value }" />
                    <span class="chip-name">{{ t(color.labelKey) }}</span>
                  </button>
                </div>
                <div class="custom-color">
                  <span>{{ t('theme.custom') }}</span>
                  <input
                    class="custom-color__input"
                    type="color"
                    :value="normalizeHex(themeStore.primaryColor)"
                    @input="onCustomColor"
                  />
                </div>
              </div>
            </div>

            <!-- 界面元素 -->
            <div class="config-section">
              <div class="section-title">
                <component :is="ic('HOutline:EyeIcon')" class="section-title__icon" />
                <span>{{ t('theme.interfaceElements') }}</span>
              </div>
              <div class="section-content toggles-row">
                <div class="toggle-item">
                  <span>{{ t('theme.showLogo') }}</span>
                  <AppSwitch
                    :model-value="themeStore.showLogo"
                    @update:model-value="(v) => themeStore.toggleShowLogo(v)"
                  />
                </div>
                <div class="toggle-item">
                  <span>{{ t('theme.showTabs') }}</span>
                  <AppSwitch
                    :model-value="themeStore.showTabs"
                    @update:model-value="(v) => themeStore.toggleShowTabs(v)"
                  />
                </div>
              </div>
            </div>
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
const menuStore = useMenuStore()
const { t } = useI18n()
const ic = (name: string) => menuStore.iconComponents[name]

const close = () => {
  themeStore.themeConfigDrawerOpen = false
}

const normalizeHex = (value: string) => {
  if (!value) return '#14B8A6'
  if (/^#[0-9a-fA-F]{6}$/.test(value)) return value
  // rgba(...) 等非 hex：回退主色默认
  return '#14B8A6'
}

const onCustomColor = (event: Event) => {
  const value = (event.target as HTMLInputElement).value
  if (value) themeStore.togglePrimaryColor(value)
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
  background: color-mix(in srgb, #0b1220 40%, transparent);
  backdrop-filter: blur(2px);
}
.theme-drawer {
  width: min(360px, 100vw);
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color);
  border-left: 1px solid var(--el-border-color-light);
  box-shadow: var(--app-shadow-lg);
}
.theme-drawer__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 14px 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.theme-drawer__title {
  margin: 0;
  font-size: 0.9375rem;
  font-weight: 650;
  color: var(--el-text-color-primary);
}
.theme-drawer__body {
  flex: 1;
  overflow: auto;
  padding: 16px;
}

.config-section {
  margin-bottom: 20px;
  &:last-child {
    margin-bottom: 0;
  }
}
.section-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 12px;
}
.section-title__icon {
  width: 16px;
  height: 16px;
  color: var(--el-color-primary);
}
.section-content {
  padding-left: 4px;
}

.mode-chip-group {
  display: flex;
  gap: 8px;
}
.mode-chip {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 8px;
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
  font-size: 12px;
  color: var(--el-text-color-regular);
  cursor: pointer;
  transition:
    border-color 0.15s,
    color 0.15s,
    background 0.15s;
}
.mode-chip__icon {
  width: 14px;
  height: 14px;
}
.mode-chip:hover {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}
.mode-chip.active {
  background: color-mix(in srgb, var(--el-color-primary) 10%, transparent);
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}

.layout-preview-group {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
}
.layout-preview-item {
  cursor: pointer;
  padding: 8px;
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
  transition: border-color 0.15s, background 0.15s;
}
.layout-preview {
  width: 100%;
  aspect-ratio: 4/3;
  border-radius: 6px;
  background: var(--el-fill-color-light);
  padding: 6px;
  margin-bottom: 8px;
}
.layout-preview.left-layout {
  display: flex;
  gap: 4px;
  .preview-sidebar {
    width: 28%;
    background: var(--el-color-primary);
    border-radius: 4px;
  }
  .preview-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .preview-header {
    height: 18%;
    background: var(--el-bg-color-overlay);
    border-radius: 4px;
    border: 1px solid var(--el-border-color-lighter);
  }
  .preview-main {
    flex: 1;
    background: var(--el-bg-color-overlay);
    border-radius: 4px;
    border: 1px solid var(--el-border-color-lighter);
  }
}
.layout-preview.top-layout {
  display: flex;
  flex-direction: column;
  gap: 4px;
  .preview-header {
    height: 22%;
    background: var(--el-color-primary);
    border-radius: 4px;
  }
  .preview-main {
    flex: 1;
    background: var(--el-bg-color-overlay);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 4px;
  }
}
.layout-label {
  text-align: center;
  font-size: 12px;
  color: var(--el-text-color-regular);
}
.layout-preview-item:hover,
.layout-preview-item.active {
  border-color: var(--el-color-primary);
  .layout-label {
    color: var(--el-color-primary);
  }
}
.layout-preview-item.active {
  background: color-mix(in srgb, var(--el-color-primary) 8%, transparent);
}

.theme-color-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.color-chip-group {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.color-chip {
  min-width: 90px;
  padding: 6px 10px;
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color-overlay);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-size: 12px;
  color: var(--el-text-color-regular);
  cursor: pointer;
}
.chip-dot {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  border: 1px solid var(--el-border-color-lighter);
}
.color-chip:hover,
.color-chip.active {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}
.color-chip.active {
  background: color-mix(in srgb, var(--el-color-primary) 10%, transparent);
}
.custom-color {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  color: var(--el-text-color-regular);
}
.custom-color__input {
  width: 36px;
  height: 28px;
  padding: 0;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  background: transparent;
  cursor: pointer;
}

.toggles-row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.toggle-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
  color: var(--el-text-color-regular);
  padding: 8px 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  background: var(--el-bg-color-overlay);
}

.theme-drawer-enter-active,
.theme-drawer-leave-active {
  transition: opacity 0.18s ease;
}
.theme-drawer-enter-active .theme-drawer,
.theme-drawer-leave-active .theme-drawer {
  transition: transform 0.18s cubic-bezier(0.4, 0, 0.2, 1);
}
.theme-drawer-enter-from,
.theme-drawer-leave-to {
  opacity: 0;
}
.theme-drawer-enter-from .theme-drawer,
.theme-drawer-leave-to .theme-drawer {
  transform: translateX(12px);
}
</style>
