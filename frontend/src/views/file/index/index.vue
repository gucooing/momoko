<template>
  <div class="file-page">
    <FileBrowser :scope="scope" :initial-path="initialPath">
      <template #nav>
      </template>
    </FileBrowser>
  </div>
</template>

<script setup lang="ts">
defineOptions({ name: 'SystemFileView' })
import { useRoute } from 'vue-router'
import FileBrowser from '@/components/file/FileBrowser.vue'
import { useFullBleed } from '@/composables/useAppLayout'
import type { FileScope } from '@/components/file/types'

// 系统级文件管理页：scope=system，可由 ?workdir= 指定初始目录。
const route = useRoute()
const scope: FileScope = { kind: 'system' }
const initialPath = String(route.query.workdir || '')

// 全出血：FileBrowser 自带页面内边距与滚动（目录树吸顶、批量条吸底都依赖它自己的滚动容器）。
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
