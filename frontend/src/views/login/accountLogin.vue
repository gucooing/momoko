<!-- 账号登录：紧凑令牌表单；密码眼睛内嵌输入框。 -->
<template>
  <div class="auth-form">
    <header class="auth-form__head">
      <h1 class="auth-form__title">{{ t('login.welcomeBack') }}</h1>
      <p class="auth-form__sub">{{ t('login.subtitle') }}</p>
    </header>

    <form class="auth-form__body" @submit.prevent="handleLogin">
      <div
        v-if="showLoginTypeSwitch"
        class="auth-seg"
        :class="`auth-seg--${loginType}`"
        role="tablist"
      >
        <span class="auth-seg__indicator" aria-hidden="true" />
        <button
          type="button"
          role="tab"
          class="auth-seg__item"
          :class="{ 'is-active': loginType === 'username' }"
          :aria-selected="loginType === 'username'"
          @click="setLoginType('username')"
        >
          {{ t('login.usernameLogin') }}
        </button>
        <button
          type="button"
          role="tab"
          class="auth-seg__item"
          :class="{ 'is-active': loginType === 'email' }"
          :aria-selected="loginType === 'email'"
          @click="setLoginType('email')"
        >
          {{ t('login.emailLogin') }}
        </button>
      </div>

      <div class="app-field">
        <label class="app-label" for="login-account">
          {{ isEmailMode ? t('login.emailLogin') : t('login.usernameLogin') }}
        </label>
        <input
          id="login-account"
          v-model="loginForm.username"
          class="app-input auth-input"
          :class="{ 'is-error': errors.username }"
          :placeholder="loginInputPlaceholder"
          autocomplete="username"
          @input="handleAccountInput"
        />
        <span v-if="errors.username" class="app-field__error">{{ errors.username }}</span>
      </div>

      <Transition name="login-fade" mode="out-in">
        <div v-if="loginType === 'username'" key="password-mode" class="app-field">
          <label class="app-label" for="login-password">{{ t('login.passwordLabel') }}</label>
          <div class="auth-input-wrap">
            <input
              id="login-password"
              v-model="loginForm.password"
              class="app-input auth-input auth-input--with-icon"
              :class="{ 'is-error': errors.password }"
              :type="showPwd ? 'text' : 'password'"
              :placeholder="t('login.passwordPlaceholder')"
              autocomplete="current-password"
            />
            <button
              type="button"
              class="auth-eye"
              :aria-label="showPwd ? t('login.hidePassword') : t('login.showPassword')"
              @click="showPwd = !showPwd"
            >
              <component
                :is="
                  menuStore.iconComponents[
                    showPwd ? 'HOutline:EyeSlashIcon' : 'HOutline:EyeIcon'
                  ]
                "
                class="auth-eye__icon"
              />
            </button>
          </div>
          <span v-if="errors.password" class="app-field__error">{{ errors.password }}</span>
        </div>

        <div v-else key="code-mode" class="app-field">
          <label class="app-label" for="login-code">{{ t('login.codePlaceholder') }}</label>
          <div class="auth-code-row">
            <input
              id="login-code"
              v-model="loginForm.password"
              class="app-input auth-input"
              :class="{ 'is-error': errors.password }"
              :placeholder="t('login.codePlaceholder')"
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
          <span v-if="errors.password" class="app-field__error">{{ errors.password }}</span>
        </div>
      </Transition>

      <div class="auth-options">
        <label class="auth-check">
          <input
            v-model="loginForm.remember"
            type="checkbox"
            @change="handleRememberChange(loginForm.remember)"
          />
          <span>{{ t('login.rememberMe') }}</span>
        </label>
        <button type="button" class="auth-text-btn" @click="emits('goToMode', 'forgot')">
          {{ t('login.forgotPassword') }}
        </button>
      </div>

      <UButton
        type="submit"
        color="primary"
        block
        class="auth-submit"
        :loading="loading"
      >
        {{ t('login.submit') }}
      </UButton>
    </form>

    <p v-if="registerEnabled" class="auth-switch">
      <span>{{ t('login.noAccount') }}</span>
      <button type="button" class="auth-text-btn" @click="emits('goToMode', 'register')">
        {{ t('login.registerNow') }}
      </button>
    </p>
  </div>
</template>

<script setup lang="ts">
import platform from 'platform'
import { login, sendLoginEmailCode } from '@/api/login'
import type { LoginRequest } from '@/types/v1/auth'
import { normalizeAuthRedirect } from '@/utils/authRedirect'
import { normalizeAuthToken } from '@/utils/request'
import { getDeviceId } from '@/utils/deviceId'
import { useFeedback } from '@/utils/feedback'
import { useI18n } from 'vue-i18n'

interface IEmits {
  (e: 'goToMode', mode: 'login' | 'forgot' | 'register'): void
}

type LoginType = 'username' | 'email'

interface ILoginFormModel {
  username: string
  password: string
  remember: boolean
}

const EMAIL_REGEXP = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
const VERIFY_CODE_REGEXP = /^\d{4,6}$/
const SEND_CODE_COUNTDOWN_SECONDS = 60
const REMEMBER_USERNAME_KEY = 'remember_username'

defineOptions({ name: 'AccountLogin' })

const props = defineProps<{
  usernameLoginEnabled: boolean
  emailLoginEnabled: boolean
  registerEnabled: boolean
}>()

const emits = defineEmits<IEmits>()

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const fb = useFeedback()
const menuStore = useMenuStore()

const loading = ref(false)
const showPwd = ref(false)
const errors = ref<Record<string, string>>({})

const loginForm = ref<ILoginFormModel>({
  username: '',
  password: '',
  remember: false,
})

const showLoginTypeSwitch = computed(() => props.usernameLoginEnabled && props.emailLoginEnabled)

const resolveDefaultLoginType = (): LoginType => {
  if (props.usernameLoginEnabled) return 'username'
  if (props.emailLoginEnabled) return 'email'
  return 'username'
}

const loginType = ref<LoginType>(resolveDefaultLoginType())
const isEmailMode = computed(() => loginType.value === 'email')
const accountValue = computed(() => loginForm.value.username.trim())
const loginInputPlaceholder = computed(() =>
  isEmailMode.value ? t('login.emailPlaceholder') : t('login.usernamePlaceholder'),
)

const sendCodeCountdown = ref(0)
let sendCodeTimer: ReturnType<typeof setInterval> | null = null

const resetCredentialInput = () => {
  loginForm.value.password = ''
  errors.value = {}
}

const setLoginType = (type: LoginType) => {
  if (loginType.value === type) return
  loginType.value = type
  resetCredentialInput()
}

const handleAccountInput = (e: Event) => {
  const value = (e.target as HTMLInputElement).value
  if (value.includes('@') && !isEmailMode.value && props.emailLoginEnabled) {
    loginType.value = 'email'
    resetCredentialInput()
  }
}

const loadRememberedUsername = () => {
  const rememberedUsername = localStorage.getItem(REMEMBER_USERNAME_KEY)
  if (rememberedUsername) {
    loginForm.value.username = rememberedUsername
    if (rememberedUsername.includes('@') && !isEmailMode.value && props.emailLoginEnabled) {
      loginType.value = 'email'
    }
    loginForm.value.remember = true
  }
}

const handleRememberChange = (value: boolean) => {
  const account = accountValue.value
  if (value) {
    if (account) localStorage.setItem(REMEMBER_USERNAME_KEY, account)
  } else {
    localStorage.removeItem(REMEMBER_USERNAME_KEY)
  }
}

const isValidEmail = (email: string) => EMAIL_REGEXP.test(email)

const sendCodeButtonText = computed(() =>
  sendCodeCountdown.value > 0
    ? t('login.resendIn', { seconds: sendCodeCountdown.value })
    : t('login.sendCode'),
)

const sendCodeDisabled = computed(
  () => loading.value || sendCodeCountdown.value > 0 || !isValidEmail(accountValue.value),
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

const handleSendCode = async () => {
  const email = accountValue.value
  if (!isValidEmail(email)) {
    fb.warning(t('login.validEmailFirst'))
    return
  }
  try {
    await sendLoginEmailCode({ email })
    fb.success(t('login.codeSent'))
    startSendCodeCountdown()
  } catch {
    fb.error(t('login.codeSendFailed'))
  }
}

const buildLoginPayload = (): LoginRequest => {
  const account = accountValue.value
  const unknownDevice = t('login.unknownDevice')
  const device =
    `${platform.os?.toString() || unknownDevice} / ${platform.name || unknownDevice} ${platform.version || ''}`.trim()
  const deviceId = getDeviceId()
  return isEmailMode.value
    ? { email: account, password: '', device, deviceId, code: loginForm.value.password }
    : { username: account, password: loginForm.value.password, device, deviceId, code: '' }
}

const validate = (): boolean => {
  const next: Record<string, string> = {}
  const account = accountValue.value
  if (!account) {
    next.username = isEmailMode.value ? t('login.emailPlaceholder') : t('login.usernamePlaceholder')
  } else if (isEmailMode.value && !EMAIL_REGEXP.test(account)) {
    next.username = t('login.emailInvalid')
  }

  const credential = loginForm.value.password.trim()
  if (!credential) {
    next.password = isEmailMode.value ? t('login.codePlaceholder') : t('login.passwordPlaceholder')
  } else if (isEmailMode.value && !VERIFY_CODE_REGEXP.test(credential)) {
    next.password = t('login.codeInvalid')
  }

  errors.value = next
  return Object.keys(next).length === 0
}

const handleLogin = async () => {
  if (!validate()) return
  loading.value = true
  try {
    const payload = buildLoginPayload()
    const { data: loginRes } = await login(payload)

    if (!loginRes?.accessToken) {
      fb.error(t('login.missingAccessToken'))
      return
    }

    localStorage.setItem('accessToken', normalizeAuthToken(loginRes.accessToken))
    if (loginRes.refreshToken) {
      localStorage.setItem('refreshToken', normalizeAuthToken(loginRes.refreshToken))
    } else {
      localStorage.removeItem('refreshToken')
    }

    if (loginForm.value.remember) {
      localStorage.setItem(REMEMBER_USERNAME_KEY, accountValue.value)
    } else {
      localStorage.removeItem(REMEMBER_USERNAME_KEY)
    }

    fb.success(t('login.success'))
    const redirectQuery = Array.isArray(route.query.redirect)
      ? route.query.redirect[0]
      : route.query.redirect
    const redirectPath = normalizeAuthRedirect(redirectQuery)
    await router.push(redirectPath || '/')
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.usernameLoginEnabled, props.emailLoginEnabled],
  () => {
    if (loginType.value === 'username' && !props.usernameLoginEnabled) {
      loginType.value = resolveDefaultLoginType()
      resetCredentialInput()
    } else if (loginType.value === 'email' && !props.emailLoginEnabled) {
      loginType.value = resolveDefaultLoginType()
      resetCredentialInput()
    }
  },
)

onMounted(() => {
  loadRememberedUsername()
})

onBeforeUnmount(() => {
  clearSendCodeTimer()
})
</script>

<style scoped lang="scss">
.auth-form {
  width: 100%;
}

.auth-form__head {
  margin-bottom: 1rem;
}

.auth-form__title {
  margin: 0 0 4px;
  font-size: 1.25rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1.3;
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
  gap: 12px;
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

.auth-seg {
  position: relative;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  padding: 3px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
  overflow: hidden;
}

.auth-seg__indicator {
  position: absolute;
  top: 3px;
  left: 3px;
  width: calc(50% - 3px);
  height: calc(100% - 6px);
  border-radius: calc(var(--app-radius-sm) - 2px);
  background: var(--el-bg-color);
  box-shadow: inset 0 0 0 1px var(--app-hairline);
  transition: transform 0.22s cubic-bezier(0.22, 1, 0.36, 1);
}

.auth-seg--email .auth-seg__indicator {
  transform: translateX(100%);
}

.auth-seg__item {
  position: relative;
  z-index: 1;
  height: 28px;
  border: none;
  background: transparent;
  color: var(--el-text-color-regular);
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
}

.auth-seg__item.is-active {
  color: var(--el-color-primary);
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

.auth-options {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: -2px;
}

.auth-check {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.75rem;
  color: var(--el-text-color-regular);
  cursor: pointer;
  user-select: none;
}

.auth-check input {
  width: 14px;
  height: 14px;
  accent-color: var(--el-color-primary);
  cursor: pointer;
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
  margin: 14px 0 0;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}

.login-fade-enter-active,
.login-fade-leave-active {
  transition:
    opacity 0.14s ease,
    transform 0.14s ease;
}
.login-fade-enter-from {
  opacity: 0;
  transform: translateY(3px);
}
.login-fade-leave-to {
  opacity: 0;
  transform: translateY(-2px);
}
</style>
