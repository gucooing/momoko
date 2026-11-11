<!-- 个人中心 · 个人资料（ProfilePC / ProfileMobile）
     PC：左（hero / 个人资料表单 / 账号安全 / 登录设备）右（我的消息 / 我的权限 / 最近登录 / 外观入口 / 退出登录）
     手机：hero → 个人资料行（点开编辑弹窗）→ 账号安全 → 偏好与消息（外观 / 语言 / 消息）→ 权限 → 设备 → 最近登录 → 退出 -->
<template>
  <div class="profile">
    <!-- ===== PC ===== -->
    <div v-if="!isMobile" class="profile-grid">
      <div class="profile-col profile-col--main">
        <ArchivesPanel />
        <PersonalInfoPanel />
        <MyInformation />
        <LoginDevices />
      </div>
      <div class="profile-col profile-col--side">
        <MyMessages />
        <MyPermission />
        <LoginLogs />
        <button type="button" class="appearance-card" @click="openAppearance">
          <span class="m-sticker m-sticker--solid-a appearance-card__sk" aria-hidden="true">
            <UIcon name="i-lucide-palette" />
          </span>
          <span class="appearance-card__text">
            <strong>{{ t('theme.title') }}</strong>
            <span>{{ appearanceSummary }} · {{ t('profile.appearance.onlyYou') }}</span>
          </span>
          <UIcon name="i-lucide-chevron-right" class="appearance-card__chev" aria-hidden="true" />
        </button>
        <AdaptiveConfirm :title="t('layout.logoutConfirmContent')" @confirm="doLogout">
          <template #reference>
            <UButton color="error" variant="soft" icon="i-lucide-log-out" class="logout-btn">
              {{ t('layout.logout') }}
            </UButton>
          </template>
        </AdaptiveConfirm>
      </div>
    </div>

    <!-- ===== 手机 ===== -->
    <div v-else class="profile-m">
      <ArchivesPanel />

      <section class="m-group" aria-labelledby="pf-g-info">
        <h2 id="pf-g-info" class="m-group__h">{{ t('profile.info.title') }}</h2>
        <AppPanel class="m-group__card">
          <ProfileRow
            icon="i-lucide-user-round"
            tone="solid-p"
            :title="t('user.nickname')"
            clickable
            chevron
            @click="editOpen = true"
          >
            <span class="m-val">{{ userStore.userInfo?.name || '—' }}</span>
          </ProfileRow>
          <ProfileRow
            icon="i-lucide-at-sign"
            tone="a2"
            :title="t('user.username')"
            :sub="t('profile.info.usernameHintShort')"
            clickable
            chevron
            @click="editOpen = true"
          >
            <span class="m-val m-mono">{{ userStore.userInfo?.username || '—' }}</span>
          </ProfileRow>
          <ProfileRow
            icon="i-lucide-mail"
            tone="solid-a2"
            :title="t('user.email')"
            :sub="userStore.userInfo?.email || t('user.noEmail')"
            clickable
            chevron
            @click="editOpen = true"
          />
          <ProfileRow
            icon="i-lucide-phone"
            tone="solid-a"
            :title="t('profile.info.phone')"
            clickable
            chevron
            @click="editOpen = true"
          >
            <span class="m-val m-num">{{ maskedPhone || t('profile.info.notSet') }}</span>
          </ProfileRow>
        </AppPanel>
      </section>

      <MyInformation />

      <section class="m-group" aria-labelledby="pf-g-pref">
        <h2 id="pf-g-pref" class="m-group__h">{{ t('profile.pref.title') }}</h2>
        <AppPanel class="m-group__card">
          <ProfileRow
            icon="i-lucide-palette"
            tone="solid-a"
            :tilt="-4"
            :title="t('theme.title')"
            :sub="t('profile.appearance.rowHint')"
            clickable
            chevron
            @click="openAppearance"
          >
            <span class="m-val">{{ appearanceSummary }}</span>
          </ProfileRow>
          <ProfileRow icon="i-lucide-globe" tone="solid-a2" :tilt="3" :title="t('language.tooltip')">
            <div class="m-lang">
              <AppSelect
                :model-value="locale"
                :options="languageSelectOptions"
                fit
                @update:model-value="onLocale"
              />
            </div>
          </ProfileRow>
          <MyMessages row />
        </AppPanel>
      </section>

      <MyPermission />
      <LoginDevices />
      <LoginLogs />

      <AdaptiveConfirm :title="t('layout.logoutConfirmContent')" @confirm="doLogout">
        <template #reference>
          <button type="button" class="m-logout">
            <UIcon name="i-lucide-log-out" />{{ t('layout.logout') }}
          </button>
        </template>
      </AdaptiveConfirm>
      <p class="m-foot m-num">{{ footText }}</p>

      <FormDialog v-model="editOpen" :title="t('profile.info.edit')" :width="560" :show-footer="false">
        <PersonalInfoPanel embedded @saved="editOpen = false" />
      </FormDialog>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useUserProfileStore } from '@/stores/user/profile'
import { languageOptions, setAppLocale } from '@/locales'
import ArchivesPanel from '@/views/profile/archivesPanel.vue'
import PersonalInfoPanel from '@/views/profile/personalInfoPanel.vue'
import MyInformation from '@/views/profile/myInformation.vue'
import MyPermission from '@/views/profile/myPermission.vue'
import MyMessages from '@/views/profile/myMessages.vue'
import LoginLogs from '@/views/profile/loginLogs.vue'
import LoginDevices from '@/views/profile/loginDevices.vue'
import ProfileRow from '@/views/profile/components/ProfileRow.vue'
import { useCurrentHostname } from '@/layouts/app/useLocalHost'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'ProfileInfoTab' })

const { t, locale } = useI18n()
const userStore = useUserStore()
const menuStore = useMenuStore()
const themeStore = useThemeStore()
const userProfileStore = useUserProfileStore()

const isMobile = computed(() => menuStore.isMobile)
const editOpen = ref(false)

const appearanceSummary = computed(() => {
  const palette = themeStore.activeName || t(themeStore.currentPalette.labelKey)
  const mode = t(`theme.${themeStore.themeMode}`)
  return `${palette} · ${mode}`
})

const maskedPhone = computed(() => {
  const p = (userStore.userInfo?.phone || '').replace(/\s+/g, '')
  if (p.length < 7) return p
  return `${p.slice(0, 3)} **** ${p.slice(-4)}`
})

const languageSelectOptions = computed(() =>
  languageOptions.map((o) => ({ label: t(o.labelKey), value: o.code as string })),
)
const onLocale = (code: string) => setAppLocale(code as never)

const router = useRouter()
const openAppearance = () => {
  void router.push({ path: '/profile', query: { tab: 'appearance' } })
}

const hostname = useCurrentHostname()
const footText = computed(() =>
  [`Momoko ${menuStore.currentVersion || 'dev'}`, hostname.value].filter(Boolean).join(' · '),
)

const doLogout = () => {
  void userStore.logout()
}

onMounted(() => {
  void userProfileStore.refreshMessages()
})
</script>

<style scoped lang="scss">
.profile {
  display: flex;
  flex-direction: column;
  gap: 22px;
  width: 100%;
  min-width: 0;
}
.profile-grid {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 24px;
}
.profile-col {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.profile-col--main {
  flex: 3 1 560px;
}
.profile-col--side {
  flex: 2 1 340px;
}

.appearance-card {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  padding: calc(var(--cmp-card-unit) * 16) calc(var(--cmp-card-unit) * 18);
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-card);
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s;
}
.appearance-card:hover {
  border-color: var(--m-p);
}
.appearance-card__sk {
  width: 42px;
  height: 42px;
  border-radius: 14px;
  font-size: 21px;
  transform: rotate(-4deg);
}
.appearance-card__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  strong {
    font-size: 0.9375rem;
    font-weight: 800;
  }
  span {
    font-size: 0.8125rem;
    color: var(--el-text-color-secondary);
  }
}
.appearance-card__chev {
  width: 18px;
  height: 18px;
  color: var(--el-text-color-secondary);
}
.logout-btn {
  align-self: flex-start;
  min-height: 40px;
}

/* ---- 手机 ---- */
.profile-m {
  display: flex;
  flex-direction: column;
  gap: 22px;
}
.m-group__h {
  margin: 0 6px 8px;
  font-family: var(--app-font-display);
  font-size: 0.9375rem;
  font-weight: 800;
  color: var(--el-text-color-secondary);
}
.m-group__card :deep(.app-panel__body) {
  padding: 2px 14px;
}
.m-group__card :deep(.pf-row) {
  min-height: 60px;
  padding: 8px 0;
}
.m-group__card :deep(.pf-row__sk) {
  width: 36px;
  height: 36px;
  border-radius: 12px;
  font-size: 18px;
}
.m-val {
  max-width: 40vw;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.m-lang {
  min-width: 120px;
}
.m-logout {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  min-height: 52px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  color: var(--m-bad);
  font: inherit;
  font-size: 0.9375rem;
  font-weight: 800;
  cursor: pointer;
}
.m-foot {
  margin: -8px 0 0;
  text-align: center;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
</style>
