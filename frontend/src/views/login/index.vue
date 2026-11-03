<!-- 登录页：左侧主色品牌区（贴纸标签 + 装饰圆）+ 右侧表单；手机上品牌区在上、表单卡片上浮。
     登录/注册/找回密码三种表单共用本页外壳与样式。 -->
<template>
  <div class="login-page">
    <section class="login-brand" aria-hidden="false">
      <span class="login-brand__blob" aria-hidden="true" />
      <span class="login-brand__ring" aria-hidden="true" />
      <svg class="login-brand__star" viewBox="0 0 24 24" aria-hidden="true">
        <path d="M12 1.5l2.6 7.9 7.9 2.6-7.9 2.6L12 22.5l-2.6-7.9L1.5 12l7.9-2.6z" />
      </svg>
      <div class="login-brand__logo">
        <img :src="APP_CONFIG.logoSrc" alt="" />
        <span>{{ APP_CONFIG.name }}</span>
      </div>
      <div class="login-brand__copy">
        <h1>
          {{ t('login.heroLine1') }}<br />
          {{ t('login.heroLine2') }}
        </h1>
        <p>{{ t('login.heroDesc') }}</p>
      </div>
      <ul class="login-brand__tags">
        <li v-for="(tag, i) in heroTags" :key="tag" :class="`is-${i % 5}`">{{ tag }}</li>
      </ul>
    </section>

    <section class="login-panel">
      <div class="login-panel__chrome">
        <LanguageMenu />
        <AppIconButton
          v-if="APP_CONFIG.showThemeConfig"
          icon="HOutline:SwatchIcon"
          :label="t('layout.themeConfig')"
          @click="themeStore.themeConfigDrawerOpen = true"
        />
        <AppIconButton
          :icon="themeStore.isDarkTheme ? 'HOutline:SunIcon' : 'HOutline:MoonIcon'"
          :label="t('login.toggleTheme')"
          @click="toggleTheme"
        />
      </div>

      <div class="login-panel__center">
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

      <p class="login-panel__copy">
        {{ t('login.copyright', { year: currentYear, name: APP_CONFIG.name }) }}
      </p>
    </section>

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
const heroTags = computed(() => t('login.heroTags').split(',').filter(Boolean))

const loginConfig = reactive<LoginConfig>({
  registerEnabled: false,
  usernameLoginEnabled: true,
  emailLoginEnabled: false,
  registerEmailVerificationRequired: false,
  accessTokenTtlMinutes: 0,
  refreshTokenTtlDays: 0,
  lockoutThreshold: 0,
  lockoutMinutes: 0,
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
  display: flex;
  flex-wrap: wrap;
  min-height: 100vh;
  min-height: 100dvh;
  width: 100%;
  background: var(--el-bg-color);
}

/* ---- 品牌区 ---- */
.login-brand {
  position: relative;
  flex: 1 1 520px;
  min-height: 100vh;
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 36px;
  padding: 48px 56px;
  overflow: hidden;
  background: var(--m-p);
  color: var(--m-pOn);
}
.login-brand__blob {
  position: absolute;
  right: -120px;
  bottom: -160px;
  width: 460px;
  height: 460px;
  border-radius: 50%;
  background: var(--m-a);
}
.login-brand__ring {
  position: absolute;
  right: 26%;
  top: 9%;
  width: 150px;
  height: 150px;
  border-radius: 50%;
  border: 22px solid var(--m-a2);
}
.login-brand__star {
  position: absolute;
  right: 80px;
  top: 120px;
  width: 60px;
  height: 60px;
  fill: var(--m-pOn);
}
.login-brand__logo {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
  font-family: var(--app-font-display);
  font-size: 1.625rem;
  font-weight: 800;
}
.login-brand__logo img {
  width: 48px;
  height: 48px;
  border-radius: 15px;
  object-fit: cover;
  transform: rotate(-6deg);
  box-shadow: 0 0 0 3px var(--m-pOn);
}
.login-brand__copy {
  position: relative;
  max-width: 560px;
}
.login-brand__copy h1 {
  font-family: var(--app-font-display);
  font-size: clamp(2.25rem, 4.2vw, 3.625rem);
  line-height: 1.08;
  font-weight: 800;
  letter-spacing: -0.02em;
}
.login-brand__copy p {
  margin-top: 18px;
  font-size: 1.125rem;
  line-height: 1.6;
  opacity: 0.92;
}
.login-brand__tags {
  position: relative;
  display: flex;
  flex-wrap: wrap;
  gap: 12px 14px;
  max-width: 460px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.login-brand__tags li {
  display: inline-flex;
  align-items: center;
  height: 46px;
  padding: 0 20px;
  border-radius: 999px;
  font-size: 1.0625rem;
  font-weight: 800;
  box-shadow: 0 10px 24px -12px rgba(0, 0, 0, 0.35);
}
.login-brand__tags .is-0 {
  background: var(--m-pOn);
  color: var(--m-p);
  transform: rotate(-5deg);
}
.login-brand__tags .is-1 {
  background: var(--m-ink);
  color: var(--m-bg);
  transform: rotate(4deg);
}
.login-brand__tags .is-2 {
  background: var(--m-a2);
  color: #fff;
  transform: rotate(3deg);
}
.login-brand__tags .is-3 {
  background: var(--m-aSoft);
  color: var(--m-aOn);
  transform: rotate(-3deg);
}
.login-brand__tags .is-4 {
  background: var(--m-pOn);
  color: var(--m-p);
  transform: rotate(5deg);
}

/* ---- 表单区 ---- */
.login-panel {
  position: relative;
  flex: 1 1 460px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  padding: 24px 32px 28px;
  background: var(--el-bg-color);
}
.login-panel__chrome {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 4px;
}
.login-panel__center {
  flex: 1;
  width: min(100%, 400px);
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 24px 0;
}
.login-panel__copy {
  margin: 0;
  text-align: center;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}

/* ---- 三种表单的统一观感（子组件 scoped 样式之上再调一档） ---- */
.login-panel :deep(.auth-form__head) {
  margin-bottom: 1.5rem;
}
.login-panel :deep(.auth-form__title) {
  font-family: var(--app-font-display);
  font-size: 2.125rem;
  font-weight: 800;
  letter-spacing: -0.01em;
}
.login-panel :deep(.auth-form__sub) {
  margin-top: 6px;
  font-size: 0.9375rem;
}
.login-panel :deep(.auth-form__body) {
  gap: 18px;
}
.login-panel :deep(.auth-input) {
  height: 50px;
  padding-left: 16px;
  border-radius: 16px;
  font-size: 0.9375rem;
}
.login-panel :deep(.auth-eye) {
  right: 6px;
  width: 40px;
  height: 40px;
  border-radius: 12px;
}
.login-panel :deep(.auth-eye__icon) {
  width: 20px;
  height: 20px;
}
.login-panel :deep(.auth-seg) {
  padding: 4px;
  border-radius: 999px;
}
.login-panel :deep(.auth-seg__indicator) {
  top: 4px;
  left: 4px;
  width: calc(50% - 4px);
  height: calc(100% - 8px);
  border-radius: 999px;
  box-shadow: var(--app-shadow-sm);
}
.login-panel :deep(.auth-seg__item) {
  height: 40px;
  font-size: 0.875rem;
  font-weight: 700;
}
.login-panel :deep(.auth-seg__item.is-active) {
  color: var(--el-text-color-primary);
}
.login-panel :deep(.auth-code-btn) {
  height: 50px !important;
}
.login-panel :deep(.auth-check),
.login-panel :deep(.auth-text-btn),
.login-panel :deep(.auth-switch) {
  font-size: 0.875rem;
}
.login-panel :deep(.auth-check) {
  min-height: 44px;
}
.login-panel :deep(.auth-check input) {
  width: 18px;
  height: 18px;
}
.login-panel :deep(.auth-text-btn) {
  min-height: 44px;
  color: var(--m-pInk);
}
.login-panel :deep(.auth-submit) {
  height: 54px !important;
  border-radius: 999px;
  font-size: 1rem !important;
  font-weight: 800;
  box-shadow: var(--app-shadow-brand);
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

/* ---- 手机：品牌区在上，表单卡片上浮 ---- */
@media (width < 900px) {
  .login-brand {
    flex: 1 1 100%;
    min-height: 0;
    padding: 22px 20px 72px;
    gap: 22px;
  }
  .login-brand__blob {
    right: -70px;
    top: -70px;
    bottom: auto;
    width: 220px;
    height: 220px;
  }
  .login-brand__ring {
    display: none;
  }
  .login-brand__star,
  .login-brand__copy p {
    display: none;
  }
  .login-brand__logo {
    font-size: 1.3rem;
  }
  .login-brand__logo img {
    width: 40px;
    height: 40px;
    border-radius: 13px;
  }
  .login-brand__copy h1 {
    font-size: 2rem;
  }
  .login-brand__tags {
    gap: 8px;
  }
  .login-brand__tags li {
    height: 34px;
    padding: 0 14px;
    font-size: 0.875rem;
  }
  .login-brand__tags li:nth-child(n + 4) {
    display: none;
  }
  .login-panel {
    position: static;
    flex: 1 1 100%;
    margin-top: -40px;
    padding: 12px 20px 24px;
    border-radius: 32px 32px 0 0;
  }
  .login-panel__chrome {
    position: absolute;
    top: 14px;
    right: 12px;
    z-index: 2;
    padding: 2px;
    border-radius: 999px;
    background: var(--el-bg-color);
    box-shadow: var(--app-shadow-sm);
  }
  .login-brand__logo {
    padding-right: 150px;
  }
  .login-panel__center {
    padding: 14px 0 20px;
  }
  .login-panel :deep(.auth-form__title) {
    font-size: 1.75rem;
  }
}
</style>
