<!-- 操作与日志：「执行操作」左边是玩家列表（搜索、勾选，翻页后勾选保留；?pid=a,b 预先勾选），右边选操作并对勾选的全部玩家执行；
     「玩家权限」（?view=permissions）管理玩家在通行证自助服务里能执行哪些操作；「GM 记录」（?view=logs）是待审批、筛选与记录表（与学マス 共用）。 -->
<template>
  <div class="azx">
    <div class="azx__seg" role="tablist">
      <RouterLink :to="{ query: { tab: 'operations' } }" replace :class="{ 'is-on': view === 'run' }">{{ t('azurGm.ops.run') }}</RouterLink>
      <RouterLink :to="{ query: { tab: 'operations', view: 'permissions' } }" replace :class="{ 'is-on': view === 'permissions' }">
        {{ t('azurGm.ops.permissions') }}
        <span v-if="ctx.selfServiceEnabled.value" class="azx__count m-num">{{ ctx.selfServiceEnabled.value }}</span>
      </RouterLink>
      <RouterLink :to="{ query: { tab: 'operations', view: 'logs' } }" replace :class="{ 'is-on': view === 'logs' }">
        {{ t('azurGm.ops.logs') }}
        <span v-if="ctx.pendingApprovals.value" class="azx__badge m-num">{{ ctx.pendingApprovals.value }}</span>
      </RouterLink>
    </div>

    <LogsPanel v-if="view === 'logs'" />
    <PermissionsPanel v-else-if="view === 'permissions'" />
    <div v-else class="azx__run">
      <AppPanel :title="t('azurGm.ops.pick')" :caption="t('azurGm.ops.picked', { n: picked.length })">
        <ListToolbar v-model:keyword="q" :placeholder="t('azurGm.players.search')" searchable @search="pages.reset()" />
        <ul class="azx__list">
          <li v-for="p in pages.items.value" :key="p.pid">
            <label>
              <input type="checkbox" :checked="picked.includes(p.pid)" @change="toggle(p.pid)" />
              <span class="azx__name">{{ p.name || t('azurGm.common.noCharacter') }}</span>
              <span class="azx__id m-num">{{ p.displayId || p.pid }}</span>
              <StatusPill v-if="p.banned" variant="error" :label="t('azurGm.players.banned')" />
              <StatusPill v-else-if="p.online" variant="success" :label="t('azurGm.players.online')" dot />
            </label>
          </li>
        </ul>
        <CursorPager
          :page="pages.page.value"
          :total="pages.total.value"
          :has-prev="pages.hasPrev.value"
          :has-next="pages.hasNext.value"
          :loading="pages.loading.value"
          @prev="pages.prev()"
          @next="pages.next()"
        />
      </AppPanel>
      <AppPanel :title="t('azurGm.ops.choose')">
        <p v-if="!picked.length" class="azx__hint">{{ t('azurGm.ops.pickHint') }}</p>
        <OperationForm :server-id="ctx.serverId.value" :pids="picked" @done="pages.load()" />
      </AppPanel>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import * as api from '@/api/azurpromilia'
import type { Player } from '@/types/v1/azurpromilia/types'
import { useGmContext } from '../context'
import LogsPanel from '../tabs/operations/LogsPanel.vue'
import PermissionsPanel from '../tabs/operations/PermissionsPanel.vue'
import CursorPager from './CursorPager.vue'
import OperationForm from './OperationForm.vue'
import { useCursorPages } from './shared'

const { t } = useI18n()
const route = useRoute()
const ctx = useGmContext()

const view = computed(() => (route.query.view === 'logs' || route.query.view === 'permissions' ? route.query.view : 'run'))
const q = ref('')
const picked = ref<string[]>([])

const pages = useCursorPages<Player>((cursor) => api.listPlayers({ serverId: ctx.serverId.value, q: q.value.trim(), cursor, limit: 20 }))

const toggle = (pid: string) => {
  picked.value = picked.value.includes(pid) ? picked.value.filter((x) => x !== pid) : [...picked.value, pid]
}

watch(
  () => route.query.pid,
  (v) => {
    if (typeof v === 'string' && v) picked.value = [...new Set(v.split(',').filter(Boolean))]
  },
  { immediate: true },
)

watch(
  view,
  (v) => {
    if (v === 'run' && !pages.items.value.length) void pages.reset()
  },
  { immediate: true },
)
</script>

<style scoped lang="scss">
.azx {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.azx__seg {
  display: inline-flex;
  gap: 4px;
  padding: 3px;
  border-radius: var(--app-radius-pill);
  background: var(--el-fill-color-light);
  align-self: flex-start;
  a {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 4px 14px;
    border-radius: var(--app-radius-pill);
    color: var(--el-text-color-secondary);
    text-decoration: none;
    &.is-on {
      background: var(--el-bg-color);
      color: var(--el-text-color-primary);
    }
  }
}
.azx__badge {
  padding: 0 6px;
  border-radius: 999px;
  background: var(--m-bad);
  color: var(--m-s);
  font-size: 11px;
}
.azx__count {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.azx__run {
  display: grid;
  grid-template-columns: minmax(280px, 1fr) minmax(0, 1.4fr);
  gap: 14px;
  @media (max-width: 960px) {
    grid-template-columns: 1fr;
  }
}
.azx__list {
  list-style: none;
  margin: 10px 0 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  label {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 8px;
    border-radius: var(--app-radius-xs);
    cursor: pointer;
    &:hover {
      background: var(--el-fill-color-light);
    }
  }
}
.azx__name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.azx__id {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.azx__hint {
  margin: 0 0 10px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
</style>
