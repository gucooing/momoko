<!-- 顶栏「终端」悬浮窗：可拖动（按住标签栏空白处）、四边四角调大小，位置大小记在本机。
     一行标签栏：终端标签（状态点 + 名称 + 关闭）、「+」新建（本机终端 / 在某个目录打开 / SSH 主机），最右外观、最小化、关闭。
     挂在外壳上，切换页面不断开；最小化只是藏起来，连接保留；关闭等同刷新，全部断开。
     外观（不透明度、模糊、字号）记在本机：半透明只透背景，文字不变淡。
     样式按 Agent 页的紧凑样式（agent.scss）：小字、30px 行高、圆角菜单。 -->
<template>
  <div
    v-show="!dock.minimized"
    class="ag td"
    :class="{ 'is-moving': moving, 'is-clear': clear, 'is-blur': clear && dock.look.blur }"
    :style="rectStyle"
    role="dialog"
    :aria-label="t('termDock.title')"
    @keydown.stop
  >
    <header class="td-head" @pointerdown="startMove" @dblclick="onHeadDblClick">
      <div v-if="dock.tabs.length" class="td-tabs ag-noscroll" role="tablist" :aria-label="t('termDock.title')">
        <div v-for="tab in dock.tabs" :key="tab.key" class="td-tab" :class="{ 'is-on': tab.key === dock.active }">
          <button
            type="button"
            role="tab"
            class="td-tab__btn"
            :aria-selected="tab.key === dock.active"
            :title="[labelOf(tab), tab.cwd, statusText(tab.status)].filter(Boolean).join(' · ')"
            @click="activate(tab.key)"
          >
            <span class="td-dot" :class="`is-${toneOf(tab.status)}`" />
            <UIcon :name="tabIcon(tab)" class="td-tab__ic" />
            <span class="td-tab__t ag-ellipsis">{{ labelOf(tab) }}</span>
          </button>
          <button
            type="button"
            class="td-tab__x"
            :aria-label="t('termDock.closeTab', { name: labelOf(tab) })"
            :title="t('termDock.closeTab', { name: labelOf(tab) })"
            @click="removeTab(tab.key)"
          >
            <UIcon name="i-lucide-x" />
          </button>
        </div>
      </div>
      <span v-else class="td-title">
        <UIcon name="i-lucide-square-terminal" />{{ t('termDock.title') }}
      </span>

      <div v-if="dock.tabs.length" ref="addRef" class="td-add">
        <button
          type="button"
          class="td-btn"
          :class="{ 'is-on': menuOpen }"
          :aria-label="t('termDock.new')"
          :title="t('termDock.new')"
          :aria-expanded="menuOpen"
          @click="toggleMenu"
        >
          <UIcon name="i-lucide-plus" />
        </button>
        <div v-if="menuOpen" class="ag-menu td-menu" role="menu" :aria-label="t('termDock.new')">
          <DockPicker @pick="menuOpen = false" />
        </div>
      </div>

      <span class="td-grow" />
      <div ref="lookRef" class="td-add">
        <button
          type="button"
          class="td-btn"
          :class="{ 'is-on': lookOpen }"
          :aria-label="t('termDock.look')"
          :title="t('termDock.look')"
          :aria-expanded="lookOpen"
          @click="toggleLook"
        >
          <UIcon name="i-lucide-sliders-horizontal" />
        </button>
        <div v-if="lookOpen" class="ag-menu td-menu td-menu--end td-pop" role="group" :aria-label="t('termDock.look')">
          <DockLook />
        </div>
      </div>
      <button type="button" class="td-btn" :aria-label="t('termDock.minimize')" :title="t('termDock.minimize')" @click="minimizeDock">
        <UIcon name="i-lucide-minus" />
      </button>
      <button type="button" class="td-btn td-btn--x" :aria-label="t('termDock.close')" :title="t('termDock.close')" @click="closeDock">
        <UIcon name="i-lucide-x" />
      </button>
    </header>

    <div class="td-body">
      <DockSession
        v-for="tab in dock.tabs"
        v-show="tab.key === dock.active"
        :key="tab.key"
        :ref="(el) => setSessionRef(tab.key, el)"
        :tab="tab"
      />
      <div v-if="!dock.tabs.length" class="td-empty">
        <div class="ag-menu td-empty__list" role="menu" :aria-label="t('termDock.new')">
          <DockPicker />
        </div>
      </div>
    </div>

    <span
      v-for="dir in HANDLES"
      :key="dir"
      class="td-rz"
      :class="`td-rz--${dir}`"
      aria-hidden="true"
      @pointerdown.stop="startResize($event, dir)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { onClickOutside } from '@vueuse/core'
import type { ConsoleSocketStatus } from '@/stores/instance/types'
import DockSession from './DockSession.vue'
import DockPicker from './DockPicker.vue'
import DockLook from './DockLook.vue'
import {
  DOCK_MIN_H,
  DOCK_MIN_W,
  closeDock,
  dock,
  fitRect,
  folderName,
  minimizeDock,
  removeTab,
  saveRect,
  type DockRect,
  type DockTab,
} from './state'

defineOptions({ name: 'TermDock' })

type Dir = 'n' | 's' | 'e' | 'w' | 'ne' | 'nw' | 'se' | 'sw'
const HANDLES: Dir[] = ['n', 's', 'e', 'w', 'ne', 'nw', 'se', 'sw']
// 标签栏上按住这些不算拖窗口、双击不算铺满
const NO_DRAG = 'button, a, input, label, [role="menu"], .td-pop'

const { t } = useI18n()

// 「+」菜单和外观面板，同时只开一个
const menuOpen = ref(false)
const lookOpen = ref(false)
const addRef = useTemplateRef<HTMLElement>('addRef')
const lookRef = useTemplateRef<HTMLElement>('lookRef')
onClickOutside(addRef, () => {
  menuOpen.value = false
})
onClickOutside(lookRef, () => {
  lookOpen.value = false
})
const toggleMenu = () => {
  menuOpen.value = !menuOpen.value
  lookOpen.value = false
}
const toggleLook = () => {
  lookOpen.value = !lookOpen.value
  menuOpen.value = false
}

const clear = computed(() => dock.look.opacity < 1)

const rectStyle = computed(() => ({
  left: `${dock.rect.x}px`,
  top: `${dock.rect.y}px`,
  width: `${dock.rect.w}px`,
  height: `${dock.rect.h}px`,
  // 背景的不透明度：窗口、标签栏、终端底色都按它画（DockSession 也读这个变量）
  '--td-alpha': `${Math.round(dock.look.opacity * 100)}%`,
}))

type SessionRef = InstanceType<typeof DockSession>
const sessionRefs = new Map<string, SessionRef>()
const setSessionRef = (key: string, el: unknown) => {
  if (el) sessionRefs.set(key, el as SessionRef)
  else sessionRefs.delete(key)
}

const refreshActive = async () => {
  await nextTick()
  const r = sessionRefs.get(dock.active)
  r?.fit()
  r?.focus()
}

const activate = (key: string) => {
  dock.active = key
  void refreshActive()
}

// 新开标签、关标签切到相邻的、从最小化还原：重新量尺寸并聚焦
watch(() => dock.active, () => void refreshActive())
watch(
  () => dock.minimized,
  (min) => {
    if (min) {
      menuOpen.value = false
      lookOpen.value = false
    } else void refreshActive()
  },
)

const tabIcon = (tab: DockTab) => {
  if (tab.kind === 'local') return 'i-lucide-square-terminal'
  if (tab.kind === 'login') return 'i-lucide-log-in'
  return 'i-lucide-server'
}

const labelOf = (tab: DockTab) => {
  if (tab.kind !== 'local') return tab.label || '…'
  if (tab.cwd) return t('termDock.localAt', { name: folderName(tab.cwd) })
  return tab.seq > 1 ? t('termDock.localN', { n: tab.seq }) : t('termDock.local')
}

const toneOf = (st: ConsoleSocketStatus) =>
  st === 'connected' ? 'ok' : st === 'connecting' ? 'warn' : st === 'error' ? 'bad' : 'off'

const statusText = (st: ConsoleSocketStatus) => {
  switch (st) {
    case 'connected':
      return t('ssh.common.connected')
    case 'connecting':
      return t('ssh.common.connecting')
    case 'error':
      return t('ssh.common.connectionFailed')
    default:
      return t('termDock.closed')
  }
}

// —— 拖动 / 调大小 ——
const moving = ref(false)
let stopTracking: (() => void) | null = null

const track = (onMove: (e: PointerEvent) => void) => {
  moving.value = true
  menuOpen.value = false
  lookOpen.value = false
  const up = () => {
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', up)
    window.removeEventListener('pointercancel', up)
    stopTracking = null
    moving.value = false
    saveRect()
  }
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', up)
  window.addEventListener('pointercancel', up)
  stopTracking = up
}

const startMove = (e: PointerEvent) => {
  if (e.button !== 0 || (e.target as HTMLElement).closest(NO_DRAG)) return
  e.preventDefault()
  const sx = e.clientX
  const sy = e.clientY
  const start = { ...dock.rect }
  track((ev) => {
    dock.rect = fitRect({ ...start, x: start.x + ev.clientX - sx, y: start.y + ev.clientY - sy })
  })
}

const clamp = (n: number, min: number, max: number) => Math.min(Math.max(n, min), Math.max(min, max))

const startResize = (e: PointerEvent, dir: Dir) => {
  if (e.button !== 0) return
  e.preventDefault()
  const sx = e.clientX
  const sy = e.clientY
  const s: DockRect = { ...dock.rect }
  track((ev) => {
    const dx = ev.clientX - sx
    const dy = ev.clientY - sy
    let left = s.x
    let top = s.y
    let right = s.x + s.w
    let bottom = s.y + s.h
    if (dir.includes('w')) left = clamp(s.x + dx, 0, right - DOCK_MIN_W)
    if (dir.includes('e')) right = clamp(s.x + s.w + dx, left + DOCK_MIN_W, window.innerWidth)
    if (dir.includes('n')) top = clamp(s.y + dy, 0, bottom - DOCK_MIN_H)
    if (dir.includes('s')) bottom = clamp(s.y + s.h + dy, top + DOCK_MIN_H, window.innerHeight)
    dock.rect = { x: left, y: top, w: right - left, h: bottom - top }
  })
}

// 双击标签栏空白处：在「铺满窗口」和上一次的位置大小之间切换
let restoreRect: DockRect | null = null
const onHeadDblClick = (e: MouseEvent) => {
  if ((e.target as HTMLElement).closest(NO_DRAG)) return
  const full ={ x: 0, y: 0, w: window.innerWidth, h: window.innerHeight }
  const isFull = dock.rect.x === 0 && dock.rect.y === 0 && dock.rect.w === full.w && dock.rect.h === full.h
  if (isFull && restoreRect) {
    dock.rect = fitRect(restoreRect)
    restoreRect = null
  } else {
    restoreRect = { ...dock.rect }
    dock.rect = full
  }
  saveRect()
}

const onViewportResize = () => {
  dock.rect = fitRect(dock.rect)
}

onMounted(() => {
  window.addEventListener('resize', onViewportResize)
  void refreshActive()
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onViewportResize)
  stopTracking?.()
})
</script>

<style lang="scss">
@use '@/views/agent/styles/agent.scss';
</style>

<style scoped lang="scss">
.td {
  position: fixed;
  z-index: 2100;
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: var(--m-s);
  border: 1px solid var(--m-line);
  border-radius: 14px;
  box-shadow: 0 24px 60px -18px rgba(20, 10, 30, 0.45);
}
// 半透明：窗口本身不画底，标签栏、终端（DockSession）、空窗口各自按 --td-alpha 画，叠起来才不会变深变浅
.td.is-clear {
  background: transparent;
}
.td.is-clear .td-head {
  background: color-mix(in srgb, var(--m-s) var(--td-alpha), transparent);
  border-radius: 13px 13px 0 0;
}
.td.is-clear .td-empty {
  background: color-mix(in srgb, var(--m-bg) var(--td-alpha), transparent);
}
.td.is-blur {
  -webkit-backdrop-filter: blur(14px) saturate(1.2);
  backdrop-filter: blur(14px) saturate(1.2);
}
.td.is-moving {
  user-select: none;
}
// 拖动、调大小时不让终端吃掉指针事件
.td.is-moving .td-body {
  pointer-events: none;
}

// —— 标签栏（也是拖动把手）——
.td-head {
  display: flex;
  align-items: center;
  gap: 4px;
  min-height: 38px;
  padding: 4px 6px 4px 6px;
  border-bottom: 1px solid var(--m-line);
  cursor: grab;
  touch-action: none;
}
.td.is-moving .td-head {
  cursor: grabbing;
}
.td-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 6px;
  font-size: 12.5px;
  font-weight: 600;
  color: var(--m-ink);
  white-space: nowrap;
}
.td-title .iconify {
  width: 15px;
  height: 15px;
  color: var(--m-ink2);
}
.td-tabs {
  flex: 0 1 auto;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 4px;
  overflow-x: auto;
}
.td-tab {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 1px;
  height: 28px;
  max-width: 220px;
  padding: 0 2px 0 8px;
  border-radius: 8px;
  color: var(--m-ink2);
  font-size: 12.5px;
  font-weight: 500;
  cursor: default;
}
.td-tab:hover {
  background: var(--m-s2);
}
.td-tab.is-on,
.td-tab.is-on:hover {
  background: var(--m-ink);
  color: var(--m-bg);
  font-weight: 600;
}
.td-tab__btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  height: 28px;
  padding: 0 2px 0 0;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  cursor: pointer;
}
.td-tab__ic {
  flex: none;
  width: 14px;
  height: 14px;
}
.td-tab__t {
  max-width: 150px;
}
.td-tab__x {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: inherit;
  cursor: pointer;
}
.td-tab__x:hover {
  background: color-mix(in srgb, currentColor 16%, transparent);
}
.td-tab__x .iconify {
  width: 13px;
  height: 13px;
}
.td-dot {
  flex: none;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--m-ink2);
}
.td-dot.is-ok {
  background: var(--m-ok);
}
.td-dot.is-warn {
  background: var(--m-warn);
}
.td-dot.is-bad {
  background: var(--m-bad);
}
.td-add {
  position: relative;
  flex: none;
}
.td-grow {
  flex: 1;
  min-width: 24px;
  align-self: stretch;
}
.td-btn {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--m-ink2);
  cursor: pointer;
}
.td-btn:hover,
.td-btn.is-on {
  background: var(--m-s2);
  color: var(--m-ink);
}
.td-btn--x:hover {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.td-btn .iconify {
  width: 15px;
  height: 15px;
}
.td-menu {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  z-index: 5;
  width: 280px;
  cursor: default;
}
// 外观面板靠右（按钮在窗口右上角）
.td-menu--end {
  left: auto;
  right: 0;
  width: 240px;
}

// —— 终端区 ——
.td-body {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  border-radius: 0 0 13px 13px;
}
.td-empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: var(--m-bg);
}
.td-empty__list {
  width: 300px;
  max-height: 100%;
  overflow-y: auto;
  box-shadow: none;
}

// —— 调大小的把手（贴着边框，四边 6px、四角 12px）——
.td-rz {
  position: absolute;
  z-index: 3;
  touch-action: none;
}
.td-rz--n,
.td-rz--s {
  left: 12px;
  right: 12px;
  height: 6px;
  cursor: ns-resize;
}
.td-rz--n {
  top: -3px;
}
.td-rz--s {
  bottom: -3px;
}
.td-rz--e,
.td-rz--w {
  top: 12px;
  bottom: 12px;
  width: 6px;
  cursor: ew-resize;
}
.td-rz--e {
  right: -3px;
}
.td-rz--w {
  left: -3px;
}
.td-rz--ne,
.td-rz--nw,
.td-rz--se,
.td-rz--sw {
  width: 12px;
  height: 12px;
}
.td-rz--ne {
  top: -3px;
  right: -3px;
  cursor: nesw-resize;
}
.td-rz--sw {
  bottom: -3px;
  left: -3px;
  cursor: nesw-resize;
}
.td-rz--nw {
  top: -3px;
  left: -3px;
  cursor: nwse-resize;
}
.td-rz--se {
  bottom: -3px;
  right: -3px;
  cursor: nwse-resize;
}
</style>
