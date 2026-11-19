<!-- 统一 diff 的逐行显示：按 @@ 段头算出行号，新增绿底、删除红底。补丁本身来自 git 或 Agent，这里只负责排版。 -->
<template>
  <div class="agc-patch ag-mono" :class="{ 'agc-patch--flat': flat }">
    <div v-for="(l, i) in shown" :key="i" class="agc-dl" :class="l.cls">
      <span class="agc-dl__n">{{ l.n }}</span>
      <span class="agc-dl__s">{{ l.s }}</span>
      <span class="agc-dl__t">{{ l.t }}</span>
    </div>
    <div v-if="!lines.length" class="agc-patch__empty">{{ empty }}</div>
  </div>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{ patch?: string; maxLines?: number; flat?: boolean; empty?: string }>(), {
  patch: '',
  maxLines: 0,
  flat: false,
  empty: '',
})

interface Line {
  n: string
  s: string
  t: string
  cls: string
}

const HUNK = /^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/

const lines = computed<Line[]>(() => {
  const out: Line[] = []
  let oldNo = 0
  let newNo = 0
  for (const raw of props.patch.replace(/\n+$/, '').split('\n')) {
    if (!raw && !out.length) continue
    const hunk = HUNK.exec(raw)
    if (hunk) {
      oldNo = Number(hunk[1])
      newNo = Number(hunk[2])
      if (out.length) out.push({ n: '', s: '', t: '⋯', cls: 'is-gap' })
      continue
    }
    if (raw.startsWith('\\')) continue
    const sign = raw.charAt(0)
    const text = raw.slice(1)
    if (sign === '+') {
      out.push({ n: String(newNo || ''), s: '+', t: text, cls: 'is-add' })
      newNo += 1
    } else if (sign === '-') {
      out.push({ n: String(oldNo || ''), s: '−', t: text, cls: 'is-del' })
      oldNo += 1
    } else {
      out.push({ n: String(newNo || ''), s: '', t: sign === ' ' ? text : raw, cls: '' })
      oldNo += 1
      newNo += 1
    }
  }
  return out
})

const shown = computed(() => (props.maxLines > 0 ? lines.value.slice(0, props.maxLines) : lines.value))
</script>
