<!-- Git 面板里的 diff：按 @@ 段显示，左边两列是旧、新行号。actions 给了时每段段头有「暂存 / 取消暂存 / 撤销这一段」，
     点 + / − 行可以选中（Shift 连选），底下出一条只对选中的行操作的按钮。conflict 时内容是冲突的文件本身，
     冲突标记和两边的内容分别标色。行号是 patch 按换行拆开后的序号，原样交给后端拼部分补丁。 -->
<template>
  <div class="agg-patch">
    <div class="agg-patch__body ag-mono" @mousedown="onMouseDown">
      <template v-for="h in hunks" :key="h.at">
        <div v-if="!conflict" class="agg-hunk">
          <span class="agg-hunk__h ag-ellipsis">{{ h.header }}</span>
          <template v-if="h.changes.length && pickable">
            <button
              v-for="a in actions"
              :key="a"
              type="button"
              class="agg-hunk__btn"
              :class="{ 'is-bad': a === 'discard' }"
              :disabled="disabled"
              :title="t(HUNK_LABEL[a])"
              @click="emit('apply', a, h.changes)"
            >
              <UIcon :name="ICON[a]" />
            </button>
          </template>
        </div>
        <div
          v-for="l in h.lines"
          :key="l.i"
          class="agc-dl agg-dl"
          :class="[l.cls, { 'is-pick': pickable && !!l.sign, 'is-sel': sel.has(l.i) }]"
          @click="pick(l, $event)"
        >
          <span class="agc-dl__n agg-dl__n">{{ l.old }}</span>
          <span class="agc-dl__n agg-dl__n">{{ l.new }}</span>
          <span class="agc-dl__s">{{ l.s }}</span>
          <span class="agc-dl__t">{{ l.t }}</span>
        </div>
      </template>
      <div v-if="!hunks.length" class="agc-patch__empty">{{ empty }}</div>
    </div>
    <div v-if="pickable && sel.size" class="agg-pick">
      <span class="ag-num agc-flex">{{ t('agent.git.diff.selected', { n: sel.size }) }}</span>
      <button
        v-for="a in actions"
        :key="a"
        type="button"
        class="ag-btn ag-btn--sm"
        :class="a === 'discard' ? 'ag-btn--stop' : 'ag-btn--p'"
        :disabled="disabled"
        @click="applySel(a)"
      >
        <UIcon :name="ICON[a]" />{{ t(LINES_LABEL[a]) }}
      </button>
      <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.git.diff.clear')" :title="t('agent.git.diff.clear')" @click="sel.clear()">
        <UIcon name="i-lucide-x" />
      </button>
    </div>
    <div v-else-if="pickable && hunks.length" class="agg-patch__tip">{{ t('agent.git.diff.selectTip') }}</div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { AgentGitLineAction } from '@/api/agent'

const props = withDefaults(
  defineProps<{
    patch: string
    /** 能对段、选中的行做的事；空的是只读 */
    actions?: AgentGitLineAction[]
    /** 能不能按行选（新文件、删掉的、改名的只能整个暂存） */
    lines?: boolean
    conflict?: boolean
    disabled?: boolean
    empty?: string
  }>(),
  { actions: () => [], lines: true, conflict: false, disabled: false, empty: '' },
)
const emit = defineEmits<{ apply: [action: AgentGitLineAction, lines: number[]] }>()
const { t } = useI18n()

const ICON: Record<AgentGitLineAction, string> = { stage: 'i-lucide-plus', unstage: 'i-lucide-minus', discard: 'i-lucide-undo-2' }
const HUNK_LABEL: Record<AgentGitLineAction, string> = {
  stage: 'agent.git.diff.hunkStage',
  unstage: 'agent.git.diff.hunkUnstage',
  discard: 'agent.git.diff.hunkDiscard',
}
const LINES_LABEL: Record<AgentGitLineAction, string> = {
  stage: 'agent.git.diff.linesStage',
  unstage: 'agent.git.diff.linesUnstage',
  discard: 'agent.git.diff.linesDiscard',
}

interface Line {
  /** patch 按换行拆开后的序号 */
  i: number
  old: string
  new: string
  s: string
  t: string
  sign: '' | '+' | '-'
  cls: string
}
interface Hunk {
  at: number
  header: string
  lines: Line[]
  /** 这一段里的 + / - 行 */
  changes: number[]
}

const HUNK = /^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@(.*)$/

const hunks = computed<Hunk[]>(() => {
  const out: Hunk[] = []
  let cur: Hunk | null = null
  let oldNo = 0
  let newNo = 0
  // 冲突的文件：在哪一边（<<<<<<< 到 ||||||| / ======= 是当前的，======= 到 >>>>>>> 是传入的）
  let side = ''
  const rows = props.patch.split('\n')
  rows.forEach((raw, i) => {
    const m = HUNK.exec(raw)
    if (m) {
      oldNo = Number(m[1])
      newNo = Number(m[2])
      cur = { at: i, header: raw.slice(0, raw.indexOf(' @@') + 3) + (m[3] ? ` ${m[3].trim()}` : ''), lines: [], changes: [] }
      out.push(cur)
      return
    }
    if (!cur || raw === '' || raw.startsWith('\\')) return
    const sign = raw.charAt(0)
    const text = raw.slice(1).replace(/\r$/, '')
    if (sign === '+') {
      cur.lines.push({ i, old: '', new: String(newNo++), s: '+', t: text, sign: '+', cls: 'is-add' })
      cur.changes.push(i)
    } else if (sign === '-') {
      cur.lines.push({ i, old: String(oldNo++), new: '', s: '−', t: text, sign: '-', cls: 'is-del' })
      cur.changes.push(i)
    } else {
      let cls = ''
      if (props.conflict) {
        if (text.startsWith('<<<<<<<')) {
          side = 'ours'
          cls = 'is-mk'
        } else if (text.startsWith('|||||||') && side) {
          side = 'base'
          cls = 'is-mk'
        } else if (text.startsWith('=======') && side) {
          side = 'theirs'
          cls = 'is-mk'
        } else if (text.startsWith('>>>>>>>') && side) {
          side = ''
          cls = 'is-mk'
        } else if (side) {
          cls = `is-${side}`
        }
      }
      cur.lines.push({ i, old: props.conflict ? '' : String(oldNo++), new: String(newNo++), s: '', t: text, sign: '', cls })
    }
  })
  return out
})

// —— 按行选 ——
const sel = reactive(new Set<number>())
const pickable = computed(() => props.lines && props.actions.length > 0 && !props.conflict)
let anchor = -1
const allChanges = computed(() => hunks.value.flatMap((h) => h.changes))
/** Shift 点选时别把文字也选上 */
const onMouseDown = (e: MouseEvent) => {
  if (e.shiftKey && pickable.value) e.preventDefault()
}
const pick = (l: Line, e: MouseEvent) => {
  if (!pickable.value || !l.sign) return
  // 在选文字（拖出了一段）时不当点选
  if (!e.shiftKey && window.getSelection()?.toString()) return
  if (e.shiftKey && anchor >= 0) {
    const [a, b] = anchor < l.i ? [anchor, l.i] : [l.i, anchor]
    for (const i of allChanges.value) if (i >= a && i <= b) sel.add(i)
    return
  }
  anchor = l.i
  if (sel.has(l.i)) sel.delete(l.i)
  else sel.add(l.i)
}
const applySel = (a: AgentGitLineAction) => {
  emit('apply', a, [...sel].sort((x, y) => x - y))
}
// 换了一份 diff（文件、来源变了，或者操作完重读了）选中的就不算数了
watch(
  () => props.patch,
  () => {
    sel.clear()
    anchor = -1
  },
)
</script>
