<!-- 外观页的弹窗外壳（导入 / 导出主题包、另存为）：PC 居中卡片，手机底部面板（顶部拖动条）。
     头部标题 + 关闭，中间内容区内部滚动，底部操作栏固定。遮罩点击、Esc 关闭。 -->
<template>
  <Teleport to="body">
    <Transition :name="isMobile ? 'tmodal-sheet' : 'tmodal'">
      <div v-if="modelValue" class="tmodal" :class="{ 'is-sheet': isMobile }" @mousedown.self="close">
        <section class="tmodal__panel" role="dialog" aria-modal="true" :aria-label="title" :style="{ '--tmodal-w': `${width}px` }">
          <span v-if="isMobile" class="tmodal__grip" aria-hidden="true" />
          <header class="tmodal__head">
            <span class="tmodal__title">
              <b>{{ title }}</b>
              <small v-if="subtitle">{{ subtitle }}</small>
            </span>
            <button type="button" class="tmodal__x" :aria-label="t('common.close')" @click="close">
              <XMarkIcon aria-hidden="true" />
            </button>
          </header>
          <div class="tmodal__body"><slot /></div>
          <footer v-if="$slots.footer" class="tmodal__foot"><slot name="footer" /></footer>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { XMarkIcon } from '@heroicons/vue/24/outline'

const props = withDefaults(defineProps<{ modelValue: boolean; title: string; subtitle?: string; width?: number }>(), {
  subtitle: '',
  width: 560,
})
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; close: [] }>()
const { t } = useI18n()
const menuStore = useMenuStore()
const isMobile = computed(() => menuStore.isMobile)

const close = () => {
  emit('update:modelValue', false)
  emit('close')
}
const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.modelValue) close()
}
watch(
  () => props.modelValue,
  (open) => {
    document.documentElement.style.overflow = open ? 'hidden' : ''
  },
)
onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  if (props.modelValue) document.documentElement.style.overflow = ''
})
</script>

<style scoped lang="scss">
.tmodal {
  position: fixed;
  inset: 0;
  z-index: 2100;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: rgba(20, 10, 30, 0.45);
}
.tmodal__panel {
  display: flex;
  flex-direction: column;
  width: var(--tmodal-w);
  max-width: 100%;
  max-height: 100%;
  overflow: hidden;
  border-radius: var(--app-radius-xl);
  background: var(--m-s);
  color: var(--m-ink);
  box-shadow: 0 30px 70px -30px rgba(20, 10, 30, 0.6);
}
.tmodal__head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px 16px 14px 24px;
}
.tmodal__title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  b {
    font-family: var(--app-font-display);
    font-size: 1.25rem;
    font-weight: 800;
  }
  small {
    font-size: 0.8125rem;
    color: var(--m-ink2);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
.tmodal__x {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  flex: none;
  border: 0;
  border-radius: 14px;
  background: transparent;
  color: var(--m-ink2);
  cursor: pointer;
  svg {
    width: 20px;
    height: 20px;
  }
  &:hover {
    background: var(--m-s2);
    color: var(--m-ink);
  }
}
.tmodal__body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 0 24px 20px;
  scrollbar-width: none;
  &::-webkit-scrollbar {
    display: none;
  }
}
.tmodal__foot {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 24px 20px;
  border-top: 1px solid var(--m-line);
}

/* 手机：底部面板 */
.tmodal.is-sheet {
  align-items: flex-end;
  padding: 0;
  .tmodal__panel {
    width: 100%;
    max-height: 92vh;
    border-radius: 28px 28px 0 0;
  }
  .tmodal__grip {
    align-self: center;
    width: 40px;
    height: 5px;
    margin: 10px 0 2px;
    border-radius: 999px;
    background: var(--m-line);
  }
  .tmodal__head {
    padding: 4px 8px 8px 20px;
  }
  .tmodal__title b {
    font-size: 1.125rem;
  }
  .tmodal__body {
    padding: 0 20px 16px;
  }
  .tmodal__foot {
    padding: 12px 16px calc(18px + env(safe-area-inset-bottom));
  }
}

.tmodal-enter-active,
.tmodal-leave-active {
  transition: opacity 0.18s;
  .tmodal__panel {
    transition: transform 0.2s;
  }
}
.tmodal-enter-from,
.tmodal-leave-to {
  opacity: 0;
  .tmodal__panel {
    transform: scale(0.97);
  }
}
.tmodal-sheet-enter-active,
.tmodal-sheet-leave-active {
  transition: opacity 0.2s;
  .tmodal__panel {
    transition: transform 0.25s cubic-bezier(0.2, 0.8, 0.2, 1);
  }
}
.tmodal-sheet-enter-from,
.tmodal-sheet-leave-to {
  opacity: 0;
  .tmodal__panel {
    transform: translateY(40px);
  }
}
</style>
