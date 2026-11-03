<!-- 智商检测页内标签：检测站点 / 提示词（/iqcheck?tab=…），标签上带各自数量（由所在页传入当前页的数量，另一页的数量自行轻量拉取）。 -->
<template>
  <SubNavTabs group="iqcheck" :counts="{ site: props.siteCount ?? fetchedSites, prompt: props.promptCount ?? fetchedPrompts }" />
</template>

<script setup lang="ts">
import { listIQCheckPrompts, listIQCheckSites } from '@/api/iqcheck'

const props = defineProps<{ siteCount?: number | null; promptCount?: number | null }>()
const fetchedSites = ref<number | null>(null)
const fetchedPrompts = ref<number | null>(null)
onMounted(async () => {
  // 只为另一页补拉数量；没有菜单权限时不请求
  try {
    if (props.siteCount === undefined) {
      const { data } = await listIQCheckSites({ page: 1, pageSize: 1 })
      fetchedSites.value = Number(data?.stats?.total ?? data?.total ?? 0)
    }
    if (props.promptCount === undefined) {
      const { data } = await listIQCheckPrompts({ page: 1, pageSize: 1 })
      fetchedPrompts.value = Number(data?.allCount ?? data?.total ?? 0)
    }
  } catch {
    // 数量只是辅助信息，失败时不显示
  }
})
</script>
