<!-- 全员 Agent：所有人的会话 / 工作区 / MCP / 技能（要 agent:admin，默认只有超级管理员有）。
     各标签和 Agent 管理用同一套列表，这里 provideAgentAll() 让它们看所有人的、只留清理操作，各标签见 config/pageGroups.ts。 -->
<template>
  <PageGroupHost group="agentAdmin" />
</template>

<script setup lang="ts">
import { provideAgentAll } from '@/views/agent/composables/useAgentAll'

defineOptions({ name: 'AgentAdminGroupView' })

provideAgentAll()
</script>
