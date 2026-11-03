<!-- 403：外壳内居中的贴纸插画错误页（见 ../components/ErrorScene.vue）。 -->
<template>
  <ErrorScene code="403" />
</template>

<script setup lang="ts">
import ErrorScene from '@/views/exception/components/ErrorScene.vue'

defineOptions({ name: 'ForbiddenException' })
</script>
