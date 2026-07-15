<!-- 个人中心（重写 · P2 选项卡）：克制头部（头像+名/角色/简介+操作）+ 令牌 Tab 条 + 各 Tab 面板。
     保留 useUserProfileStore 契约与业务流；去 EP 大 banner / BaseCard / BadgeTabsMenu（06a）。 -->
<template>
  <div class="profile-page">
    <!-- 头部：安静，非花哨渐变大图 -->
    <header class="profile-hero">
      <div class="profile-hero__main">
        <AppAvatar
          :src="userStore.resolvedUserAvatar"
          :name="displayName"
          :size="menuStore.isMobile ? 64 : 80"
          online
        />
        <div class="profile-hero__meta">
          <div class="profile-hero__name-row">
            <h1 class="profile-hero__name" :title="displayName">{{ displayName }}</h1>
            <StatusPill variant="primary" :dot="false" :label="roleLabel" />
          </div>
          <p class="profile-hero__bio">{{ bioText }}</p>
          <div v-if="locationText" class="profile-hero__loc">
            <component :is="menuStore.iconComponents['HOutline:MapPinIcon']" class="profile-hero__loc-icon" />
            <span>{{ locationText }}</span>
          </div>
        </div>
      </div>
      <div class="profile-hero__actions">
        <UButton color="neutral" variant="soft" size="sm" icon="i-lucide-key-round" @click="openPassword">
          {{ t('user.passwordCardTitle') }}
        </UButton>
        <UButton
          v-if="userProfileStore.currentTab === 'personalInfo'"
          color="primary"
          size="sm"
          icon="i-lucide-save"
          @click="saveFromHeader"
        >
          {{ t('user.saveAllChanges') }}
        </UButton>
      </div>
    </header>

    <!-- Tab 条 -->
    <div class="profile-tabs" role="tablist">
      <button
        v-for="tab in userProfileStore.menuTabs"
        :key="tab.key"
        type="button"
        role="tab"
        class="profile-tabs__btn"
        :class="{ 'is-active': userProfileStore.currentTab === tab.key }"
        :aria-selected="userProfileStore.currentTab === tab.key"
        @click="userProfileStore.currentTab = tab.key as ProfileCurrentTab"
      >
        <component
          :is="menuStore.iconComponents[tab.icon as string]"
          v-if="tab.icon && typeof tab.icon === 'string'"
          class="profile-tabs__icon"
        />
        <span class="profile-tabs__label">{{ tab.labelKey ? t(tab.labelKey) : tab.label }}</span>
        <span
          v-if="tab.key === 'messages' && userProfileStore.unreadCount"
          class="profile-tabs__badge"
        >
          {{ userProfileStore.unreadCount }}
        </span>
      </button>
    </div>

    <!-- Tab 内容 -->
    <div class="profile-body">
      <MyInformation v-if="userProfileStore.currentTab === 'personalInfo'" ref="infoRef" />
      <MyPermission v-else-if="userProfileStore.currentTab === 'permissions'" />
      <MyMessages v-else-if="userProfileStore.currentTab === 'messages'" />
      <LoginLogs v-else-if="userProfileStore.currentTab === 'logs'" />
      <LoginDevices v-else-if="userProfileStore.currentTab === 'devices'" />
      <MyInformation v-else ref="infoRef" />
    </div>

    <UpdatePassword ref="updatePasswordRef" />
  </div>
</template>

<script setup lang="ts">
import { useUserProfileStore } from '@/stores/user/profile'
import type { ProfileCurrentTab } from '@/stores/user/types'
import MyInformation from '@/views/profile/myInformation.vue'
import MyPermission from '@/views/profile/myPermission.vue'
import MyMessages from '@/views/profile/myMessages.vue'
import LoginLogs from '@/views/profile/loginLogs.vue'
import LoginDevices from '@/views/profile/loginDevices.vue'
import UpdatePassword from '@/components/dialog/UpdatePassword.vue'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'ProfileView' })

const { t } = useI18n()
const userStore = useUserStore()
const menuStore = useMenuStore()
const userProfileStore = useUserProfileStore()

const updatePasswordRef = ref<{ showDialog: () => void }>()
const infoRef = ref<{ save: () => void } | null>(null)

const displayName = computed(
  () => userStore.userInfo?.name || userStore.userInfo?.username || t('user.messages.unknownUser'),
)
const roleLabel = computed(() => userStore.getUserRoleName() || t('user.noPermission'))
const bioText = computed(() => {
  const bio = userStore.userInfo?.bio?.trim()
  return bio || t('user.defaultBio')
})
const locationText = computed(() => {
  const { country, region, city } = userProfileStore.address
  const parts = [country, region, city].filter(Boolean)
  return parts.join(' · ')
})

const openPassword = () => updatePasswordRef.value?.showDialog()
const saveFromHeader = () => infoRef.value?.save()

onMounted(() => {
  void userProfileStore.ensureAddress()
})
</script>

<style scoped lang="scss">
.profile-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-width: 1100px;
}

.profile-hero {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  padding: 4px 0 2px;
  flex-wrap: wrap;
}
.profile-hero__main {
  display: flex;
  align-items: center;
  gap: 1rem;
  min-width: 0;
  flex: 1;
}
.profile-hero__meta {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}
.profile-hero__name-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
  min-width: 0;
}
.profile-hero__name {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 650;
  letter-spacing: -0.02em;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: min(100%, 20rem);
}
.profile-hero__bio {
  margin: 0;
  font-size: 0.8125rem;
  line-height: 1.45;
  color: var(--el-text-color-secondary);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.profile-hero__loc {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
.profile-hero__loc-icon {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}
.profile-hero__actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
}

.profile-tabs {
  display: flex;
  gap: 2px;
  padding: 3px;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  overflow-x: auto;
  scrollbar-width: none;
  &::-webkit-scrollbar {
    display: none;
  }
}
.profile-tabs__btn {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  flex-shrink: 0;
  padding: 0 12px;
  height: 32px;
  border: 0;
  border-radius: calc(var(--app-radius) - 2px);
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition:
    background 0.15s,
    color 0.15s;
  white-space: nowrap;
}
.profile-tabs__btn:hover {
  color: var(--el-text-color-primary);
}
.profile-tabs__btn.is-active {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm, 0 1px 2px rgb(0 0 0 / 0.04));
  font-weight: 600;
}
.profile-tabs__icon {
  width: 15px;
  height: 15px;
  flex-shrink: 0;
}
.profile-tabs__badge {
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--el-color-primary) 14%, transparent);
  color: var(--el-color-primary);
  font-size: 0.6875rem;
  font-weight: 700;
  line-height: 18px;
  text-align: center;
  font-variant-numeric: tabular-nums;
}

.profile-body {
  min-width: 0;
}

@media (width < 640px) {
  .profile-hero__main {
    align-items: flex-start;
  }
  .profile-hero__actions {
    width: 100%;
  }
  .profile-hero__actions :deep(button) {
    flex: 1;
  }
  .profile-tabs__label {
    /* 移动保留文字（比仅图标更清晰）；窄屏可横滑 */
  }
}
</style>
