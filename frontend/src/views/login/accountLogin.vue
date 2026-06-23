<template>
  <div class="form-content-inner">
    <h2 class="title">{{ t('login.welcomeBack') }}</h2>
    <p class="subtitle">{{ t('login.subtitle') }}</p>

    <!-- 登录表单 -->
    <el-form
      ref="loginFormRef"
      :model="loginForm"
      :rules="loginRules"
      label-position="top"
      class="login-form"
      @keyup.enter="handleLogin"
    >
      <div v-if="showLoginTypeSwitch" class="login-type-switch" :class="`type-${loginType}`">
        <span class="switch-indicator"></span>
        <button
          type="button"
          class="switch-item"
          :class="{ active: loginType === 'username' }"
          @click="setLoginType('username')"
        >
          {{ t('login.usernameLogin') }}
        </button>
        <button
          type="button"
          class="switch-item"
          :class="{ active: loginType === 'email' }"
          @click="setLoginType('email')"
        >
          {{ t('login.emailLogin') }}
        </button>
      </div>

      <el-form-item prop="username">
        <el-input
          v-model="loginForm.username"
          :placeholder="loginInputPlaceholder"
          @input="handleAccountInput"
        />
      </el-form-item>

      <Transition name="fade-slide" mode="out-in">
        <el-form-item v-if="loginType === 'username'" key="password-mode" prop="password">
          <el-input
            class="credential-input credential-input--password"
            v-model="loginForm.password"
            type="password"
            show-password
            :placeholder="t('login.passwordPlaceholder')"
          />
        </el-form-item>

        <div v-else key="code-mode" class="code-row">
          <el-form-item prop="password" class="code-input-item">
            <el-input
              class="credential-input credential-input--code"
              v-model="loginForm.password"
              :placeholder="t('login.codePlaceholder')"
              maxlength="6"
            />
          </el-form-item>

          <el-button
            type="primary"
            class="send-code-btn send-code-btn--standalone"
            :class="{ 'is-countdown': sendCodeCountdown > 0 }"
            :disabled="sendCodeDisabled"
            @click="handleSendCode"
          >
            {{ sendCodeButtonText }}
          </el-button>
        </div>
      </Transition>

      <div class="form-options">
        <el-checkbox v-model="loginForm.remember" @change="handleRememberChange">
          {{ t('login.rememberMe') }}
        </el-checkbox>
        <el-link type="primary" :underline="false" @click="emits('goToMode', 'forgot')"
          >{{ t('login.forgotPassword') }}</el-link
        >
      </div>

      <el-button type="primary" class="submit-btn" :loading="loading" @click="handleLogin">
        {{ t('login.submit') }}
      </el-button>
    </el-form>

    <p v-if="registerEnabled" class="register-link">
      <span>{{ t('login.noAccount') }}</span>
      <el-link type="primary" :underline="false" @click="emits('goToMode', 'register')"
        >{{ t('login.registerNow') }}</el-link
      >
    </p>
  </div>
</template>

<script setup lang="ts">
import platform from 'platform'
import { login, sendLoginEmailCode } from '@/api/login'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules, FormItemRule } from 'element-plus'
import type { LoginRequest } from '@/types/v1/auth'
import { normalizeAuthRedirect } from '@/utils/authRedirect'
import { normalizeAuthToken } from '@/utils/request'
import { getDeviceId } from '@/utils/deviceId'
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
const loginFormRef = useTemplateRef<FormInstance>('loginFormRef')
const loading = ref(false)

// 记住我的 localStorage key
const REMEMBER_USERNAME_KEY = 'remember_username'

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
const loginInputPlaceholder = computed(() => {
  return isEmailMode.value ? t('login.emailPlaceholder') : t('login.usernamePlaceholder')
})
const sendCodeCountdown = ref(0)
let sendCodeTimer: ReturnType<typeof setInterval> | null = null

const resetCredentialInput = () => {
  loginForm.value.password = ''
  loginFormRef.value?.clearValidate(['username', 'password'])
}

const setLoginType = (type: LoginType) => {
  if (loginType.value === type) return
  loginType.value = type
  resetCredentialInput()
}

const handleAccountInput = (value: string | number) => {
  if (
    typeof value === 'string' &&
    value.includes('@') &&
    !isEmailMode.value &&
    props.emailLoginEnabled
  ) {
    loginType.value = 'email'
    resetCredentialInput()
  }
}

// 从 localStorage 读取记住的用户名
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

// 保存或清除记住的用户名
const handleRememberChange = (value: boolean | string | number) => {
  const remember = Boolean(value)
  const account = accountValue.value
  if (remember) {
    if (account) {
      localStorage.setItem(REMEMBER_USERNAME_KEY, account)
    }
  } else {
    localStorage.removeItem(REMEMBER_USERNAME_KEY)
  }
}

const isValidEmail = (email: string) => {
  return EMAIL_REGEXP.test(email)
}

const sendCodeButtonText = computed(() => {
  return sendCodeCountdown.value > 0
    ? t('login.resendIn', { seconds: sendCodeCountdown.value })
    : t('login.sendCode')
})

const sendCodeDisabled = computed(() => {
  return loading.value || sendCodeCountdown.value > 0 || !isValidEmail(accountValue.value)
})

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
    ElMessage.warning(t('login.validEmailFirst'))
    return
  }

  try {
    await sendLoginEmailCode({ email })
    ElMessage.success(t('login.codeSent'))
    startSendCodeCountdown()
  } catch {
    ElMessage.error(t('login.codeSendFailed'))
  }
}

const buildLoginPayload = async (): Promise<LoginRequest> => {
  const account = accountValue.value
  const unknownDevice = t('login.unknownDevice')
  const device = `${platform.os?.toString() || unknownDevice} / ${platform.name || unknownDevice} ${platform.version || ''}`.trim()
  const deviceId = getDeviceId()
  const useEmail = isEmailMode.value

  return useEmail
    ? { email: account, password: '', device, deviceId, code: loginForm.value.password }
    : { username: account, password: loginForm.value.password, device, deviceId, code: '' }
}

// 登录
const handleLogin = async () => {
  await loginFormRef.value?.validate()
  loading.value = true
  try {
    const payload = await buildLoginPayload()
    const { data: loginRes } = await login(payload)

    if (!loginRes?.accessToken) {
      ElMessage.error(t('login.missingAccessToken'))
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

    ElMessage.success(t('login.success'))
    const redirectQuery = Array.isArray(route.query.redirect)
      ? route.query.redirect[0]
      : route.query.redirect
    const redirectPath = normalizeAuthRedirect(redirectQuery)

    await router.push(redirectPath || '/')
  } finally {
    loading.value = false
  }
}

const validateAccount: FormItemRule['validator'] = (_rule, value, callback) => {
  const account = String(value || '').trim()
  if (!account) {
    callback(new Error(isEmailMode.value ? t('login.emailPlaceholder') : t('login.usernamePlaceholder')))
    return
  }

  if (isEmailMode.value) {
    if (!EMAIL_REGEXP.test(account)) {
      callback(new Error(t('login.emailInvalid')))
      return
    }
  }

  callback()
}

const validateCredential: FormItemRule['validator'] = (_rule, value, callback) => {
  const credential = String(value || '').trim()
  if (!credential) {
    callback(new Error(isEmailMode.value ? t('login.codePlaceholder') : t('login.passwordPlaceholder')))
    return
  }

  if (isEmailMode.value && !VERIFY_CODE_REGEXP.test(credential)) {
    callback(new Error(t('login.codeInvalid')))
    return
  }

  callback()
}

const loginRules = reactive<FormRules>({
  username: [{ validator: validateAccount, trigger: 'blur' }],
  password: [{ validator: validateCredential, trigger: 'blur' }],
})

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
.form-content-inner {
  .fade-slide-enter-active,
  .fade-slide-leave-active {
    transition: all 0.22s ease;
  }
  .fade-slide-enter-from {
    opacity: 0;
    transform: translateY(6px);
  }
  .fade-slide-leave-to {
    opacity: 0;
    transform: translateY(-6px);
  }

  .title {
    font-size: 1.75rem;
    font-weight: 700;
    color: var(--el-text-color-primary);
    margin-bottom: 0.5rem;
  }

  .subtitle {
    font-size: 0.95rem;
    color: var(--el-text-color-secondary);
    margin-bottom: 1.7rem;
  }

  .login-form {
    .credential-input {
      width: 100%;
    }

    .credential-input--code {
      width: 100%;
    }

    :deep(.el-input__wrapper),
    :deep(.el-select__wrapper) {
      padding: 0.5rem 1rem;
      border-radius: 0.5rem;
      box-shadow: 0 0 0 1px var(--el-border-color) inset;
      min-height: 2.75rem;

      &.is-focus {
        box-shadow: 0 0 0 1px var(--el-color-primary) inset;
      }
    }

    .code-row {
      display: flex;
      align-items: flex-start;
      gap: 0.75rem;
      width: 100%;
    }

    .code-input-item {
      flex: 1;
      margin-bottom: 0;

      :deep(.el-form-item__content) {
        width: 100%;
      }
    }

    .send-code-btn {
      position: relative;
      height: 2.75rem;
      min-width: 8.5rem;
      border: none;
      border-radius: 0.75rem;
      padding: 0 1rem;
      font-weight: 600;
      color: #fff;
      background: linear-gradient(
        120deg,
        color-mix(in srgb, var(--el-color-primary) 84%, #ffffff),
        var(--el-color-primary)
      );
      transition:
        transform 0.15s ease,
        filter 0.2s ease,
        box-shadow 0.2s ease,
        background 0.2s ease;
      box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--el-color-primary) 72%, transparent);

      &::after {
        content: '';
        position: absolute;
        inset: 0;
        background: radial-gradient(circle at center, rgba(255, 255, 255, 0.22), transparent 65%);
        opacity: 0;
        transform: scale(0.9);
        transition: all 0.18s ease;
        pointer-events: none;
      }

      &:hover:not(:disabled) {
        filter: brightness(1.06);
        box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--el-color-primary) 90%, transparent);
      }

      &:active:not(:disabled) {
        transform: translateY(1px) scale(0.97);
        filter: brightness(0.96);

        &::after {
          opacity: 1;
          transform: scale(1.1);
        }
      }

      &.is-countdown {
        background: color-mix(in srgb, var(--el-color-primary) 24%, var(--el-bg-color-overlay));
        color: var(--el-color-primary);
        box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--el-color-primary) 40%, transparent);
      }

      &.is-countdown:disabled {
        color: var(--el-color-primary);
        background: color-mix(in srgb, var(--el-color-primary) 24%, var(--el-bg-color-overlay));
        box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--el-color-primary) 40%, transparent);
        opacity: 0.9;
        transform: none;
        filter: none;
      }

      &:disabled:not(.is-countdown) {
        color: var(--el-text-color-placeholder);
        background: color-mix(in srgb, var(--el-color-primary) 12%, var(--el-fill-color-light));
        box-shadow: none;
        transform: none;
        filter: none;
      }
    }

    .login-type-switch {
      position: relative;
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      padding: 0.25rem;
      margin-bottom: 1rem;
      border-radius: 0.75rem;
      background: var(--el-fill-color-light);
      overflow: hidden;

      .switch-indicator {
        position: absolute;
        top: 0.25rem;
        left: 0.25rem;
        width: calc(50% - 0.25rem);
        height: calc(100% - 0.5rem);
        border-radius: 0.6rem;
        background: var(--el-bg-color-overlay);
        box-shadow: 0 0 0 1px color-mix(in srgb, var(--el-color-primary) 18%, transparent) inset;
        transition: transform 0.28s cubic-bezier(0.22, 1, 0.36, 1);
      }

      &.type-email {
        .switch-indicator {
          transform: translateX(100%);
        }
      }

      .switch-item {
        position: relative;
        z-index: 1;
        height: 2.25rem;
        border: none;
        border-radius: 0.6rem;
        background: transparent;
        color: var(--el-text-color-regular);
        font-size: 0.875rem;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s ease;

        &:hover {
          color: var(--el-color-primary);
        }

        &.active {
          color: var(--el-color-primary);
        }
      }
    }

    .form-options {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 1.5rem;
    }

    .submit-btn {
      width: 100%;
      height: 2.75rem;
      border-radius: 0.75rem;
      font-size: 1rem;
      font-weight: 600;
      margin-bottom: 1rem;
      letter-spacing: 0.5rem;
    }
  }

  @media (max-width: 400px) {
    .send-code-btn {
      min-width: 6.5rem;
      padding: 0 0.5rem;
      font-size: 0.72rem;
    }
  }

  .register-link {
    display: flex;
    justify-content: center;
    align-items: center;
    font-size: 0.875rem;
    color: var(--el-text-color-secondary);
    .el-link {
      margin-left: 0.5rem;
      font-weight: 600;
    }
  }
}
</style>
