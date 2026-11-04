<!-- 账号登录：密码登录（用户名或邮箱 + 密码）/ 邮箱验证码两种方式；「N 天内免登录」决定会话是否长期保持。
     外壳（品牌区、标题）由 login/index.vue 提供。 -->
<template>
  <form class="auth-form" novalidate @submit.prevent="handleLogin">
    <AuthSeg
      v-if="showLoginTypeSwitch"
      :model-value="loginType"
      :label="t('authPage.login.methods')"
      :options="[
        { value: 'username', label: t('authPage.login.tabPassword') },
        { value: 'email', label: t('authPage.login.tabCode') },
      ]"
      @update:model-value="setLoginType"
    />

    <div v-if="noMethod" class="auth-note auth-note--warn" role="status">
      <component :is="menuStore.iconComponents['HOutline:ExclamationTriangleIcon']" aria-hidden="true" />
      <span>{{ t('authPage.login.allMethodsOff') }}</span>
    </div>

    <Transition name="auth-fade" mode="out-in">
      <div v-if="loginType === 'username'" key="password-mode" class="auth-pw">
        <AuthField
          id="login-account"
          v-model="loginForm.username"
          :label="t('authPage.field.account')"
          icon="HOutline:UserIcon"
          :placeholder="t('authPage.field.accountPlaceholder')"
          autocomplete="username"
          :error="errors.username"
        />
        <AuthField
          id="login-password"
          v-model="loginForm.password"
          :label="t('authPage.field.password')"
          icon="HOutline:LockClosedIcon"
          type="password"
          revealable
          :placeholder="t('authPage.field.passwordPlaceholder')"
          autocomplete="current-password"
          :error="errors.password"
        />
      </div>

      <div v-else key="code-mode" class="auth-pw">
        <AuthField
          id="login-email"
          v-model="loginForm.username"
          :label="t('authPage.field.email')"
          icon="HOutline:EnvelopeIcon"
          type="email"
          inputmode="email"
          :placeholder="t('authPage.field.emailPlaceholder')"
          autocomplete="email"
          :error="errors.username"
          @input="emailCode.reset()"
        />
        <AuthField
          id="login-code"
          v-model="loginForm.password"
          :label="t('authPage.field.code')"
          icon="HOutline:ShieldCheckIcon"
          inputmode="numeric"
          :maxlength="6"
          :placeholder="t('authPage.field.codePlaceholder')"
          autocomplete="one-time-code"
          :error="errors.password"
        >
          <template #trail>
            <SendCodeButton
              :countdown="emailCode.countdown.value"
              :sending="emailCode.sending.value"
              :disabled="loading"
              @send="handleSendCode"
            />
          </template>
          <template #below>
            <span v-if="emailCode.sentMinutes.value && !errors.password" class="auth-status" role="status">
              <component :is="menuStore.iconComponents['HOutline:CheckIcon']" aria-hidden="true" />
              <span class="auth-status__long">{{ t('authPage.code.sent', { min: emailCode.sentMinutes.value }) }}</span>
              <span class="auth-status__short">{{ t('authPage.code.sentShort', { min: emailCode.sentMinutes.value }) }}</span>
            </span>
          </template>
        </AuthField>
      </div>
    </Transition>

    <div class="auth-options">
      <label class="auth-check" :title="t('authPage.login.rememberHint')">
        <input v-model="loginForm.remember" type="checkbox" @change="handleRememberChange(loginForm.remember)" />
        <span>{{ t('authPage.login.remember', { days: rememberDaysLabel }) }}</span>
      </label>
      <button type="button" class="auth-link" @click="emits('goToMode', 'forgot')">
        {{ t('login.forgotPassword') }}
      </button>
    </div>

    <div class="auth-actions">
      <button type="submit" class="auth-submit" :disabled="loading || noMethod" :aria-busy="loading">
        {{ t('login.submit') }}
        <component
          :is="menuStore.iconComponents[loading ? 'HOutline:ArrowPathIcon' : 'HOutline:ArrowRightIcon']"
          :class="{ 'is-spin': loading }"
          aria-hidden="true"
        />
      </button>
      <p v-if="registerEnabled" class="auth-switch auth-switch--register">
        {{ t('authPage.login.noAccount') }}
        <button type="button" class="auth-link" @click="emits('goToMode', 'register')">
          {{ t('authPage.login.registerLink') }}
        </button>
      </p>
    </div>
  </form>
</template>

<script setup lang="ts">
import platform from 'platform'
import { login, sendLoginEmailCode } from '@/api/login'
import type { LoginRequest } from '@/types/v1/auth'
import { normalizeAuthRedirect } from '@/utils/authRedirect'
import { normalizeAuthToken } from '@/utils/request'
import { getDeviceId } from '@/utils/deviceId'
import { useFeedback } from '@/utils/feedback'
import AuthField from '@/views/login/components/AuthField.vue'
import AuthSeg from '@/views/login/components/AuthSeg.vue'
import SendCodeButton from '@/views/login/components/SendCodeButton.vue'
import { useEmailCode } from '@/views/login/composables/useEmailCode'
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
const VERIFY_CODE_REGEXP = /^\d{6}$/
const REMEMBER_USERNAME_KEY = 'remember_username'
/** 用户取消过「免登录」时记为 '0'，下次默认不勾选 */
const REMEMBER_PREF_KEY = 'login_remember'
const DEFAULT_REMEMBER_DAYS = 14

defineOptions({ name: 'AccountLogin' })

const props = defineProps<{
  usernameLoginEnabled: boolean
  emailLoginEnabled: boolean
  registerEnabled: boolean
  /** 刷新令牌有效期（天）：勾选「免登录」时会话保持的天数 */
  rememberDays?: number
}>()

const emits = defineEmits<IEmits>()

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const fb = useFeedback()
const menuStore = useMenuStore()

const loading = ref(false)
const errors = ref<Record<string, string>>({})

const loginForm = ref<ILoginFormModel>({
  username: '',
  password: '',
  remember: true,
})

const showLoginTypeSwitch = computed(() => props.usernameLoginEnabled && props.emailLoginEnabled)
const noMethod = computed(() => !props.usernameLoginEnabled && !props.emailLoginEnabled)
const rememberDaysLabel = computed(() => props.rememberDays || DEFAULT_REMEMBER_DAYS)

const resolveDefaultLoginType = (): LoginType => {
  if (props.usernameLoginEnabled) return 'username'
  if (props.emailLoginEnabled) return 'email'
  return 'username'
}

const loginType = ref<LoginType>(resolveDefaultLoginType())
const isEmailMode = computed(() => loginType.value === 'email')
const accountValue = computed(() => loginForm.value.username.trim())

const emailCode = useEmailCode((email) => sendLoginEmailCode({ email }))

const resetCredentialInput = () => {
  loginForm.value.password = ''
  errors.value = {}
}

const setLoginType = (type: LoginType) => {
  if (loginType.value === type) return
  loginType.value = type
  resetCredentialInput()
}

const safeStorage = {
  get: (k: string) => {
    try {
      return localStorage.getItem(k)
    } catch {
      return null
    }
  },
  set: (k: string, v: string) => {
    try {
      localStorage.setItem(k, v)
    } catch {
      // 忽略：隐私模式等不可写
    }
  },
  remove: (k: string) => {
    try {
      localStorage.removeItem(k)
    } catch {
      // 忽略
    }
  },
}

const loadRememberedUsername = () => {
  loginForm.value.remember = safeStorage.get(REMEMBER_PREF_KEY) !== '0'
  const rememberedUsername = safeStorage.get(REMEMBER_USERNAME_KEY)
  if (rememberedUsername) loginForm.value.username = rememberedUsername
}

const handleRememberChange = (value: boolean) => {
  const account = accountValue.value
  if (value) {
    safeStorage.remove(REMEMBER_PREF_KEY)
    if (account) safeStorage.set(REMEMBER_USERNAME_KEY, account)
  } else {
    safeStorage.set(REMEMBER_PREF_KEY, '0')
    safeStorage.remove(REMEMBER_USERNAME_KEY)
  }
}

const isValidEmail = (email: string) => EMAIL_REGEXP.test(email)

const handleSendCode = async () => {
  const email = accountValue.value
  if (!isValidEmail(email)) {
    errors.value = { ...errors.value, username: email ? t('login.emailInvalid') : t('login.validEmailFirst') }
    return
  }
  errors.value = { ...errors.value, username: '' }
  await emailCode.request(email)
}

const buildLoginPayload = (): LoginRequest => {
  const account = accountValue.value
  const unknownDevice = t('login.unknownDevice')
  const device =
    `${platform.os?.toString() || unknownDevice} / ${platform.name || unknownDevice} ${platform.version || ''}`.trim()
  const deviceId = getDeviceId()
  const remember = loginForm.value.remember
  return isEmailMode.value
    ? { email: account, password: '', device, deviceId, code: loginForm.value.password.trim(), remember }
    : { username: account, password: loginForm.value.password, device, deviceId, code: '', remember }
}

const validate = (): boolean => {
  const next: Record<string, string> = {}
  const account = accountValue.value
  if (!account) {
    next.username = isEmailMode.value ? t('login.emailPlaceholder') : t('authPage.login.accountRequired')
  } else if (isEmailMode.value && !EMAIL_REGEXP.test(account)) {
    next.username = t('login.emailInvalid')
  }

  const credential = loginForm.value.password.trim()
  if (!credential) {
    next.password = isEmailMode.value ? t('login.codePlaceholder') : t('login.passwordPlaceholder')
  } else if (isEmailMode.value && !VERIFY_CODE_REGEXP.test(credential)) {
    next.password = t('authPage.code.invalid')
  }

  errors.value = next
  return Object.keys(next).length === 0
}

const handleLogin = async () => {
  if (noMethod.value || !validate()) return
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

    handleRememberChange(loginForm.value.remember)

    fb.success(t('login.success'))
    const redirectQuery = Array.isArray(route.query.redirect)
      ? route.query.redirect[0]
      : route.query.redirect
    const redirectPath = normalizeAuthRedirect(redirectQuery)
    await router.push(redirectPath || '/')
  } catch {
    // 请求层已提示错误
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
</script>

<style scoped lang="scss">
@use './auth-form.scss';

/* 手机：注册入口在标题行，这里不再重复 */
@media (width < 900px) {
  .auth-switch--register {
    display: none;
  }
}

.auth-fade-enter-active,
.auth-fade-leave-active {
  transition:
    opacity 0.14s ease,
    transform 0.14s ease;
}
.auth-fade-enter-from {
  opacity: 0;
  transform: translateY(3px);
}
.auth-fade-leave-to {
  opacity: 0;
  transform: translateY(-2px);
}
</style>
