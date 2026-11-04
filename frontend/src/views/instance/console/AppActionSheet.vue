<!-- 手机端应用操作面板：拖动条 + 常用操作三宫格 + 其余操作列表（可带一行说明，危险项红色）+ 取消。 -->
<template>
  <Teleport to="body">
    <Transition name="aas">
      <div v-if="open" class="aas" @keydown.esc="close">
        <button type="button" class="aas__mask" :aria-label="t('system.common.cancel')" @click="close" />
        <section class="aas__panel" role="dialog" :aria-label="title">
          <div class="aas__handle" />
          <div v-if="quick.length" class="aas__quick">
            <button
              v-for="item in quick"
              :key="item.key"
              type="button"
              class="aas__qa"
              :disabled="item.disabled"
              @click="pick(item)"
            >
              <UIcon :name="item.icon" />
              <span>{{ item.label }}</span>
            </button>
          </div>
          <div v-if="items.length" class="aas__list">
            <button
              v-for="item in items"
              :key="item.key"
              type="button"
              class="aas__li"
              :class="{ 'is-danger': item.danger }"
              :disabled="item.disabled"
              @click="pick(item)"
            >
              <UIcon :name="item.icon" class="aas__li-ico" />
              <span class="aas__li-text">
                <span>{{ item.label }}</span>
                <span v-if="item.description" class="aas__li-desc">{{ item.description }}</span>
              </span>
            </button>
          </div>
          <button type="button" class="aas__cancel" @click="close">{{ t('system.common.cancel') }}</button>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

export interface AppSheetItem {
  key: string
  label: string
  /** i-lucide-* 图标 */
  icon: string
  description?: string
  danger?: boolean
  disabled?: boolean
}

withDefaults(defineProps<{ title: string; quick?: AppSheetItem[]; items?: AppSheetItem[] }>(), {
  quick: () => [],
  items: () => [],
})
const emit = defineEmits<{ select: [key: string] }>()
const open = defineModel<boolean>({ default: false })
const { t } = useI18n()

const close = () => {
  open.value = false
}
const pick = (item: AppSheetItem) => {
  if (item.disabled) return
  open.value = false
  emit('select', item.key)
}
</script>

<style scoped lang="scss">
.aas {
  position: fixed;
  inset: 0;
  /* 高于手机底部导航（2050） */
  z-index: 2100;
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
}
.aas__mask {
  position: absolute;
  inset: 0;
  border: none;
  background: color-mix(in srgb, var(--m-ink) 50%, transparent);
  cursor: default;
}
.aas__panel {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 14px;
  max-height: 86vh;
  overflow-y: auto;
  padding: 10px 16px calc(18px + env(safe-area-inset-bottom));
  border-radius: 30px 30px 0 0;
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-lg);
  scrollbar-width: none;
}
.aas__panel::-webkit-scrollbar {
  display: none;
}
.aas__handle {
  align-self: center;
  width: 44px;
  height: 5px;
  border-radius: 999px;
  background: var(--el-border-color-light);
}
.aas__quick {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}
.aas__qa {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 76px;
  padding: 8px 4px;
  border: none;
  border-radius: 18px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.aas__qa :deep(.iconify) {
  width: 22px;
  height: 22px;
  color: var(--m-pInk);
}
.aas__qa:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.aas__list {
  display: flex;
  flex-direction: column;
}
.aas__li {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 52px;
  padding: 6px 12px;
  border: none;
  border-radius: 14px;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.9375rem;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
}
.aas__li:active:not(:disabled) {
  background: var(--el-fill-color-light);
}
.aas__li:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.aas__li-ico {
  width: 20px;
  height: 20px;
  flex: none;
  color: var(--el-text-color-secondary);
}
.aas__li.is-danger,
.aas__li.is-danger .aas__li-ico {
  color: var(--m-bad);
}
.aas__li-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.aas__li-desc {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.aas__cancel {
  min-height: 52px;
  border: none;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.9375rem;
  font-weight: 700;
  cursor: pointer;
}
.aas-enter-active,
.aas-leave-active {
  transition: opacity 0.2s ease;
}
.aas-enter-active .aas__panel,
.aas-leave-active .aas__panel {
  transition: transform 0.25s cubic-bezier(0.2, 0.8, 0.2, 1);
}
.aas-enter-from,
.aas-leave-to {
  opacity: 0;
}
.aas-enter-from .aas__panel,
.aas-leave-to .aas__panel {
  transform: translateY(40px);
}
</style>
