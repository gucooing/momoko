<template>
  <FileDialog v-model="visible" :title="name" :width="1080" @close="onClose">
    <div class="fv-body">
      <img v-if="kind === 'image'" :src="previewUrl" :alt="name" class="fv-media" />
      <video v-else-if="kind === 'video'" :src="previewUrl" class="fv-media" controls autoplay />
      <audio v-else-if="kind === 'audio'" :src="previewUrl" class="fv-audio" controls autoplay />
      <div v-else-if="loading" class="fv-state">{{ t('file.share.loading') }}</div>
      <div v-else-if="unpreviewable" class="fv-state">
        <el-icon class="fv-state-icon"><IconWarning /></el-icon>
        <p>{{ t('fileManager.cannotEditBinary') }}</p>
      </div>
      <div v-else ref="monacoEl" class="fv-monaco"></div>
    </div>

    <template #footer>
      <span class="fv-lang">{{ statusText }}</span>
      <a class="fm-btn" :href="downloadUrl" :download="name" target="_blank" rel="noopener">
        <el-icon><IconDownload /></el-icon>{{ t('fileManager.download') }}
      </a>
      <button type="button" class="fm-btn fm-btn--primary" @click="visible = false">
        {{ t('system.common.confirm') }}
      </button>
    </template>
  </FileDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useThemeStore } from '@/stores/theme'
import { isBinaryBytes } from '@/utils/fileEncoding'
import { detectMonacoLanguage, resolveFilePreviewKind } from '@/utils/file'
import { monaco } from './monaco'
import FileDialog from './FileDialog.vue'
import { IconDownload, IconWarning } from './icons'

// 只读文件查看器：与编辑器“打开文件”一致（文本走 Monaco 高亮、媒体内联预览），
// 但仅查看 + 下载，不能编辑。用于公开分享页等只读场景。
const MAX_PREVIEW_SIZE = 5 * 1024 * 1024 // 超过则不在线预览，仅下载

const props = defineProps<{
  modelValue: boolean
  name: string
  previewUrl: string
  downloadUrl: string
  size?: number
}>()

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const { t } = useI18n()
const themeStore = useThemeStore()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

const kind = computed(() => resolveFilePreviewKind(props.name))
const loading = ref(false)
const unpreviewable = ref(false)
const language = ref('plaintext')

const statusText = computed(() => {
  if (kind.value) return ''
  if (unpreviewable.value || loading.value) return ''
  return language.value === 'plaintext' ? t('fileManager.plainText') : language.value.toUpperCase()
})

const monacoEl = ref<HTMLElement | null>(null)
let editorInstance: monaco.editor.IStandaloneCodeEditor | null = null

const disposeEditor = () => {
  editorInstance?.dispose()
  editorInstance = null
}

const showText = async (text: string) => {
  language.value = detectMonacoLanguage(props.name)
  await nextTick()
  if (!monacoEl.value) return
  editorInstance = monaco.editor.create(monacoEl.value, {
    value: text,
    language: language.value,
    theme: themeStore.isDarkTheme ? 'vs-dark' : 'vs',
    readOnly: true,
    domReadOnly: true,
    automaticLayout: true,
    fontSize: 13,
    lineHeight: 20,
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    fontFamily: "'JetBrains Mono', 'Cascadia Code', Consolas, 'Courier New', monospace",
  })
}

const loadPreview = async () => {
  disposeEditor()
  unpreviewable.value = false
  loading.value = false
  language.value = 'plaintext'

  // 媒体由 <img>/<video>/<audio> 直接内联，无需拉取
  if (kind.value) return

  if (props.size !== undefined && props.size > MAX_PREVIEW_SIZE) {
    unpreviewable.value = true
    return
  }

  loading.value = true
  try {
    const res = await fetch(props.previewUrl)
    if (!res.ok) throw new Error('preview failed')
    const bytes = new Uint8Array(await res.arrayBuffer())
    if (isBinaryBytes(bytes)) {
      unpreviewable.value = true
      return
    }
    const text = new TextDecoder('utf-8', { fatal: false }).decode(bytes)
    loading.value = false
    await showText(text)
  } catch {
    unpreviewable.value = true
  } finally {
    loading.value = false
  }
}

watch(
  () => themeStore.isDarkTheme,
  (dark) => monaco.editor.setTheme(dark ? 'vs-dark' : 'vs'),
)

watch(visible, (open) => {
  if (open) loadPreview()
  else disposeEditor()
})

const onClose = () => disposeEditor()

onBeforeUnmount(disposeEditor)
</script>

<style scoped>
.fv-body {
  height: 70vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--fm-subtle);
  border-radius: var(--fm-radius-sm);
  overflow: hidden;
}
.fv-media {
  max-width: 100%;
  max-height: 70vh;
  object-fit: contain;
}
.fv-audio {
  width: 70%;
}
.fv-monaco {
  width: 100%;
  height: 100%;
}
.fv-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  color: var(--fm-text-3);
  font-size: 13px;
}
.fv-state-icon {
  font-size: 32px;
}
.fv-lang {
  flex: 1;
  font-size: 12px;
  color: var(--fm-accent);
  font-weight: 600;
}
</style>
