<!-- OIDC 授权确认页（authOnly）：主色顶带 + 居中卡片（应用 ⇄ Momoko 贴纸、标题、当前账号、请求的权限、回调地址、允许/拒绝）。
     不承载后台外壳；授权成功后跳回第三方 redirect_uri。 -->
<template>
  <div class="oa">
    <div class="oa-band" aria-hidden="true">
      <span class="oa-band__blob" />
      <span class="oa-band__ring" />
      <svg class="oa-band__star" viewBox="0 0 24 24">
        <path d="M12 1.5l2.6 7.9 7.9 2.6-7.9 2.6L12 22.5l-2.6-7.9L1.5 12l7.9-2.6z" />
      </svg>
    </div>

    <header class="oa-top">
      <img :src="APP_CONFIG.logoSrc" alt="" class="oa-top__logo" />
      <span class="oa-top__name">{{ APP_CONFIG.name }}</span>
      <span class="oa-top__sub">{{ t('publicPage.oidc.badge') }}</span>
      <span class="oa-top__spacer" />
      <div class="oa-top__chrome">
        <LanguageMenu />
        <AppIconButton
          :icon="themeStore.isDarkTheme ? 'HOutline:SunIcon' : 'HOutline:MoonIcon'"
          :label="t('login.toggleTheme')"
          @click="toggleTheme"
        />
      </div>
    </header>

    <main class="oa-main">
      <section class="oa-card" aria-labelledby="oa-title">
        <!-- 加载 -->
        <div v-if="infoLoading" class="oa-skel">
          <span class="oa-skel__line oa-skel__line--lg" />
          <span class="oa-skel__line" />
          <span class="oa-skel__line oa-skel__line--sm" />
        </div>

        <!-- 授权请求无效 -->
        <div v-else-if="loadError" class="oa-invalid">
          <span class="m-sticker m-sticker--bad oa-invalid__ico">
            <component :is="menuStore.iconComponents['HOutline:ShieldExclamationIcon']" />
          </span>
          <h1 id="oa-title">{{ t('oidc.authorize.invalidRequest') }}</h1>
          <p>{{ loadError }}</p>
          <UButton color="neutral" variant="outline" class="oa-btn oa-btn--outline" icon="i-lucide-arrow-left" @click="goBack">
            {{ t('oidc.authorize.back') }}
          </UButton>
        </div>

        <!-- 授权确认 -->
        <template v-else>
          <div class="oa-pair" aria-hidden="true">
            <span class="oa-pair__app">{{ clientInitial }}</span>
            <span class="oa-pair__dots"><i /><i /><i /></span>
            <span class="oa-pair__shield">
              <component :is="menuStore.iconComponents['HOutline:ShieldCheckIcon']" />
            </span>
            <span class="oa-pair__dots"><i /><i /><i /></span>
            <img :src="APP_CONFIG.logoSrc" alt="" class="oa-pair__me" />
          </div>

          <div class="oa-head">
            <h1 id="oa-title">{{ t('publicPage.oidc.title', { client: clientName, name: APP_CONFIG.name }) }}</h1>
            <p v-if="authorizeInfo?.clientDescription" class="oa-head__desc">{{ authorizeInfo.clientDescription }}</p>
            <div class="oa-head__meta">
              <span v-if="redirectHost" class="m-mono oa-head__host">{{ redirectHost }}</span>
              <StatusPill variant="success" :label="t('publicPage.oidc.registered')" />
            </div>
          </div>

          <div class="oa-account">
            <AppAvatar :src="userStore.resolvedUserAvatar" :name="currentUserName" :size="44" />
            <span class="oa-account__text">
              <span class="oa-account__name">{{ currentUserName }}</span>
              <span class="oa-account__sub">{{ currentUserSub }}</span>
            </span>
            <button type="button" class="oa-account__switch" @click="switchAccount">
              {{ t('publicPage.oidc.switchAccount') }}
            </button>
          </div>

          <div class="oa-scopes">
            <div class="oa-scopes__title">{{ t('publicPage.oidc.willGet', { client: clientName }) }}</div>
            <ul>
              <li v-for="(s, i) in scopeItems" :key="s.key" class="oa-scope">
                <span
                  class="m-sticker oa-scope__ico"
                  :class="`m-sticker--${SCOPE_TONES[i % SCOPE_TONES.length]}`"
                  :style="{ rotate: SCOPE_ROTS[i % SCOPE_ROTS.length] }"
                >
                  <component :is="menuStore.iconComponents[s.icon]" />
                </span>
                <span class="oa-scope__text">
                  <span class="oa-scope__label">
                    {{ s.label }}
                    <code class="oa-scope__key">{{ s.key }}</code>
                  </span>
                  <span class="oa-scope__desc">{{ s.desc }}</span>
                </span>
                <component :is="menuStore.iconComponents['HOutline:CheckIcon']" class="oa-scope__check" />
              </li>
            </ul>
          </div>

          <div v-if="redirectTarget" class="oa-redirect">
            <component :is="menuStore.iconComponents['HOutline:ArrowTopRightOnSquareIcon']" />
            <span class="oa-redirect__text">
              <span class="oa-redirect__label">{{ t('publicPage.oidc.redirectTo') }}</span>
              <span class="oa-redirect__url m-mono" :title="redirectTarget">
                <b>{{ redirectOrigin }}</b>{{ redirectRest }}
              </span>
            </span>
          </div>

          <div class="oa-actions">
            <UButton
              color="neutral"
              variant="outline"
              class="oa-btn oa-btn--outline oa-btn--deny"
              :disabled="submitting"
              @click="deny"
            >
              {{ t('oidc.authorize.deny') }}
            </UButton>
            <UButton
              color="primary"
              class="oa-btn oa-btn--primary oa-btn--allow"
              trailing-icon="i-lucide-arrow-right"
              :loading="submitting"
              @click="approve"
            >
              {{ submitting ? t('publicPage.oidc.redirecting') : t('publicPage.oidc.allow') }}
            </UButton>
          </div>

          <p class="oa-note">{{ t('publicPage.oidc.noPassword', { name: APP_CONFIG.name, client: clientName }) }}</p>
        </template>
      </section>

      <p class="oa-foot">
        {{ t('publicPage.oidc.footer', { name: APP_CONFIG.name }) }}<template v-if="issuer"> · <span class="m-mono">{{ issuer }}</span></template>
      </p>
    </main>
  </div>
</template>

<script setup lang="ts">
import { createOIDCAuthorizationCode, getOIDCAuthorizationInfo } from '@/api/oidc'
import { logoutRequest } from '@/api/login'
import { APP_CONFIG } from '@/config/app.config'
import { useUserStore } from '@/stores/user'
import { getRequestErrorMessage } from '@/utils/request'
import { useFeedback } from '@/utils/feedback'
import LanguageMenu from '@/layouts/app/LanguageMenu.vue'
import type { OIDCAuthorizationInfo } from '@/types/v1/oidc'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'OIDCAuthorizeView' })

const { t } = useI18n()
const route = useRoute()
const userStore = useUserStore()
const themeStore = useThemeStore()
const menuStore = useMenuStore()
const fb = useFeedback()

const submitting = ref(false)
const infoLoading = ref(true)
const loadError = ref('')
const authorizeInfo = ref<OIDCAuthorizationInfo | null>(null)

const SCOPE_TONES = ['a2', 'p', 'a', 'ok']
const SCOPE_ROTS = ['-4deg', '3deg', '-2deg', '4deg']

const queryValue = (key: string) => {
  const value = route.query[key]
  return Array.isArray(value) ? value[0] || '' : String(value || '')
}

const responseType = computed(() => queryValue('response_type') || 'code')
const clientId = computed(() => queryValue('client_id'))
const redirectUri = computed(() => queryValue('redirect_uri'))
const scope = computed(() => queryValue('scope') || 'openid')
const state = computed(() => queryValue('state'))
const currentUserName = computed(
  () => userStore.userInfo?.name || userStore.userInfo?.username || t('oidc.authorize.currentAccount'),
)
const currentUserSub = computed(() => {
  const info = userStore.userInfo
  if (!info) return t('oidc.authorize.signedInAs')
  return info.email || (info.name && info.username ? `@${info.username}` : t('oidc.authorize.signedInAs'))
})
const clientName = computed(() => authorizeInfo.value?.clientName || t('oidc.authorize.unknownClient'))
const clientInitial = computed(() => [...(clientName.value || '?').trim()][0]?.toUpperCase() || '?')
const issuer = computed(() => authorizeInfo.value?.issuerUrl || '')

// 授权后将跳转到的第三方地址（安全信号：让用户看清 code 发往何处）。
const redirectTarget = computed(() => authorizeInfo.value?.redirectUri || redirectUri.value)
const parsedRedirect = computed(() => {
  try {
    return redirectTarget.value ? new URL(redirectTarget.value) : null
  } catch {
    return null
  }
})
const redirectHost = computed(() => parsedRedirect.value?.host || '')
const redirectOrigin = computed(() => parsedRedirect.value?.origin || redirectTarget.value)
const redirectRest = computed(() => (parsedRedirect.value ? parsedRedirect.value.pathname : ''))

// 已知 OIDC scope → 图标 + 友好文案；未知 scope 原样展示为标签。
const SCOPE_META: Record<string, { icon: string; key: string }> = {
  openid: { icon: 'HOutline:IdentificationIcon', key: 'openid' },
  profile: { icon: 'HOutline:UserCircleIcon', key: 'profile' },
  email: { icon: 'HOutline:EnvelopeIcon', key: 'email' },
  offline_access: { icon: 'HOutline:ArrowPathIcon', key: 'offlineAccess' },
  phone: { icon: 'HOutline:DevicePhoneMobileIcon', key: 'phone' },
  address: { icon: 'HOutline:MapPinIcon', key: 'address' },
}

const scopeItems = computed(() => {
  const raw = authorizeInfo.value?.scope || scope.value || 'openid'
  const keys = [...new Set(raw.split(/\s+/).map((s) => s.trim()).filter(Boolean))]
  return keys.map((s) => {
    const meta = SCOPE_META[s]
    if (meta) {
      return {
        key: s,
        icon: meta.icon,
        label: t(`oidc.authorize.scopes.${meta.key}.label`),
        desc: t(`oidc.authorize.scopes.${meta.key}.desc`),
      }
    }
    return { key: s, icon: 'HOutline:KeyIcon', label: s, desc: t('oidc.authorize.scopes.custom') }
  })
})

const authorizationPayload = () => ({
  responseType: responseType.value,
  clientId: clientId.value,
  redirectUri: redirectUri.value,
  scope: scope.value,
})

const loadAuthorizeInfo = async () => {
  infoLoading.value = true
  loadError.value = ''
  try {
    const { data } = await getOIDCAuthorizationInfo(authorizationPayload())
    authorizeInfo.value = data?.info || null
    if (!authorizeInfo.value) loadError.value = t('oidc.authorize.invalidRequestDesc')
  } catch (error: unknown) {
    loadError.value = getRequestErrorMessage(error, t('oidc.authorize.invalidRequestDesc'))
  } finally {
    infoLoading.value = false
  }
}

// 授权页是第三方跳转入口，只生成授权码并跳回 redirect_uri，不承载后台管理布局。
const approve = async () => {
  submitting.value = true
  try {
    const { data } = await createOIDCAuthorizationCode({
      ...authorizationPayload(),
      state: state.value,
      nonce: queryValue('nonce'),
      codeChallenge: queryValue('code_challenge'),
      codeChallengeMethod: queryValue('code_challenge_method'),
    })
    if (data?.redirectUrl) {
      // 成功跳转，页面即将卸载，不复位 submitting。
      window.location.href = data.redirectUrl
      return
    }
    fb.error(t('oidc.authorize.errorGeneric'))
    submitting.value = false
  } catch (error: unknown) {
    fb.error(getRequestErrorMessage(error, t('oidc.authorize.errorGeneric')))
    submitting.value = false
  }
}

// 用户拒绝授权时按 OAuth 约定返回 access_denied，redirect_uri 无效时退回上一页。
const deny = () => {
  if (!redirectUri.value) {
    history.back()
    return
  }
  try {
    const url = new URL(redirectUri.value)
    url.searchParams.set('error', 'access_denied')
    if (state.value) url.searchParams.set('state', state.value)
    window.location.href = url.toString()
  } catch {
    history.back()
  }
}

// 切换账号：退出当前登录后回到登录页，登录完成再回到本授权页。
const switchAccount = async () => {
  const back = route.fullPath
  try {
    await logoutRequest({})
  } catch {
    // 退出失败也继续清理本地会话
  }
  userStore.logoutLocal({ redirectPath: back })
}

const goBack = () => history.back()

const toggleTheme = () => {
  themeStore.toggleThemeMode(themeStore.isDarkTheme ? 'light' : 'dark')
}

onMounted(async () => {
  await Promise.all([userStore.userInfo ? Promise.resolve() : userStore.getUserInfo(), loadAuthorizeInfo()])
})
</script>

<style scoped lang="scss">
.oa {
  position: relative;
  min-height: 100vh;
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--el-bg-color-page);
  color: var(--el-text-color-primary);
}
.oa-band {
  position: absolute;
  inset: 0 0 auto;
  height: 400px;
  overflow: hidden;
  background: var(--m-p);
}
.oa-band__blob {
  position: absolute;
  left: -90px;
  bottom: -150px;
  width: 380px;
  height: 380px;
  border-radius: 50%;
  background: var(--m-a);
}
.oa-band__ring {
  position: absolute;
  right: 12%;
  top: 70px;
  width: 150px;
  height: 150px;
  border-radius: 50%;
  border: 22px solid var(--m-a2);
}
.oa-band__star {
  position: absolute;
  left: 24%;
  top: 120px;
  width: 56px;
  height: 56px;
  fill: var(--m-pOn);
}
.oa-top {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 28px 48px;
  color: var(--m-pOn);
}
.oa-top__logo {
  width: 44px;
  height: 44px;
  border-radius: 14px;
  object-fit: cover;
  transform: rotate(-6deg);
  box-shadow: 0 0 0 3px var(--m-pOn);
}
.oa-top__name {
  font-family: var(--app-font-display);
  font-size: 1.5rem;
  font-weight: 800;
}
.oa-top__sub {
  margin-left: 4px;
  padding-left: 12px;
  border-left: 2px solid var(--m-pOn);
  font-size: 0.9375rem;
  font-weight: 700;
  white-space: nowrap;
}
.oa-top__spacer {
  flex: 1;
}
.oa-top__chrome {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px;
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}
.oa-main {
  position: relative;
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 12px 24px 40px;
}
.oa-card {
  width: 100%;
  max-width: 540px;
  display: flex;
  flex-direction: column;
  gap: 22px;
  padding: 32px;
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
  box-shadow: 0 30px 60px -30px color-mix(in srgb, var(--m-ink) 35%, transparent);
}

.oa-pair {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 14px;
}
.oa-pair__app {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 68px;
  height: 68px;
  border-radius: 22px;
  background: var(--m-aSoft);
  color: var(--m-aOn);
  font-family: var(--app-font-display);
  font-size: 1.875rem;
  font-weight: 800;
  transform: rotate(-6deg);
}
html.dark .oa-pair__app {
  color: var(--m-a);
}
.oa-pair__me {
  width: 68px;
  height: 68px;
  border-radius: 22px;
  object-fit: cover;
  transform: rotate(5deg);
  box-shadow: 0 0 0 3px var(--m-p);
}
.oa-pair__dots {
  display: flex;
  gap: 6px;
}
.oa-pair__dots i {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--el-border-color-light);
}
.oa-pair__dots i:nth-child(2) {
  background: var(--el-text-color-secondary);
}
.oa-pair__shield {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.oa-pair__shield svg {
  width: 22px;
  height: 22px;
}

.oa-head {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  text-align: center;
}
.oa-head h1 {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.625rem;
  line-height: 1.3;
  font-weight: 800;
  letter-spacing: -0.01em;
  overflow-wrap: anywhere;
  text-wrap: balance;
}
.oa-head__desc {
  margin: 0;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
  overflow-wrap: anywhere;
}
.oa-head__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 8px;
}
.oa-head__host {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}

.oa-account {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 12px 12px 14px;
  border-radius: 18px;
  background: var(--el-fill-color-light);
}
.oa-account__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.oa-account__name {
  font-size: 0.9375rem;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.oa-account__sub {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.oa-account__switch {
  flex: none;
  min-height: 44px;
  padding: 0 10px;
  border: 0;
  border-radius: 12px;
  background: transparent;
  color: var(--m-pInk);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 700;
  cursor: pointer;
}
.oa-account__switch:hover {
  background: var(--el-bg-color);
}

.oa-scopes__title {
  margin-bottom: 4px;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.oa-scopes ul {
  margin: 0;
  padding: 0;
  list-style: none;
}
.oa-scope {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.oa-scope:last-child {
  border-bottom: 0;
}
.oa-scope__ico {
  width: 42px;
  height: 42px;
  border-radius: 14px;
}
.oa-scope__ico svg {
  width: 20px;
  height: 20px;
}
.oa-scope__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.oa-scope__label {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 0.9375rem;
  font-weight: 800;
}
.oa-scope__key {
  padding: 2px 7px;
  border-radius: 6px;
  background: var(--el-fill-color-light);
  font-family: var(--app-font-mono);
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.oa-scope__desc {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  overflow-wrap: anywhere;
}
.oa-scope__check {
  flex: none;
  width: 20px;
  height: 20px;
  stroke-width: 2.6;
  color: var(--m-ok);
}

.oa-redirect {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  border: 1.5px dashed var(--el-border-color-light);
}
.oa-redirect > svg {
  flex: none;
  width: 20px;
  height: 20px;
  color: var(--el-text-color-secondary);
}
.oa-redirect__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.oa-redirect__label {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.oa-redirect__url {
  font-size: 0.75rem;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.oa-actions {
  display: flex;
  gap: 12px;
}
.oa-btn {
  justify-content: center;
  min-height: 56px;
  border-radius: 999px !important;
  font-size: 1rem !important;
  font-weight: 800 !important;
}
.oa-btn--outline {
  background: var(--el-bg-color) !important;
  box-shadow: inset 0 0 0 1.5px var(--el-border-color-light) !important;
}
.oa-btn--outline:hover:not(:disabled) {
  color: var(--m-pInk) !important;
  box-shadow: inset 0 0 0 1.5px var(--m-p) !important;
}
.oa-btn--deny {
  flex: 1;
}
.oa-btn--allow {
  flex: 1.4;
  box-shadow: var(--app-shadow-brand);
}
.oa-note {
  margin: 0;
  font-size: 0.75rem;
  line-height: 1.6;
  text-align: center;
  color: var(--el-text-color-secondary);
}
.oa-foot {
  margin: 22px 0 0;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  text-align: center;
  overflow-wrap: anywhere;
}

.oa-invalid {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 12px 0;
  text-align: center;
}
.oa-invalid__ico {
  width: 64px;
  height: 64px;
  border-radius: 20px;
  transform: rotate(-6deg);
}
.oa-invalid__ico svg {
  width: 30px;
  height: 30px;
}
.oa-invalid h1 {
  margin: 6px 0 0;
  font-family: var(--app-font-display);
  font-size: 1.5rem;
  font-weight: 800;
}
.oa-invalid p {
  margin: 0 0 8px;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
  overflow-wrap: anywhere;
}
.oa-invalid .oa-btn {
  min-height: 48px;
  padding: 0 22px;
  font-size: 0.9375rem !important;
}

.oa-skel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 8px 0 4px;
}
.oa-skel__line {
  height: 14px;
  border-radius: 6px;
  background: var(--el-fill-color);
  animation: oa-pulse 1.4s ease-in-out infinite;
}
.oa-skel__line--lg {
  width: 60%;
  height: 18px;
}
.oa-skel__line--sm {
  width: 40%;
}
@keyframes oa-pulse {
  50% {
    opacity: 0.5;
  }
}

@media (width < 768px) {
  .oa-band {
    height: 300px;
  }
  .oa-band__blob {
    left: auto;
    right: -70px;
    top: -80px;
    bottom: auto;
    width: 200px;
    height: 200px;
  }
  .oa-band__ring {
    right: auto;
    left: 28px;
    top: 150px;
    width: 64px;
    height: 64px;
    border-width: 11px;
  }
  .oa-band__star {
    display: none;
  }
  .oa-top {
    padding: 14px 12px 14px 20px;
    gap: 10px;
  }
  .oa-top__logo {
    width: 38px;
    height: 38px;
    border-radius: 12px;
  }
  .oa-top__name {
    font-size: 1.25rem;
  }
  .oa-top__sub {
    display: none;
  }
  .oa-main {
    padding: 12px 12px 24px;
  }
  .oa-card {
    padding: 24px 18px;
    gap: 18px;
    border-radius: 26px;
  }
  .oa-pair {
    gap: 10px;
  }
  .oa-pair__app,
  .oa-pair__me {
    width: 56px;
    height: 56px;
    border-radius: 18px;
    font-size: 1.5rem;
  }
  .oa-pair__dots i {
    width: 6px;
    height: 6px;
  }
  .oa-head h1 {
    font-size: 1.375rem;
  }
  .oa-actions {
    flex-direction: column-reverse;
  }
  .oa-btn--deny,
  .oa-btn--allow {
    flex: none;
    width: 100%;
  }
}

@media (prefers-reduced-motion: reduce) {
  .oa-skel__line {
    animation: none;
  }
}
</style>
