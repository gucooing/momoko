<!-- 令牌化组合框（可输入 + 可搜索建议）：文本输入自由填写 + 从 options 过滤的 teleport 建议浮层。
     用于「镜像」等既要从已有项选、又允许自定义输入的场景（替代 el-input+dropdown / datalist 原生浮层）。
     v-model=string；浮层 z-index 2300 高于 FormDialog。 -->
<template>
  <div ref="rootRef" class="app-combo">
    <input
      ref="inputRef"
      :value="modelValue"
      class="app-input"
      :class="{ 'is-error': error }"
      :placeholder="placeholder"
      :disabled="disabled"
      @input="onInput"
      @focus="openPanel"
      @keydown="onKey"
    />
    <button v-if="options.length" type="button" class="app-combo__chev" :class="{ 'is-open': open }" :disabled="disabled" @click="toggle">
      <component :is="menuStore.iconComponents['HOutline:ChevronDownIcon']" />
    </button>

    <Teleport to="body">
      <div v-if="open && visible.length" ref="panelRef" class="app-combo__panel" :style="panelStyle" role="listbox">
        <button
          v-for="(opt, i) in visible"
          :key="opt"
          type="button"
          class="app-combo__opt"
          :class="{ 'is-hl': i === highlight, 'is-active': opt === modelValue }"
          role="option"
          @mousedown.prevent="choose(opt)"
          @mouseenter="highlight = i"
        >
          {{ opt }}
        </button>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    modelValue: string
    options?: string[]
    placeholder?: string
    disabled?: boolean
    error?: boolean
  }>(),
  { options: () => [], placeholder: '' },
)
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const menuStore = useMenuStore()

const rootRef = ref<HTMLElement>()
const inputRef = ref<HTMLInputElement>()
const panelRef = ref<HTMLElement>()
const open = ref(false)
const highlight = ref(-1)

// 已输入内容作为过滤词；与当前值完全相等时不过滤（展示全部，便于换选）
const visible = computed(() => {
  const q = props.modelValue.trim().toLowerCase()
  const list = props.options
  if (!q) return list
  const exact = list.some((o) => o.toLowerCase() === q)
  if (exact) return list
  return list.filter((o) => o.toLowerCase().includes(q))
})

const panelStyle = ref<Record<string, string>>({})
const place = () => {
  const el = rootRef.value
  if (!el) return
  const r = el.getBoundingClientRect()
  const vh = window.innerHeight
  const below = vh - r.bottom
  const openUp = below < 240 && r.top > below
  const style: Record<string, string> = {
    position: 'fixed', left: `${r.left}px`, zIndex: '2300', width: `${r.width}px`, maxHeight: '240px',
  }
  if (openUp) style.bottom = `${vh - r.top + 4}px`
  else style.top = `${r.bottom + 4}px`
  panelStyle.value = style
}

const onOutside = (e: MouseEvent) => {
  const target = e.target as Node
  if (rootRef.value?.contains(target) || panelRef.value?.contains(target)) return
  close()
}
const onScrollResize = () => close()

const openPanel = () => {
  if (props.disabled || open.value) return
  place()
  open.value = true
  highlight.value = -1
  nextTick(() => {
    document.addEventListener('mousedown', onOutside, true)
    window.addEventListener('scroll', onScrollResize, true)
    window.addEventListener('resize', onScrollResize)
  })
}
const close = () => {
  if (!open.value) return
  open.value = false
  document.removeEventListener('mousedown', onOutside, true)
  window.removeEventListener('scroll', onScrollResize, true)
  window.removeEventListener('resize', onScrollResize)
}
const toggle = () => {
  if (open.value) close()
  else { inputRef.value?.focus(); openPanel() }
}

const onInput = (e: Event) => {
  emit('update:modelValue', (e.target as HTMLInputElement).value)
  if (!open.value) openPanel()
  highlight.value = -1
}
const choose = (opt: string) => {
  emit('update:modelValue', opt)
  close()
}
const onKey = (e: KeyboardEvent) => {
  if (e.key === 'Escape') { if (open.value) { e.stopPropagation(); close() } return }
  if (!open.value && (e.key === 'ArrowDown' || e.key === 'ArrowUp')) { openPanel(); return }
  if (e.key === 'ArrowDown') { e.preventDefault(); highlight.value = Math.min(highlight.value + 1, visible.value.length - 1) }
  else if (e.key === 'ArrowUp') { e.preventDefault(); highlight.value = Math.max(highlight.value - 1, 0) }
  else if (e.key === 'Enter') {
    const pick = highlight.value >= 0 ? visible.value[highlight.value] : undefined
    if (open.value && pick) { e.preventDefault(); e.stopPropagation(); choose(pick) }
  }
}

onBeforeUnmount(close)
</script>

<style scoped lang="scss">
.app-combo { position: relative; width: 100%; }
.app-combo .app-input { padding-right: 30px; }
.app-combo__chev {
  position: absolute;
  top: 0;
  right: 0;
  height: 100%;
  width: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  color: var(--el-text-color-placeholder);
  cursor: pointer;
}
.app-combo__chev :deep(svg) { width: 15px; height: 15px; transition: transform 0.18s; }
.app-combo__chev.is-open :deep(svg) { transform: rotate(180deg); }
</style>

<style lang="scss">
.app-combo__panel {
  display: flex;
  flex-direction: column;
  gap: 1px;
  padding: 4px;
  overflow-y: auto;
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius);
  box-shadow: var(--app-shadow-lg);
}
.app-combo__opt {
  padding: 6px 8px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-xs);
  color: var(--el-text-color-regular);
  font-size: 0.8125rem;
  font-family: inherit;
  text-align: left;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  cursor: pointer;
}
.app-combo__opt.is-hl { background: var(--el-fill-color-light); color: var(--el-text-color-primary); }
.app-combo__opt.is-active { color: var(--el-color-primary); font-weight: 600; }
</style>
