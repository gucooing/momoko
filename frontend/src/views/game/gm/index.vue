<!-- GM 运营：一个菜单 = 一个页面。地址 /game/gm/:serverId?tab=…；页头左侧是服务器切换（搜索 + 最近查看），
     下面是页内标签（按服务端 manifest 出现）。没有带服务器时进入最近查看的那台。 -->
<template>
  <div class="gm" :data-game="ctx.game.value || undefined">
    <template v-if="serverId">
      <div class="gm__head">
        <GameServerPicker :model-value="serverId" :selected="ctx.server.value" :search="searchServers" @select="switchServer">
          <template #trigger="{ open }">
            <button type="button" class="gm__switch" :aria-expanded="open" :aria-label="t('gm.shell.switchServer')">
              <span class="gm__switch-avatar" aria-hidden="true">{{ initial }}</span>
              <span class="gm__switch-text">
                <span class="gm__switch-name">{{ ctx.server.value?.name || '…' }}</span>
                <span class="gm__switch-sub">{{ subline }}</span>
              </span>
              <component :is="menuStore.iconComponents['HOutline:ChevronUpDownIcon']" class="gm__switch-icon" />
            </button>
          </template>
        </GameServerPicker>
        <span class="gm__spacer" />
        <AppIconButton icon="HOutline:ArrowPathIcon" :label="t('gm.shell.refresh')" :box="40" @click="ctx.refresh()" />
      </div>

      <nav v-if="tabs.length" class="gm__tabs" :aria-label="t('gm.shell.tabsLabel')">
        <RouterLink
          v-for="tab in tabs"
          :key="tab.key"
          :to="{ path: route.path, query: { tab: tab.key } }"
          replace
          class="gm__tab"
          :class="{ 'is-on': current?.key === tab.key }"
          :aria-current="current?.key === tab.key ? 'page' : undefined"
        >
          <component :is="menuStore.iconComponents[tab.icon]" class="gm__tab-icon" />
          {{ t(tab.labelKey) }}
          <span v-if="tab.key === 'logs' && ctx.pendingApprovals.value" class="gm__tab-badge m-num">{{ ctx.pendingApprovals.value }}</span>
        </RouterLink>
      </nav>

      <EmptyState
        v-if="ctx.error.value && !ctx.manifest.value"
        icon="HOutline:ExclamationTriangleIcon"
        :title="t('gm.shell.loadFailed')"
        :description="ctx.error.value"
      />
      <EmptyState
        v-else-if="ctx.server.value && !ctx.server.value.mgpConfigured"
        icon="HOutline:SignalIcon"
        :title="t('gm.shell.noMgpTitle')"
        :description="t('gm.shell.noMgpDesc')"
      />
      <EmptyState
        v-else-if="ctx.server.value && !ctx.manifest.value && !ctx.loading.value"
        icon="HOutline:SignalIcon"
        :title="t('gm.shell.noManifestTitle')"
        :description="ctx.statusError.value || t('gm.shell.noManifestDesc')"
      />
      <KeepAlive v-else-if="current">
        <component :is="currentView" :key="`${serverId}:${current.key}`" />
      </KeepAlive>
    </template>

    <EmptyState
      v-else-if="resolved"
      icon="HOutline:PuzzlePieceIcon"
      :title="t('gm.shell.noServerTitle')"
      :description="t('gm.shell.noServerDesc')"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import '@/games'
import { searchGmServers } from '@/api/gameGm'
import type { GameServerBrief } from '@/types/v1/game_common'
import GameServerPicker from '../components/GameServerPicker.vue'
import { provideGmContext } from './context'
import { GM_TABS, gmTabView } from './tabs'

defineOptions({ name: 'GameGmView' })

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const menuStore = useMenuStore()

const serverId = computed(() => (typeof route.params.serverId === 'string' ? route.params.serverId : ''))
const serverRef = ref('')
watch(serverId, (id) => {
  if (id) serverRef.value = id
}, { immediate: true })

const ctx = provideGmContext(serverRef)
watch(serverRef, () => void ctx.refresh(), { immediate: true })

const searchServers = async (q: string) => (await searchGmServers({ q, limit: 20 })).data

// 没有带服务器：进入最近查看的那台，没有最近查看时进入第一台。
const resolved = ref(false)
const resolveDefault = async () => {
  if (serverId.value) return
  try {
    const res = await searchServers('')
    const first = res.recent?.[0] || res.items?.[0]
    if (first && !serverId.value) {
      await router.replace({ path: `/game/gm/${encodeURIComponent(first.id)}`, query: route.query })
    }
  } finally {
    resolved.value = true
  }
}
watch(
  () => route.path,
  (path) => {
    if (path === '/game/gm') void resolveDefault()
  },
  { immediate: true },
)

const switchServer = (s: GameServerBrief | undefined) => {
  if (!s || s.id === serverId.value) return
  void router.push({ path: `/game/gm/${encodeURIComponent(s.id)}`, query: { tab: current.value?.key } })
}

const tabs = computed(() => GM_TABS.filter((tab) => gmTabView(tab.key) && tab.visible(ctx)))
const current = computed(() => {
  const wanted = typeof route.query.tab === 'string' ? route.query.tab : ''
  return tabs.value.find((tab) => tab.key === wanted) || tabs.value[0]
})
const currentView = computed(() => (current.value ? gmTabView(current.value.key) : undefined))

const initial = computed(() => (ctx.server.value?.name || '?').slice(0, 1).toUpperCase())
const subline = computed(() => {
  const s = ctx.server.value
  if (!s) return ''
  const parts = [t(`game.server.mode.${s.mode}`)]
  if (ctx.status.value) parts.push(t('game.server.online', { n: ctx.status.value.online }))
  parts.push(ctx.status.value ? t('gm.shell.mgpOk') : ctx.statusError.value ? t('gm.shell.mgpDown') : t('gm.shell.mgpUnknown'))
  return parts.join(' · ')
})
</script>

<style scoped lang="scss">
.gm {
  display: flex;
  flex-direction: column;
  gap: 18px;
  min-width: 0;
}
.gm__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}
.gm__spacer {
  flex: 1;
}
.gm__switch {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  padding: 6px 14px 6px 6px;
  border-radius: 999px;
  border: 1.5px solid var(--m-gLine, var(--el-border-color-light));
  background: var(--m-gSoft, var(--el-fill-color-light));
  color: var(--el-text-color-primary);
  font: inherit;
  cursor: pointer;
  max-width: 100%;
}
.gm__switch-avatar {
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--m-g, var(--m-p));
  color: var(--m-gOn, var(--m-pOn));
  font-family: var(--app-font-display);
  font-weight: 800;
  font-size: 18px;
}
.gm__switch-text {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  min-width: 0;
}
.gm__switch-name {
  font-size: 15px;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 60vw;
}
.gm__switch-sub {
  font-size: 12px;
  font-weight: 700;
  color: var(--m-gInk, var(--el-text-color-secondary));
}
.gm__switch-icon {
  width: 16px;
  height: 16px;
  color: var(--m-gInk, var(--el-text-color-secondary));
}
.gm__tabs {
  display: flex;
  gap: 6px;
  overflow-x: auto;
  padding: 6px;
  border-radius: 999px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  align-self: flex-start;
  max-width: 100%;
  scrollbar-width: none;
}
.gm__tabs::-webkit-scrollbar {
  display: none;
}
.gm__tab {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  color: var(--el-text-color-secondary);
  font-weight: 700;
  font-size: 14px;
  text-decoration: none;
  white-space: nowrap;
}
.gm__tab:hover {
  color: var(--el-text-color-primary);
}
.gm__tab.is-on {
  background: var(--m-g, var(--m-p));
  color: var(--m-gOn, var(--m-pOn));
}
.gm__tab-icon {
  width: 16px;
  height: 16px;
}
.gm__tab-badge {
  padding: 0 7px;
  border-radius: 999px;
  background: var(--m-bad);
  color: var(--m-s);
  font-size: 11px;
}
</style>
