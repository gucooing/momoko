<!-- 通用数据渲染：没有专门界面的视图 / 报表用它显示。对象数组 → 表格（列取前 20 行出现过的字段），
     对象 → 键值列表；以 At / Time 结尾的毫秒字段按时间显示。字段名来自服务端，原样显示。
     ID 字段不显示：认得的（itemId、characterId…）换成名称，其余的 ID 列直接不列。 -->
<template>
  <div class="gdata">
    <div v-if="rows" class="gdata__scroll">
      <table v-if="rows.length" class="gdata__table">
        <thead>
          <tr>
            <th v-for="col in columns" :key="col">{{ col }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, i) in rows" :key="i">
            <td v-for="col in columns" :key="col" :title="cellTitle(row[col])">{{ cell(col, row[col]) }}</td>
          </tr>
        </tbody>
      </table>
      <EmptyState v-else icon="HOutline:ClipboardDocumentListIcon" :title="t('gmData.common.noData')" />
    </div>
    <dl v-else-if="entries" class="gdata__kv">
      <div v-for="[key, value] in entries" :key="key" class="gdata__kv-row">
        <dt>{{ key }}</dt>
        <dd :title="cellTitle(value)">{{ cell(key, value) }}</dd>
      </div>
    </dl>
    <p v-else class="gdata__plain">{{ cell('', data) }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useGmContext } from '../../context'
import { useGmFormat } from './format'

const props = defineProps<{ data: unknown }>()
const { t } = useI18n()
const ctx = useGmContext()
const { dateTime } = useGmFormat()

// 引用游戏里某样东西的 ID 字段 → 名称所在的目录
const NAMED: Record<string, string> = {
  itemId: 'item',
  characterId: 'character',
  idolCardId: 'idolCard',
  idolCardSkinId: 'idolCardSkin',
  skinId: 'idolCardSkin',
  supportCardId: 'supportCard',
  costumeId: 'costume',
  costumeHeadId: 'costumeHead',
  musicId: 'music',
  produceId: 'produce',
  produceGroupId: 'produceGroup',
  gashaId: 'gasha',
  towerId: 'tower',
}
const isIdKey = (key: string) => /(^id$|^pid$|Id$|Ids$)/.test(key)
const shown = (key: string) => !isIdKey(key) || key in NAMED

type Row = Record<string, unknown>
const isRow = (v: unknown): v is Row => !!v && typeof v === 'object' && !Array.isArray(v)

// 列表页 { items, nextCursor, total } 取 items
const list = computed<unknown[] | undefined>(() => {
  const d = props.data
  if (Array.isArray(d)) return d
  if (isRow(d) && Array.isArray(d.items)) return d.items
  return undefined
})

const rows = computed<Row[] | undefined>(() => {
  const l = list.value
  if (!l) return undefined
  return l.map((v) => (isRow(v) ? v : { value: v }))
})

const columns = computed(() => {
  const out: string[] = []
  for (const row of (rows.value || []).slice(0, 20)) {
    for (const key of Object.keys(row)) {
      if (shown(key) && !out.includes(key)) out.push(key)
    }
  }
  return out.slice(0, 12)
})

const entries = computed(() => (isRow(props.data) && !list.value ? Object.entries(props.data).filter(([k]) => shown(k)) : undefined))

const isTimeKey = (key: string) => /(At|Time|time)$/.test(key)

const cell = (key: string, v: unknown): string => {
  if (v === null || v === undefined || v === '') return '—'
  const kind = NAMED[key]
  if (kind && typeof v === 'string') return ctx.label(kind, v)
  if (typeof v === 'number') return isTimeKey(key) && v > 1e11 ? dateTime(v) : v.toLocaleString()
  if (typeof v === 'boolean') return v ? '✓' : '—'
  if (typeof v === 'string') return v
  const json = JSON.stringify(v)
  return json.length > 80 ? `${json.slice(0, 80)}…` : json
}

const cellTitle = (v: unknown) => (v && typeof v === 'object' ? JSON.stringify(v) : undefined)
</script>

<style scoped>
.gdata {
  min-width: 0;
}
.gdata__scroll {
  overflow-x: auto;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
}
.gdata__table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.gdata__table th,
.gdata__table td {
  padding: 10px 12px;
  text-align: left;
  white-space: nowrap;
  max-width: 280px;
  overflow: hidden;
  text-overflow: ellipsis;
}
.gdata__table th {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 700;
  border-bottom: 1px solid var(--el-border-color-light);
}
.gdata__table td {
  border-top: 1px solid var(--el-border-color-lighter);
}
.gdata__kv {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 8px;
}
.gdata__kv-row {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
  min-width: 0;
}
.gdata__kv-row dt {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.gdata__kv-row dd {
  margin: 0;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gdata__plain {
  margin: 0;
  font-size: 13px;
}
</style>
