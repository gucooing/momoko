<!-- 任务与日志页内标签：后台任务 / 操作记录（/system/logs?tab=…）；counts 键为标签 key（task / operation）。 -->
<template>
  <SubNavTabs group="logs" :counts="counts" />
</template>

<script setup lang="ts">
defineProps<{ counts?: Partial<Record<string, number | null>> }>()
</script>
