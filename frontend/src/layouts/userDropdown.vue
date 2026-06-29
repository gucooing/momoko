<template>
  <el-dropdown
    @command="handleCommand"
    trigger="click"
    :show-arrow="false"
    placement="bottom-end"
    popper-class="user-dropdown-popper"
  >
    <div class="user-card">
      <div class="avatar-wrapper">
        <el-avatar :size="36" :src="userStore.resolvedUserAvatar" />
        <span class="status-badge"></span>
      </div>
      <div class="user-info">
        <span class="username ellipsis-text">{{
          userStore.userInfo?.name || userStore.userInfo?.username
        }}</span>
        <BaseTag :text="userRoleName" size="small" />
      </div>
    </div>

    <template #dropdown>
      <div class="user-menu-wrapper">
        <!-- 用户信息头部 -->
        <div class="user-header">
          <div class="avatar-wrapper">
            <el-avatar :size="48" :src="userStore.resolvedUserAvatar" />
            <span class="status-badge"></span>
          </div>
          <div class="user-info">
            <div class="name-row">
              <span class="user-name ellipsis-text">{{
                userStore.userInfo?.name || userStore.userInfo?.username
              }}</span>
              <BaseTag :text="userRoleName" size="small" />
            </div>
            <div class="user-email">{{ userStore.userInfo?.email || '' }}</div>
          </div>
        </div>

        <!-- 菜单项 -->
        <el-dropdown-menu class="user-menu">
          <el-dropdown-item command="profile">
            <el-icon>
              <component :is="menuStore.iconComponents['HOutline:UserCircleIcon']" />
            </el-icon>
            <span>{{ t('layout.profile') }}</span>
          </el-dropdown-item>
          <el-dropdown-item command="docs">
            <el-icon>
              <component :is="menuStore.iconComponents['HOutline:DocumentTextIcon']" />
            </el-icon>
            <span>{{ t('layout.docs') }}</span>
          </el-dropdown-item>
          <el-dropdown-item command="github">
            <el-icon><IconGithub /></el-icon>
            <span>GitHub</span>
          </el-dropdown-item>
          <el-dropdown-item v-if="canCheckUpdate" command="checkUpdate">
            <el-icon>
              <component :is="menuStore.iconComponents['HOutline:ArrowPathIcon']" />
            </el-icon>
            <span>{{ t('layout.version', { version: currentVersion }) }}</span>
          </el-dropdown-item>
          <el-dropdown-item divided command="password">
            <el-icon>
              <component :is="menuStore.iconComponents['HOutline:KeyIcon']" />
            </el-icon>
            <span>{{ t('layout.changePassword') }}</span>
          </el-dropdown-item>
          <el-dropdown-item command="logout">
            <el-icon>
              <component :is="menuStore.iconComponents['HOutline:ArrowRightOnRectangleIcon']" />
            </el-icon>
            <span>{{ t('layout.logout') }}</span>
            <span class="shortcut">⌥ Q</span>
          </el-dropdown-item>
        </el-dropdown-menu>
      </div>
    </template>
  </el-dropdown>
  <UpdatePassword ref="updatePasswordRef" />
</template>

<script setup lang="ts">
import IconGithub from '@/components/icons/IconGithub.vue'
import { useUserProfileStore } from '@/stores/user/profile'
import { Dialog } from '@/utils/dialog'
import { checkForUpdate } from '@/utils/updateCheck'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const menuStore = useMenuStore()
const userStore = useUserStore()
const userProfileStore = useUserProfileStore()
const updatePasswordRef = useTemplateRef('updatePasswordRef')
const PROJECT_LINK = 'https://github.com/gucooing/momoko'
const { t } = useI18n()

// 用户角色名称
const userRoleName = computed(() => userStore.getUserRoleName())

// 更新检查：仅有 system:update 权限时显示该项；版本号取自权限响应。
const canCheckUpdate = computed(() => menuStore.hasButtonPermission('system:update'))
const currentVersion = computed(() => menuStore.currentVersion || 'dev')
const checkingUpdate = ref(false)

const onCheckUpdate = async () => {
  if (checkingUpdate.value) return
  checkingUpdate.value = true
  try {
    await checkForUpdate()
  } finally {
    checkingUpdate.value = false
  }
}

// 用户菜单命令处理
const showLogoutConfirm = () => {
  Dialog.info({
    showCancelButton: true,
    content: t('layout.logoutConfirmContent'),
    confirmText: t('layout.logoutConfirmText'),
    cancelText: t('layout.logoutCancelText'),
    onConfirm: async () => {
      await userStore.logout()
    },
  })
}

const handleCommand = (command: string) => {
  switch (command) {
    case 'profile':
      userProfileStore.currentTab = 'personalInfo'
      router.push('/profile')
      break
    case 'docs':
    case 'github':
      window.open(PROJECT_LINK, '_blank')
      break
    case 'checkUpdate':
      onCheckUpdate()
      break
    case 'password':
      updatePasswordRef.value?.showDialog()
      break
    case 'logout':
      showLogoutConfirm()
      break
  }
}

// 监听快捷键：Alt/Option + Q 触发退出登录弹窗
const handleKeydown = (event: KeyboardEvent) => {
  const target = event.target as HTMLElement | null
  const isTyping =
    (target && ['INPUT', 'TEXTAREA'].includes(target.tagName)) ||
    target?.getAttribute('contenteditable') === 'true'

  if (isTyping) return

  // Mac 上 Option+Q 常返回 Dead key，改用 code 判断物理键位
  if (event.altKey && event.code === 'KeyQ') {
    event.preventDefault()
    showLogoutConfirm()
  }
}

onMounted(() => {
  userStore.getUserInfo()
  window.addEventListener('keydown', handleKeydown)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped lang="scss">
.user-card {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding: 4px 8px 4px 4px;
  border-radius: 6px;
  cursor: pointer;
  transition: background-color 0.2s;

  .avatar-wrapper {
    position: relative;
    flex-shrink: 0;

    .status-badge {
      position: absolute;
      bottom: 0;
      right: 0;
      width: 10px;
      height: 10px;
      background: #52c41a;
      border: 2px solid var(--el-bg-color);
      border-radius: 50%;
    }
  }

  .user-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 4px;

    .username {
      font-size: 14px;
      font-weight: 600;
      color: var(--el-text-color-primary);
      line-height: 1.2;
      max-width: 100px;
    }
  }
}

.user-menu-wrapper {
  background: var(--el-bg-color);
  border-radius: 8px;
  overflow: hidden;
}

.user-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);

  .avatar-wrapper {
    position: relative;
    flex-shrink: 0;

    .status-badge {
      position: absolute;
      bottom: 0;
      right: 0;
      width: 12px;
      height: 12px;
      background: #52c41a;
      border: 2px solid var(--el-bg-color);
      border-radius: 50%;
    }
  }

  .user-info {
    flex: 1;
    min-width: 0;

    .name-row {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 4px;

      .user-name {
        font-size: 16px;
        font-weight: 600;
        color: var(--el-text-color-primary);
        line-height: 1.2;
        max-width: 100px;
      }
    }

    .user-email {
      font-size: 13px;
      color: var(--el-text-color-regular);
      opacity: 0.8;
      line-height: 1.2;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }
}

.avatar-wrapper {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

:deep(.user-menu) {
  padding: 4px 0;
  min-width: 200px;
  background: var(--el-bg-color);

  .el-dropdown-menu__item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 10px 16px;
    transition: background-color 0.2s;
    background: transparent;

    &:hover {
      background: var(--el-fill-color-light);
      color: var(--el-color-primary);
    }

    .el-icon {
      font-size: 1.25rem;
      flex-shrink: 0;
    }

    span:not(.shortcut) {
      font-size: 14px;
      flex: 1;
    }

    .shortcut {
      font-size: 12px;
      color: var(--el-text-color-placeholder);
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      opacity: 0.6;
    }
  }
}
</style>

<style lang="scss">
.user-dropdown-popper {
  border-radius: 8px !important;
  .el-dropdown-menu {
    border-radius: 8px !important;
  }
}
</style>
