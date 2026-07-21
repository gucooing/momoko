<!-- 菜单管理（重写 · P1 树表型）：PageHeader + FilterBar + DataTable(tree) / 移动缩进卡片 + FormDialog(create.vue)。
     菜单为树结构（无分页/批量）；系统内置菜单可编辑不可删。保留 adminPermissionsList/adminDeletePermissions 接口、PERM 权限（06d）。 -->
<template>
  <div class="menu-page">
    <PageHeader :title="t('system.menu.title')" :description="t('system.menu.pageDesc')">
      <template #actions>
        <UButton
          v-permission="[PERM.MENU_ADD]"
          color="primary"
          icon="i-lucide-plus"
          @click="openCreate()"
        >
          {{ t('system.menu.addMenu') }}
        </UButton>
      </template>
    </PageHeader>

    <FilterBar @search="() => {}" @reset="reset">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('system.common.menuTitle') }}</label>
          <input v-model="queryForm.title" class="app-input" :placeholder="t('system.menu.inputPlaceholder')" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('system.common.menuPath') }}</label>
          <input v-model="queryForm.path" class="app-input" :placeholder="t('system.menu.inputPlaceholder')" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('system.common.type') }}</label>
          <AppSelect v-model="queryForm.type" :options="typeOptions" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('system.common.status') }}</label>
          <AppSelect v-model="queryForm.status" :options="statusOptions" />
        </div>
      </template>
    </FilterBar>

    <div class="menu-page__body">
      <div class="menu-page__bar">
        <span class="menu-page__bar-hint">{{ t('system.common.total', { total: flatCount }) }}</span>
      </div>

      <!-- 桌面：树表 -->
      <DataTable
        v-if="!menuStore.isMobile"
        :columns="columns"
        :rows="(displayList as unknown as Record<string, unknown>[])"
        row-key="id"
        tree
        :loading="loading"
      >
        <template #cell-type="{ row }">
          <StatusPill :variant="typeVariant(row.type)" :dot="false" :label="typeLabel(row.type)" />
        </template>
        <template #cell-icon="{ row }">
          <component
            :is="menuStore.iconComponents[row.icon as string]"
            v-if="row.icon && menuStore.iconComponents[row.icon as string]"
            class="menu-icon"
          />
          <span v-else class="text-dim">—</span>
        </template>
        <template #cell-status="{ row }">
          <StatusPill :variant="statusVariant(row.status)" :label="statusLabel(row.status)" />
        </template>
        <template #cell-createTime="{ row }">{{ fmtTime(row.createTime) }}</template>
        <template #cell-operation="{ row }">
          <ActionMenu :items="actionsFor(row)" @select="(key) => onRowAction(key, row)" />
        </template>
      </DataTable>

      <!-- 移动：缩进卡片 -->
      <template v-else>
        <div v-if="loading" class="menu-cards">
          <div v-for="i in 6" :key="i" class="menu-skeleton" />
        </div>
        <EmptyState
          v-else-if="!flatList.length"
          icon="HOutline:Bars3BottomLeftIcon"
          :title="t('system.common.noData')"
          :description="t('system.menu.emptyDesc')"
        >
          <template #action>
            <UButton
              v-permission="[PERM.MENU_ADD]"
              color="primary"
              variant="soft"
              icon="i-lucide-plus"
              @click="openCreate()"
            >
              {{ t('system.menu.addMenu') }}
            </UButton>
          </template>
        </EmptyState>
        <div v-else class="menu-cards">
          <EntityCard
            v-for="row in flatList"
            :key="row.id"
            class="menu-card"
            :style="{ marginInlineStart: `${row._depth * 16}px` }"
          >
            <template #title>
              <div class="menu-card__title">
                <component
                  :is="menuStore.iconComponents[row.icon]"
                  v-if="row.icon && menuStore.iconComponents[row.icon]"
                  class="menu-icon"
                />
                <span class="menu-card__name">{{ row.title }}</span>
              </div>
            </template>
            <template #status>
              <StatusPill :variant="statusVariant(row.status)" :label="statusLabel(row.status)" />
            </template>
            <template #meta>
              <StatusPill :variant="typeVariant(row.type)" :dot="false" :label="typeLabel(row.type)" />
              <span v-if="row.path" class="menu-card__path">{{ row.path }}</span>
              <span class="menu-card__order">{{ t('system.menu.orderLabel', { order: row.order }) }}</span>
            </template>
            <template #footer>
              <span>{{ fmtTime(row.createTime) }}</span>
              <div class="menu-card__actions">
                <UButton
                  v-permission="[PERM.MENU_EDIT]"
                  color="neutral"
                  variant="soft"
                  size="xs"
                  icon="i-lucide-pencil"
                  @click="openCreate(row.id)"
                >
                  {{ t('system.common.edit') }}
                </UButton>
                <UButton
                  v-if="!isSystemMenu(asRec(row))"
                  v-permission="[PERM.MENU_DELETE]"
                  color="error"
                  variant="soft"
                  size="xs"
                  icon="i-lucide-trash-2"
                  @click="confirmDelete(asRec(row))"
                >
                  {{ t('system.common.delete') }}
                </UButton>
              </div>
            </template>
          </EntityCard>
        </div>
      </template>
    </div>

    <MenuCreate ref="menuCreateRef" @refresh="refresh" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { adminPermissionsList, adminDeletePermissions } from '@/api/menu'
import { PERM } from '@/config/permission'
import { Dialog } from '@/utils/dialog'
import { translateKnownText } from '@/locales'
import { MenuType, MenuStatus, type MenuInfo } from '@/types/v1/system'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import MenuCreate from '@/views/system/menu/create.vue'

defineOptions({ name: 'MenuView' })

const menuStore = useMenuStore()
const { t } = useI18n()
const menuCreateRef = useTemplateRef<InstanceType<typeof MenuCreate> | null>('menuCreateRef')

const loading = ref(false)
const menuList = ref<MenuInfo[]>([])
const queryForm = ref({ title: '', path: '', type: '' as string, status: '' as string })

const typeOptions = computed<{ label: string; value: string }[]>(() => [
  { label: t('system.common.all'), value: '' },
  { label: t('system.common.directory'), value: MenuType.MenuType_Directory },
  { label: t('system.common.menu'), value: MenuType.MenuType_Menu },
  { label: t('system.common.button'), value: MenuType.MenuType_Button },
])
const statusOptions = computed<{ label: string; value: string }[]>(() => [
  { label: t('system.common.all'), value: '' },
  { label: t('system.common.enabled'), value: MenuStatus.MenuStatus_Active },
  { label: t('system.common.disabled'), value: MenuStatus.MenuStatus_InActive },
])

const columns = computed<DataTableColumn[]>(() => [
  { key: 'title', title: t('system.common.menuTitle'), minWidth: 220 },
  { key: 'path', title: t('system.common.menuPath'), minWidth: 220 },
  { key: 'type', title: t('system.common.type'), width: 110 },
  { key: 'icon', title: t('system.common.icon'), width: 80, align: 'center' },
  { key: 'order', title: t('system.common.sort'), width: 80, align: 'center' },
  { key: 'status', title: t('system.common.status'), width: 110 },
  { key: 'createTime', title: t('system.common.createTime'), width: 170 },
  { key: 'operation', title: t('system.common.operation'), width: 90, align: 'center' },
])

// —— 客户端过滤（保留有匹配后代的祖先），再翻译标题 ——
const filterTree = (menus: MenuInfo[], predicate: (m: MenuInfo) => boolean): MenuInfo[] =>
  menus
    .map((m) => {
      const children = m.children?.length ? filterTree(m.children, predicate) : []
      if (predicate(m) || children.length) {
        return { ...m, children: children.length ? children : m.children } as MenuInfo
      }
      return null
    })
    .filter((m): m is MenuInfo => m !== null)

const filteredList = computed(() => {
  let list = menuList.value
  const q = queryForm.value
  if (q.title) list = filterTree(list, (m) => m.title.includes(q.title))
  if (q.path) list = filterTree(list, (m) => (m.path || '').includes(q.path))
  if (q.type) list = filterTree(list, (m) => m.type === q.type)
  if (q.status) list = filterTree(list, (m) => m.status === q.status)
  return list
})

const translateTree = (menus: MenuInfo[]): MenuInfo[] =>
  menus.map((m) => ({
    ...m,
    title: translateKnownText(m.title) || m.title,
    children: m.children?.length ? translateTree(m.children) : m.children,
  }))

const displayList = computed(() => translateTree(filteredList.value))

// 扁平化（带深度）供移动卡片 + 计数
const flatten = (menus: MenuInfo[], depth = 0): (MenuInfo & { _depth: number })[] =>
  menus.flatMap((m) => [
    { ...m, _depth: depth },
    ...(m.children?.length ? flatten(m.children, depth + 1) : []),
  ])
const flatList = computed(() => flatten(displayList.value))
const flatCount = computed(() => flatList.value.length)

const typeLabel = (type: unknown) => {
  if (type === MenuType.MenuType_Directory) return t('system.common.directory')
  if (type === MenuType.MenuType_Menu) return t('system.common.menu')
  return t('system.common.button')
}
const typeVariant = (type: unknown): 'info' | 'primary' | 'warning' => {
  if (type === MenuType.MenuType_Directory) return 'info'
  if (type === MenuType.MenuType_Menu) return 'primary'
  return 'warning'
}
const statusVariant = (s: unknown) => (s === MenuStatus.MenuStatus_Active ? 'success' : 'neutral')
const statusLabel = (s: unknown) =>
  s === MenuStatus.MenuStatus_Active ? t('system.common.enabled') : t('system.common.disabled')
const fmtTime = (v: unknown) => (v ? dayjs(v as string | Date).format('YYYY-MM-DD HH:mm') : '—')

// MenuInfo 递归 children 使其无隐式索引签名、不能直接当 Record 传参，这里做窄化转换。
const asRec = (r: unknown) => r as Record<string, unknown>

const isSystemMenu = (menu: Record<string, unknown>) => Boolean(menu.isSystem)

const actionsFor = (row: Record<string, unknown>): ActionMenuItem[] => [
  {
    key: 'edit',
    label: t('system.common.edit'),
    icon: 'HOutline:PencilSquareIcon',
    hidden: !menuStore.hasButtonPermission(PERM.MENU_EDIT),
  },
  {
    key: 'delete',
    label: t('system.common.delete'),
    icon: 'HOutline:TrashIcon',
    danger: true,
    hidden: isSystemMenu(row) || !menuStore.hasButtonPermission(PERM.MENU_DELETE),
  },
]

const getMenuList = async () => {
  loading.value = true
  try {
    const { data: res } = await adminPermissionsList({})
    menuList.value = res?.menus || []
  } finally {
    loading.value = false
  }
}

const reset = () => {
  queryForm.value = { title: '', path: '', type: '', status: '' }
}

const openCreate = (menuId?: string) => menuCreateRef.value?.showDialog(menuId)

const onRowAction = (key: string, row: Record<string, unknown>) => {
  if (key === 'edit') openCreate(String(row.id))
  else if (key === 'delete') confirmDelete(row)
}

const confirmDelete = (row: Record<string, unknown>) => {
  if (isSystemMenu(row)) return
  Dialog.info({
    showCancelButton: true,
    content: t('system.menu.confirmDeleteSelected'),
    confirmText: t('system.common.delete'),
    cancelText: t('system.common.cancel'),
    onConfirm: async () => {
      await adminDeletePermissions({ menuId: String(row.id) })
      feedback.success(t('system.common.deleteSuccess'))
      getMenuList()
    },
  })
}

const refresh = () => getMenuList()

onMounted(getMenuList)
</script>

<style scoped lang="scss">
.menu-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.menu-page__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.menu-page__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.menu-page__bar-hint {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.menu-icon {
  width: 16px;
  height: 16px;
  color: var(--el-text-color-regular);
  vertical-align: middle;
}
.text-dim {
  color: var(--el-text-color-placeholder);
}

/* 移动卡片 */
.menu-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.menu-card__title {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.menu-card__name {
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.menu-card__path {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
  font-variant-numeric: tabular-nums;
}
.menu-card__order {
  color: var(--el-text-color-placeholder);
}
.menu-card__actions {
  display: flex;
  gap: 6px;
}

/* 卡片骨架 */
.menu-skeleton {
  height: 120px;
  border-radius: var(--app-radius-lg);
  background: linear-gradient(
    100deg,
    var(--el-fill-color-light) 30%,
    var(--el-fill-color) 50%,
    var(--el-fill-color-light) 70%
  );
  background-size: 200% 100%;
  animation: menu-shimmer 1.4s ease-in-out infinite;
}
@keyframes menu-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
</style>
