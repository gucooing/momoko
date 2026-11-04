<template>
  <div class="file-page">
    <FileBrowser ref="browserRef" :scope="scope" :initial-path="initialPath" :initial-source-id="initialSourceId">
      <template #nav><FileSubNav /></template>
    </FileBrowser>
  </div>
</template>

<script setup lang="ts">
defineOptions({ name: 'SystemFileView' })
import { useRoute } from 'vue-router'
import FileBrowser from '@/components/file/FileBrowser.vue'
import FileSubNav from '@/views/file/components/FileSubNav.vue'
import { PAGE_GROUPS } from '@/config/pageGroups'
import { useFullBleed } from '@/composables/useAppLayout'
import type { FileScope } from '@/components/file/types'

// 文件页的「文件管理」标签（/file?tab=index）：scope=system，可由 ?workdir= 指定初始目录、?source= 指定初始文件来源（文件源标签「浏览」）。
const route = useRoute()
const scope: FileScope = { kind: 'system' }
const initialPath = String(route.query.workdir || '')
const initialSourceId = String(route.query.source || '')

// 页面被缓存时 ?source= 变化不会重新挂载：直接切换来源。
const browserRef = ref<InstanceType<typeof FileBrowser> | null>(null)
watch(
  () => route.query.source,
  (source) => {
    const onThisTab = route.path === PAGE_GROUPS.file.path && (route.query.tab ?? 'index') === 'index'
    if (onThisTab && typeof source === 'string') browserRef.value?.switchSource(source)
  },
)

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
