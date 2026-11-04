<!-- 节点与 API Key 页内标签：节点 / API Key（/node?tab=…），标签上带数量。
     当前标签把自己的数量传进来（nodeCount / keyCount）；另一个标签的数量在这里按需取一次。 -->
<template>
  <SubNavTabs group="node" :counts="{ list: nodes, key: keys }" />
</template>

<script setup lang="ts">
import { listAPIKeys } from '@/api/node'

const props = defineProps<{ nodeCount?: number | null; keyCount?: number | null }>()
const nodeStore = useNodeStore()
const menuStore = useMenuStore()
const canNodes = computed(() => menuStore.hasButtonPermission('node:view'))
const canKeys = computed(() => menuStore.hasButtonPermission('api_key:view'))
const fetchedKeys = ref<number | null>(null)
// 节点数含本机卡片
const nodes = computed(() =>
  props.nodeCount !== undefined ? props.nodeCount : canNodes.value ? nodeStore.nodes.length + 1 : null,
)
const keys = computed(() => (props.keyCount !== undefined ? props.keyCount : fetchedKeys.value))

onMounted(async () => {
  if (props.nodeCount === undefined && canNodes.value && !nodeStore.nodes.length) {
    void nodeStore.loadNodes().catch(() => undefined)
  }
  if (props.keyCount === undefined && canKeys.value) {
    try {
      const { data } = await listAPIKeys({
        page: 1,
        pageSize: 1,
        allUsers: menuStore.hasButtonPermission('api_key:manage'),
      })
      fetchedKeys.value = Number(data?.total ?? 0)
    } catch {
      fetchedKeys.value = null
    }
  }
})
</script>
