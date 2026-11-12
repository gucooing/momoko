<!-- 用户通行证后台（权限 game:passport）：统计、通行证列表（状态 chips + 服务器搜索下拉 + 搜索）、右侧详情、
     通行证设置（含 OIDC 应用）与邀请码。通行证账号是独立的玩家账号，与 Momoko 用户无关。 -->
<template>
  <div class="ppa">
    <PageHeader :title="t('passportAdmin.title')">
      <template #description>{{ t('passportAdmin.count', { n: counts.all ?? 0 }) }} · {{ t('passportAdmin.description') }}</template>
      <template #actions>
        <UButton color="neutral" variant="outline" icon="i-lucide-settings" @click="settingsOpen = true">{{ t('passportAdmin.actions.settings') }}</UButton>
        <UButton icon="i-lucide-mail-plus" @click="inviteOpen = true">{{ t('passportAdmin.actions.invite') }}</UButton>
      </template>
    </PageHeader>

    <div class="ppa__stats">
      <div v-for="s in statCards" :key="s.key" class="ppa__stat">
        <span class="m-sticker ppa__stat-ico" :class="`m-sticker--${s.tone}`">
          <component :is="menuStore.iconComponents[s.icon]" />
        </span>
        <span class="ppa__stat-text">
          <span class="ppa__stat-label">{{ s.label }}</span>
          <span class="ppa__stat-value m-num">{{ s.value }}</span>
          <span class="ppa__stat-sub">{{ s.sub }}</span>
        </span>
      </div>
    </div>

    <div class="ppa__main">
      <section class="ppa__list">
        <ListToolbar
          v-model:status="status"
          v-model:keyword="keyword"
          :chips="chips"
          :chips-label="t('passportAdmin.chipsLabel')"
          :placeholder="t('passportAdmin.filter.search')"
          class="ppa__toolbar"
          @search="onFilter"
        >
          <GameServerPicker v-model="serverId" :selected="server" :search="searchServers" allow-all @select="onServer">
            <template #trigger="{ open, current }">
              <button type="button" class="ppa__server" :class="{ 'is-on': !!serverId }" :aria-expanded="open">
                <component :is="menuStore.iconComponents['HOutline:ServerStackIcon']" class="ppa__server-icon" />
                <span class="ppa__server-text">{{ current?.name ? t('passportAdmin.filter.server', { name: current.name }) : t('passportAdmin.filter.serverAll') }}</span>
                <component :is="menuStore.iconComponents['HOutline:ChevronUpDownIcon']" class="ppa__server-icon" />
              </button>
            </template>
          </GameServerPicker>
        </ListToolbar>

        <div v-if="!isMobile && items.length" class="ppa__row ppa__row--head" aria-hidden="true">
          <span>{{ t('passportAdmin.table.passport') }}</span>
          <span>{{ t('passportAdmin.table.links') }}</span>
          <span>{{ t('passportAdmin.table.lastLogin') }}</span>
          <span>{{ t('passportAdmin.table.status') }}</span>
        </div>

        <div v-if="loading && !items.length" class="ppa__loading">{{ t('passportAdmin.common.loading') }}</div>
        <EmptyState
          v-else-if="!items.length"
          icon="HOutline:IdentificationIcon"
          :title="filtered ? t('passportAdmin.empty.filtered') : t('passportAdmin.empty.title')"
          :description="filtered ? t('passportAdmin.empty.filteredDesc') : t('passportAdmin.empty.desc')"
        />
        <div v-else class="ppa__rows" :class="{ 'is-loading': loading }">
          <button
            v-for="u in items"
            :key="u.id"
            type="button"
            class="ppa__row"
            :class="{ 'is-on': u.id === selectedId, 'is-card': isMobile }"
            :aria-pressed="u.id === selectedId"
            @click="select(u.id)"
          >
            <span class="ppa__who">
              <PassportAvatar :avatar="u.avatar" :avatar-asset="u.avatarAsset" :name="u.name || u.username" :size="40" />
              <span class="ppa__who-text">
                <span class="ppa__name">{{ u.name || u.username }}</span>
                <span class="ppa__sub">@{{ u.username }}<template v-if="u.emailMasked"> · {{ u.emailMasked }}</template></span>
              </span>
              <StatusPill v-if="isMobile" :variant="statusOf(u).variant" :label="statusOf(u).label" />
            </span>
            <span class="ppa__links">
              <span v-if="u.linkCount === 1 && u.firstLink" class="ppa__one">
                <GameBadge :game="u.firstLink.game" :name="u.firstLink.gameName" :size="24" />
                <span class="ppa__one-text">{{ [u.firstLink.serverName, u.firstLink.nickname].filter(Boolean).join(' · ') }}</span>
              </span>
              <span v-else-if="u.linkCount > 1" class="ppa__many">
                <span class="ppa__stack">
                  <GameBadge v-for="g in u.games" :key="g.id" :game="g.id" :name="g.name" :size="26" />
                </span>
                <span class="ppa__many-text">{{ linkCountText(u) }}</span>
              </span>
              <span v-else class="ppa__sub">{{ t('passportAdmin.links.none') }}</span>
            </span>
            <span class="ppa__sub m-num ppa__last">{{ u.lastLoginAt ? ago(u.lastLoginAt) : t('passportAdmin.never') }}</span>
            <span v-if="!isMobile" class="ppa__status"><StatusPill :variant="statusOf(u).variant" :label="statusOf(u).label" /></span>
          </button>
        </div>

        <Pagination v-if="total > pageSize" v-model:page="page" v-model:page-size="pageSize" :total="total" class="ppa__pager" @change="load" />
      </section>

      <aside ref="sideRef" class="ppa__side">
        <UserDetail v-if="selectedId" :key="selectedId" :user-id="selectedId" @changed="onUserChanged" />
        <section v-else class="ppa__pick">
          <EmptyState icon="HOutline:UserCircleIcon" :title="t('passportAdmin.detail.pick')" :description="t('passportAdmin.detail.pickDesc')" />
        </section>

        <section class="ppa__summary">
          <div class="ppa__summary-head">
            <h2 class="ppa__summary-title">{{ t('passportAdmin.summary.title') }}</h2>
            <AppIconButton icon="HOutline:PencilSquareIcon" :label="t('passportAdmin.actions.settings')" :box="40" :size="18" @click="settingsOpen = true" />
          </div>
          <div v-for="row in summaryRows" :key="row.key" class="ppa__summary-row">
            <span class="ppa__summary-key">{{ row.label }}</span>
            <span class="ppa__summary-value">{{ row.value }}</span>
          </div>
        </section>
      </aside>
    </div>

    <SettingsDialog v-model="settingsOpen" @saved="onSettingsSaved" />
    <InviteDialog v-model="inviteOpen" :register-mode="settings?.settings?.registerMode" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import GameServerPicker from '@/views/game/components/GameServerPicker.vue'
import { getPassportSettings, getPassportStats, listPassportUsers, searchPassportServers } from '@/api/passportAdmin'
import type { GameServerBrief } from '@/types/v1/game_common'
import type { GetPassportSettingsResponse, GetPassportStatsResponse, PassportAdminUser } from '@/types/v1/passport_admin'
import PassportAvatar from './components/PassportAvatar.vue'
import GameBadge from './components/GameBadge.vue'
import UserDetail from './components/UserDetail.vue'
import SettingsDialog from './components/SettingsDialog.vue'
import InviteDialog from './components/InviteDialog.vue'
import { notifyError, usePassportTime } from './shared'
import { usePassportStatus } from './status'

defineOptions({ name: 'GamePassport' })

const STATUSES = ['all', 'linked', 'unlinked', 'disabled', 'deleting'] as const
type StatusKey = (typeof STATUSES)[number]

const { t } = useI18n()
const menuStore = useMenuStore()
const isMobile = computed(() => menuStore.isMobile)
const { ago } = usePassportTime()
const statusOf = usePassportStatus()

const stats = ref<GetPassportStatsResponse>()
const settings = ref<GetPassportSettingsResponse>()

const status = ref<StatusKey>('all')
const keyword = ref('')
const serverId = ref<string>()
const server = ref<GameServerBrief>()
const page = ref(1)
const pageSize = ref(20)
const items = ref<PassportAdminUser[]>([])
const total = ref(0)
const counts = ref<Record<string, number>>({})
const loading = ref(false)
const selectedId = ref('')
const sideRef = ref<HTMLElement>()
const settingsOpen = ref(false)
const inviteOpen = ref(false)
let seq = 0

const filtered = computed(() => status.value !== 'all' || !!keyword.value.trim() || !!serverId.value)

const chips = computed(() =>
  STATUSES.map((s) => ({ value: s, label: t(`passportAdmin.chips.${s}`), count: counts.value[s] ?? null })),
)

const statCards = computed(() => {
  const s = stats.value
  return [
    {
      key: 'players',
      icon: 'HOutline:IdentificationIcon',
      tone: 'p',
      label: t('passportAdmin.stats.players'),
      value: s?.playerCount ?? '—',
      sub: s ? t('passportAdmin.stats.playersSub', { n: s.activeThisWeek }) : '',
    },
    {
      key: 'linked',
      icon: 'HOutline:LinkIcon',
      tone: 'a2',
      label: t('passportAdmin.stats.linked'),
      value: s?.linkedAccountCount ?? '—',
      sub: s ? t('passportAdmin.stats.linkedSub', { servers: s.serverCount, games: s.gameCount }) : '',
    },
    {
      key: 'new',
      icon: 'HOutline:SparklesIcon',
      tone: 'ok',
      label: t('passportAdmin.stats.newWeek'),
      value: s?.newThisWeek ?? '—',
      sub: s ? t('passportAdmin.stats.newWeekSub', { n: s.newThisWeekLinked }) : '',
    },
    {
      key: 'accounts',
      icon: 'HOutline:ShieldCheckIcon',
      tone: s && (s.disabledCount > 0 || s.deletingCount > 0) ? 'warn' : 'a',
      label: t('passportAdmin.stats.accounts'),
      value: s ? t('passportAdmin.stats.accountsValue', { disabled: s.disabledCount, deleting: s.deletingCount }) : '—',
      sub: s ? t('passportAdmin.stats.accountsSub', { disabled: s.disabledCount, deleting: s.deletingCount }) : '',
    },
  ]
})

const onOff = (v?: boolean) => (v ? t('passportAdmin.summary.on') : t('passportAdmin.summary.off'))

const summaryRows = computed(() => {
  const info = settings.value
  const s = info?.settings
  if (!info || !s) return []
  const emailNote = info.emailConfigured ? '' : ` · ${t('passportAdmin.summary.emailMissing')}`
  return [
    {
      key: 'address',
      label: t('passportAdmin.summary.address'),
      value: !info.serverEnabled
        ? t('passportAdmin.summary.serverOff')
        : info.publicUrl
          ? t('passportAdmin.summary.addressPublic', { url: info.publicUrl, addr: info.listenAddr })
          : t('passportAdmin.summary.addressValue', { addr: info.listenAddr }),
    },
    { key: 'register', label: t('passportAdmin.summary.register'), value: t(`passportAdmin.settings.modes.${s.registerMode}`) },
    { key: 'emailVerify', label: t('passportAdmin.summary.emailVerify'), value: onOff(s.registerEmailVerification) + emailNote },
    { key: 'emailLogin', label: t('passportAdmin.summary.emailLogin'), value: onOff(s.emailLoginEnabled) + emailNote },
    {
      key: 'oidc',
      label: t('passportAdmin.summary.oidc'),
      value: s.oidcEnabled ? t('passportAdmin.summary.oidcOn', { n: info.oidcClientCount }) : t('passportAdmin.summary.oidcOff'),
    },
    { key: 'session', label: t('passportAdmin.summary.session'), value: t('passportAdmin.summary.sessionDays', { n: s.sessionDays }) },
  ]
})

const linkCountText = (u: PassportAdminUser) =>
  u.gameCount > 1
    ? t('passportAdmin.links.serversGames', { n: u.linkCount, games: u.gameCount })
    : t('passportAdmin.links.servers', { n: u.linkCount })

const searchServers = async (q: string) => (await searchPassportServers({ q, limit: 20 })).data

const load = async () => {
  const id = ++seq
  loading.value = true
  try {
    const res = await listPassportUsers({
      status: status.value === 'all' ? '' : status.value,
      q: keyword.value.trim(),
      serverId: serverId.value || '',
      page: page.value,
      pageSize: pageSize.value,
    })
    if (id !== seq) return
    items.value = res.data.items || []
    total.value = res.data.total || 0
    counts.value = res.data.counts || {}
    if (res.data.server && serverId.value === res.data.server.id) server.value = res.data.server
    if (!selectedId.value && !isMobile.value && items.value[0]) selectedId.value = items.value[0].id
  } catch (e) {
    if (id === seq) notifyError(e)
  } finally {
    if (id === seq) loading.value = false
  }
}

const loadStats = async () => {
  try {
    stats.value = (await getPassportStats()).data
  } catch (e) {
    notifyError(e)
  }
}

const loadSettings = async () => {
  try {
    settings.value = (await getPassportSettings()).data
  } catch (e) {
    notifyError(e)
  }
}

const onFilter = () => {
  page.value = 1
  void load()
}

const onServer = (s: GameServerBrief | undefined) => {
  server.value = s
  onFilter()
}

const select = async (id: string) => {
  selectedId.value = id
  if (isMobile.value) {
    await nextTick()
    sideRef.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
}

const onUserChanged = (u: PassportAdminUser) => {
  const i = items.value.findIndex((x) => x.id === u.id)
  if (i >= 0) items.value[i] = u
  void loadStats()
  void load()
}

const onSettingsSaved = (info: GetPassportSettingsResponse) => {
  settings.value = info
}

onMounted(() => {
  void load()
  void loadStats()
  void loadSettings()
})
</script>

<style scoped>
.ppa {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.ppa__stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}
.ppa__stat {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
  padding: 16px 18px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}
.ppa__stat-ico {
  width: 44px;
  height: 44px;
  transform: rotate(-4deg);
}
.ppa__stat-ico :deep(svg) {
  width: 22px;
  height: 22px;
}
.ppa__stat-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.ppa__stat-label {
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.ppa__stat-value {
  font-family: var(--app-font-display);
  font-size: 24px;
  font-weight: 800;
  line-height: 1.25;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ppa__stat-sub {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ppa__main {
  display: flex;
  flex-wrap: wrap;
  gap: 18px;
  align-items: flex-start;
}
.ppa__list {
  flex: 3 1 600px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  padding: 14px 0 10px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}
.ppa__toolbar {
  padding: 0 16px 12px;
}
.ppa__server {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  max-width: 280px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 700;
  cursor: pointer;
}
.ppa__server.is-on {
  border-color: var(--m-p);
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.ppa__server-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ppa__server-icon {
  flex: none;
  width: 16px;
  height: 16px;
}
.ppa__rows {
  display: flex;
  flex-direction: column;
  transition: opacity 0.15s;
}
.ppa__rows.is-loading {
  opacity: 0.6;
}
.ppa__row {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(0, 1.6fr) 110px 96px;
  gap: 14px;
  align-items: center;
  width: 100%;
  min-height: 66px;
  padding: 8px 16px;
  border: 0;
  border-top: 1px solid var(--el-border-color-lighter);
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.ppa__row:hover {
  background: var(--el-fill-color-light);
}
.ppa__row.is-on,
.ppa__row.is-on:hover {
  background: var(--m-pSoft);
}
.ppa__row--head {
  min-height: 36px;
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  cursor: default;
}
.ppa__row--head:hover {
  background: transparent;
}
.ppa__row.is-card {
  grid-template-columns: 1fr;
  gap: 8px;
  padding: 12px 16px;
}
.ppa__who {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.ppa__who-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.ppa__name {
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ppa__sub {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ppa__links {
  min-width: 0;
  display: flex;
}
.ppa__one {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  min-height: 32px;
  padding: 0 12px 0 4px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  font-size: 12px;
  font-weight: 700;
}
.ppa__one-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ppa__many {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.ppa__stack {
  display: inline-flex;
  align-items: center;
}
.ppa__stack > * + * {
  margin-left: -8px;
}
.ppa__many-text {
  font-size: 13px;
  font-weight: 700;
  white-space: nowrap;
}
.ppa__row.is-card .ppa__last {
  padding-left: 50px;
}
.ppa__row.is-card .ppa__links {
  padding-left: 50px;
}
.ppa__status {
  justify-self: start;
}
.ppa__loading {
  padding: 40px 16px;
  text-align: center;
  color: var(--el-text-color-secondary);
}
.ppa__pager {
  padding: 12px 16px 2px;
}
.ppa__side {
  flex: 2 1 360px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
  scroll-margin-top: 12px;
}
.ppa__pick,
.ppa__summary {
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}
.ppa__summary {
  display: flex;
  flex-direction: column;
  padding: 14px 20px 10px;
}
.ppa__summary-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}
.ppa__summary-title {
  margin: 0;
  font-size: 16px;
  font-weight: 800;
}
.ppa__summary-row {
  display: flex;
  gap: 12px;
  padding: 8px 0;
  border-top: 1px solid var(--el-border-color-lighter);
  font-size: 13px;
}
.ppa__summary-key {
  flex: none;
  width: 110px;
  font-weight: 700;
}
.ppa__summary-value {
  flex: 1;
  min-width: 0;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
  word-break: break-word;
}
@media (width < 1100px) {
  .ppa__stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (width < 640px) {
  .ppa__stats {
    grid-template-columns: 1fr;
  }
  .ppa__row {
    min-height: 44px;
  }
  .ppa__server {
    max-width: 100%;
  }
}
</style>

