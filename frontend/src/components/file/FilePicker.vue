<template>
  <FileDialog v-model="visible" :title="title || t('fileManager.selectFile')" :width="560" @close="onClose">
    <div class="fpk">
      <div class="fpk-tree">
        <FileTree
          :client="client"
          root-path=""
          :active-path="selected"
          selectable="all"
          @select="onSelect"
        />
      </div>
      <div class="fpk-selected">
        <span class="fpk-selected-label">{{ t('fileManager.filePath') }}:</span>
        <span class="fpk-selected-path" :title="selected">{{ selected || '—' }}</span>
      </div>
    </div>

    <template #footer>
      <button type="button" class="fm-btn" @click="visible = false">
        {{ t('system.common.cancel') }}
      </button>
      <button type="button" class="fm-btn fm-btn--primary" :disabled="!selected" @click="confirm">
        {{ t('system.common.confirm') }}
      </button>
    </template>
  </FileDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import FileDialog from './FileDialog.vue'
import FileTree from './FileTree.vue'
import { createFileClient } from './fileClient'
import type { FileScope } from './types'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    scope?: FileScope
    title?: string
    initialPath?: string
  }>(),
  {
    scope: () => ({ kind: 'system' }),
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  confirm: [path: string]
}>()

const { t } = useI18n()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

const client = createFileClient(props.scope)
const selected = ref('')

const onSelect = (path: string) => {
  selected.value = path
}

const confirm = () => {
  if (!selected.value) return
  emit('confirm', selected.value)
  visible.value = false
}

const onClose = () => {
  selected.value = ''
}

// 打开时以已选路径为初值，树据此自动展开并滚动到该文件/文件夹（免重复展开）。
watch(visible, (open) => {
  if (open) selected.value = props.initialPath || ''
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
.fpk-selected-path {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--fm-text);
}
</style>
