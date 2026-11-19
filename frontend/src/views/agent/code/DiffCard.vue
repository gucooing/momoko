<!-- 一次工具调用对一个文件的改动。新旧文本原样来自 ACP，比对交给 Monaco 的 diff 编辑器（只折叠显示改动附近），
     增删行数也由它算出来。滚到看得见时才建编辑器，对话里有再多改动也只建看得到的那些。 -->
<template>
  <div ref="root" class="agc-diffcard">
    <div class="agc-diffcard__head">
      <UIcon :name="created ? 'i-lucide-file-plus' : 'i-lucide-pencil'" class="ag-muted" />
      <span class="agc-tool__verb">{{ created ? t('agent.code.tool.create') : t('agent.code.tool.edit') }}</span>
      <span class="ag-mono ag-ellipsis agc-tool__arg">{{ path }}</span>
      <template v-if="counted">
        <span class="agc-plus ag-num">+{{ added }}</span>
        <span class="agc-minus ag-num">−{{ deleted }}</span>
      </template>
      <button type="button" class="ag-btn ag-btn--sm agc-diffcard__btn" @click="emit('showDiff', path)">
        {{ t('agent.code.fullDiff') }}
      </button>
    </div>
    <div ref="host" class="agc-diffcard__body" :style="{ height: `${height}px` }" />
  </div>
</template>

<script setup lang="ts">
import { useIntersectionObserver } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { monaco } from '@/components/file/monaco'
import { useThemeStore } from '@/stores/theme'
import { detectMonacoLanguage } from '@/utils/file'
import type { AgentDiff } from './types'

const props = defineProps<{ diff: AgentDiff; path: string }>()
const emit = defineEmits<{ showDiff: [path: string] }>()
const { t } = useI18n()
const themeStore = useThemeStore()

const LINE = 18
const MAX_LINES = 14

const root = ref<HTMLElement | null>(null)
const host = ref<HTMLElement | null>(null)
const created = computed(() => props.diff.oldText == null)
const added = ref(0)
const deleted = ref(0)
const counted = ref(false)
const height = ref(LINE * 4)

let editor: monaco.editor.IStandaloneDiffEditor | null = null
let models: monaco.editor.ITextModel[] = []

const build = () => {
  if (editor || !host.value) return
  const language = detectMonacoLanguage(props.path)
  const original = monaco.editor.createModel(props.diff.oldText ?? '', language)
  const modified = monaco.editor.createModel(props.diff.newText, language)
  models = [original, modified]
  editor = monaco.editor.createDiffEditor(host.value, {
    readOnly: true,
    renderSideBySide: false,
    automaticLayout: true,
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    lineNumbers: 'off',
    glyphMargin: false,
    folding: false,
    renderOverviewRuler: false,
    hideUnchangedRegions: { enabled: true, contextLineCount: 2, minimumLineCount: 3, revealLineCount: 10 },
    fontSize: 12,
    lineHeight: LINE,
    scrollbar: { alwaysConsumeMouseWheel: false },
    theme: themeStore.isDarkTheme ? 'vs-dark' : 'vs',
  })
  editor.setModel({ original, modified })
  editor.onDidUpdateDiff(() => {
    let plus = 0
    let minus = 0
    for (const c of editor?.getLineChanges() ?? []) {
      if (c.modifiedEndLineNumber > 0) plus += c.modifiedEndLineNumber - c.modifiedStartLineNumber + 1
      if (c.originalEndLineNumber > 0) minus += c.originalEndLineNumber - c.originalStartLineNumber + 1
    }
    added.value = plus
    deleted.value = minus
    counted.value = true
    const lines = editor?.getModifiedEditor().getContentHeight() ?? LINE * 4
    height.value = Math.min(lines, LINE * MAX_LINES)
  })
}

const { stop } = useIntersectionObserver(root, ([entry]) => {
  if (entry?.isIntersecting) {
    build()
    stop()
  }
})

watch(
  () => props.diff,
  (diff) => {
    if (!editor || models.length < 2) return
    models[0]!.setValue(diff.oldText ?? '')
    models[1]!.setValue(diff.newText)
  },
)
watch(() => themeStore.isDarkTheme, (dark) => monaco.editor.setTheme(dark ? 'vs-dark' : 'vs'))
onBeforeUnmount(() => {
  stop()
  editor?.dispose()
  for (const m of models) m.dispose()
  editor = null
  models = []
})
</script>
