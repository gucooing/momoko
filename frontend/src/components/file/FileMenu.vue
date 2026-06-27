<template>
  <span ref="triggerRef" class="fm-menu-trigger" @click="toggle">
    <slot />
  </span>

  <teleport to="body">
    <transition name="fm-menu-fade">
      <div
        v-if="open"
        ref="menuRef"
        class="file-module fm-menu"
        :class="{ 'is-dark': isDark }"
        :style="menuStyle"
      >
        <template v-for="item in items" :key="item.key">
          <div v-if="item.divider" class="fm-menu-divider"></div>
          <button
            v-else
            type="button"
            class="fm-menu-item"
            :class="{ 'is-danger': item.danger, 'is-disabled': item.disabled }"
            :disabled="item.disabled"
            @click="select(item)"
          >
            <el-icon v-if="item.icon" class="fm-menu-icon"><component :is="item.icon" /></el-icon>
            <span>{{ item.label }}</span>
          </button>
        </template>
      </div>
    </transition>
  </teleport>
</template>

<script setup lang="ts">
import { onClickOutside } from '@vueuse/core'
import type { Component, CSSProperties } from 'vue'
import { useThemeStore } from '@/stores/theme'

export interface FileMenuItem {
  key: string
  label: string
  icon?: Component
  danger?: boolean
  disabled?: boolean
  divider?: boolean
}

const props = withDefaults(
  defineProps<{
    items: FileMenuItem[]
    dark?: boolean
    align?: 'left' | 'right'
  }>(),
  {
    align: 'right',
  },
)

// 默认跟随 app 浅/暗；编辑器内会显式传入其自身主题。
const themeStore = useThemeStore()
const isDark = computed(() => props.dark ?? themeStore.isDarkTheme)

const emit = defineEmits<{ select: [key: string] }>()

const triggerRef = ref<HTMLElement | null>(null)
const menuRef = ref<HTMLElement | null>(null)
const open = ref(false)
const menuStyle = ref<CSSProperties>({})

const MENU_WIDTH = 184

const computePosition = () => {
  const el = triggerRef.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  const menuHeight = menuRef.value?.offsetHeight ?? 0
  // 下方空间不足时向上翻转，避免菜单落到视口外不可点击（尤其末尾行）
  let top = rect.bottom + 4
  if (menuHeight && top + menuHeight > window.innerHeight - 8) {
    top = Math.max(8, rect.top - menuHeight - 4)
  }
  let left = props.align === 'right' ? rect.right - MENU_WIDTH : rect.left
  left = Math.max(8, Math.min(left, window.innerWidth - MENU_WIDTH - 8))
  menuStyle.value = {
    position: 'fixed',
    top: `${top}px`,
    left: `${left}px`,
    minWidth: `${MENU_WIDTH}px`,
  }
}

const close = () => {
  open.value = false
}

const toggle = async () => {
  if (open.value) {
    close()
    return
  }
  computePosition()
  open.value = true
  await nextTick()
  computePosition()
}

const select = (item: FileMenuItem) => {
  if (item.disabled) return
  close()
  emit('select', item.key)
}

onClickOutside(menuRef, () => close(), { ignore: [triggerRef] })

const onViewportChange = () => {
  if (open.value) close()
}

onMounted(() => {
  window.addEventListener('scroll', onViewportChange, true)
  window.addEventListener('resize', onViewportChange)
})
onBeforeUnmount(() => {
  window.removeEventListener('scroll', onViewportChange, true)
  window.removeEventListener('resize', onViewportChange)
})
</script>

<style scoped>
.fm-menu-trigger {
  display: inline-flex;
}
.fm-menu {
  z-index: 2300;
  padding: 4px;
  background: var(--fm-surface);
  border: 1px solid var(--fm-border);
  border-radius: var(--fm-radius-sm);
  box-shadow: var(--fm-shadow);
}
.fm-menu-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  width: 100%;
  padding: 0.45rem 0.6rem;
  border: none;
  border-radius: var(--fm-radius-sm);
  background: transparent;
  color: var(--fm-text-2);
  font-size: 13px;
  text-align: left;
  white-space: nowrap;
  cursor: pointer;
  transition:
    background 0.15s,
    color 0.15s;
}
.fm-menu-item:hover:not(.is-disabled) {
  background: var(--fm-hover);
  color: var(--fm-text);
}
.fm-menu-item.is-danger {
  color: var(--fm-danger);
}
.fm-menu-item.is-danger:hover:not(.is-disabled) {
  background: var(--fm-danger-soft);
}
.fm-menu-item.is-disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.fm-menu-icon {
  font-size: 16px;
}
.fm-menu-divider {
  height: 1px;
  margin: 4px 2px;
  background: var(--fm-border);
}

.fm-menu-fade-enter-active,
.fm-menu-fade-leave-active {
  transition:
    opacity 0.14s ease,
    transform 0.14s ease;
}
.fm-menu-fade-enter-from,
.fm-menu-fade-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
