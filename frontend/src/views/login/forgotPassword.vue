<!-- 找回密码：占位流程；紧凑卡内表单。 -->
<template>
  <div class="auth-form">
    <header class="auth-form__head">
      <h1 class="auth-form__title">{{ t('forgot.title') }}</h1>
      <p class="auth-form__sub">{{ t('forgot.subtitle') }}</p>
    </header>

    <form class="auth-form__body" @submit.prevent="handleAction">
      <div class="app-field">
        <label class="app-label" for="forgot-email">{{ t('forgot.emailPlaceholder') }}</label>
        <input
          id="forgot-email"
          v-model="forgotPasswordForm.email"
          class="app-input auth-input"
          :class="{ 'is-error': error }"
          type="email"
          :placeholder="t('forgot.emailPlaceholder')"
          autocomplete="email"
        />
        <span v-if="error" class="app-field__error">{{ error }}</span>
      </div>

      <UButton type="submit" color="primary" block class="auth-submit">
        {{ t('forgot.submit') }}
      </UButton>
    </form>

    <p class="auth-switch">
      <button type="button" class="auth-text-btn" @click="emits('goToMode', 'login')">
        {{ t('forgot.backLogin') }}
      </button>
    </p>
  </div>
</template>

<script setup lang="ts">
import { useFeedback } from '@/utils/feedback'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'ForgotPassword' })

const emits = defineEmits<{
  (e: 'goToMode', mode: 'login' | 'forgot' | 'register'): void
}>()

const { t } = useI18n()
const fb = useFeedback()

const EMAIL_REGEXP = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

const forgotPasswordForm = ref({
  email: '',
})
const error = ref('')

const handleAction = () => {
  const email = forgotPasswordForm.value.email.trim()
  if (!email) {
    error.value = t('forgot.emailPlaceholder')
    return
  }
  if (!EMAIL_REGEXP.test(email)) {
    error.value = t('login.emailInvalid')
    return
  }
  error.value = ''
  fb.info(t('forgot.comingSoon'))
}
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

.auth-submit {
  margin-top: 4px;
  height: 36px !important;
  font-size: 0.875rem !important;
  font-weight: 600;
}

.auth-switch {
  display: flex;
  justify-content: center;
  margin: 14px 0 0;
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
