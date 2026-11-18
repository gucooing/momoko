<!-- GM 运营 · 操作与日志：「执行操作」先选玩家再执行（RunPanel），「玩家权限」管理玩家在通行证自助服务里能执行哪些操作
     （PermissionsPanel），「GM 记录」是这台服务器的操作记录与待审批（LogsPanel），「客户端日志」是这台服务器的玩家上传的
     客户端错误日志（ClientLogsPanel）；当前页面记在 ?view=（run / permissions / logs / clientLogs）。服务端没有可对玩家执行的
     操作时不显示前两个，默认进入 GM 记录。切换页面时各页保留状态。 -->
<template>
  <div class="gop game-skin">
    <div class="gop__head">
      <h2 class="gop__h">{{ t('gmOps.run.title') }}</h2>
      <nav class="gop__seg" role="tablist">
        <button
          v-for="v in views"
          :key="v.key"
          type="button"
          role="tab"
          class="gop__seg-btn"
          :class="{ 'is-on': view === v.key }"
          :aria-selected="view === v.key"
          @click="setView(v.key)"
        >
          {{ v.label }}
          <span v-if="v.badge" class="gop__seg-badge m-num">{{ v.badge }}</span>
        </button>
      </nav>
    </div>
    <KeepAlive>
      <component :is="panel" :key="view" />
    </KeepAlive>
  </div>
</template>

<script setup lang="ts">
import { computed, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useGmContext } from '../../context'
import { hasPlayerOperations } from '../../tabs'
import ClientLogsPanel from './ClientLogsPanel.vue'
import LogsPanel from './LogsPanel.vue'
import PermissionsPanel from './PermissionsPanel.vue'
import RunPanel from './RunPanel.vue'

defineOptions({ name: 'GmOperationsTab' })

type View = 'run' | 'permissions' | 'logs' | 'clientLogs'

const PANELS: Record<View, Component> = { run: RunPanel, permissions: PermissionsPanel, logs: LogsPanel, clientLogs: ClientLogsPanel }

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const ctx = useGmContext()

const views = computed<{ key: View; label: string; badge?: number }[]>(() => [
  ...(hasPlayerOperations(ctx)
    ? [
        { key: 'run' as const, label: t('gmOps.run.runView') },
        { key: 'permissions' as const, label: t('gmOps.run.permissionsView') },
      ]
    : []),
  { key: 'logs', label: t('gm.tabs.logs'), badge: ctx.pendingApprovals.value || undefined },
  { key: 'clientLogs', label: t('clientLogs.title') },
])
/** 默认页（地址里不写 ?view=）：有可对玩家执行的操作时是「执行操作」，否则是「GM 记录」 */
const first = computed<View>(() => views.value[0]?.key ?? 'logs')
const view = computed<View>(() => views.value.find((v) => v.key === route.query.view)?.key || first.value)
const panel = computed(() => PANELS[view.value])
const setView = (key: View) => void router.replace({ query: { ...route.query, view: key === first.value ? undefined : key } })
</script>

<style scoped>
.gop {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.gop__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px 16px;
}
.gop__h {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 30px;
  font-weight: 800;
  letter-spacing: -0.01em;
}
.gop__seg {
  display: inline-flex;
  gap: 4px;
  padding: 4px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.gop__seg-btn {
  min-height: 34px;
  padding: 0 16px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
}
.gop__seg-btn.is-on {
  background: var(--m-ink);
  color: var(--m-bg);
}
.gop__seg-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  margin-left: 6px;
  padding: 0 5px;
  border-radius: 999px;
  background: var(--m-bad);
  color: var(--m-s);
  font-size: 11px;
  font-weight: 800;
}
@media (width < 768px) {
  .gop__h {
    font-size: 24px;
  }
}
</style>
