<!-- 文件预览侧栏（原型：文件列表右侧）：文本前若干行（行号 + 键值着色）+ 编辑/下载/分享；图片缩略图 + 查看原图/下载；
     音视频 = 播放；其它类型提示下载。底部信息格：路径 / 修改时间 / 权限。 -->
<template>
  <aside class="fpv" :aria-label="t('fileManager.previewPanel')">
    <header class="fpv__head">
      <span class="fpv__sticker m-sticker" :class="`m-sticker--${tone}`" style="transform: rotate(-5deg)">
        <UIcon :name="icon" />
      </span>
      <span class="fpv__title">
        <span class="fpv__name" :title="row.name">{{ row.name }}</span>
        <span class="fpv__meta m-num">{{ formatFileSize(row.size) }} · {{ kindLabel }}</span>
      </span>
      <AppIconButton icon="HOutline:XMarkIcon" :label="t('fileManager.closePreview')" :box="36" :size="16" @click="emit('close')" />
    </header>

    <!-- 加载中 -->
    <div v-if="loading" class="fpv__skel">
      <span v-for="i in 8" :key="i" class="fpv__skel-line" :style="{ width: `${40 + ((i * 37) % 55)}%` }" />
    </div>

    <!-- 文本 -->
    <template v-else-if="mode === 'text'">
      <div class="fpv__lbl">{{ langLabel }}</div>
      <div class="fpv__code m-mono">
        <div v-for="ln in lines" :key="ln.no" class="fpv__ln">
          <span class="fpv__no m-num">{{ ln.no }}</span>
          <span class="fpv__txt" :class="ln.cls"><span>{{ ln.ind }}</span><span class="fpv__k">{{ ln.k }}</span><span class="fpv__c">{{ ln.c }}</span><span>{{ ln.v }}</span></span>
        </div>
        <div v-if="truncated" class="fpv__more">{{ t('fileManager.previewTruncated', { n: PREVIEW_LINES }) }}</div>
      </div>
      <div class="fpv__acts">
        <UButton class="fpv__grow" color="primary" size="sm" icon="i-lucide-pencil" @click="emit('edit', row)">
          {{ t('fileManager.edit') }}
        </UButton>
        <UButton color="neutral" variant="outline" size="sm" icon="i-lucide-download" @click="emit('download', row)">
          {{ t('fileManager.download') }}
        </UButton>
        <AppIconButton v-if="canShare" icon="HOutline:ShareIcon" :label="t('fileManager.share')" :box="36" :size="16" @click="emit('share', row)" />
      </div>
    </template>

    <!-- 图片 -->
    <template v-else-if="mode === 'image'">
      <div class="fpv__img" role="img" :aria-label="row.name">
        <img v-if="url" :src="url" :alt="row.name" @load="onImgLoad" />
        <span v-if="dim" class="fpv__dim m-num">{{ dim }}</span>
      </div>
      <div class="fpv__acts">
        <UButton class="fpv__grow" color="primary" size="sm" icon="i-lucide-eye" @click="emit('media', row)">
          {{ t('fileManager.viewOriginal') }}
        </UButton>
        <UButton color="neutral" variant="outline" size="sm" icon="i-lucide-download" @click="emit('download', row)">
          {{ t('fileManager.download') }}
        </UButton>
        <AppIconButton v-if="canShare" icon="HOutline:ShareIcon" :label="t('fileManager.share')" :box="36" :size="16" @click="emit('share', row)" />
      </div>
    </template>

    <!-- 音视频 -->
    <template v-else-if="mode === 'media'">
      <div class="fpv__none">
        <UIcon :name="icon" class="fpv__none-ico" />
        <span class="fpv__none-title">{{ t('fileManager.mediaPreviewHint') }}</span>
      </div>
      <div class="fpv__acts">
        <UButton class="fpv__grow" color="primary" size="sm" icon="i-lucide-play" @click="emit('media', row)">
          {{ t('fileManager.play') }}
        </UButton>
        <UButton color="neutral" variant="outline" size="sm" icon="i-lucide-download" @click="emit('download', row)">
          {{ t('fileManager.download') }}
        </UButton>
      </div>
    </template>

    <!-- 不支持 -->
    <template v-else>
      <div class="fpv__none">
        <UIcon :name="icon" class="fpv__none-ico" />
        <span class="fpv__none-title">{{ noneTitle }}</span>
        <span class="fpv__none-desc">{{ t('fileManager.previewUnsupportedHint') }}</span>
      </div>
      <div class="fpv__acts">
        <UButton class="fpv__grow" color="primary" size="sm" icon="i-lucide-download" @click="emit('download', row)">
          {{ t('fileManager.download') }}
        </UButton>
        <AppIconButton v-if="canShare" icon="HOutline:ShareIcon" :label="t('fileManager.share')" :box="36" :size="16" @click="emit('share', row)" />
      </div>
    </template>

    <dl class="fpv__info">
      <div class="fpv__info-wide">
        <dt>{{ t('fileManager.itemPath') }}</dt>
        <dd class="m-mono" :title="row.path">{{ row.path }}</dd>
      </div>
      <div>
        <dt>{{ t('fileManager.updatedAt') }}</dt>
        <dd class="m-num">{{ formatDateTime(row.updateTime) }}</dd>
      </div>
      <div>
        <dt>{{ t('fileManager.permission') }}</dt>
        <dd class="m-mono" :title="owner">{{ permText }}</dd>
      </div>
    </dl>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  detectMonacoLanguage,
  formatDateTime,
  formatFileSize,
  isFileTooLargeForEditor,
  resolveFilePreviewKind,
  resolvePreSignedFileUrl,
} from '@/utils/file'
import type { FileEntryInfo } from '@/types/v1/file'
import type { FileClient } from './types'

const props = defineProps<{
  client: FileClient
  row: FileEntryInfo
  icon: string
  tone: string
  kindLabel: string
  canShare?: boolean
}>()
const emit = defineEmits<{
  close: []
  edit: [row: FileEntryInfo]
  download: [row: FileEntryInfo]
  share: [row: FileEntryInfo]
  media: [row: FileEntryInfo]
}>()

const { t } = useI18n()
const PREVIEW_LINES = 200

type Mode = 'text' | 'image' | 'media' | 'none' | 'large'
const mode = ref<Mode>('none')
const loading = ref(false)
const url = ref('')
const dim = ref('')
type Line = { no: number; ind: string; k: string; c: string; v: string; cls: string }
const lines = ref<Line[]>([])
const truncated = ref(false)
const language = ref('plaintext')

const LANG_LABEL: Record<string, string> = {
  yaml: 'YAML',
  json: 'JSON',
  ini: 'INI',
  markdown: 'Markdown',
  javascript: 'JavaScript',
  typescript: 'TypeScript',
  shell: 'Shell',
  python: 'Python',
  go: 'Go',
  html: 'HTML',
  css: 'CSS',
  xml: 'XML',
  sql: 'SQL',
}
const langLabel = computed(() => {
  const lang = LANG_LABEL[language.value] ?? (language.value === 'plaintext' ? t('fileManager.plainText') : language.value.toUpperCase())
  return `${lang} · ${t('fileManager.lineCount', { n: lines.value.length })}`
})
const noneTitle = computed(() =>
  mode.value === 'large' ? t('fileManager.previewTooLarge') : t('fileManager.previewUnsupported'),
)
const owner = computed(() => [props.row.userName, props.row.groupName].filter(Boolean).join(':'))
const permText = computed(() => [props.row.permission, owner.value].filter(Boolean).join(' ') || '—')

// 简单着色：注释行 / Markdown 标题 / 「键: 值」「键=值」（YAML、INI、.env、JSON）。
const KEY_RE = /^(\s*)("?[\w@./-]+"?)(\s*[:=])(.*)$/
const parseLine = (text: string, i: number): Line => {
  const base: Line = { no: i + 1, ind: '', k: '', c: '', v: text, cls: '' }
  const trimmed = text.trim()
  if (language.value === 'markdown') return { ...base, cls: trimmed.startsWith('#') ? 'is-head' : '' }
  if (trimmed.startsWith('#') || trimmed.startsWith('//') || trimmed.startsWith(';')) return { ...base, cls: 'is-comment' }
  if (!['yaml', 'ini', 'json', 'plaintext', 'shell'].includes(language.value)) return base
  const m = text.match(KEY_RE)
  if (!m) return base
  return { no: i + 1, ind: m[1] ?? '', k: m[2] ?? '', c: m[3] ?? '', v: m[4] ?? '', cls: '' }
}

let seq = 0
const load = async () => {
  const row = props.row
  const id = ++seq
  url.value = ''
  dim.value = ''
  lines.value = []
  truncated.value = false
  const kind = resolveFilePreviewKind(row.name)
  if (kind === 'image') {
    mode.value = 'image'
    try {
      const path = await props.client.preSignDownload(row.path, true)
      if (id === seq) url.value = resolvePreSignedFileUrl(path)
    } catch {
      if (id === seq) mode.value = 'none'
    }
    return
  }
  if (kind === 'video' || kind === 'audio') {
    mode.value = 'media'
    return
  }
  if (isFileTooLargeForEditor(row.path, row.size)) {
    mode.value = 'large'
    return
  }
  loading.value = true
  try {
    const res = await props.client.open(row.path)
    if (id !== seq) return
    if (res.isBinary) {
      mode.value = 'none'
      return
    }
    language.value = detectMonacoLanguage(row.name)
    const all = res.text.replace(/\r\n/g, '\n').split('\n')
    if (all.length > 1 && all[all.length - 1] === '') all.pop()
    truncated.value = all.length > PREVIEW_LINES
    lines.value = all.slice(0, PREVIEW_LINES).map(parseLine)
    mode.value = 'text'
  } catch {
    if (id === seq) mode.value = 'none'
  } finally {
    if (id === seq) loading.value = false
  }
}

const onImgLoad = (event: Event) => {
  const img = event.target as HTMLImageElement
  if (img.naturalWidth) dim.value = `${img.naturalWidth} × ${img.naturalHeight}`
}

watch(() => props.row.path, load, { immediate: true })
</script>

<style scoped lang="scss">
.fpv {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  padding: 18px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
}
.fpv__head {
  display: flex;
  align-items: center;
  gap: 12px;
}
.fpv__sticker {
  flex: none;
  width: 44px;
  height: 44px;
  border-radius: 15px;
}
.fpv__sticker :deep(svg),
.fpv__sticker :deep(.iconify) {
  width: 20px;
  height: 20px;
}
.fpv__title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.fpv__name {
  font-size: 1rem;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fpv__meta {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.fpv__lbl {
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--el-text-color-secondary);
}
.fpv__code {
  max-height: 400px;
  overflow: auto;
  padding: 12px 12px 12px 8px;
  border-radius: var(--app-radius);
  background: var(--m-s2);
  font-size: 0.75rem;
  line-height: 1.7;
  scrollbar-width: none;
}
.fpv__code::-webkit-scrollbar {
  display: none;
}
.fpv__ln {
  display: flex;
  gap: 10px;
  white-space: pre;
}
.fpv__no {
  flex: none;
  width: 22px;
  text-align: right;
  color: var(--el-text-color-secondary);
}
.fpv__k {
  color: var(--m-pInk);
  font-weight: 600;
}
.fpv__c {
  color: var(--el-text-color-secondary);
}
.fpv__txt.is-comment {
  color: var(--el-text-color-secondary);
}
.fpv__txt.is-head {
  color: var(--m-pInk);
  font-weight: 700;
}
.fpv__more {
  margin-top: 6px;
  font-family: inherit;
  color: var(--el-text-color-secondary);
}
.fpv__acts {
  display: flex;
  align-items: center;
  gap: 8px;
}
.fpv__acts :deep(button) {
  min-height: 36px;
  border-radius: 999px;
  font-weight: 700;
}
.fpv__grow {
  flex: 1;
  justify-content: center;
}
.fpv__img {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
  overflow: hidden;
  border-radius: 18px;
  background: var(--m-a2Soft);
}
.fpv__img img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}
.fpv__dim {
  position: absolute;
  left: 12px;
  bottom: 12px;
  padding: 4px 10px;
  border-radius: 999px;
  background: var(--el-bg-color);
  font-size: 0.75rem;
  font-weight: 700;
}
.fpv__none {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 28px 16px;
  border-radius: 18px;
  background: var(--m-s2);
  text-align: center;
}
.fpv__none-ico {
  width: 32px;
  height: 32px;
  color: var(--el-text-color-secondary);
}
.fpv__none-title {
  font-size: 0.875rem;
  font-weight: 700;
}
.fpv__none-desc {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.fpv__skel {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  border-radius: var(--app-radius);
  background: var(--m-s2);
}
.fpv__skel-line {
  height: 10px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.fpv__info {
  display: grid;
  /* 修改时间（含秒）比权限串长，左列给宽一些，避免被截断 */
  grid-template-columns: minmax(0, 1.5fr) minmax(0, 1fr);
  gap: 10px 12px;
  margin: 0;
  padding: 12px 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
}
.fpv__info > div {
  min-width: 0;
}
.fpv__info-wide {
  grid-column: 1 / -1;
}
.fpv__info dt {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.fpv__info dd {
  margin: 2px 0 0;
  font-size: 0.8125rem;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fpv__info dd.m-mono {
  font-size: 0.75rem;
}
</style>
