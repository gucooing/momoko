<!-- 输入区：附件（含引用工作区文件）、/ 命令（Momoko 内置、Agent 自己的、技能）、上下文用量（小圆环，点开是上方的小气泡）、
     权限模式、模型和推理强度（新会话还能换 Agent）。一轮进行中且输入为空时，发送按钮变成停止。 -->
<template>
  <div ref="root" class="agc-composer" :style="{ '--agc-btn-x': `${popAt.x}px`, '--agc-btn-y': `${popAt.y}px` }">
    <div ref="box" class="agc-box" :class="{ 'is-drop': dropping, 'is-narrow': narrow }">
      <div v-if="atts.length" class="agc-box__atts">
        <template v-for="(a, i) in atts" :key="i">
          <span v-if="a.kind === 'image' && a.data" class="agc-thumbwrap">
            <AttImage :name="a.name" :src="previews[i]" />
            <button type="button" class="agc-thumbwrap__x" :aria-label="t('agent.code.composer.removeAtt', { name: a.name })" @click="atts.splice(i, 1)">
              <UIcon name="i-lucide-x" />
            </button>
          </span>
          <span v-else class="agc-att" :title="a.text ? tipOf(a.text) : undefined">
            <UIcon :name="attIcon(a.kind)" />
            <span :class="{ 'ag-mono': a.kind !== 'image' && a.kind !== 'file' }">{{ a.name }}</span>
            <span v-if="a.kind === 'paste' && a.lines" class="ag-muted ag-num">{{ t('agent.code.composer.pastedLines', { n: a.lines }, a.lines) }}</span>
            <button
              v-if="a.kind === 'paste' && a.text"
              type="button"
              class="agc-att__x"
              :aria-label="t('agent.code.composer.unpaste', { name: a.name })"
              :title="t('agent.code.composer.unpaste', { name: a.name })"
              @click="unpaste(i)"
            >
              <UIcon name="i-lucide-text-cursor-input" />
            </button>
            <button type="button" class="agc-att__x" :aria-label="t('agent.code.composer.removeAtt', { name: a.name })" @click="atts.splice(i, 1)">
              <UIcon name="i-lucide-x" />
            </button>
          </span>
        </template>
      </div>
      <textarea
        ref="input"
        v-model="draft"
        class="agc-ta"
        :rows="rows"
        :aria-label="label"
        :placeholder="placeholder"
        @input="slashHidden = false"
        @pointerdown="closePops"
        @keydown="onKey"
        @paste="onPaste"
        @compositionstart="composing = true"
        @compositionend="composing = false"
      />
      <div class="agc-box__bar">
        <button ref="clipBtn" type="button" class="ag-ib ag-ib--sm" :class="{ 'is-on': pop === 'clip' || pop === 'at' }" :aria-label="t('agent.code.composer.attach')" aria-haspopup="menu" :aria-expanded="pop === 'clip'" @click="toggle('clip')">
          <UIcon name="i-lucide-paperclip" />
        </button>
        <button v-if="!narrow" ref="slashBtn" type="button" class="ag-ib ag-ib--sm" :class="{ 'is-on': slashOpen }" :aria-label="t('agent.code.composer.slash')" aria-haspopup="menu" :aria-expanded="slashOpen" @click="toggle('slash')">
          <UIcon name="i-lucide-slash" />
        </button>
        <span class="agc-flex" />
        <span v-if="ctxSize > 0" class="agc-ctxwrap">
          <button
            type="button"
            class="agc-cl ag-num"
            aria-haspopup="dialog"
            :aria-expanded="pop === 'ctx'"
            :aria-label="t('agent.code.composer.context', { pct: ctxPct })"
            @click="toggle('ctx')"
          >
            <svg class="agc-ring" viewBox="0 0 16 16" aria-hidden="true">
              <circle cx="8" cy="8" r="6" pathLength="100" />
              <circle v-if="ctxRatio > 0" cx="8" cy="8" r="6" pathLength="100" class="agc-ring__on" :stroke-dasharray="`${ctxRatio} 100`" />
            </svg>
          </button>
          <span v-if="pop === 'ctx'" class="agc-ctxtip ag-num" role="status">
            <span class="agc-ctxtip__t">{{ t('agent.code.ctx.title') }}</span>
            <span>{{ t('agent.code.ctx.tipPct', { pct: ctxPct, left: `${100 - ctxRatio}%` }) }}</span>
            <span>{{ t('agent.code.ctx.tipTokens', { used: fmtTokens(ctxUsed), size: fmtTokens(ctxSize) }) }}</span>
          </span>
        </span>
        <button v-if="modes.length" ref="modeBtn" type="button" class="agc-cl" aria-haspopup="menu" :aria-expanded="pop === 'mode'" @click="toggle('mode')">
          {{ currentMode }}<UIcon name="i-lucide-chevron-down" />
        </button>
        <button ref="modelBtn" type="button" class="agc-cl agc-cl--model" aria-haspopup="menu" :aria-expanded="pop === 'model'" :aria-label="t('agent.code.composer.pickModel')" @click="toggle('model')">
          <AgAgentTag v-if="agentSwitch && runtime" :kind="agentKindOf(runtime.kind)" :name="runtime.runtimeName" />
          <span class="ag-mono agc-cl__model">{{ modelName || t('agent.code.composer.noModel') }}</span>
          <span v-if="efforts.length && effortName">· {{ effortName }}</span>
          <UIcon name="i-lucide-chevron-down" />
        </button>
        <button
          type="button"
          class="agc-send"
          :class="{ 'is-stop': showStop }"
          :aria-label="sendLabel"
          :title="sendLabel"
          :disabled="sending || (!showStop && !canSend)"
          @click="showStop ? emit('stop') : submit()"
        >
          <UIcon :name="sending ? 'i-lucide-loader-circle' : showStop ? 'i-lucide-square' : 'i-lucide-arrow-up'" :class="{ 'ag-spin': sending }" />
        </button>
      </div>
    </div>

    <input ref="imageInput" type="file" accept="image/*" multiple hidden @change="onFiles($event, 'image')" />
    <input ref="fileInput" type="file" multiple hidden @change="onFiles($event, 'file')" />

    <!-- 附件 -->
    <div v-if="pop === 'clip'" class="ag-menu agc-pop agc-pop--btn agc-pop--w216" role="menu" :aria-label="t('agent.code.composer.attach')">
      <button type="button" class="ag-mi" role="menuitem" @click="pickFile('image')"><UIcon name="i-lucide-image" />{{ t('agent.code.composer.uploadImage') }}</button>
      <button type="button" class="ag-mi" role="menuitem" @click="pickFile('file')"><UIcon name="i-lucide-file" />{{ t('agent.code.composer.uploadFile') }}</button>
      <button type="button" class="ag-mi" role="menuitem" @click="toggle('at')"><UIcon name="i-lucide-folder" />{{ t('agent.code.composer.fromWorkspace') }}</button>
    </div>

    <!-- 引用工作区里的文件（附件菜单里点「从工作区选」，接着挂在回形针上） -->
    <div v-if="pop === 'at'" class="ag-menu agc-pop agc-pop--btn agc-pop--w330" role="dialog" :aria-label="t('agent.code.composer.ref')">
      <template v-if="canRef">
        <label class="ag-inp agc-pop__search">
          <UIcon name="i-lucide-search" />
          <input ref="atInput" v-model="atKeyword" type="search" :placeholder="t('agent.code.composer.searchFile')" :aria-label="t('agent.code.composer.searchFile')" />
        </label>
        <div class="agc-pop__list ag-noscroll">
          <button v-for="p in atFiles" :key="p" type="button" class="ag-mi" @click="addRef(p)">
            <UIcon name="i-lucide-file" /><span class="ag-mono ag-ellipsis agc-flex">{{ p }}</span>
          </button>
          <div v-if="atLoaded && !atFiles.length" class="agc-pop__note">{{ t('agent.code.composer.noFile') }}</div>
        </div>
      </template>
      <div v-else class="agc-pop__note">{{ t('agent.code.composer.noWorkspace') }}</div>
    </div>

    <!-- / 命令 -->
    <div v-if="slashOpen" class="ag-menu agc-pop agc-pop--btn agc-pop--w380 ag-noscroll" role="menu" :aria-label="t('agent.code.composer.slash')">
      <template v-for="g in cmdGroups" :key="g.key">
        <div class="ag-mi-label">{{ g.title }}</div>
        <button
          v-for="(c, i) in g.items"
          :key="c.name"
          type="button"
          class="ag-mi agc-cmd"
          :class="{ 'is-hl': typing && g === cmdGroups[0] && i === 0 }"
          role="menuitem"
          @click="pickCmd(c.name)"
        >
          <span class="ag-mono agc-cmd__n">/{{ c.name }}</span>
          <span class="ag-ellipsis ag-muted agc-flex">{{ c.description }}</span>
        </button>
      </template>
      <div v-if="!cmdGroups.length" class="agc-pop__note">{{ t('agent.code.composer.noCommand') }}</div>
    </div>

    <!-- 权限模式 -->
    <div v-if="pop === 'mode'" class="ag-menu agc-pop agc-pop--btn agc-pop--w330" role="menu" :aria-label="t('agent.code.mode.title')">
      <div class="ag-mi-label">{{ t('agent.code.mode.by', { agent: runtime?.runtimeName || '' }) }}</div>
      <button v-for="m in modes" :key="m.id" type="button" class="ag-mi agc-mode" role="menuitemradio" :aria-checked="m.id === mode" @click="choose({ mode: m.id })">
        <span class="agc-mode__t">
          <span class="agc-mode__n">{{ modeName(m.id, m.name) }}</span>
          <span v-if="modeDesc(m.id, m.description)" class="ag-muted agc-mode__d">{{ modeDesc(m.id, m.description) }}</span>
        </span>
        <UIcon v-if="m.id === mode" name="i-lucide-check" class="agc-on" />
      </button>
    </div>

    <!-- 模型和推理强度：点模型在旁边弹出它的推理强度；选了也不关，点外面才关 -->
    <div v-if="pop === 'model'" ref="modelMenu" class="ag-menu agc-pop agc-pop--btn agc-pop--w252" role="menu" :aria-label="t('agent.code.model.title')" @scroll="placeEffort">
      <template v-if="agentSwitch">
        <button type="button" class="ag-mi" :class="{ 'is-hl': sub === 'agent' }" role="menuitem" aria-haspopup="menu" :aria-expanded="sub === 'agent'" @click="toggleSub('agent')">
          <AgAgentTag v-if="runtime" :kind="agentKindOf(runtime.kind)" :name="runtime.runtimeName" />
          <span class="ag-muted agc-flex agc-small">{{ t('agent.code.model.switchAgent') }}</span>
          <UIcon name="i-lucide-chevron-right" class="ag-muted" />
        </button>
        <div class="ag-mi-sep" />
      </template>
      <template v-if="provider">
        <div class="agc-pop__group">{{ providerLabel(provider) }}</div>
        <button
          v-for="m in provider.models"
          :key="m.id"
          type="button"
          class="ag-mi"
          :class="{ 'is-on': m.id === model }"
          role="menuitemradio"
          :aria-checked="m.id === model"
          :aria-haspopup="m.efforts.length ? 'menu' : undefined"
          :aria-expanded="m.efforts.length ? sub === 'effort' && effortModel?.id === m.id : undefined"
          :data-model="m.id"
          @click="pickModel(provider, m)"
        >
          <span class="ag-ellipsis agc-flex" :class="{ 'ag-mono': !m.name || m.name === m.id }">{{ modelLabel(m) }}</span>
          <span v-if="agentSwitch && m.id === provider.defaultModel" class="ag-tag">{{ t('agent.code.model.default') }}</span>
          <span v-if="m.id === model && efforts.length" class="ag-muted agc-small">{{ effortName }}</span>
          <UIcon v-if="m.id === model" name="i-lucide-check" class="agc-on" />
        </button>
        <div v-if="!provider.models.length" class="agc-pop__note">{{ t('agent.code.model.none') }}</div>
      </template>
      <button v-if="otherProviders.length || agentSwitch" type="button" class="ag-mi" :class="{ 'is-hl': sub === 'more' }" role="menuitem" aria-haspopup="menu" :aria-expanded="sub === 'more'" @click="toggleSub('more')">
        <span class="agc-flex">{{ t('agent.code.model.more') }}</span>
        <UIcon name="i-lucide-chevron-right" class="ag-muted" />
      </button>
    </div>

    <div v-if="pop === 'model' && sub === 'agent'" class="ag-menu agc-pop agc-pop--sub agc-pop--w216" role="menu" :aria-label="t('agent.code.model.whichAgent')">
      <button v-for="r in readyRuntimes" :key="r.runtimeId" type="button" class="ag-mi" :class="{ 'is-on': r.runtimeId === runtimeId }" role="menuitemradio" :aria-checked="r.runtimeId === runtimeId" @click="pickRuntime(r)">
        <span class="agc-mode__t">
          <span class="agc-mode__n">{{ r.runtimeName }}</span>
          <span class="ag-muted agc-mode__d">{{ runtimeSub(r) }}</span>
        </span>
        <UIcon v-if="r.runtimeId === runtimeId" name="i-lucide-check" class="agc-on" />
      </button>
      <RouterLink v-if="canRuntimes" class="ag-mi ag-muted" to="/agent/admin?tab=runtimes">{{ t('agent.code.model.addAgent') }}</RouterLink>
    </div>

    <div v-if="pop === 'model' && sub === 'more'" class="ag-menu agc-pop agc-pop--sub agc-pop--w236 ag-noscroll" role="menu" :aria-label="t('agent.code.model.more')">
      <template v-for="p in otherProviders" :key="p.providerId || 'system'">
        <div class="agc-pop__group">{{ providerLabel(p) }}</div>
        <button v-for="m in p.models" :key="m.id" type="button" class="ag-mi" role="menuitemradio" :aria-checked="false" @click="pickModel(p, m)">
          <span class="ag-ellipsis agc-flex" :class="{ 'ag-mono': !m.name || m.name === m.id }">{{ modelLabel(m) }}</span>
          <span v-if="agentSwitch && m.id === p.defaultModel" class="ag-tag">{{ t('agent.code.model.default') }}</span>
        </button>
        <div v-if="!p.models.length" class="agc-pop__note">{{ t('agent.code.model.none') }}</div>
      </template>
      <template v-if="agentSwitch && canProviders">
        <div v-if="otherProviders.length" class="ag-mi-sep" />
        <RouterLink class="ag-mi ag-muted" to="/agent/manage?tab=providers">{{ t('agent.code.model.manageProviders') }}</RouterLink>
      </template>
    </div>

    <div
      v-if="pop === 'model' && sub === 'effort' && effortModel?.efforts.length"
      ref="effortMenu"
      class="ag-menu agc-pop agc-pop--at agc-pop--w140"
      :style="{ '--agc-pop-x': `${effortPos.x}px`, '--agc-pop-y': `${effortPos.y}px` }"
      role="menu"
      :aria-label="t('agent.code.model.effort')"
    >
      <button
        v-for="e in effortModel.efforts"
        :key="e.id"
        type="button"
        class="ag-mi"
        :class="{ 'is-on': effortModel.id === model && e.id === effort }"
        role="menuitemradio"
        :aria-checked="effortModel.id === model && e.id === effort"
        @click="pickEffort(e.id)"
      >
        <span class="agc-flex">{{ e.name || e.id }}</span>
        <UIcon v-if="effortModel.id === model && e.id === effort" name="i-lucide-check" class="agc-on" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onClickOutside, useDebounceFn, useEventListener, useResizeObserver } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { type AgentAttachmentInput, searchAgentFiles } from '@/api/agent'
import type { AgentModelOption, AgentProviderOptions, AgentRuntimeOptions } from '@/types/v1/agent'
import AgAgentTag from '@/views/agent/components/AgAgentTag.vue'
import AttImage from './AttImage.vue'
import { AGENT_PATH_MIME, agentKindOf, bytesToBase64, fileToBase64, fmtTokens } from '@/views/agent/utils/format'
import { modelLabel } from '@/views/agent/composables/useAgentModels'
import { modeDesc, modeName, providerLabel } from '@/views/agent/utils/labels'
import { composerDrafts } from './drafts'
import type { AgentCommand } from './types'
import { useMenuStore } from '@/stores/menu'

export interface Selection {
  runtimeId: string
  providerId: string
  model: string
  effort: string
  mode: string
}

/** 待发送的附件：图片和文件带内容（base64），引用的只带路径；snippet 是打开的文件里选中的一段，内容是那段文字；
 *  paste 是粘贴进来的一大段文字转成的文本文件（服务器存进会话的临时目录），text 是原文、lines 是行数，可以再放回输入框 */
export type DraftAttachment = AgentAttachmentInput & {
  kind: 'image' | 'file' | 'paste' | 'ref' | 'snippet'
  name: string
  text?: string
  lines?: number
}
/** 一段选中的文字：path 相对会话目录，startLine / endLine 从 1 数 */
export interface DraftSnippet {
  path: string
  startLine: number
  endLine: number
  text: string
}
/** 一轮还没结束时怎么发：steer 插进正在做的事里（ACP v2），queue 排队、等这一轮结束再发 */
export type SendMode = 'steer' | 'queue'

defineOptions({ name: 'AgentComposer' })

const props = withDefaults(
  defineProps<{
    runtimes: AgentRuntimeOptions[]
    runtimeId: string
    providerId: string
    model: string
    effort: string
    mode: string
    /** 新会话：可以换 Agent */
    agentSwitch?: boolean
    /** 一轮正在进行 */
    busy?: boolean
    commands?: AgentCommand[]
    skills?: string[]
    sessionId?: string
    workspaceId?: string
    contextUsed?: number
    contextSize?: number
    rows?: number
    label: string
    placeholder: string
    /** 发出去；返回 false 表示没发成功，输入框保留。一轮还没结束时 mode 是 queue（排着，等这一轮结束，或者在消息上点「立即发送」） */
    onSend: (text: string, attachments: DraftAttachment[], mode?: SendMode) => Promise<boolean>
  }>(),
  {
    agentSwitch: false,
    busy: false,
    commands: () => [],
    skills: () => [],
    sessionId: '',
    workspaceId: '',
    contextUsed: 0,
    contextSize: 0,
    rows: 2,
  },
)
const emit = defineEmits<{ select: [patch: Partial<Selection>]; stop: [] }>()
const { t } = useI18n()
const menuStore = useMenuStore()
// 能管大家共用的运行时（Agent 管理页里，要 agent:manage）；自己的模型供应商在 Agent 设置页里
const canRuntimes = computed(() => menuStore.hasMenuPath('/agent/admin') && menuStore.hasButtonPermission('agent:manage'))
const canProviders = computed(() => menuStore.hasMenuPath('/agent/manage'))

const UPLOAD_MAX = 20 * 1024 * 1024

const root = ref<HTMLElement | null>(null)
const box = ref<HTMLElement | null>(null)
/** 输入框窄（手机，或者两边面板都开着）：底栏收起 /（直接打字就能用） */
const narrow = ref(false)
useResizeObserver(box, ([entry]) => {
  if (entry) narrow.value = entry.contentRect.width < 520
})
const input = ref<HTMLTextAreaElement | null>(null)
const imageInput = ref<HTMLInputElement | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const atInput = ref<HTMLInputElement | null>(null)
const draft = ref('')
const atts = ref<DraftAttachment[]>([])
// 没发出去的草稿按会话存着：切到文件标签（输入框卸下）、切到别的会话再回来，接着写
const draftKey = computed(() => props.sessionId || 'new')
const saveDraft = (key: string) => {
  if (draft.value.trim() || atts.value.length) composerDrafts.set(key, { text: draft.value, atts: [...atts.value] })
  else composerDrafts.delete(key)
}
const restoreDraft = (key: string) => {
  const saved = composerDrafts.get(key)
  draft.value = saved?.text ?? ''
  atts.value = saved ? [...saved.atts] : []
}
watch(draftKey, (key, old) => {
  saveDraft(old)
  restoreDraft(key)
})
onMounted(() => restoreDraft(draftKey.value))
onBeforeUnmount(() => saveDraft(draftKey.value))
/** 图片附件的缩略图地址：只在附件变了时拼一次（内容可能有几 MB，别每次敲字都拼） */
const previews = computed(() => atts.value.map((a) => (a.kind === 'image' && a.data ? `data:${a.mime || 'image/png'};base64,${a.data}` : '')))
const sending = ref(false)
const composing = ref(false)

type Pop = '' | 'clip' | 'at' | 'slash' | 'ctx' | 'mode' | 'model'
const pop = ref<Pop>('')
const sub = ref<'' | 'agent' | 'more' | 'effort'>('')
const toggle = (k: Pop) => {
  pop.value = pop.value === k ? '' : k
  sub.value = ''
  if (pop.value === 'at') void openAt()
  void placePop()
}

// 底栏按钮点开的弹层挂在自己的按钮上：从按钮正上方弹出（压在输入框上），左边和按钮对齐；
// 打字打出来的 / 命令挂在输入框里那个「/」上：贴着这一行字的上方弹出，跟着字走，不挡住正在打的字。
// 右边到浏览器边上放不下就往左挪。上下文的小气泡套在圆环外面，用样式放在它正上方
const clipBtn = ref<HTMLElement | null>(null)
const slashBtn = ref<HTMLElement | null>(null)
const modeBtn = ref<HTMLElement | null>(null)
const modelBtn = ref<HTMLElement | null>(null)
/** 现在开着的弹层挂在哪（视口里的左上角）：按钮，或者打出来的「/」 */
const anchorOf = (): { left: number; top: number } | null => {
  const btn =
    pop.value === 'mode'
      ? modeBtn.value
      : pop.value === 'model'
        ? modelBtn.value
        : pop.value === 'clip' || pop.value === 'at'
          ? clipBtn.value
          : pop.value === 'slash'
            ? slashBtn.value
            : null
  if (btn) return btn.getBoundingClientRect()
  // 打字打出来的：输入框里只有「/命令名」一行，「/」就在第一行开头
  const ta = input.value
  if (!typing.value || !ta) return null
  const rect = ta.getBoundingClientRect()
  const cs = getComputedStyle(ta)
  return {
    left: rect.left + parseFloat(cs.borderLeftWidth) + parseFloat(cs.paddingLeft),
    top: rect.top + parseFloat(cs.borderTopWidth) + parseFloat(cs.paddingTop) - ta.scrollTop,
  }
}
const popAt = ref({ x: 0, y: 0 })
const placePop = async () => {
  await nextTick()
  const at = anchorOf()
  const menu = root.value?.querySelector<HTMLElement>('.agc-pop--btn')
  if (!root.value || !at || !menu) return
  const r = root.value.getBoundingClientRect()
  const x = Math.min(at.left, document.documentElement.clientWidth - 8 - menu.offsetWidth)
  popAt.value = { x: x - r.left, y: r.bottom - at.top + 6 }
  if (pop.value === 'model') void placeEffort()
}
const toggleSub = (k: 'agent' | 'more') => {
  sub.value = sub.value === k ? '' : k
}
/** 收起所有弹层：点到输入区外面，或者点回输入框里。 */
const closePops = () => {
  pop.value = ''
  sub.value = ''
  slashHidden.value = true
}
onClickOutside(root, closePops)

// —— 当前选择 ——
const runtime = computed(() => props.runtimes.find((r) => r.runtimeId === props.runtimeId))
const readyRuntimes = computed(() => props.runtimes.filter((r) => r.ready))
const provider = computed(() => runtime.value?.providers.find((p) => p.providerId === props.providerId))
const otherProviders = computed(() => (runtime.value?.providers ?? []).filter((p) => p !== provider.value))
const modelOption = computed(() => provider.value?.models.find((m) => m.id === props.model))
const modelName = computed(() => modelLabel(modelOption.value, props.model))
const efforts = computed(() => modelOption.value?.efforts ?? [])
const effortName = computed(() => efforts.value.find((e) => e.id === props.effort)?.name || props.effort)
const modes = computed(() => runtime.value?.modes ?? [])
const currentMode = computed(() => {
  const m = modes.value.find((x) => x.id === props.mode) ?? (props.mode ? undefined : modes.value[0])
  return m ? modeName(m.id, m.name) : modeName(props.mode)
})

const runtimeSub = (r: AgentRuntimeOptions) => {
  const p = r.providers.find((x) => x.models.some((m) => m.id === x.defaultModel)) ?? r.providers[0]
  const m = p?.models.find((x) => x.id === p.defaultModel)
  return [p ? providerLabel(p) : '', modelLabel(m, p?.defaultModel)].filter(Boolean).join(' · ')
}

const choose = (patch: Partial<Selection>) => {
  pop.value = ''
  sub.value = ''
  emit('select', patch)
}

// —— 模型菜单：选了不关，推理强度挂在点中的模型旁边 ——
const modelMenu = ref<HTMLElement | null>(null)
const effortMenu = ref<HTMLElement | null>(null)
const effortModelId = ref('')
const effortPos = ref({ x: 0, y: 0 })
// 从「更多模型」换了供应商后，要等新供应商成了当前的，那一行出现在主菜单里才有得挂
const effortModel = computed(() => provider.value?.models.find((m) => m.id === effortModelId.value))

// 推理强度列表放在模型菜单右边，右边到浏览器边上放不下才放左边；
// 顶边对齐点中的那一行，但不超出模型菜单的上下边
const placeEffort = async () => {
  await nextTick()
  const box = root.value
  const menu = modelMenu.value
  const list = effortMenu.value
  const row = menu?.querySelector<HTMLElement>(`[data-model="${CSS.escape(effortModelId.value)}"]`)
  if (!box || !menu || !list || !row) return
  const b = box.getBoundingClientRect()
  const m = menu.getBoundingClientRect()
  const gap = 6
  const x = m.right + gap + list.offsetWidth <= document.documentElement.clientWidth - 8 ? m.right + gap : m.left - gap - list.offsetWidth
  const y = Math.max(m.top, Math.min(row.getBoundingClientRect().top - 4, m.bottom - list.offsetHeight))
  effortPos.value = { x: x - b.left, y: y - b.top }
}
watch(effortModel, () => void placeEffort())
useEventListener(window, 'resize', () => void placePop())

const pickModel = (p: AgentProviderOptions, m: AgentModelOption) => {
  effortModelId.value = m.id
  sub.value = m.efforts.length ? 'effort' : ''
  if (p.providerId !== props.providerId || m.id !== props.model) {
    emit('select', { providerId: p.providerId, model: m.id, effort: m.efforts.some((e) => e.id === props.effort) ? props.effort : m.defaultEffort })
  }
  void placeEffort()
}

const pickEffort = (id: string) => {
  if (id !== props.effort) emit('select', { effort: id })
}

const pickRuntime = (r: AgentRuntimeOptions) => {
  const p = r.providers[0]
  const m = p?.models.find((x) => x.id === p.defaultModel) ?? p?.models[0]
  sub.value = ''
  emit('select', { runtimeId: r.runtimeId, providerId: p?.providerId ?? '', model: m?.id ?? '', effort: m?.defaultEffort ?? '', mode: r.defaultMode })
}

// —— 上下文 ——
const ctxUsed = computed(() => Number(props.contextUsed || 0))
const ctxSize = computed(() => Number(props.contextSize || 0))
/** 已用占比 0–100（画成圆环） */
const ctxRatio = computed(() => (ctxSize.value > 0 ? Math.min(100, Math.round((ctxUsed.value / ctxSize.value) * 100)) : 0))
const ctxPct = computed(() => `${ctxRatio.value}%`)

// —— / 命令 ——
const BUILTIN = ['install-skill', 'install-mcp']
const slashHidden = ref(false)
const typing = computed(() => /^\/\S*$/.test(draft.value) && !slashHidden.value)
const slashOpen = computed(() => pop.value === 'slash' || typing.value)
// 打字打出 / 时命令列表也要量一下挂在哪
watch(slashOpen, (open) => {
  if (open) void placePop()
})
const cmdGroups = computed(() => {
  const q = typing.value ? draft.value.slice(1).toLowerCase() : ''
  // 只按命令名筛：开头（或插件前缀 xx: 之后）对上的排前面，名字里含有的排后面
  const rank = (name: string) => {
    if (!q) return 0
    const n = name.toLowerCase()
    if (n.startsWith(q) || n.slice(n.indexOf(':') + 1).startsWith(q)) return 0
    return n.includes(q) ? 1 : -1
  }
  const skills = new Set(props.skills)
  const groups: { key: string; title: string; items: { name: string; description: string }[] }[] = []
  if (runtime.value?.caps?.mcpHttp) {
    groups.push({ key: 'builtin', title: t('agent.code.cmd.builtin'), items: BUILTIN.map((name) => ({ name, description: t(`agent.code.cmd.${name}`) })) })
  }
  groups.push({
    key: 'agent',
    title: t('agent.code.cmd.agent', { agent: runtime.value?.runtimeName || '' }),
    items: props.commands.filter((c) => !skills.has(c.name)).map((c) => ({ name: c.name, description: c.description || '' })),
  })
  groups.push({
    key: 'skills',
    title: t('agent.code.cmd.skills'),
    items: [...skills].map((name) => ({ name, description: props.commands.find((c) => c.name === name)?.description || '' })),
  })
  // 有开头对上的组排到前面，回车选的、高亮的就是最贴近的那条
  return groups
    .map((g) => {
      const hits = g.items
        .map((c) => ({ c, r: rank(c.name) }))
        .filter((x) => x.r >= 0)
        .sort((a, b) => a.r - b.r)
      return { ...g, best: hits[0]?.r ?? 2, items: hits.map((x) => x.c) }
    })
    .filter((g) => g.items.length > 0)
    .sort((a, b) => a.best - b.best)
})
const pickCmd = (name: string) => {
  draft.value = `/${name} `
  pop.value = ''
  slashHidden.value = true
  void nextTick(() => input.value?.focus())
}

// 内置命令展开成给 Agent 的一条消息：让它读说明，再调用 Momoko 的 MCP 工具（会照常走批准）。
const expandBuiltin = (text: string): { text: string; toast: string } | null => {
  const m = /^\/(install-skill|install-mcp)\b\s*([\s\S]*)$/.exec(text)
  if (!m) return null
  const arg = (m[2] ?? '').trim()
  if (!arg) {
    feedback.warning(t(`agent.code.cmd.${m[1]}Need`))
    return { text: '', toast: '' }
  }
  if (m[1] === 'install-skill') {
    return {
      text: `Install a skill into Momoko from: ${arg}\nIf it is a Git repository URL use source "git"; if it is a link to a .zip file use source "zip". If the location is unclear, read its README first. Then call the install_skill tool of the "momoko" MCP server.`,
      toast: t('agent.code.cmd.skillHanded', { agent: runtime.value?.runtimeName || '' }),
    }
  }
  return {
    text: `Register this MCP server in Momoko: ${arg}\nRead its README or install notes to find the exact launch command, arguments and environment variables (stdio), or its URL and headers (http). Then call the add_mcp_server tool of the "momoko" MCP server.`,
    toast: t('agent.code.cmd.mcpHanded', { agent: runtime.value?.runtimeName || '' }),
  }
}

// —— 附件 ——
const pickFile = (kind: 'image' | 'file') => {
  pop.value = ''
  ;(kind === 'image' ? imageInput : fileInput).value?.click()
}
const addFiles = async (files: File[], kind?: 'image' | 'file') => {
  for (const f of files) {
    if (f.size > UPLOAD_MAX) {
      feedback.warning(t('agent.code.composer.tooLarge', { name: f.name }))
      continue
    }
    atts.value.push({ kind: kind ?? (f.type.startsWith('image/') ? 'image' : 'file'), name: f.name, mime: f.type, data: await fileToBase64(f) })
  }
}
const onFiles = (e: Event, kind: 'image' | 'file') => {
  const el = e.target as HTMLInputElement
  const files = Array.from(el.files ?? [])
  el.value = ''
  void addFiles(files, kind)
}

// 文件、图片拖到整个 Code 页上就加进附件；右侧「文件」里的文件、文件夹拖过来加成引用。拖着的时候输入框高亮
const page = ref<HTMLElement | null>(null)
const dropping = ref(false)
let dragDepth = 0
const hasFiles = (e: DragEvent) => !!e.dataTransfer?.types.some((t) => t === 'Files' || t === AGENT_PATH_MIME)
onMounted(() => {
  page.value = root.value?.closest<HTMLElement>('.agc') ?? null
})
useEventListener(page, 'dragenter', (e: DragEvent) => {
  if (!hasFiles(e)) return
  dragDepth++
  dropping.value = true
})
useEventListener(page, 'dragleave', (e: DragEvent) => {
  if (!hasFiles(e)) return
  dragDepth = Math.max(0, dragDepth - 1)
  if (!dragDepth) dropping.value = false
})
/** 落在文件树里的拖放归文件树（挪位置），这里不接 */
const inTree = (e: DragEvent) => e.target instanceof Element && !!e.target.closest('.agc-ex')
useEventListener(page, 'dragover', (e: DragEvent) => {
  if (!hasFiles(e) || inTree(e)) return
  e.preventDefault()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy'
})
useEventListener(page, 'drop', (e: DragEvent) => {
  if (!hasFiles(e)) return
  dragDepth = 0
  dropping.value = false
  if (inTree(e)) return
  e.preventDefault()
  const path = e.dataTransfer?.getData(AGENT_PATH_MIME)
  if (path) {
    if (canRef.value) addRef(path)
    return
  }
  void addFiles(Array.from(e.dataTransfer?.files ?? []))
})

const canRef = computed(() => !!props.sessionId || !!props.workspaceId)
const atKeyword = ref('')
const atFiles = ref<string[]>([])
const atLoaded = ref(false)
let atSeq = 0
const searchFiles = async () => {
  const id = ++atSeq
  try {
    const { data } = await searchAgentFiles({ sessionId: props.sessionId, workspaceId: props.workspaceId, keyword: atKeyword.value.trim() })
    if (id !== atSeq) return
    atFiles.value = data.paths ?? []
  } catch {
    if (id === atSeq) atFiles.value = []
  } finally {
    if (id === atSeq) atLoaded.value = true
  }
}
const openAt = async () => {
  if (!canRef.value) return
  atLoaded.value = false
  await nextTick()
  atInput.value?.focus()
  await searchFiles()
}
watch(atKeyword, useDebounceFn(() => void searchFiles(), 200))
const addRef = (path: string) => {
  if (!atts.value.some((a) => a.kind === 'ref' && a.path === path)) atts.value.push({ kind: 'ref', name: path, path })
  pop.value = ''
  void nextTick(() => input.value?.focus())
}
const SNIPPET_MAX = 256 * 1024
/** 打开的文件里选中的一段：加成附件，名字是「路径:起止行」 */
const addSnippet = (sn: DraftSnippet) => {
  const name = sn.startLine === sn.endLine ? `${sn.path}:${sn.startLine}` : `${sn.path}:${sn.startLine}-${sn.endLine}`
  const bytes = new TextEncoder().encode(sn.text)
  if (!bytes.length) return
  if (bytes.length > SNIPPET_MAX) {
    feedback.warning(t('agent.code.composer.snippetTooLarge'))
    return
  }
  if (!atts.value.some((a) => a.kind === 'snippet' && a.name === name)) {
    atts.value.push({ kind: 'snippet', name, path: sn.path, text: sn.text, data: bytesToBase64(bytes) })
  }
  void nextTick(() => input.value?.focus())
}
const ATT_ICON: Record<string, string> = { image: 'i-lucide-image', paste: 'i-lucide-file-text', snippet: 'i-lucide-text-select', ref: 'i-lucide-at-sign' }
const attIcon = (kind: string) => ATT_ICON[kind] ?? 'i-lucide-file'
/** 附件上悬停显示的文字：只取开头一段，几 MB 的原文不往 title 里塞 */
const TIP_MAX = 800
const tipOf = (text: string) => (text.length > TIP_MAX ? `${text.slice(0, TIP_MAX)}…` : text)

// —— 粘贴 ——
// 粘贴进来的是图片、文件：直接加成附件。是文字但太长（日志、整份文件之类）：不放进输入框，转成一个文本文件附件，
// 服务器存进会话的临时目录，Agent 收到的只是它的路径、需要时自己去读，不会把整段原文塞进上下文。
const PASTE_MAX_CHARS = 5000
const PASTE_MAX_LINES = 100
const lineCount = (text: string) => {
  let n = 1
  for (let i = text.indexOf('\n'); i !== -1 && i < text.length - 1; i = text.indexOf('\n', i + 1)) n++
  return n
}
/** 这份草稿里同一类粘贴的下一个编号：pasted-text-1.txt、pasted-text-2.txt…… */
const nextPasteNo = (prefix: string) => {
  let max = 0
  for (const a of atts.value) {
    const m = a.name.startsWith(prefix) ? /^-(\d+)\./.exec(a.name.slice(prefix.length)) : null
    if (m) max = Math.max(max, Number(m[1]))
  }
  return max + 1
}
const addPastedText = (text: string, lines: number) => {
  const name = `pasted-text-${nextPasteNo('pasted-text')}.txt`
  const bytes = new TextEncoder().encode(text)
  if (bytes.length > UPLOAD_MAX) {
    feedback.warning(t('agent.code.composer.tooLarge', { name }))
    return
  }
  atts.value.push({ kind: 'paste', name, mime: 'text/plain', data: bytesToBase64(bytes), text, lines })
}
/** 截图之类粘贴出来的图片浏览器都叫 image.png：换成带编号的名字，几张图分得清 */
const namePasted = (f: File) => {
  if (f.name && !/^image\.\w+$/i.test(f.name)) return f
  const sub = f.type.split('/')[1]?.split('+')[0] || 'png'
  const ext = sub === 'jpeg' ? 'jpg' : sub
  return new File([f], `pasted-image-${nextPasteNo('pasted-image')}.${ext}`, { type: f.type })
}
const onPaste = (e: ClipboardEvent) => {
  const cd = e.clipboardData
  if (!cd) return
  // 有文字就按文字处理：从 Excel、Word 复制时剪贴板里还会顺带一张渲染出来的图片，那张不要
  const text = cd.getData('text/plain')
  if (text) {
    const lines = lineCount(text)
    if (text.length <= PASTE_MAX_CHARS && lines <= PASTE_MAX_LINES) return
    e.preventDefault()
    addPastedText(text, lines)
    return
  }
  let files = Array.from(cd.files ?? [])
  if (!files.length) files = Array.from(cd.items ?? []).flatMap((it) => (it.kind === 'file' ? (it.getAsFile() ?? []) : []))
  if (!files.length) return
  e.preventDefault()
  // 一张一张改名：编号要看前一张有没有已经加进去
  void (async () => {
    for (const f of files) await addFiles([namePasted(f)])
  })()
}
/** 粘贴转成的文件放回输入框：插在光标处，变回普通文字 */
const unpaste = (i: number) => {
  const a = atts.value[i]
  if (a?.kind !== 'paste' || a.text === undefined) return
  const text = a.text
  atts.value.splice(i, 1)
  const el = input.value
  const start = el?.selectionStart ?? draft.value.length
  const end = el?.selectionEnd ?? start
  draft.value = draft.value.slice(0, start) + text + draft.value.slice(end)
  void nextTick(() => {
    el?.focus()
    el?.setSelectionRange(start + text.length, start + text.length)
  })
}


// —— 发送 ——
const canSend = computed(() => !!draft.value.trim() || atts.value.length > 0)
const showStop = computed(() => props.busy && !draft.value.trim() && !atts.value.length)
const sendLabel = computed(() => {
  if (showStop.value) return t('agent.code.composer.stop')
  if (!props.busy) return t('agent.code.composer.send')
  return t('agent.code.composer.queue')
})

const submit = async (mode?: SendMode) => {
  if (sending.value || !canSend.value) return
  let text = draft.value.trim()
  let toast = ''
  const builtin = expandBuiltin(text)
  if (builtin) {
    if (!builtin.text) return
    text = builtin.text
    toast = builtin.toast
  }
  sending.value = true
  try {
    const ok = await props.onSend(
      text,
      // text 只是页面上提示用的（选中的那段），发出去的是 data
      atts.value.map((a) => ({ kind: a.kind, name: a.name, mime: a.mime, data: a.data, path: a.path })),
      // 一轮还没结束：先排着，等这一轮结束再发，或者在排着的消息上点「立即发送」
      props.busy ? (mode ?? 'queue') : undefined,
    )
    if (ok) {
      draft.value = ''
      atts.value = []
      if (toast) feedback.success(toast)
    }
  } finally {
    sending.value = false
  }
}

const onKey = (e: KeyboardEvent) => {
  if (typing.value) {
    if (e.key === 'Escape') {
      slashHidden.value = true
      return
    }
    if (e.key === 'Enter' && !e.shiftKey && !composing.value) {
      const first = cmdGroups.value[0]?.items[0]
      if (first) {
        e.preventDefault()
        pickCmd(first.name)
        return
      }
    }
  }
  if (e.key === 'Enter' && !e.shiftKey && !composing.value && !e.isComposing) {
    e.preventDefault()
    if (!showStop.value) void submit()
  }
}

defineExpose({
  focus: () => input.value?.focus(),
  /** 右侧「文件」里点了「添加到对话」之类：加一个引用 */
  addRef: (path: string) => {
    if (canRef.value) addRef(path)
  },
  addSnippet,
  /** 把文字放回输入框最前面（停下时还没发出去的排队消息） */
  restore: (text: string) => {
    draft.value = draft.value ? `${text}\n\n${draft.value}` : text
  },
})
</script>
