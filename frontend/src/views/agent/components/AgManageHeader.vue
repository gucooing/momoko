<!-- Agent 设置和 Agent 管理各标签页共用的页头：项目现有的 PageHeader + 页内标签（带数量）。
     副标题只放数量（默认插槽），右上的按钮放 #actions。在 Agent 管理页里（useAgentAll）换成那一页的标题和标签。 -->
<template>
  <PageHeader :title="all ? t('agent.admin.title') : t('agent.manage.title')">
    <template #description><slot /></template>
    <template v-if="$slots.actions" #actions><slot name="actions" /></template>
  </PageHeader>
  <SubNavTabs :group="all ? 'agentAdmin' : 'agent'" :counts="counts" />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useAgentAll } from '@/views/agent/composables/useAgentAll'
import { useAgentOverview } from '@/views/agent/composables/useAgentOverview'

const { t } = useI18n()
const all = useAgentAll()
const { overview } = useAgentOverview()
const counts = computed((): Record<string, number> => {
  const o = overview.value
  if (!o) return {}
  return {
    sessions: o.sessions,
    workspaces: o.workspaces,
    providers: o.providers,
    mcp: o.mcpServers,
    skills: o.skills,
    runtimes: o.runtimes,
    notify: o.notifyChannels,
  }
})
</script>

<!-- 紧凑样式只在这里引入一次（每个管理标签页都会渲染这个页头），各标签页不用再引。 -->
<style lang="scss">
@use '@/views/agent/styles/agent.scss';
</style>
