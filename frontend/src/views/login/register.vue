<!-- 注册：用户名 / 邮箱（尾部获取验证码）/ 验证码 / 密码（强度条）/ 确认密码；外壳与分段切换由 login/index.vue 提供。 -->
<template>
  <form class="auth-form" novalidate @submit.prevent="handleRegister">
    <AuthField
      id="reg-username"
      v-model="registerForm.username"
      :label="t('authPage.field.username')"
      icon="HOutline:UserIcon"
      :placeholder="t('authPage.field.usernamePlaceholder')"
      autocomplete="username"
      :maxlength="20"
      :error="errors.username"
    />

    <AuthField
      id="reg-email"
      v-model="registerForm.email"
      :label="t('authPage.field.email')"
      icon="HOutline:EnvelopeIcon"
      type="email"
      inputmode="email"
      :placeholder="t('authPage.field.emailPlaceholder')"
      autocomplete="email"
      :error="errors.email"
      @input="handleEmailInput"
    >
      <template v-if="props.registerEmailVerificationRequired" #trail>
        <SendCodeButton
          :countdown="emailCode.countdown.value"
          :sending="emailCode.sending.value"
          :disabled="loading"
          @send="handleSendCode"
        />
      </template>
    </AuthField>

    <AuthField
      v-if="props.registerEmailVerificationRequired"
      id="reg-code"
      v-model="registerForm.code"
      :label="t('authPage.field.code')"
      icon="HOutline:ShieldCheckIcon"
      inputmode="numeric"
      :maxlength="6"
      :placeholder="t('authPage.field.codePlaceholder')"
      autocomplete="one-time-code"
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
        id="reg-password"
        v-model="registerForm.password"
        v-model:reveal="showPwd"
        :label="t('authPage.field.password')"
        icon="HOutline:LockClosedIcon"
        type="password"
        revealable
        :placeholder="t('authPage.field.newPasswordPlaceholder')"
        autocomplete="new-password"
        :error="errors.password"
      />
      <PasswordMeter
        :password="registerForm.password"
        :hint="t('authPage.strength.hint')"
        :hint-short="t('authPage.strength.hintShort')"
      />
    </div>

    <AuthField
      id="reg-confirm"
      v-model="registerForm.confirmPassword"
      :reveal="showPwd"
      :label="t('authPage.field.confirmPassword')"
      icon="HOutline:ShieldCheckIcon"
      type="password"
      :placeholder="t('authPage.field.confirmPlaceholder')"
      autocomplete="new-password"
      :error="errors.confirmPassword"
    >
      <template #below>
        <span
          v-if="registerForm.confirmPassword && !errors.confirmPassword"
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

    <div class="auth-actions auth-actions--bar">
      <button type="submit" class="auth-submit" :disabled="loading" :aria-busy="loading">
        {{ t('authPage.register.submit') }}
        <component
          :is="menuStore.iconComponents[loading ? 'HOutline:ArrowPathIcon' : 'HOutline:ArrowRightIcon']"
          :class="{ 'is-spin': loading }"
          aria-hidden="true"
        />
      </button>
      <p class="auth-switch">
        {{ t('authPage.register.haveAccount') }}
        <button type="button" class="auth-link" @click="emits('goToMode', 'login')">
          {{ t('authPage.register.toLogin') }}
        </button>
      </p>
    </div>
  </form>
</template>

<script setup lang="ts">
import { register, sendRegisterEmailCode } from '@/api/login'
import { useFeedback } from '@/utils/feedback'
import AuthField from '@/views/login/components/AuthField.vue'
import PasswordMeter from '@/views/login/components/PasswordMeter.vue'
import SendCodeButton from '@/views/login/components/SendCodeButton.vue'
import { MIN_PASSWORD_LENGTH } from '@/views/login/composables/passwordStrength'
import { useEmailCode } from '@/views/login/composables/useEmailCode'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'RegisterComponent' })

const props = defineProps<{
  registerEmailVerificationRequired: boolean
}>()

const emits = defineEmits<{
  (e: 'goToMode', mode: 'login' | 'forgot' | 'register'): void
}>()

const { t } = useI18n()
const fb = useFeedback()
const menuStore = useMenuStore()

const EMAIL_REGEXP = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
/** 与后端 registerUsernamePattern 一致 */
const USERNAME_REGEXP = /^[A-Za-z0-9_]{3,20}$/

const loading = ref(false)
const showPwd = ref(false)
const errors = ref<Record<string, string>>({})

const registerForm = ref({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  code: '',
})

const emailCode = useEmailCode((email) => sendRegisterEmailCode({ email }))

const passwordsMatch = computed(
  () => !!registerForm.value.confirmPassword && registerForm.value.confirmPassword === registerForm.value.password,
)

const isValidEmail = (email: string) => EMAIL_REGEXP.test(email)

const handleEmailInput = () => {
  if (!props.registerEmailVerificationRequired) return
  registerForm.value.code = ''
  emailCode.reset()
}

const handleSendCode = async () => {
  const email = registerForm.value.email.trim()
  if (!isValidEmail(email)) {
    errors.value = { ...errors.value, email: email ? t('login.emailInvalid') : t('login.validEmailFirst') }
    return
  }
  errors.value = { ...errors.value, email: '' }
  await emailCode.request(email)
}

const validate = (): boolean => {
  const next: Record<string, string> = {}
  const username = registerForm.value.username.trim()
  const email = registerForm.value.email.trim()
  const password = registerForm.value.password
  const confirmPassword = registerForm.value.confirmPassword
  const code = registerForm.value.code.trim()

  if (!username) next.username = t('register.usernameRequired')
  else if (!USERNAME_REGEXP.test(username)) next.username = t('authPage.register.usernameInvalid')
  if (!email) next.email = t('register.emailRequired')
  else if (!EMAIL_REGEXP.test(email)) next.email = t('login.emailInvalid')

  if (props.registerEmailVerificationRequired) {
    if (!code) next.code = t('register.emailCodeRequired')
    else if (!/^\d{6}$/.test(code)) next.code = t('authPage.code.invalid')
  }

  if (!password) next.password = t('register.passwordRequired')
  else if (password.length < MIN_PASSWORD_LENGTH) next.password = t('authPage.strength.tooShort')

  if (!confirmPassword) next.confirmPassword = t('register.confirmPasswordRequired')
  else if (confirmPassword !== password) next.confirmPassword = t('register.confirmPasswordMismatch')

  errors.value = next
  return Object.keys(next).length === 0
}

const handleRegister = async () => {
  if (!validate()) return
  loading.value = true
  try {
    await register({
      username: registerForm.value.username.trim(),
      email: registerForm.value.email.trim(),
      password: registerForm.value.password,
      code: registerForm.value.code.trim(),
    })
    fb.success(t('register.success'))
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
</style>
