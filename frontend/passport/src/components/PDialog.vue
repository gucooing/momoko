<!-- 通行证对话框：电脑上居中，手机上是底部面板。 -->
<template>
  <Teleport to="body">
    <Transition name="pd">
      <div v-if="modelValue" class="pd" role="dialog" aria-modal="true" :aria-label="title" @keydown.esc="close">
        <div class="pd__mask" @click="closeOnMask && close()" />
        <div class="pd__panel" :style="{ maxWidth: `${width}px` }">
          <header class="pd__head">
            <h2 class="pd__title">{{ title }}</h2>
            <button type="button" class="pd__x" :aria-label="t('common.close')" @click="close">
              <XMarkIcon aria-hidden="true" />
            </button>
          </header>
          <div class="pd__body">
            <slot />
          </div>
          <footer v-if="$slots.footer" class="pd__foot">
            <slot name="footer" />
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { XMarkIcon } from '@heroicons/vue/24/outline'
import { useI18n } from 'vue-i18n'

withDefaults(defineProps<{ modelValue: boolean; title?: string; width?: number; closeOnMask?: boolean }>(), {
  title: '',
  width: 480,
  closeOnMask: true,
})
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; close: [] }>()
const { t } = useI18n()

const close = () => {
  emit('update:modelValue', false)
  emit('close')
}
</script>

<style scoped>
.pd {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
}
.pd__mask {
  position: absolute;
  inset: 0;
  background: color-mix(in srgb, var(--m-ink) 45%, transparent);
}
.pd__panel {
  position: relative;
  width: 100%;
  max-height: calc(100vh - 32px);
  display: flex;
  flex-direction: column;
  border-radius: var(--app-radius-xl, 24px);
  background: var(--m-s);
  border: 1px solid var(--m-line);
  box-shadow: var(--app-shadow-lg);
}
.pd__head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 18px 20px 8px;
}
.pd__title {
  flex: 1;
  margin: 0;
  font-size: 18px;
  font-weight: 800;
}
.pd__x {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--m-ink2);
  cursor: pointer;
}
.pd__x:hover {
  background: var(--m-s2);
}
.pd__x svg {
  width: 20px;
  height: 20px;
}
.pd__body {
  padding: 8px 20px 18px;
  overflow-y: auto;
}
.pd__foot {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 12px 20px 18px;
  border-top: 1px solid var(--m-line);
}
.pd-enter-active,
.pd-leave-active {
  transition: opacity 0.18s;
}
.pd-enter-from,
.pd-leave-to {
  opacity: 0;
}
@media (max-width: 768px) {
  .pd {
    align-items: flex-end;
    padding: 0;
  }
  .pd__panel {
    max-width: none !important;
    max-height: 88vh;
    border-radius: var(--app-radius-xl, 24px) var(--app-radius-xl, 24px) 0 0;
  }
  .pd__foot > :deep(*) {
    flex: 1;
  }
}
</style>
