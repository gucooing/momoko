<!-- Agent 页面的紧凑弹窗：宽约 560、圆角 16、标题 15。内容放默认插槽，底部按钮放 #footer（没有就不显示底栏）。
     手机上贴底铺满宽度。浮层（AgSelect、AppDropdown）的层级在它之上。 -->
<template>
  <Teleport to="body">
    <Transition name="ag-dlg">
      <div v-if="modelValue" class="ag ag-dlg" @mousedown.self="onOverlay">
        <div
          class="ag-dlg__card"
          role="dialog"
          aria-modal="true"
          :aria-label="title"
          :style="{ width: typeof width === 'number' ? `${width}px` : width }"
        >
          <header class="ag-dlg__head">
            <slot name="icon" />
            <h2 class="ag-dlg__title">{{ title }}</h2>
            <button type="button" class="ag-ib" :aria-label="t('common.close')" @click="close">
              <UIcon name="i-lucide-x" />
            </button>
          </header>
          <div class="ag-dlg__body" :class="{ 'ag-dlg__body--flush': flush }">
            <slot />
          </div>
          <footer v-if="$slots.footer" class="ag-dlg__foot">
            <slot name="footer" :close="close" />
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title?: string
    width?: number | string
    /** 点遮罩关闭；表单弹窗默认不关，避免误触丢内容 */
    closeOnOverlay?: boolean
    /** 内容区不加内边距（自己排版） */
    flush?: boolean
  }>(),
  { title: '', width: 560, closeOnOverlay: false, flush: false },
)
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; close: [] }>()
const { t } = useI18n()

const close = () => {
  emit('update:modelValue', false)
  emit('close')
}
const onOverlay = () => {
  if (props.closeOnOverlay) close()
}
const onKey = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.modelValue) close()
}
onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
</script>

<style lang="scss">
.ag-dlg {
  position: fixed;
  inset: 0;
  z-index: 2200;
  display: flex;
  overflow-y: auto;
  padding: 16px;
  background: rgba(20, 10, 30, 0.45);
}
.ag-dlg__card {
  max-width: 100%;
  margin: auto;
  display: flex;
  flex-direction: column;
  background: var(--m-s);
  border: 1px solid var(--m-line);
  border-radius: 16px;
  box-shadow: 0 30px 70px -30px rgba(20, 10, 30, 0.45);
}
.ag-dlg__head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 12px 2px 18px;
}
.ag-dlg__title {
  flex: 1;
  min-width: 0;
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ag-dlg__body {
  padding: 8px 18px 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.ag-dlg__body--flush {
  padding: 0;
  gap: 0;
}
.ag-dlg__foot {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  padding: 10px 18px 14px;
  border-top: 1px solid var(--m-line);
}
.ag-dlg-enter-active,
.ag-dlg-leave-active {
  transition: opacity 0.15s;
}
.ag-dlg-enter-from,
.ag-dlg-leave-to {
  opacity: 0;
}
@media (width < 992px) {
  .ag-dlg {
    padding: 0;
    align-items: flex-end;
  }
  .ag-dlg__card {
    width: 100% !important;
    margin: auto 0 0;
    max-height: 92dvh;
    border-radius: 22px 22px 0 0;
    border-width: 0;
    overflow-y: auto;
  }
  .ag-dlg__foot {
    position: sticky;
    bottom: 0;
    background: var(--m-s);
    padding-bottom: calc(14px + env(safe-area-inset-bottom));
  }
}
</style>
