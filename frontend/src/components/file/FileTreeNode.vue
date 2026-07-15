<template>
  <div class="ftn">
    <div
      ref="rowRef"
      class="ftn-row"
      :class="{ 'is-active': isActive }"
      :style="{ paddingLeft: `${depth * 14 + 8}px` }"
      @click="onRowClick"
    >
      <span
        class="ftn-caret"
        :class="{ 'is-open': expanded, 'is-leaf': !node.isDir }"
        @click.stop="toggleExpand"
      >
        <UIcon v-if="node.isDir" name="i-lucide-chevron-right" class="ftn-caret__ico" />
      </span>
      <input
        v-if="ctx.multiple && isSelectable"
        class="ftn-check"
        type="checkbox"
        :checked="isActive"
        @click.stop
        @change="onCheckChange"
      />
      <UIcon :name="nodeIcon" class="ftn-icon" :class="node.isDir ? 'is-folder' : 'is-file'" />
      <span class="ftn-name" :title="node.name">{{ node.name }}</span>
      <UIcon v-if="loading" name="i-lucide-loader-circle" class="ftn-loading" />
    </div>

    <div v-if="expanded">
      <FileTreeNode v-for="child in children" :key="child.path" :node="child" :depth="depth + 1" />
      <div
        v-if="loaded && !children.length"
        class="ftn-empty"
        :style="{ paddingLeft: `${(depth + 1) * 14 + 24}px` }"
      >
        {{ t('fileManager.directoryEmpty') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { getRequestErrorMessage } from '@/utils/request'
import { useFeedback } from '@/utils/feedback'
import type { FileTreeNode as FileTreeNodeType } from '@/types/v1/file'
import { FILE_TREE_CONTEXT, type FileTreeContext } from './types'

defineOptions({ name: 'FileTreeNode' })

const props = defineProps<{
  node: FileTreeNodeType
  depth: number
}>()

const { t } = useI18n()
const fb = useFeedback()
const ctx = inject<FileTreeContext>(FILE_TREE_CONTEXT)!

const expanded = ref(false)
const loaded = ref(false)
const loading = ref(false)
const children = ref<FileTreeNodeType[]>([])
const rowRef = ref<HTMLElement | null>(null)

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

const isSelectable = computed(() => ctx.selectable === 'all' || !props.node.isDir)
const isExplicitlySelected = computed(() => Boolean(ctx.selectedPaths?.value.has(props.node.path)))
const selectedAncestorPath = computed(() => {
  for (const selectedPath of ctx.selectedPaths?.value ?? []) {
    if (selectedPath !== props.node.path && isSameOrDescendantPath(props.node.path, selectedPath)) {
      return selectedPath
    }
  }

  return ''
})
const isActive = computed(() => {
  if (ctx.multiple) return isExplicitlySelected.value || Boolean(selectedAncestorPath.value)
  return ctx.activePath.value === props.node.path
})
const isCurrentActive = computed(() => ctx.activePath.value === props.node.path)
const nodeIcon = computed(() => {
  if (!props.node.isDir) return 'i-lucide-file'
  return expanded.value ? 'i-lucide-folder-open' : 'i-lucide-folder'
})

const loadChildren = async () => {
  if (loaded.value || loading.value) return
  loading.value = true
  try {
    children.value = await ctx.client.tree(props.node.path)
    loaded.value = true
  } catch (error) {
    fb.error(getRequestErrorMessage(error, t('fileManager.treeLoadFailed')))
  } finally {
    loading.value = false
  }
}

const toggleExpand = async () => {
  if (!props.node.isDir) return
  if (!loaded.value) await loadChildren()
  expanded.value = !expanded.value
}

const selectNode = () => {
  if (!isSelectable.value) return
  if (ctx.multiple && selectedAncestorPath.value && !isExplicitlySelected.value) {
    ctx.select(selectedAncestorPath.value, props.node.name, true)
    return
  }

  ctx.select(props.node.path, props.node.name, props.node.isDir)
}

const onCheckChange = () => {
  selectNode()
}

const onRowClick = async () => {
  if (props.node.isDir) {
    if (ctx.selectable === 'all') {
      // 选择器：点目录=选中并下钻；用箭头折叠
      selectNode()
      if (!expanded.value) {
        if (!loaded.value) await loadChildren()
        expanded.value = true
      }
    } else {
      await toggleExpand()
    }
  } else {
    selectNode()
  }
}

// 自动展开到当前活动文件所在路径（树根在可访问根目录时，逐级揭示）。
const sep = props.node.path.includes('\\') ? '\\' : '/'
const isAncestorOfActive = () => {
  const active = ctx.activePath.value
  if (!active || !props.node.isDir) return false
  const base = props.node.path.replace(/[\\/]+$/, '')
  return (
    active.length > base.length && active.toLowerCase().startsWith(`${base}${sep}`.toLowerCase())
  )
}
const reveal = async () => {
  if (!isAncestorOfActive()) return
  if (!loaded.value) await loadChildren()
  expanded.value = true
}

// 只在最近的纵向滚动容器内对齐当前节点，禁止 scrollIntoView 连带外层/文件列表一起滚。
const scrollToActive = () => {
  nextTick(() => {
    const el = rowRef.value
    if (!el) return
    let parent: HTMLElement | null = el.parentElement
    while (parent) {
      const style = getComputedStyle(parent)
      const canScrollY =
        (style.overflowY === 'auto' || style.overflowY === 'scroll') &&
        parent.scrollHeight > parent.clientHeight + 1
      if (canScrollY) {
        const parentRect = parent.getBoundingClientRect()
        const elRect = el.getBoundingClientRect()
        const delta =
          elRect.top - parentRect.top - parent.clientHeight / 2 + elRect.height / 2
        parent.scrollTop += delta
        return
      }
      parent = parent.parentElement
    }
  })
}

watch(isCurrentActive, (active) => {
  if (active) scrollToActive()
})
watch(() => ctx.activePath.value, reveal)
onMounted(() => {
  reveal()
  if (isCurrentActive.value) scrollToActive()
})
</script>

<style scoped>
/* 令牌走可覆盖的 --ft-*（app 令牌兜底）：FileBrowser/FilePicker 直接吃 app 令牌跟随全局明暗；
   FileEditor 在 .fe-tree 上把 --ft-* 桥接到其独立主题 --fm-*，从而树跟随编辑器自身明暗。 */
.ftn-row {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  height: 28px;
  padding-right: 8px;
  color: var(--ft-fg, var(--el-text-color-regular));
  font-size: 13px;
  cursor: pointer;
  user-select: none;
  transition:
    background 0.12s,
    color 0.12s;
}
.ftn-row:hover {
  background: var(--ft-hover, var(--el-fill-color-light));
}
.ftn-row.is-active {
  background: var(--ft-active-bg, var(--el-color-primary-light-9));
  color: var(--ft-active-fg, var(--el-color-primary));
}
.ftn-caret {
  display: inline-flex;
  width: 16px;
  height: 16px;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border-radius: 3px;
  color: var(--ft-fg-dim, var(--el-text-color-placeholder));
  transition:
    transform 0.15s,
    background 0.12s;
}
.ftn-caret:hover {
  background: var(--ft-hover, var(--el-fill-color-light));
}
.ftn-caret.is-leaf {
  cursor: default;
}
.ftn-caret__ico {
  width: 13px;
  height: 13px;
}
.ftn-caret.is-open {
  transform: rotate(90deg);
}
.ftn-check {
  width: 14px;
  height: 14px;
  margin: 0 2px 0 0;
  flex-shrink: 0;
  accent-color: var(--ft-active-fg, var(--el-color-primary));
}
.ftn-icon {
  flex-shrink: 0;
  width: 16px;
  height: 16px;
}
/* 文件夹图标不走琥珀色：默认继承行文本色；激活行已有 --ft-active-fg。 */
.ftn-icon.is-folder {
  color: inherit;
}
.ftn-icon.is-file {
  color: var(--ft-fg-dim, var(--el-text-color-placeholder));
}
.ftn-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ftn-loading {
  margin-left: auto;
  width: 13px;
  height: 13px;
  color: var(--ft-fg-dim, var(--el-text-color-placeholder));
  animation: ftn-spin 0.8s linear infinite;
}
.ftn-empty {
  height: 24px;
  display: flex;
  align-items: center;
  font-size: 12px;
  color: var(--ft-fg-dim, var(--el-text-color-placeholder));
}
@keyframes ftn-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
