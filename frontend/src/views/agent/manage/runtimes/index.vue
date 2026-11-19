<!-- Agent 管理 · 运行时：每个 Agent 一行（重新检测；Claude Code 还有安装方式和检查更新），可以添加其它 ACP Agent；
     Codex 没有单文件发行包，没就绪时给出安装命令，由用户在服务器上自己执行。
     下面是全局的并发上限和空闲时长，自己输入数字。 -->
<template>
  <div class="ag ag-page">
    <AgManageHeader>
      {{ subtitle }}
      <template #actions>
        <button type="button" class="ag-btn" @click="openAdd">
          <UIcon name="i-lucide-plus" />{{ t('agent.runtimes.add') }}
        </button>
      </template>
    </AgManageHeader>

    <div class="ag-body">
      <h2 class="ag-sh">{{ t('agent.runtimes.agents') }}</h2>
      <div class="ag-grp">
        <div v-for="r in runtimes" :key="r.id" class="ag-rw">
          <div class="ag-rm">
            <div class="ag-rt">
              <span class="ag-rt-name">{{ r.name }}</span>
              <span class="ag-pill" :class="pillOf(r).cls">{{ pillOf(r).text }}</span>
              <span v-if="metaOf(r)" class="ag-rt-meta" :title="metaOf(r)">{{ metaOf(r) }}</span>
            </div>
            <div class="ag-rs ag-mono ag-rt-sub">{{ adapterOf(r) || '—' }} · {{ [r.command, ...(r.args ?? [])].join(' ') }}</div>
            <div v-if="!r.ready && INSTALL[r.id]" class="ag-rt-install">
              <div v-for="c in INSTALL[r.id]" :key="c.cmd" class="ag-rt-line">
                <code class="ag-mono ag-rt-code">{{ c.cmd }}</code>
                <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.common.copy')" :title="t('agent.common.copy')" @click="copy(c.cmd)">
                  <UIcon name="i-lucide-copy" />
                </button>
                <span class="ag-rs">{{ t(c.note) }}</span>
              </div>
            </div>
          </div>
          <div class="ag-rc">
            <template v-if="r.canInstall">
              <AppDropdown v-if="!locked(r)" align="end" :width="230" panel-class="ag ag-menu">
                <template #trigger>
                  <button
                    type="button"
                    class="ag-btn ag-btn--o ag-btn--sm ag-rt-mode"
                    aria-haspopup="listbox"
                    :aria-label="t('agent.runtimes.mode.label', { name: r.name })"
                  >
                    {{ modeText(r.managed) }}<UIcon name="i-lucide-chevron-down" />
                  </button>
                </template>
                <template #default="{ close }">
                  <button
                    v-for="o in MODES"
                    :key="String(o)"
                    type="button"
                    class="ag-mi"
                    role="option"
                    :class="{ 'is-on': r.managed === o }"
                    :aria-selected="r.managed === o"
                    @click="pickMode(r, o, close)"
                  >
                    {{ modeText(o) }}
                  </button>
                </template>
              </AppDropdown>
              <button
                v-else
                type="button"
                class="ag-btn ag-btn--o ag-btn--sm ag-rt-mode"
                disabled
                :aria-label="t('agent.runtimes.mode.label', { name: r.name })"
              >
                <UIcon v-if="spinning(r, 'mode')" name="i-lucide-loader-circle" class="ag-spin" />
                {{ modeText(busy.get(r.id)?.managed ?? r.managed) }}<UIcon name="i-lucide-chevron-down" />
              </button>
            </template>
            <button
              type="button"
              class="ag-ib ag-ib--sm"
              :aria-label="`${t('agent.runtimes.detect')} ${r.name}`"
              :title="t('agent.runtimes.detect')"
              :disabled="locked(r)"
              @click="detect(r)"
            >
              <UIcon name="i-lucide-refresh-cw" :class="{ 'ag-spin': spinning(r, 'detect') }" />
            </button>
            <button
              v-if="r.canInstall"
              type="button"
              class="ag-ib ag-ib--sm"
              :aria-label="`${t('agent.runtimes.update')} ${r.name}`"
              :title="t('agent.runtimes.update')"
              :disabled="locked(r)"
              @click="checkUpdate(r)"
            >
              <UIcon
                :name="spinning(r, 'update') ? 'i-lucide-loader-circle' : 'i-lucide-download'"
                :class="{ 'ag-spin': spinning(r, 'update') }"
              />
            </button>
            <button
              v-if="!r.builtin"
              type="button"
              class="ag-ib ag-ib--sm"
              :aria-label="`${t('agent.common.remove')} ${r.name}`"
              :title="t('agent.common.remove')"
              :disabled="locked(r)"
              @click="askRemove(r)"
            >
              <UIcon name="i-lucide-trash-2" />
            </button>
          </div>
        </div>
        <div v-if="!runtimes.length" class="ag-empty">
          {{ loaded ? t('agent.runtimes.empty') : t('agent.common.loading') }}
        </div>
      </div>

      <h2 class="ag-sh ag-gap">{{ t('agent.runtimes.settings') }}</h2>
      <div class="ag-grp">
        <div class="ag-rw ag-rt-set">
          <div class="ag-rm">
            <div class="ag-rt"><span>{{ t('agent.runtimes.maxProcesses') }}</span></div>
            <div class="ag-rs">{{ t('agent.runtimes.maxProcessesDesc') }}</div>
          </div>
          <div class="ag-rc">
            <span class="ag-fld ag-rt-num">
              <input
                class="ag-num"
                type="text"
                inputmode="numeric"
                maxlength="5"
                :value="text.maxProcesses"
                :disabled="!settingsReady"
                :aria-label="t('agent.runtimes.maxProcesses')"
                @focus="focused = 'maxProcesses'"
                @input="onDigits($event, 'maxProcesses')"
                @blur="commit('maxProcesses')"
                @keydown.enter="blurOnEnter"
              />
              <span v-if="t('agent.runtimes.unitCount')" class="ag-rt-unit">{{ t('agent.runtimes.unitCount') }}</span>
            </span>
          </div>
        </div>
        <div class="ag-rw ag-rt-set">
          <div class="ag-rm">
            <div class="ag-rt"><span>{{ t('agent.runtimes.idle') }}</span></div>
            <div class="ag-rs">{{ t('agent.runtimes.idleDesc') }}</div>
          </div>
          <div class="ag-rc">
            <span class="ag-fld ag-rt-num ag-rt-num--wide">
              <input
                class="ag-num"
                type="text"
                inputmode="numeric"
                maxlength="5"
                :value="text.idleMinutes"
                :disabled="!settingsReady"
                :aria-label="t('agent.runtimes.idleLabel')"
                @focus="focused = 'idleMinutes'"
                @input="onDigits($event, 'idleMinutes')"
                @blur="commit('idleMinutes')"
                @keydown.enter="blurOnEnter"
              />
              <span class="ag-rt-unit">{{ t('agent.runtimes.unitMinutes') }}</span>
            </span>
          </div>
        </div>
      </div>
    </div>

    <AgDialog v-model="addOpen" :title="t('agent.runtimes.add')" :width="420">
      <label class="ag-field">
        <span class="ag-fl">{{ t('agent.common.name') }}</span>
        <span class="ag-fld">
          <input
            v-model="addName"
            type="text"
            :placeholder="t('agent.runtimes.dialog.namePlaceholder')"
            :aria-label="t('agent.common.name')"
            @keydown.enter.prevent="saveAdd"
          />
        </span>
      </label>
      <label class="ag-field">
        <span class="ag-fl">{{ t('agent.runtimes.dialog.command') }}</span>
        <span class="ag-fld">
          <input
            v-model="addCommand"
            class="ag-mono ag-rt-cmd"
            type="text"
            :placeholder="t('agent.runtimes.dialog.commandPlaceholder')"
            :aria-label="t('agent.runtimes.dialog.command')"
            @keydown.enter.prevent="saveAdd"
          />
        </span>
      </label>
      <div class="ag-rt-acts">
        <button type="button" class="ag-btn" @click="addOpen = false">{{ t('agent.common.cancel') }}</button>
        <button type="button" class="ag-btn ag-btn--p" :disabled="adding" @click="saveAdd">
          <UIcon v-if="adding" name="i-lucide-loader-circle" class="ag-spin" />{{ t('agent.common.add') }}
        </button>
      </div>
    </AgDialog>

    <AgConfirm
      v-model="removeOpen"
      danger
      :title="t('agent.runtimes.removeTitle', { name: removeTarget?.name || '' })"
      :message="t('agent.runtimes.removeMsg')"
      :confirm-text="t('agent.common.remove')"
      :on-confirm="doRemove"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  createAgentRuntime,
  deleteAgentRuntime,
  detectAgentRuntime,
  listAgentRuntimes,
  setAgentRuntimeMode,
  updateAgentRunSettings,
  updateAgentRuntime,
} from '@/api/agent'
import type { AgentRunSettings, AgentRuntime } from '@/types/v1/agent'
import AgConfirm from '@/views/agent/components/AgConfirm.vue'
import AgDialog from '@/views/agent/components/AgDialog.vue'
import AgManageHeader from '@/views/agent/components/AgManageHeader.vue'
import { useAgentOverview } from '@/views/agent/composables/useAgentOverview'
import { relTime } from '@/views/agent/utils/format'

defineOptions({ name: 'AgentManageRuntimesView' })

type Op = 'mode' | 'detect' | 'update'
type SettingKey = keyof AgentRunSettings

const { t } = useI18n()
const { overview, refresh } = useAgentOverview()

const runtimes = ref<AgentRuntime[]>([])
const loaded = ref(false)

const subtitle = computed(() => {
  const o = overview.value
  return o ? t('agent.runtimes.subtitle', { n: o.runtimesReady }) : ''
})

// 适配器：Agent 自报的名字（去掉 npm 作用域）和版本
const adapterOf = (r: AgentRuntime) => [r.agentName?.split('/').pop(), r.agentVersion].filter(Boolean).join(' ')

// —— 加载；有 Agent 正在安装时每 3 秒看一次（切走标签或离开页面就停）——
let active = false
let timer: number | undefined
const stopPolling = () => {
  window.clearTimeout(timer)
  timer = undefined
}
const pollIn = (ms: number) => {
  stopPolling()
  if (active) timer = window.setTimeout(() => void load(), ms)
}
const load = async () => {
  try {
    const { data } = await listAgentRuntimes()
    runtimes.value = data.runtimes || []
    applySettings(data.settings)
    if (runtimes.value.some((r) => r.installing)) pollIn(3000)
  } catch {
    /* shown */
  } finally {
    loaded.value = true
  }
}

// —— 行上的显示 ——
const MODES = [true, false]
// 没有单文件发行包、要用户自己装的内置 Agent：在服务器上执行哪条命令
const INSTALL: Record<string, { cmd: string; note: string }[]> = {
  codex: [
    { cmd: 'npm install -g @agentclientprotocol/codex-acp', note: 'agent.runtimes.install.global' },
    { cmd: 'npx -y @agentclientprotocol/codex-acp', note: 'agent.runtimes.install.npx' },
  ],
}
const copy = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    feedback.success(t('agent.common.copied'))
  } catch {
    feedback.error(t('agent.runtimes.install.copyFailed'))
  }
}
const modeText = (managed: boolean) => t(managed ? 'agent.runtimes.mode.managed' : 'agent.runtimes.mode.system')
const pillOf = (r: AgentRuntime) => {
  if (r.installing) return { cls: 'ag-pill--warn', text: t('agent.runtimes.status.installing') }
  if (r.ready) return { cls: 'ag-pill--ok', text: t('agent.runtimes.status.ready') }
  return { cls: 'ag-pill--bad', text: t('agent.runtimes.status.notReady') }
}
const metaOf = (r: AgentRuntime) => {
  if (r.installing) return ''
  if (!r.ready) return r.error || (r.detectedAt ? '' : t('agent.runtimes.meta.never'))
  const parts: string[] = []
  const sessions = Number(r.activeSessions) || 0
  if (sessions > 0) parts.push(t('agent.runtimes.meta.sessions', { n: sessions }))
  const when = relTime(r.detectedAt)
  if (when) parts.push(t('agent.runtimes.meta.detected', { time: when }))
  return parts.join(' · ')
}

// —— 行上的操作：进行中这一行的控件都停用 ——
const busy = reactive(new Map<string, { op: Op; managed?: boolean }>())
const locked = (r: AgentRuntime) => busy.has(r.id) || r.installing
const spinning = (r: AgentRuntime, op: Op) => busy.get(r.id)?.op === op || (op === 'mode' && r.installing && !busy.has(r.id))
const run = async (r: AgentRuntime, state: { op: Op; managed?: boolean }, action: () => Promise<void>) => {
  if (locked(r)) return
  busy.set(r.id, state)
  try {
    await action()
  } catch {
    /* shown */
  } finally {
    busy.delete(r.id)
    void refresh()
    await load()
  }
}

// 切换安装方式和检查更新可能要下载，请求会开几分钟：先看一眼，让「正在安装」显示出来
const pickMode = (r: AgentRuntime, managed: boolean, close: () => void) => {
  close()
  if (r.managed === managed) return
  void run(r, { op: 'mode', managed }, async () => {
    await setAgentRuntimeMode(r.id, managed)
    feedback.success(t('agent.runtimes.mode.switched', { name: r.name, mode: modeText(managed) }))
  })
  pollIn(1500)
}
const detect = (r: AgentRuntime) =>
  run(r, { op: 'detect' }, async () => {
    const { data } = await detectAgentRuntime(r.id)
    if (data.runtime?.ready) feedback.success(t('agent.runtimes.detectOk', { name: r.name }))
    else {
      feedback.error(
        t('agent.runtimes.withMessage', { name: r.name, message: data.runtime?.error || t('agent.runtimes.status.notReady') }),
      )
    }
  })
const checkUpdate = (r: AgentRuntime) => {
  void run(r, { op: 'update' }, async () => {
    const { data } = await updateAgentRuntime(r.id)
    const { current, latest } = data
    const name = r.name
    if (data.updated) feedback.success(t('agent.runtimes.updated', { name, latest: latest || current }))
    else if (!latest) feedback.warning(t('agent.runtimes.noVersion', { name }))
    else if (latest === current) feedback.success(t('agent.runtimes.upToDate', { name }))
    else {
      const managed = data.runtime?.managed ?? r.managed
      feedback.info(
        t(managed ? 'agent.runtimes.newer' : 'agent.runtimes.newerOnServer', {
          name,
          latest,
          current: current || t('agent.common.unknown'),
        }),
      )
    }
  })
  pollIn(1500)
}

// —— 移除自己添加的 Agent ——
const removeOpen = ref(false)
const removeTarget = ref<AgentRuntime | null>(null)
const askRemove = (r: AgentRuntime) => {
  removeTarget.value = r
  removeOpen.value = true
}
const doRemove = async () => {
  const r = removeTarget.value
  if (!r) return
  await deleteAgentRuntime(r.id)
  feedback.success(t('agent.runtimes.removed', { name: r.name }))
  void refresh()
  await load()
}

// —— 添加 ACP Agent（添加后立刻检测，可能要等一会）——
const addOpen = ref(false)
const addName = ref('')
const addCommand = ref('')
const adding = ref(false)
const openAdd = () => {
  addName.value = ''
  addCommand.value = ''
  addOpen.value = true
}
const saveAdd = async () => {
  if (adding.value) return
  const name = addName.value.trim()
  const command = addCommand.value.trim()
  if (!name) {
    feedback.warning(t('agent.runtimes.dialog.needName'))
    return
  }
  if (!command) {
    feedback.warning(t('agent.runtimes.dialog.needCommand'))
    return
  }
  adding.value = true
  try {
    await createAgentRuntime(name, command)
    feedback.success(t('agent.runtimes.dialog.added', { name }))
    addOpen.value = false
    void refresh()
    await load()
  } catch {
    /* shown */
  } finally {
    adding.value = false
  }
}

// —— 运行设置：失焦或回车时，数字变了才保存 ——
const saved = reactive<AgentRunSettings>({ maxProcesses: 0, idleMinutes: 0 })
const text = reactive<Record<SettingKey, string>>({ maxProcesses: '', idleMinutes: '' })
const settingsReady = ref(false)
const focused = ref<SettingKey | ''>('')
// 正在输入的那一格不被刷新覆盖
const applySettings = (s?: AgentRunSettings) => {
  if (!s) return
  saved.maxProcesses = Number(s.maxProcesses) || 0
  saved.idleMinutes = Number(s.idleMinutes) || 0
  if (focused.value !== 'maxProcesses') text.maxProcesses = String(saved.maxProcesses)
  if (focused.value !== 'idleMinutes') text.idleMinutes = String(saved.idleMinutes)
  settingsReady.value = true
}
const onDigits = (e: Event, key: SettingKey) => {
  const el = e.target as HTMLInputElement
  const clean = el.value.replace(/[^0-9]/g, '').slice(0, 5)
  if (el.value !== clean) el.value = clean
  text[key] = clean
}
const blurOnEnter = (e: KeyboardEvent) => (e.target as HTMLInputElement).blur()
const commit = async (key: SettingKey) => {
  if (focused.value === key) focused.value = ''
  const prev = saved[key]
  // 清空了就还原；进程上限至少 1
  if (text[key] === '') {
    text[key] = String(prev)
    return
  }
  const n = key === 'maxProcesses' ? Math.max(1, Number(text[key])) : Number(text[key])
  text[key] = String(n)
  if (n === prev) return
  saved[key] = n
  try {
    const { data } = await updateAgentRunSettings({ ...saved })
    applySettings(data.settings)
    feedback.success(t('agent.runtimes.saved'))
  } catch {
    saved[key] = prev
    if (focused.value !== key) text[key] = String(prev)
  }
}

onMounted(() => {
  active = true
  void load()
})
// 标签页在 KeepAlive 里：切回来时静默刷新，切走时停掉轮询
onActivated(() => {
  active = true
  if (loaded.value) void load()
})
onDeactivated(() => {
  active = false
  stopPolling()
})
onBeforeUnmount(() => {
  active = false
  stopPolling()
})
</script>

<style lang="scss">
// 要用户自己装的 Agent：安装命令，一行一条，带复制按钮
.ag-rt-install {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 6px;
}
.ag-rt-line {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 8px;
  min-width: 0;
}
.ag-rt-code {
  padding: 2px 8px;
  border-radius: 6px;
  background: var(--m-s2);
  font-size: 12px;
  color: var(--m-ink);
  overflow-wrap: anywhere;
}
// 名称不让位，放不下时省略的是后面那段状态说明
.ag-rt-name {
  flex: 0 0 auto;
  max-width: 50%;
}
.ag-rt-meta {
  min-width: 0;
  font-size: 12px;
  font-weight: 400;
  color: var(--m-ink2);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ag-btn.ag-rt-mode {
  margin-right: 4px;
  padding-right: 8px;
}
// 数字输入：右对齐，后面带单位
.ag-fld.ag-rt-num {
  width: 104px;
}
.ag-fld.ag-rt-num--wide {
  width: 116px;
}
.ag-fld.ag-rt-num input {
  text-align: right;
}
.ag-rt-unit {
  font-size: 12.5px;
  color: var(--m-ink2);
  white-space: nowrap;
}
.ag-fld input.ag-mono.ag-rt-cmd {
  font-size: 12px;
}
.ag-rt-acts {
  display: flex;
  justify-content: flex-end;
  gap: 6px;
  padding-top: 2px;
}
@media (width < 992px) {
  .ag-rt-name {
    max-width: 100%;
  }
  .ag-rt-meta {
    flex: 1 1 100%;
    white-space: normal;
  }
  // 启动命令很长时在卡片里折行
  .ag-rt-sub {
    overflow-wrap: anywhere;
  }
  .ag-btn.ag-rt-mode {
    margin-right: auto;
  }
  // 设置行：输入框留在文字右边
  .ag-rw.ag-rt-set {
    flex-wrap: nowrap;
  }
  .ag-rt-set .ag-rm {
    flex: 1 1 0;
  }
  .ag-rt-set .ag-rc {
    flex: none;
  }
}
</style>
