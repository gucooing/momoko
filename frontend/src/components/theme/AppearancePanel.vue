<!-- 外观设置面板：配色（六套，带迷你界面预览）+ 深浅色 + 细节（紧凑 / 减少动效 / 圆角 / 标签栏）。
     variant='page'（ThemePC 左栏：分组卡片）/ 'mobile'（ThemeMobile：两列配色 + 深浅色分段）/ 'drawer'（登录页等处的外观抽屉）。
     点选即全站生效；登录后跟随账号同步到所有设备（stores/theme/sync.ts），未登录时保存在当前浏览器。 -->
<template>
  <div class="ap" :class="`ap--${variant}`">
    <!-- 配色 -->
    <section :class="isPage ? 'ap-card' : 'ap-sec'" aria-labelledby="ap-pal-h">
      <header class="ap-head">
        <h2 id="ap-pal-h" :class="isPage ? 'ap-head__title' : 'ap-h'">{{ t('theme.palette') }}</h2>
        <p v-if="isPage">{{ t('theme.paletteDesc') }}</p>
      </header>
      <div class="pal-grid">
        <button
          v-for="p in themeStore.paletteOptions"
          :key="p.key"
          type="button"
          class="pal-card"
          :class="{ 'is-on': themeStore.isBuiltin && !themeStore.usingSite && themeStore.currentTheme === p.key }"
          :aria-pressed="themeStore.isBuiltin && !themeStore.usingSite && themeStore.currentTheme === p.key"
          @click="themeStore.setPalette(p.key)"
        >
          <!-- 迷你界面：外层带 data-palette，内部直接读该配色的 --m-* -->
          <span class="pal-card__preview" :data-palette="p.key">
            <span v-if="isPage" class="pv-side">
              <i class="pv-side__on" />
              <i />
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
              <small v-if="isPage">{{ p.en }}</small>
            </span>
            <span class="ap-check" :class="{ 'is-on': themeStore.isBuiltin && !themeStore.usingSite && themeStore.currentTheme === p.key }">
              <component :is="ic('HOutline:CheckIcon')" v-if="themeStore.isBuiltin && !themeStore.usingSite && themeStore.currentTheme === p.key" />
            </span>
          </span>
        </button>
      </div>
    </section>

    <!-- 深浅色 -->
    <section :class="isPage ? 'ap-card' : 'ap-sec'" aria-labelledby="ap-mode-h">
      <header class="ap-head">
        <h2 id="ap-mode-h" :class="isPage ? 'ap-head__title' : 'ap-h'">{{ t('theme.mode') }}</h2>
        <p v-if="isPage">{{ t('theme.modeDesc') }}</p>
      </header>
      <div v-if="variant === 'mobile'" class="ap-seg ap-seg--full" role="group" :aria-label="t('theme.mode')">
        <button
          v-for="m in modes"
          :key="m.key"
          type="button"
          :class="{ 'is-on': themeStore.themeMode === m.key }"
          :aria-pressed="themeStore.themeMode === m.key"
          @click="themeStore.toggleThemeMode(m.key)"
        >
          {{ t(m.label) }}
        </button>
      </div>
      <div v-else class="mode-grid">
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
            <span class="mode-half" :data-palette="themeStore.palette" :data-scheme="m.left"><i /><i /><i /></span>
            <span class="mode-half" :data-palette="themeStore.palette" :data-scheme="m.right"><i /><i /><i /></span>
          </span>
          <span class="mode-card__foot">
            <b>{{ t(m.label) }}</b>
            <span class="ap-check" :class="{ 'is-on': themeStore.themeMode === m.key }">
              <component :is="ic('HOutline:CheckIcon')" v-if="themeStore.themeMode === m.key" />
            </span>
          </span>
        </button>
      </div>
    </section>

    <!-- 细节 -->
    <section :class="isPage ? 'ap-card ap-card--list' : 'ap-sec'" aria-labelledby="ap-detail-h">
      <h2 id="ap-detail-h" :class="isPage ? 'ap-head__title ap-head__title--list' : 'ap-h'">{{ t('theme.details') }}</h2>
      <div :class="isPage ? 'detail-list' : 'detail-list detail-list--boxed'">
        <label class="detail-row">
          <span class="detail-row__text">
            <b>{{ t('theme.compact') }}</b>
            <small>{{ isPage ? t('theme.compactDesc') : t('theme.compactDescShort') }}</small>
          </span>
          <AppSwitch :model-value="themeStore.compact" :aria-label="t('theme.compact')" @update:model-value="themeStore.setCompact" />
        </label>
        <label class="detail-row">
          <span class="detail-row__text">
            <b>{{ t('theme.reduceMotion') }}</b>
            <small>{{ isPage ? t('theme.reduceMotionDesc') : t('theme.reduceMotionDescShort') }}</small>
          </span>
          <AppSwitch
            :model-value="themeStore.reduceMotion"
            :aria-label="t('theme.reduceMotion')"
            @update:model-value="themeStore.setReduceMotion"
          />
        </label>
        <div class="detail-row">
          <span class="detail-row__text">
            <b>{{ t('theme.radius') }}</b>
            <small>{{ t('theme.radiusDesc') }}</small>
          </span>
          <div class="ap-seg" role="group" :aria-label="t('theme.radius')">
            <button
              v-for="r in radii"
              :key="r.key"
              type="button"
              :class="{ 'is-on': themeStore.radius === r.key }"
              :aria-pressed="themeStore.radius === r.key"
              @click="themeStore.setRadius(r.key)"
            >
              {{ t(r.label) }}
            </button>
          </div>
        </div>
        <label v-if="variant !== 'mobile'" class="detail-row">
          <span class="detail-row__text">
            <b>{{ t('theme.showTabs') }}</b>
            <small>{{ t('theme.showTabsDesc') }}</small>
          </span>
          <AppSwitch
            :model-value="themeStore.showTabs"
            :aria-label="t('theme.showTabs')"
            @update:model-value="themeStore.toggleShowTabs"
          />
        </label>
      </div>
    </section>

    <footer class="ap-foot">
      <template v-if="variant === 'mobile'">
        <StatusPill variant="success" :label="t('theme.previewRunning')" />
        <StatusPill variant="error" :label="t('theme.previewStopped')" />
        <span class="ap-foot__saved">{{ savedText }}</span>
      </template>
      <template v-else>
        <span class="ap-foot__saved">
          <component :is="ic('HOutline:CheckIcon')" class="ap-foot__ok" />
          {{ savedText }}
        </span>
        <span class="ap-foot__spacer" />
        <UButton color="neutral" variant="soft" class="ap-foot__reset" @click="themeStore.resetAppearance()">
          {{ t('theme.reset') }}
        </UButton>
      </template>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { RadiusMode, ThemeMode } from '@/stores/theme'

const props = withDefaults(defineProps<{ variant?: 'page' | 'mobile' | 'drawer' }>(), {
  variant: 'page',
})

const themeStore = useThemeStore()
const menuStore = useMenuStore()
const userStore = useUserStore()
const { t } = useI18n()
const ic = (name: string) => menuStore.iconComponents[name]

const isPage = computed(() => props.variant === 'page')
const savedText = computed(() => (userStore.userInfo ? t('theme.savedSync') : t('theme.savedLocal')))

const modes: { key: ThemeMode; label: string; left: 'light' | 'dark'; right: 'light' | 'dark' }[] = [
  { key: 'light', label: 'theme.light', left: 'light', right: 'light' },
  { key: 'dark', label: 'theme.dark', left: 'dark', right: 'dark' },
  { key: 'auto', label: 'theme.auto', left: 'light', right: 'dark' },
]
const radii: { key: RadiusMode; label: string }[] = [
  { key: 'round', label: 'theme.radiusRound' },
  { key: 'standard', label: 'theme.radiusStandard' },
]
</script>

<style scoped lang="scss">
.ap {
  display: flex;
  flex-direction: column;
  gap: 20px;
  min-width: 0;
}
.ap--mobile,
.ap--drawer {
  gap: 22px;
}
.ap-card {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 24px;
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.ap-card--list {
  gap: 0;
  padding: 12px 24px 16px;
}
.ap-sec {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.ap-head {
  display: flex;
  flex-direction: column;
  gap: 4px;
  p {
    margin: 0;
    font-size: 0.875rem;
    color: var(--el-text-color-secondary);
  }
}
.ap-head__title {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.25rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.ap-head__title--list {
  margin: 12px 0 8px;
}
.ap-h {
  margin: 0;
  font-size: 0.8125rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--el-text-color-secondary);
}

/* ---- 勾选圆点 ---- */
.ap-check {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  flex: none;
  border-radius: 50%;
  border: 2px solid var(--el-border-color-light);
}
.ap-check.is-on {
  border: none;
  background: var(--m-p);
  color: var(--m-pOn);
}
.ap-check :deep(svg) {
  width: 16px;
  height: 16px;
  stroke-width: 3;
}

/* ---- 配色卡 ---- */
.pal-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
  gap: 14px;
}
.ap--mobile .pal-grid,
.ap--drawer .pal-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.pal-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
  padding: 10px;
  border-radius: 22px;
  border: 2px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
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
.ap--mobile .pal-card,
.ap--drawer .pal-card {
  gap: 8px;
  padding: 8px;
  border-radius: 20px;
}
.pal-card__preview {
  display: flex;
  gap: 8px;
  height: 112px;
  padding: 10px;
  border-radius: 15px;
  background: var(--m-bg);
  border: 1px solid var(--m-line);
}
.ap--mobile .pal-card__preview,
.ap--drawer .pal-card__preview {
  height: 76px;
  padding: 8px;
  border-radius: 13px;
}
.pv-side {
  width: 24%;
  display: flex;
  flex-direction: column;
  gap: 5px;
  padding: 6px;
  border-radius: 9px;
  background: var(--m-s);
}
.pv-side i {
  display: block;
  height: 6px;
  border-radius: 3px;
  background: var(--m-line);
}
.pv-side i.pv-side__on {
  height: 9px;
  border-radius: 4px;
  background: var(--m-pSoft);
}
.pv-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.pv-hero {
  position: relative;
  flex: 1.3;
  border-radius: 10px;
  background: var(--m-p);
  overflow: hidden;
}
.pv-hero__dot {
  position: absolute;
  right: -14px;
  bottom: -20px;
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--m-a);
}
.ap--mobile .pv-hero__dot,
.ap--drawer .pv-hero__dot {
  right: -10px;
  bottom: -14px;
  width: 36px;
  height: 36px;
}
.pv-row {
  flex: 1;
  display: flex;
  gap: 6px;
}
.pv-row i {
  flex: 1;
  border-radius: 8px;
  background: var(--m-s);
}
.pv-row i.pv-row__acc {
  flex: 0 0 20px;
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
  b {
    font-size: 0.9375rem;
    font-weight: 800;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  small {
    font-size: 0.6875rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    color: var(--el-text-color-secondary);
  }
}
.ap--mobile .pal-card__name b,
.ap--drawer .pal-card__name b {
  font-size: 0.875rem;
}

/* ---- 深浅色卡 ---- */
.mode-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
}
.ap--drawer .mode-grid {
  gap: 10px;
}
.mode-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
  padding: 10px;
  border-radius: 22px;
  border: 2px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s, transform 0.15s;
}
.mode-card:hover {
  transform: translateY(-2px);
  border-color: var(--el-text-color-secondary);
}
.mode-card.is-on {
  border-color: var(--m-p);
  box-shadow: 0 0 0 4px var(--m-pSoft);
}
.mode-card__art {
  display: flex;
  height: 84px;
  border-radius: 14px;
  overflow: hidden;
  border: 1px solid var(--el-border-color-light);
}
.ap--drawer .mode-card__art {
  height: 60px;
}
.mode-half {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px;
  background: var(--m-bg);
}
.mode-half i {
  display: block;
  height: 8px;
  width: 45%;
  border-radius: 4px;
  background: var(--m-line);
}
.mode-half i:first-child {
  height: 22px;
  width: auto;
  border-radius: 7px;
  background: var(--m-p);
}
.mode-half i:nth-child(2) {
  width: 70%;
  background: var(--m-ink);
}
.ap--drawer .mode-half i:first-child {
  height: 14px;
}
.mode-card__foot {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 4px 2px;
  b {
    flex: 1;
    min-width: 0;
    font-size: 0.9375rem;
    font-weight: 800;
  }
}

/* ---- 分段 ---- */
.ap-seg {
  display: inline-flex;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  flex: none;
}
.ap-seg button {
  min-height: 36px;
  padding: 0 16px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.ap-seg button:hover {
  color: var(--el-text-color-primary);
}
.ap-seg button.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}
.ap-seg--full {
  display: flex;
}
.ap-seg--full button {
  flex: 1;
  min-height: 44px;
  font-size: 0.875rem;
}

/* ---- 细节 ---- */
.detail-list {
  display: flex;
  flex-direction: column;
}
.detail-list--boxed {
  padding: 4px 16px;
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.detail-row {
  display: flex;
  align-items: center;
  gap: 16px;
  min-height: 64px;
  padding: 8px 0;
  border-top: 1px solid var(--el-border-color-light);
  cursor: pointer;
}
.detail-list--boxed .detail-row {
  min-height: 60px;
  padding: 6px 0;
  gap: 14px;
}
.detail-list--boxed .detail-row:first-child {
  border-top: none;
}
div.detail-row {
  cursor: default;
}
.detail-row__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  b {
    font-size: 0.9375rem;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }
  small {
    font-size: 0.8125rem;
    color: var(--el-text-color-secondary);
  }
}
.detail-list--boxed .detail-row__text small {
  font-size: 0.75rem;
}

/* ---- 底部 ---- */
.ap-foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
}
.ap--mobile .ap-foot {
  gap: 8px;
}
.ap-foot__saved {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.ap--mobile .ap-foot__saved {
  font-size: 0.75rem;
}
.ap-foot__ok {
  width: 16px;
  height: 16px;
  color: var(--m-ok);
}
.ap-foot__spacer {
  flex: 1;
}
.ap-foot__reset {
  min-height: 44px;
  padding-inline: 18px;
  border-radius: 999px;
  font-weight: 600;
}

@media (width < 640px) {
  .mode-grid {
    gap: 8px;
  }
  .ap-card {
    padding: 18px;
  }
  .ap-card--list {
    padding: 8px 18px 12px;
  }
}
</style>
