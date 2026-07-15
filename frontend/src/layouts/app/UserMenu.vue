<!-- 用户菜单（复用 userStore/logout/改密/更新检查逻辑，重写视觉）。
     variant='topbar'：顶栏右上角横向触发器，向下弹出。
     variant='sidebar'：侧栏底部整行触发器，向上弹出，展开显示 用户信息/版本号/操作。 -->
<template>
  <AppDropdown
    :align="isSidebar ? 'start' : 'end'"
    :side="isSidebar ? 'top' : 'bottom'"
    :width="isSidebar ? 236 : 248"
    :block="isSidebar"
  >
    <template #trigger>
      <!-- 侧栏触发器：整行，折叠时仅头像 -->
      <button
        v-if="isSidebar"
        type="button"
        class="user-side"
        :class="{ 'user-side--collapsed': collapsed }"
      >
        <AppAvatar :src="userStore.resolvedUserAvatar" :size="34" :name="displayName" online />
        <div v-show="!collapsed" class="user-side__info">
          <span class="user-side__name">{{ displayName }}</span>
          <span class="user-side__role">{{ roleName }}</span>
        </div>
        <component
          v-show="!collapsed"
          :is="menuStore.iconComponents['HOutline:ChevronUpDownIcon']"
          class="user-side__chev"
        />
      </button>

      <!-- 顶栏触发器（默认） -->
      <button v-else type="button" class="user-trigger">
        <AppAvatar :src="userStore.resolvedUserAvatar" :size="34" :name="displayName" online />
        <div class="user-trigger__info">
          <span class="user-trigger__name">{{ displayName }}</span>
          <span class="user-trigger__role">{{ roleName }}</span>
        </div>
        <component
          :is="menuStore.iconComponents['HOutline:ChevronDownIcon']"
          class="user-trigger__chev"
        />
      </button>
    </template>

    <template #default="{ close }">
      <div class="user-menu">
        <div class="user-menu__head">
          <AppAvatar :src="userStore.resolvedUserAvatar" :size="44" :name="displayName" online />
          <div class="user-menu__id">
            <div class="user-menu__name">{{ displayName }}</div>
            <div class="user-menu__email">{{ userStore.userInfo?.email || '—' }}</div>
          </div>
        </div>
        <div class="user-menu__items">
          <button type="button" class="user-menu__item" @click="goProfile(close)">
            <component :is="menuStore.iconComponents['HOutline:UserCircleIcon']" />
            <span>{{ t('layout.profile') }}</span>
          </button>
          <button type="button" class="user-menu__item" @click="openGithub(close)">
            <IconGithub class="user-menu__gh" />
            <span>GitHub</span>
          </button>
          <button
            v-if="canCheckUpdate"
            type="button"
            class="user-menu__item"
            @click="doCheckUpdate(close)"
          >
            <component :is="menuStore.iconComponents['HOutline:ArrowPathIcon']" />
            <span>{{ t('layout.checkUpdate') }}</span>
          </button>
          <div class="user-menu__divider" />
          <button type="button" class="user-menu__item" @click="openPassword(close)">
            <component :is="menuStore.iconComponents['HOutline:KeyIcon']" />
            <span>{{ t('layout.changePassword') }}</span>
          </button>
          <button
            type="button"
            class="user-menu__item user-menu__item--danger"
            @click="onLogout(close)"
          >
            <component :is="menuStore.iconComponents['HOutline:ArrowRightOnRectangleIcon']" />
            <span>{{ t('layout.logout') }}</span>
          </button>
        </div>
        <div class="user-menu__foot">{{ t('layout.version', { version: currentVersion }) }}</div>
      </div>
    </template>
  </AppDropdown>
  <UpdatePassword ref="updatePasswordRef" />
</template>

<script setup lang="ts">
import IconGithub from '@/components/icons/IconGithub.vue'
import { useUserProfileStore } from '@/stores/user/profile'
import { Dialog } from '@/utils/dialog'
import { checkForUpdate } from '@/utils/updateCheck'
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{ variant?: 'topbar' | 'sidebar'; collapsed?: boolean }>(),
  { variant: 'topbar' },
)
const isSidebar = computed(() => props.variant === 'sidebar')

const router = useRouter()
const menuStore = useMenuStore()
const userStore = useUserStore()
const userProfileStore = useUserProfileStore()
const updatePasswordRef = ref<{ showDialog: () => void }>()
const PROJECT_LINK = 'https://github.com/gucooing/momoko'
const { t } = useI18n()

const displayName = computed(() => userStore.userInfo?.name || userStore.userInfo?.username || '—')
const roleName = computed(() => userStore.getUserRoleName())
const canCheckUpdate = computed(() => menuStore.hasButtonPermission('system:update'))
const currentVersion = computed(() => menuStore.currentVersion || 'dev')

const goProfile = (close: () => void) => {
  // 资料在左栏常驻，进入页默认消息 Tab（06a）
  userProfileStore.currentTab = 'messages'
  router.push('/profile')
  close()
}
const openGithub = (close: () => void) => {
  window.open(PROJECT_LINK, '_blank')
  close()
}
const doCheckUpdate = (close: () => void) => {
  close()
  void checkForUpdate()
}
const openPassword = (close: () => void) => {
  close()
  updatePasswordRef.value?.showDialog()
}
const onLogout = (close: () => void) => {
  close()
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
</script>

<style scoped lang="scss">
/* ── 顶栏触发器 ── */
.user-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px 4px 4px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-sm);
  cursor: pointer;
  transition: background 0.15s;
}
.user-trigger:hover {
  background: var(--el-fill-color-light);
}
.user-trigger__info {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  min-width: 0;
  max-width: 130px;
}
.user-trigger__name {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.25;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}
.user-trigger__role {
  font-size: 0.6875rem;
  color: var(--el-text-color-secondary);
  line-height: 1.2;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}
.user-trigger__chev {
  width: 15px;
  height: 15px;
  color: var(--el-text-color-placeholder);
  flex-shrink: 0;
}
@media (width <= 768px) {
  .user-trigger__info {
    display: none;
  }
  .user-trigger__chev {
    display: none;
  }
}

/* ── 侧栏触发器 ── */
.user-side {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
  width: 100%;
  padding: 6px 8px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-md);
  cursor: pointer;
  text-align: left;
  transition: background 0.15s;
}
.user-side:hover {
  background: var(--el-fill-color-light);
}
.user-side--collapsed {
  justify-content: center;
  gap: 0;
  padding: 6px;
}
.user-side__info {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}
.user-side__name {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.user-side__role {
  font-size: 0.6875rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.user-side__chev {
  width: 16px;
  height: 16px;
  color: var(--el-text-color-placeholder);
  flex-shrink: 0;
}

/* ── 下拉面板 ── */
.user-menu__head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.user-menu__id {
  min-width: 0;
}
.user-menu__name {
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.user-menu__email {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.user-menu__items {
  padding: 6px;
}
.user-menu__item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 9px 10px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-sm);
  color: var(--el-text-color-regular);
  font-size: 0.875rem;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.user-menu__item :deep(svg) {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}
.user-menu__item:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.user-menu__item--danger {
  color: var(--el-color-danger, #ef4444);
}
.user-menu__item--danger:hover {
  background: color-mix(in srgb, var(--el-color-danger, #ef4444) 10%, transparent);
  color: var(--el-color-danger, #ef4444);
}
.user-menu__gh {
  width: 18px;
  height: 18px;
}
.user-menu__divider {
  height: 1px;
  margin: 6px 4px;
  background: var(--el-border-color-lighter);
}
.user-menu__foot {
  padding: 10px 16px;
  border-top: 1px solid var(--el-border-color-lighter);
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
  font-variant-numeric: tabular-nums;
}
</style>
