<template>
  <teleport to="body">
    <transition name="fd-fade">
      <div v-if="modelValue" class="fd-overlay" @mousedown.self="onOverlay">
        <transition name="fd-zoom" appear>
          <div
            v-if="modelValue"
            class="file-module fd-panel"
            :class="{ 'is-dark': isDark }"
            :style="{ width: typeof width === 'number' ? `${width}px` : width }"
            role="dialog"
            aria-modal="true"
          >
            <header class="fd-header">
              <h3 class="fd-title">
                <slot name="title">{{ title }}</slot>
              </h3>
              <button type="button" class="fd-close" :aria-label="t('common.close')" @click="close">
                <el-icon><IconClose /></el-icon>
              </button>
            </header>

            <div class="fd-body">
              <slot />
            </div>

            <footer v-if="$slots.footer" class="fd-footer">
              <slot name="footer" />
            </footer>
          </div>
        </transition>
      </div>
    </transition>
  </teleport>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useThemeStore } from '@/stores/theme'
import { IconClose } from './icons'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title?: string
    width?: string | number
    closeOnOverlay?: boolean
  }>(),
  {
    width: 460,
    closeOnOverlay: true,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  close: []
}>()

const { t } = useI18n()
const themeStore = useThemeStore()
const isDark = computed(() => themeStore.isDarkTheme)

const close = () => {
  emit('update:modelValue', false)
  emit('close')
}

const onOverlay = () => {
  if (props.closeOnOverlay) close()
}

const onKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && props.modelValue) close()
}

watch(
  () => props.modelValue,
  (open) => {
    document.body.style.overflow = open ? 'hidden' : ''
  },
)

onMounted(() => document.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeydown)
  document.body.style.overflow = ''
})
</script>

<style scoped>
.fd-overlay {
  position: fixed;
  inset: 0;
  z-index: 2200;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  background: rgba(15, 23, 42, 0.42);
}

.fd-panel {
  max-width: calc(100vw - 2rem);
  max-height: calc(100vh - 2rem);
  display: flex;
  flex-direction: column;
  background: var(--fm-surface);
  border: 1px solid var(--fm-border);
  border-radius: var(--fm-radius);
  box-shadow: var(--fm-shadow);
  overflow: hidden;
}

.fd-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.875rem 1.125rem;
  border-bottom: 1px solid var(--fm-border);
}
.fd-title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--fm-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fd-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: none;
  border-radius: var(--fm-radius-sm);
  background: transparent;
  color: var(--fm-text-3);
  cursor: pointer;
  transition:
    background 0.15s,
    color 0.15s;
}
.fd-close:hover {
  background: var(--fm-hover);
  color: var(--fm-text);
}
.fd-close .el-icon {
  font-size: 18px;
}

.fd-body {
  padding: 1.125rem;
  overflow: auto;
}

.fd-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.625rem;
  padding: 0.875rem 1.125rem;
  border-top: 1px solid var(--fm-border);
}

.fd-fade-enter-active,
.fd-fade-leave-active {
  transition: opacity 0.18s ease;
}
.fd-fade-enter-from,
.fd-fade-leave-to {
  opacity: 0;
}
.fd-zoom-enter-active {
  transition:
    transform 0.18s ease,
    opacity 0.18s ease;
}
.fd-zoom-enter-from {
  transform: scale(0.96) translateY(8px);
  opacity: 0;
}
</style>
