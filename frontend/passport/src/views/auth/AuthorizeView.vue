<!-- 客户端浏览器授权页（/authorize?code=XXXX-XXXX）：客户端不接触通行证的账号与密码，玩家在这里授权或拒绝。
     要登录（没登录时先走正常登录再回来，code 保留）；显示客户端名称、设备名与发起时间。
     学マス 的 PC 启动器与 Android APK 要核对客户端上显示的代码（大字）；没有代码时可以手动输入。
     蓝色星原的游戏内登录是浏览器单点授权：授权页由客户端直接打开，不显示、不核对代码。 -->
<template>
  <AuthLayout :title="sso ? t('authorize.sso.title', { client: clientName }) : t('authorize.title')" :subtitle="sso ? t('authorize.sso.desc', { client: clientName }) : t('authorize.desc')">
    <div v-if="state === 'loading'" class="az__state">{{ t('authorize.loading') }}</div>

    <div v-else-if="state === 'approved'" class="az__result">
      <span class="az__mark is-ok" aria-hidden="true"><CheckCircleIcon /></span>
      <b>{{ t('authorize.approved') }}</b>
      <p>{{ t('authorize.approvedDesc') }}</p>
    </div>

    <div v-else-if="state === 'denied'" class="az__result">
      <span class="az__mark is-bad" aria-hidden="true"><XCircleIcon /></span>
      <b>{{ t('authorize.denied') }}</b>
      <p>{{ t('authorize.deniedDesc') }}</p>
    </div>

    <div v-else-if="state === 'ready' && request" class="az">
      <dl class="az__facts">
        <div>
          <dt>{{ t('authorize.client') }}</dt>
          <dd>{{ clientName }}</dd>
        </div>
        <div>
          <dt>{{ t('authorize.device') }}</dt>
          <dd>{{ request.device || '—' }}</dd>
        </div>
        <div>
          <dt>{{ t('authorize.requestedAt') }}</dt>
          <dd class="pp-num">{{ fmtTime(request.createTime) }} · {{ t('authorize.expiresIn', { left: fmtLeft(request.expireTime) }) }}</dd>
        </div>
      </dl>
      <div v-if="!sso" class="az__code">
        <span class="az__code-label">{{ t('authorize.codeLabel') }}</span>
        <span class="az__code-value pp-num">{{ request.userCode }}</span>
        <span class="pp-hint">{{ t('authorize.codeHint') }}</span>
      </div>
      <p class="pp-note">{{ t('authorize.as', { name: accountName }) }}</p>
      <p v-if="error" class="pp-error" role="alert">{{ error }}</p>
      <div class="az__acts">
        <PButton variant="ghost" size="lg" :disabled="busy" @click="decide(false)">{{ t('authorize.deny') }}</PButton>
        <PButton variant="primary" size="lg" :loading="busy" @click="decide(true)">{{ t('authorize.approve') }}</PButton>
      </div>
    </div>

    <form v-else class="pp-form" @submit.prevent="submitCode">
      <p v-if="state === 'missing'" class="pp-error" role="alert">{{ t('authorize.missing') }}</p>
      <p v-else-if="error" class="pp-error" role="alert">{{ error }}</p>
      <PField :label="t('authorize.enterCode')" :hint="t('authorize.enterCodeHint')">
        <input
          v-model="typed"
          class="pp-num az__input"
          autocomplete="off"
          autocapitalize="characters"
          spellcheck="false"
          maxlength="9"
          placeholder="XXXX-XXXX"
          required
        />
      </PField>
      <PButton type="submit" variant="primary" size="lg" block>{{ t('authorize.continue') }}</PButton>
    </form>

    <RouterLink to="/" class="pp-link">{{ t('authorize.home') }}</RouterLink>
  </AuthLayout>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { CheckCircleIcon, XCircleIcon } from '@heroicons/vue/24/outline'
import AuthLayout from '@passport/layouts/AuthLayout.vue'
import PButton from '@passport/components/PButton.vue'
import PField from '@passport/components/PField.vue'
import * as api from '@passport/api/passport'
import { PassportError } from '@passport/api/request'
import { useSessionStore } from '@passport/stores/session'
import { errorText, fmtLeft, fmtTime } from '@passport/utils/format'
import type { PassportClientAuth } from '@/types/v1/passport'

type State = 'input' | 'loading' | 'ready' | 'approved' | 'denied' | 'missing'

const { t, te } = useI18n()
const route = useRoute()
const router = useRouter()
const session = useSessionStore()

const state = ref<State>('input')
const request = ref<PassportClientAuth | null>(null)
const error = ref('')
const busy = ref(false)
const typed = ref('')

const code = computed(() => String(route.query.code || '').trim())

const clientName = computed(() => {
  const r = request.value
  if (!r) return ''
  const key = `authorize.clients.${r.client}`
  return te(key) ? t(key) : r.clientName || r.client
})
// 浏览器单点授权的客户端（蓝色星原游戏内登录）：授权页由客户端直接打开，不核对代码
const SSO_CLIENTS = ['azurpromilia']
const sso = computed(() => !!request.value && SSO_CLIENTS.includes(request.value.client))
const accountName = computed(() => {
  const me = session.me
  if (!me) return ''
  return me.name && me.name !== me.username ? `${me.name} (@${me.username})` : `@${me.username}`
})

const isNotFound = (e: unknown) => e instanceof PassportError && e.code === 'PASSPORT_AUTH_NOT_FOUND'

let seq = 0
const load = async (value: string) => {
  const id = ++seq
  error.value = ''
  request.value = null
  if (!value) {
    state.value = 'input'
    return
  }
  state.value = 'loading'
  try {
    const res = await api.getClientAuth(value)
    if (id !== seq) return
    request.value = res
    state.value = 'ready'
  } catch (e) {
    if (id !== seq) return
    typed.value = value
    if (isNotFound(e)) {
      state.value = 'missing'
    } else {
      state.value = 'input'
      error.value = errorText(e)
    }
  }
}
watch(code, (v) => void load(v), { immediate: true })

const submitCode = () => {
  const value = typed.value.trim().toUpperCase()
  if (!value) return
  if (value === code.value) void load(value)
  else void router.replace({ query: { ...route.query, code: value } })
}

const decide = async (approve: boolean) => {
  const r = request.value
  if (!r) return
  busy.value = true
  error.value = ''
  try {
    if (approve) await api.approveClientAuth(r.userCode)
    else await api.denyClientAuth(r.userCode)
    state.value = approve ? 'approved' : 'denied'
  } catch (e) {
    if (isNotFound(e)) {
      typed.value = r.userCode
      state.value = 'missing'
    } else {
      error.value = errorText(e)
    }
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.az {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.az__state {
  padding: 16px 0;
  text-align: center;
  color: var(--m-ink2);
  font-size: 13px;
}
.az__facts {
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px 16px;
  border-radius: 16px;
  background: var(--m-s2);
}
.az__facts > div {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}
.az__facts dt {
  color: var(--m-ink2);
  font-size: 13px;
}
.az__facts dd {
  margin: 0;
  min-width: 0;
  text-align: right;
  font-size: 13px;
  font-weight: 600;
  word-break: break-all;
}
.az__code {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 14px 12px;
  border-radius: 14px;
  border: 2px dashed var(--m-line);
}
.az__code-label {
  color: var(--m-ink2);
  font-size: 13px;
  font-weight: 500;
}
.az__code-value {
  font-family: var(--app-font-mono, ui-monospace, monospace);
  font-size: 26px;
  font-weight: 600;
  color: var(--m-ink);
}
.az__acts {
  display: grid;
  grid-template-columns: 1fr 2fr;
  gap: 10px;
}
.az__input {
  font-family: var(--app-font-mono, ui-monospace, monospace);
  font-size: 20px;
}
.az__result {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 12px 0 4px;
  text-align: center;
}
.az__result b {
  font-size: 15px;
}
.az__result p {
  margin: 0;
  color: var(--m-ink2);
  font-size: 13px;
  line-height: 1.6;
}
.az__mark {
  width: 44px;
  height: 44px;
}
.az__mark svg {
  width: 100%;
  height: 100%;
}
.az__mark.is-ok {
  color: var(--m-ok);
}
.az__mark.is-bad {
  color: var(--m-bad);
}
@media (max-width: 480px) {
  .az__code-value {
    font-size: 24px;
  }
  .az__acts {
    grid-template-columns: 1fr;
  }
}
</style>
