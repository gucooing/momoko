<template>
  <AuthLayout :title="t('auth.register.title')" :subtitle="t('auth.register.desc')">
    <p v-if="closed" class="pp-note">{{ t('auth.register.closed') }}</p>
    <form v-else class="pp-form" @submit.prevent="submit">
      <PField :label="t('auth.register.username')" :hint="t('auth.register.usernameHint')">
        <input v-model.trim="form.username" autocomplete="username" maxlength="20" required />
      </PField>
      <PField :label="needEmail ? t('auth.email') : t('auth.emailOptional')" :hint="t('auth.register.emailHint')">
        <input v-model.trim="form.email" type="email" autocomplete="email" :required="needEmail" />
      </PField>
      <PField v-if="needCode" :label="t('auth.code')">
        <span class="pp-code-row">
          <input v-model.trim="form.code" inputmode="numeric" autocomplete="one-time-code" maxlength="6" class="pp-num" required />
          <PButton size="sm" :disabled="left > 0 || !form.email" :loading="sending" @click="sendCode">
            {{ left > 0 ? t('common.resendIn', { n: left }) : t('common.sendCode') }}
          </PButton>
        </span>
      </PField>
      <PField :label="t('auth.register.password')" :hint="t('auth.register.passwordHint', { n: minLength })">
        <input v-model="form.password" type="password" autocomplete="new-password" required />
      </PField>
      <PField :label="t('auth.register.confirm')" :error="mismatch ? t('auth.register.mismatch') : ''">
        <input v-model="confirm" type="password" autocomplete="new-password" required />
      </PField>
      <PField :label="inviteOnly ? t('auth.register.invite') : t('auth.register.inviteOptional')" :hint="t('auth.register.inviteHint')">
        <input v-model.trim="form.inviteCode" class="pp-mono" autocomplete="off" :required="inviteOnly" />
      </PField>
      <p v-if="error" class="pp-error" role="alert">{{ error }}</p>
      <PButton type="submit" variant="primary" size="lg" block :loading="busy" :disabled="mismatch">{{ t('auth.register.submit') }}</PButton>
    </form>
    <RouterLink :to="{ path: '/login', query: route.query }" class="pp-link">{{ t('auth.register.toLogin') }}</RouterLink>
  </AuthLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
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

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const ui = useUiStore()
const { left, start } = useCountdown()

const form = reactive({ username: '', email: '', code: '', password: '', inviteCode: String(route.query.invite || '') })
const confirm = ref('')
const busy = ref(false)
const sending = ref(false)
const error = ref('')

const closed = computed(() => session.config?.registerMode === 'closed')
const inviteOnly = computed(() => session.config?.registerMode === 'invite_only')
const needCode = computed(() => !!session.config?.registerEmailVerification)
const needEmail = computed(() => needCode.value)
const minLength = computed(() => session.config?.minPasswordLength || 8)
const mismatch = computed(() => !!confirm.value && confirm.value !== form.password)

onMounted(() => {
  void session.loadConfig().catch(() => undefined)
})

const sendCode = async () => {
  sending.value = true
  error.value = ''
  try {
    const res = await api.sendRegisterCode(form.email)
    start(res.resendAfterSeconds || 60)
    ui.toast(t('common.codeSent', { n: Math.round((res.expiresInSeconds || 300) / 60) }), 'ok')
  } catch (e) {
    error.value = errorText(e)
  } finally {
    sending.value = false
  }
}

const submit = async () => {
  if (mismatch.value) return
  busy.value = true
  error.value = ''
  try {
    const res = await api.register({ ...form, deviceId: deviceId(), device: deviceName(), language: locale.value })
    session.acceptTokens(res)
    await session.loadMe()
    await router.replace(String(route.query.redirect || '/bind'))
  } catch (e) {
    error.value = errorText(e)
  } finally {
    busy.value = false
  }
}
</script>
