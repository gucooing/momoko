<!-- 角色管理（重写 · P1 列表/CRUD 范式）：PageHeader + FilterBar + 批量条 + 卡/表切换
     + EntityCard 卡片流 / DataTable 表视图 + Pagination + FormDialog(create.vue)。
     内置角色不可选/编辑/删除（rowSelectable 过滤）。保留 rolePage/deleteRole 接口、PERM 权限（06d）。 -->
<template>
  <div class="role-page">
    <PageHeader :title="t('system.role.title')" :description="t('system.role.pageDesc')">
      <template #actions>
        <UButton
          v-permission="[PERM.ROLE_ADD]"
          color="primary"
          icon="i-lucide-plus"
          @click="openCreate"
        >
          {{ t('system.role.addRole') }}
        </UButton>
      </template>
    </PageHeader>

    <FilterBar @search="search" @reset="reset">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('system.common.roleName') }}</label>
          <input
            v-model="queryForm.name"
            class="app-input"
            :placeholder="t('system.role.namePlaceholder')"
            @keyup.enter="search"
          />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('system.common.status') }}</label>
          <AppSelect v-model="queryForm.status" :options="statusOptions" />
        </div>
      </template>
    </FilterBar>

    <!-- 批量操作条（选中后出现，权限内） -->
    <div v-if="selectedIds.length && canDelete" class="role-page__batch">
      <span class="role-page__batch-count">{{ t('system.role.selectedCount', { count: selectedIds.length }) }}</span>
      <div class="role-page__batch-actions">
        <UButton color="neutral" variant="ghost" size="sm" @click="selectedIds = []">
          {{ t('system.role.clearSelection') }}
        </UButton>
        <UButton color="error" variant="soft" size="sm" icon="i-lucide-trash-2" @click="confirmDelete(selectedIds)">
          {{ t('system.role.batchDelete') }}
        </UButton>
      </div>
    </div>

    <!-- 主体：卡/表切换（桌面）；移动强制卡片 -->
    <div class="role-page__body">
      <div class="role-page__bar">
        <span class="role-page__bar-hint">{{ t('system.common.total', { total: pagination.total }) }}</span>
        <div v-if="!menuStore.isMobile" class="seg">
          <button
            type="button"
            class="seg__btn"
            :class="{ 'is-active': viewMode === 'card' }"
            @click="viewMode = 'card'"
          >
            <component :is="menuStore.iconComponents['HOutline:Squares2X2Icon']" />
            {{ t('system.common.card') }}
          </button>
          <button
            type="button"
            class="seg__btn"
            :class="{ 'is-active': viewMode === 'table' }"
            @click="viewMode = 'table'"
          >
            <component :is="menuStore.iconComponents['HOutline:Bars3Icon']" />
            {{ t('system.common.table') }}
          </button>
        </div>
      </div>

      <!-- 卡片流 -->
      <template v-if="menuStore.isMobile || viewMode === 'card'">
        <div v-if="loading" class="role-grid">
          <div v-for="i in 6" :key="i" class="role-skeleton" />
        </div>
        <EmptyState
          v-else-if="!roleList.length"
          icon="HOutline:ShieldCheckIcon"
          :title="t('system.common.noData')"
          :description="t('system.role.emptyDesc')"
        >
          <template #action>
            <UButton
              v-permission="[PERM.ROLE_ADD]"
              color="primary"
              variant="soft"
              icon="i-lucide-plus"
              @click="openCreate"
            >
              {{ t('system.role.addRole') }}
            </UButton>
          </template>
        </EmptyState>
        <div v-else class="role-grid">
          <EntityCard
            v-for="row in roleList"
            :key="row.roleId"
            class="role-card"
            :class="{ 'is-selected': selectedSet.has(row.roleId) }"
          >
            <template #title>
              <div class="role-card__id">
                <input
                  v-if="!row.isBuiltin"
                  type="checkbox"
                  class="role-card__check"
                  :checked="selectedSet.has(row.roleId)"
                  @change="toggleSelect(row.roleId)"
                />
                <span v-else class="role-card__lock">
                  <component :is="menuStore.iconComponents['HOutline:LockClosedIcon']" />
                </span>
                <span class="role-card__name">{{ row.name }}</span>
              </div>
            </template>
            <template #status>
              <StatusPill :variant="statusVariant(row.status)" :label="statusLabel(row.status)" />
            </template>
            <template #meta>
              <span class="role-card__desc">{{ row.description || '—' }}</span>
              <StatusPill
                :variant="row.isBuiltin ? 'warning' : 'info'"
                :dot="false"
                :label="row.isBuiltin ? t('system.common.builtIn') : t('system.common.custom')"
              />
            </template>
            <template #footer>
              <span>{{ fmtTime(row.createTime) }}</span>
              <div v-if="!row.isBuiltin" class="role-card__actions">
                <UButton
                  v-permission="[PERM.ROLE_EDIT]"
                  color="neutral"
                  variant="soft"
                  size="xs"
                  icon="i-lucide-pencil"
                  @click="openEdit(row)"
                >
                  {{ t('system.common.edit') }}
                </UButton>
                <UButton
                  v-permission="[PERM.ROLE_DELETE]"
                  color="error"
                  variant="soft"
                  size="xs"
                  icon="i-lucide-trash-2"
                  @click="confirmDelete([row.roleId])"
                >
                  {{ t('system.common.delete') }}
                </UButton>
              </div>
            </template>
          </EntityCard>
        </div>
      </template>

      <!-- 表视图（桌面） -->
      <DataTable
        v-else
        v-model="selectedIds"
        :columns="columns"
        :rows="roleList"
        row-key="roleId"
        selectable
        :row-selectable="(row) => !row.isBuiltin"
        :loading="loading"
      >
        <template #cell-description="{ row }">
          <span v-if="row.description">{{ row.description }}</span>
          <span v-else class="text-dim">—</span>
        </template>
        <template #cell-isBuiltin="{ row }">
          <StatusPill
            :variant="row.isBuiltin ? 'warning' : 'info'"
            :dot="false"
            :label="row.isBuiltin ? t('system.common.builtIn') : t('system.common.custom')"
          />
        </template>
        <template #cell-status="{ row }">
          <StatusPill :variant="statusVariant(row.status)" :label="statusLabel(row.status)" />
        </template>
        <template #cell-createTime="{ row }">{{ fmtTime(row.createTime) }}</template>
        <template #cell-updateTime="{ row }">{{ fmtTime(row.updateTime) }}</template>
        <template #cell-operation="{ row }">
          <ActionMenu
            v-if="!row.isBuiltin"
            :items="rowActions"
            @select="(key) => onRowAction(key, row)"
          />
          <span v-else class="text-dim">—</span>
        </template>
      </DataTable>

      <Pagination
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        @change="getRoleList"
      />
    </div>

    <RoleCreate ref="roleCreateRef" @refresh="refresh" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { deleteRole, rolePage } from '@/api/role'
import { PERM } from '@/config/permission'
import { Dialog } from '@/utils/dialog'
import { RoleStatus, type RoleInfo } from '@/types/v1/system'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import RoleCreate from '@/views/system/role/create.vue'

defineOptions({ name: 'RoleView' })

const menuStore = useMenuStore()
const { t } = useI18n()
const roleCreateRef = useTemplateRef<InstanceType<typeof RoleCreate> | null>('roleCreateRef')

const viewMode = ref<'card' | 'table'>('card')
const loading = ref(false)
const roleList = ref<RoleInfo[]>([])
const selectedIds = ref<string[]>([])
const selectedSet = computed(() => new Set(selectedIds.value))

const queryForm = ref<{ name: string; status: RoleStatus | undefined }>({
  name: '',
  status: undefined,
})
const pagination = ref({ page: 1, pageSize: 10, total: 0 })

const canDelete = computed(() => menuStore.hasButtonPermission(PERM.ROLE_DELETE))

const statusOptions = computed<{ label: string; value: RoleStatus | undefined }[]>(() => [
  { label: t('system.common.all'), value: undefined },
  { label: t('system.common.enabled'), value: RoleStatus.RoleStatus_Active },
  { label: t('system.common.inactive'), value: RoleStatus.RoleStatus_InActive },
])

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('system.common.roleName'), minWidth: 160 },
  { key: 'description', title: t('system.common.roleDescription'), minWidth: 200 },
  { key: 'isBuiltin', title: t('system.common.type'), width: 110 },
  { key: 'status', title: t('system.common.status'), width: 110 },
  { key: 'createTime', title: t('system.common.createTime'), width: 170 },
  { key: 'updateTime', title: t('system.common.updateTime'), width: 170 },
  { key: 'operation', title: t('system.common.operation'), width: 90, align: 'center' },
])

const rowActions = computed<ActionMenuItem[]>(() => [
  {
    key: 'edit',
    label: t('system.common.edit'),
    icon: 'HOutline:PencilSquareIcon',
    hidden: !menuStore.hasButtonPermission(PERM.ROLE_EDIT),
  },
  {
    key: 'delete',
    label: t('system.common.delete'),
    icon: 'HOutline:TrashIcon',
    danger: true,
    hidden: !canDelete.value,
  },
])

const statusVariant = (s: unknown) => (s === RoleStatus.RoleStatus_Active ? 'success' : 'neutral')
const statusLabel = (s: unknown) =>
  s === RoleStatus.RoleStatus_Active ? t('system.common.enabled') : t('system.common.inactive')
const fmtTime = (v: unknown) => (v ? dayjs(v as string | Date).format('YYYY-MM-DD HH:mm') : '—')

const toggleSelect = (id: string) => {
  selectedIds.value = selectedSet.value.has(id)
    ? selectedIds.value.filter((x) => x !== id)
    : [...selectedIds.value, id]
}

const getRoleList = async () => {
  loading.value = true
  try {
    const { data: res } = await rolePage({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      name: queryForm.value.name || undefined,
      status: queryForm.value.status,
    })
    roleList.value = res?.roles || []
    pagination.value.total = Number(res?.total || 0)
    pagination.value.page = Number(res?.page || pagination.value.page)
    pagination.value.pageSize = Number(res?.pageSize || pagination.value.pageSize)
  } finally {
    loading.value = false
  }
}

const search = () => {
  pagination.value.page = 1
  getRoleList()
}
const reset = () => {
  queryForm.value = { name: '', status: undefined }
  pagination.value.page = 1
  getRoleList()
}

const openCreate = () => roleCreateRef.value?.showDialog()
const openEdit = (row: RoleInfo) => {
  if (row.isBuiltin) return
  roleCreateRef.value?.showDialog(row.roleId)
}

const onRowAction = (key: string, row: Record<string, unknown>) => {
  const roleId = String(row.roleId)
  if (key === 'edit') roleCreateRef.value?.showDialog(roleId)
  else if (key === 'delete') confirmDelete([roleId])
}

const confirmDelete = (ids: string[]) => {
  // 内置角色不可删（双保险，UI 已禁用其选择/操作）
  const deletable = ids.filter((id) => !roleList.value.find((r) => r.roleId === id)?.isBuiltin)
  if (!deletable.length) return
  Dialog.info({
    showCancelButton: true,
    content: t('system.role.confirmDeleteSelected'),
    confirmText: t('system.common.delete'),
    cancelText: t('system.common.cancel'),
    onConfirm: async () => {
      await deleteRole({ roleIds: deletable })
      selectedIds.value = selectedIds.value.filter((id) => !deletable.includes(id))
      feedback.success(t('system.common.deleteSuccess'))
      getRoleList()
    },
  })
}

const refresh = (type: 'create' | 'update') => {
  if (type === 'create') pagination.value.page = 1
  getRoleList()
}

onMounted(getRoleList)
</script>

<style scoped lang="scss">
.role-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* 批量条 */
.role-page__batch {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 16px;
  border: 1px solid color-mix(in srgb, var(--el-color-primary) 30%, var(--el-border-color-lighter));
  border-radius: var(--app-radius-lg);
  background: color-mix(in srgb, var(--el-color-primary) 5%, var(--el-bg-color));
}
.role-page__batch-count {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-color-primary);
}
.role-page__batch-actions {
  display: flex;
  gap: 8px;
}

/* 主体 */
.role-page__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.role-page__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.role-page__bar-hint {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}

/* 卡/表切换 分段控件 */
.seg {
  display: inline-flex;
  padding: 3px;
  gap: 2px;
  background: var(--el-fill-color-light);
  border-radius: var(--app-radius);
}
.seg__btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 12px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-sm);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}
.seg__btn :deep(svg) {
  width: 15px;
  height: 15px;
}
.seg__btn.is-active {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}

/* 卡片流 */
.role-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 12px;
}
@media (width >= 560px) {
  .role-grid {
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  }
}
.role-card.is-selected {
  border-color: color-mix(in srgb, var(--el-color-primary) 55%, var(--el-border-color));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--el-color-primary) 30%, transparent);
}
.role-card__id {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.role-card__check {
  width: 15px;
  height: 15px;
  accent-color: var(--el-color-primary);
  cursor: pointer;
  flex-shrink: 0;
}
.role-card__lock {
  display: inline-flex;
  flex-shrink: 0;
  color: var(--el-text-color-placeholder);
}
.role-card__lock :deep(svg) {
  width: 15px;
  height: 15px;
}
.role-card__name {
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.role-card__desc {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}
.role-card__actions {
  display: flex;
  gap: 6px;
}
.text-dim {
  color: var(--el-text-color-placeholder);
}

/* 卡片骨架 */
.role-skeleton {
  height: 150px;
  border-radius: var(--app-radius-lg);
  background: linear-gradient(
    100deg,
    var(--el-fill-color-light) 30%,
    var(--el-fill-color) 50%,
    var(--el-fill-color-light) 70%
  );
  background-size: 200% 100%;
  animation: role-shimmer 1.4s ease-in-out infinite;
}
@keyframes role-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
</style>
