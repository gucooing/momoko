<!-- 编辑器中间：按页面预览。预览直接加载真实页面（同一个应用，iframe + ?__theme_preview=1），
     草稿主题通过 postMessage 套上去；电脑预览按 1280 宽布局缩放，手机预览为 390 宽。
     页面清单来自路由表（theme/pages.ts），新增页面自动出现；关掉「点选元素」后可以在预览里正常点击，
     点进别的页面（包括应用详情这类带参数的页面）编辑器会跟过去。
     「点选元素」开着时点预览里的元素，就选中它用的 token，并用虚线框标出这个 token 用在哪。 -->
<template>
  <section class="epv" :class="`epv--${variant}`">
    <!-- 页面 -->
    <div class="epv-pages" role="tablist" :aria-label="t('theme.editor.pagesLabel')">
      <button
        v-for="p in pageChips"
        :key="p.key"
        type="button"
        role="tab"
        class="epv-chip"
        :class="{ on: p.key === ed.pageKey.value }"
        :aria-selected="p.key === ed.pageKey.value"
        @click="ed.setPage(p.key)"
      >
        {{ p.label }}<span v-if="ed.pagesWithOv.value.has(p.key)" class="epv-dot" :title="t('theme.editor.hasOverride')" />
      </button>
      <AppDropdown v-if="moreGroups.length" align="start" :width="260">
        <template #trigger>
          <button type="button" class="epv-chip is-dashed"><PlusIcon aria-hidden="true" />{{ t('theme.editor.morePages') }}</button>
        </template>
        <template #default="{ close }">
          <div class="epv-more">
            <template v-for="g in moreGroups" :key="g.group">
              <div class="epv-more__lbl">{{ t(`theme.editor.pageGroups.${g.group}`) }}</div>
              <button
                v-for="p in g.items"
                :key="p.key"
                type="button"
                :disabled="p.params"
                :title="p.params ? t('theme.editor.paramsHint') : undefined"
                @click="(ed.setPage(p.key), close())"
              >
                <span>{{ p.label }}</span>
                <small>{{ p.params ? t('theme.editor.paramsHint') : p.key }}</small>
              </button>
            </template>
          </div>
        </template>
      </AppDropdown>
    </div>

    <div v-if="variant === 'pc'" class="epv-bar">
      <button type="button" class="epv-pick" :class="{ on: ed.picking.value }" :aria-pressed="ed.picking.value" @click="ed.picking.value = !ed.picking.value">
        <CursorArrowRaysIcon aria-hidden="true" />{{ t('theme.editor.pick') }}
      </button>
      <span class="epv-bar__hint">{{ ed.picking.value ? t('theme.editor.pickOn') : t('theme.editor.pickOff') }}</span>
      <span class="epv-spacer" />
      <span class="epv-bar__zoom m-num">{{ ed.device.value === 'mobile' ? t('theme.editor.zoomMobile') : t('theme.editor.zoomPc') }}</span>
    </div>

    <div ref="stageRef" class="epv-stage">
      <div class="epv-frame" :class="{ 'is-phone': phone }" :style="frameBox">
        <div v-if="!phone" class="epv-frame__bar">
          <span class="epv-frame__dots"><i /><i /><i /></span>
          <span class="m-mono">{{ urlText }}</span>
        </div>
        <div class="epv-frame__body">
          <iframe
            ref="frameRef"
            :src="src"
            :title="t('theme.editor.previewTitle', { page: ed.pageName.value })"
            :style="frameStyle"
            @load="onLoad"
          />
          <div v-if="!ready" class="epv-loading">{{ t('theme.editor.frameLoading') }}</div>
        </div>
      </div>
      <div class="epv-note" :class="{ 'is-phone': phone }">
        <i :style="{ background: ed.selHex.value }" />{{ t('theme.editor.editing', { name: ed.selName.value }) }}
        <span v-if="variant === 'pc' && ed.sel.value !== 'hero' && !ed.isSizeSel.value" class="m-mono">--{{ ed.sel.value }}</span>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { CursorArrowRaysIcon, PlusIcon } from '@heroicons/vue/24/outline'
import { THEME_PREVIEW_PARAM, type FromFrameMessage, type ToFrameMessage } from '@/theme/frame'
import { useThemeEditor } from './useThemeEditor'

const props = withDefaults(defineProps<{ variant?: 'pc' | 'mobile' }>(), { variant: 'pc' })

const ed = useThemeEditor()
const { t } = ed
const router = useRouter()

const PC_WIDTH = 1280
const PHONE_WIDTH = 390

const phone = computed(() => props.variant === 'mobile' || ed.device.value === 'mobile')

// ---- 页面标签：前几个菜单页面 + 访客页面 + 有单独设置的 / 打开过的页面；其余在「更多页面」里 ----
const MAIN_MENU_PAGES = 4
const pageChips = computed(() => {
  const all = ed.pages.value
  const keys: string[] = all.filter((p) => p.group === 'menu' && !p.params).slice(0, MAIN_MENU_PAGES).map((p) => p.key)
  const add = (k: string) => {
    if (!keys.includes(k)) keys.push(k)
  }
  for (const k of [...ed.pagesWithOv.value, ...ed.extraPages.value]) add(k)
  for (const p of all) if (p.siteTheme) add(p.key)
  add(ed.pageKey.value)
  return keys.map((key) => ({ key, label: ed.labelOf(key) }))
})
const moreGroups = computed(() => {
  const shown = new Set(pageChips.value.map((p) => p.key))
  const rest = ed.pages.value.filter((p) => !shown.has(p.key))
  return (['menu', 'shell', 'public'] as const)
    .map((group) => ({ group, items: rest.filter((p) => p.group === group) }))
    .filter((g) => g.items.length)
})

const urlText = computed(() => `${window.location.host}${ed.pagePath.value}`)

// iframe 只在第一次设置 src，之后切页面用 postMessage 让框内路由跳转（不重新加载整个应用）
const src = router.resolve({ path: ed.pagePath.value, query: { [THEME_PREVIEW_PARAM]: '1' } }).href
const ready = ref(false)
const frameRef = ref<HTMLIFrameElement>()
/** 预览框当前所在的地址（由框内上报），避免把它刚跳过去的地址又发回去 */
let framePath = ''
watch(
  () => ed.pagePath.value,
  (path) => {
    if (ready.value && path !== framePath) post({ type: 'momoko-theme:navigate', path })
  },
)

const post = (msg: ToFrameMessage) => {
  frameRef.value?.contentWindow?.postMessage(msg, window.location.origin)
}
const postDraft = () => {
  if (!ready.value) return
  post({
    type: 'momoko-theme:draft',
    content: JSON.parse(JSON.stringify(ed.content.value)),
    scheme: ed.scheme.value,
    pageKey: ed.pageKey.value,
    picking: ed.picking.value,
    pickMode: ed.tab.value === 'comp' ? 'part' : 'color',
    highlight: ed.picking.value && !ed.sel.value.startsWith('size:') ? ed.sel.value : null,
    highlightPart: ed.picking.value && ed.sel.value.startsWith('size:') ? ed.sel.value.slice(5) : null,
  })
}
let raf = 0
const schedulePost = () => {
  cancelAnimationFrame(raf)
  raf = requestAnimationFrame(postDraft)
}
watch([ed.content, ed.scheme, ed.pageKey, ed.picking, ed.sel, ed.tab], schedulePost)

const onMessage = (e: MessageEvent) => {
  if (e.origin !== window.location.origin || e.source !== frameRef.value?.contentWindow) return
  const d = e.data as FromFrameMessage
  if (!d || typeof d !== 'object') return
  if (d.type === 'momoko-theme:ready') {
    ready.value = true
    postDraft()
  } else if (d.type === 'momoko-theme:pick') {
    if (d.part) ed.select(`size:${d.part}`)
    else if (d.token) ed.select(d.token)
  } else if (d.type === 'momoko-theme:route') {
    ed.followFrame(d.key, d.path, d.terminal)
    framePath = ed.pagePath.value
  }
}
const onLoad = () => {
  // 框内应用启动后会发 ready；这里兜底再发一次草稿
  setTimeout(postDraft, 300)
}
onMounted(() => window.addEventListener('message', onMessage))
onBeforeUnmount(() => {
  window.removeEventListener('message', onMessage)
  cancelAnimationFrame(raf)
})

// ---- 尺寸：电脑预览按 1280 宽缩放 ----
const stageRef = ref<HTMLElement>()
const stage = reactive({ w: 0, h: 0 })
let ro: ResizeObserver | null = null
onMounted(() => {
  if (!stageRef.value) return
  ro = new ResizeObserver(([entry]) => {
    if (!entry) return
    stage.w = entry.contentRect.width
    stage.h = entry.contentRect.height
  })
  ro.observe(stageRef.value)
})
onBeforeUnmount(() => ro?.disconnect())

const scale = computed(() => (phone.value ? 1 : Math.min(1, stage.w / PC_WIDTH || 1)))
const frameBox = computed(() => {
  if (phone.value) {
    const w = Math.min(props.variant === 'mobile' ? stage.w : 360, PHONE_WIDTH)
    return { width: `${w}px`, height: `${Math.min(stage.h, props.variant === 'mobile' ? stage.h : 760)}px` }
  }
  return { width: `${Math.min(stage.w, PC_WIDTH)}px`, height: `${stage.h}px` }
})
const frameStyle = computed(() => {
  if (phone.value) return { width: '100%', height: '100%' }
  const s = scale.value
  const bodyH = Math.max(0, stage.h - 30)
  return { width: `${PC_WIDTH}px`, height: `${bodyH / s}px`, transform: `scale(${s})`, transformOrigin: '0 0' }
})
</script>

<style scoped lang="scss">
.epv {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--m-s2);
}
.epv-pages {
  flex: none;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 12px 16px;
  overflow-x: auto;
  border-bottom: 1px solid var(--m-line);
  background: var(--m-s);
  scrollbar-width: none;
  &::-webkit-scrollbar {
    display: none;
  }
}
.epv--mobile .epv-pages {
  padding: 4px 12px 10px;
}
.epv-chip {
  display: inline-flex;
  align-items: center;
  flex: none;
  gap: 6px;
  min-height: 34px;
  padding: 0 12px;
  border-radius: 999px;
  border: 1.5px solid var(--m-line);
  background: var(--m-s);
  color: var(--m-ink2);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
  svg {
    width: 16px;
    height: 16px;
  }
  &:hover {
    border-color: var(--m-p);
    color: var(--m-pInk);
  }
  &.on {
    background: var(--m-ink);
    border-color: var(--m-ink);
    color: var(--m-bg);
    .epv-dot {
      background: var(--m-a);
    }
  }
  &.is-dashed {
    border-style: dashed;
  }
}
.epv-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--m-p);
}
.epv-more {
  display: flex;
  flex-direction: column;
  padding: 6px;
  max-height: 420px;
  overflow-y: auto;
  button {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
    padding: 8px 10px;
    border: 0;
    border-radius: 10px;
    background: transparent;
    color: var(--m-ink);
    font: inherit;
    font-size: 0.875rem;
    font-weight: 600;
    text-align: left;
    cursor: pointer;
    &:hover:not(:disabled) {
      background: var(--m-s2);
    }
    &:disabled {
      cursor: default;
      color: var(--m-ink2);
    }
    small {
      font-size: 0.6875rem;
      color: var(--m-ink2);
    }
  }
}
.epv-more__lbl {
  padding: 8px 10px 2px;
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--m-ink2);
}
.epv-bar {
  flex: none;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px 0;
}
.epv-pick {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 36px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: var(--m-s);
  color: var(--m-ink);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 600;
  cursor: pointer;
  svg {
    width: 16px;
    height: 16px;
  }
  &.on {
    background: var(--m-ink);
    color: var(--m-bg);
  }
}
.epv-bar__hint {
  font-size: 0.75rem;
  color: var(--m-ink2);
}
.epv-bar__zoom {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--m-ink2);
}
.epv-spacer {
  flex: 1;
}
.epv-stage {
  position: relative;
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  margin: 12px 16px 16px;
}
.epv--mobile .epv-stage {
  margin: 0;
}
.epv-frame {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: 16px;
  border: 1px solid var(--m-line);
  background: #ffffff;
  box-shadow: 0 24px 50px -30px rgba(20, 10, 30, 0.35);
  &.is-phone {
    border-radius: 34px;
    border-width: 6px;
    border-color: #1c1622;
  }
}
.epv--mobile .epv-frame.is-phone {
  border: 0;
  border-radius: 0;
  box-shadow: none;
}
.epv-frame__bar {
  flex: none;
  display: flex;
  align-items: center;
  gap: 10px;
  height: 30px;
  padding: 0 12px;
  background: #f4f1f6;
  border-bottom: 1px solid #e6e0ea;
  .m-mono {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    text-align: center;
    font-size: 0.6875rem;
    color: #8a8090;
  }
}
.epv-frame__dots {
  display: flex;
  gap: 5px;
  i {
    display: block;
    width: 9px;
    height: 9px;
    border-radius: 50%;
    background: #ff5f57;
  }
  i:nth-child(2) {
    background: #febc2e;
  }
  i:nth-child(3) {
    background: #28c840;
  }
}
.epv-frame__body {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  iframe {
    display: block;
    border: 0;
    background: transparent;
  }
}
.epv-loading {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--m-s);
  color: var(--m-ink2);
  font-size: 0.8125rem;
}
.epv-note {
  position: absolute;
  right: 12px;
  bottom: 12px;
  z-index: 5;
  display: flex;
  align-items: center;
  gap: 6px;
  height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(20, 10, 30, 0.8);
  color: #ffffff;
  font-size: 0.6875rem;
  font-weight: 700;
  pointer-events: none;
  i {
    width: 10px;
    height: 10px;
    border-radius: 3px;
  }
  .m-mono {
    opacity: 0.7;
  }
  &.is-phone {
    right: 50%;
    translate: 50% 0;
    bottom: 10px;
  }
}
.epv--mobile .epv-note.is-phone {
  bottom: 24px;
}
</style>
