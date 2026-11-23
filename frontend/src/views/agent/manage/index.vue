<!-- Agent 设置：标签页分组页面（会话 / 工作区 / 模型供应商 / MCP / 技能 / 通知），各标签见 config/pageGroups.ts。 -->
<template>
  <PageGroupHost group="agent" />
</template>

<script setup lang="ts">
defineOptions({ name: 'AgentManageGroupView' })
</script>
