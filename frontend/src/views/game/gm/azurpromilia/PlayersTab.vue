<!-- 玩家：筛选（全部 / 在线 / 封禁中）与搜索、玩家表、勾选后的批量操作；点一行进入玩家详情。 -->
<template>
  <div class="azl">
    <ListToolbar
      v-model:status="filter"
      v-model:keyword="q"
      :chips="chips"
      :chips-label="t('azurGm.tabs.players')"
      :placeholder="t('azurGm.players.search')"
      searchable
      @search="pages.reset()"
    >
      <AppIconButton icon="HOutline:ArrowPathIcon" :label="t('azurGm.common.refresh')" :box="32" @click="pages.load()" />
    </ListToolbar>

    <p v-if="pages.error.value" class="azl__error">{{ t('azurGm.common.loadFailed', { msg: pages.error.value }) }}</p>
    <DataTable
      v-model="selected"
      :columns="columns"
      :rows="rows"
      row-key="pid"
      selectable
      :loading="pages.loading.value"
      :empty-text="t('azurGm.common.empty')"
      @row-click="open"
    >
      <template #cell-name="{ row }">
        <span class="azl__name">{{ row.name || t('azurGm.common.noCharacter') }}</span>
      </template>
      <template #cell-state="{ row }">
        <StatusPill v-if="row.banned" variant="error" :label="t('azurGm.players.banned')" />
        <StatusPill v-else :variant="row.online ? 'success' : 'neutral'" :label="row.online ? t('azurGm.players.online') : t('azurGm.players.offline')" dot />
      </template>
    </DataTable>
    <CursorPager
      :page="pages.page.value"
      :total="pages.total.value"
      :has-prev="pages.hasPrev.value"
      :has-next="pages.hasNext.value"
      :loading="pages.loading.value"
      @prev="pages.prev()"
      @next="pages.next()"
    />

    <div v-if="selected.length" class="azl__bar">
      <span>{{ t('azurGm.players.selected', { n: selected.length }) }}</span>
      <UButton size="sm" color="neutral" variant="soft" @click="run('player.mail')">{{ t('azurGm.player.actions.mail') }}</UButton>
      <UButton size="sm" color="neutral" variant="soft" @click="run('player.kick')">{{ t('azurGm.player.actions.kick') }}</UButton>
      <UButton size="sm" color="neutral" variant="soft" @click="run('player.ban')">{{ t('azurGm.player.actions.ban') }}</UButton>
      <UButton size="sm" color="neutral" variant="soft" @click="run('player.unban')">{{ t('azurGm.player.actions.unban') }}</UButton>
      <UButton size="sm" :to="{ query: { tab: 'operations', pid: selected.join(',') } }">{{ t('azurGm.players.toOperations') }}</UButton>
      <UButton size="sm" color="neutral" variant="ghost" @click="selected = []">{{ t('azurGm.players.clear') }}</UButton>
    </div>

    <FormDialog v-model="dialog" :title="op ? t(`gm.commands.azurpromilia.${op}`) : ''" :width="640" :show-footer="false">
      <OperationForm v-if="op" :server-id="ctx.serverId.value" :pids="selected" :fixed="op" @done="pages.load()" />
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import * as api from '@/api/azurpromilia'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { Player, PlayerCounts } from '@/types/v1/azurpromilia/types'
import type { GmPlayerLink } from '@/types/v1/game_gm'
import { useGmContext } from '../context'
import CursorPager from './CursorPager.vue'
import OperationForm from './OperationForm.vue'
import { fmtTime, useCursorPages } from './shared'

type Filter = 'all' | 'online' | 'banned'
type Op = 'player.mail' | 'player.kick' | 'player.ban' | 'player.unban'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const ctx = useGmContext()

const filter = ref<Filter>('all')
const q = ref(typeof route.query.q === 'string' ? route.query.q : '')
const summary = ref<PlayerCounts>()
const links = ref<GmPlayerLink[]>([])
const selected = ref<string[]>([])

const pages = useCursorPages<Player>(async (cursor) => {
  const res = await api.listPlayers({ serverId: ctx.serverId.value, q: q.value.trim(), filter: filter.value, cursor, limit: 20 })
  summary.value = res.summary
  links.value = res.links
  return res
})

const chips = computed(() => [
  { label: t('azurGm.players.filters.all'), value: 'all' as Filter, count: summary.value?.all ?? null },
  { label: t('azurGm.players.filters.online'), value: 'online' as Filter, count: summary.value?.online ?? null },
  { label: t('azurGm.players.filters.banned'), value: 'banned' as Filter, count: summary.value?.banned ?? null },
])

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('azurGm.players.cols.name'), minWidth: 140 },
  { key: 'displayId', title: t('azurGm.players.cols.id'), width: 110 },
  { key: 'account', title: t('azurGm.players.cols.account'), minWidth: 140, ellipsis: true },
  { key: 'level', title: t('azurGm.players.cols.level'), width: 70, align: 'right' },
  { key: 'passport', title: t('azurGm.players.cols.passport'), minWidth: 110, ellipsis: true },
  { key: 'lastLogin', title: t('azurGm.players.cols.lastLogin'), width: 150 },
  { key: 'state', title: t('azurGm.players.cols.state'), width: 100 },
])

const rows = computed(() =>
  pages.items.value.map((p) => ({
    ...p,
    level: p.displayId ? p.level : '—',
    passport: links.value.find((l) => l.pid === p.pid)?.username || '—',
    lastLogin: fmtTime(p.lastLoginAt),
  })),
)

const open = (row: Record<string, unknown>) => {
  void router.push(`/game/gm/${encodeURIComponent(ctx.serverId.value)}/azurpromilia/players/${encodeURIComponent(String(row.pid))}`)
}

const dialog = ref(false)
const op = ref<Op>()
const run = (o: Op) => {
  op.value = o
  dialog.value = true
}

watch(filter, () => void pages.reset())
watch(
  () => route.query.q,
  (v) => {
    if (typeof v === 'string' && v !== q.value) {
      q.value = v
      void pages.reset()
    }
  },
)
void pages.reset()
onActivated(() => void pages.load())
</script>

<style scoped lang="scss">
.azl {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.azl__error {
  margin: 0;
  color: var(--m-bad);
  font-size: 13px;
}
.azl__name {
  font-weight: 500;
}
.azl__bar {
  position: sticky;
  bottom: 12px;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: var(--app-radius-lg);
  background: var(--m-ink);
  color: var(--m-s);
}
</style>
