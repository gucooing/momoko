<!-- 输入区：附件、@ 引用工作区文件、/ 命令（Momoko 内置、Agent 自己的、技能）、上下文用量、权限模式、
     模型和推理强度（新会话还能换 Agent）。一轮进行中且输入为空时，发送按钮变成停止。 -->
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
          <span v-else class="agc-att">
            <UIcon :name="a.kind === 'image' ? 'i-lucide-image' : 'i-lucide-file'" />
            <span :class="{ 'ag-mono': a.kind === 'ref' }">{{ a.name }}</span>
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
        @compositionstart="composing = true"
        @compositionend="composing = false"
      />
      <div class="agc-box__bar">
        <button type="button" class="ag-ib ag-ib--sm" :class="{ 'is-on': pop === 'clip' }" :aria-label="t('agent.code.composer.attach')" aria-haspopup="menu" :aria-expanded="pop === 'clip'" @click="toggle('clip')">
          <UIcon name="i-lucide-paperclip" />
        </button>
        <button v-if="!narrow" type="button" class="ag-ib ag-ib--sm" :class="{ 'is-on': pop === 'at' }" :aria-label="t('agent.code.composer.ref')" aria-haspopup="dialog" :aria-expanded="pop === 'at'" @click="toggle('at')">
          <UIcon name="i-lucide-at-sign" />
        </button>
        <button v-if="!narrow" type="button" class="ag-ib ag-ib--sm" :class="{ 'is-on': slashOpen }" :aria-label="t('agent.code.composer.slash')" aria-haspopup="menu" :aria-expanded="slashOpen" @click="toggle('slash')">
          <UIcon name="i-lucide-slash" />
        </button>
        <span class="agc-flex" />
        <span v-if="ctxSize > 0" class="agc-ctxwrap">
          <button
            ref="ctxBtn"
            type="button"
            class="agc-cl ag-num"
            aria-haspopup="dialog"
            :aria-expanded="pop === 'ctx' || pop === 'ctxTip'"
            :aria-label="t('agent.code.composer.context', { pct: ctxPct })"
            @click="toggle(narrow ? 'ctxTip' : 'ctx')"
          >
            <svg v-if="narrow" class="agc-ring" viewBox="0 0 16 16" aria-hidden="true">
              <circle cx="8" cy="8" r="6" pathLength="100" />
              <circle v-if="ctxRatio > 0" cx="8" cy="8" r="6" pathLength="100" class="agc-ring__on" :stroke-dasharray="`${ctxRatio} 100`" />
            </svg>
            <template v-else>{{ t('agent.code.composer.context', { pct: ctxPct }) }}</template>
          </button>
          <span v-if="pop === 'ctxTip'" class="agc-ctxtip ag-num" role="status">
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
          :aria-label="showStop ? t('agent.code.composer.stop') : t('agent.code.composer.send')"
          :title="showStop ? t('agent.code.composer.stop') : t('agent.code.composer.send')"
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
    <div v-if="pop === 'clip'" class="ag-menu agc-pop agc-pop--l1" role="menu" :aria-label="t('agent.code.composer.attach')">
      <button type="button" class="ag-mi" role="menuitem" @click="pickFile('image')"><UIcon name="i-lucide-image" />{{ t('agent.code.composer.uploadImage') }}</button>
      <button type="button" class="ag-mi" role="menuitem" @click="pickFile('file')"><UIcon name="i-lucide-file" />{{ t('agent.code.composer.uploadFile') }}</button>
      <button type="button" class="ag-mi" role="menuitem" @click="toggle('at')"><UIcon name="i-lucide-folder" />{{ t('agent.code.composer.fromWorkspace') }}</button>
    </div>

    <!-- @ 引用文件 -->
    <div v-if="pop === 'at'" class="ag-menu agc-pop agc-pop--l2 agc-pop--w330" role="dialog" :aria-label="t('agent.code.composer.ref')">
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
    <div v-if="slashOpen" class="ag-menu agc-pop agc-pop--l3 agc-pop--w380 ag-noscroll" role="menu" :aria-label="t('agent.code.composer.slash')">
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

    <!-- 上下文 -->
    <div v-if="pop === 'ctx'" class="ag-menu agc-pop agc-pop--btn agc-ctx" role="dialog" :aria-label="t('agent.code.ctx.title')">
      <div class="agc-ctx__row"><span class="ag-lbl agc-flex">{{ t('agent.code.ctx.title') }}</span><span class="ag-num ag-muted">{{ t('agent.code.ctx.usedPct', { pct: ctxPct }) }}</span></div>
      <div class="agc-ctx__big ag-num">{{ fmtTokens(ctxUsed) }} <span class="ag-muted">/ {{ fmtTokens(ctxSize) }}</span></div>
      <div class="ag-meter"><span :style="{ width: ctxPct }" /></div>
      <div class="ag-num ag-muted agc-ctx__sub">{{ t('agent.code.ctx.detail', { used: fmtTokens(ctxUsed), size: fmtTokens(ctxSize), left: fmtTokens(Math.max(0, ctxSize - ctxUsed)) }) }}</div>
      <button v-if="infoButton" type="button" class="ag-btn ag-btn--sm agc-ctx__btn" @click="emit('showInfo'), (pop = '')">{{ t('agent.code.ctx.showInfo') }}</button>
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
          <span class="ag-ellipsis agc-flex" :class="{ 'ag-mono': !m.name || m.name === m.id }">{{ m.name || m.id }}</span>
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
      <RouterLink class="ag-mi ag-muted" to="/agent/manage?tab=runtimes">{{ t('agent.code.model.addAgent') }}</RouterLink>
    </div>

    <div v-if="pop === 'model' && sub === 'more'" class="ag-menu agc-pop agc-pop--sub agc-pop--w236 ag-noscroll" role="menu" :aria-label="t('agent.code.model.more')">
      <template v-for="p in otherProviders" :key="p.providerId || 'system'">
        <div class="agc-pop__group">{{ providerLabel(p) }}</div>
        <button v-for="m in p.models" :key="m.id" type="button" class="ag-mi" role="menuitemradio" :aria-checked="false" @click="pickModel(p, m)">
          <span class="ag-ellipsis agc-flex" :class="{ 'ag-mono': !m.name || m.name === m.id }">{{ m.name || m.id }}</span>
          <span v-if="agentSwitch && m.id === p.defaultModel" class="ag-tag">{{ t('agent.code.model.default') }}</span>
        </button>
        <div v-if="!p.models.length" class="agc-pop__note">{{ t('agent.code.model.none') }}</div>
      </template>
      <template v-if="agentSwitch">
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
import { agentKindOf, fileToBase64, fmtTokens } from '@/views/agent/utils/format'
import { modeDesc, modeName, providerLabel } from '@/views/agent/utils/labels'
import type { AgentCommand } from './types'

export interface Selection {
  runtimeId: string
  providerId: string
  model: string
  effort: string
  mode: string
}

/** 待发送的附件：图片和文件带内容（base64），引用的只带路径 */
export type DraftAttachment = AgentAttachmentInput & { kind: 'image' | 'file' | 'ref'; name: string }

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
    infoButton?: boolean
    /** 发出去；返回 false 表示没发成功，输入框保留 */
    onSend: (text: string, attachments: DraftAttachment[]) => Promise<boolean>
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
    infoButton: false,
  },
)
const emit = defineEmits<{ select: [patch: Partial<Selection>]; stop: []; showInfo: [] }>()
const { t } = useI18n()

const UPLOAD_MAX = 20 * 1024 * 1024

const root = ref<HTMLElement | null>(null)
const box = ref<HTMLElement | null>(null)
/** 输入框窄（手机，或者两边面板都开着）：底栏收起 @ 和 /（直接打字就能用），上下文只画个圆环，点开是小气泡 */
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
/** 图片附件的缩略图地址：只在附件变了时拼一次（内容可能有几 MB，别每次敲字都拼） */
const previews = computed(() => atts.value.map((a) => (a.kind === 'image' && a.data ? `data:${a.mime || 'image/png'};base64,${a.data}` : '')))
const sending = ref(false)
const composing = ref(false)

type Pop = '' | 'clip' | 'at' | 'slash' | 'ctx' | 'ctxTip' | 'mode' | 'model'
const pop = ref<Pop>('')
const sub = ref<'' | 'agent' | 'more' | 'effort'>('')
const toggle = (k: Pop) => {
  pop.value = pop.value === k ? '' : k
  sub.value = ''
  if (pop.value === 'at') void openAt()
  void placePop()
}

// 上下文、权限模式、模型的弹层挂在自己的按钮上：从按钮正上方弹出（压在输入框上），左边和按钮对齐，
// 右边到浏览器边上放不下就往左挪
const ctxBtn = ref<HTMLElement | null>(null)
const modeBtn = ref<HTMLElement | null>(null)
const modelBtn = ref<HTMLElement | null>(null)
const popAt = ref({ x: 0, y: 0 })
const placePop = async () => {
  await nextTick()
  const btn = pop.value === 'ctx' ? ctxBtn.value : pop.value === 'mode' ? modeBtn.value : pop.value === 'model' ? modelBtn.value : null
  const menu = root.value?.querySelector<HTMLElement>('.agc-pop--btn')
  if (!root.value || !btn || !menu) return
  const r = root.value.getBoundingClientRect()
  const b = btn.getBoundingClientRect()
  const x = Math.min(b.left, document.documentElement.clientWidth - 8 - menu.offsetWidth)
  popAt.value = { x: x - r.left, y: r.bottom - b.top + 6 }
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
const modelName = computed(() => modelOption.value?.name || props.model)
const efforts = computed(() => modelOption.value?.efforts ?? [])
const effortName = computed(() => efforts.value.find((e) => e.id === props.effort)?.name || props.effort)
const modes = computed(() => runtime.value?.modes ?? [])
const currentMode = computed(() => {
  const m = modes.value.find((x) => x.id === props.mode) ?? (props.mode ? undefined : modes.value[0])
  return m ? modeName(m.id, m.name) : modeName(props.mode)
})

const runtimeSub = (r: AgentRuntimeOptions) => {
  const p = r.providers.find((x) => x.models.some((m) => m.id === x.defaultModel)) ?? r.providers[0]
  return [p ? providerLabel(p) : '', p?.defaultModel].filter(Boolean).join(' · ')
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
/** 已用占比 0–100（窄的输入框里画成圆环） */
const ctxRatio = computed(() => (ctxSize.value > 0 ? Math.min(100, Math.round((ctxUsed.value / ctxSize.value) * 100)) : 0))
const ctxPct = computed(() => `${ctxRatio.value}%`)

// —— / 命令 ——
const BUILTIN = ['install-skill', 'install-mcp']
const slashHidden = ref(false)
const typing = computed(() => /^\/\S*$/.test(draft.value) && !slashHidden.value)
const slashOpen = computed(() => pop.value === 'slash' || typing.value)
const cmdGroups = computed(() => {
  const q = typing.value ? draft.value.slice(1).toLowerCase() : ''
  const match = (name: string, desc = '') => !q || name.toLowerCase().startsWith(q) || desc.toLowerCase().includes(q)
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
  return groups
    .map((g) => ({ ...g, items: g.items.filter((c) => match(c.name, c.description)) }))
    .filter((g) => g.items.length > 0)
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

// 文件、图片拖到整个 Code 页上就加进附件；拖着的时候输入框高亮
const page = ref<HTMLElement | null>(null)
const dropping = ref(false)
let dragDepth = 0
const hasFiles = (e: DragEvent) => !!e.dataTransfer?.types.includes('Files')
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
useEventListener(page, 'dragover', (e: DragEvent) => {
  if (!hasFiles(e)) return
  e.preventDefault()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy'
})
useEventListener(page, 'drop', (e: DragEvent) => {
  if (!hasFiles(e)) return
  e.preventDefault()
  dragDepth = 0
  dropping.value = false
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

// —— 发送 ——
const canSend = computed(() => !!draft.value.trim() || atts.value.length > 0)
const showStop = computed(() => props.busy && !draft.value.trim() && !atts.value.length)

const submit = async () => {
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
    const ok = await props.onSend(text, atts.value)
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

defineExpose({ focus: () => input.value?.focus() })
</script>
