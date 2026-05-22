<!-- 用户信息面板 -->
<template>
  <BaseCard>
    <div class="flex items-center justify-between flex-col xl:flex-row gap-8 mt-4">
      <div class="flex items-center gap-2 flex-col md:flex-row md:gap-8">
        <HoverAnimateWrapper name="flip">
          <div class="relative shrink-0">
            <el-avatar :size="110" :src="userStore.resolvedUserAvatar" />
            <div
              class="absolute h-5 w-5 bottom-2 right-2 rounded-full border-3 border-(--el-bg-color) bg-(--el-color-success)"
            ></div>
          </div>
        </HoverAnimateWrapper>

        <div class="flex flex-col gap-4 items-center md:items-start text-center md:text-left">
          <div class="flex items-center gap-2">
            <TextEllipsis
              :text="userStore.userInfo?.name! || userStore.userInfo?.username!"
              :clickable="false"
              class="text-2xl font-extrabold"
            />
            <data>
              <IconButton icon="HOutline:CheckBadgeIcon" type="primary" tooltip="实名认证用户" />
            </data>
          </div>
          <TextEllipsis
            :text="`“ ${userStore.userInfo?.bio} ”`"
            class="italic text-sm text-(--el-text-color-regular)"
          />
          <div
            class="flex items-center gap-2 text-sm font-semibold px-3 py-2 text-(--el-text-color-primary) bg-(--el-bg-color-page) rounded-lg"
          >
            <el-icon>
              <component
                :is="menuStore.iconComponents['HOutline:MapPinIcon']"
                class="text-indigo-500"
              />
            </el-icon>
            <span class="text-xs"
              >{{ userProfileStore.address.country }} · {{ userProfileStore.address.region }} ·
              {{ userProfileStore.address.city }}</span
            >
          </div>
        </div>
      </div>
    </div>
    <div class="mt-9 flex justify-center xl:justify-start">
      <div class="max-w-full">
        <BadgeTabsMenu
          v-model="userProfileStore.currentTab"
          :icon-only="menuStore.isMobile ? true : false"
          :tabs-menu-data="userProfileStore.menuTabs"
        />
      </div>
    </div>
  </BaseCard>
</template>

<script setup lang="ts">
import { useUserProfileStore } from '@/stores/user/profile'
const userStore = useUserStore()
const menuStore = useMenuStore()
const userProfileStore = useUserProfileStore()

onMounted(() => {
  void userProfileStore.ensureAddress()
})
</script>

<style scoped lang="scss">
.active {
  color: var(--el-color-primary);
  &::after {
    content: '';
    position: absolute;
    left: 0;
    bottom: 0;
    width: 100%;
    height: 0.25rem;
    background: var(--el-color-primary);
    border-radius: 4px 4px 0 0;
  }
}
</style>
