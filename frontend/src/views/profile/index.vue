<!-- 个人中心（ProfilePC / ThemePC / ProfileMobile / ThemeMobile）：一个菜单 = 一个页面，页内标签 个人资料 / 外观（?tab=）。
     PC：小标题「个人中心」+ 当前标签名 + 下划线子导航；手机：返回 + 标题（外观页右上角「恢复默认」），从「个人资料」里的外观入口进入外观。 -->
<template>
  <div class="profile-page">
    <header v-if="!isMobile" class="profile-head">
      <div class="profile-head__eyebrow">{{ t('layout.profile') }}</div>
      <h1 class="profile-head__title">{{ current ? t(current.labelKey) : t('layout.profile') }}</h1>
    </header>
    <SubNavTabs v-if="!isMobile" group="profile" class="profile-tabs" />

    <PageHeader
      v-else
      :title="onAppearance ? t('theme.title') : t('layout.profile')"
      :back="onAppearance ? true : 'auto'"
      :theme-toggle="!onAppearance"
    >
      <template v-if="onAppearance" #actions>
        <button type="button" class="profile-reset" @click="themeStore.resetAppearance()">
          {{ t('theme.reset') }}
        </button>
      </template>
    </PageHeader>

    <PageGroupHost group="profile" />
  </div>
</template>

<script setup lang="ts">
import { usePageGroupTabs } from '@/config/pageGroups'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'ProfileView' })

const { t } = useI18n()
const menuStore = useMenuStore()
const themeStore = useThemeStore()
const { current } = usePageGroupTabs('profile')

const isMobile = computed(() => menuStore.isMobile)
const onAppearance = computed(() => current.value?.key === 'appearance')
</script>

<style scoped lang="scss">
.profile-page {
  display: flex;
  flex-direction: column;
  gap: 22px;
  width: 100%;
  min-width: 0;
}
.profile-head {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.profile-head__eyebrow {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.profile-head__title {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 2.125rem;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--el-text-color-primary);
}
.profile-tabs {
  margin-top: -8px;
}
.profile-reset {
  min-height: 44px;
  padding: 0 12px;
  border: 0;
  border-radius: 14px;
  background: transparent;
  color: var(--m-pInk);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 700;
  cursor: pointer;
}
.profile-reset:hover {
  background: var(--el-fill-color-light);
}
@media (width < 992px) {
  .profile-page {
    gap: 18px;
  }
}
</style>
