<template>
  <div class="file-page">
    <FileBrowser :key="instanceId" :scope="scope" :initial-path="initialPath" />
  </div>
</template>

<script setup lang="ts">
defineOptions({ name: 'InstanceFileView' })
import { useRoute } from 'vue-router'
import FileBrowser from '@/components/file/FileBrowser.vue'
import type { FileScope } from '@/components/file/types'

// 实例级文件管理页：scope=instance:{id}，由控制台跳转携带 ?workdir=（实例工作目录）。
// 实例切换时按 id 重挂载 FileBrowser，重建对应 scope 的客户端。
const route = useRoute()
const instanceId = computed(() => String(route.params.instanceId || ''))
const scope = computed<FileScope>(() => ({ kind: 'instance', id: instanceId.value }))
const initialPath = String(route.query.workdir || '')
</script>

<style scoped>
.file-page {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
</style>
