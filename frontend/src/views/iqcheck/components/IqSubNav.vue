<!-- 智商检测页内标签：检测站点 / 提示词（/iqcheck?tab=…），标签上带各自数量（由所在页传入当前页的数量，另一页的数量自行轻量拉取）。
     桌面用下划线子导航（SubNavTabs）；手机按原型用整行分段切换。 -->
<template>
  <nav
    v-if="menuStore.isMobile && tabs.length > 1"
    class="iq-seg"
    role="tablist"
    :aria-label="t(def.labelKey)"
  >
    <RouterLink
      v-for="tab in tabs"
      :key="tab.key"
      :to="pageTabTo('iqcheck', tab.key)"
      replace
      role="tab"
      class="iq-seg__tab"
      :class="{ 'is-on': current?.key === tab.key }"
      :aria-selected="current?.key === tab.key"
    >
      {{ t(tab.labelKey) }}
      <span v-if="counts[tab.key] != null" class="iq-seg__n m-num">{{ counts[tab.key] }}</span>
    </RouterLink>
  </nav>
  <SubNavTabs v-else group="iqcheck" :counts="counts" />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { listIQCheckPrompts, listIQCheckSites } from '@/api/iqcheck'
import { pageTabTo, usePageGroupTabs } from '@/config/pageGroups'

const props = defineProps<{ siteCount?: number | null; promptCount?: number | null }>()
const { t } = useI18n()
const menuStore = useMenuStore()
const { def, tabs, current } = usePageGroupTabs('iqcheck')

const fetchedSites = ref<number | null>(null)
const fetchedPrompts = ref<number | null>(null)
const counts = computed<Record<string, number | null>>(() => ({
  site: props.siteCount ?? fetchedSites.value,
  prompt: props.promptCount ?? fetchedPrompts.value,
}))

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

<style scoped lang="scss">
.iq-seg {
  display: flex;
  gap: 2px;
  padding: 4px;
  border-radius: var(--app-radius-pill);
  background: var(--el-fill-color-light);
}
.iq-seg__tab {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 0;
  min-height: 44px;
  padding: 0 12px;
  border-radius: var(--app-radius-pill);
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
  font-weight: 700;
  text-decoration: none;
  white-space: nowrap;
  transition:
    background 0.15s,
    color 0.15s;
}
.iq-seg__tab.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}
.iq-seg__n {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
</style>
