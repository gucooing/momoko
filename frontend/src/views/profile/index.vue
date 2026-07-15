<!-- 个人中心 · 真·从零（06a P2 选项卡式）
     结构：克制色条头部 → 桌面 左资料常驻 | 右 Tabs(消息/权限/设备/日志)；移动 头压缩 + Tab 横滑 + 编辑 FormDialog。
     保留 useUserProfileStore 数据流；资料不在 Tab 内。 -->
<template>
  <div class="profile">
    <!-- ① 克制头部：低饱和主色浅条 + 头像/名/角色/简介 + 关键操作 -->
    <header class="profile-hero">
      <div class="profile-hero__wash" aria-hidden="true" />
      <div class="profile-hero__inner">
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
            <div v-if="locationText" class="profile-hero__loc">
              <component
                :is="menuStore.iconComponents['HOutline:MapPinIcon']"
                class="profile-hero__loc-icon"
              />
              <span>{{ locationText }}</span>
            </div>
          </div>
        </div>
        <div class="profile-hero__actions">
          <UButton
            color="neutral"
            variant="soft"
            size="sm"
            icon="i-lucide-pencil"
            @click="openEdit"
          >
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
    </header>

    <!-- ② 主体：左资料 | 右 Tabs -->
    <div class="profile-layout">
      <!-- 左栏：只读档案（桌面常驻；移动在资料区上方） -->
      <aside class="profile-side">
        <ArchivesPanel />
        <!-- 桌面：内嵌编辑表单；移动：只显示入口，编辑走全屏 FormDialog -->
        <div v-if="!isCompact" class="profile-side__edit">
          <PersonalInfoPanel />
        </div>
        <div v-else class="profile-side__mobile-edit">
          <UButton
            block
            color="primary"
            variant="soft"
            icon="i-lucide-pencil"
            @click="editOpen = true"
          >
            {{ t('user.editProfile') }}
          </UButton>
        </div>
      </aside>

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
    </div>

    <!-- 移动：全屏资料编辑 -->
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
import { useUserProfileStore } from '@/stores/user/profile'
import type { ProfileCurrentTab } from '@/stores/user/types'
import ArchivesPanel from '@/views/profile/archivesPanel.vue'
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

/** 与 02 一致：结构级切换用 isMobile（项目现阈值 ~992，等同紧凑态） */
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
const locationText = computed(() => {
  const { country, region, city } = userProfileStore.address
  return [country, region, city].filter(Boolean).join(' · ')
})

const openPassword = () => updatePasswordRef.value?.showDialog()
const openEdit = () => {
  if (isCompact.value) {
    editOpen.value = true
    return
  }
  // 桌面：滚到左栏编辑区
  document.querySelector('.profile-side__edit')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

onMounted(() => {
  void userProfileStore.ensureAddress()
  // 旧会话可能仍是 personalInfo
  if (!(userProfileStore.menuTabs as { key: string }[]).some((t) => t.key === userProfileStore.currentTab)) {
    userProfileStore.currentTab = 'messages'
  }
})
</script>

<style scoped lang="scss">
.profile {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 1200px;
}

/* —— 头部：克制浅主色条（01：唯一允许主色面积处，仍矮、淡）—— */
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
  height: 88px;
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
.profile-hero__inner {
  position: relative;
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
  padding: 20px 20px 18px;
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
  max-width: min(100%, 18rem);
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

/* —— 双栏布局 —— */
.profile-layout {
  display: grid;
  grid-template-columns: 1fr;
  gap: 16px;
  align-items: start;
}
@media (width >= 1024px) {
  .profile-layout {
    grid-template-columns: minmax(260px, 320px) minmax(0, 1fr);
  }
}
.profile-side {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}
.profile-side__mobile-edit {
  /* 移动：档案下的编辑入口 */
}

.profile-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
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
.profile-main__body {
  min-width: 0;
}

@media (width < 640px) {
  .profile-hero__inner {
    align-items: flex-start;
    padding: 16px;
  }
  .profile-hero__wash {
    height: 72px;
  }
  .profile-hero__actions {
    width: 100%;
  }
  .profile-hero__actions :deep(button) {
    flex: 1;
  }
}
</style>
