<!-- 登录会话：登录记录（从新到旧）：登录时间、时长（在线的那条显示「在线 …」）、平台与设备、地址；
     在线的那条可以强制下线（player.kick，走操作对话框）。手机端为卡片。 -->
<template>
  <div class="pv">
    <div v-if="list.error.value" class="pv__error">{{ list.error.value }}</div>

    <DataTable
      v-if="!menuStore.isMobile"
      :columns="columns"
      :rows="rows"
      row-key="id"
      :loading="list.loading.value"
      :empty-text="t('gmData.sessions.empty')"
    >
      <template #cell-login="{ row }">
        <span class="pv__strong m-num">{{ dateTime(s(row).loginAt) }}</span>
      </template>
      <template #cell-duration="{ row }">
        <StatusPill v-if="s(row).live" variant="success" :label="t('gmData.sessions.liveFor', { d: duration(Date.now() - s(row).loginAt) })" />
        <span v-else class="pv__dim m-num">{{ s(row).lastSeenAt ? duration(s(row).lastSeenAt - s(row).loginAt) : '—' }}</span>
      </template>
      <template #cell-device="{ row }">
        <span class="pv__ellipsis">{{ deviceOf(s(row)) }}</span>
      </template>
      <template #cell-addr="{ row }">
        <span class="pv__dim m-mono">{{ s(row).remoteAddr || '—' }}</span>
      </template>
      <template #cell-op="{ row }">
        <div class="pv__acts">
          <UButton v-if="s(row).live && canKick" size="sm" color="error" variant="soft" @click="page.run('player.kick')">
            {{ kickTitle }}
          </UButton>
        </div>
      </template>
    </DataTable>

    <template v-else>
      <EmptyState v-if="!list.loading.value && !list.items.value.length" icon="HOutline:ClockIcon" :title="t('gmData.sessions.empty')" />
      <div v-else class="pv__mcards">
        <article v-for="x in list.items.value" :key="x.id" class="pv__mcard">
          <div class="pv__mbody">
            <span class="pv__mline">
              <span class="pv__strong m-num">{{ dateTime(x.loginAt) }}</span>
              <StatusPill v-if="x.live" variant="success" :label="t('gmData.sessions.liveFor', { d: duration(Date.now() - x.loginAt) })" />
              <span v-else class="pv__dim m-num">{{ x.lastSeenAt ? duration(x.lastSeenAt - x.loginAt) : '—' }}</span>
            </span>
            <span class="pv__dim">{{ deviceOf(x) }}</span>
            <span class="pv__dim m-mono">{{ x.remoteAddr || '—' }}</span>
          </div>
          <AppIconButton
            v-if="x.live && canKick"
            icon="HOutline:ArrowLeftStartOnRectangleIcon"
            :label="kickTitle"
            :box="44"
            class="sess__kick"
            @click="page.run('player.kick')"
          />
        </article>
      </div>
    </template>

    <div v-if="list.total.value" class="pv__foot">
      <span class="m-num">{{ t('gmData.sessions.total', { n: list.total.value }) }}</span>
      <Pagination v-model:page="list.page.value" v-model:page-size="list.pageSize.value" :total="list.total.value" @change="list.load()" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { listPlayerSessions } from '@/api/gakumas'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { Session } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import { useGmFormat } from '../shared/format'
import { usePagedList } from '../shared/paging'

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()
const menuStore = useMenuStore()
const { dateTime, duration } = useGmFormat()

const list = usePagedList<Session>(
  (q) => listPlayerSessions({ serverId: ctx.serverId.value, pid: page.pid.value, cursor: q.cursor, limit: q.limit }),
  { pageSize: 20 },
)
watch(page.version, () => void list.load(), { immediate: true })

const columns = computed<DataTableColumn[]>(() => [
  { key: 'login', title: t('gmData.sessions.login'), width: 170 },
  { key: 'duration', title: t('gmData.sessions.duration'), width: 170 },
  { key: 'device', title: t('gmData.sessions.device'), minWidth: 220 },
  { key: 'addr', title: t('gmData.sessions.addr'), width: 170 },
  { key: 'op', title: '', width: 120, align: 'right' },
])
const rows = computed(() => list.items.value as unknown as Record<string, unknown>[])
const s = (row: Record<string, unknown>) => row as unknown as Session
const deviceOf = (x: Session) => [x.platform, x.device].filter(Boolean).join(' · ') || '—'

const canKick = computed(() => !!ctx.operation('player.kick'))
const kickTitle = computed(() => ctx.operation('player.kick')?.title || '')
</script>

<style scoped lang="scss">
@use '../shared/views';
.sess__kick {
  color: var(--m-bad);
}
</style>
