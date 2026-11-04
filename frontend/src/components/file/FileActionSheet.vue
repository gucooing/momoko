<!-- 手机端文件操作底部面板：头部（贴纸图标 + 名称 + 元信息 + 路径）+ 常用操作四宫格 + 其余操作列表 + 取消。 -->
<template>
  <Teleport to="body">
    <Transition name="fas">
      <div v-if="open" class="fas" @keydown.esc="close">
        <button type="button" class="fas__mask" :aria-label="t('system.common.cancel')" @click="close" />
        <section class="fas__panel" role="dialog" :aria-label="title">
          <div class="fas__handle" />
          <header class="fas__head">
            <span class="fas__sticker m-sticker" :class="`m-sticker--${tone}`">
              <UIcon :name="icon" />
            </span>
            <span class="fas__text">
              <span class="fas__title">{{ title }}</span>
              <span v-if="meta" class="fas__meta m-num">{{ meta }}</span>
            </span>
          </header>
          <div v-if="path" class="fas__path m-mono">{{ path }}</div>

          <div v-if="quick.length" class="fas__quick" :style="{ gridTemplateColumns: `repeat(${quick.length}, minmax(0, 1fr))` }">
            <button
              v-for="item in quick"
              :key="item.key"
              type="button"
              class="fas__qa"
              :disabled="item.disabled"
              @click="pick(item)"
            >
              <UIcon :name="item.icon || 'i-lucide-circle'" />
              <span>{{ item.label }}</span>
            </button>
          </div>

          <div v-if="items.length" class="fas__list">
            <button
              v-for="item in items"
              :key="item.key"
              type="button"
              class="fas__li"
              :class="{ 'is-danger': item.danger }"
              :disabled="item.disabled"
              @click="pick(item)"
            >
              <UIcon :name="item.icon || 'i-lucide-circle'" />
              <span>{{ item.label }}</span>
            </button>
          </div>

          <button type="button" class="fas__cancel" @click="close">{{ t('system.common.cancel') }}</button>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'

withDefaults(
  defineProps<{
    title: string
    meta?: string
    path?: string
    icon?: string
    tone?: string
    quick?: ActionMenuItem[]
    items?: ActionMenuItem[]
  }>(),
  { icon: 'i-lucide-file', tone: 'a2', quick: () => [], items: () => [] },
)
const emit = defineEmits<{ select: [key: string] }>()
const open = defineModel<boolean>({ default: false })
const { t } = useI18n()

const close = () => {
  open.value = false
}
const pick = (item: ActionMenuItem) => {
  if (item.disabled) return
  open.value = false
  emit('select', item.key)
}
</script>

<style scoped lang="scss">
.fas {
  position: fixed;
  inset: 0;
  /* 高于手机底部导航（2050） */
  z-index: 2100;
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
}
.fas__mask {
  position: absolute;
  inset: 0;
  border: none;
  background: color-mix(in srgb, var(--m-ink) 42%, transparent);
  cursor: default;
}
.fas__panel {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 86vh;
  overflow-y: auto;
  padding: 10px 16px calc(16px + env(safe-area-inset-bottom));
  border-radius: var(--app-radius-xl) var(--app-radius-xl) 0 0;
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-lg);
}
.fas__handle {
  width: 40px;
  height: 5px;
  margin: 0 auto 2px;
  border-radius: 999px;
  background: var(--el-border-color-light);
}
.fas__head {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.fas__sticker {
  width: 44px;
  height: 44px;
  transform: rotate(-4deg);
}
.fas__sticker.m-sticker--neutral {
  background: var(--m-s2);
  color: var(--m-ink2);
}
.fas__sticker :deep(svg),
.fas__sticker :deep(.iconify) {
  width: 22px;
  height: 22px;
}
.fas__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.fas__title {
  font-family: var(--app-font-display);
  font-size: 1.0625rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fas__meta {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.fas__path {
  padding: 8px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  word-break: break-all;
}
.fas__quick {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
}
.fas__qa {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 72px;
  padding: 8px 4px;
  border: none;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.fas__qa :deep(svg),
.fas__qa :deep(.iconify) {
  width: 22px;
  height: 22px;
  color: var(--m-pInk);
}
.fas__qa:disabled {
  opacity: 0.4;
}
.fas__list {
  display: flex;
  flex-direction: column;
  border-radius: var(--app-radius);
  border: 1px solid var(--el-border-color-lighter);
  overflow: hidden;
}
.fas__li {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 48px;
  padding: 0 16px;
  border: none;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: transparent;
  color: var(--el-text-color-primary);
  font-size: 0.9375rem;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
}
.fas__li:last-child {
  border-bottom: none;
}
.fas__li:active {
  background: var(--el-fill-color-light);
}
.fas__li :deep(svg),
.fas__li :deep(.iconify) {
  width: 19px;
  height: 19px;
  color: var(--el-text-color-secondary);
}
.fas__li.is-danger,
.fas__li.is-danger :deep(svg),
.fas__li.is-danger :deep(.iconify) {
  color: var(--m-bad);
}
.fas__li:disabled {
  opacity: 0.4;
}
.fas__cancel {
  min-height: 48px;
  border: none;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-size: 0.9375rem;
  font-weight: 700;
  cursor: pointer;
}
.fas-enter-active,
.fas-leave-active {
  transition: opacity 0.2s ease;
}
.fas-enter-active .fas__panel,
.fas-leave-active .fas__panel {
  transition: transform 0.24s cubic-bezier(0.2, 0.8, 0.2, 1);
}
.fas-enter-from,
.fas-leave-to {
  opacity: 0;
}
.fas-enter-from .fas__panel,
.fas-leave-to .fas__panel {
  transform: translateY(100%);
}
</style>
