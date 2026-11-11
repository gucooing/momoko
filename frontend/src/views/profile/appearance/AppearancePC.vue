<!-- 外观 · PC（ThemePC）：左栏 主题（内置 / 我的主题）→ 背景 → 深浅色与细节 → 自动保存与恢复默认；
     右栏吸顶「当前主题」：预览、名称与来源、标签、色块、文字对比度、在线编辑、导出 / 另存为、设为全站默认（管理员）。 -->
<template>
  <div class="apc">
    <div class="apc__main">
      <!-- 主题 -->
      <section class="apc-card">
        <header class="apc-head">
          <div class="apc-head__text">
            <h2>{{ t('theme.themes') }}</h2>
            <p>{{ t('theme.themesDesc') }}</p>
          </div>
          <button type="button" class="apc-btn" @click="openThemeImport()">
            <ArrowUpTrayIcon aria-hidden="true" />{{ t('theme.importPkg') }}
          </button>
          <button type="button" class="apc-btn is-primary" @click="editTheme()">
            <PlusIcon aria-hidden="true" class="is-bold" />{{ t('theme.newTheme') }}
          </button>
        </header>

        <div class="apc-lbl">{{ t('theme.builtin') }}</div>
        <div class="apc-grid apc-grid--builtin">
          <button
            v-for="b in builtins"
            :key="b.key"
            type="button"
            class="apc-pc"
            :class="{ on: b.on }"
            :aria-pressed="b.on"
            @click="selectBuiltin(b.key)"
          >
            <ThemeScene :content="b.content" :scheme="scheme" />
            <span class="apc-pc__foot">
              <b>{{ b.name }}</b>
              <span class="apc-chk" :class="{ on: b.on }"><CheckIcon v-if="b.on" aria-hidden="true" /></span>
            </span>
          </button>
        </div>

        <div class="apc-lbl">{{ t('theme.mine', { n: mine.length }) }}</div>
        <div class="apc-grid apc-grid--mine">
          <div v-for="m in mine" :key="m.theme.id" class="apc-pc apc-pc--mine" :class="{ on: m.on }">
            <button type="button" class="apc-pc__pick" :aria-pressed="m.on" @click="selectMine(m.theme)">
              <ThemeScene :content="m.content" :scheme="scheme">
                <span v-if="m.site" class="apc-site">{{ t('theme.siteDefaultBadge') }}</span>
              </ThemeScene>
              <span class="apc-pc__foot">
                <span class="apc-pc__name">
                  <b>{{ m.theme.name }}</b>
                  <small>{{ m.src }} · {{ m.by }}</small>
                </span>
                <span class="apc-chk" :class="{ on: m.on }"><CheckIcon v-if="m.on" aria-hidden="true" /></span>
              </span>
            </button>
            <span class="apc-pc__acts">
              <small>{{ m.pages ? t('theme.pagesShort', { n: m.pages }) : t('theme.pagesGlobal') }}</small>
              <AppIconButton icon="HOutline:PencilSquareIcon" :label="t('theme.actions.edit')" :size="16" :box="32" @click="editTheme(m.theme.id)" />
              <AppIconButton icon="HOutline:EyeIcon" :label="t('theme.actions.preview')" :size="16" :box="32" @click="previewMine(m.theme)" />
              <AppIconButton icon="HOutline:ArrowDownTrayIcon" :label="t('theme.actions.export')" :size="16" :box="32" @click="exportMine(m.theme)" />
              <ActionMenu :items="mineMenu(m)" :label="t('theme.actions.more')" @select="onMineMenu(m, $event)" />
            </span>
          </div>
          <button type="button" class="apc-new" @click="editTheme()">
            <span class="apc-new__sticker"><PlusIcon aria-hidden="true" /></span>
            <b>{{ t('theme.newTheme') }}</b>
            <small>{{ t('theme.newThemeHint') }}</small>
          </button>
        </div>
      </section>

      <!-- 背景 -->
      <section class="apc-card">
        <header class="apc-head">
          <div class="apc-head__text">
            <h2>{{ t('theme.background') }}</h2>
            <p>{{ t('theme.backgroundDesc') }}</p>
          </div>
          <StatusPill v-if="themeStore.bgModified" variant="warning" :label="t('theme.bgModified')" />
        </header>
        <BackgroundEditor v-model="bgModel" variant="page" :scheme="scheme" />
      </section>

      <!-- 深浅色与细节 -->
      <section class="apc-card apc-card--list">
        <h2 class="apc-list-title">{{ t('theme.modeAndDetails') }}</h2>
        <div class="apc-row">
          <span class="apc-row__text"><b>{{ t('theme.mode') }}</b><small>{{ t('theme.modeRowDesc') }}</small></span>
          <div class="apc-seg" role="group" :aria-label="t('theme.mode')">
            <button
              v-for="m in modes"
              :key="m"
              type="button"
              :class="{ on: themeStore.themeMode === m }"
              :aria-pressed="themeStore.themeMode === m"
              @click="themeStore.toggleThemeMode(m)"
            >
              {{ t(`theme.${m}`) }}
            </button>
          </div>
        </div>
        <label class="apc-row is-click">
          <span class="apc-row__text"><b>{{ t('theme.compact') }}</b><small>{{ t('theme.compactDesc') }}</small></span>
          <AppSwitch :model-value="themeStore.compact" :aria-label="t('theme.compact')" @update:model-value="themeStore.setCompact" />
        </label>
        <label class="apc-row is-click">
          <span class="apc-row__text"><b>{{ t('theme.reduceMotion') }}</b><small>{{ t('theme.reduceMotionDesc') }}</small></span>
          <AppSwitch :model-value="themeStore.reduceMotion" :aria-label="t('theme.reduceMotion')" @update:model-value="themeStore.setReduceMotion" />
        </label>
        <div class="apc-row">
          <span class="apc-row__text"><b>{{ t('theme.radius') }}</b><small>{{ t('theme.radiusDesc') }}</small></span>
          <div class="apc-seg" role="group" :aria-label="t('theme.radius')">
            <button
              v-for="r in radii"
              :key="r.key"
              type="button"
              :class="{ on: themeStore.radius === r.key }"
              :aria-pressed="themeStore.radius === r.key"
              @click="themeStore.setRadius(r.key)"
            >
              {{ t(r.label) }}
            </button>
          </div>
        </div>
        <label class="apc-row is-click">
          <span class="apc-row__text"><b>{{ t('theme.showTabs') }}</b><small>{{ t('theme.showTabsDesc') }}</small></span>
          <AppSwitch :model-value="themeStore.showTabs" :aria-label="t('theme.showTabs')" @update:model-value="themeStore.toggleShowTabs" />
        </label>
      </section>

      <div class="apc-foot">
        <span class="apc-foot__saved"><CheckIcon aria-hidden="true" />{{ savedText }}</span>
        <span class="apc-spacer" />
        <button type="button" class="apc-btn" @click="themeStore.resetAppearance()">{{ t('theme.reset') }}</button>
      </div>
    </div>

    <!-- 当前主题 -->
    <aside class="apc-card apc-cur" :aria-label="t('theme.current.label')">
      <div class="apc-lbl">{{ t('theme.current.label') }}</div>
      <ThemeScene
        large
        :content="themeStore.activeContent"
        :scheme="scheme"
        :override="themeStore.usingSite ? null : themeStore.bgOverride"
        :page-key="null"
      />
      <div class="apc-cur__name">
        <b>{{ currentInfo.name }}</b>
        <small>{{ currentInfo.src }} · {{ t('theme.current.basedOn', { name: currentInfo.baseName }) }}</small>
      </div>
      <div class="apc-tags">
        <span>{{ t('theme.current.bgTag', { name: currentInfo.bgName }) }}</span>
        <span>{{ currentInfo.modeName }}</span>
        <span>{{ currentInfo.pagesLong }}</span>
      </div>
      <div class="apc-sw">
        <span v-for="w in swatches" :key="w.key" :title="w.color">
          <i :style="{ background: w.color }" />{{ w.label }}
        </span>
      </div>
      <div class="apc-contrast">
        <div class="apc-contrast__title">{{ t('theme.contrastTitle') }}</div>
        <div v-for="c in contrast" :key="c.key" class="apc-contrast__row">
          <span>{{ c.label }}</span>
          <b class="m-num">{{ c.value }}</b>
          <span class="apc-contrast__ic" :class="c.ok ? 'is-ok' : 'is-warn'">
            <component :is="c.ok ? CheckIcon : ExclamationTriangleIcon" aria-hidden="true" />
          </span>
        </div>
        <p v-if="lowContrastDim != null" class="apc-contrast__warn">{{ t('theme.lowBackdrop', { n: lowContrastDim }) }}</p>
      </div>
      <button type="button" class="apc-btn is-primary is-block" @click="editCurrent">
        <PencilSquareIcon aria-hidden="true" />{{ t('theme.current.edit') }}
      </button>
      <div class="apc-cur__row">
        <button type="button" class="apc-btn is-grow" @click="openExport()">
          <ArrowDownTrayIcon aria-hidden="true" />{{ t('theme.export.title') }}
        </button>
        <button type="button" class="apc-btn is-grow" @click="appearanceUi.saveAsOpen = true">
          <DocumentDuplicateIcon aria-hidden="true" />{{ t('theme.current.saveAs') }}
        </button>
      </div>
      <label v-if="canSetSite" class="apc-cur__site">
        <span class="apc-row__text">
          <b>{{ t('theme.current.siteDefault') }}</b>
          <small>{{ t('theme.current.siteDefaultDesc') }}</small>
        </span>
        <AppSwitch
          :model-value="themeStore.isSiteDefault"
          :disabled="themeStore.usingSite"
          :aria-label="t('theme.current.siteDefault')"
          @update:model-value="toggleSiteFor(null, $event)"
        />
      </label>
    </aside>
  </div>
</template>

<script setup lang="ts">
import {
  ArrowDownTrayIcon,
  ArrowUpTrayIcon,
  CheckIcon,
  DocumentDuplicateIcon,
  ExclamationTriangleIcon,
  PencilSquareIcon,
  PlusIcon,
} from '@heroicons/vue/24/outline'
import type { RadiusMode, ThemeMode } from '@/stores/theme'
import ThemeScene from '@/components/theme/ThemeScene.vue'
import BackgroundEditor from '@/components/theme/BackgroundEditor.vue'
import { appearanceUi, openThemeImport, useAppearancePage } from './useAppearancePage'

defineOptions({ name: 'AppearancePC' })

const {
  t,
  themeStore,
  canSetSite,
  scheme,
  builtins,
  mine,
  selectBuiltin,
  selectMine,
  editTheme,
  editCurrent,
  previewMine,
  openExport,
  exportMine,
  mineMenu,
  onMineMenu,
  toggleSiteFor,
  bgModel,
  currentInfo,
  swatches,
  contrast,
  lowContrastDim,
} = useAppearancePage()

const userStore = useUserStore()
const savedText = computed(() => (userStore.userInfo ? t('theme.savedAccount') : t('theme.savedLocal')))

const modes: ThemeMode[] = ['light', 'dark', 'auto']
const radii: { key: RadiusMode; label: string }[] = [
  { key: 'round', label: 'theme.radiusRound' },
  { key: 'standard', label: 'theme.radiusStandard' },
]
</script>

<style scoped lang="scss">
.apc {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 24px;
}
.apc__main {
  flex: 3 1 560px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.apc-card {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: calc(var(--cmp-card-unit) * 24);
  border-radius: var(--app-radius-xl);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.apc-card--list {
  gap: 0;
  padding: 12px 24px 16px;
}
.apc-head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 12px;
}
.apc-head__text {
  flex: 1 1 260px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  h2 {
    margin: 0;
    font-family: var(--app-font-display);
    font-size: 1.25rem;
    font-weight: 800;
  }
  p {
    margin: 0;
    font-size: 0.875rem;
    color: var(--el-text-color-secondary);
  }
}
.apc-list-title {
  margin: 12px 0 8px;
  font-family: var(--app-font-display);
  font-size: 1.25rem;
  font-weight: 800;
}
.apc-lbl {
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--el-text-color-secondary);
}
.apc-grid {
  display: grid;
  gap: 12px;
  margin-top: -6px;
}
.apc-grid--builtin {
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
}
.apc-grid--mine {
  grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
}
.apc-pc {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 10px;
  border-radius: 22px;
  border: 2px solid var(--m-line);
  background: var(--m-s);
  color: var(--m-ink);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s, transform 0.15s;
  &:hover {
    border-color: var(--m-ink2);
    transform: translateY(-2px);
  }
  &.on,
  &.on:hover {
    border-color: var(--m-p);
    box-shadow: 0 0 0 4px var(--m-pSoft);
  }
}
.apc-pc--mine {
  gap: 8px;
  cursor: default;
}
.apc-pc__pick {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.apc-pc__foot {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 2px 2px;
  > b {
    flex: 1;
    min-width: 0;
    font-size: 0.875rem;
    font-weight: 800;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}
.apc-pc__name {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  b,
  small {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  b {
    font-size: 0.875rem;
    font-weight: 800;
  }
  small {
    font-size: 0.6875rem;
    color: var(--m-ink2);
  }
}
.apc-pc__acts {
  display: flex;
  align-items: center;
  gap: 2px;
  padding-top: 6px;
  border-top: 1px solid var(--m-line);
  small {
    flex: 1;
    min-width: 0;
    padding-left: 4px;
    font-size: 0.6875rem;
    font-weight: 700;
    color: var(--m-ink2);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  :deep(.action-menu__trigger) {
    width: 32px;
    height: 32px;
    border-radius: 10px;
  }
}
.apc-chk {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  border: 2px solid var(--m-line);
  svg {
    width: 16px;
    height: 16px;
    stroke-width: 3;
  }
  &.on {
    border: 0;
    background: var(--m-p);
    color: var(--m-pOn);
  }
}
.apc-site {
  position: absolute;
  left: 8px;
  top: 8px;
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  background: var(--m-ink);
  color: var(--m-bg);
  font-size: 0.6875rem;
  font-weight: 800;
}
.apc-new {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 170px;
  padding: 12px;
  border-radius: 22px;
  border: 2px dashed var(--m-line);
  background: transparent;
  color: var(--m-ink);
  font: inherit;
  cursor: pointer;
  b {
    font-size: 0.875rem;
    font-weight: 800;
  }
  small {
    max-width: 160px;
    font-size: 0.75rem;
    color: var(--m-ink2);
    text-align: center;
  }
  &:hover {
    border-color: var(--m-p);
  }
}
.apc-new__sticker {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 14px;
  background: var(--m-pSoft);
  color: var(--m-pInk);
  rotate: -6deg;
  svg {
    width: 20px;
    height: 20px;
    stroke-width: 2.4;
  }
}

/* 按钮 */
.apc-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 18px;
  border: 0;
  border-radius: var(--app-radius-btn);
  background: var(--m-s2);
  color: var(--m-ink);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
  svg {
    width: 16px;
    height: 16px;
  }
  .is-bold {
    stroke-width: 2.4;
  }
  &:hover {
    background: var(--m-pSoft);
    color: var(--m-pInk);
  }
  &.is-primary {
    background: var(--m-p);
    color: var(--m-pOn);
  }
  &.is-block {
    width: 100%;
  }
  &.is-grow {
    flex: 1;
  }
}

/* 深浅色与细节 */
.apc-row {
  display: flex;
  align-items: center;
  gap: 16px;
  min-height: 64px;
  padding: 8px 0;
  border-top: 1px solid var(--m-line);
  &.is-click {
    cursor: pointer;
  }
}
.apc-row__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  b {
    font-size: 0.9375rem;
    font-weight: 700;
  }
  small {
    font-size: 0.8125rem;
    color: var(--el-text-color-secondary);
  }
}
.apc-seg {
  display: inline-flex;
  flex: none;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  background: var(--m-s2);
  button {
    min-height: 36px;
    padding: 0 16px;
    border: 0;
    border-radius: 999px;
    background: transparent;
    font: inherit;
    font-size: 0.8125rem;
    font-weight: 700;
    color: var(--m-ink2);
    cursor: pointer;
    white-space: nowrap;
  }
  .on {
    background: var(--m-s);
    color: var(--m-ink);
    box-shadow: 0 1px 3px rgba(20, 10, 30, 0.12);
  }
}
.apc-foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}
.apc-foot__saved {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
  svg {
    width: 16px;
    height: 16px;
    color: var(--m-ok);
  }
}
.apc-spacer {
  flex: 1;
}

/* 当前主题 */
.apc-cur {
  flex: 2 1 320px;
  min-width: 0;
  position: sticky;
  top: 0;
  padding: 20px;
}
.apc-cur__name {
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
    color: var(--el-text-color-secondary);
  }
}
.apc-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  span {
    display: inline-flex;
    align-items: center;
    height: 26px;
    padding: 0 10px;
    border-radius: 999px;
    background: var(--m-s2);
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--m-ink2);
    white-space: nowrap;
  }
}
.apc-sw {
  display: flex;
  gap: 6px;
  span {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    font-size: 0.625rem;
    font-weight: 700;
    color: var(--m-ink2);
    white-space: nowrap;
  }
  i {
    width: 100%;
    height: 30px;
    border-radius: 10px;
    border: 1px solid var(--m-line);
  }
}
.apc-contrast {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 12px;
  border-radius: 16px;
  background: var(--m-s2);
}
.apc-contrast__title {
  padding-bottom: 4px;
  font-size: 0.75rem;
  font-weight: 800;
  color: var(--m-ink2);
}
.apc-contrast__row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 26px;
  font-size: 0.75rem;
  > span:first-child {
    flex: 1;
  }
  b {
    font-weight: 800;
  }
}
.apc-contrast__ic {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  svg {
    width: 12px;
    height: 12px;
    stroke-width: 3;
  }
  &.is-ok {
    background: var(--m-okSoft);
    color: var(--m-ok);
  }
  &.is-warn {
    background: var(--m-warnSoft);
    color: var(--m-warn);
  }
}
.apc-contrast__warn {
  margin: 0;
  padding-top: 4px;
  font-size: 0.75rem;
  color: var(--m-warn);
}
.apc-cur__row {
  display: flex;
  gap: 8px;
}
.apc-cur__site {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-top: 14px;
  border-top: 1px solid var(--m-line);
  cursor: pointer;
  .apc-row__text b {
    font-size: 0.875rem;
  }
  .apc-row__text small {
    font-size: 0.75rem;
  }
}
</style>
