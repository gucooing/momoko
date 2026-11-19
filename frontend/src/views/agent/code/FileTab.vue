<!-- 标签页里打开的文件：只读，用项目里的 Monaco 显示。这个文件有未提交改动时，头部带 +/− 和「查看改动」。 -->
<template>
  <div class="agc-file">
    <div class="agc-file__head">
      <span class="ag-mono ag-ellipsis ag-muted agc-flex">{{ fullPath }}</span>
      <template v-if="change">
        <span class="agc-plus ag-num">+{{ change.added }}</span>
        <span class="agc-minus ag-num">−{{ change.deleted }}</span>
        <button type="button" class="ag-btn ag-btn--sm" @click="emit('showDiff', path)">{{ t('agent.code.file.showDiff') }}</button>
      </template>
      <span class="ag-tag">{{ langLabel }}</span>
    </div>
    <div v-if="note" class="agc-file__note">{{ note }}</div>
    <div v-show="!note" ref="host" class="agc-file__editor" :aria-label="t('agent.code.file.content', { path })" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { readAgentSessionFile } from '@/api/agent'
import { monaco } from '@/components/file/monaco'
import type { AgentChangedFile } from '@/types/v1/agent'
import { useThemeStore } from '@/stores/theme'
import { detectMonacoLanguage } from '@/utils/file'
import { fmtBytes } from '@/views/agent/utils/format'

const props = defineProps<{ sessionId: string; path: string; cwd: string; change?: AgentChangedFile | null }>()
const emit = defineEmits<{ showDiff: [path: string] }>()
const { t } = useI18n()
const themeStore = useThemeStore()

const host = ref<HTMLElement | null>(null)
const note = ref('')
const language = computed(() => detectMonacoLanguage(props.path))
const langLabel = computed(() => (language.value === 'plaintext' ? t('agent.code.file.plain') : language.value))
const fullPath = computed(() => `${props.cwd.replace(/[\\/]+$/, '')}/${props.path}`)

let editor: monaco.editor.IStandaloneCodeEditor | null = null

const load = async () => {
  note.value = t('agent.common.loading')
  try {
    const { data } = await readAgentSessionFile(props.sessionId, props.path)
    if (data.binary) {
      note.value = t('agent.code.file.binary')
      return
    }
    if (data.tooLarge) {
      note.value = t('agent.code.file.tooLarge', { size: fmtBytes(data.size) })
      return
    }
    note.value = ''
    await nextTick()
    if (!host.value) return
    if (!editor) {
      editor = monaco.editor.create(host.value, {
        value: data.content,
        language: language.value,
        readOnly: true,
        automaticLayout: true,
        minimap: { enabled: false },
        fontSize: 12.5,
        scrollBeyondLastLine: false,
        theme: themeStore.isDarkTheme ? 'vs-dark' : 'vs',
      })
    } else {
      const model = editor.getModel()
      model?.setValue(data.content)
      if (model) monaco.editor.setModelLanguage(model, language.value)
    }
  } catch {
    note.value = t('agent.code.file.failed')
  }
}

watch(() => themeStore.isDarkTheme, (dark) => monaco.editor.setTheme(dark ? 'vs-dark' : 'vs'))
watch(() => [props.sessionId, props.path], () => void load())
onMounted(() => void load())
onBeforeUnmount(() => {
  editor?.getModel()?.dispose()
  editor?.dispose()
  editor = null
})

defineExpose({ reload: load })
</script>
