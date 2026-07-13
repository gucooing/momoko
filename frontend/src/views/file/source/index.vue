<!-- 文件来源（重写 · P1 列表）：PageHeader(新增) + FilterBar(关键词) + 内联计数 + DataTable/移动卡。
     行内 测试/编辑/删除；启用 AppSwitch；类型/直链胶囊。列表非分页（后端一次返回全部）。 -->
<template>
  <div class="src-page">
    <PageHeader :title="t('fileSource.title')" :description="t('fileSource.pageDesc')">
      <template #actions>
        <UButton color="primary" size="sm" icon="i-lucide-plus" @click="openCreate()">
          {{ t('fileSource.add') }}
        </UButton>
      </template>
    </PageHeader>

    <FilterBar @search="search" @reset="reset">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('fileSource.keyword') }}</label>
          <input
            v-model="keywords"
            class="app-input"
            :placeholder="t('fileSource.keywordPlaceholder')"
            @keyup.enter="search"
          />
        </div>
      </template>
    </FilterBar>

    <div class="src-page__body">
      <div class="src-page__bar">
        <span class="src-page__hint">
          {{ t('system.common.total', { total: items.length }) }}
          <template v-if="items.length">
            · <span class="src-page__dot src-page__dot--on" />{{ t('fileSource.enabled') }} {{ enabledCount }}
            · <span class="src-page__dot" />{{ t('fileSource.disabled') }} {{ items.length - enabledCount }}
          </template>
        </span>
      </div>

      <DataTable
        v-if="!menuStore.isMobile"
        :columns="columns"
        :rows="items"
        row-key="id"
        :loading="loading"
        :empty-text="t('fileSource.empty')"
      >
        <template #cell-name="{ row }">
          <span class="src-name">{{ row.name }}</span>
        </template>
        <template #cell-type="{ row }">
          <StatusPill :variant="typeVariant(String(row.type))" :label="typeLabel(String(row.type))" :dot="false" />
        </template>
        <template #cell-enabled="{ row }">
          <AppSwitch :model-value="!!row.enabled" @update:model-value="(v) => toggleEnabled(row, v)" />
        </template>
        <template #cell-redirect302="{ row }">
          <StatusPill
            v-if="hasRedirect(row)"
            variant="success"
            :label="t('fileSource.redirectOn')"
            :dot="false"
          />
          <span v-else class="src-muted">{{ t('fileSource.redirectOff') }}</span>
        </template>
        <template #cell-creatorName="{ row }">{{ row.creatorName || '—' }}</template>
        <template #cell-createTime="{ row }">{{ formatDateTime(row.createTime) }}</template>
        <template #cell-operation="{ row }">
          <ActionMenu :items="rowActions" @select="(key) => onRowAction(key, row)" />
        </template>
      </DataTable>

      <template v-else>
        <div v-if="loading" class="src-cards">
          <div v-for="i in 3" :key="i" class="src-skeleton" />
        </div>
        <EmptyState
          v-else-if="!items.length"
          icon="HOutline:CircleStackIcon"
          :title="t('fileSource.empty')"
          :description="t('fileSource.emptyDesc')"
        />
        <div v-else class="src-cards">
          <EntityCard v-for="row in items" :key="row.id" :title="row.name">
            <template #status>
              <StatusPill :variant="typeVariant(row.type)" :label="typeLabel(row.type)" :dot="false" />
            </template>
            <template #meta>
              <span v-if="hasRedirect(row)" class="src-card__redirect">{{ t('fileSource.redirect302') }}</span>
              <span>{{ row.creatorName || '—' }}</span>
              <span>{{ formatDateTime(row.createTime) }}</span>
            </template>
            <template #footer>
              <AppSwitch :model-value="!!row.enabled" @update:model-value="(v) => toggleEnabled(row, v)" />
              <div class="src-card__actions">
                <UButton color="neutral" variant="ghost" size="xs" @click="testExisting(row)">
                  {{ t('fileSource.test') }}
                </UButton>
                <ActionMenu :items="rowActions.filter((a) => a.key !== 'test')" @select="(key) => onRowAction(key, row)" />
              </div>
            </template>
          </EntityCard>
        </div>
      </template>
    </div>

    <SourceCreate ref="createRef" @refresh="getList" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import SourceCreate from '@/views/file/source/create.vue'
import { formatDateTime } from '@/utils/file'
import { Dialog } from '@/utils/dialog'
import {
  listFileSourcesRequest,
  updateFileSourceRequest,
  deleteFileSourceRequest,
  testFileSourceRequest,
} from '@/api/fileSource'
import type { FileSourceInfo, FileSourceConfig } from '@/types/v1/file'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'

defineOptions({ name: 'FileSourceView' })

const menuStore = useMenuStore()
const { t } = useI18n()
const createRef = useTemplateRef<InstanceType<typeof SourceCreate> | null>('createRef')

const loading = ref(false)
const items = ref<FileSourceInfo[]>([])
const keywords = ref('')

const enabledCount = computed(() => items.value.filter((i) => i.enabled).length)

const typeLabel = (type: string) =>
  type === 'oss'
    ? t('fileSource.typeOss')
    : type === 'ftp'
      ? t('fileSource.typeFtp')
      : type === 'webdav'
        ? t('fileSource.typeWebdav')
        : type
type PillVariant = 'success' | 'info' | 'warning' | 'primary'
const typeVariant = (type: string): PillVariant =>
  type === 'oss' ? 'primary' : type === 'ftp' ? 'warning' : 'success'
const hasRedirect = (row: Record<string, unknown>) => {
  const caps = row.caps as { presign?: boolean } | undefined
  return !!caps?.presign && !!row.redirect302
}

const emptyConfig = (): FileSourceConfig => ({
  endpoint: '',
  region: '',
  bucket: '',
  accessKey: '',
  secretKey: '',
  prefix: '',
  useSsl: true,
  pathStyle: false,
  host: '',
  port: 21,
  username: '',
  password: '',
  basePath: '',
  tls: false,
  url: '',
})

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('fileSource.name'), minWidth: 140 },
  { key: 'type', title: t('fileSource.type'), width: 120 },
  { key: 'enabled', title: t('fileSource.enabled'), width: 80 },
  { key: 'redirect302', title: t('fileSource.redirect302'), width: 110 },
  { key: 'creatorName', title: t('fileSource.creator'), width: 120 },
  { key: 'createTime', title: t('fileSource.createTime'), width: 170 },
  { key: 'operation', title: t('system.common.operation'), width: 80, align: 'center' },
])

const rowActions = computed<ActionMenuItem[]>(() => [
  { key: 'test', label: t('fileSource.test'), icon: 'HOutline:SignalIcon' },
  { key: 'edit', label: t('system.common.edit'), icon: 'HOutline:PencilSquareIcon' },
  { key: 'delete', label: t('system.common.delete'), icon: 'HOutline:TrashIcon', danger: true },
])

const findRow = (id: string) => items.value.find((x) => x.id === id)
const onRowAction = (key: string, row: Record<string, unknown>) => {
  const record = findRow(String(row.id))
  if (!record) return
  if (key === 'test') testExisting(record)
  else if (key === 'edit') openEdit(record)
  else if (key === 'delete') confirmDelete(record)
}

const getList = async () => {
  loading.value = true
  try {
    const { data } = await listFileSourcesRequest({ keywords: keywords.value.trim() || undefined })
    items.value = data.items ?? []
  } catch {
    items.value = []
  } finally {
    loading.value = false
  }
}
const search = () => getList()
const reset = () => {
  keywords.value = ''
  getList()
}

const openCreate = () => createRef.value?.showDialog()
const openEdit = (row: FileSourceInfo) => createRef.value?.showDialog(row)

const testExisting = async (row: FileSourceInfo) => {
  try {
    const { data } = await testFileSourceRequest({ id: row.id, type: '', config: undefined })
    if (data.ok) ElMessage.success(data.message || t('fileSource.testOk'))
    else ElMessage.error(data.message || t('fileSource.testFailed'))
  } catch {
    /* interceptor */
  }
}

const toggleEnabled = async (row: Record<string, unknown>, enabled: boolean) => {
  const record = findRow(String(row.id))
  if (!record) return
  try {
    await updateFileSourceRequest({
      id: record.id,
      name: record.name,
      enabled,
      redirect302: record.redirect302,
      config: { ...emptyConfig(), ...(record.config ?? {}), secretKey: '', password: '' },
    })
    record.enabled = enabled
    ElMessage.success(t('fileSource.updateSuccess'))
  } catch {
    /* interceptor */
  }
}

const confirmDelete = (row: FileSourceInfo) => {
  Dialog.confirm({
    content: t('fileSource.confirmDelete', { name: row.name }),
    onConfirm: async () => {
      await deleteFileSourceRequest(row.id)
      ElMessage.success(t('fileSource.deleteSuccess'))
      getList()
    },
  })
}

onMounted(getList)
</script>

<style scoped lang="scss">
.src-page {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.src-page__body {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.src-page__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.src-page__hint {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.src-page__dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  margin-right: 2px;
  border-radius: 999px;
  background: var(--el-text-color-placeholder);
}
.src-page__dot--on {
  background: var(--el-color-success, #16a34a);
}
.src-name {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.src-muted {
  color: var(--el-text-color-placeholder);
  font-size: 0.75rem;
}
.src-cards {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.src-card__redirect {
  color: var(--el-color-success, #16a34a);
}
.src-card__actions {
  display: flex;
  align-items: center;
  gap: 4px;
}
.src-skeleton {
  height: 96px;
  border-radius: var(--app-radius);
  background: linear-gradient(
    100deg,
    var(--el-fill-color-light) 30%,
    var(--el-fill-color) 50%,
    var(--el-fill-color-light) 70%
  );
  background-size: 200% 100%;
  animation: src-shimmer 1.4s ease-in-out infinite;
}
@keyframes src-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
@media (width <= 768px) {
  .src-page {
    gap: 8px;
  }
}
</style>
