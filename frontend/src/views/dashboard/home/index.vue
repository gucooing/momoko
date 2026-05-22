<template>
  <div class="flex-1 h-full flex flex-col gap-6">
    <WelcomePanel />
    <el-row :gutter="20">
      <el-col :xs="24" :lg="16">
        <RunningInstanceSection />
      </el-col>
      <el-col :xs="24" :lg="8" class="mt-5 lg:mt-0">
        <ShortcutSection class="mt-4 min-[1200px]:mt-0" />
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import WelcomePanel from '@/views/dashboard/home/welcomePanel.vue'
import RunningInstanceSection from '@/views/dashboard/home/runningInstanceSection.vue'
import ShortcutSection from '@/views/dashboard/home/shortcutSection.vue'

defineOptions({ name: 'HomeView' })
</script>

<style></style>
