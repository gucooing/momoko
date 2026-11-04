<!-- 404 / 403 共用场景：大号数字 + 中间贴纸插画（404 黄色笑脸 / 403 主色锁）+ 标题说明 + 详情胶囊 + 返回按钮。
     独立页面（无外壳）时带顶栏（Logo / 语言 / 深浅色）与底部错误代码行；
     菜单视图缺失被兜底放进外壳时，只在内容区居中显示场景。背景用淡色圆、圆环、星星点缀（不用渐变）。 -->
<template>
  <div class="err" :class="[`err--${code}`, standalone ? 'err--page' : 'err--inset']">
    <span class="err__blob" aria-hidden="true" />
    <span class="err__ring" aria-hidden="true" />
    <svg class="err__star err__star--a" viewBox="0 0 24 24" aria-hidden="true">
      <path :d="STAR" />
    </svg>
    <svg class="err__star err__star--p" viewBox="0 0 24 24" aria-hidden="true">
      <path :d="STAR" />
    </svg>

    <header v-if="standalone" class="err__top">
      <RouterLink to="/" class="err__brand" :aria-label="t('errorPage.homeLabel')">
        <img :src="APP_CONFIG.logoSrc" alt="" />
        <span>{{ APP_CONFIG.name }}</span>
      </RouterLink>
      <span class="err__spacer" />
      <span class="err__lang"><AuthLangMenu /></span>
      <span class="err__lang-m"><AuthLangMenu compact /></span>
      <AppIconButton
        :icon="themeStore.isDarkTheme ? 'HOutline:SunIcon' : 'HOutline:MoonIcon'"
        :box="44"
        :label="t('login.toggleTheme')"
        @click="toggleTheme"
      />
    </header>

    <main class="err__main">
      <div class="err__art" role="img" :aria-label="t('errorPage.artLabel', { code })">
        <span class="err__digit err__digit--first" aria-hidden="true">4</span>

        <span v-if="is404" class="err__face" aria-hidden="true">
          <span class="err__face-disc" />
          <span class="err__face-ring" />
          <span class="err__face-eye err__face-eye--l" />
          <span class="err__face-eye err__face-eye--r" />
          <svg class="err__face-mouth" viewBox="0 0 60 24"><path d="M6 18c8-10 40-10 48 0" /></svg>
          <span class="err__face-q">?</span>
        </span>
        <span v-else class="err__lock" aria-hidden="true">
          <span class="err__lock-disc" />
          <span class="err__lock-tile">
            <svg viewBox="0 0 24 24"><path d="M5 11h14v10H5zM8 11V7a4 4 0 0 1 8 0v4M12 15v2.5" /></svg>
          </span>
          <span class="err__lock-ring" />
        </span>

        <span class="err__digit err__digit--last" aria-hidden="true">{{ is404 ? '4' : '3' }}</span>
        <span class="err__tag err__tag--word" aria-hidden="true">{{ sticker }}</span>
        <span class="err__tag err__tag--code m-mono" aria-hidden="true">HTTP {{ code }}</span>
      </div>

      <div class="err__copy">
        <h1 class="err__title">{{ title }}</h1>
        <p class="err__desc">{{ desc }}</p>
        <div v-if="detailValue" class="err__detail">
          <component :is="menuStore.iconComponents[detailIcon]" class="err__detail-ico" aria-hidden="true" />
          <span class="err__detail-key">{{ detailLabel }}</span>
          <span class="err__detail-val m-mono" :title="detailValue">{{ detailValue }}</span>
        </div>
      </div>

      <div class="err__actions">
        <button v-if="canGoHome" type="button" class="err__btn err__btn--primary" @click="goHome">
          <component :is="menuStore.iconComponents['HOutline:HomeIcon']" aria-hidden="true" />
          {{ t('errorPage.goHome') }}
        </button>
        <button type="button" class="err__btn err__btn--ghost" @click="goBack">
          <component :is="menuStore.iconComponents['HOutline:ArrowLeftIcon']" aria-hidden="true" />
          {{ t('errorPage.goBack') }}
        </button>
        <p v-if="!is404 && hasToken" class="err__switch">
          {{ t('errorPage.switchHint') }}
          <button type="button" class="err__link" @click="switchAccount">
            {{ t('errorPage.switchAccount') }}
          </button>
        </p>
      </div>
    </main>

    <footer v-if="standalone" class="err__foot">
      {{ t('errorPage.footer', { code, time: occurredAt }) }}
    </footer>
  </div>
</template>

<script setup lang="ts">
import { APP_CONFIG } from '@/config/app.config'
import AuthLangMenu from '@/views/login/components/AuthLangMenu.vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ code: '403' | '404' }>()

const STAR = 'M12 1.5l2.6 7.9 7.9 2.6-7.9 2.6L12 22.5l-2.6-7.9L1.5 12l7.9-2.6z'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const menuStore = useMenuStore()
const userStore = useUserStore()
const themeStore = useThemeStore()

const pad = (n: number) => String(n).padStart(2, '0')
const now = new Date()
const occurredAt = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}`

/** 不在外壳里（独立错误页）时显示自己的顶栏与底部信息 */
const standalone = computed(() => !route.matched.some((r) => r.name === 'layout'))
const hasToken = (() => {
  try {
    return !!localStorage.getItem('accessToken')
  } catch {
    return false
  }
})()

const is404 = computed(() => props.code === '404')
const noMenu = computed(() => menuStore.hasLoadedPermissions && !menuStore.menuList?.length)
/** 一个可访问页面都没有时，「返回工作台」只会再回到这里，不显示 */
const canGoHome = computed(() => is404.value || !noMenu.value)

const title = computed(() => (is404.value ? t('errorPage.notFound.title') : t('errorPage.forbidden.title')))
const desc = computed(() => {
  if (is404.value) return t('errorPage.notFound.desc')
  return noMenu.value ? t('errorPage.forbidden.descNoMenu') : t('errorPage.forbidden.desc')
})
const sticker = computed(() => (is404.value ? t('errorPage.notFound.sticker') : t('errorPage.forbidden.sticker')))
const detailIcon = computed(() => (is404.value ? 'HOutline:LinkIcon' : 'HOutline:UserIcon'))
const detailLabel = computed(() => (is404.value ? t('errorPage.notFound.detail') : t('errorPage.forbidden.detail')))
const detailValue = computed(() => {
  if (is404.value) return route.fullPath || ''
  return userStore.userInfo ? userStore.getUserRoleName() : ''
})

function toggleTheme() {
  themeStore.toggleThemeMode(themeStore.isDarkTheme ? 'light' : 'dark')
}

function goHome() {
  router.replace({ path: '/' })
}

function goBack() {
  if (window.history.length > 1) router.back()
  else goHome()
}

function switchAccount() {
  void userStore.logout()
}

onMounted(() => {
  // 独立 403 页不经过外壳，自己补拉一次当前用户（用于显示当前角色）
  if (!is404.value && hasToken && !userStore.userInfo) {
    userStore.getUserInfo().catch(() => undefined)
  }
})
</script>

<style scoped lang="scss">
.err {
  position: relative;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.err--page {
  min-height: 100vh;
  min-height: 100dvh;
  background: var(--m-bg);
  color: var(--m-ink);
}
.err--inset {
  flex: 1;
  min-height: max(520px, calc(100dvh - 160px));
  justify-content: center;
  padding: 32px 16px 48px;
  border-radius: var(--app-radius-xl);
}

/* ---- 装饰 ---- */
.err__blob {
  position: absolute;
  right: -150px;
  bottom: -190px;
  width: 480px;
  height: 480px;
  border-radius: 50%;
  background: var(--m-pSoft);
}
.err__ring {
  position: absolute;
  left: 7%;
  top: 22%;
  width: 96px;
  height: 96px;
  border-radius: 50%;
  border: 16px solid var(--m-a2Soft);
}
.err__star {
  position: absolute;
}
.err__star--a {
  right: 14%;
  top: 18%;
  width: 48px;
  height: 48px;
  fill: var(--m-a);
}
.err__star--p {
  left: 16%;
  bottom: 20%;
  width: 28px;
  height: 28px;
  fill: var(--m-p);
}

/* ---- 顶栏 / 底栏（独立页） ---- */
.err__top {
  position: relative;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 24px 40px;
}
.err__brand {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  min-height: 44px;
  min-width: 0;
  color: var(--m-ink);
  text-decoration: none;
  font-family: var(--app-font-display);
  font-size: 1.375rem;
  font-weight: 800;
}
.err__brand img {
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: 13px;
  object-fit: cover;
  transform: rotate(-6deg);
}
.err__brand span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.err__spacer {
  flex: 1;
}
.err__lang {
  display: inline-flex;
}
.err__lang-m {
  display: none;
}
.err__foot {
  position: relative;
  padding: 0 40px 24px;
  text-align: center;
  font-size: 0.75rem;
  color: var(--m-ink2);
  font-variant-numeric: tabular-nums;
}

.err__main {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 30px;
  max-width: 100%;
  text-align: center;
}
.err--page .err__main {
  flex: 1;
  padding: 0 40px 56px;
}

/* ---- 数字插画 ---- */
.err__art {
  position: relative;
  display: flex;
  align-items: center;
  gap: 18px;
}
.err__digit {
  font-family: var(--app-font-display);
  font-size: clamp(160px, 18vw, 260px);
  line-height: 1;
  font-weight: 800;
  letter-spacing: -0.04em;
  font-variant-numeric: tabular-nums;
  color: var(--m-ink);
}
.err__digit--first {
  transform: rotate(-4deg);
}
.err__digit--last {
  color: var(--m-p);
  transform: rotate(5deg);
}
.err--403 .err__digit--last {
  color: var(--m-a2);
}

.err__face,
.err__lock {
  position: relative;
  flex: none;
  width: 220px;
  height: 220px;
}
.err__face-disc {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  background: var(--m-a);
}
.err__face-ring {
  position: absolute;
  right: -26px;
  bottom: -14px;
  width: 92px;
  height: 92px;
  border-radius: 50%;
  border: 16px solid var(--m-a2);
}
.err__face-eye {
  position: absolute;
  top: 86px;
  width: 20px;
  height: 26px;
  border-radius: 50%;
  background: var(--m-aOn);
}
.err__face-eye--l {
  left: 66px;
}
.err__face-eye--r {
  right: 66px;
}
.err__face-mouth {
  position: absolute;
  left: 80px;
  top: 138px;
  width: 60px;
  height: 24px;
  fill: none;
  stroke: var(--m-aOn);
  stroke-width: 5;
  stroke-linecap: round;
}
.err__face-q {
  position: absolute;
  right: 4px;
  top: -10px;
  width: 58px;
  height: 58px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 18px;
  background: var(--m-ink);
  color: var(--m-bg);
  font-family: var(--app-font-display);
  font-size: 34px;
  font-weight: 800;
  transform: rotate(12deg);
}

.err__lock-disc {
  position: absolute;
  left: -16px;
  top: 20px;
  width: 120px;
  height: 120px;
  border-radius: 50%;
  background: var(--m-a);
}
.err__lock-tile {
  position: absolute;
  inset: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 56px;
  background: var(--m-p);
  color: var(--m-pOn);
  transform: rotate(-8deg);
  box-shadow: var(--app-shadow-brand);
}
.err__lock-tile svg {
  width: 104px;
  height: 104px;
  fill: none;
  stroke: currentColor;
  stroke-width: 2.2;
  stroke-linecap: round;
  stroke-linejoin: round;
}
.err__lock-ring {
  position: absolute;
  right: -22px;
  bottom: -6px;
  width: 84px;
  height: 84px;
  border-radius: 50%;
  border: 15px solid var(--m-a2);
}

.err__tag {
  position: absolute;
  display: inline-flex;
  align-items: center;
  height: 46px;
  padding: 0 18px;
  border-radius: 999px;
  font-size: 1.0625rem;
  font-weight: 800;
  white-space: nowrap;
  box-shadow: 0 10px 24px -12px color-mix(in srgb, var(--m-ink) 40%, transparent);
}
.err__tag--word {
  left: -70px;
  top: 18px;
  background: var(--m-p);
  color: var(--m-pOn);
  transform: rotate(-9deg);
}
.err--403 .err__tag--word {
  background: var(--m-ink);
  color: var(--m-bg);
}
.err__tag--code {
  right: -40px;
  bottom: 6px;
  background: var(--m-s);
  color: var(--m-ink);
  border: 1.5px solid var(--m-line);
  font-size: 0.875rem;
  transform: rotate(6deg);
}

/* ---- 文案 ---- */
.err__copy {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  max-width: 620px;
}
.err__title {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 2.625rem;
  line-height: 1.2;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--m-ink);
}
.err__desc {
  margin: 0;
  font-size: 1.0625rem;
  line-height: 1.65;
  color: var(--m-ink2);
}
.err__detail {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  max-width: 100%;
  margin-top: 4px;
  padding: 8px 14px;
  border-radius: 14px;
  background: var(--m-s2);
  font-size: 0.8125rem;
  color: var(--m-ink2);
}
.err__detail-ico {
  width: 16px;
  height: 16px;
  flex: none;
}
.err__detail-key {
  white-space: nowrap;
}
.err__detail-val {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 700;
  color: var(--m-ink);
}

.err__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 12px;
}
.err__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 56px;
  padding: 0 28px;
  border-radius: 999px;
  font: inherit;
  font-size: 1rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
  transition:
    filter 0.15s,
    transform 0.15s,
    border-color 0.15s,
    color 0.15s;
}
.err__btn:active {
  transform: scale(0.98);
}
.err__btn svg {
  width: 20px;
  height: 20px;
}
.err__btn--primary {
  border: 0;
  background: var(--m-p);
  color: var(--m-pOn);
  box-shadow: var(--app-shadow-brand);
}
.err__btn--primary:hover {
  filter: brightness(1.08);
}
.err__btn--ghost {
  border: 1.5px solid var(--m-line);
  background: var(--m-s);
  color: var(--m-ink);
}
.err__btn--ghost:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.err__switch {
  flex-basis: 100%;
  margin: 0;
  font-size: 0.875rem;
  color: var(--m-ink2);
}
.err__link {
  min-height: 40px;
  padding: 0 4px;
  border: 0;
  background: transparent;
  font: inherit;
  font-weight: 700;
  color: var(--m-pInk);
  cursor: pointer;
}
.err__link:hover {
  text-decoration: underline;
  text-underline-offset: 3px;
}

/* ---- 手机 ---- */
@media (width < 720px) {
  .err__link {
    min-height: 44px;
  }
  .err--inset {
    min-height: max(480px, calc(100dvh - 200px));
    padding: 24px 4px 36px;
    justify-content: flex-start;
  }
  .err__blob {
    right: -110px;
    bottom: -120px;
    width: 300px;
    height: 300px;
  }
  .err__ring {
    left: -24px;
    top: 150px;
    width: 70px;
    height: 70px;
    border-width: 12px;
  }
  .err__star--a {
    right: 34px;
    top: 136px;
    width: 32px;
    height: 32px;
  }
  .err__star--p {
    display: none;
  }
  .err__top {
    gap: 4px;
    padding: 16px 12px 8px 20px;
  }
  .err__brand {
    gap: 10px;
    font-size: 1.25rem;
  }
  .err__brand img {
    width: 36px;
    height: 36px;
    border-radius: 12px;
  }
  .err__lang {
    display: none;
  }
  .err__lang-m {
    display: inline-flex;
  }
  .err__main {
    gap: 22px;
    width: 100%;
  }
  .err--page .err__main {
    padding: 0 24px;
  }
  .err--inset .err__main {
    padding-top: 28px;
  }
  .err__art {
    gap: 8px;
    margin-top: 10px;
  }
  .err__digit {
    font-size: 132px;
  }
  .err__face,
  .err__lock {
    width: 114px;
    height: 114px;
  }
  .err__face-ring {
    right: -14px;
    bottom: -8px;
    width: 48px;
    height: 48px;
    border-width: 9px;
  }
  .err__face-eye {
    top: 44px;
    width: 11px;
    height: 14px;
  }
  .err__face-eye--l {
    left: 34px;
  }
  .err__face-eye--r {
    right: 34px;
  }
  .err__face-mouth {
    left: 41px;
    top: 72px;
    width: 32px;
    height: 13px;
    stroke-width: 6;
  }
  .err__face-q {
    right: 0;
    top: -8px;
    width: 32px;
    height: 32px;
    border-radius: 10px;
    font-size: 19px;
  }
  .err__lock-disc {
    left: -8px;
    top: 10px;
    width: 62px;
    height: 62px;
  }
  .err__lock-tile {
    inset: 7px;
    border-radius: 30px;
    box-shadow: none;
  }
  .err__lock-tile svg {
    width: 54px;
    height: 54px;
    stroke-width: 2.3;
  }
  .err__lock-ring {
    right: -12px;
    bottom: -4px;
    width: 44px;
    height: 44px;
    border-width: 8px;
  }
  .err__tag {
    height: 34px;
    padding: 0 13px;
    font-size: 0.875rem;
    box-shadow: 0 8px 20px -10px color-mix(in srgb, var(--m-ink) 40%, transparent);
  }
  .err__tag--word {
    left: -14px;
    top: -22px;
  }
  .err__tag--code {
    display: none;
  }
  .err__copy {
    gap: 10px;
  }
  .err__title {
    font-size: 1.625rem;
    line-height: 1.3;
  }
  .err__desc {
    font-size: 0.875rem;
  }
  .err__detail {
    gap: 8px;
    padding: 8px 12px;
    border-radius: 12px;
    font-size: 0.75rem;
  }
  /* 独立页：按钮组放在底部 */
  .err--page .err__main {
    justify-content: flex-start;
  }
  .err--page .err__art {
    margin-top: auto;
  }
  .err--page .err__copy {
    margin-bottom: auto;
  }
  .err__actions {
    flex-direction: column;
    flex-wrap: nowrap;
    align-self: stretch;
    gap: 10px;
  }
  .err--page .err__actions {
    margin: 0 -8px;
    padding: 12px 0 8px;
  }
  .err__btn {
    width: 100%;
  }
  .err__btn--ghost {
    min-height: 52px;
    font-size: 0.9375rem;
  }
  .err__switch {
    line-height: 32px;
    font-size: 0.8125rem;
  }
  .err__foot {
    padding: 0 16px calc(20px + env(safe-area-inset-bottom));
    line-height: 32px;
  }
}
</style>
