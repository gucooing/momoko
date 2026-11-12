<!-- 培育结果列表：玩家详情的「培育记录」与玩法页的全服培育列表共用。
     剧本（培育组）与难度（培育）名称来自 MGP 目录；偶像显示角色名与卡面。手机端为卡片。openable 时行可点开（发出 open）。 -->
<template>
  <div class="ptab">
    <DataTable
      v-if="!menuStore.isMobile"
      :columns="columns"
      :rows="tableRows"
      row-key="memoryId"
      :loading="loading"
      :empty-text="emptyText"
      :row-clickable="openable"
      @row-click="(row) => openable && emit('open', r(row))"
    >
      <template #cell-time="{ row }">
        <span class="ptab__dim m-num">{{ when(r(row).createdTime) }}</span>
      </template>
      <template #cell-player="{ row }">
        <RouterLink v-if="r(row).pid" class="ptab__player" :to="playerLink(r(row).pid || '')">
          <span class="ptab__strong">{{ r(row).name || t('gm.player.noName') }}</span>
          <span class="ptab__dim m-num">{{ r(row).displayId }}</span>
        </RouterLink>
      </template>
      <template #cell-idol="{ row }">
        <span class="ptab__idol">
          <GameImage :src="cardImage(r(row))" class="ptab__card" width="40" height="40" />
          <span class="ptab__strong">{{ charaName(r(row)) }}</span>
        </span>
      </template>
      <template #cell-produce="{ row }">
        <span class="ptab__stack">
          <span class="ptab__strong">{{ groupName(r(row)) }}</span>
          <span class="ptab__dim">{{ produceName(r(row)) }}</span>
        </span>
      </template>
      <template #cell-grade="{ row }">
        <span class="ptab__grade">
          <GradeBadge :grade="r(row).grade" />
          <span class="ptab__strong m-num">{{ num(r(row).score) }}</span>
        </span>
      </template>
      <template #cell-stats="{ row }">
        <span class="m-num">{{ num(r(row).vocal) }} / {{ num(r(row).dance) }} / {{ num(r(row).visual) }}</span>
      </template>
    </DataTable>

    <template v-else>
      <EmptyState v-if="!loading && !rows.length" icon="HOutline:SparklesIcon" :title="emptyText" />
      <div v-else class="ptab__cards">
        <article
          v-for="row in rows"
          :key="row.memoryId"
          class="ptab__mcard"
          :class="{ 'is-clickable': openable }"
          @click="openable && emit('open', row)"
        >
          <GameImage :src="cardImage(row)" class="ptab__mimg" width="52" height="52" />
          <div class="ptab__mbody">
            <div class="ptab__mline">
              <span class="ptab__strong">{{ charaName(row) }}</span>
              <GradeBadge :grade="row.grade" small />
              <span class="ptab__strong m-num">{{ num(row.score) }}</span>
            </div>
            <div class="ptab__dim">{{ groupName(row) }} · {{ produceName(row) }}</div>
            <div class="ptab__dim m-num">
              Vo {{ num(row.vocal) }} · Da {{ num(row.dance) }} · Vi {{ num(row.visual) }}
            </div>
            <div class="ptab__mfoot">
              <RouterLink v-if="showPlayer && row.pid" class="ptab__player" :to="playerLink(row.pid)">
                {{ row.name || t('gm.player.noName') }}
              </RouterLink>
              <span class="ptab__dim m-num">{{ when(row.createdTime) }}</span>
            </div>
          </div>
        </article>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { Memory } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import { catalogImage } from './game'
import GradeBadge from './GradeBadge.vue'
import { useGmFormat } from './format'
import { useCatalogLookup } from './lookup'

export interface ProduceRow extends Memory {
  pid?: string
  displayId?: string
  name?: string
  produceGroupId?: string
}

const props = withDefaults(
  defineProps<{
    rows: ProduceRow[]
    loading?: boolean
    showPlayer?: boolean
    emptyText?: string
    /** 行可点开（发出 open） */
    openable?: boolean
  }>(),
  {
    loading: false,
    showPlayer: false,
    emptyText: '',
    openable: false,
  },
)
const emit = defineEmits<{ open: [row: ProduceRow] }>()

const { t } = useI18n()
const ctx = useGmContext()
const menuStore = useMenuStore()
const lookup = useCatalogLookup()
const { num, when } = useGmFormat()

const columns = computed<DataTableColumn[]>(() => [
  { key: 'time', title: t('gmData.produce.time'), width: 130 },
  ...(props.showPlayer ? [{ key: 'player', title: t('gmData.produce.player'), minWidth: 150 }] : []),
  { key: 'idol', title: t('gmData.produce.idol'), minWidth: 170 },
  { key: 'produce', title: t('gmData.produce.scenario'), minWidth: 140 },
  { key: 'grade', title: t('gmData.produce.grade'), width: 150 },
  { key: 'stats', title: 'Vo / Da / Vi', width: 170 },
])

const tableRows = computed(() => props.rows as unknown as Record<string, unknown>[])
const r = (row: Record<string, unknown>) => row as unknown as ProduceRow

const cardImage = (row: ProduceRow) => catalogImage(ctx.game.value, 'idolCardSkin', { id: row.idolCardId })
const charaName = (row: ProduceRow) =>
  row.characterId ? lookup.name('character', row.characterId) : lookup.name('idolCard', row.idolCardId)
const produceName = (row: ProduceRow) => (row.produceId ? lookup.name('produce', row.produceId) : '—')
const groupName = (row: ProduceRow) => {
  const group = row.produceGroupId || lookup.entry('produce', row.produceId)?.parentId || ''
  return group ? lookup.name('produceGroup', group) : '—'
}
const playerLink = (pid: string) => `/game/gm/${encodeURIComponent(ctx.serverId.value)}/players/${encodeURIComponent(pid)}`
</script>

<style scoped>
.ptab {
  min-width: 0;
}
.ptab__strong {
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.ptab__dim {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.ptab__player {
  display: inline-flex;
  flex-direction: column;
  min-width: 0;
  text-decoration: none;
  color: inherit;
}
.ptab__player:hover .ptab__strong {
  color: var(--m-pInk);
}
.ptab__idol {
  display: inline-flex;
  align-items: center;
  gap: 10px;
}
.ptab__card {
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: 10px;
}
.ptab__stack {
  display: inline-flex;
  flex-direction: column;
}
.ptab__grade {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.ptab__cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.ptab__mcard {
  display: flex;
  gap: 12px;
  padding: 12px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.ptab__mcard.is-clickable {
  cursor: pointer;
}
.ptab__mimg {
  flex: none;
  width: 52px;
  height: 52px;
  border-radius: 12px;
}
.ptab__mbody {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.ptab__mline {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.ptab__mfoot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 13px;
  font-weight: 700;
}
.ptab__mfoot .ptab__player {
  flex-direction: row;
  color: var(--m-pInk);
}
</style>
