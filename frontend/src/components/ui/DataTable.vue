<!-- 数据表格（令牌驱动，替代 VXE/el-table）：columns 列定义 + 行选择 + sticky 表头 + hover + 空/加载。
     强列型页面用它；移动端由页面改用卡片流（本组件桌面渲染，03b §7）。 -->
<template>
  <div class="data-table__scroll">
    <table class="data-table">
      <thead>
        <tr>
          <th v-if="selectable" class="data-table__check">
            <input
              ref="allCheckRef"
              type="checkbox"
              class="data-table__checkbox"
              :checked="allChecked"
              :disabled="!selectableRows.length"
              @change="toggleAll"
            />
          </th>
          <th v-if="seq" class="data-table__seq">{{ t('system.common.serialNumber') }}</th>
          <th
            v-for="col in columns"
            :key="col.key"
            :style="colStyle(col)"
            :class="[`is-${col.align || 'left'}`]"
          >
            {{ col.title }}
          </th>
        </tr>
      </thead>

      <tbody>
        <!-- 加载骨架：仅首次加载（无数据时）显示；重载（翻页/筛选）保留旧行，避免整表闪烁 -->
        <template v-if="loading && !displayRows.length">
          <tr v-for="i in 5" :key="`sk-${i}`" class="data-table__skeleton-row">
            <td v-if="selectable"><span class="data-table__skeleton" /></td>
            <td v-if="seq"><span class="data-table__skeleton" /></td>
            <td v-for="col in columns" :key="col.key"><span class="data-table__skeleton" /></td>
          </tr>
        </template>

        <!-- 空态 -->
        <tr v-else-if="!displayRows.length" class="data-table__empty-row">
          <td :colspan="totalCols">
            <slot name="empty">
              <EmptyState :title="emptyText || t('system.common.noData')" />
            </slot>
          </td>
        </tr>

        <!-- 数据行 -->
        <template v-else>
          <tr
            v-for="(entry, index) in displayRows"
            :key="rowId(entry.row)"
            :class="{
              'is-selected': selectedSet.has(rowId(entry.row)),
              'is-clickable': rowClickable,
            }"
            @click="onRowClick($event, entry.row)"
          >
            <td v-if="selectable" class="data-table__check">
              <input
                type="checkbox"
                class="data-table__checkbox"
                :checked="selectedSet.has(rowId(entry.row))"
                :disabled="!isRowSelectable(entry.row)"
                @change="toggleRow(rowId(entry.row))"
              />
            </td>
            <td v-if="seq" class="data-table__seq">{{ index + 1 }}</td>
            <td
              v-for="col in columns"
              :key="col.key"
              :class="[`is-${col.align || 'left'}`, col.ellipsis !== false ? 'is-ellipsis' : '']"
            >
              <!-- 树列：缩进 + 展开插入符 -->
              <div
                v-if="tree && col.key === treeColKey"
                class="data-table__tree-cell"
                :style="{ paddingInlineStart: `${entry.depth * treeIndent}px` }"
              >
                <button
                  v-if="entry.hasChildren"
                  type="button"
                  class="data-table__tree-caret"
                  :class="{ 'is-open': expandedSet.has(rowId(entry.row)) }"
                  @click="toggleExpand(rowId(entry.row))"
                >
                  <component :is="menuStore.iconComponents['HOutline:ChevronRightIcon']" />
                </button>
                <span v-else class="data-table__tree-caret data-table__tree-caret--leaf" />
                <slot :name="`cell-${col.key}`" :row="entry.row" :value="entry.row[col.key]" :index="index">
                  <span class="data-table__tree-label">{{ formatValue(entry.row[col.key]) }}</span>
                </slot>
              </div>
              <slot
                v-else
                :name="`cell-${col.key}`"
                :row="entry.row"
                :value="entry.row[col.key]"
                :index="index"
              >
                {{ formatValue(entry.row[col.key]) }}
              </slot>
            </td>
          </tr>
        </template>
      </tbody>
    </table>
    <!-- 重载遮罩：保留旧数据可见，仅叠加轻微蒙层 + spinner（不替换内容、不改布局） -->
    <div v-if="loading && displayRows.length" class="data-table__veil">
      <span class="data-table__spinner" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

export interface DataTableColumn {
  key: string
  title: string
  width?: string | number
  minWidth?: string | number
  align?: 'left' | 'center' | 'right'
  ellipsis?: boolean
}

const props = withDefaults(
  defineProps<{
    columns: DataTableColumn[]
    rows: Record<string, unknown>[]
    rowKey?: string
    selectable?: boolean
    /** 逐行可选判定；返回 false 的行 checkbox 禁用且不参与全选（如内置角色）。 */
    rowSelectable?: (row: Record<string, unknown>) => boolean
    modelValue?: string[]
    loading?: boolean
    emptyText?: string
    seq?: boolean
    /** 树表模式：rows 为嵌套树，按 childrenKey 展开；树列显示缩进 + 展开插入符。 */
    tree?: boolean
    childrenKey?: string
    /** 树列（放缩进/插入符）的 key；默认首列。 */
    treeColumnKey?: string
    treeIndent?: number
    defaultExpandAll?: boolean
    /** 行可点：true 时加 pointer 样式；监听 row-click 时自动为 true。 */
    rowClickable?: boolean
  }>(),
  { rowKey: 'id', modelValue: () => [], childrenKey: 'children', treeIndent: 20, defaultExpandAll: true },
)
const emit = defineEmits<{
  'update:modelValue': [value: string[]]
  'row-click': [row: Record<string, unknown>, event: MouseEvent]
}>()
const attrs = useAttrs()
/** 显式 prop 或已监听 row-click 时视为可点（父级 @row-click 会落入 attrs）。 */
const rowClickable = computed(
  () => props.rowClickable === true || typeof attrs.onRowClick === 'function',
)
const onRowClick = (event: MouseEvent, row: Record<string, unknown>) => {
  if (!rowClickable.value) return
  const target = event.target as HTMLElement | null
  // 行内交互控件自己处理，不冒成整行点击
  if (target?.closest('button, a, input, select, textarea, label, [role="button"]')) return
  emit('row-click', row, event)
}

const { t } = useI18n()
const menuStore = useMenuStore()

const rowId = (row: Record<string, unknown>) => String(row[props.rowKey])

// —— 树表：扁平化 + 展开态（非树模式退化为一层包裹，选择/序号逻辑统一走 displayRows）——
interface DisplayRow {
  row: Record<string, unknown>
  depth: number
  hasChildren: boolean
}
const treeColKey = computed(() => props.treeColumnKey || props.columns[0]?.key)
const childrenOf = (row: Record<string, unknown>) =>
  row[props.childrenKey] as Record<string, unknown>[] | undefined

const expandedSet = ref<Set<string>>(new Set())
const collectExpandable = (rows: Record<string, unknown>[], acc: Set<string>) => {
  for (const row of rows) {
    const kids = childrenOf(row)
    if (kids && kids.length) {
      acc.add(rowId(row))
      collectExpandable(kids, acc)
    }
  }
}
watch(
  () => props.rows,
  (rows) => {
    if (props.tree && props.defaultExpandAll) {
      const s = new Set<string>()
      collectExpandable(rows, s)
      expandedSet.value = s
    }
  },
  { immediate: true },
)
const toggleExpand = (id: string) => {
  const next = new Set(expandedSet.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedSet.value = next
}

const displayRows = computed<DisplayRow[]>(() => {
  if (!props.tree) return props.rows.map((row) => ({ row, depth: 0, hasChildren: false }))
  const acc: DisplayRow[] = []
  const walk = (rows: Record<string, unknown>[], depth: number) => {
    for (const row of rows) {
      const kids = childrenOf(row)
      const hasChildren = !!(kids && kids.length)
      acc.push({ row, depth, hasChildren })
      if (hasChildren && expandedSet.value.has(rowId(row))) walk(kids!, depth + 1)
    }
  }
  walk(props.rows, 0)
  return acc
})

const isRowSelectable = (row: Record<string, unknown>) =>
  props.rowSelectable ? props.rowSelectable(row) : true
const selectableRows = computed(() => displayRows.value.map((e) => e.row).filter(isRowSelectable))
const selectedSet = computed(() => new Set(props.modelValue))
const totalCols = computed(
  () => props.columns.length + (props.selectable ? 1 : 0) + (props.seq ? 1 : 0),
)
const allChecked = computed(
  () =>
    selectableRows.value.length > 0 &&
    selectableRows.value.every((r) => selectedSet.value.has(rowId(r))),
)

const allCheckRef = ref<HTMLInputElement>()
watchEffect(() => {
  if (allCheckRef.value) {
    const some = selectableRows.value.some((r) => selectedSet.value.has(rowId(r)))
    allCheckRef.value.indeterminate = some && !allChecked.value
  }
})

const toggleAll = () => {
  const selIds = selectableRows.value.map(rowId)
  if (allChecked.value) {
    const ids = new Set(selIds)
    emit(
      'update:modelValue',
      props.modelValue.filter((id) => !ids.has(id)),
    )
  } else {
    const merged = new Set([...props.modelValue, ...selIds])
    emit('update:modelValue', [...merged])
  }
}

const toggleRow = (id: string) => {
  if (selectedSet.value.has(id)) {
    emit(
      'update:modelValue',
      props.modelValue.filter((x) => x !== id),
    )
  } else {
    emit('update:modelValue', [...props.modelValue, id])
  }
}

const colStyle = (col: DataTableColumn) => {
  const style: Record<string, string> = {}
  if (col.width != null) style.width = typeof col.width === 'number' ? `${col.width}px` : col.width
  if (col.minWidth != null)
    style.minWidth = typeof col.minWidth === 'number' ? `${col.minWidth}px` : col.minWidth
  return style
}

const formatValue = (v: unknown) => (v == null || v === '' ? '—' : String(v))
</script>

<style scoped lang="scss">
.data-table__scroll {
  position: relative;
  overflow-x: auto;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  background: var(--el-bg-color);
}
/* 重载遮罩：轻蒙层 + 居中 spinner，保留旧数据可见、不改布局，替代整表骨架替换的闪烁。
   延迟 220ms 淡入：快请求（翻页秒回）根本不显示遮罩，只有慢请求才淡入，杜绝闪一下。 */
.data-table__veil {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  background: color-mix(in srgb, var(--el-bg-color) 55%, transparent);
  border-radius: var(--app-radius);
  opacity: 0;
  animation: dt-veil-in 0.2s ease 0.22s forwards;
}
@keyframes dt-veil-in {
  to {
    opacity: 1;
  }
}
.data-table__spinner {
  width: 22px;
  height: 22px;
  border: 2px solid var(--el-border-color);
  border-top-color: var(--el-color-primary);
  border-radius: 50%;
  animation: dt-spin 0.7s linear infinite;
}
@keyframes dt-spin {
  to {
    transform: rotate(360deg);
  }
}
.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.8125rem;
}
.data-table th,
.data-table td {
  padding: 7px 12px;
  text-align: left;
  white-space: nowrap;
}
.data-table thead th {
  padding-top: 8px;
  padding-bottom: 8px;
}
.data-table th.is-center,
.data-table td.is-center {
  text-align: center;
}
.data-table th.is-right,
.data-table td.is-right {
  text-align: right;
}
.data-table thead th {
  position: sticky;
  top: 0;
  z-index: 1;
  background: var(--el-fill-color-lighter);
  color: var(--el-text-color-secondary);
  font-weight: 600;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.data-table tbody td {
  color: var(--el-text-color-regular);
  border-bottom: 1px solid var(--el-border-color-lighter);
  font-variant-numeric: tabular-nums;
}
.data-table tbody tr:last-child td {
  border-bottom: none;
}
.data-table tbody tr {
  transition: background 0.12s;
}
.data-table tbody tr:hover {
  background: var(--el-fill-color-light);
}
.data-table tbody tr.is-selected {
  background: color-mix(in srgb, var(--el-color-primary) 5%, var(--el-bg-color));
}
.data-table tbody tr.is-clickable {
  cursor: pointer;
}
.data-table td.is-ellipsis {
  max-width: 260px;
  overflow: hidden;
  text-overflow: ellipsis;
}
/* 树表：缩进 + 插入符 */
.data-table__tree-cell {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}
.data-table__tree-caret {
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
.data-table__tree-caret::before {
  content: '';
  position: absolute;
  inset: -10px;
}
.data-table__tree-caret:hover {
  background: var(--el-fill-color);
}
.data-table__tree-caret:focus-visible {
  outline: none;
  box-shadow: var(--app-focus-ring, 0 0 0 3px color-mix(in srgb, var(--el-color-primary) 25%, transparent));
}
.data-table__tree-caret :deep(svg) {
  width: 14px;
  height: 14px;
}
.data-table__tree-caret.is-open {
  transform: rotate(90deg);
}
.data-table__tree-caret--leaf {
  cursor: default;
}
.data-table__tree-caret--leaf:hover {
  background: transparent;
}
.data-table__tree-caret--leaf::before {
  display: none;
}
.data-table__tree-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.data-table__check {
  width: 44px;
  text-align: center;
}
.data-table__seq {
  width: 60px;
  color: var(--el-text-color-placeholder);
}
.data-table__checkbox {
  width: 15px;
  height: 15px;
  accent-color: var(--el-color-primary);
  cursor: pointer;
  vertical-align: middle;
}
.data-table__checkbox:focus-visible {
  outline: none;
  box-shadow: var(--app-focus-ring, 0 0 0 3px color-mix(in srgb, var(--el-color-primary) 25%, transparent));
  border-radius: 3px;
}
/* 列已 44px 宽；粗指针下放大视觉勾选，便于点按 */
@media (pointer: coarse) {
  .data-table__checkbox {
    width: 18px;
    height: 18px;
  }
  .data-table__check {
    width: 48px;
  }
}
.data-table__empty-row td {
  padding: 0;
}
/* 加载骨架 */
.data-table__skeleton {
  display: block;
  height: 14px;
  border-radius: 5px;
  background: linear-gradient(
    100deg,
    var(--el-fill-color-light) 30%,
    var(--el-fill-color) 50%,
    var(--el-fill-color-light) 70%
  );
  background-size: 200% 100%;
  animation: dt-shimmer 1.4s ease-in-out infinite;
}
@keyframes dt-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
</style>
