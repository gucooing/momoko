<!-- 新会话：中间是引导语和输入区（Agent、模型、推理强度、权限模式在输入区里选），右侧是本次会话的设置：
     工作区（默认值来自工作区）、附加文件夹、在独立工作树里运行、MCP 服务器，选了工作区下面还有它的文件。
     点文件在标签页里打开（只读，选中一段可以加到附件），拖到输入框里是引用。发出第一条消息就新建会话。 -->
<template>
  <section class="agc-card agc-main" :aria-label="t('agent.common.newSession')">
    <header class="agc-head">
      <button type="button" class="ag-ib ag-ib--sm" :aria-label="leftOn ? t('agent.code.hideSide') : t('agent.code.showSide')" :title="leftOn ? t('agent.code.hideSide') : t('agent.code.showSide')" :aria-pressed="leftOn" @click="emit('toggleLeft')">
        <UIcon name="i-lucide-panel-left" />
      </button>
      <div class="agc-tabs ag-noscroll" role="tablist" :aria-label="t('agent.code.tabs')">
        <div class="agc-tt" :class="{ 'is-on': !ctab }">
          <button type="button" class="agc-tt__btn" role="tab" :aria-selected="!ctab" @click="ctab = ''">
            <UIcon name="i-lucide-message-square" /><span class="ag-ellipsis agc-tt__title">{{ t('agent.common.newSession') }}</span>
          </button>
        </div>
        <div v-for="p in openFiles" :key="p" class="agc-tt" :class="{ 'is-on': ctab === p }">
          <button type="button" class="agc-tt__btn" role="tab" :aria-selected="ctab === p" :title="p" @click="ctab = p">
            <UIcon name="i-lucide-file" /><span class="ag-mono agc-tt__file">{{ shortName(p) }}</span>
          </button>
          <button
            type="button"
            class="agc-tt__x"
            :class="{ 'is-dirty': tabDirty(p) }"
            :aria-label="t('agent.code.closeTab', { path: p })"
            :title="tabDirty(p) ? t('agent.code.file.unsaved') : undefined"
            @click="closeFile(p)"
          >
            <span class="agc-tt__dot" /><UIcon name="i-lucide-x" />
          </button>
        </div>
      </div>
      <button type="button" class="ag-ib ag-ib--sm" :aria-label="rightOn ? t('agent.code.hideRight') : t('agent.code.showRight')" :title="rightOn ? t('agent.code.hideRight') : t('agent.code.showRight')" :aria-pressed="rightOn" @click="emit('toggleRight')">
        <UIcon name="i-lucide-panel-right" />
      </button>
    </header>

    <FileTab
      v-if="ctab && workspace"
      :key="`${workspace.id}|${tabKeys.key(ctab)}`"
      :workspace-id="workspace.id"
      :path="ctab"
      :cwd="workspace.path"
      @attach="attachSnippet"
      @open="openFile"
      @navigate="onTabNavigate"
    />

    <!-- 打开文件标签时输入区只是藏起来，写了一半的话还在 -->
    <div v-show="!ctab" class="agc-hero">
      <span class="agc-hero__mark"><UIcon name="i-lucide-code-xml" /></span>
      <h1 class="agc-hero__t">{{ t('agent.code.new.title') }}</h1>
      <div v-if="!readyRuntimes.length && optionsLoaded" class="agc-hero__sub">
        {{ t('agent.code.new.noAgent') }}
        <RouterLink v-if="canRuntimes" to="/agent/admin?tab=runtimes">{{ t('agent.code.new.goRuntimes') }}</RouterLink>
      </div>
      <div v-else-if="summary" class="agc-hero__sub">
        <i18n-t scope="global" keypath="agent.code.new.defaults" tag="span">
          <template #summary><b>{{ summary }}</b></template>
          <template #where>{{ whereLine }}</template>
        </i18n-t>
      </div>
    </div>

    <div v-show="!ctab" class="agc-foot">
      <Composer
        ref="composer"
        :runtimes="options"
        :runtime-id="sel.runtimeId"
        :provider-id="sel.providerId"
        :model="sel.model"
        :effort="sel.effort"
        :mode="sel.mode"
        agent-switch
        :workspace-id="workspaceId"
        :skills="skills"
        :rows="3"
        :label="t('agent.code.new.inputLabel')"
        :placeholder="t('agent.code.new.placeholder')"
        :on-send="create"
        @select="applySelect"
      />
    </div>
  </section>

  <div
    v-if="rightOn"
    class="agc-grip"
    role="separator"
    aria-orientation="vertical"
    tabindex="0"
    :aria-valuenow="rightWidth"
    :aria-label="t('agent.code.resizeRight')"
    :title="t('agent.code.resizeRightTip')"
    @pointerdown="startRightResize"
    @dblclick="resetRightWidth"
    @keydown="onRightGripKey"
  />
  <section v-if="rightOn" class="agc-card agc-right agc-right--new ag-noscroll" :style="rightStyle(false)" :aria-label="t('agent.code.new.settings')">
    <div class="agc-new__head">
      <h2 class="agc-new__t">{{ t('agent.code.new.settings') }}</h2>
      <span class="ag-muted agc-small">{{ t('agent.code.new.fromWorkspace') }}</span>
    </div>

    <div class="agc-new__sec">
      <div class="ag-lbl">{{ t('agent.code.new.workspace') }}</div>
      <div ref="wsBox" class="agc-rel">
        <button type="button" class="ag-opt agc-wsopt" aria-haspopup="listbox" :aria-expanded="wsOpen" @click="wsOpen = !wsOpen">
          <span class="agc-wsopt__ic"><UIcon name="i-lucide-folder" /></span>
          <span class="agc-wsopt__t">
            <span class="agc-wsopt__n">{{ workspace?.name || t('agent.common.noWorkspace') }}</span>
            <span class="ag-mono ag-muted ag-ellipsis agc-small">{{ workspace ? wsSub(workspace) : t('agent.code.new.scratch') }}</span>
          </span>
          <UIcon name="i-lucide-chevron-down" class="ag-muted" />
        </button>
        <div v-if="wsOpen" class="ag-menu agc-drop ag-noscroll" role="listbox" :aria-label="t('agent.code.new.pickWorkspace')">
          <button v-for="w in workspaces" :key="w.id" type="button" class="ag-mi agc-drop__item" :class="{ 'is-on': w.id === workspaceId }" role="option" :aria-selected="w.id === workspaceId" :disabled="!!w.cloneStatus" @click="pickWorkspace(w.id)">
            <UIcon name="i-lucide-folder" class="ag-muted" />
            <span class="agc-wsopt__t">
              <span class="agc-wsopt__n">{{ w.name }}</span>
              <span class="ag-mono ag-muted ag-ellipsis agc-small">{{ w.cloneStatus ? t(`agent.code.new.clone.${w.cloneStatus}`) : wsSub(w) }}</span>
            </span>
            <UIcon v-if="w.id === workspaceId" name="i-lucide-check" class="agc-on" />
          </button>
          <button type="button" class="ag-mi agc-drop__item" :class="{ 'is-on': !workspaceId }" role="option" :aria-selected="!workspaceId" @click="pickWorkspace('')">
            <UIcon name="i-lucide-message-square" class="ag-muted" />
            <span class="agc-wsopt__t">
              <span class="agc-wsopt__n">{{ t('agent.common.noWorkspace') }}</span>
              <span class="ag-muted agc-small">{{ t('agent.code.new.scratch') }}</span>
            </span>
            <UIcon v-if="!workspaceId" name="i-lucide-check" class="agc-on" />
          </button>
          <div class="ag-mi-sep" />
          <button type="button" class="ag-mi" @click="addWsOpen = true, (wsOpen = false)"><UIcon name="i-lucide-plus" class="ag-muted" />{{ t('agent.code.new.addWorkspace') }}</button>
        </div>
      </div>

      <div v-for="(d, i) in extraDirs" :key="d" class="agc-xdir">
        <UIcon name="i-lucide-folder" class="ag-muted" />
        <span class="ag-mono ag-ellipsis agc-flex">{{ d }}</span>
        <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.code.new.removeDir', { path: d })" :title="t('agent.common.remove')" @click="extraDirs.splice(i, 1)">
          <UIcon name="i-lucide-x" />
        </button>
      </div>
      <div v-if="canExtraDirs" ref="xdBox" class="agc-rel">
        <button type="button" class="ag-btn ag-btn--sm" aria-haspopup="dialog" :aria-expanded="xdOpen" @click="xdOpen = !xdOpen">
          <UIcon name="i-lucide-plus" />{{ t('agent.code.new.addDir') }}
        </button>
        <div v-if="xdOpen" class="ag-menu agc-drop agc-xdpop" role="dialog" :aria-label="t('agent.code.new.addDir')">
          <div class="agc-xdpop__row">
            <span class="ag-fld agc-flex"><input v-model="xdDraft" class="ag-mono" type="text" :placeholder="t('agent.code.new.dirPlaceholder')" :aria-label="t('agent.code.new.dirLabel')" @keydown.enter="addDir" /></span>
            <button type="button" class="ag-btn ag-btn--p ag-btn--sm" :disabled="!xdDraft.trim()" @click="addDir">{{ t('agent.common.add') }}</button>
          </div>
          <AgDirList v-model="xdDraft" class="agc-xdpop__list" />
        </div>
      </div>
    </div>

    <label v-if="workspace?.isGit" class="agc-toggle">
      <span class="agc-toggle__t">
        <span class="agc-toggle__n">{{ t('agent.code.new.worktree') }}</span>
        <span class="ag-muted agc-small">{{ t('agent.code.new.worktreeHint') }}</span>
      </span>
      <AgSwitch v-model="worktree" :label="t('agent.code.new.worktree')" />
    </label>

    <div class="agc-new__sec">
      <div class="ag-lbl">{{ t('agent.code.new.mcp') }}</div>
      <label v-for="m in mcpServers" :key="m.id" class="agc-mcprow">
        <UIcon name="i-lucide-plug" class="ag-muted" />
        <span class="ag-mono ag-ellipsis agc-flex">{{ m.name }}</span>
        <span v-if="m.transport === 'http' && !caps?.mcpHttp" class="ag-muted agc-small">{{ t('agent.code.new.httpUnsupported') }}</span>
        <AgSwitch :model-value="mcpIds.has(m.id)" :disabled="m.transport === 'http' && !caps?.mcpHttp" :label="t('agent.code.new.mcpOn', { name: m.name })" @update:model-value="toggleMcp(m.id, $event)" />
      </label>
      <div v-if="mcpLoaded && !mcpServers.length" class="ag-muted agc-small agc-new__none">
        {{ t('agent.code.new.noMcp') }}
        <RouterLink v-if="canManage" to="/agent/manage?tab=mcp">{{ t('agent.code.new.goMcp') }}</RouterLink>
      </div>
    </div>

    <!-- 选了工作区：下面是它的文件夹，点文件或拖到输入框里就是引用 -->
    <div v-if="workspace" class="agc-new__sec agc-new__files">
      <div class="ag-lbl">{{ t('agent.code.new.files') }}</div>
      <FileTree
        :scope="{ workspaceId: workspace.id }"
        :cwd="workspace.path"
        :empty="t('agent.code.files.empty')"
        :tip="(path) => t('agent.code.new.filesTip', { path })"
        :active="ctab"
        can-ref
        @open="openFile"
        @moved="onFileMoved"
        @deleted="onFilesDeleted"
        @reference="referenceFile"
      />
    </div>
  </section>

  <AgWorkspaceDialog v-model="addWsOpen" @saved="onWorkspaceAdded" />
  <AgConfirm
    v-model="closeAskOpen"
    :title="t('agent.code.file.closeDirtyTitle')"
    :message="t('agent.code.file.closeDirtyMsg', { names: shortName(closeAsk) })"
    :confirm-text="t('agent.code.file.closeDirty')"
    danger
    :on-confirm="() => closeNow(closeAsk)"
  />
</template>

<script setup lang="ts">
import { onClickOutside } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { createAgentSession, listAgentMcpServers, listAgentSkills, listAgentWorkspaces } from '@/api/agent'
import { useMenuStore } from '@/stores/menu'
import type { AgentMcpServer, AgentRuntimeOptions, AgentSession, AgentWorkspace } from '@/types/v1/agent'
import AgConfirm from '@/views/agent/components/AgConfirm.vue'
import AgDirList from '@/views/agent/components/AgDirList.vue'
import AgSwitch from '@/views/agent/components/AgSwitch.vue'
import AgWorkspaceDialog from '@/views/agent/components/AgWorkspaceDialog.vue'
import { modelLabel } from '@/views/agent/composables/useAgentModels'
import { providerLabel } from '@/views/agent/utils/labels'
import Composer, { type DraftAttachment, type DraftSnippet, type Selection } from './Composer.vue'
import FileTab from './FileTab.vue'
import FileTree from './FileTree.vue'
import { onRightGripKey, resetRightWidth, rightStyle, rightWidth, startRightResize } from './rightWidth'
import { dropFileState, fileScope, isDirty, isUnder, movedPath, moveFileState, useTabKeys } from './fileState'

const props = defineProps<{
  options: AgentRuntimeOptions[]
  optionsLoaded: boolean
  /** 从会话栏某个工作区的「+」进来时带着它 */
  initialWorkspace: string
  leftOn: boolean
  rightOn: boolean
  /** Agent 配置（工作区、MCP、技能、供应商）变了就加一：重读这一页用到的列表 */
  configVersion: number
}>()
const emit = defineEmits<{ created: [session: AgentSession]; toggleLeft: []; toggleRight: []; workspacesChanged: [] }>()
const { t } = useI18n()
const menuStore = useMenuStore()
// 能进 Agent 设置页（自己的 MCP 在那里添加）
const canManage = computed(() => menuStore.hasMenuPath('/agent/manage'))
// 能管大家共用的运行时（Agent 管理页里的这个标签要 agent:manage）
const canRuntimes = computed(() => menuStore.hasMenuPath('/agent/admin') && menuStore.hasButtonPermission('agent:manage'))

const LAST_KEY = 'agent.code.last'
interface Last extends Selection {
  workspaceId: string
}
const readLast = (): Partial<Last> => {
  try {
    return JSON.parse(localStorage.getItem(LAST_KEY) || '{}') as Partial<Last>
  } catch {
    return {}
  }
}

const composer = ref<InstanceType<typeof Composer> | null>(null)
const workspaces = ref<AgentWorkspace[]>([])
const workspaceId = ref('')
const workspace = computed(() => workspaces.value.find((w) => w.id === workspaceId.value) ?? null)
const sel = reactive<Selection>({ runtimeId: '', providerId: '', model: '', effort: '', mode: '' })
const extraDirs = ref<string[]>([])
const worktree = ref(false)
const mcpServers = ref<AgentMcpServer[]>([])
const mcpLoaded = ref(false)
const mcpIds = reactive(new Set<string>())
const skills = ref<string[]>([])

const readyRuntimes = computed(() => props.options.filter((r) => r.ready))
const runtime = computed(() => props.options.find((r) => r.runtimeId === sel.runtimeId))
const caps = computed(() => runtime.value?.caps)
const canExtraDirs = computed(() => !!caps.value?.additionalDirs)

const wsSub = (w: AgentWorkspace) => [w.path, w.isGit ? w.branch : ''].filter(Boolean).join(' · ')

// —— 默认值 ——
/** 在某个 Agent 下挑一个能用的选择：想要的有就用想要的，没有就用默认的 */
const resolve = (want: Partial<Selection>): Selection => {
  const rt = props.options.find((r) => r.runtimeId === want.runtimeId && r.ready) ?? readyRuntimes.value[0]
  if (!rt) return { runtimeId: '', providerId: '', model: '', effort: '', mode: '' }
  const p =
    rt.providers.find((x) => x.providerId === (want.providerId ?? '') && x.models.some((m) => m.id === want.model)) ??
    rt.providers.find((x) => x.providerId === (want.providerId ?? '')) ??
    rt.providers[0]
  const m = p?.models.find((x) => x.id === want.model) ?? p?.models.find((x) => x.id === p.defaultModel) ?? p?.models[0]
  const effort = m?.efforts.some((e) => e.id === want.effort) ? (want.effort as string) : (m?.defaultEffort ?? '')
  const mode = rt.modes.some((x) => x.id === want.mode) ? (want.mode as string) : rt.defaultMode
  return { runtimeId: rt.runtimeId, providerId: p?.providerId ?? '', model: m?.id ?? want.model ?? '', effort, mode }
}

const applyDefaults = () => {
  const w = workspace.value
  const last = readLast()
  const want: Partial<Selection> = w?.runtimeId
    ? { runtimeId: w.runtimeId, providerId: w.providerId, model: w.model, effort: w.effort, mode: w.mode }
    : last
  Object.assign(sel, resolve(want))
  mcpIds.clear()
  const ids = w?.mcpIds?.length ? w.mcpIds : mcpServers.value.filter((m) => m.defaultOn).map((m) => m.id)
  for (const id of ids) if (mcpServers.value.some((m) => m.id === id)) mcpIds.add(id)
  worktree.value = false
}

const applySelect = (patch: Partial<Selection>) => {
  Object.assign(sel, patch.runtimeId && patch.runtimeId !== sel.runtimeId ? resolve({ ...patch }) : patch)
}

const pickWorkspace = (id: string) => {
  wsOpen.value = false
  workspaceId.value = id
  applyDefaults()
}

const summary = computed(() => {
  if (!runtime.value) return ''
  const p = runtime.value.providers.find((x) => x.providerId === sel.providerId)
  const m = p?.models.find((x) => x.id === sel.model)
  const effort = m?.efforts.find((e) => e.id === sel.effort)
  return [runtime.value.runtimeName, providerLabel(p), modelLabel(m, sel.model), effort?.name || ''].filter(Boolean).join(' · ')
})
const whereLine = computed(() => {
  const w = workspace.value
  if (!w) return t('agent.code.new.whereNone')
  return w.isGit ? t('agent.code.new.whereBranch', { name: w.name, branch: w.branch }) : t('agent.code.new.where', { name: w.name })
})

// —— 工作区下拉、附加文件夹 ——
const wsOpen = ref(false)
const wsBox = ref<HTMLElement | null>(null)
onClickOutside(wsBox, () => (wsOpen.value = false))
const addWsOpen = ref(false)
const onWorkspaceAdded = async (w: AgentWorkspace) => {
  await loadWorkspaces()
  emit('workspacesChanged')
  pickWorkspace(w.id)
}

const xdOpen = ref(false)
const xdDraft = ref('')
const xdBox = ref<HTMLElement | null>(null)
onClickOutside(xdBox, () => (xdOpen.value = false))
const addDir = () => {
  const p = xdDraft.value.trim()
  if (!p) return
  if (!extraDirs.value.includes(p)) extraDirs.value.push(p)
  xdDraft.value = ''
  xdOpen.value = false
}

const toggleMcp = (id: string, on: boolean) => (on ? mcpIds.add(id) : mcpIds.delete(id))

// —— 工作区的文件：点开在标签页里看 ——
/** 打开的文件（相对工作区）；ctab 为空是新会话本身 */
const openFiles = ref<string[]>([])
const ctab = ref('')
const tabKeys = useTabKeys()
const openFile = (path: string) => {
  if (!openFiles.value.includes(path)) {
    openFiles.value.push(path)
    tabKeys.fresh(path)
  }
  ctab.value = path
}
// 文件标签的 HTML 预览里跳到了别的页面：标签就地换成那个文件（已经开着那个文件的标签合并掉）
const onTabNavigate = (from: string, to: string) => {
  openFiles.value = openFiles.value.filter((p) => p !== to).map((p) => (p === from ? to : p))
  tabKeys.carry(from, to)
  if (ctab.value === from) ctab.value = to
}
const shortName = (p: string) => p.split('/').slice(-2).join('/')
const scopeKey = computed(() => fileScope('', workspaceId.value))
const tabDirty = (p: string) => isDirty(scopeKey.value, p)
/** 关掉一个标签（没保存的改动丢掉），旁边的顶上来 */
const closeNow = (path: string) => {
  dropFileState(scopeKey.value, [path])
  tabKeys.drop(path)
  const i = openFiles.value.indexOf(path)
  if (i < 0) return
  openFiles.value.splice(i, 1)
  if (ctab.value === path) ctab.value = openFiles.value[i] ?? openFiles.value[i - 1] ?? ''
}
const closeAsk = ref('')
const closeAskOpen = ref(false)
const closeFile = (path: string) => {
  if (!tabDirty(path)) {
    closeNow(path)
    return
  }
  closeAsk.value = path
  closeAskOpen.value = true
}
// 文件树里改名、挪位置、删除：打开的标签跟着变
const onFileMoved = (from: string, to: string) => {
  moveFileState(scopeKey.value, from, to)
  openFiles.value = openFiles.value.map((p) => {
    const next = movedPath(p, from, to)
    if (next !== p) {
      tabKeys.drop(p)
      tabKeys.fresh(next)
    }
    return next
  })
  if (ctab.value) ctab.value = movedPath(ctab.value, from, to)
}
const onFilesDeleted = (paths: string[]) => {
  for (const p of openFiles.value.filter((p) => paths.some((d) => isUnder(p, d)))) closeNow(p)
  dropFileState(scopeKey.value, paths)
}
// 右键「添加到对话」：回到新会话，加一个引用
const referenceFile = async (path: string) => {
  ctab.value = ''
  await nextTick()
  composer.value?.addRef(path)
  composer.value?.focus()
}
// 换了工作区：之前打开的文件是那个工作区的，都关掉（没保存的改动也丢掉）
watch(workspaceId, (_, old) => {
  dropFileState(fileScope('', old), openFiles.value)
  openFiles.value = []
  ctab.value = ''
})
// 文件里选中一段「添加到对话」：回到新会话，加进输入框的附件
const attachSnippet = async (sn: DraftSnippet) => {
  ctab.value = ''
  await nextTick()
  composer.value?.addSnippet(sn)
}

// —— 新建 ——
const create = async (text: string, attachments: DraftAttachment[]): Promise<boolean> => {
  if (!sel.runtimeId) {
    feedback.warning(t('agent.code.new.noAgent'))
    return false
  }
  try {
    const { data } = await createAgentSession({
      workspaceId: workspaceId.value,
      extraDirs: canExtraDirs.value ? extraDirs.value : [],
      worktree: !!workspace.value?.isGit && worktree.value,
      runtimeId: sel.runtimeId,
      providerId: sel.providerId,
      model: sel.model,
      effort: sel.effort,
      mode: sel.mode,
      mcpIds: [...mcpIds],
      text,
      attachments,
    })
    localStorage.setItem(LAST_KEY, JSON.stringify({ ...sel, workspaceId: workspaceId.value }))
    if (data.session) emit('created', data.session)
    return true
  } catch {
    return false
  }
}

// —— 加载 ——
const loadWorkspaces = async () => {
  try {
    const { data } = await listAgentWorkspaces()
    workspaces.value = data.workspaces ?? []
  } catch {
    /* shown */
  }
}
const loadLists = () =>
  Promise.all([
    loadWorkspaces(),
    listAgentMcpServers()
      .then(({ data }) => (mcpServers.value = data.servers ?? []))
      .catch(() => undefined)
      .finally(() => (mcpLoaded.value = true)),
    listAgentSkills({ enabled: true })
      .then(({ data }) => (skills.value = (data.skills ?? []).map((s) => s.name)))
      .catch(() => undefined),
  ])
const init = async () => {
  await loadLists()
  const want = props.initialWorkspace || readLast().workspaceId || ''
  workspaceId.value = workspaces.value.some((w) => w.id === want && !w.cloneStatus) ? want : ''
  applyDefaults()
  void nextTick(() => composer.value?.focus())
}

watch(
  () => props.initialWorkspace,
  (id) => {
    if (id && workspaces.value.some((w) => w.id === id)) pickWorkspace(id)
  },
)
// 选项（Agent、供应商）晚一步到：到了再定默认值
watch(
  () => props.optionsLoaded,
  (ok) => {
    if (ok && !sel.runtimeId) applyDefaults()
  },
)
// 管理页里改了配置：重读列表；选中的 Agent 或供应商不再可选时（比如服务器上的登录失效、供应商被删）换回默认的
watch(
  () => props.configVersion,
  () => void loadLists(),
)
watch(
  () => props.options,
  (options) => {
    const rt = options.find((r) => r.runtimeId === sel.runtimeId)
    if (sel.runtimeId && !rt?.providers.some((p) => (p.providerId ?? '') === sel.providerId)) applyDefaults()
  },
)
onMounted(() => void init())
</script>
