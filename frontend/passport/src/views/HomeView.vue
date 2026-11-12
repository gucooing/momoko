<!-- 通行证首页：先账号（资料、安全提醒），再「我的游戏」（按游戏分组），最后消息与最近活动。 -->
<template>
  <div class="home">
    <section v-if="me" class="pp-card profile">
      <div class="profile__main">
        <div class="profile__id">
          <PAvatar :src="me.avatar" :asset="me.avatarAsset" :name="name" :size="ui.isMobile ? 68 : 96" shape="rounded" tilt />
          <div class="profile__text">
            <h1 class="pp-title profile__name">{{ name }}</h1>
            <span class="profile__sub">{{ subline }}</span>
          </div>
          <div v-if="!ui.isMobile" class="profile__actions">
            <PButton size="sm" :icon="PencilSquareIcon" :to="{ path: '/account', query: { tab: 'profile' } }">{{ t('home.editProfile') }}</PButton>
            <PButton size="sm" variant="soft" :icon="ShieldCheckIcon" :to="{ path: '/account', query: { tab: 'security' } }">
              {{ t('nav.account') }}
            </PButton>
          </div>
        </div>
        <dl class="facts">
          <div v-for="f in facts" :key="f.k" class="facts__item">
            <dt class="pp-hint">{{ f.k }}</dt>
            <dd>{{ f.v }}</dd>
          </div>
        </dl>
      </div>
      <div v-if="!ui.isMobile" class="security">
        <div class="security__head">
          <span class="security__title">{{ t('home.security.title') }}</span>
          <PPill :tone="securityOk ? 'ok' : 'warn'">{{ securityOk ? t('home.security.good') : t('home.security.attention') }}</PPill>
        </div>
        <div v-for="s in security" :key="s.key" class="security__row">
          <span class="security__ico" :class="s.ok ? 'is-ok' : 'is-warn'">
            <component :is="s.ok ? CheckIcon : ExclamationTriangleIcon" aria-hidden="true" />
          </span>
          <span class="security__text">{{ s.text }}</span>
          <RouterLink v-if="s.act" :to="s.to" class="security__act">{{ s.act }}</RouterLink>
        </div>
      </div>
      <RouterLink v-else-if="warning" :to="warning.to" class="mwarn">
        <ExclamationTriangleIcon aria-hidden="true" />
        <span>{{ warning.text }}</span>
        <b>{{ warning.act }}</b>
      </RouterLink>
    </section>

    <section class="block">
      <div class="block__head">
        <h2 class="pp-title block__title">{{ t('nav.games') }}</h2>
        <span v-if="links.length && !ui.isMobile" class="pp-hint">{{ t('home.games.summary', { n: links.length, g: gameCount }) }}</span>
        <PButton size="sm" :variant="ui.isMobile ? 'soft' : 'primary'" :icon="PlusIcon" to="/bind">
          {{ ui.isMobile ? t('home.games.bindShort') : t('home.games.bind') }}
        </PButton>
      </div>
      <p v-if="linksError" class="pp-error">{{ linksError }}</p>
      <MyGames v-else-if="links.length || linksLoaded" :links="links" :compact="ui.isMobile" :bindable="bindable" />
      <p v-else class="pp-hint">{{ t('common.loading') }}</p>
    </section>

    <div class="cols">
      <section class="pp-card panel">
        <div class="panel__head">
          <h2 class="pp-title panel__title">{{ t('nav.messages') }}</h2>
          <PButton v-if="unreadCount > 0" size="sm" variant="ghost" :loading="reading" @click="readAll">{{ t('home.readAll') }}</PButton>
          <RouterLink to="/messages" class="pp-link">{{ t('common.viewAll') }}</RouterLink>
        </div>
        <PEmpty v-if="messagesLoaded && !messages.length" compact :icon="EnvelopeIcon" :title="t('home.noMessages')" />
        <div class="pp-rows">
          <MessageRow v-for="m in messages" :key="m.id" :message="m" />
        </div>
      </section>
      <section class="pp-card panel">
        <div class="panel__head">
          <h2 class="pp-title panel__title">{{ t('home.activity') }}</h2>
        </div>
        <PEmpty v-if="activityLoaded && !activity.length" compact :icon="ClockIcon" :title="t('home.noActivity')" />
        <div class="pp-rows">
          <div v-for="(a, i) in activity" :key="i" class="act">
            <span class="act__at pp-num pp-hint">{{ fmtTime(a.time) }}</span>
            <span class="pp-tag act__where">{{ activityWhere(a) }}</span>
            <span class="act__text">{{ activityText(a) }}</span>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  CheckIcon,
  ClockIcon,
  EnvelopeIcon,
  ExclamationTriangleIcon,
  PencilSquareIcon,
  PlusIcon,
  ShieldCheckIcon,
} from '@heroicons/vue/24/outline'
import MessageRow from '@passport/components/MessageRow.vue'
import MyGames from '@passport/components/MyGames.vue'
import PAvatar from '@passport/components/PAvatar.vue'
import PButton from '@passport/components/PButton.vue'
import PEmpty from '@passport/components/PEmpty.vue'
import PPill from '@passport/components/PPill.vue'
import * as api from '@passport/api/passport'
import { useSessionStore } from '@passport/stores/session'
import { useUiStore } from '@passport/stores/ui'
import { errorText, fmtAgo, fmtDate, fmtTime, toDate } from '@passport/utils/format'
import { activityText, activityWhere } from '@passport/utils/messages'
import type { PassportActivity, PassportLink, PassportMessage } from '@/types/v1/passport'

const { t } = useI18n()
const session = useSessionStore()
const ui = useUiStore()

const me = computed(() => session.me)
const name = computed(() => me.value?.name || me.value?.username || '')
const links = ref<PassportLink[]>([])
const linksLoaded = ref(false)
const linksError = ref('')
const messages = ref<PassportMessage[]>([])
const messagesLoaded = ref(false)
const unreadCount = ref(0)
const activity = ref<PassportActivity[]>([])
const activityLoaded = ref(false)
const bindable = ref('')
const reading = ref(false)

const gameCount = computed(() => new Set(links.value.map((l) => l.game)).size)

const subline = computed(() => {
  const m = me.value
  if (!m) return ''
  const parts = [`@${m.username}`, t('home.joined', { date: fmtDate(m.createTime) })]
  if (m.bio && !ui.isMobile) parts.push(`「${m.bio}」`)
  return parts.join(' · ')
})

const facts = computed(() => {
  const m = me.value
  if (!m) return []
  const email = m.email ? `${m.email} · ${m.emailVerified ? t('home.verified') : t('home.unverified')}` : t('home.noEmail')
  const last = m.lastLoginAt ? [fmtTime(m.lastLoginAt), m.lastLoginDevice].filter(Boolean).join(' · ') : '—'
  const methods = [t('common.method.password')]
  if (m.email && m.emailVerified && m.emailCodeLogin && session.config?.emailLoginEnabled) methods.push(t('common.method.email_code'))
  const all = [
    { k: t('home.facts.email'), v: ui.isMobile ? (m.emailVerified ? t('home.verified') : m.email ? t('home.unverified') : t('home.noEmail')) : email },
    { k: t('home.facts.lastLogin'), v: ui.isMobile && m.lastLoginAt ? fmtTime(m.lastLoginAt) : last },
    { k: t('home.facts.methods'), v: methods.join(' · ') },
    { k: t('home.facts.totp'), v: m.totpEnabled ? t('home.on') : t('home.off') },
  ]
  return ui.isMobile ? all.slice(0, 2) : all
})

const security = computed(() => {
  const m = me.value
  if (!m) return []
  const acc = (tab: string) => ({ path: '/account', query: { tab } })
  const changed = toDate(m.passwordUpdatedAt)
  return [
    {
      key: 'password',
      ok: true,
      text: changed ? t('home.security.password', { ago: fmtAgo(changed) }) : t('home.security.passwordSet'),
      act: '',
      to: acc('security'),
    },
    {
      key: 'email',
      ok: !!m.email && m.emailVerified,
      text: !m.email ? t('home.security.noEmail') : m.emailVerified ? t('home.security.emailOk') : t('home.security.emailUnverified'),
      act: m.email && m.emailVerified ? '' : t('home.security.fix'),
      to: acc('security'),
    },
    {
      key: 'totp',
      ok: m.totpEnabled,
      text: m.totpEnabled ? t('home.security.totpOn') : t('home.security.totpOff'),
      act: m.totpEnabled ? '' : t('home.security.enable'),
      to: acc('security'),
    },
    { key: 'devices', ok: true, text: t('home.security.devices', { n: m.deviceCount }), act: t('home.security.view'), to: acc('devices') },
  ]
})
const securityOk = computed(() => security.value.every((s) => s.ok))
const warning = computed(() => {
  const s = security.value.find((x) => !x.ok)
  return s ? { text: s.text, act: s.act, to: s.to } : null
})

const loadLinks = async () => {
  try {
    links.value = (await api.listLinks()).items || []
  } catch (e) {
    linksError.value = errorText(e)
  } finally {
    linksLoaded.value = true
  }
}

const loadMessages = async () => {
  try {
    const res = await api.listMessages(1, ui.isMobile ? 3 : 5)
    messages.value = res.items || []
    unreadCount.value = res.unread
    session.setUnread(res.unread)
  } catch {
    messages.value = []
  } finally {
    messagesLoaded.value = true
  }
}

const loadActivity = async () => {
  try {
    activity.value = (await api.listActivity(8)).items || []
  } catch {
    activity.value = []
  } finally {
    activityLoaded.value = true
  }
}

// 「这台 Momoko 上可以绑定」：可以绑定的游戏与推荐的服务器名称。
const loadBindable = async () => {
  try {
    const res = await api.searchBindServers('')
    const byGame = new Map<string, string[]>()
    for (const b of res.recommended || []) {
      if (!b.server) continue
      const list = byGame.get(b.server.game) || []
      list.push(b.server.name)
      byGame.set(b.server.game, list)
    }
    bindable.value = (res.games || [])
      .map((g) => {
        const names = byGame.get(g.id)
        const label = g.name || t('game.unnamed')
        return names?.length ? t('home.games.bindableItem', { game: label, servers: names.join(t('home.games.sep')) }) : label
      })
      .join(t('home.games.sep'))
  } catch {
    bindable.value = ''
  }
}

const readAll = async () => {
  reading.value = true
  try {
    const res = await api.readMessages([], true)
    session.setUnread(res.unread)
    unreadCount.value = res.unread
    messages.value = messages.value.map((m) => ({ ...m, read: true }))
  } catch (e) {
    ui.toast(errorText(e), 'bad')
  } finally {
    reading.value = false
  }
}

onMounted(() => {
  void loadLinks()
  void loadMessages()
  void loadActivity()
  void loadBindable()
  void session.loadMe().catch(() => undefined)
})
</script>

<style scoped>
.home {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
  padding: 28px 32px 56px;
  display: flex;
  flex-direction: column;
  gap: 28px;
}
.profile {
  padding: 26px 28px;
  display: flex;
  flex-wrap: wrap;
  gap: 28px;
}
.profile__main {
  flex: 3 1 520px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.profile__id {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 20px;
}
.profile__text {
  flex: 1 1 240px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.profile__name {
  font-size: 32px;
  overflow-wrap: anywhere;
}
.profile__sub {
  font-size: 14px;
  color: var(--m-ink2);
}
.profile__actions {
  display: flex;
  gap: 8px;
}
.facts {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  padding: 14px 16px;
  border-radius: 18px;
  background: var(--m-s2);
}
.facts__item {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.facts dd {
  margin: 0;
  font-size: 14px;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.security {
  flex: 1 1 280px;
  min-width: 0;
  padding: 18px;
  border-radius: 22px;
  background: var(--m-bg);
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.security__head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.security__title {
  flex: 1;
  font-size: 15px;
  font-weight: 800;
}
.security__row {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
}
.security__ico {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 8px;
  flex: none;
}
.security__ico svg {
  width: 14px;
  height: 14px;
}
.security__ico.is-ok {
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.security__ico.is-warn {
  background: var(--m-warnSoft);
  color: var(--m-warn);
}
.security__text {
  flex: 1;
}
.security__act {
  display: inline-flex;
  align-items: center;
  min-height: 40px;
  font-weight: 700;
  text-decoration: none;
}
.mwarn {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 44px;
  padding: 10px 12px;
  border-radius: 14px;
  background: var(--m-warnSoft);
  color: var(--m-ink);
  text-decoration: none;
  font-size: 13px;
}
.mwarn svg {
  width: 18px;
  height: 18px;
  color: var(--m-warn);
  flex: none;
}
.mwarn span {
  flex: 1;
}
.mwarn b {
  color: var(--m-warn);
}
.block {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.block__head {
  display: flex;
  align-items: center;
  gap: 12px;
}
.block__title {
  flex: 1;
  font-size: 24px;
}
.cols {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  align-items: flex-start;
}
.panel {
  flex: 1 1 460px;
  min-width: 0;
  padding: 18px 20px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.panel__head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}
.panel__title {
  flex: 1;
  font-size: 18px;
}
.act {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 10px 0;
  font-size: 13px;
}
.act__at {
  width: 96px;
  flex: none;
}
.act__where {
  flex: none;
}
.act__text {
  flex: 1;
  min-width: 0;
  line-height: 1.5;
}
@media (max-width: 768px) {
  .home {
    padding: 16px 16px 24px;
    gap: 22px;
  }
  .profile {
    padding: 18px;
    gap: 14px;
  }
  .profile__main {
    flex-basis: 100%;
  }
  .profile__id {
    gap: 14px;
    flex-wrap: nowrap;
  }
  .profile__name {
    font-size: 22px;
  }
  .profile__sub {
    font-size: 12px;
  }
  .facts {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .block__title {
    font-size: 19px;
  }
  .panel {
    flex-basis: 100%;
    padding: 16px;
  }
  .act {
    flex-wrap: wrap;
    gap: 6px 10px;
  }
  .act__at {
    width: auto;
  }
  .act__text {
    flex-basis: 100%;
  }
}
</style>
