<!-- Agent 管理：所有人的会话 / 工作区 / 模型供应商 / MCP / 技能 / 通知（要 agent:admin，默认只有超级管理员有），
     加上运行时和官方账户（还要 agent:manage）。会话到技能这几个标签和 Agent 设置用同一套列表，这里 provideAgentAll()
     让它们看所有人的、别人的只留清理操作，共享的在这里加和改；各标签见 config/pageGroups.ts。 -->
<template>
  <PageGroupHost group="agentAdmin" />
</template>

<script setup lang="ts">
import { provideAgentAll } from '@/views/agent/composables/useAgentAll'

defineOptions({ name: 'AgentAdminGroupView' })

provideAgentAll()
</script>
