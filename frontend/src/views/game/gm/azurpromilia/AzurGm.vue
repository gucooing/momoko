<!-- 蓝色星原的 GM 页面（/game/gm/:serverId?tab=…）：工作台、玩家、邮件补偿、兑换码、操作与日志。
     服务器切换与页头由 GM 页面外壳提供（../index.vue），这里只有页内标签与各标签的内容。 -->
<template>
  <div class="azg">
    <nav class="azg__tabs" :aria-label="t('gm.shell.tabsLabel')">
      <RouterLink
        v-for="tab in TABS"
        :key="tab.key"
        :to="{ path: route.path, query: { tab: tab.key } }"
        replace
        class="azg__tab"
        :class="{ 'is-on': current === tab.key }"
        :aria-current="current === tab.key ? 'page' : undefined"
      >
        <component :is="menuStore.iconComponents[tab.icon]" class="azg__tab-icon" />
        {{ t(`azurGm.tabs.${tab.key}`) }}
        <span v-if="tab.key === 'operations' && ctx.pendingApprovals.value" class="azg__badge m-num">{{ ctx.pendingApprovals.value }}</span>
      </RouterLink>
    </nav>
    <KeepAlive>
      <component :is="views[current]" :key="current" />
    </KeepAlive>
  </div>
</template>

<script setup lang="ts">
import { computed, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useGmContext } from '../context'
import HomeTab from './HomeTab.vue'
import PlayersTab from './PlayersTab.vue'
import MailTab from './MailTab.vue'
import CodesTab from './CodesTab.vue'
import OperationsTab from './OperationsTab.vue'

type TabKey = 'home' | 'players' | 'mail' | 'codes' | 'operations'

const TABS: { key: TabKey; icon: string }[] = [
  { key: 'home', icon: 'HOutline:PresentationChartLineIcon' },
  { key: 'players', icon: 'HOutline:UsersIcon' },
  { key: 'mail', icon: 'HOutline:EnvelopeIcon' },
  { key: 'codes', icon: 'HOutline:TicketIcon' },
  { key: 'operations', icon: 'HOutline:CommandLineIcon' },
]

const views: Record<TabKey, Component> = {
  home: HomeTab,
  players: PlayersTab,
  mail: MailTab,
  codes: CodesTab,
  operations: OperationsTab,
}

const { t } = useI18n()
const route = useRoute()
const menuStore = useMenuStore()
const ctx = useGmContext()

const current = computed<TabKey>(() => {
  const q = typeof route.query.tab === 'string' ? route.query.tab : ''
  return (TABS.find((x) => x.key === q)?.key ?? 'home') as TabKey
})
</script>

<style scoped lang="scss">
.azg {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}
.azg__tabs {
  display: flex;
  gap: 4px;
  overflow-x: auto;
  scrollbar-width: none;
  border-bottom: 1px solid var(--el-border-color-light);
}
.azg__tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  border-bottom: 2px solid transparent;
  color: var(--el-text-color-secondary);
  text-decoration: none;
  white-space: nowrap;
  &.is-on {
    border-bottom-color: var(--m-p);
    color: var(--el-text-color-primary);
    font-weight: 600;
  }
}
.azg__tab-icon {
  width: 16px;
  height: 16px;
}
.azg__badge {
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 9px;
  background: var(--m-bad);
  color: var(--m-s);
  font-size: 11px;
  line-height: 18px;
  text-align: center;
}
</style>
