<!-- Docker 页内标签：容器 / 镜像 / 网络 / 连接配置（/docker?tab=…）；引擎已连接时标签上带数量。 -->
<template>
  <SubNavTabs group="docker" :counts="counts" />
</template>

<script setup lang="ts">
import { useDockerStatus } from './useDockerStatus'

const { status, refresh } = useDockerStatus()
const counts = computed(() => {
  const info = status.value?.connected ? status.value.info : undefined
  return {
    container: info ? info.containers : null,
    image: info ? info.images : null,
    network: info ? (status.value?.networks ?? null) : null,
  }
})
onMounted(() => {
  void refresh()
})
</script>
