<template>
  <BaseDialog
    v-model="visible"
    :title="title || t('fileManager.selectFile')"
    :width="pickerWidth"
    @close="onClose"
  >
    <div class="file-module fpk" :class="{ 'is-dark': isDark }">
      <!-- 来源切换（仅系统级）：可在不同来源间切换，选择跨来源累积 -->
      <div v-if="isSystemScope" class="fpk-source">
        <span class="fpk-source-label">{{ t('fileSource.switchSource') }}:</span>
        <select v-model="currentSourceId" class="fm-input fpk-source-select">
          <option value="">{{ t('fileSource.localDisk') }}</option>
          <option v-for="s in sources" :key="s.id" :value="s.id">
            {{ s.name }}（{{ s.type.toUpperCase() }}）
          </option>
        </select>
      </div>
      <div class="fpk-tree" :class="{ 'is-multiple': multiple }">
        <FileTree
          :key="currentSourceId"
          :client="client"
          root-path=""
          :active-path="activePath"
          :selected-paths="selectedPathsInCurrentSource"
          :multiple="multiple"
          selectable="all"
          @select="onSelect"
        />
      </div>
      <div class="fpk-selected">
        <span class="fpk-selected-label">
          {{
            multiple
              ? t('fileManager.selectedItems', { count: selectedList.length })
              : t('fileManager.filePath')
          }}:
        </span>
        <div v-if="multiple" class="fpk-selected-tags">
          <el-tag
            v-for="item in selectedList"
            :key="itemKey(item)"
            class="fpk-selected-tag"
            closable
            @close="removeSelected(item)"
          >
            <span v-if="sourceLabel(item.sourceId)" class="fpk-selected-source">{{
              sourceLabel(item.sourceId)
            }}</span
            >{{ item.path }}
          </el-tag>
          <span v-if="!selectedList.length" class="fpk-selected-empty">—</span>
        </div>
        <span v-else class="fpk-selected-path" :title="single?.path">
          <template v-if="single">
            <span v-if="sourceLabel(single.sourceId)" class="fpk-selected-source">{{
              sourceLabel(single.sourceId)
            }}</span
            >{{ single.path }}
          </template>
          <template v-else>—</template>
        </span>
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
import { listFileSourcesRequest } from '@/api/fileSource'
import type { FileSourceInfo } from '@/types/v1/file'
import type { FileScope, PickedFile } from './types'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    scope?: FileScope
    title?: string
    // 打开时预置的已选条目（编辑分享时回填）
    initialItems?: PickedFile[]
    multiple?: boolean
  }>(),
  {
    scope: () => ({ kind: 'system' }),
    initialItems: () => [],
    multiple: false,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  confirm: [value: PickedFile[]]
}>()

const { t } = useI18n()
const themeStore = useThemeStore()
const isDark = computed(() => themeStore.isDarkTheme)

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

const isSystemScope = props.scope.kind === 'system'
// 当前树展示的来源；source_id 动态读取，切换来源时给 FileTree 换 key 触发重载。
const currentSourceId = ref('')
const sources = ref<FileSourceInfo[]>([])
const client = createFileClient(props.scope, () => currentSourceId.value)

// 已选条目（跨来源累积），键 = sourceId + '\n' + path。
const selected = ref<Map<string, PickedFile>>(new Map())
const itemKey = (i: PickedFile) => `${i.sourceId}\n${i.path}`
const selectedList = computed(() => Array.from(selected.value.values()))
const single = computed<PickedFile | null>(() => selectedList.value[0] ?? null)
const activePath = ref('')

const pickerWidth = computed(() => (props.multiple ? 760 : 560))
const canConfirm = computed(() => selectedList.value.length > 0)

// 传给树的高亮路径：仅当前来源下的已选路径（跨来源不相关）。
const selectedPathsInCurrentSource = computed(() =>
  selectedList.value.filter((i) => i.sourceId === currentSourceId.value).map((i) => i.path),
)

const sourceLabel = (sourceId: string) => {
  if (!sourceId) return ''
  const s = sources.value.find((x) => x.id === sourceId)
  return s ? `${s.name}: ` : ''
}

const loadSources = async () => {
  if (!isSystemScope) return
  try {
    const { data } = await listFileSourcesRequest({ enabledOnly: true })
    sources.value = data.items ?? []
  } catch {
    sources.value = []
  }
}

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
  activePath.value = path
  const sourceId = currentSourceId.value
  const key = itemKey({ sourceId, path })

  if (!props.multiple) {
    selected.value = new Map([[key, { sourceId, path }]])
    return
  }

  const next = new Map(selected.value)
  // 祖先/后代去重仅在同一来源内进行（跨来源路径互不相关）。
  const sameSource = selectedList.value.filter((i) => i.sourceId === sourceId)
  const selectedAncestor = sameSource.find(
    (i) => i.path !== path && isSameOrDescendantPath(path, i.path),
  )

  if (selectedAncestor) {
    next.delete(itemKey(selectedAncestor))
  } else if (next.has(key)) {
    next.delete(key)
  } else {
    for (const i of sameSource) {
      if (i.path !== path && isSameOrDescendantPath(i.path, path)) next.delete(itemKey(i))
    }
    next.set(key, { sourceId, path })
  }

  selected.value = next
}

const removeSelected = (item: PickedFile) => {
  const next = new Map(selected.value)
  next.delete(itemKey(item))
  selected.value = next
  if (activePath.value === item.path && item.sourceId === currentSourceId.value) {
    activePath.value = selectedPathsInCurrentSource.value[0] || ''
  }
}

const confirm = () => {
  if (!canConfirm.value) return
  emit('confirm', selectedList.value)
  visible.value = false
}

const onClose = () => {
  selected.value = new Map()
  activePath.value = ''
}

// 打开时以已选条目为初值，并将树切到首个条目的来源（便于展开定位）。
watch(visible, (open) => {
  if (!open) {
    onClose()
    return
  }
  loadSources()
  const map = new Map<string, PickedFile>()
  for (const it of props.initialItems) {
    map.set(itemKey(it), { sourceId: it.sourceId, path: it.path })
  }
  selected.value = map
  const first = props.initialItems[0]
  currentSourceId.value = first?.sourceId ?? ''
  activePath.value = first && first.sourceId === currentSourceId.value ? first.path : ''
})
</script>

<style scoped>
.fpk {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.fpk-source {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 13px;
}
.fpk-source-label {
  flex-shrink: 0;
  color: var(--fm-text-3);
}
.fpk-source-select {
  flex: 1;
  min-width: 0;
}
.fpk-tree {
  height: 360px;
  overflow: auto;
  border: 1px solid var(--fm-border);
  border-radius: var(--fm-radius-sm);
  background: var(--fm-surface);
}
.fpk-tree.is-multiple {
  height: 420px;
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
.fpk-selected-source {
  color: var(--fm-text-3);
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
