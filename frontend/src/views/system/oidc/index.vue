<!-- OIDC 客户端（重写 · P1 列表/CRUD）：PageHeader(OIDC配置 + 生成客户端) + FilterBar + DataTable / 移动卡片 + Pagination。
     子弹窗：configDialog(服务端配置) / clientForm(创建编辑) / 内联「客户端配置」只读展示（完整 secret 仅一次）。
     行内 编辑/刷新密钥/删除（Dialog.info 确认）。文案走 system.oidc.*。 -->
<template>
  <div class="oidc-page">
    <PageHeader :title="t('system.oidc.title')" :description="t('system.oidc.pageDesc')">
      <template #actions>
        <UButton color="neutral" variant="soft" icon="i-lucide-settings" @click="openConfig">
          {{ t('system.oidc.openConfig') }}
        </UButton>
        <UButton
          v-permission="[PERM.OIDC_EDIT]"
          color="primary"
          icon="i-lucide-plus"
          @click="openClient()"
        >
          {{ t('system.oidc.createClient') }}
        </UButton>
      </template>
    </PageHeader>

    <FilterBar @search="reload" @reset="reset">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('system.oidc.clientName') }}</label>
          <input
            v-model="queryForm.keywords"
            class="app-input"
            :placeholder="t('system.oidc.clientNamePlaceholder')"
            @keyup.enter="reload"
          />
        </div>
      </template>
    </FilterBar>

    <div class="oidc-page__body">
      <div class="oidc-page__bar">
        <span class="oidc-page__bar-hint">{{ t('system.common.total', { total: pagination.total }) }}</span>
      </div>

      <!-- 桌面：表格 -->
      <DataTable
        v-if="!menuStore.isMobile"
        :columns="columns"
        :rows="clients"
        row-key="id"
        :loading="loading"
      >
        <template #cell-clientSecret="{ row }">
          <span class="oidc-mono">{{ row.clientSecret }}</span>
        </template>
        <template #cell-active="{ row }">
          <StatusPill
            :variant="row.active ? 'success' : 'neutral'"
            :label="row.active ? t('system.common.enabled') : t('system.common.inactive')"
          />
        </template>
        <template #cell-redirectUris="{ row }">{{ joinArr(row.redirectUris, ', ') }}</template>
        <template #cell-scopes="{ row }">{{ joinArr(row.scopes, ' ') }}</template>
        <template #cell-operation="{ row }">
          <ActionMenu v-if="canEdit" :items="rowActions" @select="(key) => onRowAction(key, row)" />
          <span v-else class="text-dim">—</span>
        </template>
      </DataTable>

      <!-- 移动：卡片 -->
      <template v-else>
        <div v-if="loading" class="oidc-cards">
          <div v-for="i in 4" :key="i" class="oidc-skeleton" />
        </div>
        <EmptyState
          v-else-if="!clients.length"
          icon="HOutline:KeyIcon"
          :title="t('system.oidc.emptyTitle')"
          :description="t('system.oidc.emptyDesc')"
        />
        <div v-else class="oidc-cards">
          <EntityCard v-for="row in clients" :key="row.id">
            <template #title>{{ row.name }}</template>
            <template #status>
              <StatusPill
                :variant="row.active ? 'success' : 'neutral'"
                :label="row.active ? t('system.common.enabled') : t('system.common.inactive')"
              />
            </template>
            <template #meta>
              <span class="oidc-card__full oidc-mono">{{ row.clientId }}</span>
              <span class="oidc-card__full">{{ t('system.oidc.colScopes') }}: {{ row.scopes.join(' ') }}</span>
              <span class="oidc-card__full oidc-card__uris">{{ row.redirectUris.join(', ') }}</span>
            </template>
            <template #footer>
              <span>{{ fmtTime(row.createTime) }}</span>
              <ActionMenu v-if="canEdit" :items="rowActions" @select="(key) => onRowAction(key, row)" />
            </template>
          </EntityCard>
        </div>
      </template>

      <Pagination
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        @change="loadClients"
      />
    </div>

    <!-- 服务端配置 / 创建编辑 -->
    <OIDCConfigDialog ref="configRef" :can-edit="canEdit" @saved="onConfigSaved" />
    <OIDCClientForm ref="clientFormRef" @refresh="loadClients" @reveal="openReveal" />

    <!-- 客户端配置（完整 secret 仅一次） -->
    <FormDialog v-model="revealVisible" :title="t('system.oidc.revealTitle')" :width="720">
      <div class="oidc-reveal">
        <div class="oidc-reveal__warn">
          {{ t('system.oidc.secretOnceWarn') }}
        </div>
        <div class="oidc-reveal__grid">
          <div v-for="item in clientConfigItems" :key="item.label" class="app-field">
            <label class="app-label">{{ item.label }}</label>
            <div class="oidc-config__inline">
              <textarea
                v-if="item.multiline"
                class="app-textarea"
                :value="item.value"
                readonly
                rows="2"
              />
              <input v-else class="app-input" :value="item.value" readonly />
              <AppIconButton
                icon="HOutline:ClipboardDocumentIcon"
                :label="t('common.copy')"
                :box="32"
                @click="copyText(item.value)"
              />
            </div>
          </div>
        </div>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('common.close') }}</UButton>
        <UButton color="primary" icon="i-lucide-copy" @click="copyText(clientConfigText)">
          {{ t('system.oidc.copyAll') }}
        </UButton>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { listOIDCClients, deleteOIDCClient, refreshOIDCClientSecret, getOIDCConfig } from '@/api/oidc'
import { PERM } from '@/config/permission'
import { Dialog } from '@/utils/dialog'
import type { OIDCClientInfo, OIDCConfig } from '@/types/v1/oidc'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import OIDCConfigDialog from '@/views/system/oidc/configDialog.vue'
import OIDCClientForm from '@/views/system/oidc/clientForm.vue'

defineOptions({ name: 'SystemOIDCView' })

const menuStore = useMenuStore()
const { t } = useI18n()

const configRef = useTemplateRef<InstanceType<typeof OIDCConfigDialog> | null>('configRef')
const clientFormRef = useTemplateRef<InstanceType<typeof OIDCClientForm> | null>('clientFormRef')

const canEdit = computed(() => menuStore.hasButtonPermission(PERM.OIDC_EDIT))

const loading = ref(false)
const clients = ref<OIDCClientInfo[]>([])
const config = ref<OIDCConfig | null>(null)
const queryForm = ref({ keywords: '' })
const pagination = ref({ page: 1, pageSize: 10, total: 0 })

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('system.oidc.colName'), minWidth: 140 },
  { key: 'clientId', title: t('system.oidc.colClientId'), minWidth: 220 },
  { key: 'clientSecret', title: t('system.oidc.colClientSecret'), minWidth: 160 },
  { key: 'active', title: t('system.common.status'), width: 90 },
  { key: 'redirectUris', title: t('system.oidc.colRedirectUris'), minWidth: 220 },
  { key: 'scopes', title: t('system.oidc.colScopes'), minWidth: 150 },
  { key: 'operation', title: t('system.common.operation'), width: 70, align: 'center' },
])

const rowActions = computed<ActionMenuItem[]>(() => [
  { key: 'edit', label: t('system.common.edit'), icon: 'HOutline:PencilSquareIcon' },
  { key: 'refresh', label: t('system.oidc.refreshSecret'), icon: 'HOutline:ArrowPathIcon' },
  { key: 'delete', label: t('system.common.delete'), icon: 'HOutline:TrashIcon', danger: true },
])

const joinArr = (v: unknown, sep: string) => (Array.isArray(v) ? v.join(sep) : '')
const fmtTime = (v: unknown) => (v ? dayjs(v as string | Date).format('YYYY-MM-DD HH:mm') : '—')

const loadClients = async () => {
  loading.value = true
  try {
    const { data } = await listOIDCClients({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keywords: queryForm.value.keywords || undefined,
    })
    clients.value = data?.clients || []
    pagination.value.total = Number(data?.total || 0)
  } finally {
    loading.value = false
  }
}

const reload = () => {
  pagination.value.page = 1
  loadClients()
}
const reset = () => {
  queryForm.value.keywords = ''
  pagination.value.page = 1
  loadClients()
}

const openConfig = () => configRef.value?.showDialog()
const onConfigSaved = (value: OIDCConfig) => {
  config.value = value
}

const openClient = (row?: OIDCClientInfo) => clientFormRef.value?.showDialog(row)

const onRowAction = (key: string, row: Record<string, unknown>) => {
  const client = clients.value.find((c) => c.id === String(row.id))
  if (!client) return
  if (key === 'edit') openClient(client)
  else if (key === 'refresh') refreshSecret(client)
  else if (key === 'delete') confirmDelete(client)
}

const refreshSecret = (client: OIDCClientInfo) => {
  Dialog.info({
    showCancelButton: true,
    content: t('system.oidc.refreshSecretConfirm'),
    confirmText: t('system.oidc.refresh'),
    cancelText: t('common.cancel'),
    onConfirm: async () => {
      const { data } = await refreshOIDCClientSecret({ id: client.id })
      if (data?.client) openReveal(data.client)
      loadClients()
    },
  })
}

const confirmDelete = (client: OIDCClientInfo) => {
  Dialog.info({
    showCancelButton: true,
    content: t('system.oidc.deleteConfirm', { name: client.name }),
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    onConfirm: async () => {
      await deleteOIDCClient({ id: client.id })
      feedback.success(t('common.deleteSuccess'))
      loadClients()
    },
  })
}

// —— 客户端配置只读展示（含完整 secret）——
const revealVisible = ref(false)
const revealClient = ref<OIDCClientInfo | null>(null)

const clientConfigItems = computed<{ label: string; value: string; multiline?: boolean }[]>(() => {
  const client = revealClient.value
  if (!client) return []
  const e = config.value?.endpoints
  return [
    { label: t('system.oidc.colName'), value: client.name },
    { label: t('system.oidc.colClientId'), value: client.clientId },
    { label: t('system.oidc.colClientSecret'), value: client.clientSecret },
    { label: t('system.oidc.colRedirectUris'), value: client.redirectUris.join('\n'), multiline: true },
    { label: t('system.oidc.colScopes'), value: client.scopes.join(' ') },
    { label: t('system.oidc.issuerUrl'), value: e?.issuerUrl || '' },
    { label: t('system.oidc.discoveryUrl'), value: e?.discoveryUrl || '' },
    { label: t('system.oidc.authorizeUrl'), value: e?.authorizeUrl || '' },
    { label: t('system.oidc.tokenUrl'), value: e?.tokenUrl || '' },
    { label: t('system.oidc.userinfoUrl'), value: e?.userinfoUrl || '' },
    { label: t('system.oidc.jwksUrl'), value: e?.jwksUrl || '' },
    { label: t('system.oidc.tokenAuthMethod'), value: e?.tokenEndpointAuthMethod || 'client_secret_post' },
    { label: t('system.oidc.signingAlgs'), value: e?.allowedIdTokenSigningAlgs || 'RS256,ES256,PS256' },
  ].filter((item) => item.value)
})
const clientConfigText = computed(() =>
  clientConfigItems.value.map((item) => `${item.label}: ${item.value}`).join('\n'),
)

const openReveal = (client: OIDCClientInfo) => {
  revealClient.value = client
  revealVisible.value = true
}

const copyText = async (value: string) => {
  await navigator.clipboard.writeText(value)
  feedback.success(t('system.oidc.copied'))
}

const loadConfig = async () => {
  const { data } = await getOIDCConfig()
  config.value = data?.config || null
}

onMounted(() => {
  loadClients()
  loadConfig()
})
</script>

<style scoped lang="scss">
.oidc-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.oidc-page__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.oidc-page__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.oidc-page__bar-hint {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.oidc-mono {
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.8125rem;
}
.text-dim {
  color: var(--el-text-color-placeholder);
}

/* 客户端配置只读展示 */
.oidc-reveal {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.oidc-reveal__warn {
  padding: 8px 12px;
  border-radius: var(--app-radius-sm);
  background: color-mix(in srgb, var(--el-color-warning, #f59e0b) 12%, transparent);
  color: var(--el-color-warning, #f59e0b);
  font-size: 0.8125rem;
}
.oidc-reveal__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.oidc-config__inline {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}
.oidc-config__inline .app-input,
.oidc-config__inline .app-textarea {
  flex: 1;
  min-width: 0;
}

/* 移动卡片 */
.oidc-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.oidc-card__full {
  flex-basis: 100%;
}
.oidc-card__uris {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--el-text-color-placeholder);
}
.oidc-skeleton {
  height: 120px;
  border-radius: var(--app-radius);
  background: linear-gradient(
    100deg,
    var(--el-fill-color-light) 30%,
    var(--el-fill-color) 50%,
    var(--el-fill-color-light) 70%
  );
  background-size: 200% 100%;
  animation: oidc-shimmer 1.4s ease-in-out infinite;
}
@keyframes oidc-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
@media (width <= 768px) {
  .oidc-reveal__grid {
    grid-template-columns: 1fr;
  }
}
</style>
