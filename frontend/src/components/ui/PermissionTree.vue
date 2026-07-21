<!-- 权限勾选树（令牌驱动，替代 el-tree show-checkbox）：三态复选（选中/半选/未选）、展开折叠、
     父子联动。v-model = 扁平选中集合（含全选节点 + 半选父节点），与后端 menuIds 语义一致：
     onToggle 后重算「任一后代叶子被选中」的节点集合 == getCheckedKeys(false) ∪ getHalfCheckedKeys()。
     角色/菜单权限分配复用（03b 自建令牌组件路线）。 -->
<template>
  <div class="perm-tree">
    <div class="perm-tree__bar">
      <button type="button" class="perm-tree__link" @click="setAll(true)">
        {{ t('system.role.selectAll') }}
      </button>
      <button type="button" class="perm-tree__link" @click="setAll(false)">
        {{ t('system.role.clearAll') }}
      </button>
      <span class="perm-tree__sep" />
      <button type="button" class="perm-tree__link" @click="expandAll(true)">
        {{ t('system.role.expandAll') }}
      </button>
      <button type="button" class="perm-tree__link" @click="expandAll(false)">
        {{ t('system.role.collapseAll') }}
      </button>
    </div>

    <div class="perm-tree__body">
      <EmptyState v-if="!visibleRows.length" :title="t('system.common.noData')" />
      <div
        v-for="row in visibleRows"
        :key="row.node.id"
        class="perm-tree__row"
        :style="{ paddingInlineStart: `${row.depth * 22 + 4}px` }"
      >
        <button
          v-if="row.hasChildren"
          type="button"
          class="perm-tree__caret"
          :class="{ 'is-open': expandedSet.has(row.node.id) }"
          :aria-label="t('system.role.expandAll')"
          @click="toggleExpand(row.node.id)"
        >
          <component :is="menuStore.iconComponents['HOutline:ChevronRightIcon']" />
        </button>
        <span v-else class="perm-tree__caret perm-tree__caret--leaf" />

        <button
          type="button"
          class="perm-tree__box"
          :class="{
            'is-checked': stateOf(row.node.id).checked,
            'is-indeterminate': stateOf(row.node.id).indeterminate,
          }"
          role="checkbox"
          :aria-checked="stateOf(row.node.id).indeterminate ? 'mixed' : stateOf(row.node.id).checked"
          @click="onToggle(row.node)"
        >
          <component
            v-if="stateOf(row.node.id).checked"
            :is="menuStore.iconComponents['HOutline:CheckIcon']"
          />
          <span v-else-if="stateOf(row.node.id).indeterminate" class="perm-tree__dash" />
        </button>

        <span class="perm-tree__title" @click="onToggle(row.node)">{{ row.node.title }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

export interface PermNode {
  id: string
  title: string
  children?: PermNode[]
}

const props = defineProps<{ nodes: PermNode[]; modelValue: string[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: string[]] }>()

const menuStore = useMenuStore()
const { t } = useI18n()

// —— 预计算：节点索引 / 每个节点下的叶子 id / 叶子集合 / 全部父节点 id ——
interface TreeMeta {
  order: string[]
  descLeaves: Map<string, string[]>
  parentIds: string[]
}
const meta = computed<TreeMeta>(() => {
  const order: string[] = []
  const descLeaves = new Map<string, string[]>()
  const parentIds: string[] = []

  const visit = (node: PermNode): string[] => {
    order.push(node.id)
    const children = node.children || []
    if (!children.length) {
      const self = [node.id]
      descLeaves.set(node.id, self)
      return self
    }
    parentIds.push(node.id)
    const leaves: string[] = []
    for (const child of children) leaves.push(...visit(child))
    descLeaves.set(node.id, leaves)
    return leaves
  }
  props.nodes.forEach(visit)
  return { order, descLeaves, parentIds }
})

// —— 展开态（默认全展开，随节点变化重置）——
const expandedSet = ref<Set<string>>(new Set())
watch(
  () => meta.value.parentIds,
  (ids) => {
    expandedSet.value = new Set(ids)
  },
  { immediate: true },
)

const toggleExpand = (id: string) => {
  const next = new Set(expandedSet.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedSet.value = next
}
const expandAll = (open: boolean) => {
  expandedSet.value = open ? new Set(meta.value.parentIds) : new Set()
}

// —— 选中态：v-model 为扁平集合，内部只以「被选中的叶子」为真源，父态由叶子推导 ——
const leafSet = computed(() => {
  const s = new Set<string>()
  for (const [id, leaves] of meta.value.descLeaves) if (leaves.length === 1 && leaves[0] === id) s.add(id)
  return s
})
const selectedLeaves = computed(() => new Set(props.modelValue.filter((id) => leafSet.value.has(id))))

const stateOf = (id: string) => {
  const leaves = meta.value.descLeaves.get(id) || []
  if (!leaves.length) return { checked: false, indeterminate: false }
  let n = 0
  for (const l of leaves) if (selectedLeaves.value.has(l)) n++
  return { checked: n === leaves.length, indeterminate: n > 0 && n < leaves.length }
}

/** 由选中叶子集合重算扁平 menuIds = 任一后代叶子被选中的节点（= 全选节点 ∪ 半选父节点）。 */
const emitFrom = (leaves: Set<string>) => {
  const out: string[] = []
  for (const id of meta.value.order) {
    const dl = meta.value.descLeaves.get(id) || []
    if (dl.some((l) => leaves.has(l))) out.push(id)
  }
  emit('update:modelValue', out)
}

const onToggle = (node: PermNode) => {
  const leaves = meta.value.descLeaves.get(node.id) || []
  const willCheck = !stateOf(node.id).checked // 全选态→取消；半选/未选→全选
  const next = new Set(selectedLeaves.value)
  for (const l of leaves) {
    if (willCheck) next.add(l)
    else next.delete(l)
  }
  emitFrom(next)
}

const setAll = (checked: boolean) => {
  emitFrom(checked ? new Set(leafSet.value) : new Set())
}

// —— 扁平渲染行（遵循展开态）——
interface Row {
  node: PermNode
  depth: number
  hasChildren: boolean
}
const visibleRows = computed(() => {
  const rows: Row[] = []
  const walk = (nodes: PermNode[], depth: number) => {
    for (const node of nodes) {
      const hasChildren = !!(node.children && node.children.length)
      rows.push({ node, depth, hasChildren })
      if (hasChildren && expandedSet.value.has(node.id)) walk(node.children!, depth + 1)
    }
  }
  walk(props.nodes, 0)
  return rows
})
</script>

<style scoped lang="scss">
.perm-tree {
  width: 100%;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  background: var(--el-bg-color);
  overflow: hidden;
}
.perm-tree__bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-lighter);
}
.perm-tree__link {
  border: none;
  background: transparent;
  padding: 4px 2px;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--el-color-primary);
  cursor: pointer;
  border-radius: var(--app-radius-xs);
}
.perm-tree__link:hover {
  text-decoration: underline;
}
.perm-tree__link:focus-visible {
  outline: none;
  box-shadow: var(--app-focus-ring, 0 0 0 3px color-mix(in srgb, var(--el-color-primary) 25%, transparent));
}
.perm-tree__sep {
  width: 1px;
  height: 12px;
  background: var(--el-border-color);
}
.perm-tree__body {
  max-height: 320px;
  overflow-y: auto;
  padding: 6px 0;
}
.perm-tree__row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 3px 12px 3px 0;
  min-height: 30px;
}
.perm-tree__caret {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-xs);
  color: var(--el-text-color-secondary);
  cursor: pointer;
  transition: transform 0.15s, background 0.15s, box-shadow 0.15s;
}
/* 扩大可点区，视觉仍保持 18px 图标 */
.perm-tree__caret::before {
  content: '';
  position: absolute;
  inset: -8px;
}
.perm-tree__caret:hover {
  background: var(--el-fill-color);
}
.perm-tree__caret:focus-visible {
  outline: none;
  box-shadow: var(--app-focus-ring, 0 0 0 3px color-mix(in srgb, var(--el-color-primary) 25%, transparent));
}
.perm-tree__caret :deep(svg) {
  width: 14px;
  height: 14px;
}
.perm-tree__caret.is-open {
  transform: rotate(90deg);
}
.perm-tree__caret--leaf {
  cursor: default;
}
.perm-tree__caret--leaf:hover {
  background: transparent;
}
.perm-tree__caret--leaf::before {
  display: none;
}
.perm-tree__box {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 17px;
  height: 17px;
  flex-shrink: 0;
  border: 1.5px solid var(--el-border-color);
  border-radius: var(--app-radius-xs);
  background: var(--el-bg-color);
  color: var(--ui-text-inverted, #fff);
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, box-shadow 0.15s;
}
.perm-tree__box::before {
  content: '';
  position: absolute;
  inset: -10px;
}
.perm-tree__box :deep(svg) {
  width: 12px;
  height: 12px;
  stroke-width: 3;
}
.perm-tree__box.is-checked {
  background: var(--el-color-primary);
  border-color: var(--el-color-primary);
}
.perm-tree__box.is-indeterminate {
  border-color: var(--el-color-primary);
}
.perm-tree__box:focus-visible {
  outline: none;
  box-shadow: var(--app-focus-ring, 0 0 0 3px color-mix(in srgb, var(--el-color-primary) 25%, transparent));
}
@media (pointer: coarse) {
  .perm-tree__row {
    min-height: 40px;
  }
}
.perm-tree__dash {
  width: 9px;
  height: 2px;
  border-radius: 1px;
  background: var(--el-color-primary);
}
.perm-tree__title {
  font-size: 0.8125rem;
  color: var(--el-text-color-regular);
  cursor: pointer;
  user-select: none;
}
.perm-tree__title:hover {
  color: var(--el-text-color-primary);
}
</style>
