<!-- 兼容壳：资料已拆到左栏 ArchivesPanel + PersonalInfoPanel（06a）。
     保留文件以免旧 import 断裂；不再作为独立 Tab 使用。 -->
<template>
  <div class="info-compat">
    <ArchivesPanel />
    <PersonalInfoPanel />
  </div>
</template>

<script setup lang="ts">
import ArchivesPanel from '@/views/profile/archivesPanel.vue'
import PersonalInfoPanel from '@/views/profile/personalInfoPanel.vue'
</script>

<style scoped lang="scss">
.info-compat {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
