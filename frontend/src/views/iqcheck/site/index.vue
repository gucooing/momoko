<!-- 智商检测 · 检测配置：PageHeader(新增站点) + FilterBar + 列表/卡片两种视图（桌面可切换并记住偏好，移动端固定卡片）+ Pagination。
     子弹窗：siteForm(创建编辑，含拉取模型列表勾选)。行内 编辑/立即检测/复制公示页链接/删除。
     有站点正在检测时每 8 秒静默刷新一次列表。文案走 iqcheck.site.*。 -->
<template>
  <div class="iq-site-page">
    <PageHeader :title="t('iqcheck.site.title')" :description="t('iqcheck.site.pageDesc')">
      <template #actions>
        <UButton v-permission="[PERM.IQCHECK_EDIT]" color="primary" icon="i-lucide-plus" @click="openForm()">
          {{ t('iqcheck.site.create') }}
        </UButton>
      </template>
    </PageHeader>

    <FilterBar @search="reload" @reset="reset">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('iqcheck.site.colName') }}</label>
          <input
            v-model="queryForm.keywords"
            class="app-input"
            :placeholder="t('iqcheck.site.keywordsPlaceholder')"
            @keyup.enter="reload"
          />
        </div>
      </template>
    </FilterBar>

    <div class="iq-site-page__body">
      <div class="iq-site-page__bar">
        <span class="iq-site-page__hint">{{ t('system.common.total', { total: pagination.total }) }}</span>
        <div v-if="!menuStore.isMobile" class="iq-site-page__view">
          <UButton
            size="sm"
            :color="viewMode === 'table' ? 'primary' : 'neutral'"
            :variant="viewMode === 'table' ? 'solid' : 'soft'"
            icon="i-lucide-list"
            :title="t('iqcheck.site.viewTable')"
            @click="setView('table')"
          />
          <UButton
            size="sm"
            :color="viewMode === 'card' ? 'primary' : 'neutral'"
            :variant="viewMode === 'card' ? 'solid' : 'soft'"
            icon="i-lucide-layout-grid"
            :title="t('iqcheck.site.viewCard')"
            @click="setView('card')"
          />
        </div>
      </div>

      <!-- 列表视图 -->
      <DataTable v-if="!useCards" :columns="columns" :rows="sites" row-key="id" :loading="loading">
        <template #cell-endpoint="{ row }">
          <span class="iq-mono iq-site-endpoint" :title="String(row.endpoint)">{{ row.endpoint }}</span>
        </template>
        <template #cell-protocol="{ row }">{{ protocolLabel(row.protocol as IQCheckProtocol) }}</template>
        <template #cell-models="{ row }">
          <span :title="modelsTitle(row.models as IQCheckModelConfig[])">
            {{ t('iqcheck.site.modelsCount', { count: (row.models as IQCheckModelConfig[]).length }) }}
          </span>
        </template>
        <template #cell-enabled="{ row }">
          <StatusPill :variant="statusVariant(row as unknown as IQCheckSiteInfo)" :label="statusLabel(row as unknown as IQCheckSiteInfo)" />
        </template>
        <template #cell-intervalMinutes="{ row }">
          {{ t('iqcheck.site.intervalMinutes', { minutes: row.intervalMinutes }) }}
        </template>
        <template #cell-runCount="{ row }">{{ runsLabel(row as unknown as IQCheckSiteInfo) }}</template>
        <template #cell-lastCheckAt="{ row }">{{ fmtTime(row.lastCheckAt) }}</template>
        <template #cell-public="{ row }">
          <a class="iq-site-link" :href="publicUrl(String(row.slug))" target="_blank" rel="noopener">
            {{ t('iqcheck.site.openPublic') }} ↗
          </a>
        </template>
        <template #cell-operation="{ row }">
          <ActionMenu v-if="canEdit" :items="rowActions" @select="(key) => onRowAction(key, row)" />
          <span v-else class="text-dim">—</span>
        </template>
      </DataTable>

      <!-- 卡片视图 -->
      <template v-else>
        <div v-if="loading" class="iq-site-cards" :class="{ 'iq-site-cards--grid': !menuStore.isMobile }">
          <div v-for="i in 4" :key="i" class="iq-site-skeleton" />
        </div>
        <EmptyState
          v-else-if="!sites.length"
          icon="HOutline:AdjustmentsHorizontalIcon"
          :title="t('iqcheck.site.emptyTitle')"
          :description="t('iqcheck.site.emptyDesc')"
        />
        <div v-else class="iq-site-cards" :class="{ 'iq-site-cards--grid': !menuStore.isMobile }">
          <EntityCard v-for="row in sites" :key="row.id">
            <template #title>{{ row.name }}</template>
            <template #status>
              <StatusPill :variant="statusVariant(row)" :label="statusLabel(row)" />
            </template>
            <template #meta>
              <span class="iq-site-card__full iq-mono">{{ row.endpoint }}</span>
              <span class="iq-site-card__full">
                {{ protocolLabel(row.protocol) }} · {{ t('iqcheck.site.modelsCount', { count: row.models.length }) }}
                · {{ t('iqcheck.site.intervalMinutes', { minutes: row.intervalMinutes }) }} · {{ runsLabel(row) }}
              </span>
              <span v-if="row.models.length" class="iq-site-card__full iq-site-card__models" :title="modelsTitle(row.models)">
                {{ row.models.map((m) => m.model).join(', ') }}
              </span>
              <a class="iq-site-link iq-site-card__full" :href="publicUrl(row.slug)" target="_blank" rel="noopener">
                {{ t('iqcheck.site.openPublic') }} ↗
              </a>
            </template>
            <template #footer>
              <span>{{ t('iqcheck.site.colLastCheck') }}: {{ fmtTime(row.lastCheckAt) }}</span>
              <ActionMenu v-if="canEdit" :items="rowActions" @select="(key) => onRowAction(key, row)" />
            </template>
          </EntityCard>
        </div>
      </template>

      <Pagination
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        @change="loadSites()"
      />
    </div>

    <SiteForm ref="formRef" @refresh="loadSites()" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import dayjs from 'dayjs'
import { listIQCheckSites, deleteIQCheckSite, triggerIQCheck } from '@/api/iqcheck'
import { PERM } from '@/config/permission'
import { Dialog } from '@/utils/dialog'
import { IQCheckProtocol, type IQCheckModelConfig, type IQCheckSiteInfo } from '@/types/v1/iqcheck'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import SiteForm from '@/views/iqcheck/site/siteForm.vue'

defineOptions({ name: 'IQCheckSiteView' })

const menuStore = useMenuStore()
const router = useRouter()
const { t } = useI18n()

const formRef = useTemplateRef<InstanceType<typeof SiteForm> | null>('formRef')
const canEdit = computed(() => menuStore.hasButtonPermission(PERM.IQCHECK_EDIT))

// 视图偏好：桌面端可在列表/卡片之间切换并记住；移动端固定卡片。
type ViewMode = 'table' | 'card'
const VIEW_STORAGE_KEY = 'iqcheck-site-view'
const readView = (): ViewMode => {
  try {
    return localStorage.getItem(VIEW_STORAGE_KEY) === 'card' ? 'card' : 'table'
  } catch {
    return 'table'
  }
}
const viewMode = ref<ViewMode>(readView())
const setView = (mode: ViewMode) => {
  viewMode.value = mode
  try {
    localStorage.setItem(VIEW_STORAGE_KEY, mode)
  } catch {
    // 浏览器禁用存储时忽略
  }
}
const useCards = computed(() => menuStore.isMobile || viewMode.value === 'card')

const loading = ref(false)
const sites = ref<IQCheckSiteInfo[]>([])
const queryForm = ref({ keywords: '' })
const pagination = ref({ page: 1, pageSize: 10, total: 0 })

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('iqcheck.site.colName'), minWidth: 140 },
  { key: 'endpoint', title: t('iqcheck.site.colEndpoint'), minWidth: 220 },
  { key: 'protocol', title: t('iqcheck.site.colProtocol'), width: 110 },
  { key: 'models', title: t('iqcheck.site.colModels'), width: 100 },
  { key: 'enabled', title: t('iqcheck.site.colEnabled'), width: 100 },
  { key: 'intervalMinutes', title: t('iqcheck.site.colInterval'), width: 100 },
  { key: 'runCount', title: t('iqcheck.site.colRuns'), width: 100 },
  { key: 'lastCheckAt', title: t('iqcheck.site.colLastCheck'), width: 160 },
  { key: 'public', title: t('iqcheck.site.colPublic'), width: 150 },
  { key: 'operation', title: t('system.common.operation'), width: 70, align: 'center' },
])

const rowActions = computed<ActionMenuItem[]>(() => [
  { key: 'edit', label: t('system.common.edit'), icon: 'HOutline:PencilSquareIcon' },
  { key: 'check', label: t('iqcheck.site.checkNow'), icon: 'HOutline:PlayIcon' },
  { key: 'copy', label: t('iqcheck.site.copyPublic'), icon: 'HOutline:LinkIcon' },
  { key: 'delete', label: t('system.common.delete'), icon: 'HOutline:TrashIcon', danger: true },
])

const protocolLabels: Record<string, string> = {
  [IQCheckProtocol.IQ_CHECK_PROTOCOL_CHAT]: 'chat',
  [IQCheckProtocol.IQ_CHECK_PROTOCOL_RESPONSES]: 'responses',
  [IQCheckProtocol.IQ_CHECK_PROTOCOL_MESSAGES]: 'messages',
}
const protocolLabel = (p: IQCheckProtocol) => protocolLabels[p] || '—'
const modelsTitle = (models: IQCheckModelConfig[]) =>
  models.map((m) => (m.effort ? `${m.model} · ${m.effort}` : m.model)).join('\n')
const runsLabel = (site: IQCheckSiteInfo) =>
  site.maxRuns > 0
    ? t('iqcheck.site.runsLimited', { count: site.runCount, max: site.maxRuns })
    : t('iqcheck.site.runsUnlimited', { count: site.runCount })
const statusVariant = (site: IQCheckSiteInfo) => (site.checking ? 'warning' : site.enabled ? 'success' : 'neutral')
const statusLabel = (site: IQCheckSiteInfo) =>
  site.checking ? t('iqcheck.site.checking') : site.enabled ? t('system.common.enabled') : t('system.common.inactive')
const fmtTime = (v: unknown) => (v ? dayjs(v as string | Date).format('YYYY-MM-DD HH:mm') : '—')

// 公示页地址：前端固定路由前缀 /public/iqcheck + 站点随机路径。
const publicUrl = (slug: string) => {
  const href = router.resolve({ name: 'PublicIQCheck', params: { slug } }).href
  return new URL(href, window.location.origin).href
}

let pollTimer: number | undefined
const schedulePoll = () => {
  window.clearTimeout(pollTimer)
  if (sites.value.some((s) => s.checking)) {
    pollTimer = window.setTimeout(() => loadSites(true), 8000)
  }
}

const loadSites = async (silent = false) => {
  if (!silent) loading.value = true
  try {
    const { data } = await listIQCheckSites({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keywords: queryForm.value.keywords || undefined,
    })
    sites.value = data?.sites || []
    pagination.value.total = Number(data?.total || 0)
    schedulePoll()
  } finally {
    if (!silent) loading.value = false
  }
}

const reload = () => {
  pagination.value.page = 1
  loadSites()
}
const reset = () => {
  queryForm.value.keywords = ''
  pagination.value.page = 1
  loadSites()
}

const openForm = (row?: IQCheckSiteInfo) => formRef.value?.showDialog(row)

const onRowAction = (key: string, row: Record<string, unknown>) => {
  const site = sites.value.find((s) => s.id === String(row.id))
  if (!site) return
  if (key === 'edit') openForm(site)
  else if (key === 'check') checkNow(site)
  else if (key === 'copy') copyPublic(site)
  else if (key === 'delete') confirmDelete(site)
}

const checkNow = async (site: IQCheckSiteInfo) => {
  const { data } = await triggerIQCheck({ id: site.id })
  feedback.success(t('iqcheck.site.checkStarted', { count: data?.modelCount ?? site.models.length }))
  loadSites(true)
}

const copyPublic = async (site: IQCheckSiteInfo) => {
  await navigator.clipboard.writeText(publicUrl(site.slug))
  feedback.success(t('iqcheck.site.copied'))
}

const confirmDelete = (site: IQCheckSiteInfo) => {
  Dialog.info({
    showCancelButton: true,
    content: t('iqcheck.site.deleteConfirm', { name: site.name }),
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    onConfirm: async () => {
      await deleteIQCheckSite({ id: site.id })
      feedback.success(t('common.deleteSuccess'))
      loadSites()
    },
  })
}

onMounted(() => loadSites())
onBeforeUnmount(() => window.clearTimeout(pollTimer))
</script>

<style scoped lang="scss">
.iq-site-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.iq-site-page__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.iq-site-page__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.iq-site-page__hint {
  font-size: 0.78rem;
  color: var(--el-text-color-secondary);
}
.iq-site-page__view {
  display: flex;
  gap: 4px;
}
.iq-mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 0.78rem;
}
.iq-site-endpoint {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: bottom;
}
.iq-site-link {
  color: var(--el-color-primary);
  text-decoration: none;
  white-space: nowrap;
  &:hover {
    text-decoration: underline;
  }
}
.iq-site-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
  &--grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: 12px;
  }
}
.iq-site-card__full {
  display: block;
  width: 100%;
  overflow-wrap: anywhere;
}
.iq-site-card__models {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  color: var(--el-text-color-secondary);
  font-size: 0.78rem;
}
.iq-site-skeleton {
  height: 150px;
  border-radius: 12px;
  background: var(--el-fill-color-light);
}
</style>
