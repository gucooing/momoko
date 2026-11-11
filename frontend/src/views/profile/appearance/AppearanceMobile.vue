<!-- 外观 · 手机（ThemeMobile）：当前主题卡片（在线编辑 / 导出）→ 主题（横向滑动，内置 + 我的主题）→ 背景 → 深浅色 → 细节；
     页头的「导入主题包」「恢复默认」在 profile/index.vue。导入 / 导出是底部面板。 -->
<template>
  <div class="amb">
    <!-- 当前主题 -->
    <section class="amb-card amb-cur">
      <div class="amb-cur__top">
        <ThemeScene
          class="amb-cur__mini"
          compact
          :content="themeStore.activeContent"
          :scheme="scheme"
          :override="themeStore.usingSite ? null : themeStore.bgOverride"
        />
        <span class="amb-cur__text">
          <span class="amb-lbl">{{ t('theme.current.label') }}</span>
          <b>{{ currentInfo.name }}</b>
          <small>{{ t('theme.current.bgTag', { name: currentInfo.bgName }) }} · {{ currentInfo.pagesLong }}</small>
        </span>
      </div>
      <div class="amb-cur__acts">
        <button type="button" class="amb-btn is-primary" @click="editCurrent">
          <PencilSquareIcon aria-hidden="true" />{{ t('theme.current.editShort') }}
        </button>
        <button type="button" class="amb-btn" @click="openExport()">
          <ArrowDownTrayIcon aria-hidden="true" />{{ t('theme.export.title') }}
        </button>
      </div>
    </section>

    <!-- 主题 -->
    <section class="amb-sec">
      <div class="amb-sec__head">
        <h2 class="amb-h">{{ t('theme.themes') }}</h2>
        <button type="button" class="amb-link" @click="editTheme()"><PlusIcon aria-hidden="true" />{{ t('theme.newShort') }}</button>
      </div>
      <div class="amb-scroll">
        <button
          v-for="b in builtins"
          :key="b.key"
          type="button"
          class="amb-pc"
          :class="{ on: b.on }"
          :aria-pressed="b.on"
          @click="selectBuiltin(b.key)"
        >
          <ThemeScene class="amb-pc__mini" compact :content="b.content" :scheme="scheme" />
          <span class="amb-pc__foot">
            <b>{{ b.name }}</b>
            <span class="amb-chk" :class="{ on: b.on }"><CheckIcon v-if="b.on" aria-hidden="true" /></span>
          </span>
        </button>
        <div v-for="m in mine" :key="m.theme.id" class="amb-pc" :class="{ on: m.on }">
          <button type="button" class="amb-pc__pick" :aria-pressed="m.on" @click="selectMine(m.theme)">
            <ThemeScene class="amb-pc__mini" compact :content="m.content" :scheme="scheme">
              <span class="amb-tag">{{ m.site ? t('theme.siteDefaultBadge') : m.theme.source === 'import' ? t('theme.tagImported') : t('theme.tagMine') }}</span>
            </ThemeScene>
            <span class="amb-pc__foot">
              <b>{{ m.theme.name }}</b>
              <span class="amb-chk" :class="{ on: m.on }"><CheckIcon v-if="m.on" aria-hidden="true" /></span>
            </span>
          </button>
          <span class="amb-pc__menu">
            <ActionMenu :items="mobileMenu(m)" :label="t('theme.actions.more')" @select="onMobileMenu(m, $event)" />
          </span>
        </div>
      </div>
    </section>

    <!-- 背景 -->
    <section class="amb-sec">
      <div class="amb-sec__head">
        <h2 class="amb-h">{{ t('theme.background') }}</h2>
        <StatusPill v-if="themeStore.bgModified" variant="warning" :label="t('theme.bgModifiedShort')" />
      </div>
      <BackgroundEditor v-model="bgModel" variant="mobile" :scheme="scheme" />
    </section>

    <!-- 深浅色 -->
    <section class="amb-sec">
      <h2 class="amb-h">{{ t('theme.mode') }}</h2>
      <div class="amb-seg" role="group" :aria-label="t('theme.mode')">
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
    </section>

    <!-- 细节 -->
    <section class="amb-sec">
      <h2 class="amb-h">{{ t('theme.details') }}</h2>
      <div class="amb-card amb-list">
        <label class="amb-row">
          <span class="amb-row__text"><b>{{ t('theme.compact') }}</b><small>{{ t('theme.compactDescShort') }}</small></span>
          <AppSwitch :model-value="themeStore.compact" :aria-label="t('theme.compact')" @update:model-value="themeStore.setCompact" />
        </label>
        <label class="amb-row">
          <span class="amb-row__text"><b>{{ t('theme.reduceMotion') }}</b><small>{{ t('theme.reduceMotionDescShort') }}</small></span>
          <AppSwitch :model-value="themeStore.reduceMotion" :aria-label="t('theme.reduceMotion')" @update:model-value="themeStore.setReduceMotion" />
        </label>
        <div class="amb-row">
          <span class="amb-row__text"><b>{{ t('theme.radius') }}</b><small>{{ t('theme.radiusDesc') }}</small></span>
          <div class="amb-seg amb-seg--sm" role="group" :aria-label="t('theme.radius')">
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
      </div>
    </section>

    <p class="amb-saved">{{ userStore.userInfo ? t('theme.mobileSaved') : t('theme.savedLocal') }}</p>
  </div>
</template>

<script setup lang="ts">
import { ArrowDownTrayIcon, CheckIcon, PencilSquareIcon, PlusIcon } from '@heroicons/vue/24/outline'
import type { RadiusMode, ThemeMode } from '@/stores/theme'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import ThemeScene from '@/components/theme/ThemeScene.vue'
import BackgroundEditor from '@/components/theme/BackgroundEditor.vue'
import { useAppearancePage } from './useAppearancePage'

defineOptions({ name: 'AppearanceMobile' })

const {
  t,
  themeStore,
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
  bgModel,
  currentInfo,
} = useAppearancePage()

const userStore = useUserStore()
const modes: ThemeMode[] = ['light', 'dark', 'auto']
const radii: { key: RadiusMode; label: string }[] = [
  { key: 'round', label: 'theme.radiusRound' },
  { key: 'standard', label: 'theme.radiusStandard' },
]

type MineRow = (typeof mine.value)[number]
const mobileMenu = (m: MineRow): ActionMenuItem[] => [
  { key: 'edit', label: t('theme.actions.edit'), icon: 'i-lucide-pencil' },
  { key: 'preview', label: t('theme.actions.preview'), icon: 'i-lucide-eye' },
  { key: 'export', label: t('theme.actions.export'), icon: 'i-lucide-download' },
  ...mineMenu(m),
]
const onMobileMenu = (m: MineRow, key: string) => {
  if (key === 'edit') editTheme(m.theme.id)
  else if (key === 'preview') previewMine(m.theme)
  else if (key === 'export') exportMine(m.theme)
  else onMineMenu(m, key)
}
</script>

<style scoped lang="scss">
.amb {
  display: flex;
  flex-direction: column;
  gap: 22px;
  min-width: 0;
}
.amb-card {
  border-radius: var(--app-radius-xl);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.amb-cur {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
}
.amb-cur__top {
  display: flex;
  align-items: center;
  gap: 12px;
}
.amb-cur__mini {
  flex: none;
  width: 112px;
  height: 76px !important;
  padding: 6px !important;
}
.amb-cur__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
  b {
    font-family: var(--app-font-display);
    font-size: 1.125rem;
    font-weight: 800;
  }
  b,
  small {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  small {
    font-size: 0.75rem;
    color: var(--el-text-color-secondary);
  }
}
.amb-cur__acts {
  display: flex;
  gap: 8px;
  .amb-btn {
    flex: 1;
  }
}
.amb-lbl {
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--el-text-color-secondary);
}
.amb-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 14px;
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
  &.is-primary {
    background: var(--m-p);
    color: var(--m-pOn);
  }
}
.amb-sec {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.amb-sec__head {
  display: flex;
  align-items: center;
  gap: 8px;
  .amb-h {
    flex: 1;
  }
}
.amb-h {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
}
.amb-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-height: 44px;
  padding: 0 4px;
  border: 0;
  background: transparent;
  color: var(--m-pInk);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 700;
  cursor: pointer;
  svg {
    width: 16px;
    height: 16px;
    stroke-width: 2.4;
  }
}
.amb-scroll {
  display: flex;
  gap: 10px;
  margin: 0 -16px;
  padding: 2px 16px 6px;
  overflow-x: auto;
  scrollbar-width: none;
  &::-webkit-scrollbar {
    display: none;
  }
}
.amb-pc {
  position: relative;
  flex: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 136px;
  padding: 8px;
  border-radius: 20px;
  border: 2px solid var(--m-line);
  background: var(--m-s);
  color: var(--m-ink);
  font: inherit;
  text-align: left;
  cursor: pointer;
  &.on {
    border-color: var(--m-p);
    box-shadow: 0 0 0 4px var(--m-pSoft);
  }
}
.amb-pc__pick {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.amb-pc__mini {
  height: 70px !important;
  padding: 6px !important;
}
.amb-pc__foot {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 2px;
  b {
    flex: 1;
    min-width: 0;
    font-size: 0.8125rem;
    font-weight: 800;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}
.amb-pc__menu {
  position: absolute;
  right: 10px;
  top: 10px;
  border-radius: 12px;
  background: color-mix(in srgb, var(--m-s) 86%, transparent);
  :deep(.action-menu__trigger) {
    width: 32px;
    height: 32px;
  }
}
.amb-tag {
  position: absolute;
  left: 6px;
  top: 6px;
  display: inline-flex;
  align-items: center;
  height: 18px;
  padding: 0 6px;
  border-radius: 999px;
  background: var(--m-ink);
  color: var(--m-bg);
  font-size: 0.625rem;
  font-weight: 800;
}
.amb-chk {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 2px solid var(--m-line);
  svg {
    width: 13px;
    height: 13px;
    stroke-width: 3;
  }
  &.on {
    border: 0;
    background: var(--m-p);
    color: var(--m-pOn);
  }
}
.amb-seg {
  display: flex;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  background: var(--m-s2);
  button {
    flex: 1;
    min-height: 40px;
    padding: 0 12px;
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
.amb-seg--sm {
  flex: none;
  button {
    min-height: 34px;
  }
}
.amb-list {
  padding: 4px 16px;
}
.amb-row {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 64px;
  padding: 8px 0;
  border-top: 1px solid var(--m-line);
  cursor: pointer;
  &:first-child {
    border-top: 0;
  }
}
.amb-row__text {
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
    font-size: 0.75rem;
    color: var(--el-text-color-secondary);
  }
}
.amb-saved {
  margin: 0;
  text-align: center;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
</style>
