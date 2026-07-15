<!-- 个人中心 · 对齐 06a
     ① 一体头部：克制色条 + 头像/名/角色/简介 + 详细档案（同一卡内，不拆第二张）+ 编辑/改密
     ② 主体：全宽 Tabs（消息 / 权限 / 设备 / 日志）
     ③ 编辑资料 / 改密 → FormDialog（不在页面内嵌长表单） -->
<template>
  <div class="profile">
    <!-- ① 一体头部（头像区 + 档案字段，同一表面） -->
    <header class="profile-hero">
      <div class="profile-hero__wash" aria-hidden="true" />

      <div class="profile-hero__top">
        <div class="profile-hero__identity">
          <AppAvatar
            :src="userStore.resolvedUserAvatar"
            :name="displayName"
            :size="isCompact ? 56 : 72"
            online
          />
          <div class="profile-hero__text">
            <div class="profile-hero__title-row">
              <h1 class="profile-hero__name" :title="displayName">{{ displayName }}</h1>
              <StatusPill variant="primary" :dot="false" :label="roleLabel" />
            </div>
            <p class="profile-hero__bio">{{ bioText }}</p>
          </div>
        </div>
        <div class="profile-hero__actions">
          <UButton color="neutral" variant="soft" size="sm" icon="i-lucide-pencil" @click="openEdit">
            {{ t('user.editProfile') }}
          </UButton>
          <UButton
            color="primary"
            variant="soft"
            size="sm"
            icon="i-lucide-key-round"
            @click="openPassword"
          >
            {{ t('user.passwordCardTitle') }}
          </UButton>
        </div>
      </div>

      <!-- 详细档案：同一卡内，细线分隔，多列 DescriptionList -->
      <div class="profile-hero__archive">
        <div class="profile-hero__archive-label">{{ t('user.detailArchive') }}</div>
        <DescriptionList :items="archiveItems" :columns="archiveColumns" />
        <div v-if="tagNames.length" class="profile-hero__tags">
          <span v-for="name in tagNames" :key="name" class="profile-hero__chip">{{ name }}</span>
        </div>
      </div>
    </header>

    <!-- ② 全宽 Tabs -->
    <section class="profile-main">
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
          <span>{{ tab.labelKey ? t(tab.labelKey) : tab.label }}</span>
          <span
            v-if="tab.key === 'messages' && userProfileStore.unreadCount"
            class="profile-tabs__badge"
          >
            {{ userProfileStore.unreadCount }}
          </span>
        </button>
      </div>

      <div class="profile-main__body">
        <MyMessages v-if="userProfileStore.currentTab === 'messages'" />
        <MyPermission v-else-if="userProfileStore.currentTab === 'permissions'" />
        <LoginDevices v-else-if="userProfileStore.currentTab === 'devices'" />
        <LoginLogs v-else-if="userProfileStore.currentTab === 'logs'" />
        <MyMessages v-else />
      </div>
    </section>

    <!-- ③ 编辑资料 FormDialog（桌面/移动统一） -->
    <FormDialog
      v-model="editOpen"
      :title="t('user.profileCenter')"
      :width="560"
      :show-footer="false"
    >
      <PersonalInfoPanel embedded @saved="editOpen = false" />
    </FormDialog>

    <UpdatePassword ref="updatePasswordRef" />
  </div>
</template>

<script setup lang="ts">
import dayjs from 'dayjs'
import { UserStatus } from '@/types/v1/user'
import { useUserProfileStore } from '@/stores/user/profile'
import type { ProfileCurrentTab } from '@/stores/user/types'
import PersonalInfoPanel from '@/views/profile/personalInfoPanel.vue'
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

const isCompact = computed(() => menuStore.isMobile)
const updatePasswordRef = ref<{ showDialog: () => void }>()
const editOpen = ref(false)

const displayName = computed(
  () => userStore.userInfo?.name || userStore.userInfo?.username || t('user.messages.unknownUser'),
)
const roleLabel = computed(() => userStore.getUserRoleName() || t('user.noPermission'))
const bioText = computed(() => {
  const bio = userStore.userInfo?.bio?.trim()
  return bio || t('user.defaultBio')
})

const createTimeText = computed(() => {
  if (!userStore.userInfo?.createTime) return ''
  return dayjs(userStore.userInfo.createTime).format('YYYY-MM-DD HH:mm:ss')
})

const archiveItems = computed(() => [
  { label: t('user.username'), value: userStore.userInfo?.username },
  { label: t('user.email'), value: userStore.userInfo?.email || t('user.noEmail') },
  { label: t('user.nickname'), value: userStore.userInfo?.name },
  {
    label: t('user.accountStatus'),
    value:
      userStore.userInfo?.status === UserStatus.Active ? t('common.enabled') : t('common.disabled'),
  },
  { label: t('user.joinTime'), value: createTimeText.value },
])

const archiveColumns = computed(() => {
  if (menuStore.isMobile) return 1
  return 3
})

const tagNames = computed(() => {
  const raw = userStore.userInfo?.tags?.trim()
  if (!raw) return [] as string[]
  return raw
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
})

const openPassword = () => updatePasswordRef.value?.showDialog()
const openEdit = () => {
  editOpen.value = true
}

onMounted(() => {
  if (!(userProfileStore.menuTabs as { key: string }[]).some((tab) => tab.key === userProfileStore.currentTab)) {
    userProfileStore.currentTab = 'messages'
  }
})
</script>

<style scoped lang="scss">
.profile {
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
  min-width: 0;
}

/* —— 一体头部：色条 + 身份 + 档案同卡 —— */
.profile-hero {
  position: relative;
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
}
.profile-hero__wash {
  position: absolute;
  inset: 0 0 auto 0;
  height: 96px;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--el-color-primary) 18%, transparent) 0%,
    color-mix(in srgb, var(--el-color-primary) 6%, transparent) 55%,
    transparent 100%
  );
  pointer-events: none;
}
html.dark .profile-hero__wash {
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--el-color-primary) 22%, transparent) 0%,
    color-mix(in srgb, var(--el-color-primary) 8%, transparent) 50%,
    transparent 100%
  );
}
.profile-hero__top {
  position: relative;
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
  padding: 20px 20px 0;
}
.profile-hero__identity {
  display: flex;
  align-items: center;
  gap: 1rem;
  min-width: 0;
  flex: 1;
}
.profile-hero__text {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}
.profile-hero__title-row {
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
  max-width: min(100%, 22rem);
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
.profile-hero__actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
  padding-bottom: 2px;
}

/* 档案区：同一卡内，hairline 上分界，靠排版分层（01） */
.profile-hero__archive {
  position: relative;
  margin-top: 16px;
  padding: 14px 20px 18px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.profile-hero__archive-label {
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.02em;
  color: var(--el-text-color-secondary);
  margin-bottom: 0.65rem;
}
.profile-hero__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  margin-top: 0.75rem;
}
.profile-hero__chip {
  display: inline-flex;
  align-items: center;
  padding: 2px 9px;
  border-radius: 999px;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-extra-light);
}

/* —— Tabs —— */
.profile-main {
  min-width: 0;
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.profile-tabs {
  display: flex;
  gap: 2px;
  padding: 3px;
  max-width: 100%;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  overflow-x: auto;
  overscroll-behavior-x: contain;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
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
.profile-main__body {
  min-width: 0;
  width: 100%;
}

@media (width < 640px) {
  .profile-hero__top {
    align-items: flex-start;
    padding: 16px 16px 0;
  }
  .profile-hero__wash {
    height: 80px;
  }
  .profile-hero__actions {
    width: 100%;
  }
  .profile-hero__actions :deep(button) {
    flex: 1;
  }
  .profile-hero__archive {
    padding: 12px 16px 16px;
  }
}
</style>
