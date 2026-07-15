<!-- 我的资料：左档案摘要 + 右编辑表单 -->
<template>
  <div class="info-layout">
    <ArchivesPanel class="info-layout__side" />
    <PersonalInfoPanel ref="formRef" class="info-layout__main" />
  </div>
</template>

<script setup lang="ts">
import ArchivesPanel from '@/views/profile/archivesPanel.vue'
import PersonalInfoPanel from '@/views/profile/personalInfoPanel.vue'

const formRef = ref<{ save: () => void } | null>(null)

defineExpose({
  save: () => formRef.value?.save(),
})
</script>

<style scoped lang="scss">
.info-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 12px;
}
@media (width >= 1024px) {
  .info-layout {
    grid-template-columns: minmax(220px, 280px) minmax(0, 1fr);
    align-items: start;
  }
}
</style>
