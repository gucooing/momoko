<template>
  <div>
    <GradientHeader />
    <div class="relative left-0 -top-50 px-2 md:px-10">
      <userMainPanel />
      <div class="mt-4">
        <Transition name="slide-up" :style="{ '--animation-duration': '0.6s' }" mode="out-in">
          <div :key="userProfileStore.currentTab">
            <MyInformation v-if="userProfileStore.currentTab === 'personalInfo'" />
            <MyPermission v-else-if="userProfileStore.currentTab === 'permissions'" />
            <MyMessages v-else-if="userProfileStore.currentTab === 'messages'" />
            <LoginLogs v-else-if="userProfileStore.currentTab === 'logs'" />
            <LoginDevices v-else-if="userProfileStore.currentTab === 'devices'" />
            <MyInformation v-else />
          </div>
        </Transition>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useUserProfileStore } from '@/stores/user/profile'
import GradientHeader from '@/views/profile/gradientHeader.vue'
import userMainPanel from '@/views/profile/userMainPanel.vue'
import MyInformation from '@/views/profile/myInformation.vue'
import MyPermission from '@/views/profile/myPermission.vue'
import MyMessages from '@/views/profile/myMessages.vue'
import LoginLogs from '@/views/profile/loginLogs.vue'
import LoginDevices from '@/views/profile/loginDevices.vue'

defineOptions({ name: 'ProfileView' })

const userProfileStore = useUserProfileStore()
</script>

<style scoped lang="scss"></style>
