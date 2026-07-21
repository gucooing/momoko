<!-- OIDC 授权确认页（P7 · authOnly）：独立居中卡，展示应用信息 + 请求权限 + 当前账户，授权/拒绝。
     不承载后台外壳；授权成功后跳回第三方 redirect_uri。角标仅语言/明暗，与登录门面同一套观感。 -->
<template>
  <div class="oidc-page">
    <div class="oidc-page__chrome">
      <LanguageMenu />
      <AppIconButton
        :icon="themeStore.isDarkTheme ? 'HOutline:SunIcon' : 'HOutline:MoonIcon'"
        :label="t('login.toggleTheme')"
        :box="32"
        @click="toggleTheme"
      />
    </div>

    <div class="oidc-page__center">
      <div class="oidc-card">
        <div class="oidc-card__brand">
          <img :src="APP_CONFIG.logoSrc" alt="" class="oidc-card__logo" />
          <span class="oidc-card__name">{{ APP_CONFIG.name }}</span>
        </div>

        <!-- 加载授权信息 -->
        <div v-if="infoLoading" class="oidc-skel">
          <span class="oidc-skel__line oidc-skel__line--lg" />
          <span class="oidc-skel__line" />
          <span class="oidc-skel__line oidc-skel__line--sm" />
        </div>

        <!-- 授权请求无效 -->
        <EmptyState
          v-else-if="loadError"
          icon="HOutline:ShieldExclamationIcon"
          :title="t('oidc.authorize.invalidRequest')"
          :description="loadError"
        >
          <template #action>
            <UButton color="neutral" variant="ghost" size="sm" @click="goBack">
              {{ t('oidc.authorize.back') }}
            </UButton>
          </template>
        </EmptyState>

        <!-- 授权确认 -->
        <template v-else>
          <div class="oidc-head">
            <span class="oidc-head__icon">
              <component :is="menuStore.iconComponents['HOutline:ShieldCheckIcon']" />
            </span>
            <div class="oidc-head__text">
              <h1 class="oidc-head__title">{{ t('oidc.authorize.title') }}</h1>
              <p class="oidc-head__sub">
                {{ t('oidc.authorize.clientRequests', { client: clientName }) }}
              </p>
            </div>
          </div>

          <div class="oidc-account">
            <AppAvatar :src="userStore.resolvedUserAvatar" :name="currentUserName" :size="34" />
            <div class="oidc-account__text">
              <span class="oidc-account__label">{{ t('oidc.authorize.signedInAs') }}</span>
              <span class="oidc-account__name">{{ currentUserName }}</span>
            </div>
          </div>

          <div class="oidc-scopes">
            <p class="oidc-scopes__title">{{ t('oidc.authorize.scopeTitle') }}</p>
            <ul class="oidc-scopes__list">
              <li v-for="s in scopeItems" :key="s.key" class="oidc-scope">
                <span class="oidc-scope__icon">
                  <component :is="menuStore.iconComponents[s.icon]" />
                </span>
                <div class="oidc-scope__text">
                  <span class="oidc-scope__label">{{ s.label }}</span>
                  <span class="oidc-scope__desc">{{ s.desc }}</span>
                </div>
              </li>
            </ul>
          </div>

          <p v-if="redirectHost" class="oidc-redirect">
            {{ t('oidc.authorize.redirectHint', { host: redirectHost }) }}
          </p>

          <div class="oidc-actions">
            <UButton
              class="oidc-actions__btn"
              color="neutral"
              variant="ghost"
              :disabled="submitting"
              @click="deny"
            >
              {{ t('oidc.authorize.deny') }}
            </UButton>
            <UButton
              class="oidc-actions__btn"
              color="primary"
              :loading="submitting"
              @click="approve"
            >
              {{ t('oidc.authorize.approve') }}
            </UButton>
          </div>
        </template>
      </div>

      <p class="oidc-page__copy">
        {{ t('login.copyright', { year: currentYear, name: APP_CONFIG.name }) }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { createOIDCAuthorizationCode, getOIDCAuthorizationInfo } from '@/api/oidc'
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
const currentYear = new Date().getFullYear()

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
  () =>
    userStore.userInfo?.name || userStore.userInfo?.username || t('oidc.authorize.currentAccount'),
)
const clientName = computed(
  () => authorizeInfo.value?.clientName || t('oidc.authorize.unknownClient'),
)

// 展示授权后将跳转到的第三方域名（安全信号：让用户看清 code 发往何处）。
const redirectHost = computed(() => {
  const uri = authorizeInfo.value?.redirectUri || redirectUri.value
  if (!uri) return ''
  try {
    return new URL(uri).host
  } catch {
    return ''
  }
})

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

const goBack = () => history.back()

const toggleTheme = () => {
  themeStore.toggleThemeMode(themeStore.isDarkTheme ? 'light' : 'dark')
}

onMounted(async () => {
  await Promise.all([
    userStore.userInfo ? Promise.resolve() : userStore.getUserInfo(),
    loadAuthorizeInfo(),
  ])
})
</script>

<style scoped lang="scss">
.oidc-page {
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

.oidc-page__chrome {
  position: absolute;
  top: 12px;
  right: 12px;
  display: flex;
  align-items: center;
  gap: 2px;
  z-index: 2;
}

.oidc-page__center {
  width: min(100%, 440px);
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 16px;
}

.oidc-card {
  background: var(--el-bg-color);
  border: 1px solid var(--app-hairline);
  border-radius: var(--app-radius-lg);
  padding: 24px 24px 22px;
  box-shadow: var(--app-shadow-sm);
}

.oidc-card__brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-bottom: 18px;
  margin-bottom: 18px;
  border-bottom: 1px solid var(--app-hairline);
}
.oidc-card__logo {
  width: 30px;
  height: 30px;
  object-fit: contain;
  flex-shrink: 0;
}
.oidc-card__name {
  font-size: 1.0625rem;
  font-weight: 650;
  letter-spacing: -0.01em;
  color: var(--el-text-color-primary);
}

.oidc-head {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 18px;
}
.oidc-head__icon {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--app-radius);
  color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 12%, transparent);
}
.oidc-head__icon :deep(svg) {
  width: 22px;
  height: 22px;
}
.oidc-head__title {
  margin: 0;
  font-size: 1.05rem;
  font-weight: 650;
  color: var(--el-text-color-primary);
}
.oidc-head__sub {
  margin: 4px 0 0;
  font-size: 0.8125rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
  overflow-wrap: anywhere;
}

.oidc-account {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid var(--app-hairline);
  border-radius: var(--app-radius);
  margin-bottom: 18px;
}
.oidc-account__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.oidc-account__label {
  font-size: 0.6875rem;
  color: var(--el-text-color-secondary);
}
.oidc-account__name {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.oidc-scopes__title {
  margin: 0 0 10px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  color: var(--el-text-color-secondary);
}
.oidc-scopes__list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.oidc-scope {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}
.oidc-scope__icon {
  flex-shrink: 0;
  width: 30px;
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--app-radius-sm);
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-light);
}
.oidc-scope__icon :deep(svg) {
  width: 17px;
  height: 17px;
}
.oidc-scope__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: 1px;
}
.oidc-scope__label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.oidc-scope__desc {
  font-size: 0.75rem;
  line-height: 1.45;
  color: var(--el-text-color-secondary);
  overflow-wrap: anywhere;
}

.oidc-redirect {
  margin: 16px 0 0;
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
  text-align: center;
  overflow-wrap: anywhere;
}

.oidc-actions {
  display: flex;
  gap: 10px;
  margin-top: 20px;
}
.oidc-actions__btn {
  flex: 1;
  justify-content: center;
}

.oidc-skel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 8px 0 4px;
}
.oidc-skel__line {
  height: 14px;
  border-radius: 6px;
  background: var(--el-fill-color);
  animation: oidc-pulse 1.4s ease-in-out infinite;
}
.oidc-skel__line--lg {
  width: 60%;
  height: 18px;
}
.oidc-skel__line--sm {
  width: 40%;
}
@keyframes oidc-pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

.oidc-page__copy {
  margin: 0;
  text-align: center;
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}

@media (prefers-reduced-motion: reduce) {
  .oidc-skel__line {
    animation: none;
  }
}

@media (max-width: 480px) {
  .oidc-page {
    padding: 56px 12px 32px;
  }
  .oidc-card {
    padding: 20px 16px 18px;
    border-radius: var(--app-radius);
  }
  .oidc-actions {
    flex-direction: column-reverse;
  }
}
</style>
