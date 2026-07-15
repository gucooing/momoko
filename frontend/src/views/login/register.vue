<!-- 注册：紧凑令牌表单，密码眼睛内嵌。 -->
<template>
  <div class="auth-form">
    <header class="auth-form__head">
      <h1 class="auth-form__title">{{ t('register.title') }}</h1>
      <p class="auth-form__sub">{{ t('register.subtitle', { name: APP_CONFIG.name }) }}</p>
    </header>

    <form class="auth-form__body" @submit.prevent="handleRegister">
      <div class="app-field">
        <label class="app-label" for="reg-username">{{ t('register.usernamePlaceholder') }}</label>
        <input
          id="reg-username"
          v-model="registerForm.username"
          class="app-input auth-input"
          :class="{ 'is-error': errors.username }"
          :placeholder="t('register.usernamePlaceholder')"
          autocomplete="username"
        />
        <span v-if="errors.username" class="app-field__error">{{ errors.username }}</span>
      </div>

      <div class="app-field">
        <label class="app-label" for="reg-email">{{ t('register.emailPlaceholder') }}</label>
        <input
          id="reg-email"
          v-model="registerForm.email"
          class="app-input auth-input"
          :class="{ 'is-error': errors.email }"
          :placeholder="t('register.emailPlaceholder')"
          autocomplete="email"
          @input="handleEmailInput"
        />
        <span v-if="errors.email" class="app-field__error">{{ errors.email }}</span>
      </div>

      <div v-if="props.registerEmailVerificationRequired" class="app-field">
        <label class="app-label" for="reg-code">{{ t('register.codePlaceholder') }}</label>
        <div class="auth-code-row">
          <input
            id="reg-code"
            v-model="registerForm.code"
            class="app-input auth-input"
            :class="{ 'is-error': errors.code }"
            :placeholder="t('register.codePlaceholder')"
            maxlength="6"
            inputmode="numeric"
            autocomplete="one-time-code"
          />
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
      </div>

      <div class="app-field">
        <label class="app-label" for="reg-password">{{ t('register.passwordPlaceholder') }}</label>
        <div class="auth-input-wrap">
          <input
            id="reg-password"
            v-model="registerForm.password"
            class="app-input auth-input auth-input--with-icon"
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
      </div>

      <div class="app-field">
        <label class="app-label" for="reg-confirm">{{
          t('register.confirmPasswordPlaceholder')
        }}</label>
        <div class="auth-input-wrap">
          <input
            id="reg-confirm"
            v-model="registerForm.confirmPassword"
            class="app-input auth-input auth-input--with-icon"
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
      </div>

      <UButton type="submit" color="primary" block class="auth-submit" :loading="loading">
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

.auth-input-wrap {
  position: relative;
}

.auth-input--with-icon {
  padding-right: 36px;
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

.auth-code-row .app-input {
  flex: 1;
  min-width: 0;
}

.auth-code-btn {
  flex-shrink: 0;
  min-width: 6.5rem;
  height: 36px !important;
}

.auth-submit {
  margin-top: 4px;
  height: 36px !important;
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
  font-weight: 600;
  color: var(--el-color-primary);
  cursor: pointer;
}

.auth-text-btn:hover {
  text-decoration: underline;
  text-underline-offset: 2px;
}
</style>
