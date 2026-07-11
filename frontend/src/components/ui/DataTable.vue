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
              :disabled="!rows.length"
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
        <!-- 加载骨架 -->
        <template v-if="loading">
          <tr v-for="i in 5" :key="`sk-${i}`" class="data-table__skeleton-row">
            <td v-if="selectable"><span class="data-table__skeleton" /></td>
            <td v-if="seq"><span class="data-table__skeleton" /></td>
            <td v-for="col in columns" :key="col.key"><span class="data-table__skeleton" /></td>
          </tr>
        </template>

        <!-- 空态 -->
        <tr v-else-if="!rows.length" class="data-table__empty-row">
          <td :colspan="totalCols">
            <slot name="empty">
              <EmptyState :title="emptyText || t('system.common.noData')" />
            </slot>
          </td>
        </tr>

        <!-- 数据行 -->
        <template v-else>
          <tr
            v-for="(row, index) in rows"
            :key="rowId(row)"
            :class="{ 'is-selected': selectedSet.has(rowId(row)) }"
          >
            <td v-if="selectable" class="data-table__check">
              <input
                type="checkbox"
                class="data-table__checkbox"
                :checked="selectedSet.has(rowId(row))"
                @change="toggleRow(rowId(row))"
              />
            </td>
            <td v-if="seq" class="data-table__seq">{{ index + 1 }}</td>
            <td
              v-for="col in columns"
              :key="col.key"
              :class="[`is-${col.align || 'left'}`, col.ellipsis !== false ? 'is-ellipsis' : '']"
            >
              <slot :name="`cell-${col.key}`" :row="row" :value="row[col.key]" :index="index">
                {{ formatValue(row[col.key]) }}
              </slot>
            </td>
          </tr>
        </template>
      </tbody>
    </table>
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
    modelValue?: string[]
    loading?: boolean
    emptyText?: string
    seq?: boolean
  }>(),
  { rowKey: 'id', modelValue: () => [] },
)
const emit = defineEmits<{ 'update:modelValue': [value: string[]] }>()

const { t } = useI18n()

const rowId = (row: Record<string, unknown>) => String(row[props.rowKey])
const selectedSet = computed(() => new Set(props.modelValue))
const totalCols = computed(
  () => props.columns.length + (props.selectable ? 1 : 0) + (props.seq ? 1 : 0),
)
const allChecked = computed(
  () => props.rows.length > 0 && props.rows.every((r) => selectedSet.value.has(rowId(r))),
)

const allCheckRef = ref<HTMLInputElement>()
watchEffect(() => {
  if (allCheckRef.value) {
    const some = props.rows.some((r) => selectedSet.value.has(rowId(r)))
    allCheckRef.value.indeterminate = some && !allChecked.value
  }
})

const toggleAll = () => {
  if (allChecked.value) {
    const ids = new Set(props.rows.map(rowId))
    emit(
      'update:modelValue',
      props.modelValue.filter((id) => !ids.has(id)),
    )
  } else {
    const merged = new Set([...props.modelValue, ...props.rows.map(rowId)])
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
  overflow-x: auto;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
}
.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.8125rem;
}
.data-table th,
.data-table td {
  padding: 11px 14px;
  text-align: left;
  white-space: nowrap;
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
.data-table td.is-ellipsis {
  max-width: 260px;
  overflow: hidden;
  text-overflow: ellipsis;
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
