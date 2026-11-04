<!-- 外观实时预览（ThemePC 右栏 / ThemeMobile 顶部）：用当前配色画一小块工作台。
     variant='card'：完整预览卡（问候卡 + 两个圆环指标 + 应用行 + 按钮与状态）；'hero'：只有主色问候卡（手机外观页顶部）。 -->
<template>
  <aside v-if="variant === 'card'" class="pv" :aria-label="t('theme.preview')">
    <div class="pv__head">
      <h2 class="pv__title">{{ t('theme.preview') }}</h2>
      <span class="pv__chip">{{ paletteName }} · {{ modeName }}</span>
    </div>
    <div class="pv__box">
      <div class="pv-hero">
        <span class="pv-hero__blob" aria-hidden="true" />
        <span class="pv-hero__ring" aria-hidden="true" />
        <div class="pv-hero__body">
          <b>{{ greetingLine }}</b>
          <small>{{ t('theme.previewSub') }}</small>
          <span class="pv-hero__btns">
            <span class="pv-btn pv-btn--on">{{ t('theme.previewAction') }}</span>
            <span class="pv-btn pv-btn--soft">{{ t('dashboard.home.quickTerminal') }}</span>
          </span>
        </div>
      </div>
      <div class="pv__stats">
        <div class="pv-stat">
          <svg viewBox="0 0 64 64" aria-hidden="true">
            <circle cx="32" cy="32" r="26" class="pv-ring__track" />
            <circle cx="32" cy="32" r="26" class="pv-ring__bar" stroke-dasharray="27 163.4" transform="rotate(-90 32 32)" style="stroke: var(--m-p)" />
          </svg>
          <div><small>CPU</small><b class="m-num">16.5%</b></div>
        </div>
        <div class="pv-stat">
          <svg viewBox="0 0 64 64" aria-hidden="true">
            <circle cx="32" cy="32" r="26" class="pv-ring__track" />
            <circle cx="32" cy="32" r="26" class="pv-ring__bar" stroke-dasharray="75.2 163.4" transform="rotate(-90 32 32)" style="stroke: var(--m-a2)" />
          </svg>
          <div><small>{{ t('dashboard.home.statMemory') }}</small><b class="m-num">46.0%</b></div>
        </div>
      </div>
      <div class="pv-app">
        <span class="pv-app__av m-sticker m-sticker--p" aria-hidden="true">B</span>
        <span class="pv-app__text"><b>blog-api</b><small>{{ t('theme.previewApp') }}</small></span>
        <StatusPill variant="success" :label="t('theme.previewRunning')" />
      </div>
      <div class="pv__row">
        <span class="pv-btn pv-btn--p">{{ t('theme.previewStart') }}</span>
        <span class="pv-btn pv-btn--s">{{ t('theme.previewSecondary') }}</span>
        <StatusPill variant="warning" :label="t('theme.previewMaintain')" />
        <StatusPill variant="error" :label="t('theme.previewStopped')" />
      </div>
    </div>
  </aside>

  <div v-else class="pv-hero pv-hero--top">
    <span class="pv-hero__blob" aria-hidden="true" />
    <span class="pv-hero__ring" aria-hidden="true" />
    <div class="pv-hero__body">
      <small class="pv-hero__cur">{{ t('theme.currentLabel', { name: paletteName, mode: modeName }) }}</small>
      <b class="pv-hero__big">{{ greetingLine }}</b>
      <span class="pv-hero__btns">
        <span class="pv-btn pv-btn--on">{{ t('theme.previewAction') }}</span>
        <span class="pv-btn pv-btn--soft">{{ t('dashboard.home.quickTerminalShort') }}</span>
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

withDefaults(defineProps<{ variant?: 'card' | 'hero' }>(), { variant: 'card' })

const { t } = useI18n()
const themeStore = useThemeStore()
const userStore = useUserStore()

const paletteName = computed(() => t(themeStore.currentPalette.labelKey))
const modeName = computed(() => t(`theme.${themeStore.themeMode}`))
const greetingLine = computed(() => {
  const h = new Date().getHours()
  const key =
    h < 6 ? 'lateNight' : h < 9 ? 'morning' : h < 12 ? 'forenoon' : h < 14 ? 'noon' : h < 18 ? 'afternoon' : 'evening'
  const name = userStore.userInfo?.name || userStore.userInfo?.username || ''
  return name ? `${t(`dashboard.home.greeting.${key}`)}，${name}` : t(`dashboard.home.greeting.${key}`)
})
</script>

<style scoped lang="scss">
.pv {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  padding: 20px;
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.pv__head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.pv__title {
  flex: 1;
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.pv__chip {
  display: inline-flex;
  align-items: center;
  height: 28px;
  padding: 0 12px;
  border-radius: 999px;
  background: var(--m-pSoft);
  color: var(--m-pInk);
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
}
.pv__box {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  border-radius: 20px;
  background: var(--el-bg-color-page);
  border: 1px solid var(--el-border-color-light);
}
.pv-hero {
  position: relative;
  overflow: hidden;
  padding: 18px;
  border-radius: 18px;
  background: var(--m-p);
  color: var(--m-pOn);
}
.pv-hero--top {
  border-radius: var(--app-radius-lg);
}
.pv-hero__blob {
  position: absolute;
  right: -36px;
  bottom: -50px;
  width: 140px;
  height: 140px;
  border-radius: 50%;
  background: var(--m-a);
}
.pv-hero__ring {
  position: absolute;
  right: 80px;
  top: -20px;
  width: 54px;
  height: 54px;
  border-radius: 50%;
  border: 9px solid var(--m-a2);
}
.pv-hero__body {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 4px;
  b {
    font-family: var(--app-font-display);
    font-size: 1.25rem;
    font-weight: 800;
  }
  small {
    font-size: 0.8125rem;
    opacity: 0.9;
  }
}
.pv-hero__cur {
  font-size: 0.75rem !important;
  font-weight: 700;
}
.pv-hero__big {
  font-size: 1.375rem !important;
}
.pv-hero__btns {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}
.pv-btn {
  display: inline-flex;
  align-items: center;
  min-height: 36px;
  padding: 0 14px;
  border-radius: 999px;
  font-size: 0.8125rem;
  font-weight: 600;
  white-space: nowrap;
}
.pv-btn--on {
  background: var(--m-pOn);
  color: var(--m-p);
}
.pv-btn--soft {
  background: color-mix(in srgb, var(--m-pOn) 18%, transparent);
  color: var(--m-pOn);
}
.pv-btn--p {
  background: var(--m-p);
  color: var(--m-pOn);
}
.pv-btn--s {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.pv__stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}
.pv-stat {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px;
  border-radius: 16px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  svg {
    width: 44px;
    height: 44px;
    flex: none;
  }
  small {
    display: block;
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--el-text-color-secondary);
  }
  b {
    font-size: 1.25rem;
    font-weight: 800;
    color: var(--el-text-color-primary);
  }
}
.pv-ring__track {
  fill: none;
  stroke: var(--el-fill-color-light);
  stroke-width: 9;
}
.pv-ring__bar {
  fill: none;
  stroke-width: 9;
  stroke-linecap: round;
}
.pv-app {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 16px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.pv-app__av {
  width: 36px;
  height: 36px;
  border-radius: 12px;
  font-size: 0.9375rem;
}
.pv-app__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  b {
    font-size: 0.875rem;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }
  small {
    font-size: 0.75rem;
    color: var(--el-text-color-secondary);
  }
}
.pv__row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
</style>
