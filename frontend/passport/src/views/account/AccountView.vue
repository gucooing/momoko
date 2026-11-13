<!-- 账号与安全：个人资料、登录与安全、登录设备、授权的应用、游戏账号、注销（?tab= 切换）。 -->
<template>
  <div class="acc">
    <nav class="acc__nav" :class="ui.isMobile ? 'pp-scroll-x' : 'pp-card'" :aria-label="t('nav.account')">
      <button
        v-for="s in sections"
        :key="s.key"
        type="button"
        :class="ui.isMobile ? ['pp-chip', { 'is-on': tab === s.key }] : ['acc__sn', { 'is-on': tab === s.key, 'is-danger': s.key === 'delete' }]"
        :aria-current="tab === s.key ? 'page' : undefined"
        @click="go(s.key)"
      >
        <component :is="s.icon" v-if="!ui.isMobile" class="acc__ico" aria-hidden="true" />
        <span class="acc__label">{{ s.label }}</span>
        <span v-if="s.n" class="pp-num pp-hint">{{ s.n }}</span>
      </button>
    </nav>
    <main class="acc__main">
      <KeepAlive>
        <component :is="current" />
      </KeepAlive>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, type Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ComputerDesktopIcon, KeyIcon, LinkIcon, LockClosedIcon, TrashIcon, UserIcon } from '@heroicons/vue/24/outline'
import { useSessionStore } from '@passport/stores/session'
import { useUiStore } from '@passport/stores/ui'
import ProfileSection from './ProfileSection.vue'
import SecuritySection from './SecuritySection.vue'
import DevicesSection from './DevicesSection.vue'
import AppsSection from './AppsSection.vue'
import GamesSection from './GamesSection.vue'
import DeleteSection from './DeleteSection.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const ui = useUiStore()
const session = useSessionStore()

const views: Record<string, Component> = {
  profile: ProfileSection,
  security: SecuritySection,
  devices: DevicesSection,
  apps: AppsSection,
  games: GamesSection,
  delete: DeleteSection,
}

const tab = computed(() => {
  const q = String(route.query.tab || 'profile')
  return q in views ? q : 'profile'
})
const current = computed(() => views[tab.value])

const sections = computed(() => {
  const me = session.me
  return [
    { key: 'profile', label: t('account.nav.profile'), icon: UserIcon, n: 0 },
    { key: 'security', label: t('account.nav.security'), icon: LockClosedIcon, n: 0 },
    { key: 'devices', label: t('account.nav.devices'), icon: ComputerDesktopIcon, n: me?.deviceCount || 0 },
    { key: 'apps', label: t('account.nav.apps'), icon: KeyIcon, n: me?.appCount || 0 },
    { key: 'games', label: t('account.nav.games'), icon: LinkIcon, n: me?.linkCount || 0 },
    { key: 'delete', label: t('account.nav.delete'), icon: TrashIcon, n: 0 },
  ]
})

const go = (key: string) => {
  void router.replace({ query: { ...route.query, tab: key } })
}
</script>

<style scoped>
.acc {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
  padding: 28px 32px 56px;
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  align-items: flex-start;
}
.acc__nav.pp-card {
  flex: 0 0 240px;
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  position: sticky;
  top: 88px;
}
.acc__sn {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-height: 46px;
  padding: 0 14px;
  border: 0;
  border-radius: 14px;
  background: transparent;
  color: var(--m-ink2);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
}
.acc__sn:hover {
  background: var(--m-s2);
  color: var(--m-ink);
}
.acc__sn.is-on {
  background: var(--m-pSoft);
  color: var(--m-pInk);
  font-weight: 800;
}
.acc__sn.is-danger {
  color: var(--m-bad);
}
.acc__ico {
  width: 20px;
  height: 20px;
  flex: none;
}
.acc__label {
  flex: 1;
}
.acc__main {
  flex: 1 1 600px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 18px;
}
@media (max-width: 768px) {
  /* 单列时不能换行：多行 flex 的行宽按最宽的子项（横向滚动的 chips）计算，会把整页撑出屏幕 */
  .acc {
    padding: 12px 16px 24px;
    gap: 14px;
    flex-direction: column;
    flex-wrap: nowrap;
    align-items: stretch;
  }
  .acc__nav {
    min-width: 0;
    margin: 0 -16px;
    padding: 0 16px;
  }
  .acc__main {
    flex-basis: auto;
  }
}
</style>
