<!-- 注册：前置图标输入框 + 密码强度条 + 两次输入一致提示；外壳与样式基调由 login/index.vue 提供。 -->
<template>
  <div class="auth-form">
    <header class="auth-form__head">
      <h1 class="auth-form__title">{{ t('register.title') }}</h1>
      <p class="auth-form__sub">{{ t('register.subtitle', { name: APP_CONFIG.name }) }}</p>
    </header>

    <form class="auth-form__body" @submit.prevent="handleRegister">
      <div class="app-field">
        <label class="app-label" for="reg-username">{{ t('user.username') }}</label>
        <div class="auth-fld">
          <component :is="menuStore.iconComponents['HOutline:UserIcon']" class="auth-fld__ico" />
          <input
            id="reg-username"
            v-model="registerForm.username"
            class="app-input auth-input auth-input--lead"
            :class="{ 'is-error': errors.username }"
            :placeholder="t('register.usernamePlaceholder')"
            autocomplete="username"
          />
        </div>
        <span v-if="errors.username" class="app-field__error">{{ errors.username }}</span>
      </div>

      <div class="app-field">
        <label class="app-label" for="reg-email">{{ t('user.email') }}</label>
        <div class="auth-fld">
          <component :is="menuStore.iconComponents['HOutline:EnvelopeIcon']" class="auth-fld__ico" />
          <input
            id="reg-email"
            v-model="registerForm.email"
            class="app-input auth-input auth-input--lead"
            :class="{ 'is-error': errors.email }"
            type="email"
            :placeholder="t('register.emailPlaceholder')"
            autocomplete="email"
            @input="handleEmailInput"
          />
        </div>
        <span v-if="errors.email" class="app-field__error">{{ errors.email }}</span>
      </div>

      <div v-if="props.registerEmailVerificationRequired" class="app-field">
        <label class="app-label" for="reg-code">{{ t('register.codeLabel') }}</label>
        <div class="auth-code-row">
          <div class="auth-fld">
            <component
              :is="menuStore.iconComponents['HOutline:ShieldCheckIcon']"
              class="auth-fld__ico"
            />
            <input
              id="reg-code"
              v-model="registerForm.code"
              class="app-input auth-input auth-input--lead"
              :class="{ 'is-error': errors.code }"
              :placeholder="t('register.codePlaceholder')"
              maxlength="6"
              inputmode="numeric"
              autocomplete="one-time-code"
            />
          </div>
          <UButton
            type="button"
            color="primary"
            variant="soft"
            class="auth-code-btn"
            :disabled="sendCodeDisabled"
            @click="handleSendCode"
          >
            {{ sendCodeButtonText }}
          </UButton>
        </div>
        <span v-if="errors.code" class="app-field__error">{{ errors.code }}</span>
        <span v-else-if="sendCodeCountdown > 0" class="auth-hint auth-hint--ok" role="status">
          <component :is="menuStore.iconComponents['HOutline:CheckIcon']" class="auth-hint__ico" />
          {{ t('login.codeSent') }}
        </span>
      </div>

      <div class="app-field">
        <label class="app-label" for="reg-password">{{ t('login.passwordLabel') }}</label>
        <div class="auth-input-wrap auth-fld">
          <component :is="menuStore.iconComponents['HOutline:LockClosedIcon']" class="auth-fld__ico" />
          <input
            id="reg-password"
            v-model="registerForm.password"
            class="app-input auth-input auth-input--lead auth-input--with-icon"
            :class="{ 'is-error': errors.password }"
            :type="showPwd ? 'text' : 'password'"
            :placeholder="t('register.passwordPlaceholder')"
            autocomplete="new-password"
          />
          <button
            type="button"
            class="auth-eye"
            :aria-label="showPwd ? t('login.hidePassword') : t('login.showPassword')"
            @click="showPwd = !showPwd"
          >
            <component
              :is="
                menuStore.iconComponents[showPwd ? 'HOutline:EyeSlashIcon' : 'HOutline:EyeIcon']
              "
              class="auth-eye__icon"
            />
          </button>
        </div>
        <span v-if="errors.password" class="app-field__error">{{ errors.password }}</span>
        <div v-if="registerForm.password" class="auth-meter" aria-live="polite">
          <span class="auth-meter__bars" aria-hidden="true">
            <span
              v-for="n in 4"
              :key="n"
              class="auth-meter__bar"
              :class="n <= strength.score ? `is-${strength.tone}` : ''"
            />
          </span>
          <span class="auth-meter__label" :class="`is-${strength.tone}`">
            {{ t('register.strength', { level: strength.label }) }}
          </span>
        </div>
        <span class="auth-hint">{{ t('register.passwordTip') }}</span>
      </div>

      <div class="app-field">
        <label class="app-label" for="reg-confirm">{{ t('register.confirmPasswordLabel') }}</label>
        <div class="auth-input-wrap auth-fld">
          <component :is="menuStore.iconComponents['HOutline:ShieldCheckIcon']" class="auth-fld__ico" />
          <input
            id="reg-confirm"
            v-model="registerForm.confirmPassword"
            class="app-input auth-input auth-input--lead auth-input--with-icon"
            :class="{ 'is-error': errors.confirmPassword }"
            :type="showConfirmPwd ? 'text' : 'password'"
            :placeholder="t('register.confirmPasswordPlaceholder')"
            autocomplete="new-password"
          />
          <button
            type="button"
            class="auth-eye"
            :aria-label="showConfirmPwd ? t('login.hidePassword') : t('login.showPassword')"
            @click="showConfirmPwd = !showConfirmPwd"
          >
            <component
              :is="
                menuStore.iconComponents[
                  showConfirmPwd ? 'HOutline:EyeSlashIcon' : 'HOutline:EyeIcon'
                ]
              "
              class="auth-eye__icon"
            />
          </button>
        </div>
        <span v-if="errors.confirmPassword" class="app-field__error">{{
          errors.confirmPassword
        }}</span>
        <span
          v-else-if="registerForm.confirmPassword"
          class="auth-hint"
          :class="passwordsMatch ? 'auth-hint--ok' : 'auth-hint--bad'"
        >
          <component
            :is="menuStore.iconComponents[passwordsMatch ? 'HOutline:CheckIcon' : 'HOutline:XMarkIcon']"
            class="auth-hint__ico"
          />
          {{ passwordsMatch ? t('register.passwordMatch') : t('register.confirmPasswordMismatch') }}
        </span>
      </div>

      <UButton
        type="submit"
        color="primary"
        block
        class="auth-submit"
        trailing-icon="i-lucide-arrow-right"
        :loading="loading"
      >
        {{ t('register.submit') }}
      </UButton>
    </form>

    <p class="auth-switch">
      <span>{{ t('register.haveAccount') }}</span>
      <button type="button" class="auth-text-btn" @click="emits('goToMode', 'login')">
        {{ t('register.backLogin') }}
      </button>
    </p>
  </div>
</template>

<script setup lang="ts">
import { APP_CONFIG } from '@/config/app.config'
import { register, sendRegisterEmailCode } from '@/api/login'
import { useFeedback } from '@/utils/feedback'
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

const loading = ref(false)
const showPwd = ref(false)
const showConfirmPwd = ref(false)
const errors = ref<Record<string, string>>({})

const EMAIL_REGEXP = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
const SEND_CODE_COUNTDOWN_SECONDS = 60

const registerForm = ref({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  code: '',
})

const sendCodeCountdown = ref(0)
let sendCodeTimer: ReturnType<typeof setInterval> | null = null

const isValidEmail = (email: string) => EMAIL_REGEXP.test(email)

// 密码强度：仅作提示，不参与校验（后端规则仍以 register.passwordMin 为准）。
const strength = computed(() => {
  const pw = registerForm.value.password
  const rules = [
    pw.length >= 8,
    /[a-z]/.test(pw) && /[A-Z]/.test(pw),
    /\d/.test(pw),
    /[^A-Za-z0-9]/.test(pw),
  ]
  let score = rules.filter(Boolean).length
  if (pw.length < 6) score = Math.min(score, 1)
  const levels = ['weak', 'weak', 'fair', 'strong', 'strong'] as const
  const tone = levels[score] ?? 'weak'
  const levelKey = ['veryWeak', 'weak', 'fair', 'strong', 'veryStrong'][score] ?? 'weak'
  const label = t(`register.strengthLevels.${levelKey}`)
  return { score: Math.max(score, 1), tone, label }
})

const passwordsMatch = computed(
  () =>
    !!registerForm.value.confirmPassword &&
    registerForm.value.confirmPassword === registerForm.value.password,
)

const sendCodeButtonText = computed(() =>
  sendCodeCountdown.value > 0
    ? t('login.resendIn', { seconds: sendCodeCountdown.value })
    : t('login.sendCode'),
)

const sendCodeDisabled = computed(
  () =>
    loading.value || sendCodeCountdown.value > 0 || !isValidEmail(registerForm.value.email.trim()),
)

const clearSendCodeTimer = () => {
  if (sendCodeTimer) {
    clearInterval(sendCodeTimer)
    sendCodeTimer = null
  }
}

const startSendCodeCountdown = () => {
  sendCodeCountdown.value = SEND_CODE_COUNTDOWN_SECONDS
  clearSendCodeTimer()
  sendCodeTimer = setInterval(() => {
    if (sendCodeCountdown.value <= 1) {
      sendCodeCountdown.value = 0
      clearSendCodeTimer()
      return
    }
    sendCodeCountdown.value -= 1
  }, 1000)
}

const handleEmailInput = () => {
  if (!props.registerEmailVerificationRequired) return
  registerForm.value.code = ''
}

const handleSendCode = async () => {
  const email = registerForm.value.email.trim()
  if (!isValidEmail(email)) {
    fb.warning(t('login.validEmailFirst'))
    return
  }
  try {
    await sendRegisterEmailCode({ email })
    fb.success(t('login.codeSent'))
    startSendCodeCountdown()
  } catch {
    fb.error(t('login.codeSendFailed'))
  }
}

const validate = (): boolean => {
  const next: Record<string, string> = {}
  const username = registerForm.value.username.trim()
  const email = registerForm.value.email.trim()
  const password = registerForm.value.password
  const confirmPassword = registerForm.value.confirmPassword
  const code = registerForm.value.code.trim()

  if (!username) next.username = t('register.usernameRequired')
  if (!email) next.email = t('register.emailRequired')
  else if (!EMAIL_REGEXP.test(email)) next.email = t('login.emailInvalid')

  if (props.registerEmailVerificationRequired && !code) {
    next.code = t('register.emailCodeRequired')
  }

  if (!password) next.password = t('register.passwordRequired')
  else if (password.length < 6) next.password = t('register.passwordMin')

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
  } finally {
    loading.value = false
  }
}

onBeforeUnmount(() => {
  clearSendCodeTimer()
})
</script>

<style scoped lang="scss">
.auth-form {
  width: 100%;
}

.auth-form__head {
  margin-bottom: 0.9rem;
}

.auth-form__title {
  margin: 0 0 4px;
  font-size: 1.25rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--el-text-color-primary);
}

.auth-form__sub {
  margin: 0;
  font-size: 0.8125rem;
  line-height: 1.45;
  color: var(--el-text-color-secondary);
}

.auth-form__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.auth-input {
  height: 36px;
  font-size: 0.8125rem;
}

/* 带前置图标的输入框：图标绝对定位，输入左内边距让位（特异性高于外壳的 :deep 覆盖） */
.auth-fld {
  position: relative;
  flex: 1;
  min-width: 0;
}
.auth-fld__ico {
  position: absolute;
  left: 16px;
  top: 50%;
  width: 20px;
  height: 20px;
  transform: translateY(-50%);
  color: var(--el-text-color-secondary);
  pointer-events: none;
  z-index: 1;
}
.auth-fld .auth-input.auth-input--lead {
  padding-left: 46px;
  background: var(--m-s2);
  border-color: transparent;
}
.auth-fld .auth-input.auth-input--lead:focus {
  background: var(--el-bg-color);
  border-color: var(--el-color-primary);
}
.auth-fld .auth-input.auth-input--lead.is-error {
  border-color: var(--el-color-danger);
}
.auth-fld:focus-within .auth-fld__ico {
  color: var(--m-pInk);
}

.auth-input-wrap {
  position: relative;
}

.auth-input--with-icon {
  padding-right: 44px;
}

.auth-eye {
  position: absolute;
  top: 50%;
  right: 4px;
  transform: translateY(-50%);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: none;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--el-text-color-placeholder);
  cursor: pointer;
  padding: 0;
  z-index: 1;
}

.auth-eye:hover {
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-light);
}

.auth-eye__icon {
  width: 16px;
  height: 16px;
}

.auth-code-row {
  display: flex;
  align-items: stretch;
  gap: 8px;
}

.auth-code-btn {
  flex-shrink: 0;
  min-width: 6.5rem;
  height: 36px !important;
  border-radius: 999px;
  justify-content: center;
  font-weight: 700;
}

/* 强度条 */
.auth-meter {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 4px;
}
.auth-meter__bars {
  display: flex;
  flex: 1;
  gap: 6px;
}
.auth-meter__bar {
  flex: 1;
  height: 6px;
  border-radius: 999px;
  background: var(--m-line);
  transition: background 0.2s;
}
.auth-meter__bar.is-weak {
  background: var(--m-bad);
}
.auth-meter__bar.is-fair {
  background: var(--m-warn);
}
.auth-meter__bar.is-strong {
  background: var(--m-ok);
}
.auth-meter__label {
  flex: none;
  font-size: 0.8125rem;
  font-weight: 800;
}
.auth-meter__label.is-weak {
  color: var(--m-bad);
}
.auth-meter__label.is-fair {
  color: var(--m-warn);
}
.auth-meter__label.is-strong {
  color: var(--m-ok);
}

.auth-hint {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.75rem;
  line-height: 1.4;
  color: var(--el-text-color-secondary);
}
.auth-hint--ok {
  font-weight: 600;
  color: var(--m-ok);
}
.auth-hint--bad {
  font-weight: 600;
  color: var(--m-bad);
}
.auth-hint__ico {
  width: 14px;
  height: 14px;
  flex: none;
  stroke-width: 2.4;
}

.auth-submit {
  margin-top: 4px;
  height: 36px !important;
  justify-content: center;
  font-size: 0.875rem !important;
  font-weight: 600;
}

.auth-switch {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 0.35rem;
  margin: 12px 0 0;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}

.auth-text-btn {
  border: none;
  background: transparent;
  padding: 0;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-color-primary);
  cursor: pointer;
}

.auth-text-btn:hover {
  text-decoration: underline;
  text-underline-offset: 2px;
}
</style>
