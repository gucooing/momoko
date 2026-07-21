<!-- API Key（重写 · P1 列表）：PageHeader + FilterBar(名称) + 计数条 + DataTable / 移动卡 + Pagination。
     行内 编辑/复制/刷新（ActionMenu）；复制走 FormDialog 只读展示。保留 list/copy/refresh/create/update 契约。 -->
<template>
  <div class="key-page">
    <PageHeader :title="t('node.key.title')" :description="t('node.key.pageDesc')">
      <template #actions>
        <UButton color="neutral" variant="soft" icon="i-lucide-rotate-cw" @click="getList">
          {{ t('node.key.refresh') }}
        </UButton>
        <UButton color="primary" icon="i-lucide-plus" @click="apiKeyCreateRef?.showDialog()">
          {{ t('node.key.add') }}
        </UButton>
      </template>
    </PageHeader>

    <FilterBar @search="reload" @reset="reset">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('node.key.name') }}</label>
          <input
            v-model="queryForm.keywords"
            class="app-input"
            :placeholder="t('node.key.namePlaceholder')"
            @keyup.enter="reload"
          />
        </div>
      </template>
    </FilterBar>

    <div class="key-page__body">
      <div class="key-page__bar">
        <span class="key-page__hint">{{ t('system.common.total', { total: pagination.total }) }}</span>
      </div>

      <!-- 桌面：表格 -->
      <DataTable
        v-if="!menuStore.isMobile"
        :columns="columns"
        :rows="list"
        row-key="id"
        :loading="loading"
        :empty-text="t('node.key.noData')"
      >
        <template #cell-apiKey="{ row }">
          <span class="key-mono">{{ row.apiKey }}</span>
        </template>
        <template #cell-expiresAt="{ row }">
          <StatusPill v-if="!row.expiresAt" variant="success" :label="t('node.key.permanent')" />
          <span v-else>{{ formatTime(row.expiresAt) }}</span>
        </template>
        <template #cell-createTime="{ row }">{{ formatTime(row.createTime) }}</template>
        <template #cell-updateTime="{ row }">{{ formatTime(row.updateTime) }}</template>
        <template #cell-operation="{ row }">
          <ActionMenu :items="rowActions" @select="(key) => onRowAction(key, row)" />
        </template>
      </DataTable>

      <!-- 移动：卡片 -->
      <template v-else>
        <div v-if="loading" class="key-cards">
          <div v-for="i in 4" :key="i" class="key-skeleton" />
        </div>
        <EmptyState
          v-else-if="!list.length"
          icon="HOutline:KeyIcon"
          :title="t('node.key.noData')"
          :description="t('node.key.emptyDesc')"
        />
        <div v-else class="key-cards">
          <EntityCard v-for="row in list" :key="row.id">
            <template #title>{{ row.name }}</template>
            <template #status>
              <StatusPill v-if="!row.expiresAt" variant="success" :label="t('node.key.permanent')" />
            </template>
            <template #meta>
              <span class="key-card__full key-mono">{{ row.apiKey }}</span>
              <span v-if="row.expiresAt">{{ t('node.key.expiresAt') }}: {{ formatTime(row.expiresAt) }}</span>
            </template>
            <template #footer>
              <span>{{ t('node.key.createTime') }}: {{ formatTime(row.createTime) }}</span>
              <ActionMenu :items="rowActions" @select="(key) => onRowAction(key, row)" />
            </template>
          </EntityCard>
        </div>
      </template>

      <Pagination
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        @change="getList"
      />
    </div>

    <ApiKeyCreate ref="apiKeyCreateRef" @refresh="onFormRefresh" />

    <!-- 复制 API Key（完整值仅一次） -->
    <FormDialog v-model="copyDialogOpen" :title="t('node.key.copyTitle')" :width="560">
      <div class="key-copy">
        <div class="key-copy__warn">{{ t('node.key.copyWarning') }}</div>
        <div class="key-copy__inline">
          <input class="app-input key-mono" :value="copyKeyValue" readonly />
          <AppIconButton
            icon="HOutline:ClipboardDocumentIcon"
            :label="t('node.key.copy')"
            :box="32"
            @click="doCopy"
          />
        </div>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('node.key.close') }}</UButton>
        <UButton color="primary" icon="i-lucide-copy" @click="doCopy">{{ t('node.key.copy') }}</UButton>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { copyAPIKey, listAPIKeys, refreshAPIKey } from '@/api/node'
import type { APIKeyInfo } from '@/types/v1/node'
import { Dialog } from '@/utils/dialog'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import ApiKeyCreate from '@/views/node/key/create.vue'

defineOptions({ name: 'ApiKeyView' })

const menuStore = useMenuStore()
const { t } = useI18n()
const apiKeyCreateRef = useTemplateRef<InstanceType<typeof ApiKeyCreate> | null>('apiKeyCreateRef')

const loading = ref(false)
const list = ref<APIKeyInfo[]>([])
const copyDialogOpen = ref(false)
const copyKeyValue = ref('')

const queryForm = ref({ keywords: '' })
const pagination = ref({ page: 1, pageSize: 10, total: 0 })

const formatTime = (value: unknown): string => {
  if (!value) return '—'
  const d = new Date(value as string | Date)
  if (Number.isNaN(d.getTime())) return '—'
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('node.key.name'), minWidth: 150 },
  { key: 'apiKey', title: 'API Key', minWidth: 260 },
  { key: 'expiresAt', title: t('node.key.expiresAt'), minWidth: 170 },
  { key: 'createTime', title: t('node.key.createTime'), width: 170 },
  { key: 'updateTime', title: t('node.key.updateTime'), width: 170 },
  { key: 'operation', title: t('node.key.operation'), width: 80, align: 'center' },
])

const rowActions = computed<ActionMenuItem[]>(() => [
  { key: 'edit', label: t('node.key.edit'), icon: 'HOutline:PencilSquareIcon' },
  { key: 'copy', label: t('node.key.copy'), icon: 'HOutline:ClipboardDocumentIcon' },
  { key: 'refresh', label: t('node.key.refresh'), icon: 'HOutline:ArrowPathIcon' },
])

const getList = async () => {
  loading.value = true
  try {
    const { data: res } = await listAPIKeys({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keywords: queryForm.value.keywords || undefined,
    })
    list.value = res?.infos || []
    pagination.value.total = Number(res?.total || 0)
    pagination.value.page = Number(res?.page || pagination.value.page)
    pagination.value.pageSize = Number(res?.pageSize || pagination.value.pageSize)
  } finally {
    loading.value = false
  }
}

const reload = () => {
  pagination.value.page = 1
  getList()
}
const reset = () => {
  queryForm.value.keywords = ''
  pagination.value.page = 1
  getList()
}

const onRowAction = (key: string, row: Record<string, unknown>) => {
  const record = list.value.find((x) => x.id === String(row.id))
  if (!record) return
  if (key === 'edit') apiKeyCreateRef.value?.showDialog(record)
  else if (key === 'copy') openCopy(record)
  else if (key === 'refresh') openRefresh(record)
}

const openCopy = async (row: APIKeyInfo) => {
  const { data: res } = await copyAPIKey({ id: row.id })
  copyKeyValue.value = res?.info?.apiKey || ''
  copyDialogOpen.value = true
}

const openRefresh = (row: APIKeyInfo) => {
  Dialog.confirm({
    title: t('node.key.refreshTitle'),
    content: t('node.key.refreshContent'),
    confirmText: t('node.key.confirm'),
    cancelText: t('node.key.cancel'),
    onConfirm: async () => {
      await refreshAPIKey({ id: row.id })
      feedback.success(t('node.key.refreshSuccess'))
      getList()
    },
  })
}

const doCopy = async () => {
  await navigator.clipboard.writeText(copyKeyValue.value)
  feedback.success(t('node.key.copied'))
}

const onFormRefresh = (type: 'create' | 'update') => {
  if (type === 'create') pagination.value.page = 1
  getList()
}

onMounted(() => {
  getList()
})
</script>

<style scoped lang="scss">
.key-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.key-page__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.key-page__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.key-page__hint {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.key-mono {
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.8125rem;
}

/* 复制弹窗 */
.key-copy {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.key-copy__warn {
  padding: 8px 12px;
  border-radius: var(--app-radius-sm);
  background: color-mix(in srgb, var(--el-color-warning, #f59e0b) 12%, transparent);
  color: var(--el-color-warning, #f59e0b);
  font-size: 0.8125rem;
  line-height: 1.5;
}
.key-copy__inline {
  display: flex;
  align-items: center;
  gap: 8px;
}
.key-copy__inline .app-input {
  flex: 1;
  min-width: 0;
}

/* 移动卡片 */
.key-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.key-card__full {
  flex-basis: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.key-skeleton {
  height: 120px;
  border-radius: var(--app-radius);
  background: linear-gradient(
    100deg,
    var(--el-fill-color-light) 30%,
    var(--el-fill-color) 50%,
    var(--el-fill-color-light) 70%
  );
  background-size: 200% 100%;
  animation: key-shimmer 1.4s ease-in-out infinite;
}
@keyframes key-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
</style>
