<!-- 在线主题编辑器（ThemeEditorPC / ThemeEditorLoginPC / ThemeEditorMobile）：独立全屏页 /profile/theme/editor/:id（id 为主题 id 或 new）。
     PC：顶栏（返回、名称、草稿状态、浅色 / 深色、电脑 / 手机、撤销 / 重做、导出、保存并使用）+ 左栏设置 + 中间逐页预览 + 右栏取色与检查。
     手机：顶栏 + 页面标签 + 预览 + 底部编辑面板（全局 / 仅本页、token、色相、推荐色、HEX，「背景 · 字体 · 组件」在底部面板里）。 -->
<template>
  <div class="ted" :class="{ 'is-mobile': isMobile }">
    <div v-if="ed.loading.value" class="ted-state">{{ t('theme.editor.loading') }}</div>
    <div v-else-if="ed.notFound.value" class="ted-state">
      <EmptyState :title="t('theme.editor.notFound')" icon="HOutline:SwatchIcon" />
      <UButton color="primary" @click="goBack">{{ t('theme.editor.back') }}</UButton>
    </div>

    <template v-else>
      <!-- 顶栏 -->
      <header class="ted-top">
        <AppIconButton icon="HOutline:ChevronLeftIcon" :label="t('theme.editor.back')" :box="44" @click="goBack" />
        <span v-if="!isMobile" class="ted-sticker"><SwatchIcon aria-hidden="true" /></span>
        <span class="ted-title">
          <input
            class="ted-name"
            type="text"
            maxlength="40"
            :value="ed.name.value"
            :placeholder="t('theme.editor.namePh')"
            :aria-label="t('theme.editor.namePh')"
            @input="ed.setName(($event.target as HTMLInputElement).value)"
          />
          <small>
            <template v-if="!isMobile">{{ t('theme.editor.title') }} · {{ t('theme.editor.draftSaved') }} · </template>
            <b :class="{ 'is-clean': !ed.changeCount.value }">
              {{ ed.changeCount.value ? t('theme.editor.changes', { n: ed.changeCount.value }) : t('theme.editor.noChanges') }}
            </b>
          </small>
        </span>

        <template v-if="!isMobile">
          <span class="ted-spacer" />
          <div class="ted-seg" role="group" :aria-label="t('theme.editor.schemeLabel')">
            <button v-for="m in schemes" :key="m.key" type="button" :class="{ on: ed.scheme.value === m.key }" @click="ed.scheme.value = m.key">
              <component :is="m.icon" aria-hidden="true" />{{ t(`theme.${m.key}`) }}
            </button>
          </div>
          <div class="ted-seg" role="group" :aria-label="t('theme.editor.deviceLabel')">
            <button v-for="d in devices" :key="d.key" type="button" :class="{ on: ed.device.value === d.key }" @click="ed.device.value = d.key">
              <component :is="d.icon" aria-hidden="true" />{{ t(`theme.editor.devices.${d.key}`) }}
            </button>
          </div>
          <span class="ted-spacer" />
          <AppIconButton icon="HOutline:ArrowUturnLeftIcon" :label="t('theme.editor.undo')" :box="44" :class="{ 'is-off': !ed.canUndo.value }" @click="ed.undo()" />
          <AppIconButton icon="HOutline:ArrowUturnRightIcon" :label="t('theme.editor.redo')" :box="44" :class="{ 'is-off': !ed.canRedo.value }" @click="ed.redo()" />
          <AppDropdown align="end" :width="300">
            <template #trigger="{ open }">
              <button type="button" class="ted-btn" :aria-expanded="open">
                <ArrowDownTrayIcon aria-hidden="true" />{{ t('theme.editor.export') }}<ChevronDownIcon aria-hidden="true" />
              </button>
            </template>
            <template #default="{ close }">
              <div class="ted-menu" role="menu">
                <button v-for="it in exportItems" :key="it.key" type="button" role="menuitem" class="ted-mi" @click="(onExport(it.key), close())">
                  <span class="ted-mi__ic" :class="`is-${it.key}`"><component :is="it.icon" aria-hidden="true" /></span>
                  <span class="ted-mi__text"><b>{{ it.title }}</b><small>{{ it.sub }}</small></span>
                </button>
              </div>
            </template>
          </AppDropdown>
          <button type="button" class="ted-btn is-primary" :disabled="ed.saving.value" @click="ed.save()">
            <CheckIcon aria-hidden="true" class="is-bold" />{{ t('theme.editor.saveAndUse') }}
          </button>
        </template>
        <template v-else>
          <AppIconButton
            :icon="ed.scheme.value === 'dark' ? 'HOutline:SunIcon' : 'HOutline:MoonIcon'"
            :label="t('theme.editor.schemeLabel')"
            :box="44"
            @click="ed.scheme.value = ed.scheme.value === 'dark' ? 'light' : 'dark'"
          />
          <AppIconButton icon="HOutline:ArrowDownTrayIcon" :label="t('theme.editor.exportPkg')" :box="44" @click="onExport('pkg')" />
          <button type="button" class="ted-btn is-primary is-sm" :disabled="ed.saving.value" @click="ed.save()">{{ t('theme.save') }}</button>
        </template>
      </header>

      <!-- PC -->
      <div v-if="!isMobile" class="ted-body">
        <aside class="ted-left">
          <EditorSettings />
        </aside>
        <EditorPreview variant="pc" />
        <aside class="ted-right">
          <EditorInspector />
        </aside>
      </div>

      <!-- 手机 -->
      <template v-else>
        <EditorPreview variant="mobile" class="ted-mpv" />
        <section class="ted-panel" :aria-label="t('theme.editor.panelLabel')">
          <span class="ted-panel__grip" aria-hidden="true" />
          <div class="ted-seg ted-seg--fill">
            <button type="button" :class="{ on: ed.scope.value === 'all' }" @click="ed.scopeChoice.value = 'all'">{{ t('theme.editor.scopeAll') }}</button>
            <button type="button" :class="{ on: ed.scope.value === 'page' }" @click="ed.scopeChoice.value = 'page'">
              {{ t('theme.editor.scopePage', { page: ed.pageName.value }) }}
            </button>
          </div>
          <div class="ted-tchips">
            <button
              v-for="k in mobileTokens"
              :key="k"
              type="button"
              class="ted-tchip"
              :class="{ on: ed.sel.value === k }"
              :aria-pressed="ed.sel.value === k"
              @click="ed.select(k)"
            >
              <i :style="{ background: k === 'hero' ? 'var(--m-p)' : ed.tokenValue(k) }" />
              {{ k === 'hero' ? t('theme.tokens.hero.name') : t(`theme.tokens.${k}.name`) }}
              <span v-if="k !== 'hero' && ed.isPageToken(k)" class="ted-tagx">{{ t('theme.editor.pageTag') }}</span>
            </button>
          </div>
          <template v-if="ed.sel.value !== 'hero' && !ed.isSizeSel.value">
            <ColorPicker
              variant="compact"
              :model-value="ed.selHex.value"
              :suggestions="ed.suggestions.value"
              @update:model-value="ed.setToken(ed.sel.value, $event)"
            />
          </template>
          <div class="ted-hexrow">
            <label v-if="ed.sel.value !== 'hero' && !ed.isSizeSel.value" class="ted-hex">
              <i :style="{ background: ed.selHex.value }" />
              <input class="m-mono" type="text" :value="ed.selHex.value" :aria-label="t('theme.editor.hexLabel')" @change="onHex" />
            </label>
            <span v-else class="ted-hexrow__hint">{{ ed.isSizeSel.value ? t('theme.editor.sizeInPanel') : t('theme.editor.bannerInPanel') }}</span>
            <button type="button" class="ted-btn is-sm" @click="moreOpen = true">{{ t('theme.editor.mobileMore') }}</button>
          </div>
          <p class="ted-panel__foot">
            {{ t('theme.editor.mobileOv', { page: ed.pageName.value, n: ed.pageOvCount.value }) }}
          </p>
        </section>
        <ThemeModal v-model="moreOpen" :title="t('theme.editor.mobileMore')">
          <div class="ted-more">
            <BannerEditor v-if="ed.sel.value === 'hero'" />
            <SizeInspector v-else-if="ed.isSizeSel.value" />
            <EditorSettings :tabs="['bg', 'shape', 'comp']" :show-scope="false" />
          </div>
        </ThemeModal>
      </template>

      <ThemeExportDialog
        v-model="exportOpen"
        :name="ed.name.value || t('theme.editor.defaultName')"
        :content="ed.content.value"
        :theme-id="ed.themeId.value"
        :only-page="exportOnlyPage"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import {
  ArrowDownTrayIcon,
  CheckIcon,
  ChevronDownIcon,
  ComputerDesktopIcon,
  CubeIcon,
  DevicePhoneMobileIcon,
  DocumentDuplicateIcon,
  DocumentTextIcon,
  MoonIcon,
  SunIcon,
  SwatchIcon,
} from '@heroicons/vue/24/outline'
import ColorPicker from '@/components/theme/ColorPicker.vue'
import ThemeModal from '@/components/theme/ThemeModal.vue'
import ThemeExportDialog from '@/components/theme/ThemeExportDialog.vue'
import { themeCssCode } from '@/theme/content'
import { normalizeHex } from '@/theme/color'
import { copyTextToClipboard } from '@/utils/file'
import EditorSettings from './EditorSettings.vue'
import EditorPreview from './EditorPreview.vue'
import EditorInspector from './EditorInspector.vue'
import BannerEditor from './BannerEditor.vue'
import SizeInspector from './SizeInspector.vue'
import { provideThemeEditor } from './useThemeEditor'

defineOptions({ name: 'ThemeEditorView' })

const route = useRoute()
const router = useRouter()
const menuStore = useMenuStore()
const isMobile = computed(() => menuStore.isMobile)

const routeId = computed(() => String(route.params.id || 'new'))
const ed = provideThemeEditor(routeId)
const { t } = ed

const userStore = useUserStore()
onMounted(() => {
  // 编辑器在外壳之外：直接打开（刷新）时自己拉一次账号信息，外观同步才会开始
  if (!userStore.userInfo) void userStore.getUserInfo()
  void ed.load()
  window.addEventListener('keydown', onKeydown)
})
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))

const goBack = () => void router.push({ path: '/profile', query: { tab: 'appearance' } })

const schemes = [
  { key: 'light' as const, icon: SunIcon },
  { key: 'dark' as const, icon: MoonIcon },
]
const devices = [
  { key: 'pc' as const, icon: ComputerDesktopIcon },
  { key: 'mobile' as const, icon: DevicePhoneMobileIcon },
]

// 撤销 / 重做快捷键（输入框里交给浏览器）
const onKeydown = (e: KeyboardEvent) => {
  if (!(e.ctrlKey || e.metaKey)) return
  const el = e.target as HTMLElement | null
  if (el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.isContentEditable)) return
  const k = e.key.toLowerCase()
  if (k === 'z' && !e.shiftKey) {
    e.preventDefault()
    ed.undo()
  } else if ((k === 'z' && e.shiftKey) || k === 'y') {
    e.preventDefault()
    ed.redo()
  }
}

// 导出
const exportOpen = ref(false)
const exportOnlyPage = ref('')
const exportItems = computed(() => [
  { key: 'pkg', icon: CubeIcon, title: t('theme.editor.exportPkg'), sub: t('theme.editor.exportPkgSub') },
  { key: 'page', icon: DocumentTextIcon, title: t('theme.editor.exportPage', { page: ed.pageName.value }), sub: t('theme.editor.exportPageSub') },
  { key: 'code', icon: DocumentDuplicateIcon, title: t('theme.editor.copyCode'), sub: t('theme.editor.copyCodeSub') },
])
const onExport = async (key: string) => {
  if (key === 'code') {
    await copyTextToClipboard(themeCssCode(ed.name.value, ed.content.value))
    feedback.success(t('theme.export.copied'))
    return
  }
  if (key === 'page' && !ed.page.value) {
    feedback.warning(t('theme.editor.exportPageEmpty', { page: ed.pageName.value }))
    return
  }
  exportOnlyPage.value = key === 'page' ? ed.pageKey.value : ''
  exportOpen.value = true
}

// 手机：token 横向标签（配色 + 终端页的终端色 + 欢迎横幅）
const moreOpen = ref(false)
const mobileTokens = computed(() => [
  ...ed.editorTokens,
  ...(ed.isTermPage.value ? ['termBg', 'termFg', 'termCursor'] : []),
  'hero',
])
const onHex = (e: Event) => {
  const input = e.target as HTMLInputElement
  const v = normalizeHex(input.value)
  if (v) ed.setToken(ed.sel.value, v)
  else input.value = ed.selHex.value
}
</script>

<style scoped lang="scss">
.ted {
  position: fixed;
  inset: 0;
  display: flex;
  flex-direction: column;
  background: var(--m-s2);
  color: var(--m-ink);
}
.ted-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  color: var(--m-ink2);
}
.ted-top {
  flex: none;
  display: flex;
  align-items: center;
  gap: 12px;
  height: 68px;
  padding: 0 16px;
  background: var(--m-s);
  border-bottom: 1px solid var(--m-line);
  :deep(.is-off) {
    opacity: 0.4;
    pointer-events: none;
  }
}
.ted-sticker {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 38px;
  height: 38px;
  border-radius: 12px;
  background: var(--m-p);
  color: var(--m-pOn);
  rotate: -6deg;
  svg {
    width: 20px;
    height: 20px;
    stroke-width: 2.1;
  }
}
.ted-title {
  min-width: 0;
  display: flex;
  flex-direction: column;
  small {
    font-size: 0.75rem;
    color: var(--m-ink2);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  b {
    color: var(--m-warn);
    font-weight: 700;
  }
  b.is-clean {
    color: var(--m-ink2);
    font-weight: 500;
  }
}
.ted-name {
  width: 220px;
  max-width: 100%;
  padding: 0;
  border: 0;
  background: transparent;
  font: inherit;
  font-family: var(--app-font-display);
  font-size: 1.0625rem;
  font-weight: 800;
  color: var(--m-ink);
  outline: none;
  &:focus-visible {
    box-shadow: 0 2px 0 var(--m-p);
  }
}
.ted-spacer {
  flex: 1;
}
.ted-seg {
  display: inline-flex;
  flex: none;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  background: var(--m-s2);
  button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    min-height: 36px;
    padding: 0 14px;
    border: 0;
    border-radius: 999px;
    background: transparent;
    font: inherit;
    font-size: 0.8125rem;
    font-weight: 700;
    color: var(--m-ink2);
    white-space: nowrap;
    cursor: pointer;
    svg {
      width: 16px;
      height: 16px;
    }
  }
  .on {
    background: var(--m-s);
    color: var(--m-ink);
    box-shadow: 0 1px 3px rgba(20, 10, 30, 0.12);
  }
}
.ted-seg--fill {
  display: flex;
  button {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}
.ted-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  flex: none;
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
    stroke-width: 2.6;
  }
  &:hover:not(:disabled) {
    background: var(--m-pSoft);
    color: var(--m-pInk);
  }
  &.is-primary {
    background: var(--m-p);
    color: var(--m-pOn);
  }
  &.is-sm {
    min-height: 40px;
    padding: 0 14px;
    font-size: 0.8125rem;
  }
  &:disabled {
    opacity: 0.6;
    cursor: progress;
  }
}
.ted-menu {
  display: flex;
  flex-direction: column;
  padding: 6px;
}
.ted-mi {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 54px;
  padding: 6px 8px;
  border: 0;
  border-radius: 12px;
  background: transparent;
  color: var(--m-ink);
  font: inherit;
  text-align: left;
  cursor: pointer;
  &:hover {
    background: var(--m-s2);
  }
}
.ted-mi__ic {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 34px;
  height: 34px;
  border-radius: 11px;
  svg {
    width: 16px;
    height: 16px;
  }
  &.is-pkg {
    background: var(--m-pSoft);
    color: var(--m-pInk);
  }
  &.is-page {
    background: var(--m-a2Soft);
    color: var(--m-ink);
  }
  &.is-code {
    background: var(--m-aSoft);
    color: var(--m-aOn);
  }
}
.ted-mi__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  b {
    font-size: 0.875rem;
  }
  small {
    font-size: 0.75rem;
    color: var(--m-ink2);
  }
}

/* PC 三栏 */
.ted-body {
  flex: 1;
  min-height: 0;
  display: flex;
}
.ted-left,
.ted-right {
  flex: none;
  overflow-y: auto;
  padding: 16px;
  background: var(--m-s);
  scrollbar-width: none;
  &::-webkit-scrollbar {
    display: none;
  }
}
.ted-left {
  width: 330px;
  border-right: 1px solid var(--m-line);
}
.ted-right {
  width: 300px;
  border-left: 1px solid var(--m-line);
}

/* 手机 */
.ted.is-mobile {
  .ted-top {
    gap: 4px;
    height: auto;
    padding: 12px 10px 8px;
    border-bottom: 0;
  }
  .ted-title {
    flex: 1;
  }
  .ted-name {
    width: 100%;
  }
}
.ted-mpv {
  flex: 1;
  min-height: 0;
}
.ted-panel {
  position: relative;
  z-index: 2;
  flex: none;
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: -14px;
  padding: 8px 16px calc(18px + env(safe-area-inset-bottom));
  border-radius: 24px 24px 0 0;
  background: var(--m-s);
  box-shadow: 0 -12px 30px -18px rgba(20, 10, 30, 0.35);
}
.ted-panel__grip {
  align-self: center;
  width: 40px;
  height: 5px;
  border-radius: 999px;
  background: var(--m-line);
}
.ted-tchips {
  display: flex;
  gap: 6px;
  margin: 0 -16px;
  padding: 0 16px;
  overflow-x: auto;
  scrollbar-width: none;
  &::-webkit-scrollbar {
    display: none;
  }
}
.ted-tchip {
  display: inline-flex;
  align-items: center;
  flex: none;
  gap: 6px;
  min-height: 40px;
  padding: 0 12px 0 8px;
  border-radius: 999px;
  border: 1.5px solid var(--m-line);
  background: var(--m-s);
  color: var(--m-ink);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
  i {
    width: 18px;
    height: 18px;
    border-radius: 6px;
    border: 1px solid rgba(0, 0, 0, 0.1);
  }
  &.on {
    border-color: var(--m-p);
    background: var(--m-pSoft);
  }
}
.ted-tagx {
  display: inline-flex;
  align-items: center;
  height: 18px;
  padding: 0 7px;
  border-radius: 999px;
  background: var(--m-aSoft);
  color: var(--m-aOn);
  font-size: 0.625rem;
  font-weight: 800;
}
.ted-hexrow {
  display: flex;
  align-items: center;
  gap: 8px;
}
.ted-hexrow__hint {
  flex: 1;
  font-size: 0.75rem;
  color: var(--m-ink2);
}
.ted-hex {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  padding: 0 14px;
  border-radius: 999px;
  background: var(--m-s);
  border: 1.5px solid var(--m-line);
  i {
    flex: none;
    width: 22px;
    height: 22px;
    border-radius: 7px;
  }
  input {
    flex: 1;
    min-width: 0;
    border: 0;
    background: transparent;
    font-size: 0.875rem;
    color: var(--m-ink);
    outline: none;
  }
  &:focus-within {
    border-color: var(--m-p);
  }
}
.ted-panel__foot {
  margin: 0;
  font-size: 0.75rem;
  color: var(--m-ink2);
}
.ted-more {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
</style>
