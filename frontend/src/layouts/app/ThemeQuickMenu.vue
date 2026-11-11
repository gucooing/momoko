<!-- 顶栏主题快捷菜单（DashboardPC）：调色板按钮 → 六套内置主题 + 我的主题 + 浅色/深色/跟随系统 + 更多外观设置（个人中心 · 外观）。 -->
<template>
  <AppDropdown align="end" :width="340">
    <template #trigger="{ open }">
      <AppIconButton
        icon="HOutline:SwatchIcon"
        :label="t('shell.themeAndAppearance')"
        :box="44"
        :active="open"
        :aria-expanded="open"
      />
    </template>
    <template #default="{ close }">
      <div class="tq">
        <header class="tq__head">
          <span class="tq__title">{{ t('theme.quickTitle') }}</span>
          <button type="button" class="tq__more" @click="openAppearance(close)">
            {{ t('theme.more') }}
            <component :is="ic('HOutline:ChevronRightIcon')" />
          </button>
        </header>

        <div class="tq__grid">
          <button
            v-for="p in themeStore.paletteOptions"
            :key="p.key"
            type="button"
            class="tq__sw"
            :class="{ 'is-on': themeStore.isBuiltin && !themeStore.usingSite && themeStore.currentTheme === p.key }"
            :aria-pressed="themeStore.isBuiltin && !themeStore.usingSite && themeStore.currentTheme === p.key"
            @click="themeStore.setPalette(p.key)"
          >
            <span class="tq__dot" :data-palette="p.key">
              <i class="is-p" />
              <i class="is-a" />
              <i class="is-a2" />
            </span>
            <span class="tq__name">{{ t(p.labelKey) }}</span>
          </button>
        </div>

        <template v-if="mineList.length">
          <div class="tq__lbl">{{ t('theme.mineShort') }}</div>
          <div class="tq__grid">
            <button
              v-for="m in mineList"
              :key="m.id"
              type="button"
              class="tq__sw"
              :class="{ 'is-on': m.on }"
              :aria-pressed="m.on"
              @click="themeStore.setTheme(m.id, m.theme)"
            >
              <span class="tq__dot">
                <i class="is-p" :style="{ background: m.colors[0] }" />
                <i class="is-a" :style="{ background: m.colors[1] }" />
                <i class="is-a2" :style="{ background: m.colors[2] }" />
              </span>
              <span class="tq__name">{{ m.name }}</span>
            </button>
          </div>
        </template>

        <div class="tq__seg" role="group" :aria-label="t('theme.mode')">
          <button
            v-for="m in modes"
            :key="m.key"
            type="button"
            :class="{ 'is-on': themeStore.themeMode === m.key }"
            @click="themeStore.toggleThemeMode(m.key)"
          >
            <component :is="ic(m.icon)" />
            {{ t(m.label) }}
          </button>
        </div>
      </div>
    </template>
  </AppDropdown>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ThemeMode } from '@/stores/theme'
import { normalizeContent, resolveTokens } from '@/theme/content'

const themeStore = useThemeStore()
const menuStore = useMenuStore()
const { t } = useI18n()
const ic = (name: string) => menuStore.iconComponents[name]

const modes: { key: ThemeMode; label: string; icon: string }[] = [
  { key: 'light', label: 'theme.light', icon: 'HOutline:SunIcon' },
  { key: 'dark', label: 'theme.dark', icon: 'HOutline:MoonIcon' },
  { key: 'auto', label: 'theme.auto', icon: 'HOutline:ComputerDesktopIcon' },
]

// 我的主题（最多 6 个）：色块取主题实际的主色 / 强调色 / 辅助色
const mineList = computed(() =>
  themeStore.myThemes.slice(0, 6).map((th) => {
    const tk = resolveTokens(normalizeContent(th.content), themeStore.scheme)
    return {
      id: th.id,
      theme: th,
      name: th.name,
      colors: [tk.p, tk.a, tk.a2],
      on: !themeStore.usingSite && themeStore.currentTheme === th.id,
    }
  }),
)
onMounted(() => {
  if (!themeStore.themesLoaded) themeStore.loadThemes().catch(() => undefined)
})

const router = useRouter()
const openAppearance = (close: () => void) => {
  close()
  void router.push({ path: '/profile', query: { tab: 'appearance' } })
}
</script>

<style scoped lang="scss">
.tq {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 16px;
}
.tq__head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.tq__title {
  flex: 1;
  font-size: 1rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.tq__more {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  border: none;
  background: transparent;
  color: var(--m-pInk);
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.tq__more :deep(svg) {
  width: 14px;
  height: 14px;
}
.tq__grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 6px;
}
.tq__sw {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 7px;
  padding: 10px 4px;
  border-radius: 16px;
  border: 2px solid transparent;
  background: transparent;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
}
.tq__sw:hover {
  background: var(--el-fill-color-light);
}
.tq__sw.is-on {
  border-color: var(--m-p);
  background: var(--m-pSoft);
}
.tq__dot {
  display: flex;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  overflow: hidden;
  transform: rotate(-35deg);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--m-ink) 10%, transparent);
}
.tq__dot i {
  display: block;
}
.tq__dot .is-p {
  flex: 1.4;
  background: var(--m-p);
}
.tq__dot .is-a {
  flex: 1;
  background: var(--m-a);
}
.tq__dot .is-a2 {
  flex: 0.7;
  background: var(--m-a2);
}
.tq__name {
  max-width: 100%;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tq__lbl {
  margin-bottom: -6px;
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--el-text-color-secondary);
}
.tq__seg {
  display: flex;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.tq__seg button {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  min-height: 34px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 700;
  cursor: pointer;
}
.tq__seg button :deep(svg) {
  width: 14px;
  height: 14px;
}
.tq__seg button.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}
</style>
