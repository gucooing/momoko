<!-- 结构化改动列表：按顶层条目分组（例如某个卡池、某条公告），每条显示字段、改前与改后。
     时间字段按服务器时区显示，概率按百分比显示；整条新增 / 删除的对象只显示它的名称。
     配置里的 ID（引用别的条目、卡、素材）不直接显示：交给 refLabel 换成名称，换不了的显示「未命名」。 -->
<template>
  <div class="cdv">
    <EmptyState v-if="!entries.length" icon="HOutline:CheckCircleIcon" :title="emptyText || t('gmConfig.diff.none')" />
    <template v-else>
      <section v-for="group in groups" :key="group.key" class="cdv__group">
        <header class="cdv__head">
          <span class="cdv__title">{{ group.label }}</span>
          <span class="cdv__count m-num">{{ group.items.length }}</span>
        </header>
        <div v-for="(e, i) in group.items" :key="i" class="cdv__row">
          <span class="cdv__kind" :class="`is-${e.kind}`">{{ t(`gmConfig.diff.kind.${e.kind}`) }}</span>
          <span class="cdv__path">{{ fieldPath(e) }}</span>
          <span class="cdv__values">
            <template v-if="e.kind === 'changed'">
              <span class="cdv__before">{{ format(e, e.before) }}</span>
              <UIcon name="i-lucide-arrow-right" class="cdv__arrow" />
              <span class="cdv__after">{{ format(e, e.after) }}</span>
            </template>
            <span v-else-if="e.kind === 'added'" class="cdv__after">{{ format(e, e.after) }}</span>
            <span v-else class="cdv__before is-removed">{{ format(e, e.before) }}</span>
          </span>
        </div>
      </section>
      <p v-if="entries.length > max" class="cdv__more">{{ t('gmConfig.diff.more', { n: entries.length - max }) }}</p>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { diffGroupKey, isPlainObject, objectLabel, type DiffEntry } from './diff'
import { formatZoned } from './time'

const props = withDefaults(
  defineProps<{
    entries: DiffEntry[]
    zone: string
    /** 分组标题（参数为分组路径的前两段），不传时显示路径 */
    groupLabel?: (path: string[]) => string
    /**
     * 引用的名称：collection 是这个 ID 所在的集合或字段（gashas、lotteryId、poolIds、resourceId…），
     * 返回空串表示认不出（显示「未命名」）
     */
    refLabel?: (collection: string, id: string) => string
    emptyText?: string
    max?: number
  }>(),
  { groupLabel: undefined, refLabel: undefined, emptyText: '', max: 300 },
)

const { t, te } = useI18n()

const groups = computed(() => {
  const map = new Map<string, { key: string; label: string; items: DiffEntry[] }>()
  for (const e of props.entries.slice(0, props.max)) {
    const key = diffGroupKey(e) || '·'
    let g = map.get(key)
    if (!g) {
      const head = e.path.slice(0, 2)
      g = { key, label: props.groupLabel?.(head) || head.map((seg, i) => segLabel(head, i)).join(' › ') || t('gmConfig.diff.root'), items: [] }
      map.set(key, g)
    }
    g.items.push(e)
  }
  return [...map.values()]
})

// 字段名、按位置对齐的元素（#n）、按 id 对齐的元素（显示它引用的名称）
const ID_KEY = /(^id$|Id$|Ids$)/
const ASSET_KEY = /AssetId$/
const segLabel = (path: string[], i: number) => {
  const seg = path[i] || ''
  if (te(`gmConfig.fields.${seg}`)) return t(`gmConfig.fields.${seg}`)
  if (/^#\d+$/.test(seg)) return t('gmConfig.diff.nth', { n: seg.slice(1) })
  const parent = path[i - 1]
  if (parent) return refName(parent, seg)
  return seg
}

const refName = (collection: string, id: string) => {
  if (!id) return t('gmConfig.diff.empty')
  if (ASSET_KEY.test(collection)) return t('gmConfig.diff.asset')
  return props.refLabel?.(collection, id) || t('game.unnamed')
}

const fieldPath = (e: DiffEntry) => {
  const from = Math.min(2, e.path.length - 1)
  return e.path
    .slice(Math.max(0, from))
    .map((_, i) => segLabel(e.path, Math.max(0, from) + i))
    .join(' › ')
}

const lastKey = (e: DiffEntry) => {
  for (let i = e.path.length - 1; i >= 0; i--) {
    const seg = e.path[i] || ''
    if (!seg.startsWith('#')) return seg
  }
  return ''
}

// 值是 ID 时它属于哪个集合：字段 xxxId / xxxIds 就是字段本身；元素自己的 id 属于元素所在的数组
const refCollection = (e: DiffEntry, key: string) => {
  if (key !== 'id') return key
  const i = e.path.lastIndexOf('id')
  return e.path[i - 2] || ''
}

const isPrimitive = (v: unknown) => v === null || typeof v !== 'object'

const format = (e: DiffEntry, v: unknown): string => {
  const key = lastKey(e)
  if (v === undefined) return '—'
  if (v === null) return t('gmConfig.diff.null')
  if (typeof v === 'number') {
    if (/Time$/.test(key)) return v === 0 ? t('gmConfig.diff.zeroTime') : formatZoned(v, props.zone, true)
    if (/[pP]robability$/.test(key)) return `${(v / 1000).toFixed(3)}%`
    return v.toLocaleString()
  }
  if (typeof v === 'string') {
    if (v === '') return t('gmConfig.diff.empty')
    return ID_KEY.test(key) ? refName(refCollection(e, key), v) : v
  }
  if (typeof v === 'boolean') return v ? t('gmConfig.diff.yes') : t('gmConfig.diff.no')
  if (Array.isArray(v)) {
    if (!v.length) return t('gmConfig.diff.emptyList')
    if (v.every(isPrimitive)) {
      // 按 id 对齐的数组只是顺序变了时记的是元素 id；ID 数组（poolIds 等）也换成名称
      return e.order || ID_KEY.test(key) ? v.map((x) => refName(key, String(x))).join('、') : v.join(', ')
    }
    return t('gmConfig.diff.items', { n: v.length })
  }
  if (isPlainObject(v)) {
    const id = ['id', 'resourceId'].map((k) => v[k]).find((x): x is string => typeof x === 'string' && !!x)
    const collection = typeof v.id === 'string' ? e.path[e.path.length - 2] || '' : 'resourceId'
    return objectLabel(v) || (id ? refName(collection, id) : t('gmConfig.diff.object'))
  }
  return String(v)
}
</script>

<style scoped lang="scss">
.cdv {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}
.cdv__group {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 12px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.cdv__head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-bottom: 6px;
}
.cdv__title {
  flex: 1;
  min-width: 0;
  font-weight: 800;
  font-size: 14px;
  overflow-wrap: anywhere;
}
.cdv__count {
  padding: 0 8px;
  border-radius: 999px;
  background: var(--m-s2);
  font-size: 12px;
  font-weight: 700;
}
.cdv__row {
  display: grid;
  grid-template-columns: 52px minmax(0, 0.9fr) minmax(0, 1.4fr);
  gap: 8px;
  align-items: baseline;
  padding: 6px 0;
  border-top: 1px solid var(--el-border-color-lighter);
  font-size: 13px;
}
.cdv__kind {
  justify-self: start;
  padding: 1px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 800;
  white-space: nowrap;
}
.cdv__kind.is-added {
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.cdv__kind.is-removed {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.cdv__kind.is-changed {
  background: var(--m-warnSoft);
  color: var(--m-warn);
}
.cdv__path {
  color: var(--el-text-color-secondary);
  font-weight: 700;
  overflow-wrap: anywhere;
}
.cdv__values {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 6px;
  min-width: 0;
  overflow-wrap: anywhere;
}
.cdv__before {
  color: var(--el-text-color-secondary);
}
.cdv__before.is-removed {
  text-decoration: line-through;
}
.cdv__after {
  font-weight: 700;
}
.cdv__arrow {
  width: 14px;
  height: 14px;
  color: var(--el-text-color-secondary);
  flex: none;
}
.cdv__more {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
@media (max-width: 640px) {
  .cdv__row {
    grid-template-columns: 52px minmax(0, 1fr);
  }
  .cdv__values {
    grid-column: 2;
  }
}
</style>
