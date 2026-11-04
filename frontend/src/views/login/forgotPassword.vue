<!-- 找回密码：注册邮箱（尾部获取验证码）→ 验证码 → 新密码（强度条）→ 确认新密码 → 重置；
     重置后该账号所有会话失效。邮件服务未开启时只给出联系管理员的提示。外壳由 login/index.vue 提供。 -->
<template>
  <form class="auth-form" novalidate @submit.prevent="handleReset">
    <div v-if="!mailEnabled" class="auth-note auth-note--warn" role="status">
      <component :is="menuStore.iconComponents['HOutline:ExclamationTriangleIcon']" aria-hidden="true" />
      <span>{{ t('authPage.reset.mailDisabled') }}</span>
    </div>

    <AuthField
      id="forgot-email"
      v-model="form.email"
      :label="t('authPage.field.email')"
      icon="HOutline:EnvelopeIcon"
      type="email"
      inputmode="email"
      :placeholder="t('authPage.field.emailPlaceholder')"
      autocomplete="email"
      :disabled="!mailEnabled"
      :error="errors.email"
      @input="handleEmailInput"
    >
      <template #trail>
        <SendCodeButton
          :countdown="emailCode.countdown.value"
          :sending="emailCode.sending.value"
          :disabled="loading || !mailEnabled"
          @send="handleSendCode"
        />
      </template>
    </AuthField>

    <AuthField
      id="forgot-code"
      v-model="form.code"
      :label="t('authPage.field.code')"
      icon="HOutline:ShieldCheckIcon"
      inputmode="numeric"
      :maxlength="6"
      :placeholder="t('authPage.field.codePlaceholder')"
      autocomplete="one-time-code"
      :disabled="!mailEnabled"
      :error="errors.code"
    >
      <template #below>
        <span v-if="emailCode.sentMinutes.value && !errors.code" class="auth-status" role="status">
          <component :is="menuStore.iconComponents['HOutline:CheckIcon']" aria-hidden="true" />
          <span class="auth-status__long">{{ t('authPage.code.sent', { min: emailCode.sentMinutes.value }) }}</span>
          <span class="auth-status__short">{{ t('authPage.code.sentShort', { min: emailCode.sentMinutes.value }) }}</span>
        </span>
      </template>
    </AuthField>

    <div class="auth-pw">
      <AuthField
        id="forgot-password"
        v-model="form.password"
        v-model:reveal="showPwd"
        :label="t('authPage.field.newPassword')"
        icon="HOutline:LockClosedIcon"
        type="password"
        revealable
        :placeholder="t('authPage.field.newPasswordPlaceholder')"
        autocomplete="new-password"
        :disabled="!mailEnabled"
        :error="errors.password"
      />
      <PasswordMeter
        :password="form.password"
        :hint="t('authPage.strength.hint')"
        :hint-short="t('authPage.strength.hintShort')"
      />
    </div>

    <AuthField
      id="forgot-confirm"
      v-model="form.confirmPassword"
      :reveal="showPwd"
      :label="t('authPage.field.confirmPassword')"
      icon="HOutline:ShieldCheckIcon"
      type="password"
      :placeholder="t('authPage.field.confirmPlaceholder')"
      autocomplete="new-password"
      :disabled="!mailEnabled"
      :error="errors.confirmPassword"
    >
      <template #below>
        <span
          v-if="form.confirmPassword && !errors.confirmPassword"
          class="auth-status"
          :class="{ 'is-bad': !passwordsMatch }"
        >
          <component
            :is="menuStore.iconComponents[passwordsMatch ? 'HOutline:CheckIcon' : 'HOutline:XMarkIcon']"
            aria-hidden="true"
          />
          {{ passwordsMatch ? t('authPage.strength.match') : t('authPage.strength.mismatch') }}
        </span>
      </template>
    </AuthField>

    <!-- 手机：说明放在表单末尾，底部操作条只留按钮 -->
    <div class="auth-note reset-note--m">
      <component :is="menuStore.iconComponents['HOutline:InformationCircleIcon']" aria-hidden="true" />
      <span>{{ t('authPage.reset.noteShort') }}</span>
    </div>

    <div class="auth-actions auth-actions--bar">
      <button type="submit" class="auth-submit" :disabled="loading || !mailEnabled" :aria-busy="loading">
        {{ t('authPage.reset.submit') }}
        <component
          :is="menuStore.iconComponents[loading ? 'HOutline:ArrowPathIcon' : 'HOutline:ArrowRightIcon']"
          :class="{ 'is-spin': loading }"
          aria-hidden="true"
        />
      </button>
      <div class="auth-note reset-note--pc">
        <component :is="menuStore.iconComponents['HOutline:InformationCircleIcon']" aria-hidden="true" />
        <span>{{ t('authPage.reset.note') }}</span>
      </div>
      <p class="auth-switch">
        {{ t('authPage.reset.remembered') }}
        <button type="button" class="auth-link" @click="emits('goToMode', 'login')">
          {{ t('authPage.backToLogin') }}
        </button>
      </p>
    </div>
  </form>
</template>

<script setup lang="ts">
import { resetPassword, sendResetPasswordEmailCode } from '@/api/login'
import { useFeedback } from '@/utils/feedback'
import AuthField from '@/views/login/components/AuthField.vue'
import PasswordMeter from '@/views/login/components/PasswordMeter.vue'
import SendCodeButton from '@/views/login/components/SendCodeButton.vue'
import { MIN_PASSWORD_LENGTH } from '@/views/login/composables/passwordStrength'
import { useEmailCode } from '@/views/login/composables/useEmailCode'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'ForgotPassword' })

withDefaults(defineProps<{ mailEnabled?: boolean }>(), { mailEnabled: true })

const emits = defineEmits<{
  (e: 'goToMode', mode: 'login' | 'forgot' | 'register'): void
}>()

const { t } = useI18n()
const fb = useFeedback()
const menuStore = useMenuStore()

const EMAIL_REGEXP = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

const loading = ref(false)
const showPwd = ref(false)
const errors = ref<Record<string, string>>({})
const form = ref({ email: '', code: '', password: '', confirmPassword: '' })

const emailCode = useEmailCode((email) => sendResetPasswordEmailCode({ email }))

const passwordsMatch = computed(
  () => !!form.value.confirmPassword && form.value.confirmPassword === form.value.password,
)

const handleEmailInput = () => {
  form.value.code = ''
  emailCode.reset()
}

const handleSendCode = async () => {
  const email = form.value.email.trim()
  if (!EMAIL_REGEXP.test(email)) {
    errors.value = { ...errors.value, email: email ? t('login.emailInvalid') : t('login.validEmailFirst') }
    return
  }
  errors.value = { ...errors.value, email: '' }
  await emailCode.request(email)
}

const validate = (): boolean => {
  const next: Record<string, string> = {}
  const email = form.value.email.trim()
  const code = form.value.code.trim()
  if (!email) next.email = t('forgot.emailPlaceholder')
  else if (!EMAIL_REGEXP.test(email)) next.email = t('login.emailInvalid')
  if (!code) next.code = t('register.emailCodeRequired')
  else if (!/^\d{6}$/.test(code)) next.code = t('authPage.code.invalid')
  if (!form.value.password) next.password = t('register.passwordRequired')
  else if (form.value.password.length < MIN_PASSWORD_LENGTH) next.password = t('authPage.strength.tooShort')
  if (!form.value.confirmPassword) next.confirmPassword = t('register.confirmPasswordRequired')
  else if (form.value.confirmPassword !== form.value.password) {
    next.confirmPassword = t('register.confirmPasswordMismatch')
  }
  errors.value = next
  return Object.keys(next).length === 0
}

const handleReset = async () => {
  if (!validate()) return
  loading.value = true
  try {
    await resetPassword({
      email: form.value.email.trim(),
      code: form.value.code.trim(),
      newPassword: form.value.password,
    })
    // 本机若还留着该账号的旧令牌，已随重置失效，一并清掉。
    localStorage.removeItem('accessToken')
    localStorage.removeItem('refreshToken')
    fb.success(t('authPage.reset.success'))
    emits('goToMode', 'login')
  } catch {
    // 请求层已提示错误
  } finally {
    loading.value = false
  }
}
</script>

<style scoped lang="scss">
@use './auth-form.scss';

.reset-note--m {
  display: none;
}
@media (width < 900px) {
  .reset-note--m {
    display: flex;
  }
  .reset-note--pc {
    display: none;
  }
}
</style>
