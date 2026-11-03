<!-- 标签页分组的页面壳：一个菜单路由（如 /docker）里按 ?tab= 显示对应子视图。
     子视图用 keep-alive 缓存：切换标签不重新挂载、不重新拉数据，保留筛选、滚动与未保存的表单；
     子视图首次打开时才加载（按需）。没有任何可见标签时显示无权限提示。 -->
<template>
  <KeepAlive>
    <component :is="current.component" v-if="current" :key="current.key" />
  </KeepAlive>
  <EmptyState
    v-if="!current"
    icon="HOutline:LockClosedIcon"
    :title="t('pageGroup.noAccessTitle')"
    :description="t('pageGroup.noAccessDesc')"
  />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { usePageGroupTabs, type PageGroupKey } from '@/config/pageGroups'

const props = defineProps<{ group: PageGroupKey }>()
const { t } = useI18n()
const { current } = usePageGroupTabs(props.group)
</script>
