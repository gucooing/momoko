<!-- Git 面板共用的弹出菜单：右键时在鼠标处，点按钮时挂在按钮下面；放不下时往里挪。上下键挪、回车选、Esc 关。 -->
<template>
  <Teleport to="body">
    <div
      v-if="menu"
      ref="el"
      class="ag ag-menu agc-ctx agg-menu"
      role="menu"
      :style="{ left: `${pos.x}px`, top: `${pos.y}px` }"
      @contextmenu.prevent
      @keydown="onKey"
    >
      <template v-for="(it, i) in items" :key="i">
        <div v-if="it.sep" class="ag-mi-sep" />
        <div v-else-if="it.title" class="ag-mi-label">{{ it.title }}</div>
        <button v-else type="button" class="ag-mi" :class="{ 'is-bad': it.danger }" role="menuitem" :disabled="it.disabled" @click="pick(it)">
          <UIcon v-if="it.icon" :name="it.icon" />
          <span class="agc-flex">{{ it.label }}</span>
          <span v-if="it.hint" class="agc-ctx__k">{{ it.hint }}</span>
        </button>
      </template>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { onClickOutside, useEventListener } from '@vueuse/core'
import { useGit, type GitMenuItem } from './useGit'

const git = useGit()
const menu = computed(() => git.menu.value)
/** 去掉开头、结尾和连着的分隔线 */
const items = computed(() => {
  const out: GitMenuItem[] = []
  for (const it of menu.value?.items ?? []) {
    if (it.sep && (!out.length || out[out.length - 1]!.sep)) continue
    out.push(it)
  }
  while (out.length && out[out.length - 1]!.sep) out.pop()
  return out
})
const el = ref<HTMLElement | null>(null)
const pos = ref({ x: 0, y: 0 })

watch(
  menu,
  async (m) => {
    if (!m) return
    pos.value = { x: m.x, y: m.y }
    await nextTick()
    const box = el.value
    if (!box) return
    const { width, height } = box.getBoundingClientRect()
    pos.value = {
      x: Math.max(8, Math.min(m.x, window.innerWidth - width - 8)),
      y: m.y + height > window.innerHeight - 8 ? Math.max(8, m.y - height - 4) : m.y,
    }
    box.querySelector<HTMLElement>('.ag-mi:not(:disabled)')?.focus()
  },
  { flush: 'post' },
)
const close = () => {
  git.menu.value = null
}
const pick = (it: GitMenuItem) => {
  close()
  void it.run?.()
}
const onKey = (e: KeyboardEvent) => {
  if (e.key === 'Escape') {
    e.stopPropagation()
    close()
    return
  }
  if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return
  e.preventDefault()
  const list = [...(el.value?.querySelectorAll<HTMLElement>('.ag-mi:not(:disabled)') ?? [])]
  if (!list.length) return
  const at = list.indexOf(document.activeElement as HTMLElement)
  const next = at < 0 ? 0 : (at + (e.key === 'ArrowDown' ? 1 : -1) + list.length) % list.length
  list[next]?.focus()
}
// 点的是打开它的那个按钮：按钮的点击接着会再打开一次，记下来让它别开（再点一次就是关上）
onClickOutside(el, (e) => {
  git.menuClosedBy(e.target)
  close()
})
useEventListener(window, 'blur', close)
useEventListener(window, 'resize', close)
</script>
