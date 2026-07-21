<!-- 用户管理（重写 · P1 列表/CRUD 样板）：PageHeader + FilterBar + 批量条 + 卡/表切换
     + EntityCard 卡片流 / DataTable 表视图 + Pagination + FormDialog。
     保留 userPage/deleteUser 接口、PERM 权限、Dialog.info 删除确认（06d）。 -->
<template>
  <div class="user-page">
    <PageHeader :title="t('system.user.title')" :description="t('system.user.pageDesc')">
      <template #actions>
        <UButton
          v-permission="[PERM.USER_ADD]"
          color="primary"
          icon="i-lucide-plus"
          @click="openCreate"
        >
          {{ t('system.user.addUser') }}
        </UButton>
      </template>
    </PageHeader>

    <FilterBar @search="search" @reset="reset">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('system.common.username') }}</label>
          <input
            v-model="queryForm.username"
            class="app-input"
            :placeholder="t('system.user.usernamePlaceholder')"
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
    <div v-if="selectedIds.length && canDelete" class="user-page__batch">
      <span class="user-page__batch-count">{{ t('system.user.selectedCount', { count: selectedIds.length }) }}</span>
      <div class="user-page__batch-actions">
        <UButton color="neutral" variant="ghost" size="sm" @click="selectedIds = []">
          {{ t('system.user.clearSelection') }}
        </UButton>
        <UButton color="error" variant="soft" size="sm" icon="i-lucide-trash-2" @click="confirmDelete(selectedIds)">
          {{ t('system.user.batchDelete') }}
        </UButton>
      </div>
    </div>

    <!-- 主体：卡/表切换（桌面）；移动强制卡片 -->
    <div class="user-page__body">
      <div class="user-page__bar">
        <span class="user-page__bar-hint">{{ t('system.common.total', { total: pagination.total }) }}</span>
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
        <div v-if="loading" class="user-grid">
          <div v-for="i in 6" :key="i" class="user-skeleton" />
        </div>
        <EmptyState
          v-else-if="!userList.length"
          icon="HOutline:UserGroupIcon"
          :title="t('system.common.noData')"
          :description="t('system.user.emptyDesc')"
        >
          <template #action>
            <UButton
              v-permission="[PERM.USER_ADD]"
              color="primary"
              variant="soft"
              icon="i-lucide-plus"
              @click="openCreate"
            >
              {{ t('system.user.addUser') }}
            </UButton>
          </template>
        </EmptyState>
        <div v-else class="user-grid">
          <EntityCard
            v-for="row in userList"
            :key="row.userId"
            class="user-card"
            :class="{ 'is-selected': selectedSet.has(row.userId) }"
          >
            <template #title>
              <div class="user-card__id">
                <input
                  type="checkbox"
                  class="user-card__check"
                  :checked="selectedSet.has(row.userId)"
                  @change="toggleSelect(row.userId)"
                />
                <AppAvatar :src="avatarOf(row)" :size="38" :name="row.username" />
                <div class="user-card__name">
                  <span class="user-card__username">{{ row.username }}</span>
                  <span class="user-card__realname">{{ row.name || '—' }}</span>
                </div>
              </div>
            </template>
            <template #status>
              <StatusPill :variant="statusVariant(row.status)" :label="statusLabel(row.status)" />
            </template>
            <template #meta>
              <span class="user-card__email">{{ row.email || '—' }}</span>
              <StatusPill
                v-if="row.roleName"
                variant="primary"
                :dot="false"
                :label="row.roleName"
              />
            </template>
            <template #footer>
              <span>{{ fmtTime(row.createTime) }}</span>
              <div class="user-card__actions">
                <UButton
                  v-permission="[PERM.USER_EDIT]"
                  color="neutral"
                  variant="soft"
                  size="xs"
                  icon="i-lucide-pencil"
                  @click="openEdit(row)"
                >
                  {{ t('system.common.edit') }}
                </UButton>
                <UButton
                  v-permission="[PERM.USER_DELETE]"
                  color="error"
                  variant="soft"
                  size="xs"
                  icon="i-lucide-trash-2"
                  @click="confirmDelete([row.userId])"
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
        :rows="userList"
        row-key="userId"
        selectable
        :loading="loading"
      >
        <template #cell-roleName="{ row }">
          <StatusPill v-if="row.roleName" variant="primary" :dot="false" :label="String(row.roleName)" />
          <span v-else class="text-dim">—</span>
        </template>
        <template #cell-status="{ row }">
          <StatusPill :variant="statusVariant(row.status)" :label="statusLabel(row.status)" />
        </template>
        <template #cell-createTime="{ row }">{{ fmtTime(row.createTime) }}</template>
        <template #cell-operation="{ row }">
          <ActionMenu :items="rowActions" @select="(key) => onRowAction(key, row)" />
        </template>
      </DataTable>

      <Pagination
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        @change="getUserList"
      />
    </div>

    <UserCreate ref="userCreateRef" @refresh="refresh" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { deleteUser, userPage } from '@/api/user'
import { PERM } from '@/config/permission'
import { Dialog } from '@/utils/dialog'
import { resolveAvatarUrl } from '@/utils/assets'
import { UserStatus, type UserInfo } from '@/types/v1/user'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import UserCreate from '@/views/system/user/create.vue'

defineOptions({ name: 'UserView' })

const menuStore = useMenuStore()
const { t } = useI18n()
const userCreateRef = useTemplateRef<InstanceType<typeof UserCreate> | null>('userCreateRef')

const viewMode = ref<'card' | 'table'>('card')
const loading = ref(false)
const userList = ref<UserInfo[]>([])
const selectedIds = ref<string[]>([])
const selectedSet = computed(() => new Set(selectedIds.value))

const queryForm = ref<{ username: string; status: UserStatus | undefined }>({
  username: '',
  status: undefined,
})
const pagination = ref({ page: 1, pageSize: 10, total: 0 })

const canDelete = computed(() => menuStore.hasButtonPermission(PERM.USER_DELETE))

const statusOptions = computed<{ label: string; value: UserStatus | undefined }[]>(() => [
  { label: t('system.common.all'), value: undefined },
  { label: t('system.common.enabled'), value: UserStatus.Active },
  { label: t('system.common.inactive'), value: UserStatus.InActive },
])

const columns = computed<DataTableColumn[]>(() => [
  { key: 'username', title: t('system.common.username'), minWidth: 150 },
  { key: 'name', title: t('system.common.name'), minWidth: 130 },
  { key: 'email', title: t('system.common.email'), minWidth: 180 },
  { key: 'roleName', title: t('system.common.role'), minWidth: 130 },
  { key: 'status', title: t('system.common.status'), width: 110 },
  { key: 'createTime', title: t('system.common.createTime'), width: 170 },
  { key: 'operation', title: t('system.common.operation'), width: 90, align: 'center' },
])

const rowActions = computed<ActionMenuItem[]>(() => [
  {
    key: 'edit',
    label: t('system.common.edit'),
    icon: 'HOutline:PencilSquareIcon',
    hidden: !menuStore.hasButtonPermission(PERM.USER_EDIT),
  },
  {
    key: 'delete',
    label: t('system.common.delete'),
    icon: 'HOutline:TrashIcon',
    danger: true,
    hidden: !canDelete.value,
  },
])

const avatarOf = (row: UserInfo) => resolveAvatarUrl(row.avatar)
const statusVariant = (s: unknown) => (s === UserStatus.Active ? 'success' : 'neutral')
const statusLabel = (s: unknown) =>
  s === UserStatus.Active ? t('system.common.enabled') : t('system.common.inactive')
const fmtTime = (v: unknown) => (v ? dayjs(v as string | Date).format('YYYY-MM-DD HH:mm') : '—')

const toggleSelect = (id: string) => {
  selectedIds.value = selectedSet.value.has(id)
    ? selectedIds.value.filter((x) => x !== id)
    : [...selectedIds.value, id]
}

const getUserList = async () => {
  loading.value = true
  try {
    const { data: res } = await userPage({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      username: queryForm.value.username || undefined,
      status: queryForm.value.status,
    })
    userList.value = res?.users || []
    pagination.value.total = Number(res?.total || 0)
    pagination.value.page = Number(res?.page || pagination.value.page)
    pagination.value.pageSize = Number(res?.pageSize || pagination.value.pageSize)
  } finally {
    loading.value = false
  }
}

const search = () => {
  pagination.value.page = 1
  getUserList()
}
const reset = () => {
  queryForm.value = { username: '', status: undefined }
  pagination.value.page = 1
  getUserList()
}

const openCreate = () => userCreateRef.value?.showDialog()
const openEdit = (row: UserInfo) =>
  userCreateRef.value?.showDialog({ userId: row.userId, roleName: row.roleName })

const onRowAction = (key: string, row: Record<string, unknown>) => {
  const userId = String(row.userId)
  if (key === 'edit')
    userCreateRef.value?.showDialog({
      userId,
      roleName: row.roleName ? String(row.roleName) : undefined,
    })
  else if (key === 'delete') confirmDelete([userId])
}

const confirmDelete = (ids: string[]) => {
  if (!ids.length) return
  Dialog.info({
    showCancelButton: true,
    content: t('system.user.confirmDeleteSelected'),
    confirmText: t('system.common.delete'),
    cancelText: t('system.common.cancel'),
    onConfirm: async () => {
      await deleteUser({ userIds: ids })
      selectedIds.value = selectedIds.value.filter((id) => !ids.includes(id))
      feedback.success(t('system.common.deleteSuccess'))
      getUserList()
    },
  })
}

const refresh = (type: 'create' | 'update') => {
  if (type === 'create') pagination.value.page = 1
  getUserList()
}

onMounted(getUserList)
</script>

<style scoped lang="scss">
.user-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* 批量条 */
.user-page__batch {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 16px;
  border: 1px solid color-mix(in srgb, var(--el-color-primary) 30%, var(--el-border-color-lighter));
  border-radius: var(--app-radius-lg);
  background: color-mix(in srgb, var(--el-color-primary) 5%, var(--el-bg-color));
}
.user-page__batch-count {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-color-primary);
}
.user-page__batch-actions {
  display: flex;
  gap: 8px;
}

/* 主体 */
.user-page__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.user-page__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.user-page__bar-hint {
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
.user-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 12px;
}
@media (width >= 560px) {
  .user-grid {
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  }
}
.user-card.is-selected {
  border-color: color-mix(in srgb, var(--el-color-primary) 55%, var(--el-border-color));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--el-color-primary) 30%, transparent);
}
.user-card__id {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.user-card__check {
  width: 15px;
  height: 15px;
  accent-color: var(--el-color-primary);
  cursor: pointer;
  flex-shrink: 0;
}
.user-card__name {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.user-card__username {
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.user-card__realname {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.user-card__email {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}
.user-card__actions {
  display: flex;
  gap: 6px;
}
.text-dim {
  color: var(--el-text-color-placeholder);
}

/* 卡片骨架 */
.user-skeleton {
  height: 150px;
  border-radius: var(--app-radius-lg);
  background: linear-gradient(
    100deg,
    var(--el-fill-color-light) 30%,
    var(--el-fill-color) 50%,
    var(--el-fill-color-light) 70%
  );
  background-size: 200% 100%;
  animation: user-shimmer 1.4s ease-in-out infinite;
}
@keyframes user-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
</style>
