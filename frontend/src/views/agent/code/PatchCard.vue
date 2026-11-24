<!-- ACP v2 的一个文件级改动：Agent 给的是操作（新建、删除、修改、移动、复制）和这个文件的 git patch。
     平时是一行「已编辑 文件名 +增 −删」，点开按行显示整份 patch（卡片里上下滚着看）。二进制文件之类没有 patch，只有一行说明。 -->
<template>
  <div class="agc-edit">
    <button type="button" class="agc-edit__row" :aria-expanded="open" :title="fullName" @click="open = !open">
      <span class="agc-edit__verb">{{ t(`agent.code.edit.${OPS.includes(op) ? op : 'modify'}`) }}</span>
      <span class="ag-mono ag-ellipsis agc-edit__name">{{ baseName(path) }}</span>
      <template v-if="diff.patch">
        <span v-if="counts.added || !counts.deleted" class="agc-plus ag-num">+{{ counts.added }}</span>
        <span v-if="counts.deleted" class="agc-minus ag-num">−{{ counts.deleted }}</span>
      </template>
      <UIcon :name="open ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" class="agc-edit__chev" />
    </button>
    <div v-if="open" class="agc-diffcard">
      <div class="agc-diffcard__head">
        <span class="ag-mono ag-ellipsis agc-tool__arg">{{ fullName }}</span>
        <button type="button" class="ag-btn ag-btn--sm agc-diffcard__btn" @click="emit('showDiff', path)">
          {{ t('agent.code.fullDiff') }}
        </button>
      </div>
      <!-- 整个文件的改动都在这里，超过十几行就在卡片里上下滚着看；特别长的只放前面一段 -->
      <PatchView class="agc-diffcard__patch" :patch="body" :max-lines="MAX_LINES" flat :empty="t('agent.code.diff.noPatch')" />
      <div v-if="trimmed" class="agc-patch__empty">{{ t('agent.code.diffTrimmed', { n: MAX_LINES }) }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { patchBody, patchDelta } from '@/views/agent/utils/diff'
import PatchView from './PatchView.vue'
import type { AgentDiff } from './types'

const props = defineProps<{ diff: AgentDiff; path: string; oldPath?: string }>()
const emit = defineEmits<{ showDiff: [path: string] }>()
const { t } = useI18n()

const OPS = ['add', 'delete', 'modify', 'move', 'copy']
const op = computed(() => props.diff.op || 'modify')
const open = ref(false)

const baseName = (p: string) => p.slice(Math.max(p.lastIndexOf('/'), p.lastIndexOf('\\')) + 1) || p
const fullName = computed(() => (props.oldPath ? `${props.oldPath} → ${props.path}` : props.path))
const body = computed(() => patchBody(props.diff.patch))
const counts = computed(() => patchDelta(props.diff.patch))
/** 卡片里最多放多少行（再多页面会卡），其余的去「完整改动」里看 */
const MAX_LINES = 3000
const trimmed = computed(() => {
  // 和 PatchView 一样数：每个段头变成一行「⋯」（第一个不算），每行内容一行，「\ No newline」不算
  let n = -1
  for (const l of body.value.split('\n')) if (l && !l.startsWith('\\')) n += 1
  return n > MAX_LINES
})
</script>
