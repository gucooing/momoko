<template>
  <AuthLayout :title="t('auth.forgot.title')" :subtitle="t('auth.forgot.desc')">
    <p v-if="session.config && !session.config.emailEnabled" class="pp-note">{{ t('auth.forgot.unavailable') }}</p>
    <form v-else class="pp-form" @submit.prevent="submit">
      <PField :label="t('auth.email')">
        <input v-model.trim="email" type="email" autocomplete="email" required />
      </PField>
      <PField :label="t('auth.code')">
        <span class="pp-code-row">
          <input v-model.trim="code" inputmode="numeric" autocomplete="one-time-code" maxlength="6" class="pp-num" required />
          <PButton size="sm" :disabled="left > 0 || !email" :loading="sending" @click="sendCode">
            {{ left > 0 ? t('common.resendIn', { n: left }) : t('common.sendCode') }}
          </PButton>
        </span>
      </PField>
      <PField :label="t('auth.forgot.password')" :hint="t('auth.register.passwordHint', { n: session.config?.minPasswordLength || 8 })">
        <input v-model="password" type="password" autocomplete="new-password" required />
      </PField>
      <p v-if="error" class="pp-error" role="alert">{{ error }}</p>
      <PButton type="submit" variant="primary" size="lg" block :loading="busy">{{ t('auth.forgot.submit') }}</PButton>
    </form>
    <RouterLink to="/login" class="pp-link">{{ t('auth.register.toLogin') }}</RouterLink>
  </AuthLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AuthLayout from '@passport/layouts/AuthLayout.vue'
import PButton from '@passport/components/PButton.vue'
import PField from '@passport/components/PField.vue'
import * as api from '@passport/api/passport'
import { useSessionStore } from '@passport/stores/session'
import { useUiStore } from '@passport/stores/ui'
import { errorText } from '@passport/utils/format'
import { useCountdown } from '@passport/utils/countdown'

const { t } = useI18n()
const router = useRouter()
const session = useSessionStore()
const ui = useUiStore()
const { left, start } = useCountdown()

const email = ref('')
const code = ref('')
const password = ref('')
const busy = ref(false)
const sending = ref(false)
const error = ref('')

onMounted(() => {
  void session.loadConfig().catch(() => undefined)
})

const sendCode = async () => {
  sending.value = true
  error.value = ''
  try {
    const res = await api.sendResetCode(email.value)
    start(res.resendAfterSeconds || 60)
    ui.toast(t('auth.forgot.sent'), 'ok')
  } catch (e) {
    error.value = errorText(e)
  } finally {
    sending.value = false
  }
}

const submit = async () => {
  busy.value = true
  error.value = ''
  try {
    await api.resetPassword(email.value, code.value, password.value)
    ui.toast(t('auth.forgot.done'), 'ok')
    await router.replace('/login')
  } catch (e) {
    error.value = errorText(e)
  } finally {
    busy.value = false
  }
}
</script>
