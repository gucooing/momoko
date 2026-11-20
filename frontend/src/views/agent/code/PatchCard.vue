<!-- ACP v2 的一个文件级改动：Agent 给的是操作（新建、删除、修改、移动、复制）和这个文件的 git patch，
     直接按行显示 patch（只显示前面一段），增删行数从 patch 里数。二进制文件之类没有 patch，只有一行说明。 -->
<template>
  <div class="agc-diffcard">
    <div class="agc-diffcard__head">
      <UIcon :name="OP_ICON[op] || 'i-lucide-pencil'" class="ag-muted" />
      <span class="agc-tool__verb">{{ t(`agent.code.tool.op.${OP_ICON[op] ? op : 'modify'}`) }}</span>
      <span class="ag-mono ag-ellipsis agc-tool__arg">
        <template v-if="oldPath">{{ oldPath }} → </template>{{ path }}
      </span>
      <template v-if="diff.patch">
        <span class="agc-plus ag-num">+{{ counts.added }}</span>
        <span class="agc-minus ag-num">−{{ counts.deleted }}</span>
      </template>
      <button type="button" class="ag-btn ag-btn--sm agc-diffcard__btn" @click="emit('showDiff', path)">
        {{ t('agent.code.fullDiff') }}
      </button>
    </div>
    <PatchView :patch="body" :max-lines="14" flat :empty="t('agent.code.diff.noPatch')" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import PatchView from './PatchView.vue'
import type { AgentDiff } from './types'

const props = defineProps<{ diff: AgentDiff; path: string; oldPath?: string }>()
const emit = defineEmits<{ showDiff: [path: string] }>()
const { t } = useI18n()

const OP_ICON: Record<string, string> = {
  add: 'i-lucide-file-plus',
  delete: 'i-lucide-file-minus',
  modify: 'i-lucide-pencil',
  move: 'i-lucide-file-symlink',
  copy: 'i-lucide-copy',
}
const op = computed(() => props.diff.op || 'modify')

/** patch 从第一个 @@ 段开始的部分：前面的 diff --git、index、--- / +++ 是文件头，不按行显示 */
const body = computed(() => {
  const patch = props.diff.patch || ''
  if (patch.startsWith('@@')) return patch
  const at = patch.indexOf('\n@@')
  return at >= 0 ? patch.slice(at + 1) : ''
})
/** 增删行数 */
const counts = computed(() => {
  let added = 0
  let deleted = 0
  for (const line of body.value.split('\n')) {
    if (line.startsWith('+')) added++
    else if (line.startsWith('-')) deleted++
  }
  return { added, deleted }
})
</script>
