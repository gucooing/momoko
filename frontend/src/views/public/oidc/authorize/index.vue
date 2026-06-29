<template>
  <main class="oidc-authorize-page">
    <section class="authorize-card">
      <header class="brand-header">
        <div class="brand">
          <img :src="APP_CONFIG.logoSrc" :alt="APP_CONFIG.name" class="brand-logo" />
          <span class="brand-name">{{ APP_CONFIG.name }}</span>
        </div>
        <div class="account-chip">
          <el-avatar :size="32" :src="userStore.resolvedUserAvatar" />
          <span>{{ currentUserName }}</span>
        </div>
      </header>

      <div class="authorize-header">
        <el-icon class="authorize-icon"><Key /></el-icon>
        <div class="authorize-title">
          <h1>OIDC 授权</h1>
          <p>{{ authorizeInfo?.clientName || '未知客户端' }}</p>
        </div>
      </div>

      <el-alert v-if="errorMessage" :title="errorMessage" type="error" show-icon :closable="false" class="authorize-alert" />

      <div class="info-list" v-loading="infoLoading">
        <div class="info-row">
          <span>授权范围</span>
          <strong>{{ authorizeInfo?.scope || scope || 'openid' }}</strong>
        </div>
      </div>

      <footer class="action-row">
        <el-button :icon="Close" :disabled="submitting" @click="deny">取消</el-button>
        <el-button type="primary" :icon="Check" :loading="submitting" @click="approve">允许授权</el-button>
      </footer>
    </section>
  </main>
</template>

<script setup lang="ts">
import { createOIDCAuthorizationCode, getOIDCAuthorizationInfo } from '@/api/oidc'
import { APP_CONFIG } from '@/config/app.config'
import { useUserStore } from '@/stores/user'
import type { OIDCAuthorizationInfo } from '@/types/v1/oidc'
import { Check, Close, Key } from '@element-plus/icons-vue'

defineOptions({ name: 'OIDCAuthorizeView' })

const route = useRoute()
const userStore = useUserStore()
const submitting = ref(false)
const infoLoading = ref(false)
const errorMessage = ref('')
const authorizeInfo = ref<OIDCAuthorizationInfo | null>(null)

const queryValue = (key: string) => {
  const value = route.query[key]
  return Array.isArray(value) ? value[0] || '' : String(value || '')
}

const responseType = computed(() => queryValue('response_type') || 'code')
const clientId = computed(() => queryValue('client_id'))
const redirectUri = computed(() => queryValue('redirect_uri'))
const scope = computed(() => queryValue('scope') || 'openid')
const state = computed(() => queryValue('state'))
const currentUserName = computed(() => userStore.userInfo?.name || userStore.userInfo?.username || '当前账户')

const authorizationPayload = () => ({
  responseType: responseType.value,
  clientId: clientId.value,
  redirectUri: redirectUri.value,
  scope: scope.value,
})

const loadAuthorizeInfo = async () => {
  infoLoading.value = true
  errorMessage.value = ''
  try {
    const { data } = await getOIDCAuthorizationInfo(authorizationPayload())
    authorizeInfo.value = data?.info || null
  } catch (error: unknown) {
    errorMessage.value = error instanceof Error ? error.message : 'OIDC 授权请求无效'
  } finally {
    infoLoading.value = false
  }
}

// 授权页是第三方跳转入口，只生成授权码并跳回 redirect_uri，不承载后台管理布局。
const approve = async () => {
  submitting.value = true
  errorMessage.value = ''
  try {
    const { data } = await createOIDCAuthorizationCode({
      ...authorizationPayload(),
      state: state.value,
      nonce: queryValue('nonce'),
      codeChallenge: queryValue('code_challenge'),
      codeChallengeMethod: queryValue('code_challenge_method'),
    })
    if (data?.redirectUrl) {
      window.location.href = data.redirectUrl
    }
  } catch (error: unknown) {
    errorMessage.value = error instanceof Error ? error.message : 'OIDC 授权失败'
  } finally {
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
    if (state.value) {
      url.searchParams.set('state', state.value)
    }
    window.location.href = url.toString()
  } catch {
    history.back()
  }
}

onMounted(async () => {
  await Promise.all([
    userStore.userInfo ? Promise.resolve() : userStore.getUserInfo(),
    loadAuthorizeInfo(),
  ])
})
</script>

<style scoped lang="scss">
.oidc-authorize-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.25rem;
  background:
    radial-gradient(circle at top left, rgb(20 184 166 / 12%), transparent 18rem),
    var(--el-bg-color-page);
}

.authorize-card {
  width: min(100%, 36rem);
  padding: 1.25rem;
  border: 1px solid var(--el-border-color-light);
  border-radius: 0.5rem;
  background: var(--el-bg-color-overlay);
  box-shadow: var(--el-box-shadow-light);
}

.brand-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1.25rem;
}

.brand,
.account-chip {
  display: flex;
  align-items: center;
  min-width: 0;
}

.brand {
  gap: 0.65rem;
}

.brand-logo {
  width: 2.15rem;
  height: 2.15rem;
  border-radius: 50%;
  object-fit: cover;
}

.brand-name {
  font-size: 1rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

.account-chip {
  gap: 0.5rem;
  max-width: 14rem;
  color: var(--el-text-color-secondary);
  font-size: 0.84rem;

  span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.authorize-header {
  display: flex;
  align-items: center;
  gap: 0.8rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.authorize-icon {
  width: 2.4rem;
  height: 2.4rem;
  flex-shrink: 0;
  border-radius: 0.45rem;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.authorize-title {
  min-width: 0;

  h1 {
    margin: 0;
    font-size: 1.1rem;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  p {
    margin: 0.25rem 0 0;
    font-size: 0.82rem;
    color: var(--el-text-color-secondary);
    overflow-wrap: anywhere;
  }
}

.authorize-alert {
  margin-top: 1rem;
}

.info-list {
  margin-top: 1rem;
}

.info-row {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.65rem 0;
  border-bottom: 1px solid var(--el-border-color-lighter);

  span {
    flex-shrink: 0;
    font-size: 0.84rem;
    color: var(--el-text-color-secondary);
  }

  strong {
    font-size: 0.84rem;
    font-weight: 600;
    color: var(--el-text-color-primary);
    text-align: right;
    overflow-wrap: anywhere;
  }
}

.action-row {
  display: flex;
  justify-content: flex-end;
  gap: 0.6rem;
  margin-top: 1.1rem;
}

@media (width <= 520px) {
  .authorize-card {
    padding: 1rem;
  }

  .brand-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .info-row,
  .action-row {
    flex-direction: column;
  }

  .info-row strong {
    text-align: left;
  }

  .action-row :deep(.el-button) {
    width: 100%;
    margin-left: 0;
  }
}
</style>
