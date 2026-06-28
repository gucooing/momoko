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
        <el-icon v-if="node.isDir"><IconChevronRight /></el-icon>
      </span>
      <el-icon class="ftn-icon" :class="node.isDir ? 'is-folder' : 'is-file'">
        <component :is="nodeIcon" />
      </el-icon>
      <span class="ftn-name" :title="node.name">{{ node.name }}</span>
      <el-icon v-if="loading" class="ftn-loading"><IconRefresh /></el-icon>
    </div>

    <div v-if="expanded">
      <FileTreeNode
        v-for="child in children"
        :key="child.path"
        :node="child"
        :depth="depth + 1"
      />
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
import { showRequestError } from '@/utils/request'
import type { FileTreeNode as FileTreeNodeType } from '@/types/v1/file'
import { FILE_TREE_CONTEXT, type FileTreeContext } from './types'
import { IconChevronRight, IconFolder, IconFolderOpen, IconFile, IconRefresh } from './icons'

defineOptions({ name: 'FileTreeNode' })

const props = defineProps<{
  node: FileTreeNodeType
  depth: number
}>()

const { t } = useI18n()
const ctx = inject<FileTreeContext>(FILE_TREE_CONTEXT)!

const expanded = ref(false)
const loaded = ref(false)
const loading = ref(false)
const children = ref<FileTreeNodeType[]>([])
const rowRef = ref<HTMLElement | null>(null)

const isActive = computed(() => ctx.activePath.value === props.node.path)
const nodeIcon = computed(() => {
  if (!props.node.isDir) return IconFile
  return expanded.value ? IconFolderOpen : IconFolder
})

const loadChildren = async () => {
  if (loaded.value || loading.value) return
  loading.value = true
  try {
    children.value = await ctx.client.tree(props.node.path)
    loaded.value = true
  } catch (error) {
    showRequestError(error, t('fileManager.treeLoadFailed'))
  } finally {
    loading.value = false
  }
}

const toggleExpand = async () => {
  if (!props.node.isDir) return
  if (!loaded.value) await loadChildren()
  expanded.value = !expanded.value
}

const onRowClick = async () => {
  if (props.node.isDir) {
    if (ctx.selectable === 'all') {
      // 选择器：点目录=选中并下钻；用箭头折叠
      ctx.select(props.node.path, props.node.name, true)
      if (!expanded.value) {
        if (!loaded.value) await loadChildren()
        expanded.value = true
      }
    } else {
      await toggleExpand()
    }
  } else {
    ctx.select(props.node.path, props.node.name, false)
  }
}

// 自动展开到当前活动文件所在路径（树根在可访问根目录时，逐级揭示）。
const sep = props.node.path.includes('\\') ? '\\' : '/'
const isAncestorOfActive = () => {
  const active = ctx.activePath.value
  if (!active || !props.node.isDir) return false
  const base = props.node.path.replace(/[\\/]+$/, '')
  return active.length > base.length && active.toLowerCase().startsWith(`${base}${sep}`.toLowerCase())
}
const reveal = async () => {
  if (!isAncestorOfActive()) return
  if (!loaded.value) await loadChildren()
  expanded.value = true
}

// 把当前活动文件滚动到树视图中央（节点挂载时即为 active 的情况 watch 不触发，故 onMounted 也处理一次）。
const scrollToActive = () => {
  nextTick(() => rowRef.value?.scrollIntoView({ block: 'center' }))
}

watch(isActive, (active) => {
  if (active) scrollToActive()
})
watch(() => ctx.activePath.value, reveal)
onMounted(() => {
  reveal()
  if (isActive.value) scrollToActive()
})
</script>

<style scoped>
.ftn-row {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  height: 28px;
  padding-right: 8px;
  color: var(--fm-text-2);
  font-size: 13px;
  cursor: pointer;
  user-select: none;
  transition:
    background 0.12s,
    color 0.12s;
}
.ftn-row:hover {
  background: var(--fm-hover);
}
.ftn-row.is-active {
  background: var(--fm-accent-soft);
  color: var(--fm-accent);
}
.ftn-caret {
  display: inline-flex;
  width: 16px;
  height: 16px;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border-radius: 3px;
  color: var(--fm-text-3);
  transition:
    transform 0.15s,
    background 0.12s;
}
.ftn-caret:hover {
  background: var(--fm-hover);
}
.ftn-caret.is-leaf {
  cursor: default;
}
.ftn-caret .el-icon {
  font-size: 13px;
}
.ftn-caret.is-open {
  transform: rotate(90deg);
}
.ftn-icon {
  flex-shrink: 0;
  font-size: 16px;
}
.ftn-icon.is-folder {
  color: var(--fm-folder);
}
.ftn-icon.is-file {
  color: var(--fm-text-3);
}
.ftn-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ftn-loading {
  margin-left: auto;
  font-size: 13px;
  color: var(--fm-text-3);
  animation: ftn-spin 0.8s linear infinite;
}
.ftn-empty {
  height: 24px;
  display: flex;
  align-items: center;
  font-size: 12px;
  color: var(--fm-text-3);
}
@keyframes ftn-spin {
  from {
    transform: perspective(120px) rotateY(0deg);
  }
  to {
    transform: perspective(120px) rotateY(360deg);
  }
}
</style>
