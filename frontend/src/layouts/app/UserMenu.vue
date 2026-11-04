<!-- 侧栏底部账号卡（复用 userStore/logout/改密/更新检查逻辑）。
     头像 + 昵称点击进入个人中心（当前在个人中心时高亮）；「⋯」打开账号菜单：个人中心 / 外观 / 语言 / 全屏 / GitHub / 检查更新 / 修改密码 / 退出。
     折叠侧栏时只显示头像，点击直接打开账号菜单。 -->
<template>
  <div class="user-card" :class="{ 'is-on': onProfile, 'is-collapsed': collapsed }">
    <button
      v-if="!collapsed"
      type="button"
      class="user-card__main"
      :aria-current="onProfile ? 'page' : undefined"
      @click="goProfile()"
    >
      <span class="user-card__av">
        <AppAvatar :src="userStore.resolvedUserAvatar" :size="40" :name="displayName" />
      </span>
      <span class="user-card__text">
        <span class="user-card__name">{{ displayName }}</span>
        <span class="user-card__sub">{{ onProfile ? t('layout.profile') : t('shell.user.cardHint') }}</span>
      </span>
    </button>

    <AppDropdown align="start" side="top" :width="256">
      <template #trigger="{ open }">
        <button
          v-if="collapsed"
          type="button"
          class="user-card__av user-card__av--btn"
          :aria-label="t('shell.user.menu')"
          :aria-expanded="open"
        >
          <AppAvatar :src="userStore.resolvedUserAvatar" :size="40" :name="displayName" />
        </button>
        <AppIconButton
          v-else
          icon="HOutline:EllipsisHorizontalIcon"
          :label="t('shell.user.menu')"
          :box="36"
          :size="20"
          :active="open"
          :aria-expanded="open"
        />
      </template>

      <template #default="{ close }">
        <div class="user-menu">
          <div class="user-menu__head">
            <AppAvatar :src="userStore.resolvedUserAvatar" :size="44" :name="displayName" />
            <div class="user-menu__id">
              <div class="user-menu__name">{{ displayName }}</div>
              <div class="user-menu__email">{{ userStore.userInfo?.email || roleName }}</div>
            </div>
          </div>
          <div class="user-menu__items">
            <button type="button" class="user-menu__item" @click="goProfile(close)">
              <component :is="ic('HOutline:UserCircleIcon')" />
              <span>{{ t('layout.profile') }}</span>
            </button>
            <button type="button" class="user-menu__item" @click="goAppearance(close)">
              <component :is="ic('HOutline:SwatchIcon')" />
              <span>{{ t('theme.title') }}</span>
              <small>{{ appearanceSummary }}</small>
            </button>
            <button
              type="button"
              class="user-menu__item"
              :aria-expanded="langOpen"
              @click="langOpen = !langOpen"
            >
              <component :is="ic('HOutline:LanguageIcon')" />
              <span>{{ t('language.tooltip') }}</span>
              <small>{{ currentLanguage }}</small>
              <component :is="ic(langOpen ? 'HOutline:ChevronUpIcon' : 'HOutline:ChevronDownIcon')" class="user-menu__chev" />
            </button>
            <div v-if="langOpen" class="user-menu__langs">
              <button
                v-for="item in languageOptions"
                :key="item.code"
                type="button"
                class="user-menu__lang"
                :class="{ 'is-on': locale === item.code }"
                :aria-pressed="locale === item.code"
                @click="pickLocale(item.code, close)"
              >
                <span class="user-menu__lang-code">{{ item.shortLabel }}</span>
                <span>{{ t(item.labelKey) }}</span>
                <component :is="ic('HOutline:CheckIcon')" v-if="locale === item.code" class="user-menu__check" />
              </button>
            </div>
            <button type="button" class="user-menu__item" @click="onFullscreen(close)">
              <component :is="ic(isFullscreen ? 'HOutline:ArrowsPointingInIcon' : 'HOutline:ArrowsPointingOutIcon')" />
              <span>{{ isFullscreen ? t('layout.exitFullscreen') : t('layout.fullscreen') }}</span>
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
              <component :is="ic('HOutline:ArrowPathIcon')" />
              <span>{{ t('layout.checkUpdate') }}</span>
            </button>
            <div class="user-menu__divider" />
            <button type="button" class="user-menu__item" @click="openPassword(close)">
              <component :is="ic('HOutline:KeyIcon')" />
              <span>{{ t('layout.changePassword') }}</span>
            </button>
            <button
              type="button"
              class="user-menu__item user-menu__item--danger"
              @click="onLogout(close)"
            >
              <component :is="ic('HOutline:ArrowRightOnRectangleIcon')" />
              <span>{{ t('layout.logout') }}</span>
            </button>
          </div>
          <div class="user-menu__foot m-num">{{ t('layout.version', { version: currentVersion }) }}</div>
        </div>
      </template>
    </AppDropdown>
    <UpdatePassword ref="updatePasswordRef" />
  </div>
</template>

<script setup lang="ts">
import IconGithub from '@/components/icons/IconGithub.vue'
import { useFullscreen } from '@vueuse/core'
import { languageOptions, setAppLocale } from '@/locales'
import { useUserProfileStore } from '@/stores/user/profile'
import { Dialog } from '@/utils/dialog'
import { checkForUpdate } from '@/utils/updateCheck'
import { useI18n } from 'vue-i18n'

withDefaults(defineProps<{ collapsed?: boolean }>(), { collapsed: false })

const route = useRoute()
const router = useRouter()
const menuStore = useMenuStore()
const userStore = useUserStore()
const themeStore = useThemeStore()
const userProfileStore = useUserProfileStore()
const updatePasswordRef = ref<{ showDialog: () => void }>()
const PROJECT_LINK = 'https://github.com/gucooing/momoko'
const { t, locale } = useI18n()
const ic = (name: string) => menuStore.iconComponents[name]
const { isFullscreen, toggle: toggleFullscreen } = useFullscreen()
const langOpen = ref(false)

const displayName = computed(() => userStore.userInfo?.name || userStore.userInfo?.username || '—')
const roleName = computed(() => userStore.getUserRoleName())
const canCheckUpdate = computed(() => menuStore.hasButtonPermission('system:update'))
const currentVersion = computed(() => menuStore.currentVersion || 'dev')
const onProfile = computed(() => route.path === '/profile')
const appearanceSummary = computed(
  () => `${t(themeStore.currentPalette.labelKey)} · ${t(`theme.${themeStore.themeMode}`)}`,
)
const currentLanguage = computed(() => {
  const item = languageOptions.find((o) => o.code === locale.value)
  return item ? t(item.labelKey) : ''
})

const goProfile = (close?: () => void) => {
  close?.()
  userProfileStore.currentTab = 'messages'
  if (route.path !== '/profile' || route.query.tab === 'appearance') {
    void router.push({ path: '/profile', query: { tab: 'info' } })
  }
}
const goAppearance = (close: () => void) => {
  close()
  void router.push({ path: '/profile', query: { tab: 'appearance' } })
}
const pickLocale = (code: string, close: () => void) => {
  setAppLocale(code as never)
  langOpen.value = false
  close()
}
const onFullscreen = (close: () => void) => {
  close()
  void toggleFullscreen()
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
.user-card {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px;
  border-radius: 18px;
  background: var(--el-bg-color-page);
  transition: background 0.15s;
}
.user-card.is-on {
  background: var(--m-pSoft);
}
.user-card.is-collapsed {
  justify-content: center;
  padding: 6px;
}
.user-card__main {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.user-card__av {
  display: inline-flex;
  flex: none;
  border-radius: 50%;
  :deep(.app-avatar) {
    background: var(--m-a);
    color: var(--m-aOn);
  }
}
.user-card__av--btn {
  padding: 0;
  border: none;
  background: transparent;
  cursor: pointer;
}
.user-card__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.user-card__name {
  font-size: 0.875rem;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.user-card__sub {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.user-card.is-on .user-card__sub {
  color: var(--m-pInk);
  font-weight: 600;
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
  font-weight: 800;
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
  min-height: 40px;
  padding: 0 10px;
  border: none;
  background: transparent;
  border-radius: 12px;
  color: var(--el-text-color-regular);
  font: inherit;
  font-size: 0.875rem;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.user-menu__item > span {
  flex: 1;
  min-width: 0;
}
.user-menu__item small {
  max-width: 110px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.user-menu__item :deep(svg) {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}
.user-menu__item :deep(svg.user-menu__chev) {
  width: 14px;
  height: 14px;
  color: var(--el-text-color-secondary);
}
.user-menu__item:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.user-menu__item--danger {
  color: var(--m-bad);
}
.user-menu__item--danger:hover {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.user-menu__langs {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 2px 0 4px 28px;
}
.user-menu__lang {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 36px;
  padding: 0 10px;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: var(--el-text-color-regular);
  font: inherit;
  font-size: 0.8125rem;
  text-align: left;
  cursor: pointer;
}
.user-menu__lang > span:nth-child(2) {
  flex: 1;
}
.user-menu__lang:hover {
  background: var(--el-fill-color-light);
}
.user-menu__lang.is-on {
  color: var(--m-pInk);
  font-weight: 700;
}
.user-menu__lang-code {
  min-width: 28px;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.user-menu__check {
  width: 16px;
  height: 16px;
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
  color: var(--el-text-color-secondary);
}
</style>
