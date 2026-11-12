<!-- 绑定游戏账号：选服务器（搜索 + 按游戏筛选 + 推荐，不列出全部）→ 证明这是你的账号（服务器声明支持的方式）。 -->
<template>
  <div class="bind">
    <header v-if="ui.isMobile" class="mhead">
      <button type="button" class="mhead__ib" :aria-label="t('nav.back')" @click="back"><ChevronLeftIcon aria-hidden="true" /></button>
      <span class="mhead__title">{{ t('bind.mobileTitle') }}</span>
      <button type="button" class="mhead__ib" :aria-label="t('nav.toggleDark')" @click="ui.toggleDark()">
        <component :is="ui.dark ? SunIcon : MoonIcon" aria-hidden="true" />
      </button>
    </header>

    <div class="bind__body">
      <h1 v-if="!ui.isMobile" class="pp-title bind__title">{{ t('home.games.bind') }}</h1>
      <div v-if="session.me" class="who pp-card">
        <PAvatar :src="session.me.avatar" :asset="session.me.avatarAsset" :name="session.me.name || session.me.username" :size="44" />
        <span class="who__text">
          <span class="who__name">{{ session.me.name || session.me.username }}</span>
          <span class="pp-hint">{{ t('bind.signedIn') }}</span>
        </span>
        <PPill :tone="linkCount ? 'ok' : 'warn'">{{ linkCount ? t('bind.linkedN', { n: linkCount }) : t('bind.unlinked') }}</PPill>
      </div>

      <section class="step">
        <span class="step__title">{{ t('bind.step1') }}</span>
        <label class="search">
          <MagnifyingGlassIcon aria-hidden="true" />
          <input v-model="q" type="search" :placeholder="t('bind.searchHint')" :aria-label="t('bind.searchHint')" />
        </label>
        <div v-if="games.length > 1" class="pp-scroll-x" role="tablist" :aria-label="t('bind.byGame')">
          <button type="button" role="tab" class="pp-chip" :class="{ 'is-on': !game }" :aria-selected="!game" @click="game = ''">
            {{ t('bind.allGames') }}
          </button>
          <button
            v-for="g in games"
            :key="g.id"
            type="button"
            role="tab"
            class="pp-chip"
            :class="{ 'is-on': game === g.id }"
            :aria-selected="game === g.id"
            @click="game = g.id"
          >
            {{ g.name || t('game.unnamed') }} <span class="pp-chip__n">{{ g.servers }}</span>
          </button>
        </div>
        <span class="pp-hint">{{ q.trim() ? t('bind.resultsHint') : t('bind.recommendHint') }}</span>
        <p v-if="error" class="pp-error">{{ error }}</p>
        <PEmpty v-else-if="loaded && !list.length" compact :icon="ServerStackIcon" :title="q.trim() ? t('bind.noResults') : t('bind.noServers')" />
        <button
          v-for="s in list"
          :key="s.server?.id"
          type="button"
          class="srv"
          :class="{ 'is-on': selectedId === s.server?.id }"
          :disabled="!s.bindable"
          :aria-pressed="selectedId === s.server?.id"
          :data-game="s.server?.game"
          @click="select(s)"
        >
          <GameMark :game="s.server?.game || ''" :name="s.server?.gameName || s.server?.game" :size="40" />
          <span class="srv__text">
            <span class="srv__name">
              {{ s.server?.gameName ? `${s.server.gameName} · ` : '' }}{{ s.server?.name }}
              <span v-if="s.viaInvite" class="pp-tag pp-tag--p">{{ t('bind.viaInvite') }}</span>
            </span>
            <span class="pp-hint">{{ srvLine(s) }}</span>
          </span>
          <CheckCircleIcon v-if="selectedId === s.server?.id" class="srv__check" aria-hidden="true" />
        </button>
        <span class="pp-hint center">{{ t('bind.notFound') }}</span>
      </section>

      <section class="step">
        <span class="step__title">{{ t('bind.step2') }}</span>
        <p v-if="!selected" class="pp-note">{{ t('bind.pickFirst') }}</p>
        <template v-else>
          <div class="pp-seg" role="radiogroup" :aria-label="t('bind.method')">
            <button v-for="m in selected.methods" :key="m" type="button" role="radio" :aria-checked="method === m" :class="{ 'is-on': method === m }" @click="method = m">
              {{ t(`bind.methods.${m}`) }}
            </button>
          </div>
          <form v-if="method === 'password'" id="bind-form" class="pp-form" @submit.prevent="submit">
            <PField :label="t('bind.account')">
              <input v-model.trim="account" autocomplete="off" :placeholder="t('bind.accountHint')" required />
            </PField>
            <PField :label="t('bind.password')">
              <input v-model="password" type="password" autocomplete="off" :placeholder="t('bind.passwordHint')" required />
            </PField>
          </form>
        </template>
      </section>

      <p class="pp-note">{{ t('bind.rules', { days: session.config?.unbindCooldownDays ?? 7 }) }}</p>
    </div>

    <div class="cta">
      <PButton
        type="submit"
        form="bind-form"
        variant="primary"
        size="lg"
        block
        :loading="busy"
        :disabled="!selected || !account || !password"
      >
        {{ t('bind.submit') }}
      </PButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import {
  CheckCircleIcon,
  ChevronLeftIcon,
  MagnifyingGlassIcon,
  MoonIcon,
  ServerStackIcon,
  SunIcon,
} from '@heroicons/vue/24/outline'
import GameMark from '@passport/components/GameMark.vue'
import PAvatar from '@passport/components/PAvatar.vue'
import PButton from '@passport/components/PButton.vue'
import PEmpty from '@passport/components/PEmpty.vue'
import PField from '@passport/components/PField.vue'
import PPill from '@passport/components/PPill.vue'
import * as api from '@passport/api/passport'
import { useSessionStore } from '@passport/stores/session'
import { useUiStore } from '@passport/stores/ui'
import { errorText, fmtDate, fmtNum } from '@passport/utils/format'
import type { PassportBindGame, PassportBindServer } from '@/types/v1/passport'

const { t } = useI18n()
const router = useRouter()
const session = useSessionStore()
const ui = useUiStore()

const q = ref('')
const game = ref('')
const items = ref<PassportBindServer[]>([])
const recommended = ref<PassportBindServer[]>([])
const games = ref<PassportBindGame[]>([])
const loaded = ref(false)
const error = ref('')
const selected = ref<PassportBindServer | null>(null)
const method = ref('')
const account = ref('')
const password = ref('')
const busy = ref(false)

const selectedId = computed(() => selected.value?.server?.id || '')
const linkCount = computed(() => session.me?.linkCount ?? 0)
const list = computed(() => (q.value.trim() ? items.value : recommended.value))

let seq = 0
const load = async () => {
  const id = ++seq
  error.value = ''
  try {
    const res = await api.searchBindServers(q.value.trim(), game.value)
    if (id !== seq) return
    items.value = res.items || []
    recommended.value = res.recommended || []
    games.value = res.games || []
  } catch (e) {
    if (id === seq) error.value = errorText(e)
  } finally {
    if (id === seq) loaded.value = true
  }
}
const debounced = useDebounceFn(load, 300)
watch(q, () => void debounced())
watch(game, () => void load())
onMounted(() => void load())

const srvLine = (s: PassportBindServer) => {
  switch (s.reason) {
    case 'linked':
      return t('bind.reason.linked')
    case 'cooldown':
      return t('bind.reason.cooldown', { date: fmtDate(s.cooldownUntil) })
    case 'no_mgp':
      return t('bind.reason.no_mgp')
    case 'unsupported':
      return t('bind.reason.unsupported')
  }
  const parts = [t('bind.accounts', { n: fmtNum(s.linkedAccounts) })]
  if (s.selfService) parts.push(t('bind.selfService', { n: s.selfService }))
  return parts.join(' · ')
}

const select = (s: PassportBindServer) => {
  if (!s.bindable) return
  selected.value = s
  method.value = s.methods[0] || ''
}

const back = () => {
  if (window.history.length > 1) router.back()
  else void router.push('/')
}

const submit = async () => {
  const s = selected.value
  if (!s?.server) return
  busy.value = true
  try {
    await api.bindAccount({
      serverId: s.server.id,
      inviteCode: s.viaInvite ? q.value.trim() : '',
      account: account.value,
      password: password.value,
    })
    password.value = ''
    ui.toast(t('bind.done'), 'ok')
    void session.loadMe().catch(() => undefined)
    await router.push(`/games/${encodeURIComponent(s.server.id)}`)
  } catch (e) {
    ui.toast(errorText(e), 'bad')
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.bind {
  min-height: 100%;
  display: flex;
  flex-direction: column;
}
.mhead {
  position: sticky;
  top: 0;
  z-index: 20;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 12px;
  background: var(--m-bg);
  border-bottom: 1px solid var(--m-line);
}
.mhead__title {
  flex: 1;
  font-size: 17px;
  font-weight: 800;
}
.mhead__ib {
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
.mhead__ib svg {
  width: 22px;
  height: 22px;
}
.bind__body {
  width: 100%;
  max-width: 640px;
  margin: 0 auto;
  padding: 28px 24px 24px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.bind__title {
  font-size: 28px;
}
.who {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
}
.who__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.who__name {
  font-size: 15px;
  font-weight: 800;
}
.step {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.step__title {
  font-size: 14px;
  font-weight: 800;
}
.search {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 46px;
  padding: 0 14px;
  border-radius: 14px;
  border: 1.5px solid var(--m-line);
  background: var(--m-s);
}
.search:focus-within {
  border-color: var(--m-p);
}
.search svg {
  width: 18px;
  height: 18px;
  color: var(--m-ink2);
  flex: none;
}
.search input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--m-ink);
  font: inherit;
  font-size: 14px;
}
.srv {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-height: 64px;
  padding: 12px;
  border-radius: 20px;
  border: 2px solid var(--m-line);
  background: var(--m-s);
  color: var(--m-ink);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.srv.is-on {
  border-color: var(--m-p);
  background: var(--m-pSoft);
}
.srv:disabled {
  cursor: default;
  opacity: 0.7;
}
.srv__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.srv__name {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  font-size: 15px;
  font-weight: 800;
}
.srv__check {
  width: 24px;
  height: 24px;
  color: var(--m-pInk);
  flex: none;
}
.center {
  text-align: center;
}
.cta {
  position: sticky;
  bottom: 0;
  width: 100%;
  max-width: 640px;
  margin: 0 auto;
  padding: 12px 24px 18px;
  background: var(--m-bg);
}
@media (max-width: 768px) {
  .bind__body {
    padding: 16px 16px 12px;
  }
  .cta {
    max-width: none;
    padding: 12px 16px calc(18px + env(safe-area-inset-bottom));
    background: var(--m-s);
    border-top: 1px solid var(--m-line);
  }
}
</style>
