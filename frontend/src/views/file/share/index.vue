<!-- 文件分享（重写 · P1 列表）：PageHeader(新建) + FilterBar(关键词) + 内联计数 + DataTable/移动卡 + Pagination。
     行内 复制链接/编辑/删除；启用 AppSwitch；有效/停用/过期/达上限 状态胶囊；ShareFormDialog 创建/编辑。 -->
<template>
  <div class="shr-page">
    <PageHeader :title="t('file.share.pageTitle')" :description="t('file.share.pageDesc')">
      <template #actions>
        <UButton color="primary" size="sm" icon="i-lucide-plus" @click="openCreate">
          {{ t('file.share.create') }}
        </UButton>
      </template>
    </PageHeader>

    <FilterBar @search="search" @reset="reset">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('file.share.keyword') }}</label>
          <input
            v-model="keywords"
            class="app-input"
            :placeholder="t('file.share.keywordPlaceholder')"
            @keyup.enter="search"
          />
        </div>
      </template>
    </FilterBar>

    <div class="shr-page__body">
      <div class="shr-page__bar">
        <span class="shr-page__hint">{{ t('system.common.total', { total: pagination.total }) }}</span>
      </div>

      <DataTable
        v-if="!menuStore.isMobile"
        :columns="columns"
        :rows="items"
        row-key="id"
        :loading="loading"
        :empty-text="t('system.common.noData')"
      >
        <template #cell-name="{ row }">
          <span class="shr-name">
            <component :is="menuStore.iconComponents['HOutline:FolderIcon']" class="shr-name__ico" />
            <span class="shr-name__text">{{ row.name || '—' }}</span>
          </span>
        </template>
        <template #cell-content="{ row }">
          <div class="shr-items" :title="fullPaths(row)">
            <span class="shr-items__badge">{{ t('file.share.itemsCount', { count: itemCount(row) }) }}</span>
            <span class="shr-items__first">{{ firstName(row) }}</span>
          </div>
        </template>
        <template #cell-code="{ row }">
          <span v-if="row.code" class="shr-code">
            <component :is="menuStore.iconComponents['HOutline:LockClosedIcon']" class="shr-code__ico" />
            {{ row.code }}
          </span>
          <span v-else class="shr-muted">{{ t('file.share.noCode') }}</span>
        </template>
        <template #cell-expires="{ row }">
          {{ row.expiresAt ? formatDateTime(row.expiresAt) : t('file.share.never') }}
        </template>
        <template #cell-downloads="{ row }">
          <span class="shr-num">
            {{ Number(row.downloadCount || 0) }}<template v-if="Number(row.maxDownloads) > 0"> / {{ row.maxDownloads }}</template>
          </span>
        </template>
        <template #cell-enabled="{ row }">
          <AppSwitch :model-value="!!row.enabled" @update:model-value="() => toggleEnabled(row)" />
        </template>
        <template #cell-status="{ row }">
          <StatusPill :variant="statusVariant(row)" :label="statusLabel(row)" />
        </template>
        <template #cell-operation="{ row }">
          <ActionMenu :items="rowActions" @select="(key) => onRowAction(key, row)" />
        </template>
      </DataTable>

      <template v-else>
        <div v-if="loading" class="shr-cards">
          <div v-for="i in 3" :key="i" class="shr-skeleton" />
        </div>
        <EmptyState
          v-else-if="!items.length"
          icon="HOutline:ShareIcon"
          :title="t('system.common.noData')"
          :description="t('file.share.emptyDesc')"
        />
        <div v-else class="shr-cards">
          <EntityCard v-for="row in items" :key="row.id" :title="row.name || t('file.share.noCode')">
            <template #status>
              <StatusPill :variant="statusVariant(row)" :label="statusLabel(row)" />
            </template>
            <template #meta>
              <span class="shr-card__path" :title="fullPaths(row)">
                {{ t('file.share.itemsCount', { count: itemCount(row) }) }} · {{ firstName(row) }}
              </span>
            </template>
            <template #footer>
              <div class="shr-card__stat">
                <span v-if="row.code" class="shr-code">
                  <component :is="menuStore.iconComponents['HOutline:LockClosedIcon']" class="shr-code__ico" />{{ row.code }}
                </span>
                <span>↓ {{ Number(row.downloadCount || 0) }}<template v-if="Number(row.maxDownloads) > 0">/{{ row.maxDownloads }}</template></span>
              </div>
              <div class="shr-card__actions">
                <AppSwitch :model-value="!!row.enabled" @update:model-value="() => toggleEnabled(row)" />
                <ActionMenu :items="rowActions" @select="(key) => onRowAction(key, row)" />
              </div>
            </template>
          </EntityCard>
        </div>
      </template>

      <Pagination
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        @change="loadList"
      />
    </div>

    <ShareFormDialog v-model="formOpen" :share="editing" @saved="loadList" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { formatDateTime, copyTextToClipboard, getBaseName } from '@/utils/file'
import { Dialog } from '@/utils/dialog'
import {
  listSharesRequest,
  deleteShareRequest,
  updateShareRequest,
  buildShareLink,
} from '@/api/share'
import ShareFormDialog from '@/components/share/ShareFormDialog.vue'
import type { ShareInfo, ShareItem } from '@/types/v1/file'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'

defineOptions({ name: 'FileShareView' })

const { t } = useI18n()
const menuStore = useMenuStore()

const items = ref<ShareInfo[]>([])
const loading = ref(false)
const keywords = ref('')
const pagination = ref({ page: 1, pageSize: 20, total: 0 })

const formOpen = ref(false)
const editing = ref<ShareInfo | null>(null)

// 分享内容可含多来源多文件：列表里不铺开全部路径（会撑长/成逗号汤），
// 只显示「数量徽标 + 首个文件名」，完整清单走 title 悬浮提示。
const itemsOf = (row: Record<string, unknown>) => (row.items as ShareItem[] | undefined) || []
const itemCount = (row: Record<string, unknown>) => itemsOf(row).length
const fullPaths = (row: Record<string, unknown>) => itemsOf(row).map((i) => i.path).join('\n')
const firstName = (row: Record<string, unknown>) => {
  const first = itemsOf(row)[0]
  if (!first) return '—'
  return getBaseName(first.path) || first.path
}

type PillVariant = 'success' | 'warning' | 'neutral'
const isExpired = (row: Record<string, unknown>) =>
  !!row.expiresAt && new Date(row.expiresAt as string).getTime() < Date.now()
const isExhausted = (row: Record<string, unknown>) =>
  Number(row.maxDownloads) > 0 && Number(row.downloadCount) >= Number(row.maxDownloads)
const statusVariant = (row: Record<string, unknown>): PillVariant => {
  if (!row.enabled) return 'neutral'
  if (isExpired(row) || isExhausted(row)) return 'warning'
  return 'success'
}
const statusLabel = (row: Record<string, unknown>) => {
  if (!row.enabled) return t('file.share.statusDisabled')
  if (isExpired(row)) return t('file.share.statusExpired')
  if (isExhausted(row)) return t('file.share.statusExhausted')
  return t('file.share.statusValid')
}

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('file.share.name'), minWidth: 150 },
  { key: 'content', title: t('file.share.content'), minWidth: 200 },
  { key: 'code', title: t('file.share.code'), width: 110 },
  { key: 'expires', title: t('file.share.expires'), width: 170 },
  { key: 'downloads', title: t('file.share.downloads'), width: 100 },
  { key: 'enabled', title: t('file.share.enabled'), width: 80 },
  { key: 'status', title: t('system.common.status'), width: 90 },
  { key: 'operation', title: t('system.common.operation'), width: 80, align: 'center' },
])

const rowActions = computed<ActionMenuItem[]>(() => [
  { key: 'copy', label: t('file.share.copyLink'), icon: 'HOutline:LinkIcon' },
  { key: 'edit', label: t('system.common.edit'), icon: 'HOutline:PencilSquareIcon' },
  { key: 'delete', label: t('system.common.delete'), icon: 'HOutline:TrashIcon', danger: true },
])

const findRow = (id: string) => items.value.find((x) => x.id === id)
const onRowAction = (key: string, row: Record<string, unknown>) => {
  const record = findRow(String(row.id))
  if (!record) return
  if (key === 'copy') copyLink(record)
  else if (key === 'edit') openEdit(record)
  else if (key === 'delete') remove(record)
}

const loadList = async () => {
  loading.value = true
  try {
    const { data } = await listSharesRequest({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keywords: keywords.value.trim() || undefined,
    })
    items.value = data?.items || []
    // 后端 total 为 int64，protobuf-JSON 序列化成字符串，这里转回 Number 供分页组件使用。
    pagination.value.total = Number(data?.total || 0)
  } catch {
    items.value = []
    pagination.value.total = 0
  } finally {
    loading.value = false
  }
}
const search = () => {
  pagination.value.page = 1
  loadList()
}
const reset = () => {
  keywords.value = ''
  search()
}

const openCreate = () => {
  editing.value = null
  formOpen.value = true
}
const openEdit = (row: ShareInfo) => {
  editing.value = row
  formOpen.value = true
}

const toggleEnabled = async (row: Record<string, unknown>) => {
  const record = findRow(String(row.id))
  if (!record) return
  try {
    await updateShareRequest({
      id: record.id,
      name: record.name,
      code: record.code,
      expiresAt: record.expiresAt,
      maxDownloads: record.maxDownloads,
      enabled: !record.enabled,
      items: [],
    })
    record.enabled = !record.enabled
  } catch {
    /* interceptor */
  }
}

const copyLink = async (row: ShareInfo) => {
  try {
    await copyTextToClipboard(buildShareLink(row.token))
    ElMessage.success(t('file.share.copied'))
  } catch {
    ElMessage.error(t('file.share.copyFailed'))
  }
}

const remove = (row: ShareInfo) => {
  Dialog.confirm({
    content: t('file.share.confirmDelete'),
    onConfirm: async () => {
      await deleteShareRequest(row.id)
      ElMessage.success(t('system.common.deleteSuccess'))
      loadList()
    },
  })
}

onMounted(loadList)
</script>

<style scoped lang="scss">
.shr-page {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.shr-page__body {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.shr-page__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.shr-page__hint {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.shr-name {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.shr-name__ico {
  width: 15px;
  height: 15px;
  flex-shrink: 0;
  color: var(--el-color-primary);
}
.shr-name__text {
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.shr-items {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  max-width: 100%;
}
.shr-items__badge {
  flex-shrink: 0;
  padding: 1px 7px;
  border-radius: 999px;
  background: var(--el-fill-color);
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.shr-items__first {
  min-width: 0;
  font-size: 0.8rem;
  color: var(--el-text-color-regular);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.shr-code {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 0.78rem;
  font-variant-numeric: tabular-nums;
  color: var(--el-text-color-regular);
}
.shr-code__ico {
  width: 12px;
  height: 12px;
  color: var(--el-text-color-placeholder);
}
.shr-num {
  font-variant-numeric: tabular-nums;
}
.shr-muted {
  color: var(--el-text-color-placeholder);
  font-size: 0.75rem;
}
.shr-cards {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.shr-card__path {
  min-width: 0;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.shr-card__stat {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  font-variant-numeric: tabular-nums;
}
.shr-card__actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}
.shr-skeleton {
  height: 96px;
  border-radius: var(--app-radius);
  background: linear-gradient(
    100deg,
    var(--el-fill-color-light) 30%,
    var(--el-fill-color) 50%,
    var(--el-fill-color-light) 70%
  );
  background-size: 200% 100%;
  animation: shr-shimmer 1.4s ease-in-out infinite;
}
@keyframes shr-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
@media (width <= 768px) {
  .shr-page {
    gap: 8px;
  }
}
</style>
