<!-- 一次工具调用对一个文件的改动（ACP v1：新旧文本）。平时是一行「已编辑 文件名 +增 −删」，增删行数在页面上按行比出来；
     点开才建 Monaco 的 diff 编辑器（只折叠显示改动附近），收起就销毁，对话里有再多改动也只建点开的那些。 -->
<template>
  <div class="agc-edit">
    <button v-if="!bare" type="button" class="agc-edit__row" :aria-expanded="isOpen" :title="path" @click="open = !open">
      <span class="agc-edit__verb">{{ created ? t('agent.code.edit.add') : t('agent.code.edit.modify') }}</span>
      <span class="ag-mono ag-ellipsis agc-edit__name">{{ baseName }}</span>
      <span v-if="counts.added || !counts.deleted" class="agc-plus ag-num">+{{ counts.added }}</span>
      <span v-if="counts.deleted" class="agc-minus ag-num">−{{ counts.deleted }}</span>
      <UIcon :name="isOpen ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" class="agc-edit__chev" />
    </button>
    <div v-if="isOpen" class="agc-diffcard">
      <div class="agc-diffcard__head">
        <span class="ag-mono ag-ellipsis agc-tool__arg">{{ path }}</span>
        <button type="button" class="ag-btn ag-btn--sm agc-diffcard__btn" @click="emit('showDiff', path)">
          {{ t('agent.code.fullDiff') }}
        </button>
      </div>
      <div ref="host" class="agc-diffcard__body" :style="{ height: `${height}px` }" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { monaco } from '@/components/file/monaco'
import { useThemeStore } from '@/stores/theme'
import { detectMonacoLanguage } from '@/utils/file'
import { diffDelta } from '@/views/agent/utils/diff'
import type { AgentDiff } from './types'

/** bare：不要那一行，直接显示 diff（右侧面板里子 Agent 的改动用） */
const props = defineProps<{ diff: AgentDiff; path: string; bare?: boolean }>()
const emit = defineEmits<{ showDiff: [path: string] }>()
const { t } = useI18n()
const themeStore = useThemeStore()

const LINE = 18
const MAX_LINES = 14

const host = ref<HTMLElement | null>(null)
const open = ref(false)
const isOpen = computed(() => props.bare || open.value)
const created = computed(() => props.diff.oldText == null)
const counts = computed(() => diffDelta(props.diff))
const baseName = computed(() => props.path.slice(Math.max(props.path.lastIndexOf('/'), props.path.lastIndexOf('\\')) + 1) || props.path)
const height = ref(LINE * 4)

let editor: monaco.editor.IStandaloneDiffEditor | null = null
let models: monaco.editor.ITextModel[] = []

const dispose = () => {
  editor?.dispose()
  for (const m of models) m.dispose()
  editor = null
  models = []
}
const build = () => {
  if (editor || !host.value) return
  const language = detectMonacoLanguage(props.path)
  const original = monaco.editor.createModel(props.diff.oldText ?? '', language)
  const modified = monaco.editor.createModel(props.diff.newText ?? '', language)
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
    const lines = editor?.getModifiedEditor().getContentHeight() ?? LINE * 4
    height.value = Math.min(lines, LINE * MAX_LINES)
  })
}

watch(
  isOpen,
  async (on) => {
    if (!on) {
      dispose()
      return
    }
    await nextTick()
    build()
  },
  { immediate: true },
)
watch(
  () => props.diff,
  (diff) => {
    if (!editor || models.length < 2) return
    models[0]!.setValue(diff.oldText ?? '')
    models[1]!.setValue(diff.newText ?? '')
  },
)
watch(() => themeStore.isDarkTheme, (dark) => monaco.editor.setTheme(dark ? 'vs-dark' : 'vs'))
onBeforeUnmount(dispose)
</script>
