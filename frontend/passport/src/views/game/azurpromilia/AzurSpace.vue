<!-- 蓝色星原 的游戏空间：?tab=overview|items|currencies|heroes|pets|mails|logins|self 切换页内标签。
     概览是资料与各项数量；道具、货币、角色、奇波、邮箱、登录记录取自游戏服务端已保存的数据；自助服务是管理员开放的操作。 -->
<template>
  <div class="az">
    <nav class="az__tabs pp-scroll-x" :aria-label="t('azur.tabs.overview')">
      <button
        v-for="k in TABS"
        :key="k"
        type="button"
        class="az__tab"
        :class="{ 'is-on': tab === k }"
        :aria-current="tab === k ? 'page' : undefined"
        @click="setTab(k)"
      >
        {{ t(`azur.tabs.${k}`) }}
      </button>
    </nav>

    <template v-if="tab === 'overview'">
      <section class="pp-card az__card">
        <div class="az__head">
          <span class="az__av">{{ (player.name || player.account || '?').slice(0, 1) }}</span>
          <span class="az__head-text">
            <span class="az__name">{{ player.name || t('game.noName') }}</span>
            <span class="pp-hint pp-num">
              {{ player.displayId ? t('azur.displayId', { id: player.displayId }) : t('azur.noCharacter') }}
            </span>
          </span>
          <span class="pp-tag" :class="{ 'pp-tag--g': player.online }">
            {{ player.online ? t('azur.online') : t('azur.offline') }}
          </span>
        </div>
        <dl class="az__facts">
          <div v-for="f in facts" :key="f.key" class="az__fact">
            <dt class="pp-hint">{{ f.label }}</dt>
            <dd class="pp-num">{{ f.value }}</dd>
          </div>
        </dl>
        <p v-if="overview?.sign" class="az__sign">{{ overview.sign }}</p>
      </section>
      <section v-if="overview" class="az__stats">
        <div v-for="s in stats" :key="s.key" class="pp-card az__stat">
          <span class="pp-hint">{{ s.label }}</span>
          <span class="pp-num az__stat-v">{{ fmtNum(s.value) }}</span>
        </div>
      </section>
      <p v-if="overview" class="pp-hint">{{ t('azur.savedHint') }}</p>
    </template>
    <AzurSelf v-else-if="tab === 'self'" :server-id="serverId" :server-name="serverName" :nickname="player.name" />
    <AzurList v-else :key="`${serverId}:${tab}`" :server-id="serverId" :kind="tab" :has-character="!!player.displayId" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { fmtNum, fmtTime } from '@passport/utils/format'
import type { PlayerDetail } from '@/types/v1/azurpromilia/types'
import AzurList from './AzurList.vue'
import AzurSelf from './AzurSelf.vue'

const TABS = ['overview', 'items', 'currencies', 'heroes', 'pets', 'mails', 'logins', 'self'] as const
type Tab = (typeof TABS)[number]

const props = defineProps<{ serverId: string; serverName: string; detail: PlayerDetail }>()
const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const tab = computed<Tab>(() => ((TABS as readonly string[]).includes(String(route.query.tab)) ? (route.query.tab as Tab) : 'overview'))
const setTab = (k: Tab) => void router.replace({ query: { tab: k } })

const player = computed(() => props.detail.player!)
const overview = computed(() => props.detail.overview)

const facts = computed(() => {
  const p = player.value
  const o = overview.value
  const out = [
    { key: 'level', label: t('azur.level'), value: o ? String(p.level) : '—' },
    { key: 'exp', label: t('azur.exp'), value: o ? fmtNum(o.exp) : '—' },
    { key: 'account', label: t('azur.account'), value: p.account || '—' },
    { key: 'created', label: t('azur.accountCreated'), value: fmtTime(p.createdAt) },
    { key: 'register', label: t('azur.characterCreated'), value: o ? fmtTime(o.registerAt) : '—' },
    { key: 'login', label: t('azur.lastLogin'), value: fmtTime(p.lastLoginAt) },
    { key: 'logout', label: t('azur.lastLogout'), value: o ? fmtTime(o.logoutAt) : '—' },
  ]
  if (o && o.birthdayMonth && o.birthdayDay) {
    out.push({ key: 'birthday', label: t('azur.birthday'), value: t('azur.birthdayValue', { m: o.birthdayMonth, d: o.birthdayDay }) })
  }
  return out
})

const stats = computed(() => {
  const o = overview.value
  if (!o) return []
  return [
    { key: 'heroes', label: t('azur.heroes'), value: o.heroes },
    { key: 'pets', label: t('azur.pets'), value: o.pets },
    { key: 'achievements', label: t('azur.achievements'), value: o.achievements },
    { key: 'logins', label: t('azur.loginCount'), value: o.logins },
  ]
})
</script>

<style scoped>
.az {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.az__tabs {
  display: flex;
  gap: 4px;
  border-bottom: 1px solid var(--m-line);
}
.az__tab {
  padding: 8px 12px;
  border: 0;
  border-bottom: 2px solid transparent;
  background: transparent;
  color: var(--m-ink2);
  font: inherit;
  white-space: nowrap;
  cursor: pointer;
}
.az__tab.is-on {
  border-bottom-color: var(--m-p);
  color: var(--m-ink);
  font-weight: 600;
}
.az__card {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.az__head {
  display: flex;
  align-items: center;
  gap: 12px;
}
.az__av {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--m-pSoft);
  color: var(--m-pInk);
  font-family: var(--app-font-display);
  font-size: 20px;
  font-weight: 600;
  flex: none;
}
.az__head-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.az__name {
  font-family: var(--app-font-display);
  font-size: 18px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.az__facts {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
  padding: 14px;
  border-radius: var(--app-radius-lg, 16px);
  background: var(--m-s2);
}
.az__fact {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.az__fact dd {
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.az__sign {
  margin: 0;
  color: var(--m-ink2);
  white-space: pre-wrap;
  word-break: break-word;
}
.az__stats {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 12px;
}
.az__stat {
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.az__stat-v {
  font-size: 22px;
  font-weight: 600;
}
</style>
