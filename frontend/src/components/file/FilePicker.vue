<template>
  <BaseDialog
    v-model="visible"
    :title="title || t('fileManager.selectFile')"
    :width="pickerWidth"
    @close="onClose"
  >
    <div class="file-module fpk" :class="{ 'is-dark': isDark }">
      <div class="fpk-tree" :class="{ 'is-multiple': multiple }">
        <FileTree
          :client="client"
          root-path=""
          :active-path="selected"
          :selected-paths="selectedPathList"
          :multiple="multiple"
          selectable="all"
          @select="onSelect"
        />
      </div>
      <div class="fpk-selected">
        <span class="fpk-selected-label">
          {{
            multiple
              ? t('fileManager.selectedItems', { count: selectedPathList.length })
              : t('fileManager.filePath')
          }}:
        </span>
        <div v-if="multiple" class="fpk-selected-tags">
          <el-tag
            v-for="path in selectedPathList"
            :key="path"
            class="fpk-selected-tag"
            closable
            @close="removeSelected(path)"
          >
            {{ path }}
          </el-tag>
          <span v-if="!selectedPathList.length" class="fpk-selected-empty">—</span>
        </div>
        <span v-else class="fpk-selected-path" :title="selected">{{ selected || '—' }}</span>
      </div>
    </div>

    <template #footer>
      <el-button @click="visible = false">
        {{ t('system.common.cancel') }}
      </el-button>
      <el-button type="primary" :disabled="!canConfirm" @click="confirm">
        {{ t('system.common.confirm') }}
      </el-button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useThemeStore } from '@/stores/theme'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import FileTree from './FileTree.vue'
import { createFileClient } from './fileClient'
import type { FileScope } from './types'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    scope?: FileScope
    title?: string
    initialPath?: string
    initialPaths?: string[]
    multiple?: boolean
  }>(),
  {
    scope: () => ({ kind: 'system' }),
    initialPaths: () => [],
    multiple: false,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  confirm: [value: string | string[]]
}>()

const { t } = useI18n()
const themeStore = useThemeStore()
const isDark = computed(() => themeStore.isDarkTheme)

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

const client = createFileClient(props.scope)
const selected = ref('')
const selectedPaths = ref<Set<string>>(new Set())
const selectedPathList = computed(() => Array.from(selectedPaths.value))
const pickerWidth = computed(() => (props.multiple ? 760 : 560))
const canConfirm = computed(() =>
  props.multiple ? selectedPathList.value.length > 0 : !!selected.value,
)

const normalizeComparePath = (path: string) => path.replace(/\\/g, '/').replace(/\/+$/, '')
const isWindowsPath = (path: string) =>
  /^[a-z]:\//i.test(normalizeComparePath(path)) || path.includes('\\')
const isSameOrDescendantPath = (path: string, parent: string) => {
  const normalizedPath = normalizeComparePath(path)
  const normalizedParent = normalizeComparePath(parent)
  if (!normalizedPath || !normalizedParent) return false

  const ignoreCase = isWindowsPath(path) || isWindowsPath(parent)
  const current = ignoreCase ? normalizedPath.toLowerCase() : normalizedPath
  const base = ignoreCase ? normalizedParent.toLowerCase() : normalizedParent
  return current === base || current.startsWith(`${base}/`)
}

const onSelect = (path: string) => {
  selected.value = path
  if (!props.multiple) return

  const next = new Set(selectedPaths.value)
  const selectedAncestor = Array.from(next).find(
    (selectedPath) => selectedPath !== path && isSameOrDescendantPath(path, selectedPath),
  )

  if (selectedAncestor) {
    next.delete(selectedAncestor)
  } else if (next.has(path)) {
    next.delete(path)
  } else {
    for (const selectedPath of next) {
      if (selectedPath !== path && isSameOrDescendantPath(selectedPath, path)) {
        next.delete(selectedPath)
      }
    }
    next.add(path)
  }

  selectedPaths.value = next
}

const removeSelected = (path: string) => {
  const next = new Set(selectedPaths.value)
  next.delete(path)
  selectedPaths.value = next
  if (selected.value === path) selected.value = selectedPathList.value[0] || ''
}

const confirm = () => {
  if (!canConfirm.value) return
  emit('confirm', props.multiple ? selectedPathList.value : selected.value)
  visible.value = false
}

const onClose = () => {
  selected.value = ''
  selectedPaths.value = new Set()
}

// 打开时以已选路径为初值，树据此自动展开并滚动到该文件/文件夹（免重复展开）。
watch(visible, (open) => {
  if (!open) {
    onClose()
    return
  }

  const initialPaths = props.initialPaths.length
    ? props.initialPaths
    : props.initialPath
      ? [props.initialPath]
      : []
  selected.value = initialPaths[0] || ''
  selectedPaths.value = new Set(initialPaths)
})
</script>

<style scoped>
.fpk {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.fpk-tree {
  height: 360px;
  overflow: auto;
  border: 1px solid var(--fm-border);
  border-radius: var(--fm-radius-sm);
  background: var(--fm-surface);
}
.fpk-tree.is-multiple {
  height: 460px;
}
.fpk-selected {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 13px;
  min-width: 0;
}
.fpk-selected-label {
  flex-shrink: 0;
  color: var(--fm-text-3);
}
.fpk-selected-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  min-width: 0;
  max-height: 88px;
  overflow: auto;
}
.fpk-selected-tag {
  max-width: 100%;
}
.fpk-selected-empty {
  color: var(--fm-text);
}
.fpk-selected-path {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--fm-text);
}
</style>
