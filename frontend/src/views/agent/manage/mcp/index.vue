<!-- Agent 设置 · MCP：一行一个服务器（名称、传输方式、连接状态、说明和命令 / 地址），
     行尾是「新会话默认开启」、测试连接、编辑、删除；右上可以粘贴 mcpServers JSON 导入。
     哪些工作区能用、某个会话里连没连上都不在这里显示。
     自己的服务器，后面是管理员共享给自己的（只能用，没有操作按钮）。
     在 Agent 管理页里（useAgentAll）分两组：共享的（管理员在这里加、改、删，选共享给所有人还是指定的人），
     和各人自己的（行上带主人，只能删除）。 -->
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
        <button type="button" class="ag-btn" @click="openPaste">
          <UIcon name="i-lucide-code-xml" />{{ t('agent.mcp.pasteImport') }}
        </button>
        <button type="button" class="ag-btn ag-btn--p" @click="openForm(null)">
          <UIcon name="i-lucide-plus" />{{ all ? t('agent.share.addShared') : t('agent.mcp.add') }}
        </button>
      </template>
    </AgManageHeader>

    <div class="ag-body">
      <template v-for="g in groups" :key="g.key">
        <h2 v-if="all" class="ag-sh" :class="{ 'ag-gap': g.key === 'personal' }">{{ g.title }}</h2>
        <div class="ag-grp">
        <div v-for="r in g.rows" :key="r.s.id" class="ag-rw">
          <div class="ag-rm">
            <div class="ag-rt">
              <span class="ag-mono">{{ r.s.name }}</span>
              <span class="ag-tag">{{ r.tag }}</span>
              <span v-if="r.shareTag" class="ag-tag ag-tag--p" :title="r.scope">{{ r.shareTag }}</span>
              <span class="ag-pill" :class="r.pill.cls">{{ r.pill.text }}</span>
              <span v-if="r.note" class="ag-mcp-note" :title="r.note">{{ r.note }}</span>
            </div>
            <div class="ag-rs">
              <template v-if="all && !r.shared">{{ r.owner }} · </template>
              <template v-if="all && r.shared && r.scope">{{ r.scope }} · </template>
              {{ r.s.description || t('agent.mcp.noDesc') }} · <span class="ag-mono ag-mcp-cmd" :title="r.cmd">{{ r.cmd }}</span>
            </div>
          </div>
          <div v-if="all && !r.shared" class="ag-rc">
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
          <div v-else-if="!all && r.shared" class="ag-rc">
            <span class="ag-mcp-lock" :title="t('agent.share.sharedReadonly')">
              <UIcon name="i-lucide-lock" />{{ r.s.defaultOn ? t('agent.mcp.defaultOn') : '' }}
            </span>
          </div>
          <div v-else class="ag-rc">
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
        <div v-else-if="!g.rows.length" class="ag-empty">{{ g.empty }}</div>
        </div>
      </template>
    </div>

    <McpFormDialog v-model="formOpen" :server="formServer" :shared="formShared" @saved="onSaved" />

    <AgDialog v-model="pasteOpen" :title="t('agent.mcp.pasteImport')" :width="480">
      <textarea
        v-model="pasteText"
        class="ag-ta ag-noscroll"
        rows="9"
        spellcheck="false"
        :aria-label="t('agent.mcp.paste.label')"
        :placeholder="pastePlaceholder"
      />
      <AgShareField v-if="all" v-model="pasteShare" />
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
  AGENT_SHARED_OWNER,
  type AgentShare,
  deleteAgentMcpServer,
  importAgentMcpServers,
  listAgentMcpServers,
  setAgentMcpDefault,
  testAgentMcpServer,
} from '@/api/agent'
import type { AgentMcpServer, AgentUser } from '@/types/v1/agent'
import AgConfirm from '@/views/agent/components/AgConfirm.vue'
import AgDialog from '@/views/agent/components/AgDialog.vue'
import AgManageHeader from '@/views/agent/components/AgManageHeader.vue'
import AgShareField from '@/views/agent/components/AgShareField.vue'
import AgSwitch from '@/views/agent/components/AgSwitch.vue'
import { useAgentAll } from '@/views/agent/composables/useAgentAll'
import { useAgentOverview } from '@/views/agent/composables/useAgentOverview'
import { relTime } from '@/views/agent/utils/format'
import McpFormDialog from './McpFormDialog.vue'

defineOptions({ name: 'AgentManageMcpView' })

const { t } = useI18n()
/** 在 Agent 管理页里：所有人的服务器 */
const all = useAgentAll()
const { overview, refresh } = useAgentOverview()

// —— 列表 ——
const servers = ref<AgentMcpServer[]>([])
const users = ref<AgentUser[]>([])
const loaded = ref(false)
const load = async () => {
  try {
    const { data } = await listAgentMcpServers(all)
    servers.value = data.servers || []
    users.value = data.users || []
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
    const u = all ? users.value.find((x) => x.id === s.userId) : undefined
    const owner = u ? u.name || u.username : t('agent.admin.goneUser')
    const shared = s.userId === AGENT_SHARED_OWNER
    return { s, tag: http ? 'HTTP' : 'stdio', pill, note, cmd, owner, shared, ...shareLabels(s, shared) }
  }),
)

// —— 共享的 ——
const userName = (id: string) => {
  const u = users.value.find((x) => x.id === id)
  return u ? u.name || u.username : t('agent.admin.goneUser')
}
/** 共享的服务器行上的标记：Agent 管理页里写共享给谁，自己的列表里写「管理员共享」 */
const shareLabels = (s: AgentMcpServer, shared: boolean) => {
  if (!shared) return { shareTag: '', scope: '' }
  if (!all) return { shareTag: t('agent.share.tagSharedToMe'), scope: t('agent.share.sharedReadonly') }
  const ids = s.shareUserIds || []
  if (s.share !== 'users') return { shareTag: t('agent.share.tagAll'), scope: '' }
  return { shareTag: t('agent.share.tagUsers', { n: ids.length }), scope: ids.map(userName).join(t('agent.share.sep')) }
}
// Agent 管理页里分两组：共享的在前；自己的列表不分组
const groups = computed(() => {
  if (!all) return [{ key: 'mine', title: '', rows: rows.value, empty: t('agent.mcp.empty') }]
  return [
    { key: 'shared', title: t('agent.share.sharedTitle'), rows: rows.value.filter((r) => r.shared), empty: t('agent.share.noShared') },
    { key: 'personal', title: t('agent.share.personalTitle'), rows: rows.value.filter((r) => !r.shared), empty: t('agent.mcp.empty') },
  ]
})

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

// —— 添加 / 编辑：Agent 管理页里加的是共享的 ——
const formOpen = ref(false)
const formServer = ref<AgentMcpServer | null>(null)
const formShared = ref(false)
const openForm = (s: AgentMcpServer | null) => {
  formServer.value = s
  formShared.value = s ? s.userId === AGENT_SHARED_OWNER : all
  formOpen.value = true
}
const onSaved = async () => {
  await reloadAll()
}

// —— 粘贴 JSON 导入：Agent 管理页里导入成共享的 ——
const pasteOpen = ref(false)
const pasteText = ref('')
const pasting = ref(false)
const pasteShare = ref<AgentShare>({ share: 'all', shareUserIds: [] })
const openPaste = () => {
  pasteShare.value = { share: 'all', shareUserIds: [] }
  pasteOpen.value = true
}
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
  if (all && pasteShare.value.share === 'users' && !pasteShare.value.shareUserIds.length) {
    feedback.warning(t('agent.share.needUsers'))
    return
  }
  pasting.value = true
  try {
    const { data } = await importAgentMcpServers(pasteText.value, all ? pasteShare.value : undefined)
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
// 管理员共享给自己的：只能用，行尾只有一把锁（默认开启的写在旁边）
.ag-mcp-lock {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: var(--m-ink2);
  white-space: nowrap;
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
