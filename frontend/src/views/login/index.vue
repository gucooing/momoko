<!-- 登录门面（P7）：居中卡 + 页底版权；语言/主题角标。
     不用半屏渐变品牌区（易显空、廉价）；主色只落在按钮与焦点。 -->
<template>
  <div class="login-page">
    <div class="login-page__chrome">
      <LanguageMenu />
      <AppIconButton
        v-if="APP_CONFIG.showThemeConfig"
        icon="HOutline:Cog6ToothIcon"
        :label="t('layout.themeConfig')"
        :box="32"
        @click="themeStore.themeConfigDrawerOpen = true"
      />
      <AppIconButton
        :icon="themeStore.isDarkTheme ? 'HOutline:SunIcon' : 'HOutline:MoonIcon'"
        :label="t('login.toggleTheme')"
        :box="32"
        @click="toggleTheme"
      />
    </div>

    <div class="login-page__center">
      <div class="login-card">
        <div class="login-card__brand">
          <img :src="APP_CONFIG.logoSrc" alt="" class="login-card__logo" />
          <span class="login-card__name">{{ APP_CONFIG.name }}</span>
        </div>

        <div class="login-card__body">
          <Transition name="login-fade" mode="out-in">
            <AccountLogin
              v-if="loginMode === 'login'"
              key="login"
              :username-login-enabled="loginConfig.usernameLoginEnabled"
              :email-login-enabled="loginConfig.emailLoginEnabled"
              :register-enabled="loginConfig.registerEnabled"
              @go-to-mode="goToMode"
            />
            <ForgotPassword
              v-else-if="loginMode === 'forgot'"
              key="forgot"
              @go-to-mode="goToMode"
            />
            <Register
              v-else
              key="register"
              :register-email-verification-required="loginConfig.registerEmailVerificationRequired"
              @go-to-mode="goToMode"
            />
          </Transition>
        </div>
      </div>

      <p class="login-page__copy">
        {{ t('login.copyright', { year: currentYear, name: APP_CONFIG.name }) }}
      </p>
    </div>

    <ThemeConfig />
  </div>
</template>

<script setup lang="ts">
import { APP_CONFIG } from '@/config/app.config'
import { getLoginConfig } from '@/api/login'
import AccountLogin from '@/views/login/accountLogin.vue'
import ForgotPassword from '@/views/login/forgotPassword.vue'
import Register from '@/views/login/register.vue'
import ThemeConfig from '@/components/ThemeConfig.vue'
import LanguageMenu from '@/layouts/app/LanguageMenu.vue'
import type { LoginConfig } from '@/types/v1/system'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'LoginView' })

const themeStore = useThemeStore()
const { t } = useI18n()

type LoginMode = 'login' | 'forgot' | 'register'

const loginMode = ref<LoginMode>('login')
const currentYear = new Date().getFullYear()

const loginConfig = reactive<LoginConfig>({
  registerEnabled: false,
  usernameLoginEnabled: true,
  emailLoginEnabled: false,
  registerEmailVerificationRequired: false,
})

const goToMode = (mode: LoginMode) => {
  if (mode === 'register' && !loginConfig.registerEnabled) return
  loginMode.value = mode
}

const toggleTheme = () => {
  themeStore.toggleThemeMode(themeStore.isDarkTheme ? 'light' : 'dark')
}

onMounted(async () => {
  try {
    const { data } = await getLoginConfig()
    if (data?.config) {
      loginConfig.registerEnabled = data.config.registerEnabled
      loginConfig.usernameLoginEnabled = data.config.usernameLoginEnabled
      loginConfig.emailLoginEnabled = data.config.emailLoginEnabled
      loginConfig.registerEmailVerificationRequired = data.config.registerEmailVerificationRequired
    }
  } catch {
    // 使用默认值
  }
})
</script>

<style scoped lang="scss">
.login-page {
  position: relative;
  min-height: 100vh;
  min-height: 100dvh;
  width: 100%;
  background: var(--el-bg-color-page);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 16px 32px;
  box-sizing: border-box;
}

.login-page__chrome {
  position: absolute;
  top: 12px;
  right: 12px;
  display: flex;
  align-items: center;
  gap: 2px;
  z-index: 2;
}

.login-page__center {
  width: min(100%, 400px);
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 16px;
}

.login-card {
  background: var(--el-bg-color);
  border: 1px solid var(--app-hairline);
  border-radius: var(--app-radius-lg);
  padding: 28px 28px 24px;
  box-shadow: var(--app-shadow-sm);
}

.login-card__brand {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 22px;
}

.login-card__logo {
  width: 32px;
  height: 32px;
  object-fit: contain;
  flex-shrink: 0;
}

.login-card__name {
  font-size: 1.0625rem;
  font-weight: 650;
  letter-spacing: -0.01em;
  color: var(--el-text-color-primary);
}

.login-card__body {
  min-height: 0;
}

.login-page__copy {
  margin: 0;
  text-align: center;
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}

.login-fade-enter-active,
.login-fade-leave-active {
  transition:
    opacity 0.16s ease,
    transform 0.16s ease;
}
.login-fade-enter-from {
  opacity: 0;
  transform: translateY(4px);
}
.login-fade-leave-to {
  opacity: 0;
  transform: translateY(-3px);
}

@media (max-width: 480px) {
  .login-page {
    /* 与桌面一致：整页垂直居中，不贴顶 */
    padding: 56px 12px 32px;
    align-items: center;
  }

  .login-card {
    padding: 22px 18px 20px;
    border-radius: var(--app-radius);
  }
}
</style>
