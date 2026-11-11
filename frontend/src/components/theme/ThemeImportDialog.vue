<!-- 导入主题包（ThemeImportPC / ThemeImportMobile）：选择或拖入 .momoko-theme → 服务端校验并转换图片 →
     预览（工作台 / 登录页 / 终端 / 手机 × 浅色 / 深色，含主题包的页面单独设置）→ 检查结果 → 同名主题另存副本或覆盖 →
     只导入 / 导入并使用；也可以「在整个界面里预览」临时套用（不保存）。 -->
<template>
  <ThemeModal :model-value="modelValue" :title="t('theme.import.title')" :width="600" @update:model-value="emit('update:modelValue', $event)">
    <div class="tim">
      <!-- 文件 -->
      <label
        v-if="!info && !loading"
        class="tim-drop"
        :class="{ 'is-drop': dragging }"
        @dragover.prevent="dragging = true"
        @dragleave="dragging = false"
        @drop.prevent="onDrop"
      >
        <input type="file" accept=".momoko-theme,.zip,application/zip" class="tim-file" @change="onPick" />
        <span class="tim-sticker"><CubeIcon aria-hidden="true" /></span>
        <b>{{ t('theme.import.choose') }}</b>
        <small>{{ error || t('theme.import.dropHint') }}</small>
      </label>
      <div v-else class="tim-filerow">
        <span class="tim-sticker"><CubeIcon aria-hidden="true" /></span>
        <span class="tim-filerow__text">
          <b class="m-mono">{{ file?.name }}</b>
          <small>{{ loading ? t('theme.import.reading') : t('theme.import.read', { size: formatFileSize(file?.size || 0) }) }}</small>
        </span>
        <label class="tim-btn tim-btn--sm">
          <input type="file" accept=".momoko-theme,.zip,application/zip" class="tim-file" @change="onPick" />
          {{ t('theme.import.change') }}
        </label>
      </div>

      <template v-if="info && content">
        <!-- 预览 -->
        <ThemeScene
          class="tim-pv"
          :class="{ 'is-short': isMobile }"
          :content="content"
          :scheme="pvDark ? 'dark' : 'light'"
          :variant="pvPage"
          :page-key="pvPageKey"
          :resolver="resolver"
        >
          <div class="tim-pv__chips">
            <button v-for="p in pvPages" :key="p" type="button" :class="{ on: pvPage === p }" @click="pvPage = p">
              {{ t(`theme.import.pages.${p}`) }}
            </button>
            <span class="tim-pv__sep" />
            <button v-for="d in [false, true]" :key="String(d)" type="button" :class="{ on: pvDark === d }" @click="pvDark = d">
              {{ d ? t('theme.dark') : t('theme.light') }}
            </button>
          </div>
          <button type="button" class="tim-pv__full" @click="previewAll">
            <EyeIcon aria-hidden="true" />{{ isMobile ? t('theme.import.fullShort') : t('theme.import.fullPreview') }}
          </button>
        </ThemeScene>

        <!-- 信息 -->
        <div class="tim-meta">
          <div class="tim-meta__title">
            <b>{{ info.name }}</b>
            <small>{{ metaLine }}</small>
          </div>
          <p v-if="info.description">{{ info.description }}</p>
          <div class="tim-tags">
            <span v-for="tag in tags" :key="tag">{{ tag }}</span>
          </div>
        </div>

        <!-- 检查 -->
        <ul class="tim-checks">
          <li v-for="(c, i) in checkLines" :key="i" :class="{ 'is-warn': c.warn }">
            <component :is="c.warn ? ExclamationTriangleIcon : CheckIcon" aria-hidden="true" />
            <span>{{ c.text }}</span>
          </li>
        </ul>

        <!-- 同名 -->
        <div v-if="info.conflictThemeId" class="tim-conflict">
          <span>{{ t('theme.import.conflict', { name: info.name, version: info.conflictVersion || '1.0' }) }}</span>
          <div class="tim-seg">
            <button v-for="c in ['copy', 'overwrite'] as const" :key="c" type="button" :class="{ on: conflict === c }" @click="conflict = c">
              {{ t(`theme.import.conflicts.${c}`) }}
            </button>
          </div>
        </div>
      </template>
    </div>

    <template #footer>
      <button type="button" class="tim-btn" @click="emit('update:modelValue', false)">{{ t('common.cancel') }}</button>
      <span class="tim-spacer" />
      <button type="button" class="tim-btn" :disabled="!info || busy" @click="doImport(false)">{{ t('theme.import.onlyImport') }}</button>
      <button type="button" class="tim-btn is-primary" :disabled="!info || busy" @click="doImport(true)">
        {{ t('theme.import.importAndUse') }}
      </button>
    </template>
  </ThemeModal>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { CheckIcon, CubeIcon, ExclamationTriangleIcon, EyeIcon } from '@heroicons/vue/24/outline'
import type { ThemePackageInfo } from '@/types/v1/theme'
import { fileToBase64, importThemePackage, inspectThemePackage } from '@/api/theme'
import { normalizeContent, resolveTokens, contrastChecks, DEFAULT_RADIUS, pagesWithOverrides, type AssetResolver } from '@/theme/content'
import { formatRatio } from '@/theme/color'
import { useThemePages } from '@/theme/pages'
import { formatFileSize } from '@/utils/file'
import ThemeModal from './ThemeModal.vue'
import ThemeScene from './ThemeScene.vue'

const props = defineProps<{ modelValue: boolean; initialFile?: File | null }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const { t } = useI18n()
const themeStore = useThemeStore()
const menuStore = useMenuStore()
const isMobile = computed(() => menuStore.isMobile)
const { labelOf, siteThemeKey } = useThemePages()

const file = ref<File | null>(null)
const payload = ref('')
const info = ref<ThemePackageInfo | null>(null)
const loading = ref(false)
const busy = ref(false)
const error = ref('')
const dragging = ref(false)
const conflict = ref<'copy' | 'overwrite'>('copy')
const pvPage = ref<'dash' | 'login' | 'term' | 'phone'>('dash')
const pvDark = ref(true)

const pvPages = computed(() => (isMobile.value ? (['dash', 'login', 'term'] as const) : (['dash', 'login', 'term', 'phone'] as const)))
// 预览「登录页」取访客页面（路由 meta.siteTheme）的单独设置；「终端」取主题包里带终端色的那一页（没有就用全局）
const termPageKey = computed(() => {
  const pages = info.value?.content?.pages || {}
  return (
    Object.keys(pages).find((k) =>
      [...Object.keys(pages[k]?.tokensLight || {}), ...Object.keys(pages[k]?.tokensDark || {})].some((t) => t.startsWith('term') || t.startsWith('ansi')),
    ) ?? null
  )
})
const pvPageKey = computed(() => (pvPage.value === 'login' ? siteThemeKey.value : pvPage.value === 'term' ? termPageKey.value : null))

const content = computed(() => (info.value?.content ? normalizeContent(info.value.content) : null))
const imageMap = computed(() => new Map((info.value?.images || []).map((im) => [im.path, im.dataUrl])))
const resolver: AssetResolver = (ref) => imageMap.value.get(ref) || ''

const reset = () => {
  file.value = null
  payload.value = ''
  info.value = null
  error.value = ''
  conflict.value = 'copy'
  pvPage.value = 'dash'
}

const inspect = async (f: File) => {
  dragging.value = false
  if (!/\.(momoko-theme|zip)$/i.test(f.name)) {
    error.value = t('theme.import.invalidFile')
    return
  }
  reset()
  file.value = f
  loading.value = true
  try {
    payload.value = await fileToBase64(f)
    const { data } = await inspectThemePackage({ fileName: f.name, data: payload.value })
    info.value = data?.info ?? null
    if (info.value?.conflictThemeId) conflict.value = 'copy'
  } catch {
    file.value = null
    payload.value = ''
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.modelValue, props.initialFile] as const,
  ([open, f]) => {
    if (open && f && f !== file.value) void inspect(f)
    if (!open && !themeStore.preview) reset()
  },
  { immediate: true },
)

const onPick = (e: Event) => {
  const input = e.target as HTMLInputElement
  const f = input.files?.[0]
  input.value = ''
  if (f) void inspect(f)
}
const onDrop = (e: DragEvent) => {
  const f = e.dataTransfer?.files?.[0]
  if (f) void inspect(f)
}

const metaLine = computed(() => {
  const i = info.value
  if (!i) return ''
  const parts = [] as string[]
  if (i.author) parts.push(t('theme.import.author', { name: i.author }))
  parts.push(t('theme.import.version', { v: i.version || '1.0' }))
  parts.push(t('theme.import.format', { n: i.format || 1 }))
  return parts.join(' · ')
})

const tags = computed(() => {
  const c = content.value
  const i = info.value
  if (!c || !i) return []
  const out: string[] = []
  const light = Object.keys(c.tokensLight).length > 0
  const dark = Object.keys(c.tokensDark).length > 0
  out.push(light && dark ? t('theme.import.tagTokensBoth') : dark ? t('theme.import.tagTokensDark') : light ? t('theme.import.tagTokensLight') : t('theme.import.tagTokensBase', { name: t(`theme.palettes.${c.base}`) }))
  if (i.images.length) out.push(t('theme.import.tagImages', { n: i.images.length }))
  const pages = pagesWithOverrides(c)
  if (pages.length) out.push(t('theme.import.tagPages', { pages: pages.map((k) => labelOf(k)).join(t('theme.listSep')) }))
  const r = c.shape?.radius
  out.push(t('theme.import.tagRadius', { v: r == null || r === DEFAULT_RADIUS ? t('theme.radiusRound') : `${r} px` }))
  const sizeCount = Object.keys(c.sizes || {}).length + Object.values(c.pages).reduce((n, p) => n + Object.keys(p.sizes || {}).length, 0)
  if (sizeCount) out.push(t('theme.import.tagSizes', { n: sizeCount }))
  const fonts = Object.keys(i.fonts || {})
  if (fonts.length) out.push(t('theme.import.tagFont', { name: c.shape?.fontName || fonts[0] }))
  return out
})

const checkLines = computed(() => {
  const i = info.value
  const c = content.value
  if (!i || !c) return []
  const lines: { text: string; warn: boolean }[] = i.checks.map((ck) => {
    const p = { ...ck.params }
    if (ck.code === 'image_converted') {
      p.from = formatFileSize(Number(p.from || 0))
      p.to = formatFileSize(Number(p.to || 0))
    }
    if (ck.code === 'font_ok') p.size = formatFileSize(Number(p.size || 0))
    return { text: t(`theme.import.checks.${ck.code}`, p), warn: ck.level === 'warn' }
  })
  for (const scheme of ['light', 'dark'] as const) {
    for (const ck of contrastChecks(resolveTokens(c, scheme))) {
      if (ck.ok) continue
      lines.push({
        text: isMobile.value
          ? t('theme.import.contrastShort', { mode: t(`theme.${scheme}`), item: t(`theme.checks.${ck.key}`) })
          : t('theme.import.contrast', { mode: t(`theme.${scheme}`), item: t(`theme.checks.${ck.key}`), ratio: formatRatio(ck.ratio) }),
        warn: true,
      })
    }
  }
  return lines
})

const doImport = async (use: boolean) => {
  if (!file.value || !payload.value || busy.value) return
  busy.value = true
  try {
    const { data } = await importThemePackage({
      fileName: file.value.name,
      data: payload.value,
      conflict: info.value?.conflictThemeId ? conflict.value : 'copy',
    })
    const theme = data?.theme
    if (!theme) return
    themeStore.upsertTheme(theme)
    if (use) themeStore.setTheme(theme.id, theme)
    feedback.success(t('theme.import.imported', { name: theme.name }))
    emit('update:modelValue', false)
    reset()
  } finally {
    busy.value = false
  }
}

/** 在整个界面里预览：临时套用，悬浮条上可直接「导入并使用」 */
const previewAll = () => {
  if (!info.value || !content.value) return
  const map = new Map(imageMap.value)
  themeStore.startPreview({
    name: info.value.name,
    content: content.value,
    scheme: pvDark.value ? 'dark' : 'light',
    from: 'import',
    resolver: (ref) => map.get(ref) || '',
    accept: () => doImport(true),
  })
  emit('update:modelValue', false)
}
</script>

<style scoped lang="scss">
.tim {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.tim-file {
  position: absolute;
  width: 1px;
  height: 1px;
  opacity: 0;
  pointer-events: none;
}
.tim-sticker {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: 12px;
  background: var(--m-p);
  color: var(--m-pOn);
  rotate: -4deg;
  svg {
    width: 20px;
    height: 20px;
  }
}
.tim-drop {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 32px 16px;
  border-radius: 20px;
  border: 1.5px dashed var(--m-p);
  background: var(--m-pSoft);
  text-align: center;
  cursor: pointer;
  b {
    font-size: 0.9375rem;
  }
  small {
    font-size: 0.8125rem;
    color: var(--m-ink2);
  }
  &.is-drop {
    box-shadow: 0 0 0 4px var(--m-pSoft);
  }
  &:focus-within {
    box-shadow: var(--app-focus-ring);
  }
}
.tim-filerow {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border-radius: 16px;
  border: 1.5px dashed var(--m-p);
  background: var(--m-pSoft);
}
.tim-filerow__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  b {
    font-size: 0.8125rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  small {
    font-size: 0.75rem;
    color: var(--m-ink2);
  }
}
.tim-pv {
  height: 230px;
  border-radius: 18px;
  &.is-short {
    height: 200px;
  }
}
.tim-pv__chips {
  position: absolute;
  left: 12px;
  top: 12px;
  right: 12px;
  display: flex;
  align-items: center;
  gap: 2px;
  width: max-content;
  max-width: calc(100% - 24px);
  padding: 4px;
  border-radius: 999px;
  background: rgba(15, 10, 25, 0.62);
  -webkit-backdrop-filter: blur(8px);
  backdrop-filter: blur(8px);
  overflow-x: auto;
  scrollbar-width: none;
  &::-webkit-scrollbar {
    display: none;
  }
  button {
    flex: none;
    min-height: 28px;
    padding: 0 11px;
    border: 0;
    border-radius: 999px;
    background: transparent;
    color: rgba(255, 255, 255, 0.78);
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
.tim-pv__sep {
  flex: none;
  width: 1px;
  height: 18px;
  margin: 0 4px;
  background: rgba(255, 255, 255, 0.3);
}
.tim-pv__full {
  position: absolute;
  right: 12px;
  bottom: 12px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 36px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: #ffffff;
  color: #2a1630;
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 600;
  box-shadow: 0 6px 18px -6px rgba(0, 0, 0, 0.5);
  cursor: pointer;
  svg {
    width: 16px;
    height: 16px;
  }
}
.tim-meta {
  display: flex;
  flex-direction: column;
  gap: 6px;
  p {
    margin: 0;
    font-size: 0.8125rem;
    line-height: 1.5;
  }
}
.tim-meta__title {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 4px 10px;
  b {
    font-family: var(--app-font-display);
    font-size: 1.125rem;
    font-weight: 800;
  }
  small {
    font-size: 0.75rem;
    color: var(--m-ink2);
  }
}
.tim-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 2px;
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
.tim-checks {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 0;
  padding: 12px 14px;
  list-style: none;
  border-radius: 16px;
  background: var(--m-s2);
  font-size: 0.8125rem;
  li {
    display: flex;
    align-items: flex-start;
    gap: 8px;
  }
  svg {
    flex: none;
    width: 16px;
    height: 16px;
    margin-top: 2px;
    color: var(--m-ok);
    stroke-width: 2.4;
  }
  .is-warn {
    color: var(--m-warn);
    svg {
      color: var(--m-warn);
    }
  }
}
.tim-conflict {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  > span {
    flex: 1 1 220px;
    font-size: 0.8125rem;
  }
}
.tim-seg {
  display: inline-flex;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  background: var(--m-s2);
  button {
    min-height: 32px;
    padding: 0 12px;
    border: 0;
    border-radius: 999px;
    background: transparent;
    font: inherit;
    font-size: 0.75rem;
    font-weight: 700;
    color: var(--m-ink2);
    cursor: pointer;
  }
  .on {
    background: var(--m-s);
    color: var(--m-ink);
    box-shadow: 0 1px 3px rgba(20, 10, 30, 0.12);
  }
}
.tim-spacer {
  flex: 1;
}
.tim-btn {
  position: relative;
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
  &:hover:not(:disabled) {
    background: var(--m-pSoft);
    color: var(--m-pInk);
  }
  &.is-primary {
    background: var(--m-p);
    color: var(--m-pOn);
  }
  &:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }
}
.tim-btn--sm {
  min-height: 36px;
  padding: 0 14px;
  font-size: 0.8125rem;
}
@media (width < 992px) {
  .tim-btn {
    padding: 0 14px;
  }
}
</style>
