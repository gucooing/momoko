<template>
  <div class="login-container">
    <div class="card-wrapper">
      <!-- 背景装饰圆圈 -->
      <div class="bg-decoration-orange"></div>
      <div class="bg-decoration-blue"></div>

      <div class="login-card">
        <!-- 顶部区域 -->
        <div class="login-card-top">
          <!-- logo -->
          <div class="brand">
            <img :src="APP_CONFIG.logoSrc" alt="logo" class="logo" />
            <span class="brand-name">{{ APP_CONFIG.name }}</span>
          </div>
          <!-- 操作按钮 -->
          <div class="top-actions">
            <I18nDropdown />
            <HoverAnimateWrapper name="rotate">
              <IconButton
                icon="HOutline:Cog6ToothIcon"
                :tooltip="t('layout.themeConfig')"
                @click="themeStore.themeConfigDrawerOpen = true"
              />
            </HoverAnimateWrapper>
          </div>
        </div>

        <!-- 底部区域 -->
        <div class="login-card-bottom">
          <div class="login-form-wrap">
            <Transition name="fade-slide" mode="out-in">
              <AccountLogin
                v-if="loginMode === 'login'"
                :username-login-enabled="loginConfig.usernameLoginEnabled"
                :email-login-enabled="loginConfig.emailLoginEnabled"
                :register-enabled="loginConfig.registerEnabled"
                @goToMode="goToMode"
              />
              <ForgotPassword v-else-if="loginMode === 'forgot'" @goToMode="goToMode" />
              <Register v-else :register-email-verification-required="loginConfig.registerEmailVerificationRequired" @goToMode="goToMode" />
            </Transition>
          </div>
        </div>
      </div>
    </div>

    <ThemeConfig />

    <!-- 版权信息 -->
    <div class="login-copyright">Copyright &copy; 2025 DFANNN</div>
  </div>
</template>

<script setup lang="ts">
import { APP_CONFIG } from '@/config/app.config'
import { getLoginConfig } from '@/api/login'
import AccountLogin from '@/views/login/accountLogin.vue'
import ForgotPassword from '@/views/login/forgotPassword.vue'
import Register from '@/views/login/register.vue'
import ThemeConfig from '@/components/ThemeConfig.vue'
import I18nDropdown from '@/layouts/i18nDropdown.vue'
import type { LoginConfig } from '@/types/v1/system'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'LoginView' })

const themeStore = useThemeStore()
const { t } = useI18n()

type LoginMode = 'login' | 'forgot' | 'register'

const loginMode = ref<LoginMode>('login')

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
.login-container {
  min-height: 100vh;
  width: 100%;
  background-color: var(--el-bg-color-page);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  padding: 20px;

  .card-wrapper {
    width: 100%;
    position: relative;
    z-index: 10;
    display: flex;
    align-items: center;
    justify-content: center;

    // 背景装饰 (保留原始颜色)
    .bg-decoration-orange {
      position: absolute;
      bottom: -100px;
      left: -100px;
      width: 400px;
      height: 400px;
      background-color: #f99c7d;
      border-radius: 50%;
      opacity: 0.8;
      z-index: -1;
      animation: float-orange 20s infinite ease-in-out;
      filter: blur(20px);
    }

    .bg-decoration-blue {
      position: absolute;
      top: -120px;
      right: -100px;
      width: 350px;
      height: 450px;
      background-color: #5bbff9;
      border-radius: 40% 60% 70% 30% / 40% 50% 60% 50%;
      opacity: 0.8;
      z-index: -1;
      transform: rotate(15deg);
      animation: float-blue 25s infinite ease-in-out;
      filter: blur(20px);
    }
  }

  .login-card {
    width: min(100%, 38rem);
    max-width: 95%;
    background: var(--el-bg-color-overlay);
    border-radius: 16px;
    box-shadow: var(--el-box-shadow-light);
    display: flex;
    flex-direction: column;
    z-index: 10;
    overflow: hidden;
    padding: 2.5rem;

    .login-card-top {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 2rem;

      .brand {
        display: flex;
        align-items: center;
        gap: 1rem;
        .logo {
          width: 2.5rem;
          height: 2.5rem;
        }
        .brand-name {
          font-size: 1.5rem;
          font-weight: 600;
          color: var(--el-text-color-primary);
        }
      }

      .top-actions {
        display: flex;
        align-items: center;
        .el-link {
          font-size: 0.9rem;
          color: var(--el-text-color-secondary);
        }
      }
    }

    .login-card-bottom {
      display: flex;
      justify-content: center;

      .login-form-wrap {
        width: 100%;
        max-width: 29.5rem;
        min-height: 32rem;
        display: flex;
        flex-direction: column;
        justify-content: center;
        margin: 0 auto;
      }
    }
  }

  .login-copyright {
    position: absolute;
    bottom: 20px;
    left: 0;
    right: 0;
    text-align: center;
    font-size: 0.85rem;
    color: var(--el-text-color-placeholder);
    z-index: 20;
  }
}

@keyframes float-orange {
  0%,
  100% {
    transform: translate(0, 0);
  }
  50% {
    transform: translate(30px, -20px);
  }
}

@keyframes float-blue {
  0%,
  100% {
    transform: rotate(15deg) translate(0, 0);
  }
  50% {
    transform: rotate(20deg) translate(-20px, 30px);
  }
}

:deep(.el-divider__text) {
  background-color: var(--el-bg-color-overlay);
}

@media (max-width: 992px) {
  .login-container {
    padding: 10px;

    .card-wrapper {
      width: 100%;
    }

    .login-card {
      width: 98%;
      max-width: 98%;
      padding: 2rem 1.5rem;
    }
  }
}
</style>
