<!-- 令牌化下拉选择（替代原生 <select>，规避 OS 直角浮层）：触发器复用输入控件外观，
     浮层 teleport 到 body、圆角+hairline+阴影，z-index 高于 FormDialog(2200)，可用于弹窗内。
     支持键盘（↑↓/Enter/Esc）、外部点击/滚动/缩放关闭、向上翻转。
     searchable=面板内本地过滤（长列表/用户选择）；fit=按内容宽（分页每页用）。 -->
<template>
  <div ref="rootRef" class="app-sel" :class="{ 'is-fit': fit }">
    <button
      type="button"
      class="app-sel__trigger"
      :class="{ 'is-open': open, 'is-error': error, 'is-placeholder': !selected }"
      :disabled="disabled"
      @click="toggle"
      @keydown="onTriggerKey"
    >
      <span class="app-sel__value">{{ selected ? selected.label : placeholder }}</span>
      <component
        :is="menuStore.iconComponents['HOutline:ChevronDownIcon']"
        class="app-sel__chev"
        :class="{ 'is-open': open }"
      />
    </button>

    <Teleport to="body">
      <div v-if="open" ref="panelRef" class="app-sel__panel" :style="panelStyle" role="listbox">
        <div v-if="searchable" class="app-sel__search">
          <input
            ref="searchRef"
            v-model="query"
            class="app-sel__search-input"
            :placeholder="searchPlaceholder"
            @keydown="onSearchKey"
          />
        </div>
        <div class="app-sel__opts">
          <button
            v-for="(opt, i) in visible"
            :key="String(opt.value)"
            type="button"
            class="app-sel__opt"
            :class="{
              'is-active': opt.value === modelValue,
              'is-hl': i === highlight,
              'is-disabled': opt.disabled,
            }"
            :disabled="opt.disabled"
            role="option"
            :aria-selected="opt.value === modelValue"
            @click="choose(opt)"
            @mouseenter="highlight = i"
          >
            <span class="app-sel__opt-label">{{ opt.label }}</span>
            <component
              :is="menuStore.iconComponents['HOutline:CheckIcon']"
              v-if="opt.value === modelValue"
              class="app-sel__opt-check"
            />
          </button>
          <div v-if="!visible.length" class="app-sel__empty">{{ noMatchText }}</div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts" generic="T extends string | number | boolean | undefined">
interface AppSelectOption {
  label: string
  value: T
  disabled?: boolean
}

const props = withDefaults(
  defineProps<{
    modelValue: T
    options: AppSelectOption[]
    placeholder?: string
    disabled?: boolean
    error?: boolean
    /** 按内容宽（不撑满容器），用于分页每页等紧凑场景。 */
    fit?: boolean
    /** 面板内显示搜索框，按 label 本地过滤（长列表）。 */
    searchable?: boolean
    searchPlaceholder?: string
    noMatchText?: string
  }>(),
  { placeholder: '', searchPlaceholder: '', noMatchText: '—' },
)
const emit = defineEmits<{ 'update:modelValue': [value: T] }>()

const menuStore = useMenuStore()

const rootRef = ref<HTMLElement>()
const panelRef = ref<HTMLElement>()
const searchRef = ref<HTMLInputElement>()
const open = ref(false)
const highlight = ref(-1)
const query = ref('')

const selected = computed(() => props.options.find((o) => o.value === props.modelValue))

// 可见项：searchable 且有输入时按 label 本地过滤，否则全量
const visible = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!props.searchable || !q) return props.options
  return props.options.filter((o) => o.label.toLowerCase().includes(q))
})

const panelStyle = ref<Record<string, string>>({})

const place = () => {
  const el = rootRef.value
  if (!el) return
  const r = el.getBoundingClientRect()
  const vh = window.innerHeight
  // 面板最高 260；预留上下边距，避免贴边裁切
  const maxPanel = 260
  const below = vh - r.bottom - 8
  const above = r.top - 8
  const openUp = below < 160 && above > below
  const available = Math.max(120, openUp ? above : below)
  const panelMax = Math.min(maxPanel, available)
  // 搜索行约 36px；选项区吃掉剩余高度，否则 maxHeight 只在 panel 上、opts 无法滚动
  const searchH = props.searchable ? 36 : 0
  const optsMax = Math.max(80, panelMax - searchH - 8)
  const style: Record<string, string> = {
    position: 'fixed',
    left: `${r.left}px`,
    zIndex: '2300',
    minWidth: `${r.width}px`,
    width: props.fit ? 'auto' : `${r.width}px`,
    maxHeight: `${panelMax}px`,
    // 传给样式：选项区上限
    ['--app-sel-opts-max' as string]: `${optsMax}px`,
  }
  if (openUp) style.bottom = `${vh - r.top + 4}px`
  else style.top = `${r.bottom + 4}px`
  panelStyle.value = style
}

const onOutside = (e: MouseEvent) => {
  const t = e.target as Node
  if (rootRef.value?.contains(t) || panelRef.value?.contains(t)) return
  close()
}
// 面板内滚轮/滚动不关；外层容器滚动或窗口缩放时关闭。
// 用 capture scroll 时，panel 内 .app-sel__opts 的 scroll 也会冒泡到 window，必须过滤。
const onScrollResize = (e: Event) => {
  if (e.type === 'resize') {
    close()
    return
  }
  const t = e.target
  if (t instanceof Node && panelRef.value?.contains(t)) return
  close()
}

const openPanel = () => {
  if (props.disabled) return
  place()
  query.value = ''
  open.value = true
  highlight.value = props.options.findIndex((o) => o.value === props.modelValue)
  nextTick(() => {
    place() // 打开后再算一次（search 行渲染后高度更准）
    document.addEventListener('mousedown', onOutside, true)
    // 不在 capture 上拦 wheel；只监听外层 scroll/resize
    window.addEventListener('scroll', onScrollResize, true)
    window.addEventListener('resize', onScrollResize)
    if (props.searchable) searchRef.value?.focus()
  })
}
const close = () => {
  if (!open.value) return
  open.value = false
  document.removeEventListener('mousedown', onOutside, true)
  window.removeEventListener('scroll', onScrollResize, true)
  window.removeEventListener('resize', onScrollResize)
}
const toggle = () => (open.value ? close() : openPanel())

const choose = (opt: AppSelectOption) => {
  if (opt.disabled) return
  emit('update:modelValue', opt.value as T)
  close()
}

const moveHighlight = (dir: 1 | -1) => {
  const n = visible.value.length
  if (!n) return
  let i = highlight.value
  for (let step = 0; step < n; step++) {
    i = (i + dir + n) % n
    if (!visible.value[i]?.disabled) break
  }
  highlight.value = i
}

// 输入变化时高亮重置到首个匹配项
watch(query, () => {
  highlight.value = visible.value.length ? 0 : -1
})

const onTriggerKey = (e: KeyboardEvent) => {
  if (props.disabled) return
  if (e.key === 'Escape') {
    // 面板打开时吞掉 Esc，避免冒泡连带关闭外层弹窗（FormDialog）
    if (open.value) {
      e.stopPropagation()
      close()
    }
    return
  }
  if (!open.value) {
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp' || e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      openPanel()
    }
    return
  }
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    moveHighlight(1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    moveHighlight(-1)
  } else if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    e.stopPropagation()
    const opt = visible.value[highlight.value]
    if (opt) choose(opt)
  }
}

// searchable 面板内输入框的键盘（焦点在输入框时）
const onSearchKey = (e: KeyboardEvent) => {
  if (e.key === 'Escape') {
    e.stopPropagation()
    close()
  } else if (e.key === 'ArrowDown') {
    e.preventDefault()
    moveHighlight(1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    moveHighlight(-1)
  } else if (e.key === 'Enter') {
    e.preventDefault()
    e.stopPropagation()
    const opt = visible.value[highlight.value]
    if (opt) choose(opt)
  }
}

onBeforeUnmount(close)
</script>

<style scoped lang="scss">
.app-sel {
  position: relative;
  width: 100%;
}
.app-sel.is-fit {
  width: auto;
}

/* 触发器：与 .app-input/.app-select 视觉一致（32 高、hairline、圆角、聚焦环） */
.app-sel__trigger {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  height: 32px;
  padding: 0 8px 0 10px;
  font-size: 0.8125rem;
  font-family: inherit;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: var(--app-radius-sm);
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s;
  outline: none;
}
.app-sel.is-fit .app-sel__trigger {
  width: auto;
  min-width: 88px;
}
.app-sel__trigger:hover {
  border-color: var(--el-text-color-placeholder);
}
.app-sel__trigger.is-open,
.app-sel__trigger:focus-visible {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--el-color-primary) 15%, transparent);
}
.app-sel__trigger.is-error {
  border-color: var(--el-color-danger, #ef4444);
}
.app-sel__trigger:disabled {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-placeholder);
  cursor: not-allowed;
}
.app-sel__value {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: left;
}
.app-sel__trigger.is-placeholder .app-sel__value {
  color: var(--el-text-color-placeholder);
}
.app-sel__chev {
  width: 15px;
  height: 15px;
  flex-shrink: 0;
  color: var(--el-text-color-placeholder);
  transition: transform 0.18s ease;
}
.app-sel__chev.is-open {
  transform: rotate(180deg);
}
</style>

<style lang="scss">
/* 浮层为 teleport 到 body 的全局节点，样式不加 scoped */
.app-sel__panel {
  display: flex;
  flex-direction: column;
  padding: 4px;
  overflow: hidden;
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius);
  box-shadow: var(--app-shadow-lg);
}
.app-sel__search {
  padding: 2px 2px 4px;
}
.app-sel__search-input {
  width: 100%;
  height: 28px;
  padding: 0 8px;
  font-size: 0.8125rem;
  font-family: inherit;
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-xs);
  outline: none;
}
.app-sel__search-input:focus {
  border-color: var(--el-color-primary);
}
.app-sel__opts {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-height: 0;
  /* 关键高度上限，否则 panel maxHeight 不会约束内部 flex 子项，导致列表无法滚动 */
  max-height: var(--app-sel-opts-max, 220px);
  overflow-y: auto;
  overscroll-behavior: contain;
}
.app-sel__opt {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 6px 8px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-xs);
  color: var(--el-text-color-regular);
  font-size: 0.8125rem;
  font-family: inherit;
  text-align: left;
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
}
.app-sel__opt-label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}
.app-sel__opt-check {
  width: 15px;
  height: 15px;
  flex-shrink: 0;
  color: var(--el-color-primary);
}
.app-sel__opt.is-hl:not(.is-disabled) {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.app-sel__opt.is-active {
  color: var(--el-color-primary);
  font-weight: 600;
}
.app-sel__opt.is-disabled {
  color: var(--el-text-color-placeholder);
  cursor: not-allowed;
}
.app-sel__empty {
  padding: 10px 8px;
  text-align: center;
  font-size: 0.8125rem;
  color: var(--el-text-color-placeholder);
}
</style>
