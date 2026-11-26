<!-- 通行证外壳：顶栏导航（手机上同一排，标签用短名、不带图标；消息在铃铛里）。绑定页与游戏空间在手机上自带头部。 -->
<template>
  <div class="shell">
    <header v-if="!hideChrome" class="shell__top">
      <RouterLink to="/" class="shell__brand">
        <span class="shell__logo pp-sticker">M</span>
        <span class="shell__name">{{ t('nav.brand') }}</span>
      </RouterLink>
      <nav class="shell__nav pp-scroll-x" :aria-label="t('nav.brand')">
        <RouterLink v-for="n in topNav" :key="n.key" :to="n.to" class="shell__tnav" :class="{ 'is-on': active === n.key }">
          <component :is="n.icon" v-if="!ui.isMobile" class="shell__ico" aria-hidden="true" />
          {{ ui.isMobile ? n.short : n.label }}
        </RouterLink>
      </nav>
      <button type="button" class="shell__ib" :aria-label="t('nav.toggleDark')" @click="ui.toggleDark()">
        <component :is="ui.dark ? SunIcon : MoonIcon" aria-hidden="true" />
      </button>
      <RouterLink to="/messages" class="shell__ib" :aria-label="t('nav.messagesUnread', { n: session.unread })">
        <BellIcon aria-hidden="true" />
        <span v-if="session.unread > 0" class="shell__dot" />
      </RouterLink>
      <RouterLink v-if="!ui.isMobile && session.me" to="/account" class="shell__me">
        <PAvatar :src="session.me.avatar" :asset="session.me.avatarAsset" :name="displayName" :size="28" />
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

  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  BellIcon,
  ExclamationTriangleIcon,
  HomeIcon,
  MoonIcon,
  PuzzlePieceIcon,
  ShieldCheckIcon,
  SunIcon,
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
  { key: 'home', to: '/', label: t('nav.home'), short: t('nav.home'), icon: HomeIcon },
  { key: 'games', to: '/games', label: t('nav.games'), short: t('nav.gamesShort'), icon: PuzzlePieceIcon },
  { key: 'account', to: '/account', label: t('nav.account'), short: t('nav.me'), icon: ShieldCheckIcon },
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
  gap: 8px;
  padding: 8px 28px;
  background: var(--m-s);
  border-bottom: 1px solid var(--m-line);
}
.shell__brand {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-right: 12px;
  color: var(--m-ink);
  text-decoration: none;
}
.shell__logo {
  width: 32px;
  height: 32px;
  border-radius: 10px;
  background: var(--m-p);
  color: var(--m-pOn);
  font-size: 16px;
  transform: rotate(-6deg);
}
.shell__name {
  font-size: 16px;
  font-weight: 700;
  white-space: nowrap;
}
.shell__nav {
  display: flex;
  gap: 2px;
  flex: 1;
  min-width: 0;
}
.shell__tnav {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: none;
  min-height: 32px;
  padding: 0 12px;
  border-radius: 999px;
  color: var(--m-ink2);
  text-decoration: none;
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
}
.shell__tnav:hover {
  background: var(--m-s2);
  color: var(--m-ink);
}
.shell__tnav.is-on {
  background: var(--m-pSoft);
  color: var(--m-pInk);
  font-weight: 600;
}
.shell__ico {
  width: 16px;
  height: 16px;
}
.shell__ib {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 36px;
  height: 36px;
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
  width: 19px;
  height: 19px;
}
.shell__dot {
  position: absolute;
  top: 7px;
  right: 8px;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--m-p);
  border: 2px solid var(--m-s);
}
.shell__me {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 36px;
  padding: 4px 12px 4px 4px;
  border-radius: 999px;
  background: var(--m-s2);
  color: var(--m-ink);
  text-decoration: none;
  font-size: 13px;
  font-weight: 500;
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
  gap: 8px;
  padding: 8px 28px;
  background: var(--m-warnSoft);
  color: var(--m-ink);
  font-size: 13px;
}
.shell__warn svg {
  width: 16px;
  height: 16px;
  color: var(--m-warn);
  flex: none;
}
.shell__warn a {
  margin-left: auto;
  font-weight: 600;
  color: var(--m-warn);
  white-space: nowrap;
}
.shell__main {
  flex: 1;
  width: 100%;
  min-width: 0;
  overflow-x: clip;
}
.shell__main {
  padding-bottom: env(safe-area-inset-bottom);
}
@media (max-width: 768px) {
  .shell__top {
    gap: 4px;
    padding: 6px 6px 6px 12px;
    background: var(--m-bg);
  }
  .shell__brand {
    margin-right: 4px;
  }
  .shell__logo {
    width: 30px;
    height: 30px;
    border-radius: 9px;
    font-size: 15px;
  }
  .shell__name {
    display: none;
  }
  .shell__tnav {
    padding: 0 10px;
  }
  .shell__warn {
    padding: 8px 14px;
  }
}
</style>
