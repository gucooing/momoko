<!-- 页内标签：一个页面里的子视图（如 Docker 的 容器 / 镜像 / 网络 / 连接配置），地址形如 /docker?tab=image。
     分组定义见 config/pageGroups.ts；只显示当前用户在当前节点有权限的标签，只剩一个时整条隐藏。
     切换只改 ?tab=，页面不重新挂载（由 PageGroupHost 用 keep-alive 保留各标签状态）。 -->
<template>
  <nav v-if="tabs.length > 1" class="subnav" :aria-label="t(def.labelKey)" role="tablist">
    <RouterLink
      v-for="tab in tabs"
      :key="tab.key"
      :to="pageTabTo(group, tab.key)"
      replace
      role="tab"
      class="subnav__tab"
      :class="{ 'is-on': current?.key === tab.key }"
      :aria-selected="current?.key === tab.key"
    >
      {{ t(tab.labelKey) }}
      <span v-if="counts?.[tab.key] != null" class="subnav__count m-num">{{ counts[tab.key] }}</span>
    </RouterLink>
  </nav>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { pageTabTo, usePageGroupTabs, type PageGroupKey } from '@/config/pageGroups'

const props = defineProps<{
  group: PageGroupKey
  /** 各标签上显示的数量，键为标签 key */
  counts?: Partial<Record<string, number | null>>
}>()

const { t } = useI18n()
const { def, tabs, current } = usePageGroupTabs(props.group)
</script>

<style scoped lang="scss">
.subnav {
  display: flex;
  gap: 26px;
  overflow-x: auto;
  border-bottom: 1px solid var(--el-border-color-light);
  scrollbar-width: none;
}
.subnav::-webkit-scrollbar {
  display: none;
}
.subnav__tab {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: var(--cmp-subnav-height);
  padding: 0 2px;
  border-bottom: 3px solid transparent;
  margin-bottom: -1px;
  color: var(--el-text-color-secondary);
  font-size: var(--cmp-subnav-font-size);
  font-weight: 700;
  text-decoration: none;
  white-space: nowrap;
  transition: color 0.15s, border-color 0.15s;
}
.subnav__tab:hover {
  color: var(--el-text-color-primary);
}
.subnav__tab.is-on {
  color: var(--el-text-color-primary);
  border-bottom-color: var(--m-p);
}
.subnav__count {
  padding: 1px 8px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
}
</style>
