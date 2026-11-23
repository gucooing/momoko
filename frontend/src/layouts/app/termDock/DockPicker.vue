<!-- 新建终端的选择列表：本机终端（shell:use）+ SSH 主机（ssh:manage）。「+」菜单和空窗口共用。
     本机终端还能直接开在某个目录：Code 页开着会话时「在当前会话的目录打开」；能用 Agent 的再列出自己的工作区。 -->
<template>
  <div class="tdp">
    <template v-if="canLocal">
      <button type="button" class="ag-mi" role="menuitem" @click="pickLocal()">
        <UIcon name="i-lucide-square-terminal" />
        <span class="ag-ellipsis">{{ t('termDock.local') }}</span>
      </button>
      <button
        v-if="sessionDir.cwd"
        type="button"
        class="ag-mi"
        role="menuitem"
        :title="sessionDir.name ? `${sessionDir.name} · ${sessionDir.cwd}` : sessionDir.cwd"
        @click="pickLocal(sessionDir.cwd)"
      >
        <UIcon name="i-lucide-folder-code" />
        <span class="tdp__name ag-ellipsis">{{ t('termDock.here') }}</span>
        <span class="tdp__sub ag-muted ag-mono ag-ellipsis">{{ folderName(sessionDir.cwd) }}</span>
      </button>
      <template v-if="workspaces.length">
        <div class="ag-mi-label">{{ t('termDock.workspaces') }}</div>
        <div class="tdp__list tdp__list--ws">
          <button v-for="w in workspaces" :key="w.id" type="button" class="ag-mi" role="menuitem" :title="w.path" @click="pickLocal(w.path)">
            <UIcon name="i-lucide-folder" />
            <span class="tdp__name ag-ellipsis">{{ w.name }}</span>
            <span class="tdp__sub ag-muted ag-mono ag-ellipsis">{{ w.path }}</span>
          </button>
        </div>
      </template>
    </template>
    <template v-if="canSsh">
      <div v-if="canLocal" class="ag-mi-sep" />
      <div class="ag-mi-label">{{ t('termDock.sshHosts') }}</div>
      <div class="tdp__list">
        <button v-for="h in hosts" :key="h.id" type="button" class="ag-mi" role="menuitem" :title="`${h.username}@${h.host}:${h.port}`" @click="pickSsh(h)">
          <UIcon name="i-lucide-server" />
          <span class="tdp__name ag-ellipsis">{{ h.name }}</span>
          <span class="tdp__sub ag-muted ag-mono ag-ellipsis">{{ h.username }}@{{ h.host }}</span>
        </button>
      </div>
      <div v-if="loading && !hosts.length" class="tdp__note ag-muted">{{ t('termDock.loading') }}</div>
      <div v-else-if="!hosts.length" class="tdp__note ag-muted">{{ t('termDock.noHosts') }}</div>
      <RouterLink to="/openssh/management" class="ag-mi" role="menuitem" @click="emit('pick')">
        <UIcon name="i-lucide-settings-2" />
        <span class="ag-ellipsis">{{ t('termDock.manageHosts') }}</span>
      </RouterLink>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { listAgentWorkspaces } from '@/api/agent'
import { getSshHosts } from '@/api/openssh'
import { SHELL_PERMISSION } from '@/api/shell'
import { PERM } from '@/config/permission'
import type { AgentWorkspace } from '@/types/v1/agent'
import type { SSHHostInfo } from '@/types/v1/openssh'
import { addTab, folderName, sessionDir } from './state'

defineOptions({ name: 'DockPicker' })

const emit = defineEmits<{ pick: [] }>()

const { t } = useI18n()
const menuStore = useMenuStore()

const canLocal = computed(() => menuStore.hasButtonPermission(SHELL_PERMISSION))
const canSsh = computed(() => menuStore.hasButtonPermission('ssh:manage'))
const canAgent = computed(() => menuStore.hasButtonPermission(PERM.AGENT_USE))

// 自己的工作区（目录已不在的、还在克隆或克隆失败的不列）；取不到就不显示这一组
const workspaces = ref<AgentWorkspace[]>([])
const loadWorkspaces = async () => {
  if (!canLocal.value || !canAgent.value) return
  try {
    const { data } = await listAgentWorkspaces()
    workspaces.value = (data?.workspaces || []).filter((w) => w.path && !w.missing && !w.cloneStatus)
  } catch {
    workspaces.value = []
  }
}

const hosts = ref<SSHHostInfo[]>([])
const loading = ref(false)

const loadHosts = async () => {
  if (!canSsh.value) return
  loading.value = true
  try {
    const { data } = await getSshHosts({ page: 1, pageSize: 50 })
    hosts.value = data?.infos || []
  } catch {
    hosts.value = []
  } finally {
    loading.value = false
  }
}

/** cwd：在这个目录打开；不给就是主目录 */
const pickLocal = (cwd = '') => {
  addTab('local', { cwd })
  emit('pick')
}

const pickSsh = (h: SSHHostInfo) => {
  addTab('ssh', { hostId: h.id, label: `${h.username}@${h.name || h.host}` })
  emit('pick')
}

onMounted(() => {
  void loadHosts()
  void loadWorkspaces()
})
</script>

<style scoped lang="scss">
.tdp {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.tdp__list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 240px;
  overflow-y: auto;
}
// 工作区和 SSH 主机可能同时很长，工作区这组矮一点
.tdp__list--ws {
  max-height: 180px;
}
.tdp__name {
  flex: 0 1 auto;
}
.tdp__sub {
  flex: 1 1 0;
  margin-left: auto;
  font-size: 11.5px;
  text-align: right;
}
.tdp__note {
  padding: 4px 10px 6px;
  font-size: 12px;
}
</style>
