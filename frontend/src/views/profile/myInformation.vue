<!-- 我的资料 -->
<template>
  <el-row :gutter="20">
    <el-col :lg="6">
      <ArchivesPanel />
    </el-col>
    <el-col :lg="18" class="mt-4 min-[1200px]:mt-0">
      <PersonalInfoPanel />
    </el-col>
  </el-row>
</template>

<script setup lang="ts">
import ArchivesPanel from '@/views/profile/archivesPanel.vue'
import PersonalInfoPanel from '@/views/profile/personalInfoPanel.vue'
</script>

<style></style>
