<!-- 外观设置面板：配色（六套，带迷你预览）+ 深浅色 + 细节开关 + 实时预览。
     配色/深浅色写在 <html> 上，点选即全站生效；偏好保存在当前浏览器。 -->
<template>
  <div class="appearance" :class="{ 'appearance--narrow': narrow }">
    <section class="appearance__section">
      <header class="appearance__head">
        <h3>{{ t('theme.palette') }}</h3>
        <p>{{ t('theme.paletteDesc') }}</p>
      </header>
      <div class="pal-grid">
        <button
          v-for="p in themeStore.paletteOptions"
          :key="p.key"
          type="button"
          class="pal-card"
          :class="{ 'is-on': themeStore.palette === p.key }"
          :aria-pressed="themeStore.palette === p.key"
          @click="themeStore.setPalette(p.key)"
        >
          <!-- 迷你界面预览：外层带 data-palette，内部用 --m-* 自动取该配色 -->
          <span class="pal-card__preview" :data-palette="p.key">
            <span class="pv-side">
              <i class="pv-side__on" />
              <i />
              <i />
            </span>
            <span class="pv-main">
              <span class="pv-hero"><i class="pv-hero__dot" /></span>
              <span class="pv-row">
                <i />
                <i />
                <i class="pv-row__acc" />
              </span>
            </span>
          </span>
          <span class="pal-card__foot">
            <span class="pal-card__name">
              <b>{{ t(p.labelKey) }}</b>
              <small>{{ p.en }}</small>
            </span>
            <span class="pal-card__check" :class="{ 'is-on': themeStore.palette === p.key }">
              <component :is="ic('HOutline:CheckIcon')" v-if="themeStore.palette === p.key" />
            </span>
          </span>
        </button>
      </div>
    </section>

    <section class="appearance__section">
      <header class="appearance__head">
        <h3>{{ t('theme.mode') }}</h3>
        <p>{{ t('theme.modeDesc') }}</p>
      </header>
      <div class="mode-grid">
        <button
          v-for="m in modes"
          :key="m.key"
          type="button"
          class="mode-card"
          :class="{ 'is-on': themeStore.themeMode === m.key }"
          :aria-pressed="themeStore.themeMode === m.key"
          @click="themeStore.toggleThemeMode(m.key)"
        >
          <span class="mode-card__art">
            <span :class="['mode-half', m.left]"><i /><i /></span>
            <span :class="['mode-half', m.right]"><i /><i /></span>
          </span>
          <span class="mode-card__label">
            <component :is="ic(m.icon)" />
            {{ t(m.label) }}
          </span>
        </button>
      </div>
    </section>

    <section class="appearance__section">
      <header class="appearance__head">
        <h3>{{ t('theme.details') }}</h3>
      </header>
      <div class="detail-list">
        <div class="detail-row">
          <span class="detail-row__text">
            <b>{{ t('theme.compact') }}</b>
            <small>{{ t('theme.compactDesc') }}</small>
          </span>
          <AppSwitch :model-value="themeStore.compact" @update:model-value="themeStore.setCompact" />
        </div>
        <div class="detail-row">
          <span class="detail-row__text">
            <b>{{ t('theme.reduceMotion') }}</b>
            <small>{{ t('theme.reduceMotionDesc') }}</small>
          </span>
          <AppSwitch
            :model-value="themeStore.reduceMotion"
            @update:model-value="themeStore.setReduceMotion"
          />
        </div>
        <div class="detail-row">
          <span class="detail-row__text">
            <b>{{ t('theme.showTabs') }}</b>
            <small>{{ t('theme.showTabsDesc') }}</small>
          </span>
          <AppSwitch
            :model-value="themeStore.showTabs"
            @update:model-value="themeStore.toggleShowTabs"
          />
        </div>
      </div>
    </section>

    <section v-if="showPreview" class="appearance__section">
      <header class="appearance__head">
        <h3>{{ t('theme.preview') }}</h3>
      </header>
      <div class="preview">
        <div class="preview__hero">
          <span class="preview__blob" />
          <span class="preview__ring" />
          <div class="preview__hero-text">
            <b>{{ t('theme.previewTitle') }}</b>
            <small>{{ t('theme.previewSub') }}</small>
          </div>
        </div>
        <div class="preview__row">
          <UButton color="primary" size="sm" icon="i-lucide-plus">{{ t('theme.previewAction') }}</UButton>
          <UButton color="neutral" variant="soft" size="sm" icon="i-lucide-square-terminal">
            {{ t('theme.previewSecondary') }}
          </UButton>
          <StatusPill variant="success" :label="t('theme.previewRunning')" />
          <StatusPill variant="error" :label="t('theme.previewStopped')" />
        </div>
      </div>
    </section>

    <footer class="appearance__foot">
      <span class="appearance__saved">
        <component :is="ic('HOutline:CheckCircleIcon')" />
        {{ t('theme.saved') }}
      </span>
      <UButton color="neutral" variant="soft" size="sm" @click="themeStore.resetAppearance()">
        {{ t('theme.reset') }}
      </UButton>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ThemeMode } from '@/stores/theme'

withDefaults(defineProps<{ narrow?: boolean; showPreview?: boolean }>(), {
  narrow: false,
  showPreview: true,
})

const themeStore = useThemeStore()
const menuStore = useMenuStore()
const { t } = useI18n()
const ic = (name: string) => menuStore.iconComponents[name]

const modes: { key: ThemeMode; label: string; icon: string; left: string; right: string }[] = [
  { key: 'light', label: 'theme.light', icon: 'HOutline:SunIcon', left: 'is-light', right: 'is-light' },
  { key: 'dark', label: 'theme.dark', icon: 'HOutline:MoonIcon', left: 'is-dark', right: 'is-dark' },
  { key: 'auto', label: 'theme.auto', icon: 'HOutline:ComputerDesktopIcon', left: 'is-light', right: 'is-dark' },
]
</script>

<style scoped lang="scss">
.appearance {
  display: flex;
  flex-direction: column;
  gap: 26px;
}
.appearance__section {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.appearance__head h3 {
  font-size: 1rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.appearance__head p {
  margin-top: 3px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}

/* ---- 配色卡 ---- */
.pal-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(168px, 1fr));
  gap: 12px;
}
.appearance--narrow .pal-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
.pal-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 8px;
  border-radius: var(--app-radius-lg);
  border: 2px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s, transform 0.15s;
}
.pal-card:hover {
  transform: translateY(-2px);
  border-color: var(--el-text-color-secondary);
}
.pal-card.is-on {
  border-color: var(--m-p);
  box-shadow: 0 0 0 4px var(--m-pSoft);
}
.pal-card__preview {
  display: flex;
  gap: 6px;
  height: 84px;
  padding: 8px;
  border-radius: 14px;
  background: var(--m-bg);
  border: 1px solid var(--m-line);
}
.pv-side {
  width: 24%;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 5px;
  border-radius: 8px;
  background: var(--m-s);
}
.pv-side i {
  height: 5px;
  border-radius: 3px;
  background: var(--m-line);
}
.pv-side i.pv-side__on {
  height: 7px;
  background: var(--m-pSoft);
}
.pv-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.pv-hero {
  position: relative;
  flex: 1.3;
  border-radius: 9px;
  background: var(--m-p);
  overflow: hidden;
}
.pv-hero__dot {
  position: absolute;
  right: -10px;
  bottom: -14px;
  width: 38px;
  height: 38px;
  border-radius: 50%;
  background: var(--m-a);
}
.pv-row {
  flex: 1;
  display: flex;
  gap: 5px;
}
.pv-row i {
  flex: 1;
  border-radius: 7px;
  background: var(--m-s);
}
.pv-row i.pv-row__acc {
  flex: 0 0 16px;
  background: var(--m-a2);
}
.pal-card__foot {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 4px 2px;
}
.pal-card__name {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.pal-card__name b {
  font-size: 0.9375rem;
  font-weight: 800;
}
.pal-card__name small {
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--el-text-color-secondary);
}
.pal-card__check {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  flex: none;
  border-radius: 50%;
  border: 2px solid var(--el-border-color-light);
}
.pal-card__check.is-on {
  border: none;
  background: var(--m-p);
  color: var(--m-pOn);
}
.pal-card__check :deep(svg) {
  width: 14px;
  height: 14px;
  stroke-width: 3;
}

/* ---- 深浅色 ---- */
.mode-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}
.mode-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px;
  border-radius: var(--app-radius);
  border: 2px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.mode-card.is-on {
  border-color: var(--m-p);
  box-shadow: 0 0 0 4px var(--m-pSoft);
}
.mode-card__art {
  display: flex;
  height: 54px;
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid var(--el-border-color-light);
}
.mode-half {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 5px;
  padding: 8px 7px;
}
.mode-half.is-light {
  background: #fbf8fb;
}
.mode-half.is-dark {
  background: #1c1520;
}
.mode-half i {
  height: 7px;
  border-radius: 4px;
  background: var(--m-p);
}
.mode-half i + i {
  width: 60%;
  opacity: 0.35;
}
.mode-half.is-light i + i {
  background: #2a1630;
}
.mode-half.is-dark i + i {
  background: #f3e9ef;
}
.mode-card__label {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.mode-card__label :deep(svg) {
  width: 16px;
  height: 16px;
}

/* ---- 细节 ---- */
.detail-list {
  display: flex;
  flex-direction: column;
  border-radius: var(--app-radius);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  padding: 0 16px;
}
.detail-row {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 62px;
  padding: 8px 0;
}
.detail-row + .detail-row {
  border-top: 1px solid var(--el-border-color-lighter);
}
.detail-row__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.detail-row__text b {
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.detail-row__text small {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}

/* ---- 预览 ---- */
.preview {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color-page);
  border: 1px solid var(--el-border-color-light);
}
.preview__hero {
  position: relative;
  overflow: hidden;
  padding: 18px;
  border-radius: 18px;
  background: var(--m-p);
  color: var(--m-pOn);
}
.preview__blob {
  position: absolute;
  right: -34px;
  bottom: -46px;
  width: 130px;
  height: 130px;
  border-radius: 50%;
  background: var(--m-a);
}
.preview__ring {
  position: absolute;
  right: 74px;
  top: -18px;
  width: 50px;
  height: 50px;
  border-radius: 50%;
  border: 9px solid var(--m-a2);
}
.preview__hero-text {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.preview__hero-text b {
  font-size: 1.15rem;
  font-weight: 800;
}
.preview__hero-text small {
  font-size: 0.8125rem;
  opacity: 0.92;
}
.preview__row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.appearance__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.appearance__saved {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.appearance__saved :deep(svg) {
  width: 16px;
  height: 16px;
  color: var(--m-ok);
}
</style>
