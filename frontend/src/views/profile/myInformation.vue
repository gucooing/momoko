<!-- 账号安全：登录密码（上次修改时间）/ 邮箱绑定 / 最近一次登录 -->
<template>
  <AppPanel class="sec" :padded="true">
    <template #header>
      <h2 class="pf-card-title">{{ t('profile.security.title') }}</h2>
    </template>

    <ProfileRow
      icon="i-lucide-lock-keyhole"
      tone="p"
      :tilt="-3"
      :title="t('profile.security.password')"
      :sub="passwordSub"
      :clickable="compact"
      :chevron="compact"
      @click="openPassword"
    >
      <UButton v-if="!compact" color="primary" variant="outline" size="sm" class="pf-btn-sm" @click="openPassword">
        {{ t('user.passwordCardTitle') }}
      </UButton>
    </ProfileRow>

    <ProfileRow
      icon="i-lucide-mail-check"
      tone="ok"
      :tilt="3"
      :title="t('profile.security.email')"
      :sub="emailSub"
    >
      <StatusPill
        :variant="hasEmail ? 'success' : 'warning'"
        :label="hasEmail ? t('profile.security.emailBound') : t('profile.security.emailUnbound')"
      />
    </ProfileRow>

    <ProfileRow
      icon="i-lucide-log-in"
      tone="a2"
      :tilt="-2"
      :title="t('profile.security.lastLogin')"
      :sub="lastLoginSub"
      num-sub
    />

    <UpdatePassword ref="updatePasswordRef" />
  </AppPanel>
</template>

<script setup lang="ts">
import UpdatePassword from '@/components/dialog/UpdatePassword.vue'
import { useI18n } from 'vue-i18n'
import ProfileRow from './components/ProfileRow.vue'
import { useProfileFormat } from './components/useProfileFormat'

const userStore = useUserStore()
const menuStore = useMenuStore()
const { t } = useI18n()
const { daysAgo, relTime, scopeLabel, methodLabel } = useProfileFormat()

const compact = computed(() => menuStore.isMobile)
const updatePasswordRef = ref<{ showDialog: () => void }>()
const openPassword = () => updatePasswordRef.value?.showDialog()

const passwordSub = computed(() => {
  const days = daysAgo(userStore.userInfo?.passwordUpdateTime)
  if (days === null) return t('profile.security.passwordNever')
  const when = days === 0 ? t('profile.security.today') : t('profile.security.daysAgo', { n: days })
  return days >= 90
    ? t('profile.security.passwordOld', { when })
    : t('profile.security.passwordChanged', { when })
})

const hasEmail = computed(() => Boolean(userStore.userInfo?.email))
const emailSub = computed(() =>
  hasEmail.value
    ? t('profile.security.emailSub', { email: userStore.userInfo?.email || '' })
    : t('profile.security.emailMissing'),
)

const lastLoginSub = computed(() => {
  const u = userStore.userInfo
  if (!u?.lastLoginTime) return '—'
  const parts = [relTime(u.lastLoginTime)]
  if (u.lastLoginIp) {
    const scope = scopeLabel(ipScopeOf(u.lastLoginIp))
    parts.push(scope ? `${u.lastLoginIp} · ${scope}` : u.lastLoginIp)
  }
  parts.push(methodLabel(u.lastLoginMethod))
  return parts.join(' · ')
})

/** 粗略判断：本机 / 局域网 / 公网（与后端口径一致） */
const ipScopeOf = (ip: string) => {
  if (ip === '::1' || ip.startsWith('127.')) return 'loopback'
  if (
    /^10\./.test(ip) ||
    /^192\.168\./.test(ip) ||
    /^172\.(1[6-9]|2\d|3[01])\./.test(ip) ||
    /^169\.254\./.test(ip) ||
    /^f[cd]/i.test(ip) ||
    /^fe80/i.test(ip)
  )
    return 'lan'
  return 'public'
}
</script>

<style scoped lang="scss">
.pf-btn-sm {
  min-height: 36px;
}
.sec :deep(.app-panel__body) {
  padding-top: 8px;
  padding-bottom: 8px;
}
.pf-card-title {
  font-family: var(--app-font-display);
  font-size: 1.25rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
</style>
