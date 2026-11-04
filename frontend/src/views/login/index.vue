<!-- 登录 / 注册 / 找回密码共用的公开页外壳：左侧主色品牌区（文案随模式切换 + 贴纸 + 装饰圆）+ 右侧表单；
     手机上品牌区在上、表单卡片上浮，主按钮固定在底部。模式写在 ?mode=register|forgot，可直接链接、可后退。 -->
<template>
  <div class="auth" :class="`auth--${loginMode}`">
    <section class="auth-brand">
      <span class="auth-brand__blob" aria-hidden="true" />
      <span class="auth-brand__ring" aria-hidden="true" />
      <svg class="auth-brand__star" viewBox="0 0 24 24" aria-hidden="true">
        <path d="M12 1.5l2.6 7.9 7.9 2.6-7.9 2.6L12 22.5l-2.6-7.9L1.5 12l7.9-2.6z" />
      </svg>

      <div class="auth-brand__bar">
        <button
          v-if="loginMode !== 'login'"
          type="button"
          class="auth-brand__btn auth-brand__back"
          :aria-label="t('authPage.backToLogin')"
          @click="goToMode('login')"
        >
          <component :is="menuStore.iconComponents['HOutline:ChevronLeftIcon']" aria-hidden="true" />
        </button>
        <div class="auth-brand__logo">
          <img :src="APP_CONFIG.logoSrc" alt="" />
          <span>{{ brandName }}</span>
        </div>
        <div class="auth-brand__tools">
          <AuthLangMenu compact on-brand />
          <button
            v-if="APP_CONFIG.showThemeConfig"
            type="button"
            class="auth-brand__btn auth-brand__btn--chip"
            :aria-label="t('layout.themeConfig')"
            @click="themeStore.themeConfigDrawerOpen = true"
          >
            <component :is="menuStore.iconComponents['HOutline:SwatchIcon']" aria-hidden="true" />
          </button>
          <button
            type="button"
            class="auth-brand__btn auth-brand__btn--mode"
            :aria-label="t('login.toggleTheme')"
            @click="toggleTheme"
          >
            <component
              :is="menuStore.iconComponents[themeStore.isDarkTheme ? 'HOutline:SunIcon' : 'HOutline:MoonIcon']"
              aria-hidden="true"
            />
          </button>
        </div>
      </div>

      <div class="auth-brand__copy">
        <h1>
          {{ hero.line1 }}<br />
          {{ hero.line2 }}
        </h1>
        <p>{{ hero.desc }}</p>
      </div>

      <ul class="auth-brand__tags" aria-hidden="true">
        <li v-for="(tag, i) in heroTags" :key="tag" :class="`is-${i % 5}`">{{ tag }}</li>
      </ul>
      <span v-if="hero.sticker" class="auth-brand__msticker" aria-hidden="true">{{ hero.sticker }}</span>
    </section>

    <section class="auth-panel">
      <div class="auth-panel__chrome">
        <AuthLangMenu />
        <AppIconButton
          v-if="APP_CONFIG.showThemeConfig"
          icon="HOutline:SwatchIcon"
          :box="44"
          :label="t('layout.themeConfig')"
          @click="themeStore.themeConfigDrawerOpen = true"
        />
        <AppIconButton
          :icon="themeStore.isDarkTheme ? 'HOutline:SunIcon' : 'HOutline:MoonIcon'"
          :box="44"
          :label="t('login.toggleTheme')"
          @click="toggleTheme"
        />
      </div>

      <div class="auth-panel__center">
        <header class="auth-head">
          <div class="auth-head__row">
            <h2 class="auth-head__title">{{ formHead.title }}</h2>
            <button
              v-if="loginMode === 'login' && loginConfig.registerEnabled"
              type="button"
              class="auth-head__reg"
              @click="goToMode('register')"
            >
              {{ t('authPage.login.registerShort') }}
            </button>
          </div>
          <p class="auth-head__sub">{{ formHead.sub }}</p>
        </header>

        <AuthSeg
          v-if="loginMode !== 'login' && loginConfig.registerEnabled"
          v-model="altMode"
          class="auth-modes"
          :label="t('authPage.modes.label')"
          :options="[
            { value: 'register', label: t('authPage.modes.register') },
            { value: 'forgot', label: t('authPage.modes.reset') },
          ]"
        />

        <Transition name="auth-fade" mode="out-in">
          <AccountLogin
            v-if="loginMode === 'login'"
            key="login"
            :username-login-enabled="loginConfig.usernameLoginEnabled"
            :email-login-enabled="loginConfig.emailLoginEnabled"
            :register-enabled="loginConfig.registerEnabled"
            :remember-days="loginConfig.refreshTokenTtlDays"
            @go-to-mode="goToMode"
          />
          <ForgotPassword
            v-else-if="loginMode === 'forgot'"
            key="forgot"
            :mail-enabled="mailEnabled"
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

      <p class="auth-panel__copy">{{ t('authPage.footer', { name: APP_CONFIG.name }) }}</p>
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
import AuthLangMenu from '@/views/login/components/AuthLangMenu.vue'
import AuthSeg from '@/views/login/components/AuthSeg.vue'
import ThemeConfig from '@/components/ThemeConfig.vue'
import { i18n } from '@/locales'
import { APP_LOCALES, type AppLocale } from '@/locales/types'
import type { LoginConfig } from '@/types/v1/system'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'LoginView' })

type LoginMode = 'login' | 'forgot' | 'register'

const themeStore = useThemeStore()
const menuStore = useMenuStore()
const route = useRoute()
const router = useRouter()
const { t } = useI18n()

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
const siteName = ref('')
const mailEnabled = ref(true)
const configLoaded = ref(false)

const brandName = computed(() => siteName.value || APP_CONFIG.name)

// 模式跟随地址栏 ?mode=，切换用 replace 以免堆历史；注册关闭时注册模式退回登录。
const loginMode = computed<LoginMode>(() => {
  const raw = Array.isArray(route.query.mode) ? route.query.mode[0] : route.query.mode
  if (raw === 'forgot') return 'forgot'
  if (raw === 'register' && (loginConfig.registerEnabled || !configLoaded.value)) return 'register'
  return 'login'
})

const goToMode = (mode: LoginMode) => {
  if (mode === 'register' && !loginConfig.registerEnabled) return
  const query = { ...route.query }
  if (mode === 'login') delete query.mode
  else query.mode = mode
  void router.replace({ query })
}

const altMode = computed<'register' | 'forgot'>({
  get: () => (loginMode.value === 'register' ? 'register' : 'forgot'),
  set: (v) => goToMode(v),
})

const hero = computed(() => {
  if (loginMode.value === 'register') {
    return {
      line1: t('authPage.hero.register.line1', { name: brandName.value }),
      line2: t('authPage.hero.register.line2'),
      desc: t('authPage.hero.register.desc'),
      sticker: t('authPage.hero.register.sticker'),
    }
  }
  if (loginMode.value === 'forgot') {
    return {
      line1: t('authPage.hero.reset.line1'),
      line2: t('authPage.hero.reset.line2'),
      desc: t('authPage.hero.reset.desc'),
      sticker: t('authPage.hero.reset.sticker'),
    }
  }
  return { line1: t('login.heroLine1'), line2: t('login.heroLine2'), desc: t('login.heroDesc'), sticker: '' }
})

const heroTags = computed(() => t('login.heroTags').split(',').filter(Boolean))

const formHead = computed(() => {
  if (loginMode.value === 'register') {
    return { title: t('authPage.register.title'), sub: t('authPage.register.subtitle') }
  }
  if (loginMode.value === 'forgot') {
    return { title: t('authPage.reset.title'), sub: t('authPage.reset.subtitle') }
  }
  return { title: t('login.welcomeBack'), sub: t('authPage.login.subtitle', { name: brandName.value }) }
})

const toggleTheme = () => {
  themeStore.toggleThemeMode(themeStore.isDarkTheme ? 'light' : 'dark')
}

// 站点默认语言：只在访客还没自己选过语言时生效，不写入本地偏好。
const applyDefaultLanguage = (lang: string) => {
  let stored: string | null = null
  try {
    stored = localStorage.getItem('app-locale')
  } catch {
    stored = null
  }
  if (stored || !APP_LOCALES.includes(lang as AppLocale)) return
  i18n.global.locale.value = lang as AppLocale
  document.documentElement.lang = lang
}

onMounted(async () => {
  try {
    const { data } = await getLoginConfig()
    if (data?.config) Object.assign(loginConfig, data.config)
    siteName.value = data?.siteName?.trim() || ''
    mailEnabled.value = data?.mailEnabled ?? true
    if (data?.defaultLanguage) applyDefaultLanguage(data.defaultLanguage)
  } catch {
    // 使用默认值
  } finally {
    configLoaded.value = true
    if (route.query.mode === 'register' && !loginConfig.registerEnabled) goToMode('login')
  }
})
</script>

<style scoped lang="scss">
.auth {
  position: relative;
  display: flex;
  flex-wrap: wrap;
  min-height: 100vh;
  min-height: 100dvh;
  width: 100%;
  background: var(--el-bg-color);
}

/* ---- 品牌区 ---- */
.auth-brand {
  position: relative;
  flex: 1 1 560px;
  min-height: 100vh;
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 40px;
  padding: 48px 56px;
  overflow: hidden;
  background: var(--m-p);
  color: var(--m-pOn);
}
.auth-brand__blob {
  position: absolute;
  right: -120px;
  bottom: -160px;
  width: 460px;
  height: 460px;
  border-radius: 50%;
  background: var(--m-a);
}
.auth-brand__ring {
  position: absolute;
  left: 46%;
  top: 54%;
  width: 150px;
  height: 150px;
  border-radius: 50%;
  border: 22px solid var(--m-a2);
}
.auth-brand__star {
  position: absolute;
  right: 80px;
  top: 120px;
  width: 64px;
  height: 64px;
  fill: var(--m-pOn);
}
.auth-brand__bar {
  position: relative;
  display: flex;
  align-items: center;
  gap: 8px;
}
.auth-brand__logo {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  font-family: var(--app-font-display);
  font-size: 1.625rem;
  font-weight: 800;
}
.auth-brand__logo span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.auth-brand__logo img {
  flex: none;
  width: 48px;
  height: 48px;
  border-radius: 15px;
  object-fit: cover;
  transform: rotate(-6deg);
  box-shadow: 0 0 0 3px var(--m-pOn);
}
.auth-brand__tools,
.auth-brand__bar .auth-brand__back,
.auth-brand__msticker {
  display: none;
}
.auth-brand__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 44px;
  height: 44px;
  border: 0;
  border-radius: 14px;
  background: transparent;
  color: var(--m-pOn);
  cursor: pointer;
  transition: background-color 0.15s;
}
.auth-brand__btn:hover {
  background: color-mix(in srgb, var(--m-pOn) 14%, transparent);
}
.auth-brand__btn svg {
  width: 20px;
  height: 20px;
}
/* 主题按钮：实心底 + 主色图标（与语言按钮一致），压在黄色圆上也看得清 */
.auth-brand__btn--chip,
.auth-brand__btn--chip:hover {
  background: var(--m-pOn);
  color: var(--m-p);
}
.auth-brand__btn--mode,
.auth-brand__btn--mode:hover {
  background: var(--m-a);
  color: var(--m-aOn);
}
.auth-brand__copy {
  position: relative;
  max-width: 560px;
}
.auth-brand__copy h1 {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: clamp(2.5rem, 4.2vw, 3.625rem);
  line-height: 1.08;
  font-weight: 800;
  letter-spacing: -0.02em;
}
.auth-brand__copy p {
  margin: 18px 0 0;
  font-size: 1.125rem;
  line-height: 1.6;
}
.auth-brand__tags {
  position: relative;
  display: flex;
  flex-wrap: wrap;
  gap: 14px 12px;
  max-width: 440px;
  min-height: 120px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.auth-brand__tags li {
  display: inline-flex;
  align-items: center;
  height: 48px;
  padding: 0 20px;
  border-radius: 999px;
  font-size: 1.0625rem;
  font-weight: 800;
  white-space: nowrap;
  box-shadow: 0 10px 24px -12px color-mix(in srgb, var(--m-ink) 60%, transparent);
}
.auth-brand__tags .is-0 {
  background: var(--m-pOn);
  color: var(--m-p);
  transform: rotate(-5deg);
}
.auth-brand__tags .is-1 {
  background: var(--m-ink);
  color: var(--m-bg);
  transform: translateY(14px) rotate(4deg);
}
.auth-brand__tags .is-2 {
  background: var(--m-a2);
  color: #fff;
  transform: rotate(3deg);
}
.auth-brand__tags .is-3 {
  background: var(--m-aSoft);
  color: var(--m-aOn);
  transform: translateY(-6px) rotate(6deg);
}
/* 深色下黄色淡底很暗，文字改用强调色本身（同 .m-sticker--a） */
html.dark .auth-brand__tags .is-3 {
  color: var(--m-a);
}
.auth-brand__tags .is-4 {
  background: var(--m-pOn);
  color: var(--m-p);
  transform: translateY(8px) rotate(-3deg);
}

/* ---- 表单区 ---- */
.auth-panel {
  position: relative;
  flex: 1 1 480px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  padding: 28px 32px 40px;
  background: var(--el-bg-color);
}
.auth-panel__chrome {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 6px;
}
.auth-panel__center {
  flex: 1;
  width: min(100%, 400px);
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 20px;
  padding: 24px 0;
}
.auth-panel__copy {
  margin: 0;
  text-align: center;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}

.auth-head {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.auth-head__row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
}
.auth-head__title {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 2.25rem;
  line-height: 1.15;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--el-text-color-primary);
}
.auth-head__sub {
  margin: 0;
  font-size: 0.9375rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.auth-head__reg {
  display: none;
  min-height: 44px;
  padding: 0 4px;
  border: 0;
  background: transparent;
  color: var(--m-pInk);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 700;
  cursor: pointer;
}

.auth-fade-enter-active,
.auth-fade-leave-active {
  transition:
    opacity 0.16s ease,
    transform 0.16s ease;
}
.auth-fade-enter-from {
  opacity: 0;
  transform: translateY(4px);
}
.auth-fade-leave-to {
  opacity: 0;
  transform: translateY(-3px);
}

/* ---- 手机：品牌区在上（返回 / Logo / 工具按钮 + 标题 + 贴纸），表单卡片上浮，主按钮贴底 ---- */
@media (width < 900px) {
  .auth {
    flex-direction: column;
    flex-wrap: nowrap;
    overflow-x: clip;
  }
  .auth-brand {
    flex: none;
    min-height: 330px;
    justify-content: flex-start;
    gap: 0;
    padding: 16px 16px 64px 20px;
  }
  .auth--register .auth-brand,
  .auth--forgot .auth-brand {
    min-height: 214px;
    padding-bottom: 60px;
  }
  .auth-brand__blob {
    right: -70px;
    top: -70px;
    bottom: auto;
    width: 220px;
    height: 220px;
  }
  .auth-brand__ring {
    left: 200px;
    top: 150px;
    width: 70px;
    height: 70px;
    border-width: 12px;
  }
  .auth--register .auth-brand__ring,
  .auth--forgot .auth-brand__ring {
    left: 230px;
    top: 112px;
    width: 62px;
    height: 62px;
    border-width: 11px;
  }
  .auth-brand__star,
  .auth-brand__copy p {
    display: none;
  }
  .auth-brand__bar {
    gap: 6px;
  }
  .auth-brand__bar .auth-brand__back {
    display: inline-flex;
    margin-left: -10px;
  }
  .auth-brand__logo {
    flex: 1;
    gap: 10px;
    font-size: 1.3125rem;
  }
  .auth-brand__logo img {
    width: 40px;
    height: 40px;
    border-radius: 13px;
  }
  .auth--register .auth-brand__logo img,
  .auth--forgot .auth-brand__logo img {
    width: 36px;
    height: 36px;
    border-radius: 12px;
  }
  .auth-brand__tools {
    display: flex;
    align-items: center;
    gap: 2px;
    flex: none;
  }
  .auth-brand__copy {
    margin-top: 34px;
  }
  .auth--register .auth-brand__copy,
  .auth--forgot .auth-brand__copy {
    margin-top: 22px;
  }
  .auth-brand__copy h1 {
    font-size: 2.125rem;
    line-height: 1.12;
    letter-spacing: -0.01em;
  }
  .auth--register .auth-brand__copy h1,
  .auth--forgot .auth-brand__copy h1 {
    font-size: 1.875rem;
    line-height: 1.15;
  }
  .auth-brand__tags {
    position: absolute;
    left: 20px;
    right: 16px;
    bottom: 76px;
    flex-wrap: nowrap;
    gap: 6px;
    min-height: 0;
    max-width: none;
  }
  .auth-brand__tags li {
    height: 36px;
    padding: 0 14px;
    font-size: 0.875rem;
    box-shadow: none;
  }
  .auth-brand__tags li:nth-child(n + 4) {
    display: none;
  }
  .auth-brand__tags .is-1 {
    transform: translateY(6px) rotate(4deg);
  }
  .auth-brand__tags .is-2 {
    transform: translateY(-4px) rotate(-3deg);
  }
  .auth--register .auth-brand__tags,
  .auth--forgot .auth-brand__tags {
    display: none;
  }
  .auth--register .auth-brand__msticker,
  .auth--forgot .auth-brand__msticker {
    position: absolute;
    right: 20px;
    bottom: 50px;
    display: inline-flex;
    align-items: center;
    height: 36px;
    padding: 0 14px;
    border-radius: 999px;
    background: var(--m-ink);
    color: var(--m-bg);
    font-size: 0.875rem;
    font-weight: 800;
    white-space: nowrap;
    transform: rotate(5deg);
  }

  .auth-panel {
    position: relative;
    z-index: 1;
    flex: 1 0 auto;
    margin-top: -36px;
    padding: 26px 20px 0;
    border-radius: 32px 32px 0 0;
  }
  .auth-panel__chrome,
  .auth-panel__copy {
    display: none;
  }
  .auth-panel__center {
    width: 100%;
    justify-content: flex-start;
    gap: 16px;
    padding: 0;
  }
  .auth-head__title {
    font-size: 1.75rem;
  }
  .auth-head__sub {
    display: none;
  }
  .auth-head__reg {
    display: inline-flex;
    align-items: center;
  }
  /* 注册 / 找回：手机上不显示标题，分段切换在最上 */
  .auth--register .auth-head,
  .auth--forgot .auth-head {
    display: none;
  }
}
</style>
