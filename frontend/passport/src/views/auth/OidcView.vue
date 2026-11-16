<!-- 「用户通行证登录」授权确认页（OIDC authorization_endpoint）。已授权过且范围不变时直接回到应用。
     应用关联了游戏服务器时显示你在那台服务器上的游戏账号：已绑定时以它登录；没有绑定时要先绑定已有账号（去绑定页，
     绑定后回到这里）或新建一个账号并继续，这时不会自动同意。 -->
<template>
  <AuthLayout :title="t('auth.oidc.title')" :subtitle="info ? t('auth.oidc.desc', { app: info.clientName }) : ''">
    <p v-if="loading" class="pp-hint">{{ t('common.loading') }}</p>
    <template v-else-if="info">
      <div class="app">
        <GameMark v-if="info.server" :game="info.server.game" :name="info.server.gameName || info.server.name" :size="48" />
        <span v-else class="app__mark pp-sticker">{{ info.clientName.slice(0, 1).toUpperCase() }}</span>
        <span class="app__body">
          <span class="app__name">{{ info.clientName }}</span>
          <span v-if="info.server" class="pp-hint">{{ info.server.gameName || info.server.game }} · {{ info.server.name }}</span>
          <span v-if="info.description" class="pp-hint">{{ info.description }}</span>
        </span>
      </div>
      <div class="scopes">
        <span class="scopes__title">{{ t('auth.oidc.wants') }}</span>
        <span v-for="s in info.scopes" :key="s" class="scope">
          <CheckCircleIcon aria-hidden="true" />
          {{ scopeLabel(s) }}
        </span>
      </div>
      <div v-if="session.me" class="who">
        <PAvatar :src="session.me.avatar" :asset="session.me.avatarAsset" :name="session.me.name || session.me.username" :size="36" />
        <span>{{ t('auth.oidc.as', { name: session.me.name || session.me.username }) }}</span>
      </div>

      <section v-if="info.gameAccount || info.gameAccountRequired" class="game" :aria-label="t('auth.oidc.game.title')">
        <span class="game__title">{{ t('auth.oidc.game.title') }}</span>
        <div v-if="info.gameAccount" class="game__acc" :data-game="info.server?.game">
          <LinkAvatar :game="info.server?.game || ''" :asset="info.gameAccount.avatarAsset" :name="accountName" :size="44" />
          <span class="game__body">
            <span class="game__name">{{ accountName }}</span>
            <span class="pp-hint">{{ accountLine }}</span>
          </span>
          <span class="pp-tag pp-tag--p">{{ t('auth.oidc.game.linked') }}</span>
        </div>
        <template v-else>
          <p class="pp-note">{{ t('auth.oidc.game.required', { server: serverName }) }}</p>
          <div class="game__choices">
            <PButton v-if="bindTo" block :to="bindTo" :icon="LinkIcon">{{ t('auth.oidc.game.bind') }}</PButton>
            <PButton
              block
              variant="primary"
              :icon="PlusCircleIcon"
              :loading="busy"
              :disabled="!info.gameAccountCreatable"
              @click="approve(true)"
            >
              {{ t('auth.oidc.game.create') }}
            </PButton>
          </div>
          <p class="pp-hint">{{ createLine }}</p>
        </template>
      </section>

      <p class="pp-hint">{{ t('auth.oidc.redirectTo', { uri: redirectHost }) }}</p>
      <p v-if="error" class="pp-error" role="alert">{{ error }}</p>
      <div class="actions">
        <PButton block @click="deny">{{ t('auth.oidc.deny') }}</PButton>
        <PButton v-if="!info.gameAccountRequired" block variant="primary" :loading="busy" @click="approve(false)">
          {{ t('auth.oidc.allow') }}
        </PButton>
      </div>
    </template>
    <p v-else class="pp-error" role="alert">{{ error }}</p>
  </AuthLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, type RouteLocationRaw } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { CheckCircleIcon, LinkIcon, PlusCircleIcon } from '@heroicons/vue/24/outline'
import AuthLayout from '@passport/layouts/AuthLayout.vue'
import GameMark from '@passport/components/GameMark.vue'
import LinkAvatar from '@passport/components/LinkAvatar.vue'
import PAvatar from '@passport/components/PAvatar.vue'
import PButton from '@passport/components/PButton.vue'
import * as api from '@passport/api/passport'
import { useSessionStore } from '@passport/stores/session'
import { errorText, fmtDate } from '@passport/utils/format'
import type { PassportOidcAuthorizationInfo, PassportOidcAuthorizeRequest } from '@/types/v1/passport'

const { t, te } = useI18n()
const route = useRoute()
const session = useSessionStore()

const info = ref<PassportOidcAuthorizationInfo | null>(null)
const loading = ref(true)
const busy = ref(false)
const error = ref('')

const q = (k: string) => String(route.query[k] ?? '')
const request = computed<Omit<PassportOidcAuthorizeRequest, 'createGameAccount'>>(() => ({
  responseType: q('response_type'),
  clientId: q('client_id'),
  redirectUri: q('redirect_uri'),
  scope: q('scope'),
  state: q('state'),
  nonce: q('nonce'),
  codeChallenge: q('code_challenge'),
  codeChallengeMethod: q('code_challenge_method'),
}))

const redirectHost = computed(() => {
  try {
    return new URL(info.value?.redirectUri || '').host
  } catch {
    return info.value?.redirectUri || ''
  }
})

const scopeLabel = (s: string) => (te(`auth.oidc.scopes.${s}`) ? t(`auth.oidc.scopes.${s}`) : s)

// ---------- 游戏账号 ----------
const serverName = computed(() => info.value?.server?.name || t('auth.oidc.game.unknownServer'))
const accountName = computed(() => {
  const a = info.value?.gameAccount
  return a?.nickname || a?.account || t('auth.oidc.game.noName')
})
const spaced = (id: string) => id.replace(/(\d{3})(?=\d)/g, '$1 ')
const accountLine = computed(() => {
  const a = info.value?.gameAccount
  if (!a) return ''
  const parts: string[] = []
  if (a.level) parts.push(t('home.games.level', { n: a.level }))
  if (a.displayId) parts.push(`ID ${spaced(a.displayId)}`)
  if (a.account && a.nickname) parts.push(a.account)
  return parts.join(' · ')
})
// 去绑定页绑定已有账号（预选这台服务器），绑定后回到本页
const bindTo = computed<RouteLocationRaw | undefined>(() => {
  const id = info.value?.server?.id
  return id ? { path: '/bind', query: { server: id, redirect: route.fullPath } } : undefined
})
const createLine = computed(() => {
  const i = info.value
  if (!i || i.gameAccountCreatable) return t('auth.oidc.game.createHint')
  const reason = i.gameAccountReason || 'unsupported'
  return t(`auth.oidc.game.blocked.${reason}`, { date: fmtDate(i.gameAccountCooldownUntil) })
})

const approve = async (createGameAccount: boolean) => {
  busy.value = true
  error.value = ''
  try {
    const res = await api.approveOidcAuthorization({ ...request.value, createGameAccount })
    window.location.assign(res.redirectUrl)
  } catch (e) {
    error.value = errorText(e)
    busy.value = false
  }
}

// 拒绝：按 OAuth 约定带 error=access_denied 回到（已校验过的）应用回调地址。
const deny = () => {
  if (!info.value) return
  const u = new URL(info.value.redirectUri)
  u.searchParams.set('error', 'access_denied')
  if (request.value.state) u.searchParams.set('state', request.value.state)
  window.location.assign(u.toString())
}

onMounted(async () => {
  try {
    info.value = await api.getOidcAuthorization(request.value)
    // 已授权过、范围不变且不需要先处理游戏账号时直接继续
    if (info.value.granted && !info.value.gameAccountRequired) {
      await approve(false)
      return
    }
  } catch (e) {
    error.value = errorText(e)
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.app {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px;
  border-radius: 18px;
  background: var(--m-s2);
}
.app__mark {
  width: 48px;
  height: 48px;
  border-radius: 15px;
  background: var(--m-a2Soft);
  color: var(--m-ink);
  font-size: 20px;
}
.app__body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.app__name {
  font-size: 16px;
  font-weight: 800;
}
.scopes {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.scopes__title {
  font-size: 13px;
  font-weight: 800;
}
.scope {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}
.scope svg {
  width: 20px;
  height: 20px;
  color: var(--m-ok);
}
.who {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
  font-weight: 700;
}
.game {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.game__title {
  font-size: 13px;
  font-weight: 800;
}
.game__acc {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border-radius: 18px;
  border: 1.5px solid var(--m-line);
  background: var(--m-s);
}
.game__body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.game__name {
  font-size: 15px;
  font-weight: 800;
}
.game__choices {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.actions {
  display: flex;
  gap: 10px;
}
</style>
