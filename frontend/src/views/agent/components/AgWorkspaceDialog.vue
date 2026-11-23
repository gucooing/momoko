<!-- 新建 / 编辑工作区的弹窗。workspace 为空 = 新建。
     目录从哪来（已有目录 / 从 Git 克隆，只在新建时选）、名称、目录、默认模型（连带确定 Agent 和供应商）、
     默认推理强度、默认权限模式、默认开启的 MCP、项目指令（AGENTS.md / CLAUDE.md，直接读写目录里的文件）。 -->
<template>
  <AgDialog
    :model-value="modelValue"
    :title="isEdit ? t('agent.workspaces.dialog.editTitle') : t('agent.workspaces.dialog.createTitle')"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="ag-ws-form">
      <div v-if="!isEdit" class="ag-field">
        <span class="ag-fl">{{ t('agent.workspaces.dialog.source') }}</span>
        <div class="ag-seg" role="radiogroup" :aria-label="t('agent.workspaces.dialog.source')">
          <button
            v-for="s in sources"
            :key="s.value"
            type="button"
            role="radio"
            :class="{ 'is-on': form.source === s.value }"
            :aria-checked="form.source === s.value"
            @click="setSource(s.value)"
          >
            {{ s.label }}
          </button>
        </div>
      </div>

      <template v-if="!isGit">
        <div class="ag-ws-form__grid ag-ws-form__grid--dir ag-grid-2">
          <label class="ag-field">
            <span class="ag-fl">{{ t('agent.common.name') }}</span>
            <span class="ag-fld">
              <input v-model="form.name" type="text" :placeholder="t('agent.workspaces.dialog.namePlaceholder')" />
            </span>
          </label>
          <div class="ag-field">
            <label class="ag-fl" :for="dirId">{{ t('agent.workspaces.dialog.dir') }}</label>
            <span class="ag-fld ag-ws-form__dir">
              <UIcon name="i-lucide-folder" />
              <input
                :id="dirId"
                v-model="form.path"
                class="ag-mono"
                type="text"
                autocomplete="off"
                spellcheck="false"
                :placeholder="t('agent.workspaces.dialog.dirPlaceholder')"
              />
              <button
                type="button"
                class="ag-btn ag-btn--sm ag-btn--plain"
                :aria-expanded="browseOpen"
                @click="browseOpen = !browseOpen"
              >
                {{ t('agent.common.browse') }}
              </button>
            </span>
          </div>
        </div>
        <AgDirList v-if="browseOpen" :model-value="form.path" @update:model-value="onPickDir" @loaded="onDirsLoaded" />
      </template>

      <template v-else>
        <div class="ag-field">
          <label class="ag-fl" :for="gitId">{{ t('agent.workspaces.dialog.gitUrl') }}</label>
          <span class="ag-fld">
            <input
              :id="gitId"
              v-model="form.gitUrl"
              class="ag-mono"
              type="text"
              autocomplete="off"
              spellcheck="false"
              :placeholder="t('agent.workspaces.dialog.gitUrlPlaceholder')"
            />
          </span>
        </div>
        <div class="ag-ws-form__grid ag-ws-form__grid--git ag-grid-3">
          <label class="ag-field">
            <span class="ag-fl">{{ t('agent.common.name') }}</span>
            <span class="ag-fld">
              <input v-model="form.name" type="text" :placeholder="t('agent.workspaces.dialog.gitNamePlaceholder')" />
            </span>
          </label>
          <label class="ag-field">
            <span class="ag-fl">{{ t('agent.workspaces.dialog.branch') }}</span>
            <span class="ag-fld">
              <input
                v-model="form.gitBranch"
                class="ag-mono"
                type="text"
                autocomplete="off"
                spellcheck="false"
                :placeholder="t('agent.workspaces.dialog.branchPlaceholder')"
              />
            </span>
          </label>
          <label class="ag-field">
            <span class="ag-fl">{{ t('agent.workspaces.dialog.cloneTo') }}</span>
            <span class="ag-fld">
              <input
                v-model="form.clonePath"
                class="ag-mono"
                type="text"
                autocomplete="off"
                spellcheck="false"
                :placeholder="clonePlaceholder"
              />
            </span>
          </label>
        </div>
        <span class="ag-fh">{{ t('agent.workspaces.dialog.gitHint') }}</span>
      </template>

      <div class="ag-ws-form__grid ag-ws-form__grid--model ag-grid-3">
        <div class="ag-field">
          <span class="ag-fl ag-ws-form__fl">
            {{ t('agent.workspaces.dialog.model') }}
            <span v-if="runtimeName" class="ag-ws-form__by">
              {{ [t('agent.workspaces.dialog.runBy', { name: runtimeName }), providerName].filter(Boolean).join(' · ') }}
            </span>
          </span>
          <AgSelect
            :model-value="currentKey"
            :options="modelOptions"
            :disabled="!modelOptions.length"
            :placeholder="modelPlaceholder"
            :label="t('agent.workspaces.dialog.model')"
            @update:model-value="pickModel"
          />
        </div>
        <div class="ag-field">
          <span class="ag-fl">{{ t('agent.workspaces.dialog.effort') }}</span>
          <AgSelect
            v-model="form.effort"
            :options="effortOptions"
            :disabled="!effortOptions.length"
            :placeholder="effortPlaceholder"
            :label="t('agent.workspaces.dialog.effort')"
          />
        </div>
        <div class="ag-field">
          <span class="ag-fl">{{ t('agent.workspaces.dialog.mode') }}</span>
          <AgSelect
            v-model="form.mode"
            :options="modeOptions"
            :disabled="!modeOptions.length"
            :placeholder="modePlaceholder"
            :label="t('agent.workspaces.dialog.mode')"
          />
        </div>
      </div>

      <div class="ag-field">
        <span class="ag-fl">{{ t('agent.workspaces.dialog.mcp') }}</span>
        <div v-if="servers.length" class="ag-ws-form__chips">
          <button
            v-for="s in servers"
            :key="s.id"
            type="button"
            class="ag-chip"
            :class="{ 'is-on': form.mcpIds.includes(s.id) }"
            :aria-pressed="form.mcpIds.includes(s.id)"
            @click="toggleMcp(s.id)"
          >
            <UIcon name="i-lucide-plug" />
            {{ s.name }}
          </button>
        </div>
        <span v-else class="ag-fh">
          {{ serversState === 'loading' ? t('agent.common.loading') : t('agent.workspaces.dialog.noMcp') }}
        </span>
      </div>

      <div class="ag-field">
        <span class="ag-fl">{{ t('agent.workspaces.dialog.docs') }}</span>
        <div class="ag-ws-form__grid ag-ws-form__grid--docs ag-grid-2">
          <div v-for="d in docs" :key="d.name" class="ag-ws-doc">
            <span class="ag-mono ag-ellipsis ag-ws-doc__name">{{ d.name }}</span>
            <span class="ag-pill" :class="{ 'ag-pill--ok': d.exists }">
              {{ d.exists ? t('agent.workspaces.dialog.docExists') : t('agent.workspaces.dialog.docMissing') }}
            </span>
            <button type="button" class="ag-btn ag-btn--sm" @click="docAct(d)">
              {{
                !d.exists
                  ? t('agent.workspaces.dialog.docCreate')
                  : editing === d.name
                    ? t('agent.workspaces.dialog.docCollapse')
                    : t('agent.common.edit')
              }}
            </button>
          </div>
        </div>
        <div v-if="editingDoc" class="ag-ws-editor">
          <div class="ag-ws-editor__head">
            <span class="ag-mono ag-ws-editor__name">{{ editingDoc.name }}</span>
            <button type="button" class="ag-btn ag-btn--sm ag-btn--plain" @click="editing = ''">
              {{ t('agent.workspaces.dialog.docCollapse') }}
            </button>
          </div>
          <textarea
            class="ag-mono ag-ws-editor__ta"
            rows="6"
            spellcheck="false"
            :value="editingDoc.content"
            :aria-label="t('agent.workspaces.dialog.docContent', { name: editingDoc.name })"
            @input="onDocInput"
          />
        </div>
        <span class="ag-fh">{{ t('agent.workspaces.dialog.docsHint') }}</span>
      </div>
    </div>

    <template #footer>
      <button type="button" class="ag-btn" @click="emit('update:modelValue', false)">
        {{ t('agent.common.cancel') }}
      </button>
      <button type="button" class="ag-btn ag-btn--p" :disabled="saving" @click="save">
        <UIcon v-if="saving" name="i-lucide-loader-circle" class="ag-spin" />
        {{ t('agent.common.save') }}
      </button>
    </template>
  </AgDialog>
</template>

<script setup lang="ts">
import { modelLabel, useAgentModels } from '@/views/agent/composables/useAgentModels'
import { modeName, providerLabel } from '@/views/agent/utils/labels'
import { useDebounceFn } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import {
  createAgentWorkspace,
  getAgentDocs,
  listAgentDirs,
  listAgentMcpServers,
  updateAgentWorkspace,
} from '@/api/agent'
import type {
  AgentDoc,
  AgentMcpServer,
  AgentModelOption,
  AgentProviderOptions,
  AgentRuntimeOptions,
  AgentWorkspace,
  ListAgentDirsResponse,
} from '@/types/v1/agent'
import AgDialog from './AgDialog.vue'
import AgDirList from './AgDirList.vue'
import AgSelect from './AgSelect.vue'

type Source = 'dir' | 'git'

interface ModelChoice {
  key: string
  runtime: AgentRuntimeOptions
  provider: AgentProviderOptions
  model: AgentModelOption
}

/** 项目指令文件在弹窗里的状态：created = 点了「创建」，edited = 改过内容；保存时只写这两种 */
interface DocState {
  name: string
  exists: boolean
  content: string
  created: boolean
  edited: boolean
}

const props = withDefaults(defineProps<{ modelValue: boolean; workspace?: AgentWorkspace | null }>(), {
  workspace: null,
})
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; saved: [workspace: AgentWorkspace] }>()
const { t } = useI18n()

const DOC_NAMES = ['AGENTS.md', 'CLAUDE.md']
const blankDocs = (): DocState[] =>
  DOC_NAMES.map((name) => ({ name, exists: false, content: '', created: false, edited: false }))

const dirId = useId()
const gitId = useId()

const form = reactive({
  source: 'dir' as Source,
  name: '',
  path: '',
  gitUrl: '',
  gitBranch: '',
  clonePath: '',
  runtimeId: '',
  providerId: '',
  model: '',
  effort: '',
  mode: '',
  mcpIds: [] as string[],
})
const docs = ref<DocState[]>(blankDocs())
const editing = ref('')
const browseOpen = ref(false)
const saving = ref(false)

// 模型清单和别的页面共用一份（模型供应商页同步、配置的），每次打开弹窗重读一次
const { runtimes, refresh: refreshModels } = useAgentModels()
const optionsLoaded = ref(false)
const servers = ref<AgentMcpServer[]>([])
const serversState = ref<'loading' | 'ok' | 'failed'>('loading')
const cloneBase = ref('')

// 浏览时自动带出的名称；用户没自己改过就跟着所选目录变
let autoName = ''
let mcpTouched = false
let openSeq = 0
let docSeq = 0
let docsPath = ''

const isEdit = computed(() => !!props.workspace)
const isGit = computed(() => !isEdit.value && form.source === 'git')
const sources = computed<{ value: Source; label: string }[]>(() => [
  { value: 'dir', label: t('agent.workspaces.dialog.sourceDir') },
  { value: 'git', label: t('agent.workspaces.dialog.sourceGit') },
])

// —— 默认模型 / 推理强度 / 权限模式 ——
const modelKey = (runtimeId: string, providerId: string, model: string) =>
  JSON.stringify([runtimeId, providerId, model])

const choices = computed<ModelChoice[]>(() =>
  runtimes.value
    .filter((rt) => rt.ready)
    .flatMap((rt) =>
      (rt.providers ?? []).flatMap((p) =>
        (p.models ?? []).map((m) => ({
          key: modelKey(rt.runtimeId, p.providerId, m.id),
          runtime: rt,
          provider: p,
          model: m,
        })),
      ),
    ),
)
const currentKey = computed(() => (form.model ? modelKey(form.runtimeId, form.providerId, form.model) : ''))
const currentChoice = computed(() => choices.value.find((c) => c.key === currentKey.value))
// 之前选的模型现在选不到了（供应商被删、运行时没就绪）：照原样显示，不替用户改掉
const stale = computed(() => optionsLoaded.value && !!currentKey.value && !currentChoice.value)

// 选项就是模型供应商里那份清单：按 Agent · 供应商分组，每一项只是模型名，和「编辑模型」里列的一样
const savedProvider = computed(() => {
  const w = props.workspace
  return w ? providerLabel({ kind: w.providerKind, vendor: w.providerVendor, name: w.providerName }) : ''
})
const modelOptions = computed(() => {
  const list: { value: string; label: string; group?: string }[] = choices.value.map((c) => ({
    value: c.key,
    label: modelLabel(c.model),
    group: `${c.runtime.runtimeName} · ${providerLabel(c.provider)}`,
  }))
  if (stale.value) {
    list.unshift({
      value: currentKey.value,
      label: form.model,
      group: [props.workspace?.runtimeName, savedProvider.value].filter(Boolean).join(' · ') || undefined,
    })
  }
  return list
})
const effortOptions = computed(() =>
  (currentChoice.value?.model.efforts ?? []).map((e) => ({ value: e.id, label: e.name || e.id })),
)
const modeOptions = computed(() =>
  (currentChoice.value?.runtime.modes ?? []).map((m) => ({ value: m.id, label: modeName(m.id, m.name) })),
)

const runtimeName = computed(
  () => currentChoice.value?.runtime.runtimeName || (stale.value ? props.workspace?.runtimeName : '') || '',
)
const providerName = computed(() =>
  currentChoice.value ? providerLabel(currentChoice.value.provider) : stale.value ? savedProvider.value : '',
)
const modelPlaceholder = computed(() => {
  if (!optionsLoaded.value) return t('agent.common.loading')
  return modelOptions.value.length ? t('agent.workspaces.dialog.pickModel') : t('agent.workspaces.dialog.noOptions')
})
const effortPlaceholder = computed(() => {
  if (effortOptions.value.length) return '—'
  return (stale.value && form.effort) || t('agent.common.notAdjustable')
})
const modePlaceholder = computed(() => {
  if (modeOptions.value.length) return '—'
  return (stale.value && form.mode && modeName(form.mode, props.workspace?.modeName)) || '—'
})

const pickModel = (key: string) => {
  const c = choices.value.find((x) => x.key === key)
  if (!c) return
  form.runtimeId = c.runtime.runtimeId
  form.providerId = c.provider.providerId
  form.model = c.model.id
  const efforts = c.model.efforts ?? []
  form.effort = efforts.length ? c.model.defaultEffort || (efforts[0]?.id ?? '') : ''
  const modes = c.runtime.modes ?? []
  if (!modes.some((m) => m.id === form.mode)) form.mode = c.runtime.defaultMode || (modes[0]?.id ?? '')
}

// 新建时先带上第一个可用供应商的默认模型，要改再选
const applyDefaultModel = () => {
  if (isEdit.value || form.model) return
  for (const rt of runtimes.value) {
    if (!rt.ready) continue
    for (const p of rt.providers ?? []) {
      const models = p.models ?? []
      const m = models.find((x) => x.id === p.defaultModel) ?? models[0]
      if (m) {
        pickModel(modelKey(rt.runtimeId, p.providerId, m.id))
        return
      }
    }
  }
}

// —— MCP ——
const toggleMcp = (id: string) => {
  mcpTouched = true
  form.mcpIds = form.mcpIds.includes(id) ? form.mcpIds.filter((x) => x !== id) : [...form.mcpIds, id]
}

// —— 目录 ——
const setSource = (source: Source) => {
  form.source = source
  browseOpen.value = false
  if (source === 'git' && !cloneBase.value) {
    listAgentDirs('')
      .then(({ data }) => onDirsLoaded(data))
      .catch(() => {})
  }
}
const onDirsLoaded = (data: ListAgentDirsResponse) => {
  if (data.cloneBase) cloneBase.value = data.cloneBase
}
const onPickDir = (path: string) => {
  form.path = path
  const leaf = path.split(/[\\/]/).filter(Boolean).pop() ?? ''
  if (leaf && (!form.name.trim() || form.name === autoName)) {
    form.name = leaf
    autoName = leaf
  }
}

const repoName = computed(() => {
  const m = form.gitUrl
    .trim()
    .replace(/[\\/]+$/, '')
    .match(/([^/\\:]+?)(?:\.git)?$/)
  return m?.[1] ?? ''
})
const clonePlaceholder = computed(() => {
  const leaf = form.name.trim() || repoName.value || t('agent.workspaces.dialog.repoName')
  const base = cloneBase.value
  if (!base) return leaf
  const sep = base.includes('\\') && !base.includes('/') ? '\\' : '/'
  return `${base.replace(/[\\/]+$/, '')}${sep}${leaf}`
})

// —— 项目指令 ——
const editingDoc = computed(() => docs.value.find((d) => d.name === editing.value))

// 目录里的实际情况盖到列表上；用户改过的内容留着，点了「创建」而目录里确实没有的也留着
const applyDocs = (list: AgentDoc[]) => {
  docs.value = docs.value.map((d) => {
    const s = list.find((x) => x.name === d.name)
    if (d.edited || (d.created && !s?.exists)) return d
    return { name: d.name, exists: !!s?.exists, content: s?.content ?? '', created: false, edited: false }
  })
  if (editing.value && !editingDoc.value?.exists) editing.value = ''
}
const loadDocs = async (path: string) => {
  const id = ++docSeq
  docsPath = path
  try {
    const { data } = await getAgentDocs(path)
    if (id === docSeq) applyDocs(data.docs ?? [])
  } catch {
    // 请求层已经提示过
  }
}
const loadDocsLater = useDebounceFn((path: string) => {
  if (!props.modelValue || isGit.value || form.path.trim() !== path || path === docsPath) return
  void loadDocs(path)
}, 500)

watch(
  () => [isGit.value, form.path] as const,
  ([git, path]) => {
    if (!props.modelValue) return
    const p = path.trim()
    if (git || !p) {
      // 克隆之前目录还不存在，两个文件都算没有
      docSeq += 1
      docsPath = ''
      applyDocs([])
      return
    }
    if (p !== docsPath) void loadDocsLater(p)
  },
)

const docDefault = (name: string) =>
  name === 'CLAUDE.md' ? '@AGENTS.md' : `# ${form.name.trim() || t('agent.workspaces.dialog.projectName')}\n\n`
const docAct = (d: DocState) => {
  if (!d.exists) {
    d.exists = true
    d.created = true
    d.content = docDefault(d.name)
    editing.value = d.name
    return
  }
  editing.value = editing.value === d.name ? '' : d.name
}
const onDocInput = (e: Event) => {
  const d = editingDoc.value
  if (!d) return
  d.content = (e.target as HTMLTextAreaElement).value
  d.edited = true
}

// —— 打开 / 保存 ——
const reset = () => {
  const w = props.workspace
  form.source = 'dir'
  form.name = w?.name ?? ''
  form.path = w?.path ?? ''
  form.gitUrl = ''
  form.gitBranch = ''
  form.clonePath = ''
  form.runtimeId = w?.runtimeId ?? ''
  form.providerId = w?.providerId ?? ''
  form.model = w?.model ?? ''
  form.effort = w?.effort ?? ''
  form.mode = w?.mode ?? ''
  form.mcpIds = [...(w?.mcpIds ?? [])]
  docs.value = blankDocs()
  editing.value = ''
  browseOpen.value = false
  saving.value = false
  autoName = ''
  mcpTouched = false
  docsPath = ''
  docSeq += 1
}

const loadAll = () => {
  const id = ++openSeq
  optionsLoaded.value = false
  serversState.value = 'loading'
  // 读不到时请求层已经提示过；选择器按「没有可选项」显示，照样可以保存
  void refreshModels().then(() => {
    if (id !== openSeq) return
    optionsLoaded.value = true
    applyDefaultModel()
  })
  listAgentMcpServers()
    .then(({ data }) => {
      if (id !== openSeq) return
      servers.value = data.servers ?? []
      serversState.value = 'ok'
      // 新建时先勾上全局「新会话默认开启」的那些
      if (!isEdit.value && !mcpTouched) form.mcpIds = servers.value.filter((s) => s.defaultOn).map((s) => s.id)
    })
    .catch(() => {
      if (id === openSeq) serversState.value = 'failed'
    })
  const path = (props.workspace?.path ?? '').trim()
  if (!path) return
  // 还在克隆或目录已经不在的工作区读不到文件，不去读
  if (props.workspace?.cloneStatus || props.workspace?.missing) docsPath = path
  else void loadDocs(path)
}

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    reset()
    loadAll()
  },
  { immediate: true },
)

const save = async () => {
  if (saving.value) return
  const git = isGit.value
  if (git && !form.gitUrl.trim()) {
    feedback.warning(t('agent.workspaces.dialog.needGitUrl'))
    return
  }
  if (!git && !form.path.trim()) {
    feedback.warning(t('agent.workspaces.dialog.needDir'))
    return
  }
  const name = form.name.trim()
  const common = {
    name,
    runtimeId: form.runtimeId,
    providerId: form.providerId,
    model: form.model,
    effort: form.effort,
    mode: form.mode,
    // 已经删掉的 MCP 服务器不再带上
    mcpIds:
      serversState.value === 'ok'
        ? form.mcpIds.filter((id) => servers.value.some((s) => s.id === id))
        : [...form.mcpIds],
    docs: docs.value
      .filter((d) => d.created || d.edited)
      .map((d) => ({ name: d.name, exists: true, content: d.content })),
  }
  saving.value = true
  try {
    const w = props.workspace
    const { data } = w
      ? await updateAgentWorkspace(w.id, { ...common, path: form.path.trim() })
      : await createAgentWorkspace({
          ...common,
          source: form.source,
          path: git ? form.clonePath.trim() : form.path.trim(),
          gitUrl: git ? form.gitUrl.trim() : '',
          gitBranch: git ? form.gitBranch.trim() : '',
        })
    const saved = data.workspace
    feedback.success(
      t(w ? 'agent.workspaces.dialog.saved' : 'agent.workspaces.dialog.created', { name: saved?.name || name }),
    )
    if (saved) emit('saved', saved)
    emit('update:modelValue', false)
  } catch {
    // 请求层已经提示过，弹窗留着
  } finally {
    saving.value = false
  }
}
</script>

<style lang="scss">
.ag-ws-form {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}
.ag-ws-form__grid {
  display: grid;
  gap: 8px;
}
.ag-ws-form__grid--dir {
  grid-template-columns: minmax(0, 1fr) minmax(0, 2fr);
}
.ag-ws-form__grid--git {
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) minmax(0, 1.6fr);
}
.ag-ws-form__grid--model {
  grid-template-columns: minmax(0, 1.7fr) minmax(0, 0.8fr) minmax(0, 1fr);
}
.ag-ws-form__grid--docs {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
.ag-ws-form__dir {
  padding-right: 3px;
}
.ag-ws-form__fl {
  display: flex;
  align-items: baseline;
  gap: 8px;
  min-width: 0;
  white-space: nowrap;
}
.ag-ws-form__by {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  font-weight: 400;
  color: var(--m-ink2);
}
.ag-ws-form__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.ag-ws-form .ag-fld:disabled {
  cursor: not-allowed;
}
.ag-ws-form .ag-fld:disabled > .iconify {
  opacity: 0.5;
}
.ag-ws-doc {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 32px;
  padding: 0 3px 0 12px;
  border-radius: 10px;
  border: 1px solid var(--m-line);
}
.ag-ws-doc__name {
  flex: 1;
  font-size: 13px;
  font-weight: 600;
}
.ag-ws-editor {
  border: 1px solid var(--m-line);
  border-radius: 10px;
  overflow: hidden;
}
.ag-ws-editor__head {
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 32px;
  padding: 0 3px 0 10px;
  background: var(--m-s2);
}
.ag-ws-editor__name {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  font-weight: 600;
}
.ag-ws-editor__ta {
  display: block;
  width: 100%;
  border: 0;
  outline: none;
  resize: vertical;
  padding: 6px 10px;
  background: var(--m-bg);
  color: var(--m-ink);
  font-size: 12px;
  line-height: 1.6;
}
@media (width < 992px) {
  .ag-ws-doc,
  .ag-ws-editor__head {
    min-height: 40px;
    padding-right: 4px;
  }
  .ag-ws-editor__ta {
    font-size: 14px;
  }
}
</style>
