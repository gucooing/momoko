<!-- API Key：标题 + 状态计数说明 + 子导航 → 新建 / 刷新结果横幅（完整密钥 + 复制）→ 状态 chips（带数量）+ 搜索
     → 桌面表格（可多选，行内 复制 / 刷新 / 编辑 / 删除 小图标）/ 手机卡片 → 说明条 + 分页；多选时底部悬浮批量条（吊销 / 删除）。
     API Key 是远程主控调用本节点的签名种子：主控只发送派生的 Key ID 与时间戳签名，授权范围决定其可用功能。
     保留 list / copy / refresh / create / update / delete 契约。 -->
<template>
  <div class="key-page">
    <PageHeader :title="t('node.page.title')">
      <template #description>
        <span class="key-summary">
          {{ summaryParts.head }}
          <b class="is-ok">{{ summaryParts.valid }}</b>
          <b v-if="counts.expiring" class="is-warn">{{ summaryParts.soon }}</b>
          <b v-if="counts.expired" class="is-bad">{{ summaryParts.expired }}</b>
        </span>
      </template>
      <template #actions>
        <template v-if="!menuStore.isMobile">
          <UButton color="neutral" variant="soft" icon="i-lucide-rotate-cw" @click="getList">
            {{ t('node.key.refresh') }}
          </UButton>
          <UButton color="primary" icon="i-lucide-plus" @click="apiKeyCreateRef?.showDialog()">
            {{ t('node.key.add') }}
          </UButton>
        </template>
        <UButton
          v-else
          color="neutral"
          variant="ghost"
          icon="i-lucide-rotate-cw"
          :aria-label="t('node.key.refresh')"
          @click="getList"
        />
      </template>
    </PageHeader>
    <NodeSubNav :key-count="counts.total" />

    <!-- 新建 / 刷新结果：完整密钥 -->
    <section v-if="created" class="key-created" role="status">
      <span class="key-created__av m-sticker m-sticker--ok">
        <component :is="menuStore.iconComponents['HOutline:CheckIcon']" />
      </span>
      <div class="key-created__body">
        <div class="key-created__head">
          <h2 class="key-created__title">
            {{ created.kind === 'refresh' ? t('node.keyx.refreshedTitle') : t('node.keyx.createdTitle') }}
          </h2>
          <p class="key-created__desc">
            {{ t('node.keyx.createdDesc', { name: created.info.name, owner: ownerName, expiry: expiryShort(created.info) }) }}
          </p>
        </div>
        <div class="key-created__row">
          <div class="key-created__key m-mono">
            <component :is="menuStore.iconComponents['HOutline:KeyIcon']" />
            <span>{{ created.info.apiKey }}</span>
          </div>
          <UButton color="primary" size="xl" :icon="createdCopied ? 'i-lucide-check' : 'i-lucide-copy'" class="key-created__copy" @click="copyCreated">
            {{ createdCopied ? t('node.keyx.copied') : t('node.key.copy') }}
          </UButton>
        </div>
        <div class="key-created__warn">
          <component :is="menuStore.iconComponents['HOutline:ExclamationTriangleIcon']" />
          <span>{{ t('node.keyx.saveTip') }}</span>
        </div>
      </div>
      <div class="key-created__side">
        <AppIconButton icon="HOutline:XMarkIcon" :label="t('node.key.close')" @click="closeCreated" />
        <span class="key-created__spacer" />
        <UButton color="neutral" variant="soft" @click="closeCreated">{{ t('node.keyx.saved') }}</UButton>
      </div>
    </section>

    <ListToolbar
      v-model:status="statusFilter"
      v-model:keyword="queryForm.keywords"
      :chips="chips"
      :chips-label="t('node.list.status')"
      :placeholder="t('access.key.searchPlaceholder')"
      @search="reload"
    />

    <!-- 桌面：表格 -->
    <DataTable
      v-if="!menuStore.isMobile"
      v-model="selectedIds"
      class="key-table"
      :columns="columns"
      :rows="tableRows"
      row-key="id"
      selectable
      :loading="loading"
      :empty-text="hasFilter ? t('node.keyx.noMatch') : t('node.key.emptyDesc')"
    >
      <template #cell-name="{ row }">
        <div class="key-cell is-name">
          <span class="key-cell__title">
            {{ asKey(row).name }}
            <span v-if="isNew(asKey(row))" class="key-new">{{ t('node.keyx.newTag') }}</span>
          </span>
          <span v-if="asKey(row).remark" class="key-cell__sub" :title="asKey(row).remark">{{ asKey(row).remark }}</span>
        </div>
      </template>
      <template #cell-apiKey="{ row }">
        <div class="key-cell is-key">
          <span class="m-mono key-cell__mono">{{ asKey(row).apiKey }}</span>
          <span class="m-mono key-cell__sub" :title="t('node.key.keyIdTip')">{{ asKey(row).keyId }}</span>
        </div>
      </template>
      <template #cell-permissions="{ row }">
        <NodePermissionChips variant="groups" only-granted :permissions="asKey(row).permissions" class="key-perms" />
      </template>
      <template #cell-expiresAt="{ row }">
        <div class="key-cell">
          <StatusPill v-if="expiry(asKey(row)).pill" :variant="expiry(asKey(row)).pill" :label="expiry(asKey(row)).main" />
          <span v-else class="key-cell__strong m-num" :class="{ 'is-muted': !asKey(row).expiresAt }">
            {{ expiry(asKey(row)).main }}
          </span>
          <span class="key-cell__sub m-num">{{ expiry(asKey(row)).sub }}</span>
        </div>
      </template>
      <template #cell-lastUsed="{ row }">
        <div class="key-cell">
          <span class="key-cell__strong m-num" :class="{ 'is-muted': !asKey(row).lastUsedTime }" :title="dateTime(asKey(row).lastUsedTime)">
            {{ asKey(row).lastUsedTime ? ago(asKey(row).lastUsedTime) : t('node.keyx.neverUsed') }}
          </span>
          <span class="m-mono key-cell__sub">{{ asKey(row).lastUsedIp || '—' }}</span>
        </div>
      </template>
      <template #cell-createTime="{ row }">
        <span class="key-cell__sub m-num" :title="dateTime(asKey(row).createTime)">{{ date(asKey(row).createTime) }}</span>
      </template>
      <template #cell-operation="{ row }">
        <div class="key-ops">
          <AppIconButton icon="HOutline:ClipboardDocumentIcon" :box="34" :size="17" :label="t('node.key.copy')" @click="openCopy(asKey(row))" />
          <AppIconButton icon="HOutline:ArrowPathIcon" :box="34" :size="17" :label="t('node.key.refresh')" @click="openRefresh(asKey(row))" />
          <AppIconButton icon="HOutline:PencilSquareIcon" :box="34" :size="17" :label="t('node.key.edit')" @click="apiKeyCreateRef?.showDialog(asKey(row))" />
          <AppIconButton class="is-danger" icon="HOutline:TrashIcon" :box="34" :size="17" :label="t('node.key.delete')" @click="openDelete(asKey(row))" />
        </div>
      </template>
    </DataTable>

    <!-- 手机：卡片 -->
    <template v-else>
      <div v-if="loading && !list.length" class="key-cards">
        <div v-for="i in 3" :key="i" class="key-skeleton" />
      </div>
      <EmptyState
        v-else-if="!list.length"
        icon="HOutline:KeyIcon"
        :title="hasFilter ? t('node.keyx.noMatch') : t('node.key.noData')"
        :description="hasFilter ? '' : t('node.key.emptyDesc')"
      />
      <div v-else class="key-cards">
        <article v-for="(row, idx) in list" :key="row.id" class="key-card">
          <header class="key-card__head">
            <span class="key-card__av m-sticker" :class="`m-sticker--${TONES[idx % TONES.length]}`" :style="{ transform: `rotate(${ROTS[idx % ROTS.length]})` }">
              <component :is="menuStore.iconComponents['HOutline:KeyIcon']" />
            </span>
            <span class="key-card__title">
              <span class="key-card__name">
                {{ row.name }}
                <span v-if="isNew(row)" class="key-new">{{ t('node.keyx.newTag') }}</span>
              </span>
              <span class="key-card__mask m-mono">{{ row.apiKey }}</span>
            </span>
            <StatusPill :variant="statusPill(row).variant" :label="statusPill(row).label" />
          </header>
          <p v-if="row.remark" class="key-card__remark">{{ row.remark }}</p>
          <dl class="key-card__info">
            <div>
              <dt>{{ t('node.key.expiresAt') }}</dt>
              <dd class="m-num" :class="expiryTone(row)">{{ expiry(row).main }}</dd>
            </div>
            <div>
              <dt>{{ t('node.keyx.lastUsed') }}</dt>
              <dd class="m-num">{{ row.lastUsedTime ? ago(row.lastUsedTime) : t('node.keyx.neverUsed') }}</dd>
            </div>
            <div>
              <dt>{{ t('node.keyx.createdAt') }}</dt>
              <dd class="m-num">{{ date(row.createTime) }}</dd>
            </div>
            <div>
              <dt>{{ t('node.keyx.sourceIp') }}</dt>
              <dd class="m-mono">{{ row.lastUsedIp || '—' }}</dd>
            </div>
          </dl>
          <NodePermissionChips variant="groups" only-granted :permissions="row.permissions" class="key-card__perms" />
          <footer class="key-card__foot">
            <span class="key-card__spacer" />
            <AppIconButton icon="HOutline:ClipboardDocumentIcon" :box="44" :label="t('node.key.copy')" @click="openCopy(row)" />
            <AppIconButton icon="HOutline:ArrowPathIcon" :box="44" :label="t('node.key.refresh')" @click="openRefresh(row)" />
            <AppIconButton icon="HOutline:PencilSquareIcon" :box="44" :label="t('node.key.edit')" @click="apiKeyCreateRef?.showDialog(row)" />
            <AppIconButton class="is-danger" icon="HOutline:TrashIcon" :box="44" :label="t('node.key.delete')" @click="openDelete(row)" />
          </footer>
        </article>
      </div>
    </template>

    <div class="key-foot">
      <div class="key-tip">
        <component :is="menuStore.iconComponents['HOutline:ShieldCheckIcon']" />
        <span>{{ t('node.keyx.formula') }} {{ t('access.key.tip') }}</span>
      </div>
    </div>

    <Pagination
      v-model:page="pagination.page"
      v-model:page-size="pagination.pageSize"
      :total="pagination.total"
      @change="getList"
    />

    <!-- 批量操作条 -->
    <Transition name="key-pop">
      <div v-if="selectedIds.length" class="key-batch" role="toolbar">
        <span class="key-batch__count">{{ t('node.keyx.selected', { count: selectedIds.length }) }}</span>
        <button type="button" class="key-batch__btn" @click="bulkRevoke">
          <component :is="menuStore.iconComponents['HOutline:NoSymbolIcon']" />{{ t('node.keyx.revoke') }}
        </button>
        <button type="button" class="key-batch__btn is-danger" @click="bulkDelete">
          <component :is="menuStore.iconComponents['HOutline:TrashIcon']" />{{ t('node.key.delete') }}
        </button>
        <button type="button" class="key-batch__btn is-icon" :aria-label="t('node.keyx.clearSelection')" @click="selectedIds = []">
          <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
        </button>
      </div>
    </Transition>

    <!-- 手机：悬浮新建 -->
    <div v-if="menuStore.isMobile && !created" class="key-fab">
      <UButton color="primary" size="xl" icon="i-lucide-plus" class="key-fab__btn" @click="apiKeyCreateRef?.showDialog()">
        {{ t('node.key.add') }}
      </UButton>
    </div>

    <ApiKeyCreate ref="apiKeyCreateRef" @refresh="onFormRefresh" />

    <!-- 复制 API Key -->
    <FormDialog v-model="copyDialogOpen" :title="t('node.key.copyTitle')" :width="560">
      <div class="key-copy">
        <div class="key-copy__key m-mono">{{ copyKeyValue }}</div>
        <div class="key-copy__warn">
          <component :is="menuStore.iconComponents['HOutline:ExclamationTriangleIcon']" />
          <span>{{ t('node.key.copyWarning') }}</span>
        </div>
        <p class="key-copy__desc">{{ t('node.key.seedHint') }}</p>
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
import { copyAPIKey, deleteAPIKey, listAPIKeys, refreshAPIKey, updateAPIKey } from '@/api/node'
import { APIKeyStatus, type APIKeyInfo } from '@/types/v1/node'
import { Dialog } from '@/utils/dialog'
import { useUserStore } from '@/stores/user'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'
import ApiKeyCreate from '@/views/node/key/create.vue'
import NodePermissionChips from '@/views/node/components/permissionChips.vue'
import NodeSubNav from '@/views/node/components/NodeSubNav.vue'
import { toDayjs, useNodeTime } from '@/views/node/components/useNodeTime'

defineOptions({ name: 'ApiKeyView' })

type StatusFilter = 'all' | APIKeyStatus

const menuStore = useMenuStore()
const userStore = useUserStore()
const { t } = useI18n()
const { ago, date, dateTime, daysUntil } = useNodeTime()
const apiKeyCreateRef = useTemplateRef<InstanceType<typeof ApiKeyCreate> | null>('apiKeyCreateRef')

const TONES = ['p', 'a2', 'a', 'ok']
const ROTS = ['-4deg', '3deg', '-2deg', '4deg']

const loading = ref(false)
const list = ref<APIKeyInfo[]>([])
const copyDialogOpen = ref(false)
const copyKeyValue = ref('')
const selectedIds = ref<string[]>([])
const statusFilter = ref<StatusFilter>('all')
const counts = ref({ total: 0, valid: 0, expiring: 0, expired: 0 })
const created = ref<{ kind: 'create' | 'refresh'; info: APIKeyInfo } | null>(null)
const createdCopied = ref(false)

const queryForm = ref({ keywords: '' })
const pagination = ref({ page: 1, pageSize: 10, total: 0 })

const tableRows = computed(() => list.value as unknown as Record<string, unknown>[])
const asKey = (row: Record<string, unknown>) => row as unknown as APIKeyInfo
const hasFilter = computed(() => statusFilter.value !== 'all' || !!queryForm.value.keywords.trim())
const ownerName = computed(() => userStore.userInfo?.name || userStore.userInfo?.username || '—')

// 「共 n 个密钥 · x 个有效 · y 个即将过期 · z 个已过期」拆成可着色的片段
const summaryParts = computed(() => {
  const c = counts.value
  const full = t('access.key.summary', { total: c.total, valid: c.valid, soon: c.expiring, expired: c.expired })
  const parts = full.split(' · ')
  return {
    head: parts[0] + ' · ',
    valid: parts[1] || '',
    soon: parts[2] ? ' · ' + parts[2] : '',
    expired: parts[3] ? ' · ' + parts[3] : '',
  }
})

const chips = computed<ToolbarChip<StatusFilter>[]>(() => [
  { label: t('node.page.filterAll'), value: 'all', count: counts.value.total },
  { label: t('access.key.valid'), value: APIKeyStatus.APIKeyStatus_Valid, count: counts.value.valid },
  { label: t('access.key.expiring'), value: APIKeyStatus.APIKeyStatus_Expiring, count: counts.value.expiring },
  { label: t('access.key.expired'), value: APIKeyStatus.APIKeyStatus_Expired, count: counts.value.expired },
])

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('node.key.name'), minWidth: 150 },
  { key: 'apiKey', title: t('node.keyx.colKey'), minWidth: 150 },
  { key: 'permissions', title: t('node.key.permissions'), minWidth: 200, ellipsis: false },
  { key: 'expiresAt', title: t('node.key.expiresAt'), minWidth: 130 },
  { key: 'lastUsed', title: t('node.keyx.lastUsed'), minWidth: 110 },
  { key: 'createTime', title: t('node.keyx.createdAt'), width: 108 },
  { key: 'operation', title: t('node.key.operation'), width: 160, align: 'right' },
])

// —— 有效期展示 ——
type PillVariant = 'success' | 'warning' | 'error'
const expiry = (row: APIKeyInfo): { main: string; sub: string; pill?: PillVariant } => {
  if (!row.expiresAt) return { main: t('node.key.permanent'), sub: t('access.key.suggestExpiry') }
  const days = daysUntil(row.expiresAt) ?? 0
  const d = date(row.expiresAt)
  if (row.status === APIKeyStatus.APIKeyStatus_Expired) {
    return { main: t('access.key.expired'), sub: t('node.keyx.expiryOn', { date: d }), pill: 'error' }
  }
  if (row.status === APIKeyStatus.APIKeyStatus_Expiring) {
    return { main: t('access.key.expiresIn', { days: Math.max(days, 0) }), sub: t('node.keyx.expiryOn', { date: d }), pill: 'warning' }
  }
  return { main: d, sub: t('access.key.daysLeft', { days }) }
}
const expiryTone = (row: APIKeyInfo) => {
  if (row.status === APIKeyStatus.APIKeyStatus_Expired) return 'is-bad'
  if (row.status === APIKeyStatus.APIKeyStatus_Expiring) return 'is-warn'
  return row.expiresAt ? '' : 'is-muted'
}
const expiryShort = (row: APIKeyInfo) =>
  row.expiresAt ? t('node.keyx.expiryOn', { date: date(row.expiresAt) }) : t('node.key.permanent')
const statusPill = (row: APIKeyInfo): { variant: PillVariant; label: string } => {
  if (row.status === APIKeyStatus.APIKeyStatus_Expired) return { variant: 'error', label: t('access.key.expired') }
  if (row.status === APIKeyStatus.APIKeyStatus_Expiring) return { variant: 'warning', label: t('access.key.expiring') }
  return { variant: 'success', label: t('access.key.valid') }
}
// 24 小时内创建的显示「新」
const isNew = (row: APIKeyInfo) => {
  const d = toDayjs(row.createTime)
  return !!d && Date.now() - d.valueOf() < 24 * 3600 * 1000
}

// —— 数据 ——
const getList = async () => {
  loading.value = true
  try {
    const { data: res } = await listAPIKeys({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keywords: queryForm.value.keywords.trim() || undefined,
      status: statusFilter.value === 'all' ? undefined : statusFilter.value,
    })
    list.value = res?.infos || []
    pagination.value.total = Number(res?.total || 0)
    pagination.value.page = Number(res?.page || pagination.value.page)
    pagination.value.pageSize = Number(res?.pageSize || pagination.value.pageSize)
    const valid = Number(res?.validCount || 0)
    const expiring = Number(res?.expiringCount || 0)
    const expired = Number(res?.expiredCount || 0)
    counts.value = { total: valid + expiring + expired, valid, expiring, expired }
    const ids = new Set(list.value.map((x) => x.id))
    selectedIds.value = selectedIds.value.filter((id) => ids.has(id))
  } finally {
    loading.value = false
  }
}

const reload = () => {
  pagination.value.page = 1
  getList()
}

const openDelete = (row: APIKeyInfo) => {
  Dialog.confirm({
    title: t('node.key.deleteTitle'),
    content: t('node.key.deleteContent', { name: row.name }),
    confirmText: t('node.key.confirm'),
    cancelText: t('node.key.cancel'),
    onConfirm: async () => {
      await deleteAPIKey({ id: row.id })
      feedback.success(t('node.key.deleteSuccess'))
      getList()
    },
  })
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
      const { data: res } = await refreshAPIKey({ id: row.id })
      feedback.success(t('node.key.refreshSuccess'))
      if (res?.info) showCreated('refresh', res.info)
      getList()
    },
  })
}

const writeClipboard = async (value: string) => {
  await navigator.clipboard.writeText(value)
  feedback.success(t('node.key.copied'))
}
const doCopy = () => writeClipboard(copyKeyValue.value)

const showCreated = (kind: 'create' | 'refresh', info: APIKeyInfo) => {
  created.value = { kind, info }
  createdCopied.value = false
  window.scrollTo({ top: 0, behavior: 'smooth' })
}
const closeCreated = () => {
  created.value = null
  createdCopied.value = false
}
const copyCreated = async () => {
  if (!created.value) return
  await writeClipboard(created.value.info.apiKey)
  createdCopied.value = true
}

// —— 批量 ——
const runBatch = async (fn: (id: string) => Promise<unknown>) => {
  const ids = [...selectedIds.value]
  const results = await Promise.allSettled(ids.map((id) => fn(id)))
  return results.filter((r) => r.status === 'fulfilled').length
}
const bulkRevoke = () => {
  const count = selectedIds.value.length
  Dialog.confirm({
    title: t('node.keyx.revokeTitle'),
    content: t('node.keyx.revokeContent', { count }),
    confirmText: t('node.keyx.revoke'),
    cancelText: t('node.key.cancel'),
    onConfirm: async () => {
      const now = new Date()
      const ok = await runBatch((id) => updateAPIKey({ id, expiresAt: now, permissions: undefined }))
      feedback.success(t('node.keyx.revokeSuccess', { count: ok }))
      selectedIds.value = []
      getList()
    },
  })
}
const bulkDelete = () => {
  const count = selectedIds.value.length
  Dialog.confirm({
    title: t('node.keyx.bulkDeleteTitle'),
    content: t('node.keyx.bulkDeleteContent', { count }),
    confirmText: t('node.key.delete'),
    cancelText: t('node.key.cancel'),
    onConfirm: async () => {
      const ok = await runBatch((id) => deleteAPIKey({ id }))
      feedback.success(t('node.keyx.bulkDeleteSuccess', { count: ok }))
      selectedIds.value = []
      getList()
    },
  })
}

const onFormRefresh = (type: 'create' | 'update', info?: APIKeyInfo) => {
  if (type === 'create') {
    pagination.value.page = 1
    if (info?.apiKey) showCreated('create', info)
  }
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
  gap: 18px;
}
.key-summary b {
  font-weight: 700;
}
.key-summary .is-ok {
  color: var(--m-ok);
}
.key-summary .is-warn {
  color: var(--m-warn);
}
.key-summary .is-bad {
  color: var(--m-bad);
}

/* 新建结果横幅 */
.key-created {
  display: flex;
  align-items: flex-start;
  gap: 20px;
  padding: 22px 22px 22px 24px;
  border-radius: var(--app-radius-lg);
  border: 2px solid var(--m-ok);
  background: var(--el-bg-color);
}
.key-created__av {
  width: 56px;
  height: 56px;
  border-radius: 18px;
  flex: none;
  transform: rotate(-6deg);
}
.key-created__av :deep(svg) {
  width: 28px;
  height: 28px;
  stroke-width: 2.4;
}
.key-created__body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.key-created__head {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.key-created__title {
  font-family: var(--app-font-display);
  font-size: 1.25rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.key-created__desc {
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.key-created__row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.key-created__key {
  flex: 1 1 380px;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  padding: 10px 16px;
  border-radius: var(--app-radius);
  border: 1.5px dashed var(--el-border-color-light);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-size: 0.9375rem;
  font-weight: 600;
  letter-spacing: 0.02em;
  word-break: break-all;
}
.key-created__key :deep(svg) {
  width: 16px;
  height: 16px;
  flex: none;
  color: var(--el-text-color-secondary);
}
.key-created__copy {
  min-height: 52px;
  padding: 0 22px;
  border-radius: 999px;
}
.key-created__warn {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 14px;
  border-radius: var(--app-radius-sm);
  background: var(--m-warnSoft);
  color: var(--m-warn);
  font-size: 0.875rem;
  font-weight: 600;
  line-height: 1.5;
}
.key-created__warn :deep(svg) {
  width: 18px;
  height: 18px;
  flex: none;
  margin-top: 1px;
}
.key-created__side {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 10px;
  align-self: stretch;
}
.key-created__spacer {
  flex: 1;
}
.key-created__side :deep(button:last-child) {
  min-height: 44px;
  border-radius: 999px;
  padding: 0 18px;
  font-weight: 700;
}

/* 表格 */
.key-cell {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 3px;
  min-width: 0;
}
.key-cell.is-name {
  max-width: 190px;
}
.key-cell.is-key {
  max-width: 150px;
}
.key-perms {
  max-width: 220px;
}
.key-cell__title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.key-cell__strong {
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.key-cell__strong.is-muted {
  color: var(--el-text-color-secondary);
}
.key-cell__mono {
  font-size: 0.8125rem;
  white-space: nowrap;
}
.key-cell__sub {
  max-width: 100%;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.key-new {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 7px;
  border-radius: 6px;
  background: var(--m-a);
  color: var(--m-aOn);
  font-size: 0.6875rem;
  font-weight: 800;
}
.key-perms :deep(.perm-chip) {
  margin: 0;
}
.key-ops {
  display: flex;
  justify-content: flex-end;
  gap: 2px;
}
.key-ops :deep(.is-danger),
.key-card__foot :deep(.is-danger) {
  color: var(--m-bad);
}
.key-ops :deep(.is-danger:hover),
.key-card__foot :deep(.is-danger:hover) {
  background: var(--m-badSoft);
}

/* 说明条 */
.key-foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}
.key-tip {
  flex: 1 1 520px;
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 16px;
  border-radius: 18px;
  background: var(--el-fill-color-light);
  font-size: 0.8125rem;
  line-height: 1.55;
  color: var(--el-text-color-secondary);
}
.key-tip :deep(svg) {
  width: 20px;
  height: 20px;
  flex: none;
  color: var(--m-a2);
}

/* 手机卡片 */
.key-cards {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.key-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
  padding: 16px 12px 6px 16px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.key-card__head {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  padding-right: 4px;
}
.key-card__av {
  width: 44px;
  height: 44px;
  border-radius: 15px;
  flex: none;
}
.key-card__av :deep(svg) {
  width: 20px;
  height: 20px;
}
.key-card__title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.key-card__name {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-family: var(--app-font-display);
  font-size: 1rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  white-space: nowrap;
}
.key-card__mask {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.key-card__remark {
  margin-right: 4px;
  font-size: 0.8125rem;
  line-height: 1.45;
  color: var(--el-text-color-regular);
}
.key-card__info {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 12px;
  margin: 0 4px 0 0;
  padding: 10px 12px;
  border-radius: 14px;
  background: var(--el-fill-color-light);
}
.key-card__info > div {
  min-width: 0;
}
.key-card__info dt {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.key-card__info dd {
  margin: 2px 0 0;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.key-card__info dd.m-mono {
  font-size: 0.75rem;
  font-weight: 500;
}
.key-card__info dd.is-warn {
  color: var(--m-warn);
}
.key-card__info dd.is-bad {
  color: var(--m-bad);
}
.key-card__info dd.is-muted {
  color: var(--el-text-color-secondary);
}
.key-card__perms {
  padding-right: 4px;
}
.key-card__foot {
  display: flex;
  align-items: center;
  padding-top: 6px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.key-card__spacer {
  flex: 1;
}
.key-skeleton {
  height: 220px;
  border-radius: var(--app-radius-lg);
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

/* 复制弹窗 */
.key-copy {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.key-copy__key {
  padding: 14px 16px;
  border-radius: var(--app-radius);
  border: 1.5px dashed var(--el-border-color-light);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-size: 0.9375rem;
  font-weight: 600;
  line-height: 1.5;
  word-break: break-all;
  user-select: all;
}
.key-copy__warn {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 14px;
  border-radius: var(--app-radius-sm);
  background: var(--m-warnSoft);
  color: var(--m-warn);
  font-size: 0.8125rem;
  font-weight: 600;
  line-height: 1.5;
}
.key-copy__warn :deep(svg) {
  width: 18px;
  height: 18px;
  flex: none;
}
.key-copy__desc {
  font-size: 0.8125rem;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}

/* 批量条 */
.key-batch {
  position: sticky;
  bottom: 18px;
  z-index: 20;
  align-self: center;
  display: flex;
  align-items: center;
  gap: 2px;
  max-width: 100%;
  padding: 6px 6px 6px 18px;
  border-radius: 999px;
  background: var(--m-ink);
  color: var(--m-bg);
  box-shadow: var(--app-shadow-lg);
  overflow-x: auto;
  scrollbar-width: none;
}
.key-batch::-webkit-scrollbar {
  display: none;
}
.key-batch__count {
  margin-right: 10px;
  font-size: 0.875rem;
  font-weight: 800;
  white-space: nowrap;
}
.key-batch__btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 14px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: inherit;
  font-size: 0.875rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.key-batch__btn:hover {
  background: color-mix(in srgb, var(--m-bg) 14%, transparent);
}
.key-batch__btn.is-danger {
  color: var(--m-a);
}
.key-batch__btn.is-icon {
  width: 40px;
  padding: 0;
  justify-content: center;
}
.key-batch__btn :deep(svg) {
  width: 16px;
  height: 16px;
}
.key-pop-enter-active,
.key-pop-leave-active {
  transition: opacity 0.18s, transform 0.18s;
}
.key-pop-enter-from,
.key-pop-leave-to {
  opacity: 0;
  transform: translateY(12px);
}

/* 手机 */
.key-fab {
  position: sticky;
  bottom: calc(92px + env(safe-area-inset-bottom));
  display: flex;
  justify-content: flex-end;
  margin-top: -56px;
  pointer-events: none;
  z-index: 5;
}
.key-fab__btn {
  pointer-events: auto;
  min-height: 56px;
  padding: 0 22px;
  border-radius: var(--app-radius-pill);
  box-shadow: var(--app-shadow-brand);
}
@media (width < 768px) {
  .key-page {
    gap: 14px;
    padding-bottom: 72px;
  }
  .key-created {
    flex-wrap: wrap;
    gap: 14px;
    padding: 18px 16px;
  }
  .key-created__av {
    display: none;
  }
  .key-created__body {
    flex: 1 1 100%;
  }
  .key-created__side {
    order: 3;
    flex: 1 1 100%;
    flex-direction: row;
  }
  .key-created__side > :first-child,
  .key-created__spacer {
    display: none;
  }
  .key-created__side :deep(button:last-child) {
    flex: 1;
    justify-content: center;
    min-height: 52px;
  }
  .key-created__key {
    flex-basis: 100%;
  }
  .key-created__copy {
    flex: 1 1 100%;
    justify-content: center;
  }
}
</style>
