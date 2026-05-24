<template>
  <div class="form-content-inner">
    <h2 class="title">创建账号</h2>
    <p class="subtitle">加入 {{ APP_CONFIG.name }}，开始您的管理之旅</p>

    <el-form
      ref="registerFormRef"
      :model="registerForm"
      :rules="registerRules"
      label-position="top"
      class="register-form"
      @keyup.enter="handleRegister"
    >
      <el-form-item prop="username">
        <el-input v-model="registerForm.username" placeholder="设置用户名" />
      </el-form-item>

      <el-form-item prop="email">
        <el-input v-model="registerForm.email" placeholder="输入电子邮箱" @input="handleEmailInput" />
      </el-form-item>

      <div v-if="props.registerEmailVerificationRequired" class="code-row">
        <el-form-item prop="code" class="code-input-item">
          <el-input
            v-model="registerForm.code"
            placeholder="请输入验证码"
            maxlength="6"
          />
        </el-form-item>
        <el-button
          type="primary"
          class="send-code-btn"
          :class="{ 'is-countdown': sendCodeCountdown > 0 }"
          :disabled="sendCodeDisabled"
          @click="handleSendCode"
        >
          {{ sendCodeButtonText }}
        </el-button>
      </div>

      <el-form-item prop="password">
        <el-input
          v-model="registerForm.password"
          type="password"
          show-password
          placeholder="设置登录密码"
        />
      </el-form-item>

      <el-form-item prop="confirmPassword">
        <el-input
          v-model="registerForm.confirmPassword"
          type="password"
          show-password
          placeholder="确认您的密码"
        />
      </el-form-item>

      <el-button type="primary" class="submit-btn" :loading="loading" @click="handleRegister">
        立即注册
      </el-button>

      <div class="back-link">
        <span class="have-account">已有账号？</span>
        <el-link :underline="false" @click="emits('goToMode', 'login')">
          <el-icon><component :is="menuStore.iconComponents['Element:ArrowLeft']" /></el-icon>
          返回登录
        </el-link>
      </div>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { APP_CONFIG } from '@/config/app.config'
import { register, sendRegisterEmailCode } from '@/api/login'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules, FormItemRule } from 'element-plus'

defineOptions({ name: 'RegisterComponent' })

const props = defineProps<{
  registerEmailVerificationRequired: boolean
}>()

const emits = defineEmits<{
  (e: 'goToMode', mode: 'login' | 'forgot' | 'register'): void
}>()

const menuStore = useMenuStore()
const registerFormRef = useTemplateRef<FormInstance>('registerFormRef')
const loading = ref(false)

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

const sendCodeButtonText = computed(() => {
  return sendCodeCountdown.value > 0 ? `${sendCodeCountdown.value}s 后重发` : '发送验证码'
})

const sendCodeDisabled = computed(() => {
  return loading.value || sendCodeCountdown.value > 0 || !isValidEmail(registerForm.value.email.trim())
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

const handleEmailInput = () => {
  if (!props.registerEmailVerificationRequired) return
  registerForm.value.code = ''
}

const handleSendCode = async () => {
  const email = registerForm.value.email.trim()
  if (!isValidEmail(email)) {
    ElMessage.warning('请先输入正确的邮箱地址')
    return
  }

  try {
    await sendRegisterEmailCode({ email })
    ElMessage.success('验证码已发送，请注意查收')
    startSendCodeCountdown()
  } catch {
    ElMessage.error('验证码发送失败，请稍后重试')
  }
}

const validateUsername: FormItemRule['validator'] = (_rule, value, callback) => {
  const val = String(value || '').trim()
  if (!val) {
    callback(new Error('请输入用户名'))
    return
  }
  callback()
}

const validateEmail: FormItemRule['validator'] = (_rule, value, callback) => {
  const val = String(value || '').trim()
  if (!val) {
    callback(new Error('请输入邮箱'))
    return
  }
  if (!EMAIL_REGEXP.test(val)) {
    callback(new Error('邮箱格式不正确'))
    return
  }
  callback()
}

const validatePassword: FormItemRule['validator'] = (_rule, value, callback) => {
  const val = String(value || '')
  if (!val) {
    callback(new Error('请输入密码'))
    return
  }
  if (val.length < 6) {
    callback(new Error('密码长度不能少于6位'))
    return
  }
  callback()
}

const validateConfirmPassword: FormItemRule['validator'] = (_rule, value, callback) => {
  const val = String(value || '')
  if (!val) {
    callback(new Error('请确认密码'))
    return
  }
  if (val !== registerForm.value.password) {
    callback(new Error('两次输入的密码不一致'))
    return
  }
  callback()
}

const validateCode: FormItemRule['validator'] = (_rule, value, callback) => {
  if (!props.registerEmailVerificationRequired) {
    callback()
    return
  }
  const val = String(value || '').trim()
  if (!val) {
    callback(new Error('请输入邮箱验证码'))
    return
  }
  callback()
}

const registerRules = reactive<FormRules>({
  username: [{ validator: validateUsername, trigger: 'blur' }],
  email: [{ validator: validateEmail, trigger: 'blur' }],
  password: [{ validator: validatePassword, trigger: 'blur' }],
  confirmPassword: [{ validator: validateConfirmPassword, trigger: 'blur' }],
  code: [{ validator: validateCode, trigger: 'blur' }],
})

const handleRegister = async () => {
  await registerFormRef.value?.validate()
  loading.value = true
  try {
    await register({
      username: registerForm.value.username.trim(),
      email: registerForm.value.email.trim(),
      password: registerForm.value.password,
      code: registerForm.value.code.trim(),
    })
    ElMessage.success('注册成功，请登录')
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
.form-content-inner {
  .title {
    font-size: 1.75rem;
    font-weight: 700;
    color: var(--el-text-color-primary);
    margin-bottom: 0.5rem;
  }

  .subtitle {
    font-size: 0.95rem;
    color: var(--el-text-color-secondary);
    margin-bottom: 2rem;
  }

  .register-form {
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

    @media (max-width: 768px) {
      .code-row {
        flex-direction: column;
      }

      .code-input-item {
        width: 100%;
      }

      .send-code-btn {
        width: 100%;
      }
    }

    .submit-btn {
      width: 100%;
      height: 2.75rem;
      border-radius: 0.75rem;
      font-size: 1rem;
      font-weight: 600;
      margin-top: 0.9rem;
      margin-bottom: 1.5rem;
    }

    .back-link {
      display: flex;
      justify-content: center;
      align-items: center;
      gap: 0.5rem;

      .have-account {
        font-size: 0.875rem;
        color: var(--el-text-color-secondary);
      }

      .el-link {
        font-size: 0.9rem;
        font-weight: 600;
        transition: all 0.3s;
        color: var(--el-text-color-secondary);

        &:hover {
          color: var(--el-color-primary);
          transform: translateX(-4px);
        }
      }
    }
  }
}
</style>
