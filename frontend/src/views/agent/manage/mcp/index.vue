<!-- Agent 管理 · MCP：一行一个服务器（名称、传输方式、连接状态、说明和命令 / 地址），
     行尾是「新会话默认开启」、测试连接、编辑、删除；右上可以粘贴 mcpServers JSON 导入。
     哪些工作区能用、某个会话里连没连上都不在这里显示。 -->
<template>
  <div class="ag ag-page">
    <AgManageHeader>
      <template #default>
        <template v-if="loaded">
          {{ t('agent.mcp.count', { n: servers.length }) }} ·
          <span class="ag-mcp-okn">{{ t('agent.mcp.okCount', { n: okCount }) }}</span>
        </template>
        <template v-else>{{ t('agent.common.loading') }}</template>
      </template>
      <template #actions>
        <button type="button" class="ag-btn" @click="pasteOpen = true">
          <UIcon name="i-lucide-code-xml" />{{ t('agent.mcp.pasteImport') }}
        </button>
        <button type="button" class="ag-btn ag-btn--p" @click="openForm(null)">
          <UIcon name="i-lucide-plus" />{{ t('agent.mcp.add') }}
        </button>
      </template>
    </AgManageHeader>

    <div class="ag-body">
      <div class="ag-grp">
        <div v-for="r in rows" :key="r.s.id" class="ag-rw">
          <div class="ag-rm">
            <div class="ag-rt">
              <span class="ag-mono">{{ r.s.name }}</span>
              <span class="ag-tag">{{ r.tag }}</span>
              <span class="ag-pill" :class="r.pill.cls">{{ r.pill.text }}</span>
              <span v-if="r.note" class="ag-mcp-note" :title="r.note">{{ r.note }}</span>
            </div>
            <div class="ag-rs">
              {{ r.s.description || t('agent.mcp.noDesc') }} · <span class="ag-mono ag-mcp-cmd" :title="r.cmd">{{ r.cmd }}</span>
            </div>
          </div>
          <div class="ag-rc">
            <label class="ag-mcp-def">
              {{ t('agent.mcp.defaultOn') }}
              <AgSwitch
                :model-value="r.s.defaultOn"
                :label="t('agent.mcp.defaultOnOf', { name: r.s.name })"
                @update:model-value="setDefault(r.s, $event)"
              />
            </label>
            <button
              type="button"
              class="ag-ib ag-ib--sm"
              :aria-label="t('agent.mcp.testOf', { name: r.s.name })"
              :title="t('agent.common.testConnection')"
              :disabled="testing.has(r.s.id)"
              @click="testRow(r.s)"
            >
              <UIcon v-if="testing.has(r.s.id)" name="i-lucide-loader-circle" class="ag-spin" />
              <UIcon v-else name="i-lucide-zap" />
            </button>
            <button
              type="button"
              class="ag-ib ag-ib--sm"
              :aria-label="t('agent.mcp.editOf', { name: r.s.name })"
              :title="t('agent.common.edit')"
              @click="openForm(r.s)"
            >
              <UIcon name="i-lucide-pencil" />
            </button>
            <button
              type="button"
              class="ag-ib ag-ib--sm"
              :aria-label="t('agent.mcp.deleteOf', { name: r.s.name })"
              :title="t('agent.common.delete')"
              @click="askDelete(r.s)"
            >
              <UIcon name="i-lucide-trash-2" />
            </button>
          </div>
        </div>
        <div v-if="!loaded" class="ag-empty">{{ t('agent.common.loading') }}</div>
        <div v-else-if="!servers.length" class="ag-empty">{{ t('agent.mcp.empty') }}</div>
      </div>
    </div>

    <McpFormDialog v-model="formOpen" :server="formServer" @saved="onSaved" />

    <AgDialog v-model="pasteOpen" :title="t('agent.mcp.pasteImport')" :width="480">
      <textarea
        v-model="pasteText"
        class="ag-ta ag-noscroll"
        rows="9"
        spellcheck="false"
        :aria-label="t('agent.mcp.paste.label')"
        :placeholder="pastePlaceholder"
      />
      <div class="ag-mcp-acts">
        <button type="button" class="ag-btn" @click="pasteOpen = false">{{ t('agent.common.cancel') }}</button>
        <button type="button" class="ag-btn ag-btn--p" :disabled="pasting" @click="doPaste">
          <UIcon v-if="pasting" name="i-lucide-loader-circle" class="ag-spin" />{{ t('agent.mcp.paste.submit') }}
        </button>
      </div>
    </AgDialog>

    <AgConfirm
      v-model="deleteOpen"
      danger
      :title="t('agent.mcp.confirmDelete.title')"
      :message="t('agent.mcp.confirmDelete.message', { name: deleteTarget?.name ?? '' })"
      :confirm-text="t('agent.common.delete')"
      :on-confirm="doDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  deleteAgentMcpServer,
  importAgentMcpServers,
  listAgentMcpServers,
  setAgentMcpDefault,
  testAgentMcpServer,
} from '@/api/agent'
import type { AgentMcpServer } from '@/types/v1/agent'
import AgConfirm from '@/views/agent/components/AgConfirm.vue'
import AgDialog from '@/views/agent/components/AgDialog.vue'
import AgManageHeader from '@/views/agent/components/AgManageHeader.vue'
import AgSwitch from '@/views/agent/components/AgSwitch.vue'
import { useAgentOverview } from '@/views/agent/composables/useAgentOverview'
import { relTime } from '@/views/agent/utils/format'
import McpFormDialog from './McpFormDialog.vue'

defineOptions({ name: 'AgentManageMcpView' })

const { t } = useI18n()
const { overview, refresh } = useAgentOverview()

// —— 列表 ——
const servers = ref<AgentMcpServer[]>([])
const loaded = ref(false)
const load = async () => {
  try {
    const { data } = await listAgentMcpServers()
    servers.value = data.servers || []
  } catch {
    // 请求层已经提示过
  } finally {
    loaded.value = true
  }
}
/** 数量变了：列表和标签上的数字一起刷新 */
const reloadAll = () => Promise.all([load(), refresh()])
onMounted(load)
// 标签页是 keep-alive 的：再切回来时悄悄刷新一次（会话里的 /install-mcp 也会往这里加服务器）
onActivated(() => {
  if (loaded.value) void load()
})

const okCount = computed(() => Number(overview.value?.mcpOk ?? 0))

// 失败原因短的直接写在状态里，长的状态写「连接失败」、原因放在后面
const FAIL_PILL_MAX = 14
const rows = computed(() =>
  servers.value.map((s) => {
    const http = s.transport === 'http'
    const cmd = http ? s.url : [s.command, ...(s.args || [])].join(' ').trim()
    let pill = { cls: '', text: t('agent.common.notTested') }
    let note = t('agent.mcp.note.untested')
    if (s.status === 'ok') {
      const time = relTime(s.testedAt)
      pill = { cls: 'ag-pill--ok', text: t('agent.mcp.status.ok') }
      note = time ? t('agent.mcp.note.ok', { n: s.toolCount, time }) : t('agent.mcp.note.okNoTime', { n: s.toolCount })
    } else if (s.status === 'fail') {
      const reason = (s.statusText || '').trim()
      const inPill = !!reason && reason.length <= FAIL_PILL_MAX
      pill = { cls: 'ag-pill--bad', text: inPill ? reason : t('agent.mcp.status.fail') }
      note = inPill ? '' : reason
    }
    return { s, tag: http ? 'HTTP' : 'stdio', pill, note, cmd }
  }),
)

// —— 新会话默认开启 ——
const setDefault = async (s: AgentMcpServer, on: boolean) => {
  s.defaultOn = on
  try {
    await setAgentMcpDefault(s.id, on)
  } catch {
    s.defaultOn = !on
  }
}

// —— 测试连接 ——
const testing = ref(new Set<string>())
const testRow = async (s: Pick<AgentMcpServer, 'id' | 'name'>) => {
  if (testing.value.has(s.id)) return
  testing.value.add(s.id)
  try {
    const { data } = await testAgentMcpServer({ id: s.id })
    if (data.ok) feedback.success(t('agent.mcp.toast.testOk', { name: s.name, n: (data.tools || []).length }))
    else feedback.error(t('agent.mcp.toast.testFail', { name: s.name, message: data.message || t('agent.mcp.status.fail') }))
    await reloadAll()
  } catch {
    // 请求层已经提示过
  } finally {
    testing.value.delete(s.id)
  }
}

// —— 添加 / 编辑 ——
const formOpen = ref(false)
const formServer = ref<AgentMcpServer | null>(null)
const openForm = (s: AgentMcpServer | null) => {
  formServer.value = s
  formOpen.value = true
}
const onSaved = async () => {
  await reloadAll()
}

// —— 粘贴 JSON 导入 ——
const pasteOpen = ref(false)
const pasteText = ref('')
const pasting = ref(false)
const pastePlaceholder = computed(
  () =>
    `{ "mcpServers": { "${t('agent.mcp.paste.phName')}": { "command": "npx", "args": ["-y", "${t('agent.mcp.paste.phPackage')}"] } } }`,
)
const doPaste = async () => {
  if (pasting.value) return
  if (!pasteText.value.trim()) {
    feedback.warning(t('agent.mcp.toast.needJson'))
    return
  }
  pasting.value = true
  try {
    const { data } = await importAgentMcpServers(pasteText.value)
    feedback.success(t('agent.mcp.toast.imported', { n: (data.servers || []).length }))
    pasteOpen.value = false
    pasteText.value = ''
    await reloadAll()
  } catch {
    // 后端看不懂这段 JSON 时会回 400，请求层已经提示过，弹窗留着
  } finally {
    pasting.value = false
  }
}

// —— 删除 ——
const deleteOpen = ref(false)
const deleteTarget = ref<AgentMcpServer | null>(null)
const askDelete = (s: AgentMcpServer) => {
  deleteTarget.value = s
  deleteOpen.value = true
}
const doDelete = async () => {
  const s = deleteTarget.value
  if (!s) return
  await deleteAgentMcpServer(s.id)
  feedback.success(t('agent.mcp.toast.deleted', { name: s.name }))
  await reloadAll()
}
</script>

<style lang="scss">
.ag-mcp-okn {
  color: var(--m-ok);
  font-weight: 600;
}
.ag-mcp-note {
  min-width: 0;
  font-size: 12px;
  font-weight: 400;
  color: var(--m-ink2);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ag-mcp-def {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-right: 6px;
  font-size: 12px;
  color: var(--m-ink2);
  white-space: nowrap;
  cursor: pointer;
}
.ag-mcp-acts {
  display: flex;
  justify-content: flex-end;
  gap: 6px;
}
@media (width < 992px) {
  .ag-mcp-note {
    white-space: normal;
  }
  // 很长的地址或命令在卡片里折行，不撑出屏幕
  .ag-mcp-cmd {
    overflow-wrap: anywhere;
  }
  // 卡片里：开关靠左，三个图标按钮靠右
  .ag-mcp-def {
    margin-right: auto;
  }
}
</style>
