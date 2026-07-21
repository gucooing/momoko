<template>
  <div class="file-page">
    <FileBrowser :key="instanceId" :scope="scope" :initial-path="initialPath" />
  </div>
</template>

<script setup lang="ts">
defineOptions({ name: 'InstanceFileView' })
import { useRoute } from 'vue-router'
import FileBrowser from '@/components/file/FileBrowser.vue'
import { useFullBleed } from '@/composables/useAppLayout'
import type { FileScope } from '@/components/file/types'

// 实例级文件管理页：scope=instance:{id}。
// 后端实例文件 API 已以实例目录为根，默认 path='' 即根；?workdir= 仅允许相对该根的子路径。
// 实例切换时按 id 重挂载 FileBrowser。全出血：与系统文件页 / 终端同理。
const route = useRoute()
const instanceId = computed(() => String(route.params.instanceId || ''))
const scope = computed<FileScope>(() => ({ kind: 'instance', id: instanceId.value }))
// 仅接受相对实例根的子路径；丢弃 host 绝对/相对实例路径（如 ./servers/mc）
const rawWorkdir = String(route.query.workdir || '')
const initialPath =
  rawWorkdir && !rawWorkdir.startsWith('.') && !rawWorkdir.includes(':') && !rawWorkdir.startsWith('/')
    ? rawWorkdir
    : ''
useFullBleed()
</script>

<style scoped>
.file-page {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
</style>
