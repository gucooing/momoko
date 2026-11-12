<template>
  <AuthLayout :title="mfaToken ? t('auth.mfa.title') : t('auth.login.title')" :subtitle="mfaToken ? t('auth.mfa.desc') : t('auth.login.desc')">
    <!-- 两步验证 -->
    <form v-if="mfaToken" class="form" @submit.prevent="submitMfa">
      <PField v-if="!useRecovery" :label="t('auth.mfa.code')">
        <input v-model="mfaCode" inputmode="numeric" autocomplete="one-time-code" maxlength="6" class="pp-num code" required />
      </PField>
      <PField v-else :label="t('auth.mfa.recovery')" :hint="t('auth.mfa.recoveryHint')">
        <input v-model="recoveryCode" autocomplete="off" class="pp-mono" required />
      </PField>
      <button type="button" class="link" @click="useRecovery = !useRecovery">
        {{ useRecovery ? t('auth.mfa.useCode') : t('auth.mfa.useRecovery') }}
      </button>
      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <PButton type="submit" variant="primary" size="lg" block :loading="busy">{{ t('auth.mfa.submit') }}</PButton>
      <button type="button" class="link" @click="resetMfa">{{ t('auth.mfa.back') }}</button>
    </form>

    <template v-else>
      <div v-if="codeLogin" class="pp-seg" role="tablist" :aria-label="t('auth.login.method')">
        <button type="button" role="tab" :class="{ 'is-on': mode === 'password' }" :aria-selected="mode === 'password'" @click="mode = 'password'">
          {{ t('auth.login.byPassword') }}
        </button>
        <button type="button" role="tab" :class="{ 'is-on': mode === 'code' }" :aria-selected="mode === 'code'" @click="mode = 'code'">
          {{ t('auth.login.byCode') }}
        </button>
      </div>
      <form class="form" @submit.prevent="submit">
        <template v-if="mode === 'password'">
          <PField :label="t('auth.login.account')">
            <input v-model.trim="account" autocomplete="username" :placeholder="t('auth.login.accountHint')" required />
          </PField>
          <PField :label="t('auth.login.password')">
            <input v-model="password" type="password" autocomplete="current-password" required />
          </PField>
        </template>
        <template v-else>
          <PField :label="t('auth.email')">
            <input v-model.trim="email" type="email" autocomplete="email" required />
          </PField>
          <PField :label="t('auth.code')">
            <span class="code-row">
              <input v-model.trim="code" inputmode="numeric" autocomplete="one-time-code" maxlength="6" class="pp-num" required />
              <PButton size="sm" :disabled="left > 0 || !email" :loading="sending" @click="sendCode">
                {{ left > 0 ? t('common.resendIn', { n: left }) : t('common.sendCode') }}
              </PButton>
            </span>
          </PField>
        </template>
        <label class="remember">
          <input v-model="remember" type="checkbox" />
          <span>{{ t('auth.login.remember') }}</span>
        </label>
        <p v-if="error" class="error" role="alert">{{ error }}</p>
        <PButton type="submit" variant="primary" size="lg" block :loading="busy">{{ t('auth.login.submit') }}</PButton>
      </form>
      <div class="links">
        <RouterLink v-if="canRegister" :to="{ path: '/register', query: route.query }">{{ t('auth.login.toRegister') }}</RouterLink>
        <span v-else class="pp-hint">{{ t('auth.register.closed') }}</span>
        <RouterLink v-if="session.config?.emailEnabled" to="/forgot">{{ t('auth.login.forgot') }}</RouterLink>
      </div>
    </template>
  </AuthLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AuthLayout from '@passport/layouts/AuthLayout.vue'
import PButton from '@passport/components/PButton.vue'
import PField from '@passport/components/PField.vue'
import * as api from '@passport/api/passport'
import { deviceId, deviceName } from '@passport/api/request'
import { useSessionStore } from '@passport/stores/session'
import { useUiStore } from '@passport/stores/ui'
import { errorText } from '@passport/utils/format'
import { useCountdown } from '@passport/utils/countdown'
import type { PassportLoginResponse } from '@/types/v1/passport'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const ui = useUiStore()
const { left, start } = useCountdown()

const mode = ref<'password' | 'code'>('password')
const account = ref('')
const password = ref('')
const email = ref('')
const code = ref('')
const remember = ref(true)
const busy = ref(false)
const sending = ref(false)
const error = ref('')
const mfaToken = ref('')
const mfaCode = ref('')
const recoveryCode = ref('')
const useRecovery = ref(false)

const codeLogin = computed(() => !!session.config?.emailLoginEnabled)
const canRegister = computed(() => session.config?.registerMode !== 'closed')

onMounted(() => {
  void session.loadConfig().catch(() => undefined)
})

const sendCode = async () => {
  sending.value = true
  error.value = ''
  try {
    const res = await api.sendLoginCode(email.value)
    start(res.resendAfterSeconds || 60)
    ui.toast(t('common.codeSent', { n: Math.round((res.expiresInSeconds || 300) / 60) }), 'ok')
  } catch (e) {
    error.value = errorText(e)
  } finally {
    sending.value = false
  }
}

const finish = async (res: PassportLoginResponse) => {
  if (res.mfaRequired) {
    mfaToken.value = res.mfaToken
    return
  }
  session.acceptTokens(res)
  await session.loadMe()
  await router.replace(String(route.query.redirect || '/'))
}

const submit = async () => {
  busy.value = true
  error.value = ''
  try {
    const res =
      mode.value === 'password'
        ? await api.login({ account: account.value, password: password.value, deviceId: deviceId(), device: deviceName(), remember: remember.value })
        : await api.login({ email: email.value, code: code.value, deviceId: deviceId(), device: deviceName(), remember: remember.value })
    await finish(res)
  } catch (e) {
    error.value = errorText(e)
  } finally {
    busy.value = false
  }
}

const submitMfa = async () => {
  busy.value = true
  error.value = ''
  try {
    const res = useRecovery.value
      ? await api.loginMfa(mfaToken.value, '', recoveryCode.value)
      : await api.loginMfa(mfaToken.value, mfaCode.value)
    await finish(res)
  } catch (e) {
    error.value = errorText(e)
  } finally {
    busy.value = false
  }
}

const resetMfa = () => {
  mfaToken.value = ''
  mfaCode.value = ''
  recoveryCode.value = ''
  error.value = ''
}
</script>

<style scoped>
.form {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.code-row {
  display: flex;
  gap: 8px;
}
.code-row input {
  flex: 1;
}
.code {
  letter-spacing: 0.4em;
  font-size: 20px !important;
  text-align: center;
}
.remember {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  font-size: 14px;
  cursor: pointer;
}
.remember input {
  width: 18px;
  height: 18px;
  accent-color: var(--m-p);
}
.error {
  margin: 0;
  padding: 10px 12px;
  border-radius: 12px;
  background: var(--m-badSoft);
  color: var(--m-bad);
  font-size: 13px;
  font-weight: 700;
}
.links {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  font-size: 14px;
  font-weight: 700;
}
.links a,
.link {
  color: var(--m-pInk);
  text-decoration: none;
}
.link {
  align-self: flex-start;
  min-height: 40px;
  padding: 0;
  border: 0;
  background: transparent;
  font: inherit;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
}
</style>
