<!-- 手机端 Docker 行操作底部面板：拖拽条 + 头部（倾斜贴纸 + 名称 + 等宽副标题 + 状态 pill）
     + 常用操作宫格（3–4 个）+ 其余操作列表（危险项红色）+ 取消。图标支持 i-lucide-* 与 HOutline:*。 -->
<template>
  <Teleport to="body">
    <Transition name="das">
      <div v-if="open" class="das" @keydown.esc="close">
        <button type="button" class="das__mask" :aria-label="t('docker.common.cancel')" @click="close" />
        <section class="das__panel" role="dialog" aria-modal="true" :aria-label="title">
          <div class="das__handle" aria-hidden="true" />
          <header class="das__head">
            <span class="das__sticker m-sticker" :class="`m-sticker--${tone}`">
              <UIcon v-if="isLucide(icon)" :name="icon" />
              <component :is="menuStore.iconComponents[icon]" v-else />
            </span>
            <span class="das__text">
              <span class="das__title">{{ title }}</span>
              <span v-if="subtitle" class="das__sub m-mono">{{ subtitle }}</span>
            </span>
            <StatusPill v-if="pill" :variant="pill.variant" :label="pill.label" />
          </header>

          <div v-if="quick.length" class="das__quick" :style="{ gridTemplateColumns: `repeat(${Math.min(4, quick.length)}, minmax(0, 1fr))` }">
            <button
              v-for="item in quick"
              :key="item.key"
              type="button"
              class="das__qa"
              :disabled="item.disabled"
              @click="pick(item)"
            >
              <UIcon v-if="item.icon && isLucide(item.icon)" :name="item.icon" />
              <component :is="menuStore.iconComponents[item.icon]" v-else-if="item.icon" />
              <span>{{ item.label }}</span>
            </button>
          </div>

          <div v-if="items.length" class="das__list">
            <button
              v-for="item in items"
              :key="item.key"
              type="button"
              class="das__li"
              :class="{ 'is-danger': item.danger }"
              :disabled="item.disabled"
              @click="pick(item)"
            >
              <UIcon v-if="item.icon && isLucide(item.icon)" :name="item.icon" />
              <component :is="menuStore.iconComponents[item.icon]" v-else-if="item.icon" />
              <span>{{ item.label }}</span>
            </button>
          </div>

          <button type="button" class="das__cancel" @click="close">{{ t('docker.common.cancel') }}</button>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'

type PillVariant = 'success' | 'warning' | 'error' | 'info' | 'neutral' | 'primary'

const props = withDefaults(
  defineProps<{
    title: string
    subtitle?: string
    icon?: string
    tone?: 'p' | 'a' | 'a2' | 'ok' | 'bad' | 'warn'
    pill?: { variant: PillVariant; label: string } | null
    quick?: ActionMenuItem[]
    items?: ActionMenuItem[]
  }>(),
  { icon: 'HOutline:CubeIcon', tone: 'a2', pill: null, quick: () => [], items: () => [] },
)
const emit = defineEmits<{ select: [key: string] }>()
const open = defineModel<boolean>({ default: false })
const { t } = useI18n()
const menuStore = useMenuStore()

const quick = computed(() => props.quick.filter((i) => !i.hidden))
const items = computed(() => props.items.filter((i) => !i.hidden))
const isLucide = (icon: string) => icon.startsWith('i-')

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
.das {
  position: fixed;
  inset: 0;
  z-index: 2100;
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
}
.das__mask {
  position: absolute;
  inset: 0;
  border: none;
  background: color-mix(in srgb, var(--m-ink) 45%, transparent);
  cursor: default;
}
.das__panel {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-height: 88vh;
  overflow-y: auto;
  scrollbar-width: none;
  padding: 10px 16px calc(20px + env(safe-area-inset-bottom));
  border-radius: 30px 30px 0 0;
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-lg);
}
.das__panel::-webkit-scrollbar {
  display: none;
}
.das__handle {
  align-self: center;
  width: 44px;
  height: 5px;
  border-radius: 999px;
  background: var(--el-border-color-light);
}
.das__head {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.das__sticker {
  flex: none;
  width: 48px;
  height: 48px;
  border-radius: 16px;
  transform: rotate(-4deg);
}
.das__sticker :deep(svg),
.das__sticker :deep(.iconify) {
  width: 24px;
  height: 24px;
}
.das__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.das__title {
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.das__sub {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.das__quick {
  display: grid;
  gap: 8px;
}
.das__qa {
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
  font-size: 0.8125rem;
  font-weight: 700;
  text-align: center;
  cursor: pointer;
}
.das__qa :deep(svg),
.das__qa :deep(.iconify) {
  width: 22px;
  height: 22px;
  color: var(--m-pInk);
}
.das__qa:disabled,
.das__li:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.das__list {
  display: flex;
  flex-direction: column;
}
.das__li {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 52px;
  padding: 0 6px;
  border: none;
  border-top: 1px solid var(--el-border-color-lighter);
  background: transparent;
  color: var(--el-text-color-primary);
  font-size: 0.9375rem;
  font-weight: 700;
  text-align: left;
  cursor: pointer;
}
.das__li:first-child {
  border-top: none;
}
.das__li :deep(svg),
.das__li :deep(.iconify) {
  flex: none;
  width: 20px;
  height: 20px;
  color: var(--el-text-color-secondary);
}
.das__li.is-danger,
.das__li.is-danger :deep(svg),
.das__li.is-danger :deep(.iconify) {
  color: var(--m-bad);
}
.das__cancel {
  min-height: 52px;
  border: none;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-size: 0.9375rem;
  font-weight: 700;
  cursor: pointer;
}
.das-enter-active,
.das-leave-active {
  transition: opacity 0.2s ease;
}
.das-enter-active .das__panel,
.das-leave-active .das__panel {
  transition: transform 0.24s cubic-bezier(0.2, 0.8, 0.2, 1);
}
.das-enter-from,
.das-leave-to {
  opacity: 0;
}
.das-enter-from .das__panel,
.das-leave-to .das__panel {
  transform: translateY(100%);
}
</style>
