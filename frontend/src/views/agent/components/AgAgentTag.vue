<!-- Agent 标记：Claude Code / Codex / Antigravity / 其它 ACP Agent。dot 为单字母的小方块（会话栏用）。 -->
<template>
  <span :class="[dot ? 'ag-agent-dot' : 'ag-agent', `ag-agent--${kind || 'custom'}`]" :aria-label="name" :title="dot ? name : undefined">
    {{ dot ? letter : name }}
  </span>
</template>

<script setup lang="ts">
const props = defineProps<{ kind?: string; name?: string; dot?: boolean }>()
const letter = computed(() => {
  if (props.kind === 'claude') return 'C'
  if (props.kind === 'codex') return 'X'
  if (props.kind === 'antigravity') return 'A'
  return (props.name || '?').slice(0, 1).toUpperCase()
})
</script>
