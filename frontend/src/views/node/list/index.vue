<!-- 节点列表：主控经 gRPC 接入的远程节点。API Key 在目标节点上创建（只作签名种子，不在网络上传输），
     其授权范围决定主控在该节点上可见的页面与操作。行内：切换 / 测试 / 编辑 / 删除。 -->
<template>
  <div class="node-page">
    <PageHeader :title="t('node.list.title')" :description="t('node.list.pageDesc')">
      <template #actions>
        <UButton color="neutral" variant="soft" icon="i-lucide-rotate-cw" @click="getList">
          {{ t('node.list.refresh') }}
        </UButton>
        <UButton v-if="canManage" color="primary" icon="i-lucide-plus" @click="nodeCreateRef?.showDialog()">
          {{ t('node.list.add') }}
        </UButton>
      </template>
    </PageHeader>

    <div class="node-guide">
      <UIcon name="i-lucide-info" class="node-guide__icon" />
      <ol class="node-guide__steps">
        <li>{{ t('node.list.guideStep1') }}</li>
        <li>{{ t('node.list.guideStep2') }}</li>
        <li>{{ t('node.list.guideStep3') }}</li>
      </ol>
    </div>

    <FilterBar @search="getList" @reset="reset">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('node.list.keywords') }}</label>
          <input
            v-model="keywords"
            class="app-input"
            :placeholder="t('node.list.keywordsPlaceholder')"
            @keyup.enter="getList"
          />
        </div>
      </template>
    </FilterBar>

    <div class="node-page__body">
      <span class="node-page__hint">{{ t('system.common.total', { total: list.length }) }}</span>

      <DataTable
        v-if="!menuStore.isMobile"
        :columns="columns"
        :rows="tableRows"
        row-key="id"
        :loading="loading"
        :empty-text="t('node.list.noData')"
      >
        <template #cell-name="{ row }">
          <div class="node-cell">
            <span class="node-cell__title">
              {{ asNode(row).name }}
              <StatusPill v-if="asNode(row).id === nodeStore.nodeId" variant="primary" :label="t('node.list.current')" />
            </span>
            <span v-if="asNode(row).remark" class="node-cell__sub">{{ asNode(row).remark }}</span>
          </div>
        </template>
        <template #cell-address="{ row }">
          <div class="node-cell">
            <span class="node-mono">{{ asNode(row).address }}</span>
            <span class="node-cell__sub">{{ asNode(row).tls ? (asNode(row).tlsSkipVerify ? t('node.list.tlsSelfSigned') : 'TLS') : t('node.list.plaintext') }}</span>
          </div>
        </template>
        <template #cell-status="{ row }">
          <div class="node-cell">
            <StatusPill :variant="stateVariant(asNode(row))" :label="stateLabel(asNode(row))" />
            <span class="node-cell__sub" :title="asNode(row).status?.message">
              {{ statusCaption(asNode(row)) }}
            </span>
          </div>
        </template>
        <template #cell-remote="{ row }">
          <div v-if="asNode(row).status?.remote" class="node-cell">
            <span>{{ asNode(row).status?.remote?.hostname }}</span>
            <span class="node-cell__sub">
              {{ asNode(row).status?.remote?.version }} · {{ asNode(row).status?.remote?.os }}/{{ asNode(row).status?.remote?.arch }}
            </span>
          </div>
          <span v-else class="node-cell__sub">—</span>
        </template>
        <template #cell-permissions="{ row }">
          <NodePermissionChips v-if="asNode(row).status?.remote" :permissions="asNode(row).status?.remote?.permissions" />
          <span v-else class="node-cell__sub">—</span>
        </template>
        <template #cell-keyId="{ row }">
          <span class="node-mono node-cell__sub">{{ asNode(row).keyId }}</span>
        </template>
        <template #cell-operation="{ row }">
          <ActionMenu :items="rowActions(asNode(row))" @select="(key) => onRowAction(key, asNode(row))" />
        </template>
      </DataTable>

      <template v-else>
        <div v-if="loading && !list.length" class="node-cards">
          <div v-for="i in 3" :key="i" class="node-skeleton" />
        </div>
        <EmptyState
          v-else-if="!list.length"
          icon="HOutline:ServerStackIcon"
          :title="t('node.list.noData')"
          :description="t('node.list.emptyDesc')"
        />
        <div v-else class="node-cards">
          <EntityCard v-for="node in list" :key="node.id">
            <template #title>{{ node.name }}</template>
            <template #status>
              <StatusPill :variant="stateVariant(node)" :label="stateLabel(node)" />
            </template>
            <template #meta>
              <span class="node-mono">{{ node.address }}</span>
              <span v-if="node.status?.remote">{{ node.status.remote.hostname }} · {{ node.status.remote.version }}</span>
              <span v-else-if="node.status?.message" class="node-card__full">{{ node.status.message }}</span>
              <NodePermissionChips v-if="node.status?.remote" class="node-card__full" :permissions="node.status.remote.permissions" />
            </template>
            <template #footer>
              <span>{{ statusCaption(node) }}</span>
              <ActionMenu :items="rowActions(node)" @select="(key) => onRowAction(key, node)" />
            </template>
          </EntityCard>
        </div>
      </template>
    </div>

    <NodeCreate ref="nodeCreateRef" @refresh="onFormRefresh" />

    <FormDialog v-model="testDialog.open" :title="t('node.list.testTitle', { name: testDialog.name })" :width="520" :show-footer="false">
      <div v-if="testDialog.loading" class="node-test-loading">
        <span class="node-spinner" />
        <span>{{ t('node.list.testing') }}</span>
      </div>
      <NodeTestResult v-else-if="testDialog.status" :status="testDialog.status" />
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { deleteNode, listNodes, testNode } from '@/api/node'
import { NodeState, type NodeInfo, type NodeStatus } from '@/types/v1/node'
import { Dialog } from '@/utils/dialog'
import { formatDateTime } from '@/utils/file'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import NodeCreate from '@/views/node/list/create.vue'
import NodeTestResult from '@/views/node/list/testResult.vue'
import NodePermissionChips from '@/views/node/components/permissionChips.vue'

defineOptions({ name: 'NodeListView' })

const menuStore = useMenuStore()
const nodeStore = useNodeStore()
const { t } = useI18n()
const nodeCreateRef = useTemplateRef<InstanceType<typeof NodeCreate> | null>('nodeCreateRef')

const loading = ref(false)
const list = ref<NodeInfo[]>([])
const keywords = ref('')
const testDialog = ref({ open: false, loading: false, name: '', status: null as NodeStatus | null })

const canManage = computed(() => menuStore.hasButtonPermission('node:manage'))
const tableRows = computed(() => list.value as unknown as Record<string, unknown>[])
const asNode = (row: Record<string, unknown>) => row as unknown as NodeInfo

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('node.list.name'), minWidth: 150 },
  { key: 'address', title: t('node.list.address'), minWidth: 170 },
  { key: 'status', title: t('node.list.status'), minWidth: 150 },
  { key: 'remote', title: t('node.list.nodeInfo'), minWidth: 170 },
  { key: 'permissions', title: t('node.list.grantedPermissions'), minWidth: 220, ellipsis: false },
  { key: 'keyId', title: 'Key ID', minWidth: 150 },
  { key: 'operation', title: t('node.list.operation'), width: 80, align: 'center' },
])

const stateVariant = (node: NodeInfo) => {
  const state = node.status?.state
  if (state === NodeState.NodeState_Online) return 'success'
  if (state === NodeState.NodeState_Offline) return 'error'
  if (state === NodeState.NodeState_Unauthorized) return 'warning'
  return 'neutral'
}
const stateLabel = (node: NodeInfo) => {
  const state = node.status?.state
  if (state === NodeState.NodeState_Online) return t('node.state.online')
  if (state === NodeState.NodeState_Offline) return t('node.state.offline')
  if (state === NodeState.NodeState_Unauthorized) return t('node.state.unauthorized')
  return t('node.state.unknown')
}
const statusCaption = (node: NodeInfo) => {
  const status = node.status
  if (!status?.checkTime) return t('node.list.notChecked')
  const time = formatDateTime(status.checkTime)
  if (status.state === NodeState.NodeState_Online) return `${status.latencyMs} ms · ${time}`
  return status.message || time
}

const rowActions = (node: NodeInfo): ActionMenuItem[] => [
  {
    key: 'switch',
    label: t('node.list.switchTo'),
    icon: 'i-lucide-arrow-right-left',
    hidden: node.id === nodeStore.nodeId,
  },
  { key: 'test', label: t('node.list.test'), icon: 'i-lucide-plug-zap', hidden: !canManage.value },
  { key: 'edit', label: t('node.list.edit'), icon: 'i-lucide-pencil', hidden: !canManage.value },
  { key: 'delete', label: t('node.list.delete'), icon: 'i-lucide-trash-2', danger: true, hidden: !canManage.value },
]

const getList = async () => {
  loading.value = true
  try {
    const { data } = await listNodes({ keywords: keywords.value.trim() || undefined })
    list.value = data?.infos || []
    if (!keywords.value.trim()) nodeStore.nodes = list.value
  } finally {
    loading.value = false
  }
}

const reset = () => {
  keywords.value = ''
  getList()
}

const runTest = async (node: NodeInfo) => {
  testDialog.value = { open: true, loading: true, name: node.name, status: null }
  try {
    const { data } = await testNode({ id: node.id })
    testDialog.value.status = data?.status || null
    void getList()
  } catch {
    testDialog.value.open = false
  } finally {
    testDialog.value.loading = false
  }
}

const onRowAction = (key: string, node: NodeInfo) => {
  if (key === 'switch') nodeStore.switchNode(node.id)
  else if (key === 'test') runTest(node)
  else if (key === 'edit') nodeCreateRef.value?.showDialog(node)
  else if (key === 'delete') confirmDelete(node)
}

const confirmDelete = (node: NodeInfo) => {
  Dialog.confirm({
    title: t('node.list.deleteTitle'),
    content: t('node.list.deleteContent', { name: node.name }),
    confirmText: t('node.list.confirm'),
    cancelText: t('node.list.cancel'),
    onConfirm: async () => {
      await deleteNode({ id: node.id })
      feedback.success(t('node.list.deleteSuccess'))
      // 删除的是当前节点：回到本机
      if (node.id === nodeStore.nodeId) {
        nodeStore.switchNode('')
        return
      }
      getList()
    },
  })
}

const onFormRefresh = () => {
  getList()
  // 新增 / 编辑后后台会异步探测一次，稍后刷新状态
  setTimeout(() => void getList(), 2500)
}

onMounted(() => {
  getList()
})
</script>

<style scoped lang="scss">
.node-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.node-guide {
  display: flex;
  gap: 10px;
  padding: 10px 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  background: color-mix(in srgb, var(--el-color-primary) 5%, transparent);
  font-size: 0.8125rem;
  line-height: 1.6;
  color: var(--el-text-color-regular);
}
.node-guide__icon {
  width: 16px;
  height: 16px;
  margin-top: 3px;
  flex-shrink: 0;
  color: var(--el-color-primary);
}
.node-guide__steps {
  margin: 0;
  padding-left: 1.1rem;
  list-style: decimal;
}
.node-page__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.node-page__hint {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.node-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.node-cell__title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.node-cell__sub {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.node-mono {
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.8125rem;
}
.node-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.node-card__full {
  flex-basis: 100%;
}
.node-skeleton {
  height: 120px;
  border-radius: var(--app-radius);
  background: linear-gradient(
    100deg,
    var(--el-fill-color-light) 30%,
    var(--el-fill-color) 50%,
    var(--el-fill-color-light) 70%
  );
  background-size: 200% 100%;
  animation: node-shimmer 1.4s ease-in-out infinite;
}
@keyframes node-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
.node-test-loading {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px 4px;
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
}
.node-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--el-border-color);
  border-top-color: var(--el-color-primary);
  border-radius: 999px;
  animation: node-spin 0.8s linear infinite;
}
@keyframes node-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
