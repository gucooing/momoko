<!-- 主题预览模式的悬浮条（ThemePreviewPC）：临时套用一个主题，只在当前会话里生效、不写入账号；
     期间可以随便切换页面，退出或刷新后恢复原主题。 -->
<template>
  <Teleport to="body">
    <Transition name="pvbar">
      <div v-if="pv" class="pvbar" role="status" :class="{ 'is-mobile': isMobile }">
        <span class="pvbar__icon"><EyeIcon aria-hidden="true" /></span>
        <span class="pvbar__text">
          <b>{{ t('theme.preview.title', { name: pv.name }) }}</b>
          <small v-if="!isMobile">{{ t('theme.preview.hint') }}</small>
        </span>
        <span class="pvbar__seg" role="group" :aria-label="t('theme.mode')">
          <button
            v-for="s in schemes"
            :key="s"
            type="button"
            :class="{ on: pv.scheme === s }"
            :aria-pressed="pv.scheme === s"
            @click="themeStore.setPreviewScheme(s)"
          >
            {{ t(`theme.${s}`) }}
          </button>
        </span>
        <button type="button" class="pvbar__btn" @click="themeStore.endPreview()">{{ t('theme.preview.exit') }}</button>
        <button v-if="pv.accept" type="button" class="pvbar__btn is-primary" :disabled="busy" @click="accept">
          {{ pv.from === 'import' ? t('theme.import.importAndUse') : t('theme.preview.use') }}
        </button>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { EyeIcon } from '@heroicons/vue/24/outline'

defineOptions({ name: 'ThemePreviewBar' })

const { t } = useI18n()
const themeStore = useThemeStore()
const menuStore = useMenuStore()
const isMobile = computed(() => menuStore.isMobile)
const pv = computed(() => themeStore.preview)
const schemes = ['light', 'dark'] as const
const busy = ref(false)

const accept = async () => {
  const fn = pv.value?.accept
  if (!fn) return
  busy.value = true
  try {
    await fn()
    themeStore.endPreview()
  } finally {
    busy.value = false
  }
}
</script>

<style scoped lang="scss">
.pvbar {
  position: fixed;
  top: 14px;
  left: 50%;
  z-index: 3000;
  display: flex;
  align-items: center;
  gap: 12px;
  max-width: calc(100vw - 24px);
  padding: 8px;
  border-radius: 999px;
  background: rgba(15, 10, 25, 0.9);
  color: #ffffff;
  box-shadow: 0 18px 40px -16px rgba(0, 0, 0, 0.6);
  -webkit-backdrop-filter: blur(10px);
  backdrop-filter: blur(10px);
  transform: translateX(-50%);
}
.pvbar__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 36px;
  height: 36px;
  border-radius: 12px;
  background: var(--m-p);
  color: var(--m-pOn);
  svg {
    width: 18px;
    height: 18px;
  }
}
.pvbar__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
  b {
    font-size: 0.875rem;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  small {
    font-size: 0.75rem;
    opacity: 0.75;
    white-space: nowrap;
  }
}
.pvbar__seg {
  display: inline-flex;
  flex: none;
  padding: 3px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.12);
  button {
    min-height: 30px;
    padding: 0 11px;
    border: 0;
    border-radius: 999px;
    background: transparent;
    color: rgba(255, 255, 255, 0.75);
    font: inherit;
    font-size: 0.75rem;
    font-weight: 700;
    cursor: pointer;
    white-space: nowrap;
  }
  .on {
    background: #ffffff;
    color: #2a1630;
  }
}
.pvbar__btn {
  flex: none;
  min-height: 36px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.14);
  color: #ffffff;
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
  &.is-primary {
    background: var(--m-p);
    color: var(--m-pOn);
  }
  &:disabled {
    opacity: 0.6;
    cursor: progress;
  }
}
.pvbar.is-mobile {
  top: auto;
  bottom: calc(84px + env(safe-area-inset-bottom));
  left: 12px;
  right: 12px;
  flex-wrap: wrap;
  gap: 8px;
  border-radius: 24px;
  transform: none;
  .pvbar__text {
    flex: 1 1 0;
  }
}
.pvbar-enter-active,
.pvbar-leave-active {
  transition: opacity 0.2s, translate 0.2s;
}
.pvbar-enter-from,
.pvbar-leave-to {
  opacity: 0;
  translate: 0 -8px;
}
</style>
