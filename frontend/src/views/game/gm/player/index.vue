<!-- 玩家详情：/game/gm/:serverId/players/:pid?tab=…（独立路由，自己提供 GM 上下文）。
     返回「玩家」→ GM 运营标签 → 玩家卡（担当、昵称、在线状态、ID、通行证、等级经验、钱包、常用指令）
     → 分页（来自 manifest 的玩家视图 + Momoko 的 GM 记录）。所有数据来自 MGP / Momoko，指令按 manifest 出现。 -->
<template>
  <div class="gp game-skin" :data-game="ctx.game.value || undefined">
    <div class="gp__top">
      <RouterLink class="gp__back" :to="playersLink">
        <component :is="menuStore.iconComponents['HOutline:ChevronLeftIcon']" class="gp__back-icon" />
        {{ t('gmData.player.back') }}
      </RouterLink>
      <span v-if="ctx.server.value" class="gp__server">{{ ctx.server.value.name }}</span>
      <span class="gp__spacer" />
      <AppIconButton icon="HOutline:ArrowPathIcon" :label="t('gmData.common.refresh')" :box="40" @click="reloadAll" />
    </div>

    <nav v-if="gmTabs.length" class="gp__gmtabs" :aria-label="t('gm.shell.tabsLabel')">
      <RouterLink
        v-for="tab in gmTabs"
        :key="tab.key"
        :to="{ path: gmPath, query: { tab: tab.key } }"
        class="gp__gmtab"
        :class="{ 'is-on': tab.key === 'players' }"
      >
        <component :is="menuStore.iconComponents[tab.icon]" class="gp__gmtab-icon" />
        {{ t(tab.labelKey) }}
      </RouterLink>
    </nav>

    <EmptyState
      v-if="ctx.error.value && !ctx.manifest.value"
      icon="HOutline:ExclamationTriangleIcon"
      :title="t('gm.shell.loadFailed')"
      :description="ctx.error.value"
    />
    <EmptyState
      v-else-if="error && !player"
      icon="HOutline:UserIcon"
      :title="t('gmData.player.loadFailed')"
      :description="error"
    >
      <template #action>
        <UButton color="neutral" variant="outline" @click="loadPlayer">{{ t('gmData.common.retry') }}</UButton>
      </template>
    </EmptyState>
    <div v-else-if="!player || !ctx.manifest.value" class="gp__skeleton" aria-busy="true" />

    <template v-else>
      <section class="gp-hero">
        <div class="gp-hero__main">
          <div class="gp-hero__id">
            <span class="gp-hero__avatar">
              <GameImage v-if="bust" :src="bust" :alt="homeName" class="gp-hero__bust" eager />
              <PlayerAvatar v-else :idol-card-id="player.avatarIdolCardId" :name="player.name" :size="88" />
            </span>
            <div class="gp-hero__text">
              <div class="gp-hero__name-row">
                <h1 class="gp-hero__name">{{ player.name || t('gm.player.noName') }}</h1>
                <StatusPill :variant="statusPill.variant" :label="statusPill.label" />
              </div>
              <div class="gp-hero__meta">
                <span class="m-num">User ID {{ player.displayId }}</span>
                <span>{{ t('gmData.player.account', { account: player.account }) }}</span>
                <span v-if="homeName">{{ t('gmData.player.home', { name: homeName }) }}</span>
                <RouterLink v-if="link" class="gp-hero__link" :to="passportLink">
                  <component :is="menuStore.iconComponents['HOutline:LinkIcon']" />
                  {{ t('gmData.player.passport', { name: link.username || link.name }) }}
                </RouterLink>
                <span v-else>{{ t('gmData.players.unlinked') }}</span>
              </div>
              <div v-if="overview" class="gp-hero__level">
                <span class="gp-hero__lv m-num">PLv {{ overview.level }}</span>
                <span class="gp-hero__bar" role="progressbar" :aria-valuenow="expPct" aria-valuemin="0" aria-valuemax="100">
                  <span :style="{ width: `${expPct}%` }" />
                </span>
                <span class="gp-hero__exp m-num">{{ overview.nextLevelExp ? `EXP ${expPct}%` : t('gmData.player.maxLevel') }}</span>
              </div>
            </div>
          </div>

          <div v-if="overview" class="gp-hero__wallet">
            <div v-for="w in wallet" :key="w.key" class="gp-hero__cur">
              <GameImage v-if="w.image" :src="w.image" class="gp-hero__cur-img" width="36" height="36" />
              <span v-else class="gp-hero__cur-icon" aria-hidden="true">
                <component :is="menuStore.iconComponents[w.icon]" />
              </span>
              <span class="gp-hero__cur-text">
                <span class="gp-hero__cur-label">{{ w.label }}</span>
                <span class="gp-hero__cur-value m-num">{{ w.value }}</span>
              </span>
            </div>
          </div>

          <div class="gp-hero__actions">
            <UButton v-if="ctx.hasCommand('gift.send')" color="primary" icon="i-lucide-gift" @click="openGift">
              {{ t('gmData.player.sendGift') }}
            </UButton>
            <AppDropdown v-if="resourceCommands.length" align="start" :width="240">
              <template #trigger>
                <UButton color="neutral" variant="outline" icon="i-lucide-sliders-horizontal" trailing-icon="i-lucide-chevron-down">
                  {{ t('gmData.player.adjust') }}
                </UButton>
              </template>
              <template #default="{ close }">
                <div class="gp-menu">
                  <button
                    v-for="cmd in resourceCommands"
                    :key="cmd.id"
                    type="button"
                    class="gp-menu__item"
                    @click="close(); run(cmd.id)"
                  >
                    <span class="gp-menu__title">{{ cmd.title }}</span>
                    <GmRiskPill :risk="cmd.risk" />
                  </button>
                </div>
              </template>
            </AppDropdown>
            <UButton
              v-if="ctx.hasCommand('session.kick')"
              color="error"
              variant="outline"
              icon="i-lucide-log-out"
              :disabled="!player.online"
              @click="run('session.kick')"
            >
              {{ ctx.command('session.kick')?.title }}
            </UButton>
            <UButton v-if="ctx.hasCommand('archive.export')" color="neutral" variant="outline" icon="i-lucide-download" @click="run('archive.export')">
              {{ ctx.command('archive.export')?.title }}
            </UButton>
            <ActionMenu v-if="moreItems.length" :items="moreItems" :label="t('gmData.player.more')" @select="run" />
          </div>
        </div>
        <div v-if="art" class="gp-hero__art">
          <GameImage :src="art" class="gp-hero__art-img" eager />
          <span class="gp-hero__art-tag">{{ t('gmData.player.lentMemory') }}</span>
        </div>
      </section>

      <nav class="gp-tabs" role="tablist" :aria-label="t('gmData.player.tabsLabel')">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          type="button"
          role="tab"
          class="gp-tabs__tab"
          :class="{ 'is-on': tab.id === current?.id }"
          :aria-selected="tab.id === current?.id"
          @click="goto(tab.id)"
        >
          {{ tab.title }}
          <span v-if="tab.count" class="gp-tabs__count m-num">{{ tab.count }}</span>
        </button>
      </nav>

      <KeepAlive>
        <component :is="current.component" v-if="current" :key="current.id" :view="current.view" />
      </KeepAlive>
    </template>

    <GiftDrawer v-model="giftOpen" :players="giftPlayers" @done="afterCommand" />
    <GmCommandDialog
      v-model="dialog.open"
      :command="dialog.command"
      :targets="dialog.targets"
      :initial-params="dialog.params"
      :summary="dialog.summary"
      @done="afterCommand"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, shallowRef, watch, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import '@/games'
import { getGmPlayer } from '@/api/gameGm'
import { getRequestErrorMessage } from '@/utils/request'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import type { ManifestCommand, ManifestView } from '@/games/mgp'
import type { PlayerDetail } from '@/games/gakumas/types'
import type { GmPlayerLink } from '@/types/v1/game_gm'
import { provideGmContext } from '../context'
import { GM_TABS, gmTabView } from '../tabs'
import GmCommandDialog from '../components/GmCommandDialog.vue'
import GmRiskPill from '../components/GmRiskPill.vue'
import type { GmTargetRef } from '../components/types'
import { GM_LOG_TAB, providePlayerPage } from './context'
import GiftDrawer from './GiftDrawer.vue'
import PlayerAvatar from './shared/PlayerAvatar.vue'
import { catalogImage, charaBust, idolCardArt, isPermanentBan } from './shared/game'
import { useGmFormat } from './shared/format'
import { catalogLookupFor } from './shared/lookup'
import { gmLogView, playerViewDef } from './views'

defineOptions({ name: 'GameGmPlayerView' })

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const menuStore = useMenuStore()
const { num, ago, date, pct } = useGmFormat()

// 页面实例按路径缓存：服务器与玩家在这个实例里不会变
const pagePath = route.path
const serverRef = ref(typeof route.params.serverId === 'string' ? route.params.serverId : '')
const pid = ref(typeof route.params.pid === 'string' ? route.params.pid : '')

const ctx = provideGmContext(serverRef)
void ctx.refresh()
const lookup = catalogLookupFor(ctx)

const gmPath = computed(() => `/game/gm/${encodeURIComponent(serverRef.value)}`)
const playersLink = computed(() => ({ path: gmPath.value, query: { tab: 'players' } }))
const gmTabs = computed(() => GM_TABS.filter((tab) => gmTabView(tab.key) && tab.visible(ctx)))

// ---------- 玩家 ----------
const player = shallowRef<PlayerDetail>()
const link = shallowRef<GmPlayerLink>()
const gmCount = ref(0)
const error = ref('')
const version = ref(0)

const loadPlayer = async () => {
  error.value = ''
  try {
    const { data } = await getGmPlayer(serverRef.value, pid.value)
    player.value = (data.player as PlayerDetail | undefined) || undefined
    link.value = data.link || undefined
    gmCount.value = data.gmExecutions || 0
    if (!player.value) error.value = t('gmData.player.notFound')
  } catch (e) {
    error.value = getRequestErrorMessage(e)
  }
}
void loadPlayer()

const reloadAll = async () => {
  await Promise.all([ctx.refresh(), loadPlayer()])
  version.value++
}

const overview = computed(() => player.value?.overview)
const homeName = computed(() => (overview.value?.homeCharacterId ? lookup.name('character', overview.value.homeCharacterId) : ''))
const bust = computed(() => charaBust(ctx.game.value, overview.value?.homeCharacterId || ''))
const art = computed(() => idolCardArt(ctx.game.value, player.value?.avatarIdolCardId || ''))
const passportLink = computed(() => ({ path: '/game/passport', query: { user: link.value?.passportUserId } }))

const expPct = computed(() => {
  const o = overview.value
  if (!o) return 0
  if (!o.nextLevelExp) return 100
  return Math.min(100, pct(o.levelExp, o.nextLevelExp))
})

const statusPill = computed(() => {
  const p = player.value
  if (!p) return { variant: 'neutral' as const, label: '' }
  if (p.bannedUntil) {
    return {
      variant: 'error' as const,
      label: isPermanentBan(p.bannedUntil) ? t('gmData.players.bannedForever') : t('gmData.players.bannedUntil', { date: date(p.bannedUntil) }),
    }
  }
  if (p.online) {
    const where = [p.platform, p.device].filter(Boolean).join(' · ')
    return { variant: 'success' as const, label: where ? t('gmData.players.onlineOn', { where }) : t('gmData.players.online') }
  }
  return {
    variant: 'neutral' as const,
    label: p.lastLoginAt ? t('gmData.players.offlineSince', { ago: ago(p.lastLoginAt) }) : t('gmData.players.neverLogin'),
  }
})

const wallet = computed(() => {
  const o = overview.value
  if (!o) return []
  const game = ctx.game.value
  return [
    {
      key: 'jewel',
      label: t('gmData.player.wallet.jewel'),
      value: `${num(o.jewelFree)} / ${num(o.jewelPaid + o.jewelWebStorePaid)}`,
      image: catalogImage(game, 'currency', { id: 'jewelFree' }),
      icon: '',
    },
    { key: 'money', label: t('gmData.player.wallet.money'), value: num(o.money), image: catalogImage(game, 'currency', { id: 'money' }), icon: '' },
    {
      key: 'ap',
      label: t('gmData.player.wallet.ap'),
      value: `${num(o.actionPoint)} / ${num(o.actionPointMax)}`,
      image: catalogImage(game, 'currency', { id: 'actionPoint' }),
      icon: '',
    },
    { key: 'fans', label: t('gmData.player.wallet.fans'), value: num(o.totalFanCount), image: '', icon: 'HOutline:UsersIcon' },
  ]
})

// ---------- 指令 ----------
const playerCommands = computed(() =>
  (ctx.manifest.value?.commands || []).filter((c) => c.target === 'player' || c.target === 'players'),
)
// 「调整资源」：与 jewel.adjust 同一分组的玩家指令（分组由服务端给出）
const resourceCommands = computed<ManifestCommand[]>(() => {
  const group = ctx.command('jewel.adjust')?.group
  return group ? playerCommands.value.filter((c) => c.group === group) : []
})
const SHOWN = ['gift.send', 'session.kick', 'archive.export']
const moreItems = computed<ActionMenuItem[]>(() =>
  playerCommands.value
    .filter((c) => !SHOWN.includes(c.id) && !resourceCommands.value.includes(c))
    .map((c) => ({ key: c.id, label: c.title, danger: c.risk === 'danger' })),
)

const self = computed<GmTargetRef[]>(() => (player.value ? [{ pid: player.value.pid, name: player.value.name }] : []))

const dialog = reactive<{
  open: boolean
  command?: ManifestCommand
  targets: GmTargetRef[]
  params: Record<string, unknown>
  summary: Record<string, string>
}>({ open: false, command: undefined, targets: [], params: {}, summary: {} })

const run = (commandId: string, initialParams: Record<string, unknown> = {}, summary: Record<string, string> = {}) => {
  const cmd = ctx.command(commandId)
  if (!cmd) return
  dialog.command = cmd
  dialog.targets = self.value
  dialog.params = initialParams
  dialog.summary = summary
  dialog.open = true
}

const giftOpen = ref(false)
const giftPlayers = computed(() =>
  player.value ? [{ pid: player.value.pid, name: player.value.name, avatarIdolCardId: player.value.avatarIdolCardId }] : [],
)
const openGift = () => {
  giftOpen.value = true
}

const afterCommand = () => {
  void loadPlayer()
  version.value++
}

// ---------- 分页 ----------
interface PlayerTab {
  id: string
  title: string
  view?: ManifestView
  component: Component
  count?: number
}

const tabs = computed<PlayerTab[]>(() => {
  const views = ctx.manifest.value?.views || []
  const ids = new Set(views.map((v) => v.id))
  const list: PlayerTab[] = []
  for (const v of views) {
    const def = playerViewDef(v.id)
    if (def.mergedInto && ids.has(def.mergedInto)) continue
    list.push({ id: v.id, title: v.title || t('game.unnamed'), view: v, component: def.component, count: player.value ? def.count?.(player.value) : undefined })
  }
  list.push({ id: GM_LOG_TAB, title: t('gmData.player.gmLog'), component: gmLogView, count: gmCount.value })
  return list
})

// 只在本页面可见时跟随地址里的 ?tab=（缓存的实例不跟着别的页面变）
const wanted = ref(typeof route.query.tab === 'string' ? route.query.tab : '')
watch(
  () => [route.path, route.query.tab] as const,
  ([path, tab]) => {
    if (path === pagePath) wanted.value = typeof tab === 'string' ? tab : ''
  },
)
const current = computed(() => tabs.value.find((tab) => tab.id === wanted.value) || tabs.value[0])

const goto = (tab: string) => {
  if (tab === current.value?.id) return
  void router.replace({ path: pagePath, query: { ...route.query, tab } })
}

providePlayerPage({
  pid,
  player,
  link,
  gmCount,
  version,
  run,
  openGift,
  goto,
  hasTab: (tab) => tabs.value.some((x) => x.id === tab),
  refresh: loadPlayer,
})
</script>

<style scoped lang="scss">
.gp {
  display: flex;
  flex-direction: column;
  gap: 18px;
  min-width: 0;
}
.gp__top {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.gp__back {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding-right: 8px;
  font-size: 14px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  text-decoration: none;
}
.gp__back:hover {
  color: var(--el-text-color-primary);
}
.gp__back-icon {
  width: 16px;
  height: 16px;
}
.gp__server {
  padding: 4px 12px;
  border-radius: 999px;
  background: var(--m-gSoft);
  color: var(--m-gInk);
  font-size: 12px;
  font-weight: 800;
  max-width: 60vw;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gp__spacer {
  flex: 1;
}
.gp__gmtabs {
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
.gp__gmtabs::-webkit-scrollbar {
  display: none;
}
.gp__gmtab {
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
.gp__gmtab:hover {
  color: var(--el-text-color-primary);
}
.gp__gmtab.is-on {
  background: var(--m-g);
  color: var(--m-gOn);
}
.gp__gmtab-icon {
  width: 16px;
  height: 16px;
}
.gp__skeleton {
  height: 280px;
  border-radius: var(--app-radius-lg);
  background: var(--el-fill-color-light);
  animation: gp-pulse 1.4s ease-in-out infinite;
}
@keyframes gp-pulse {
  50% {
    opacity: 0.55;
  }
}

/* ---------- 玩家卡 ---------- */
.gp-hero {
  display: flex;
  overflow: hidden;
  container-type: inline-size;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-card);
}
.gp-hero__main {
  flex: 1 1 560px;
  min-width: 0;
  padding: 22px 24px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.gp-hero__id {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 18px;
}
.gp-hero__avatar {
  flex: none;
  width: 96px;
  height: 96px;
  border-radius: 50%;
  overflow: hidden;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--m-gSoft);
  border: 4px solid var(--el-bg-color);
  box-shadow: 0 0 0 3px var(--m-g);
}
.gp-hero__bust {
  flex: none;
  width: 170px;
  height: 170px;
  max-width: none;
  background: transparent;
}
.gp-hero__text {
  flex: 1 1 260px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.gp-hero__name-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.gp-hero__name {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 30px;
  font-weight: 800;
  line-height: 1.15;
  word-break: break-all;
}
.gp-hero__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 14px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.gp-hero__link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-weight: 700;
  color: var(--m-pInk);
  text-decoration: none;
}
.gp-hero__link svg {
  width: 14px;
  height: 14px;
}
.gp-hero__level {
  display: flex;
  align-items: center;
  gap: 10px;
  max-width: 420px;
}
.gp-hero__lv {
  flex: none;
  display: inline-flex;
  align-items: center;
  height: 26px;
  padding: 0 10px;
  border-radius: 999px;
  background: var(--m-gSoft);
  color: var(--m-gInk);
  font-size: 13px;
  font-weight: 800;
}
.gp-hero__bar {
  flex: 1;
  height: 8px;
  border-radius: 999px;
  background: var(--m-s2);
  overflow: hidden;
}
.gp-hero__bar span {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: var(--m-g);
}
.gp-hero__exp {
  flex: none;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.gp-hero__wallet {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}
.gp-hero__cur {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 10px 12px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.gp-hero__cur-img {
  flex: none;
  width: 36px;
  height: 36px;
  object-fit: contain;
  background: transparent;
}
.gp-hero__cur-icon {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 12px;
  background: var(--m-gSoft);
  color: var(--m-gInk);
}
.gp-hero__cur-icon svg {
  width: 20px;
  height: 20px;
}
.gp-hero__cur-text {
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.gp-hero__cur-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.gp-hero__cur-value {
  font-size: 17px;
  font-weight: 800;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.gp-hero__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.gp-hero__actions :deep(button[data-slot='base']) {
  min-height: 40px;
}
/* 出借记忆的卡面：玩家信息右侧的一栏，高度跟随左侧；卡片窄时不显示（头像已经是这张卡） */
.gp-hero__art {
  position: relative;
  flex: 0 0 32%;
  max-width: 320px;
  background: var(--m-gSoft);
}
@container (max-width: 760px) {
  .gp-hero__art {
    display: none;
  }
}
.gp-hero__art-img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-position: 40% 30%;
}
.gp-hero__art-tag {
  position: absolute;
  right: 14px;
  bottom: 14px;
  padding: 4px 10px;
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font-size: 12px;
  font-weight: 700;
}
.gp-menu {
  padding: 6px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gp-menu__item {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 40px;
  padding: 6px 10px;
  border: 0;
  border-radius: 12px;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
}
.gp-menu__item:hover {
  background: var(--el-fill-color-light);
}
.gp-menu__title {
  flex: 1;
  min-width: 0;
}

/* ---------- 分页 ---------- */
.gp-tabs {
  display: flex;
  gap: 24px;
  overflow-x: auto;
  border-bottom: 1px solid var(--el-border-color-light);
  scrollbar-width: none;
}
.gp-tabs::-webkit-scrollbar {
  display: none;
}
.gp-tabs__tab {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 44px;
  padding: 0 2px;
  border: 0;
  border-bottom: 3px solid transparent;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 15px;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.gp-tabs__tab:hover {
  color: var(--el-text-color-primary);
}
.gp-tabs__tab.is-on {
  color: var(--el-text-color-primary);
  border-bottom-color: var(--m-g);
}
.gp-tabs__count {
  padding: 1px 8px;
  border-radius: 999px;
  background: var(--m-s2);
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

@media (max-width: 760px) {
  .gp-hero__main {
    padding: 18px 16px;
  }
  .gp-hero__wallet {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .gp-hero__name {
    font-size: 24px;
  }
  .gp-hero__avatar {
    width: 72px;
    height: 72px;
  }
  .gp-hero__bust {
    width: 128px;
    height: 128px;
  }
  .gp-hero__actions :deep(button[data-slot='base']) {
    min-height: 44px;
  }
  .gp-tabs {
    gap: 18px;
  }
}
</style>
