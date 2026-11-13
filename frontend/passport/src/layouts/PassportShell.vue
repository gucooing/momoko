<!-- 通行证外壳：电脑上顶栏导航，手机上顶栏 + 底部导航（绑定页与游戏空间在手机上自带头部与导航）。 -->
<template>
  <div class="shell">
    <header v-if="!hideChrome" class="shell__top">
      <RouterLink to="/" class="shell__brand">
        <span class="shell__logo pp-sticker">M</span>
        <span class="shell__name">{{ t('nav.brand') }}</span>
      </RouterLink>
      <nav v-if="!ui.isMobile" class="shell__nav" :aria-label="t('nav.brand')">
        <RouterLink v-for="n in topNav" :key="n.key" :to="n.to" class="shell__tnav" :class="{ 'is-on': active === n.key }">
          <component :is="n.icon" class="shell__ico" aria-hidden="true" />
          {{ n.label }}
        </RouterLink>
      </nav>
      <span v-else class="shell__spacer" />
      <button type="button" class="shell__ib" :aria-label="t('nav.toggleDark')" @click="ui.toggleDark()">
        <component :is="ui.dark ? SunIcon : MoonIcon" aria-hidden="true" />
      </button>
      <RouterLink to="/messages" class="shell__ib" :aria-label="t('nav.messagesUnread', { n: session.unread })">
        <BellIcon aria-hidden="true" />
        <span v-if="session.unread > 0" class="shell__dot" />
      </RouterLink>
      <RouterLink v-if="!ui.isMobile && session.me" to="/account" class="shell__me">
        <PAvatar :src="session.me.avatar" :asset="session.me.avatarAsset" :name="displayName" :size="36" />
        <span class="shell__me-name">{{ displayName }}</span>
      </RouterLink>
    </header>

    <div v-if="session.me?.deletionScheduledAt && !hideChrome" class="shell__warn" role="alert">
      <ExclamationTriangleIcon aria-hidden="true" />
      <span>{{ t('nav.deletionBanner', { date: fmtDate(session.me.deletionScheduledAt) }) }}</span>
      <RouterLink :to="{ path: '/account', query: { tab: 'delete' } }">{{ t('nav.deletionCancel') }}</RouterLink>
    </div>

    <main class="shell__main" :class="{ 'shell__main--bare': hideChrome }">
      <RouterView />
    </main>

    <nav v-if="ui.isMobile && !hideChrome" class="shell__bottom" :aria-label="t('nav.brand')">
      <RouterLink v-for="n in bottomNav" :key="n.key" :to="n.to" class="shell__bn" :class="{ 'is-on': active === n.key }">
        <component :is="n.icon" aria-hidden="true" />
        {{ n.label }}
        <span v-if="n.key === 'messages' && session.unread > 0" class="shell__bn-dot" />
      </RouterLink>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  BellIcon,
  EnvelopeIcon,
  ExclamationTriangleIcon,
  HomeIcon,
  MoonIcon,
  PuzzlePieceIcon,
  ShieldCheckIcon,
  SunIcon,
  UserIcon,
} from '@heroicons/vue/24/outline'
import PAvatar from '@passport/components/PAvatar.vue'
import { useSessionStore } from '@passport/stores/session'
import { useUiStore } from '@passport/stores/ui'
import { fmtDate } from '@passport/utils/format'

const { t } = useI18n()
const route = useRoute()
const ui = useUiStore()
const session = useSessionStore()

const active = computed(() => String(route.meta.nav || ''))
const hideChrome = computed(() => ui.isMobile && !!route.meta.ownMobileChrome)
const displayName = computed(() => session.me?.name || session.me?.username || '')

const topNav = computed(() => [
  { key: 'home', to: '/', label: t('nav.home'), icon: HomeIcon },
  { key: 'games', to: '/games', label: t('nav.games'), icon: PuzzlePieceIcon },
  { key: 'account', to: '/account', label: t('nav.account'), icon: ShieldCheckIcon },
])
const bottomNav = computed(() => [
  { key: 'home', to: '/', label: t('nav.home'), icon: HomeIcon },
  { key: 'games', to: '/games', label: t('nav.gamesShort'), icon: PuzzlePieceIcon },
  { key: 'messages', to: '/messages', label: t('nav.messages'), icon: EnvelopeIcon },
  { key: 'account', to: '/account', label: t('nav.me'), icon: UserIcon },
])
</script>

<style scoped>
.shell {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--m-bg);
  color: var(--m-ink);
}
.shell__top {
  position: sticky;
  top: 0;
  z-index: 50;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 32px;
  background: var(--m-s);
  border-bottom: 1px solid var(--m-line);
}
.shell__brand {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-right: 16px;
  color: var(--m-ink);
  text-decoration: none;
}
.shell__logo {
  width: 38px;
  height: 38px;
  border-radius: 12px;
  background: var(--m-p);
  color: var(--m-pOn);
  font-size: 19px;
  transform: rotate(-6deg);
}
.shell__name {
  font-size: 18px;
  font-weight: 800;
  white-space: nowrap;
}
.shell__nav {
  display: flex;
  gap: 4px;
  flex: 1;
}
.shell__spacer {
  flex: 1;
}
.shell__tnav {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  color: var(--m-ink2);
  text-decoration: none;
  font-size: 14px;
  font-weight: 600;
}
.shell__tnav:hover {
  background: var(--m-s2);
  color: var(--m-ink);
}
.shell__tnav.is-on {
  background: var(--m-pSoft);
  color: var(--m-pInk);
  font-weight: 800;
}
.shell__ico {
  width: 18px;
  height: 18px;
}
.shell__ib {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--m-ink2);
  cursor: pointer;
}
.shell__ib:hover {
  background: var(--m-s2);
  color: var(--m-ink);
}
.shell__ib svg {
  width: 22px;
  height: 22px;
}
.shell__dot {
  position: absolute;
  top: 10px;
  right: 11px;
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: var(--m-p);
  border: 2px solid var(--m-s);
}
.shell__me {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  padding: 4px 14px 4px 4px;
  border-radius: 999px;
  background: var(--m-s2);
  color: var(--m-ink);
  text-decoration: none;
  font-size: 14px;
  font-weight: 700;
}
.shell__me-name {
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.shell__warn {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 32px;
  background: var(--m-warnSoft);
  color: var(--m-ink);
  font-size: 13px;
}
.shell__warn svg {
  width: 18px;
  height: 18px;
  color: var(--m-warn);
  flex: none;
}
.shell__warn a {
  margin-left: auto;
  font-weight: 800;
  color: var(--m-warn);
  white-space: nowrap;
}
.shell__main {
  flex: 1;
  width: 100%;
  min-width: 0;
  overflow-x: clip;
}
.shell__bottom {
  position: sticky;
  bottom: 0;
  z-index: 50;
  display: flex;
  gap: 4px;
  padding: 6px 8px calc(10px + env(safe-area-inset-bottom));
  background: var(--m-s);
  border-top: 1px solid var(--m-line);
}
.shell__bn {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
  flex: 1;
  min-height: 56px;
  border-radius: 16px;
  color: var(--m-ink2);
  text-decoration: none;
  font-size: 11px;
  font-weight: 700;
}
.shell__bn svg {
  width: 22px;
  height: 22px;
}
.shell__bn.is-on {
  color: var(--m-pInk);
}
.shell__bn-dot {
  position: absolute;
  top: 8px;
  left: calc(50% + 8px);
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--m-p);
}
@media (max-width: 768px) {
  .shell__top {
    padding: 8px 10px 8px 14px;
    background: var(--m-bg);
  }
  .shell__logo {
    width: 32px;
    height: 32px;
    border-radius: 10px;
    font-size: 16px;
  }
  .shell__name {
    font-size: 17px;
  }
  .shell__warn {
    padding: 10px 16px;
  }
}
</style>
