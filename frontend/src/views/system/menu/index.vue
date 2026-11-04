<!-- 菜单（第二期改版）：组标题 + 计数说明（目录/菜单/按钮）+ AccessSubNav；类型 chips（带数量）+ 状态分段 + 搜索
     （名称/路由/权限标识）；全部展开/收起 + 类型图例；PC 树表（缩进 + 展开箭头 + 贴纸图标、类型、路由、权限标识、排序、
     「显示」开关（只从侧边栏隐藏，页面与权限仍有效）、新增子项/编辑/启停/删除），手机树形列表（显示开关 + ⋯）+ 底部操作抽屉
     （新增子项/编辑/调整排序/启停/删除）+ 右下悬浮新增。系统内置菜单可编辑不可删。 -->
<template>
  <div class="mn-page">
    <PageHeader :title="t('access.nav.access')">
      <template #description>
        <span class="tnum">{{ t('access.menu.summary', { total: stats.total, dir: stats.dir, menu: stats.menu, btn: stats.btn }) }}</span>
        <template v-if="!menuStore.isMobile">
          <span class="mn-dot">·</span>{{ t('accessAdmin.menu.summaryTail') }}
        </template>
      </template>
      <template v-if="canAdd && !menuStore.isMobile" #actions>
        <UButton color="primary" icon="i-lucide-plus" @click="openCreate()">
          {{ t('system.menu.addMenu') }}
        </UButton>
      </template>
    </PageHeader>

    <AccessSubNav :menu-count="stats.total || null" />

    <ListToolbar
      v-model:status="typeFilter"
      v-model:keyword="keyword"
      :chips="typeChips"
      :chips-label="t('system.common.type')"
      :placeholder="t('access.menu.searchPlaceholder')"
    >
      <div class="mn-seg" role="group" :aria-label="t('system.common.status')">
        <button
          v-for="opt in statusOptions"
          :key="opt.value"
          type="button"
          class="mn-seg__btn"
          :class="{ 'is-on': statusFilter === opt.value }"
          :aria-pressed="statusFilter === opt.value"
          @click="statusFilter = opt.value"
        >
          {{ opt.label }}
        </button>
      </div>
    </ListToolbar>

    <div class="mn-tools">
      <UButton color="neutral" variant="outline" size="sm" icon="i-lucide-chevrons-up-down" :disabled="filtering" @click="expandAll">
        {{ t('access.menu.expandAll') }}
      </UButton>
      <UButton color="neutral" variant="outline" size="sm" icon="i-lucide-chevrons-down-up" :disabled="filtering" @click="collapseAll">
        {{ t('access.menu.collapseAll') }}
      </UButton>
      <div v-if="!menuStore.isMobile" class="mn-legend">
        <span class="mn-ty is-dir">{{ t('system.common.directory') }}</span>{{ t('access.menu.legendDir') }}
        <span class="mn-ty is-menu">{{ t('system.common.menu') }}</span>{{ t('access.menu.legendMenu') }}
        <span class="mn-ty is-btn">{{ t('system.common.button') }}</span>{{ t('access.menu.legendBtn') }}
      </div>
      <div v-else class="mn-legend is-compact" aria-hidden="true">
        <span class="mn-ty is-dir">{{ t('system.common.directory') }}</span>
        <span class="mn-ty is-menu">{{ t('system.common.menu') }}</span>
        <span class="mn-ty is-btn">{{ t('system.common.button') }}</span>
      </div>
    </div>

    <!-- PC：树表 -->
    <DataTable
      v-if="!menuStore.isMobile"
      :columns="columns"
      :rows="(visibleRows as unknown as Record<string, unknown>[])"
      row-key="id"
      :loading="loading"
      :empty-text="filtering ? t('accessAdmin.menu.emptyFiltered') : t('system.menu.emptyDesc')"
    >
      <template #cell-title="{ row }">
        <div class="mn-name" :style="{ paddingLeft: `${Number(row.depth) * 28}px` }">
          <button
            v-if="row.hasKids"
            type="button"
            class="mn-caret"
            :class="{ 'is-open': row.open }"
            :aria-label="`${row.open ? t('access.menu.collapseAll') : t('access.menu.expandAll')} ${row.label}`"
            :aria-expanded="!!row.open"
            :disabled="filtering"
            @click="toggleOpen(String(row.id))"
          >
            <component :is="menuStore.iconComponents['HOutline:ChevronRightIcon']" />
          </button>
          <span v-else class="mn-caret mn-caret--leaf"><span /></span>
          <span
            class="m-sticker mn-ico"
            :class="stickerCls(asRow(row))"
            :style="{ transform: `rotate(${ROTS[Number(row.rot)]}deg)` }"
          >
            <component :is="menuStore.iconComponents[iconOf(asRow(row))]" />
          </span>
          <span class="mn-name__text" :class="[`is-d${Math.min(Number(row.depth), 2)}`, { 'is-btn': row.type === MenuType.MenuType_Button, 'is-off': !row.active }]">
            {{ row.label }}
          </span>
          <span v-if="!row.active" class="mn-note">· {{ t('accessAdmin.menu.disabledNote') }}</span>
          <span v-else-if="row.hidden" class="mn-note">· {{ t('accessV2.menu.hiddenNote') }}</span>
          <span v-if="row.hasKids && !row.open" class="mn-kids tnum">{{ row.kidsCount }}</span>
        </div>
      </template>
      <template #cell-type="{ row }">
        <span class="mn-ty" :class="typeCls(row.type)">{{ typeLabel(row.type) }}</span>
      </template>
      <template #cell-path="{ row }">
        <span class="m-mono mn-mono" :class="{ 'is-dim': !row.path }">{{ row.path || '—' }}</span>
      </template>
      <template #cell-permissions="{ row }">
        <span class="m-mono mn-mono mn-mono--sm is-dim">{{ row.permissions || '—' }}</span>
      </template>
      <template #cell-order="{ row }">
        <span class="m-num mn-order">{{ row.order }}</span>
      </template>
      <template #cell-visible="{ row }">
        <AppSwitch
          v-if="row.type !== MenuType.MenuType_Button"
          :model-value="!row.hidden"
          :disabled="!canEdit || busyId === row.id"
          :aria-label="t('accessV2.menu.toggleVisible', { name: row.label })"
          @update:model-value="(v) => setVisible(asRow(row), v)"
        />
        <span v-else class="mn-mono is-dim">—</span>
      </template>
      <template #cell-operation="{ row }">
        <div class="mn-acts">
          <AppIconButton
            v-if="canAdd"
            icon="HOutline:PlusIcon"
            :label="t('accessAdmin.menu.addChildOf', { name: row.label })"
            :box="36"
            :size="18"
            :disabled="row.type === MenuType.MenuType_Button"
            :class="{ 'is-locked': row.type === MenuType.MenuType_Button }"
            @click="addChild(asRow(row))"
          />
          <AppIconButton
            v-if="canEdit"
            icon="HOutline:PencilSquareIcon"
            :label="`${t('system.common.edit')} ${row.label}`"
            :box="36"
            :size="18"
            @click="openCreate(String(row.id))"
          />
          <AppIconButton
            v-if="canEdit"
            :icon="row.active ? 'HOutline:NoSymbolIcon' : 'HOutline:CheckCircleIcon'"
            :label="t('accessAdmin.menu.toggleStatus', { name: row.label })"
            :box="36"
            :size="18"
            :class="row.active ? 'is-stop' : 'is-start'"
            :disabled="busyId === row.id"
            @click="setStatus(asRow(row), !row.active)"
          />
          <AppIconButton
            v-if="canDelete"
            icon="HOutline:TrashIcon"
            :label="row.isSystem ? t('accessAdmin.menu.systemLocked') : t('system.common.delete')"
            :box="36"
            :size="18"
            :disabled="!!row.isSystem"
            :class="row.isSystem ? 'is-locked' : 'is-danger'"
            @click="confirmDelete(asRow(row))"
          />
        </div>
      </template>
    </DataTable>

    <!-- 手机：树形列表 -->
    <template v-else>
      <div v-if="loading && !menuList.length" class="mn-skeleton" />
      <EmptyState
        v-else-if="!visibleRows.length"
        icon="HOutline:Bars3BottomLeftIcon"
        :title="filtering ? t('accessAdmin.menu.emptyFiltered') : t('system.common.noData')"
        :description="filtering ? '' : t('system.menu.emptyDesc')"
      />
      <div v-else class="mn-tree" role="tree" :aria-label="t('system.menu.title')">
        <div
          v-for="row in visibleRows"
          :key="row.id"
          class="mn-tr"
          role="treeitem"
          :aria-level="row.depth + 1"
          :aria-expanded="row.hasKids ? row.open : undefined"
          :style="{ paddingLeft: `${row.depth * 18}px` }"
        >
          <button
            v-if="row.hasKids"
            type="button"
            class="mn-caret mn-caret--lg"
            :class="{ 'is-open': row.open }"
            :aria-label="`${row.open ? t('access.menu.collapseAll') : t('access.menu.expandAll')} ${row.label}`"
            :disabled="filtering"
            @click="toggleOpen(row.id)"
          >
            <component :is="menuStore.iconComponents['HOutline:ChevronRightIcon']" />
          </button>
          <span v-else class="mn-caret mn-caret--lg mn-caret--leaf"><span /></span>
          <button type="button" class="mn-tr__main" :aria-label="t('accessAdmin.menu.actions', { name: row.label })" @click="openSheet(row)">
            <span class="m-sticker mn-ico" :class="stickerCls(row)" :style="{ transform: `rotate(${ROTS[row.rot]}deg)` }">
              <component :is="menuStore.iconComponents[iconOf(row)]" />
            </span>
            <span class="mn-tr__text">
              <span class="mn-tr__title" :class="{ 'is-off': !row.active || row.hidden }">
                <span class="mn-tr__label">{{ row.label }}</span>
                <span class="mn-ty mn-ty--xs" :class="typeCls(row.type)">{{ typeLabel(row.type) }}</span>
                <span v-if="row.hasKids && !row.open" class="mn-kids tnum">{{ row.kidsCount }}</span>
              </span>
              <span class="mn-tr__sub">
                <span v-if="!row.active" class="mn-tr__flag">{{ t('accessAdmin.menu.disabledNote') }}</span>
                <span class="m-mono">{{ row.path || row.permissions || t('accessAdmin.menu.noPath') }}</span>
              </span>
            </span>
          </button>
          <AppSwitch
            v-if="row.type !== MenuType.MenuType_Button"
            :model-value="!row.hidden"
            :disabled="!canEdit || busyId === row.id"
            :aria-label="t('accessV2.menu.toggleVisible', { name: row.label })"
            @update:model-value="(v) => setVisible(row, v)"
          />
          <AppIconButton
            icon="HOutline:EllipsisHorizontalIcon"
            :box="44"
            :label="t('accessAdmin.menu.actions', { name: row.label })"
            @click="openSheet(row)"
          />
        </div>
      </div>
    </template>

    <p class="mn-tip">
      <component :is="menuStore.iconComponents['HOutline:InformationCircleIcon']" />
      <span>{{ menuStore.isMobile ? t('accessV2.menu.footTipShort') : t('accessV2.menu.footTip') }}</span>
    </p>

    <!-- 手机：悬浮新增 -->
    <button
      v-if="menuStore.isMobile && canAdd"
      type="button"
      class="mn-fab"
      :aria-label="t('system.menu.addMenu')"
      @click="openCreate()"
    >
      <component :is="menuStore.iconComponents['HOutline:PlusIcon']" />
    </button>

    <!-- 手机：操作抽屉 -->
    <FormDialog
      v-if="menuStore.isMobile"
      v-model="sheetOpen"
      :title="sheetRow?.label || ''"
      :show-footer="false"
    >
      <div v-if="sheetRow" class="mn-sheet">
        <div class="mn-sheet__head">
          <span class="m-sticker mn-ico mn-ico--lg" :class="stickerCls(sheetRow)">
            <component :is="menuStore.iconComponents[iconOf(sheetRow)]" />
          </span>
          <span class="mn-sheet__title">
            <span class="mn-sheet__name">
              {{ sheetRow.label }}
              <span class="mn-ty" :class="typeCls(sheetRow.type)">{{ typeLabel(sheetRow.type) }}</span>
            </span>
            <span class="m-mono mn-sheet__sub">{{ sheetRow.path || sheetRow.permissions || t('accessAdmin.menu.noPath') }}</span>
          </span>
          <StatusPill
            :variant="sheetRow.active ? (sheetRow.hidden ? 'neutral' : 'success') : 'neutral'"
            :label="!sheetRow.active ? t('system.common.disabled') : sheetRow.hidden ? t('accessV2.menu.hiddenNote') : t('system.common.enabled')"
          />
        </div>
        <dl class="mn-dl">
          <div>
            <dt>{{ t('system.common.menuPath') }}</dt>
            <dd class="m-mono">{{ sheetRow.path || '—' }}</dd>
          </div>
          <div>
            <dt>{{ t('system.common.sort') }}</dt>
            <dd class="m-num">{{ sheetRow.order }}</dd>
          </div>
          <div class="mn-dl__full">
            <dt>{{ t('system.menu.permissions') }}</dt>
            <dd class="m-mono">{{ sheetRow.permissions || '—' }}</dd>
          </div>
        </dl>
        <div class="mn-sheet__acts">
          <button v-if="canAdd && sheetRow.type !== MenuType.MenuType_Button" type="button" class="mn-sheet__act" @click="sheetDo('add')">
            <component :is="menuStore.iconComponents['HOutline:PlusIcon']" />{{ t('access.menu.addChild') }}
          </button>
          <button v-if="canEdit" type="button" class="mn-sheet__act" @click="sheetDo('edit')">
            <component :is="menuStore.iconComponents['HOutline:PencilSquareIcon']" />{{ t('system.common.edit') }}
          </button>
          <button v-if="canEdit && !sheetRow.isSystem" type="button" class="mn-sheet__act" @click="sheetDo('order')">
            <component :is="menuStore.iconComponents['HOutline:ArrowsUpDownIcon']" />{{ t('accessV2.menu.adjustOrder') }}
          </button>
          <button v-if="canEdit" type="button" class="mn-sheet__act" @click="sheetDo('status')">
            <component :is="menuStore.iconComponents[sheetRow.active ? 'HOutline:NoSymbolIcon' : 'HOutline:CheckCircleIcon']" />
            {{ sheetRow.active ? t('accessV2.menu.disable') : t('accessV2.menu.enable') }}
          </button>
          <button
            v-if="canDelete"
            type="button"
            class="mn-sheet__act is-danger"
            :disabled="sheetRow.isSystem"
            @click="sheetDo('delete')"
          >
            <component :is="menuStore.iconComponents['HOutline:TrashIcon']" />
            {{ sheetRow.isSystem ? t('accessAdmin.menu.systemLocked') : t('system.common.delete') }}
          </button>
        </div>
        <UButton color="neutral" variant="soft" block class="mn-sheet__cancel" @click="sheetOpen = false">
          {{ t('system.common.cancel') }}
        </UButton>
      </div>
    </FormDialog>

    <!-- 调整排序 -->
    <FormDialog
      v-model="orderOpen"
      :title="t('accessV2.menu.orderTitle', { name: orderRow?.label || '' })"
      :width="420"
      :loading="!!busyId"
      @confirm="submitOrder"
    >
      <div class="mn-order-form">
        <label class="app-label" for="mn-order-input">{{ t('system.common.sort') }}</label>
        <div class="mn-order-form__row">
          <AppIconButton icon="HOutline:MinusIcon" :box="44" :label="t('accessV2.menu.orderUp')" @click="orderValue = Math.max(0, orderValue - 1)" />
          <input id="mn-order-input" v-model.number="orderValue" type="number" min="0" max="999" class="app-input m-num" />
          <AppIconButton icon="HOutline:PlusIcon" :box="44" :label="t('accessV2.menu.orderDown')" @click="orderValue = Math.min(999, orderValue + 1)" />
        </div>
        <p class="mn-order-form__hint">{{ t('accessV2.menu.orderHint') }}</p>
      </div>
    </FormDialog>

    <MenuCreate ref="menuCreateRef" @refresh="refresh" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { adminPermissionsList, adminDeletePermissions, adminEditPermissions } from '@/api/menu'
import { PERM } from '@/config/permission'
import { Dialog } from '@/utils/dialog'
import type { AdminEditPermissionsRequest } from '@/types/v1/system'
import { translateKnownText } from '@/locales'
import { MenuType, MenuStatus, type MenuInfo } from '@/types/v1/system'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'
import MenuCreate from '@/views/system/menu/create.vue'
import AccessSubNav from '@/views/system/components/AccessSubNav.vue'

defineOptions({ name: 'MenuView' })

const menuStore = useMenuStore()
const { t } = useI18n()
const menuCreateRef = useTemplateRef<InstanceType<typeof MenuCreate> | null>('menuCreateRef')
const ROTS = [-4, 3, -2, 4]

interface Row {
  id: string
  raw: MenuInfo
  label: string
  type: MenuType
  path: string
  permissions: string
  icon: string
  order: number
  active: boolean
  hidden: boolean
  isSystem: boolean
  depth: number
  hasKids: boolean
  kidsCount: number
  open: boolean
  rot: number
}

const loading = ref(false)
const menuList = ref<MenuInfo[]>([])
const keyword = ref('')
const typeFilter = ref<string>('')
const statusFilter = ref<string>('')
const opened = ref<Set<string>>(new Set())
const busyId = ref('')

const canAdd = computed(() => menuStore.hasButtonPermission(PERM.MENU_ADD))
const canEdit = computed(() => menuStore.hasButtonPermission(PERM.MENU_EDIT))
const canDelete = computed(() => menuStore.hasButtonPermission(PERM.MENU_DELETE))

const tr = (s: string) => translateKnownText(s) || s

const flatAll = computed(() => {
  const out: MenuInfo[] = []
  const walk = (list: MenuInfo[]) =>
    list.forEach((m) => {
      out.push(m)
      if (m.children?.length) walk(m.children)
    })
  walk(menuList.value)
  return out
})
const stats = computed(() => {
  const s = { total: 0, dir: 0, menu: 0, btn: 0 }
  for (const m of flatAll.value) {
    s.total++
    if (m.type === MenuType.MenuType_Directory) s.dir++
    else if (m.type === MenuType.MenuType_Menu) s.menu++
    else s.btn++
  }
  return s
})

const typeChips = computed<ToolbarChip<string>[]>(() => [
  { label: t('system.common.all'), value: '', count: stats.value.total },
  { label: t('system.common.directory'), value: MenuType.MenuType_Directory, count: stats.value.dir },
  { label: t('system.common.menu'), value: MenuType.MenuType_Menu, count: stats.value.menu },
  { label: t('system.common.button'), value: MenuType.MenuType_Button, count: stats.value.btn },
])
const statusOptions = computed(() => [
  { label: t('access.menu.allStatus'), value: '' },
  { label: t('system.common.enabled'), value: MenuStatus.MenuStatus_Active as string },
  { label: t('system.common.disabled'), value: MenuStatus.MenuStatus_InActive as string },
])

const columns = computed<DataTableColumn[]>(() => [
  { key: 'title', title: t('system.common.menuTitle'), minWidth: 300 },
  { key: 'type', title: t('system.common.type'), width: 90 },
  { key: 'path', title: t('system.common.menuPath'), minWidth: 170 },
  { key: 'permissions', title: t('system.menu.permissions'), minWidth: 160 },
  { key: 'order', title: t('system.common.sort'), width: 70, align: 'center' },
  { key: 'visible', title: t('accessV2.menu.colVisible'), width: 80 },
  { key: 'operation', title: t('system.common.operation'), width: 180, align: 'right' },
])

const filtering = computed(() => !!(keyword.value.trim() || typeFilter.value || statusFilter.value))
const matches = (m: MenuInfo) => {
  const kw = keyword.value.trim().toLowerCase()
  if (kw) {
    const hay = `${m.title} ${tr(m.title)} ${m.path || ''} ${m.permissions || ''}`.toLowerCase()
    if (!hay.includes(kw)) return false
  }
  if (typeFilter.value && m.type !== typeFilter.value) return false
  if (statusFilter.value && m.status !== statusFilter.value) return false
  return true
}

const visibleRows = computed<Row[]>(() => {
  const rows: Row[] = []
  // 过滤时：保留命中项及其祖先，全部展开
  const keep = (m: MenuInfo): boolean => matches(m) || (m.children || []).some(keep)
  const walk = (list: MenuInfo[], depth: number) => {
    list.forEach((m, i) => {
      if (filtering.value && !keep(m)) return
      const kids = m.children || []
      const open = filtering.value || opened.value.has(m.id)
      rows.push({
        id: m.id,
        raw: m,
        label: tr(m.title),
        type: m.type,
        path: m.path || '',
        permissions: m.permissions || '',
        icon: m.icon || '',
        order: m.order ?? 0,
        active: m.status === MenuStatus.MenuStatus_Active,
        hidden: !!m.hidden,
        isSystem: !!m.isSystem,
        depth,
        hasKids: kids.length > 0,
        kidsCount: kids.length,
        open,
        rot: (i + depth) % ROTS.length,
      })
      if (kids.length && open) walk(kids, depth + 1)
    })
  }
  walk(menuList.value, 0)
  return rows
})

const asRow = (r: Record<string, unknown>) => r as unknown as Row

const typeLabel = (type: unknown) => {
  if (type === MenuType.MenuType_Directory) return t('system.common.directory')
  if (type === MenuType.MenuType_Menu) return t('system.common.menu')
  return t('system.common.button')
}
const typeCls = (type: unknown) =>
  type === MenuType.MenuType_Directory ? 'is-dir' : type === MenuType.MenuType_Menu ? 'is-menu' : 'is-btn'
const stickerCls = (row: Row) => {
  if (!row.active) return 'mn-ico--off'
  if (row.type === MenuType.MenuType_Directory) return 'm-sticker--a2'
  if (row.type === MenuType.MenuType_Menu) return 'm-sticker--p'
  return 'm-sticker--a'
}
const iconOf = (row: Row) =>
  row.icon || (row.type === MenuType.MenuType_Button ? 'HOutline:CursorArrowRaysIcon' : 'HOutline:Squares2X2Icon')

const toggleOpen = (id: string) => {
  const next = new Set(opened.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  opened.value = next
}
const expandAll = () => {
  opened.value = new Set(flatAll.value.filter((m) => m.children?.length).map((m) => m.id))
}
const collapseAll = () => {
  opened.value = new Set()
}

const getMenuList = async (initial = false) => {
  loading.value = true
  try {
    const { data: res } = await adminPermissionsList({})
    menuList.value = res?.menus || []
    if (initial) {
      // 默认展开目录，菜单下的按钮收起
      opened.value = new Set(
        flatAll.value.filter((m) => m.type === MenuType.MenuType_Directory && m.children?.length).map((m) => m.id),
      )
    }
  } finally {
    loading.value = false
  }
}

// 编辑接口要求完整字段：以当前菜单为底，只改传入的部分
const patchMenu = async (row: Row, patch: Partial<AdminEditPermissionsRequest>) => {
  const m = row.raw
  busyId.value = row.id
  try {
    await adminEditPermissions({
      menuId: m.id,
      title: m.title,
      icon: m.icon || '',
      order: m.order ?? 0,
      path: m.path || '',
      status: m.status,
      permissions: m.permissions || '',
      hidden: !!m.hidden,
      ...patch,
    })
    if (patch.status !== undefined) m.status = patch.status
    if (patch.hidden !== undefined) m.hidden = patch.hidden
    if (patch.order !== undefined) m.order = patch.order
    return true
  } catch {
    return false
  } finally {
    busyId.value = ''
  }
}

const setStatus = async (row: Row, on: boolean) => {
  if (!canEdit.value) return
  const status = on ? MenuStatus.MenuStatus_Active : MenuStatus.MenuStatus_InActive
  if (await patchMenu(row, { status }))
    feedback.success(t(on ? 'accessAdmin.menu.statusOn' : 'accessAdmin.menu.statusOff', { name: row.label }))
}

// 「显示」：只影响侧边栏，刷新当前用户的菜单让侧边栏立即生效
const setVisible = async (row: Row, visible: boolean) => {
  if (!canEdit.value) return
  if (await patchMenu(row, { hidden: !visible })) {
    feedback.success(t(visible ? 'accessV2.menu.shown' : 'accessV2.menu.hidden', { name: row.label }))
    void menuStore.getUserPermissions().catch(() => undefined)
  }
}

// 调整排序（手机抽屉）
const orderOpen = ref(false)
const orderRow = ref<Row | null>(null)
const orderValue = ref(0)
const openOrder = (row: Row) => {
  orderRow.value = row
  orderValue.value = row.order
  orderOpen.value = true
}
const submitOrder = async () => {
  const row = orderRow.value
  if (!row) return
  const order = Math.min(999, Math.max(0, Math.round(Number(orderValue.value) || 0)))
  if (await patchMenu(row, { order })) {
    orderOpen.value = false
    feedback.success(t('system.common.editSuccess'))
    getMenuList()
  }
}

const openCreate = (menuId?: string) => menuCreateRef.value?.showDialog(menuId)
const addChild = (row: Row) => {
  if (row.type === MenuType.MenuType_Button) return
  menuCreateRef.value?.showDialog(undefined, {
    parentId: row.id,
    type: row.type === MenuType.MenuType_Directory ? MenuType.MenuType_Menu : MenuType.MenuType_Button,
  })
}

const confirmDelete = (row: Row) => {
  if (row.isSystem) return
  Dialog.info({
    showCancelButton: true,
    content: t('system.menu.confirmDeleteSelected'),
    confirmText: t('system.common.delete'),
    cancelText: t('system.common.cancel'),
    onConfirm: async () => {
      await adminDeletePermissions({ menuId: row.id })
      feedback.success(t('system.common.deleteSuccess'))
      sheetOpen.value = false
      getMenuList()
    },
  })
}

// 手机操作抽屉
const sheetOpen = ref(false)
const sheetRow = ref<Row | null>(null)
const openSheet = (row: Row) => {
  sheetRow.value = row
  sheetOpen.value = true
}
const sheetDo = (action: 'add' | 'edit' | 'delete' | 'order' | 'status') => {
  const row = sheetRow.value
  if (!row) return
  if (action === 'delete') return confirmDelete(row)
  sheetOpen.value = false
  if (action === 'add') addChild(row)
  else if (action === 'order') openOrder(row)
  else if (action === 'status') void setStatus(row, !row.active)
  else openCreate(row.id)
}

const refresh = () => getMenuList()

onMounted(() => getMenuList(true))
</script>

<style scoped lang="scss">
.mn-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.mn-dot {
  margin: 0 6px;
  opacity: 0.6;
}

/* 状态分段 */
.mn-seg {
  display: inline-flex;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  background: var(--m-s2);
}
.mn-seg__btn {
  min-height: 34px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.mn-seg__btn.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}

/* 工具行 */
.mn-tools {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  margin-top: -4px;
}
.mn-tools :deep(button) {
  min-height: 40px;
  border-radius: 999px;
  font-weight: 700;
}
.mn-legend {
  display: inline-flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 10px;
  margin-left: 8px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.mn-legend .mn-ty:not(:first-child) {
  margin-left: 8px;
}

/* 类型标签 */
.mn-ty {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 7px;
  font-size: 0.6875rem;
  font-weight: 800;
  white-space: nowrap;
}
.mn-ty.is-dir { background: var(--m-a2Soft); color: var(--m-ink); }
.mn-ty.is-menu { background: var(--m-pSoft); color: var(--m-pInk); }
.mn-ty.is-btn { background: var(--m-aSoft); color: var(--m-aOn, var(--m-ink)); }
/* 深色下黄色淡底很暗，文字改用强调色本身（同 .m-sticker--a） */
html.dark .mn-ty.is-btn { color: var(--m-a); }
.mn-ty--xs {
  height: 18px;
  padding: 0 6px;
  font-size: 0.625rem;
}

/* 名称列 */
.mn-name {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.mn-caret {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 32px;
  height: 32px;
  border: 0;
  border-radius: 10px;
  background: transparent;
  color: var(--el-text-color-primary);
  cursor: pointer;
}
.mn-caret:hover:not(:disabled) {
  background: var(--m-s2);
}
.mn-caret:disabled {
  cursor: default;
}
.mn-caret :deep(svg) {
  width: 16px;
  height: 16px;
  stroke-width: 2.4;
  transition: transform 0.15s;
}
.mn-caret.is-open :deep(svg) {
  transform: rotate(90deg);
}
.mn-caret--leaf {
  cursor: default;
}
.mn-caret--leaf > span {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--el-border-color);
}
.mn-caret--lg {
  width: 40px;
  height: 44px;
}
.mn-ico {
  width: 32px;
  height: 32px;
  border-radius: 10px;
}
.mn-ico :deep(svg) {
  width: 16px;
  height: 16px;
}
.mn-ico--off {
  background: var(--m-s2);
  color: var(--m-ink2);
}
.mn-ico--lg {
  width: 44px;
  height: 44px;
  border-radius: 14px;
}
.mn-ico--lg :deep(svg) {
  width: 22px;
  height: 22px;
}
.mn-name__text {
  font-weight: 700;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.mn-name__text.is-d0 {
  font-weight: 800;
}
.mn-name__text.is-btn {
  font-weight: 500;
}
.mn-name__text.is-off {
  color: var(--el-text-color-secondary);
}
.mn-note {
  flex: none;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.mn-kids {
  flex: none;
  padding: 1px 8px;
  border-radius: 999px;
  background: var(--m-s2);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 700;
}
.mn-mono {
  font-size: 0.8125rem;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}
.mn-mono--sm {
  font-size: 0.75rem;
}
.mn-mono.is-dim {
  color: var(--el-text-color-secondary);
}
.mn-order {
  font-weight: 700;
}
.mn-acts {
  display: inline-flex;
  justify-content: flex-end;
  gap: 2px;
}
.mn-acts :deep(.is-danger) {
  color: var(--m-bad);
}
.mn-acts :deep(.is-danger:hover) {
  background: var(--m-badSoft);
}
.mn-acts :deep(.is-stop) {
  color: var(--m-bad);
}
.mn-acts :deep(.is-start) {
  color: var(--m-ok);
}
.mn-acts :deep(.is-stop:hover) {
  background: var(--m-badSoft);
}
.mn-acts :deep(.is-start:hover) {
  background: var(--m-okSoft);
}
.mn-acts :deep(.is-locked) {
  color: var(--el-border-color);
  background: transparent;
  cursor: not-allowed;
}

/* 底部说明 */
.mn-tip {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin: 0;
  font-size: 0.8125rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.mn-tip :deep(svg) {
  width: 16px;
  height: 16px;
  flex: none;
  margin-top: 2px;
}

/* 手机树 */
.mn-tree {
  padding: 4px 8px 4px 4px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
}
.mn-tr {
  display: flex;
  align-items: center;
  gap: 4px;
  min-height: 56px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.mn-tr:last-child {
  border-bottom: 0;
}
.mn-tr__main {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  padding: 4px;
  border: 0;
  border-radius: 12px;
  background: transparent;
  text-align: left;
  font: inherit;
  cursor: pointer;
}
.mn-tr__main:active {
  background: var(--m-s2);
}
.mn-tr__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.mn-tr__title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.9375rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.mn-tr__title.is-off {
  color: var(--el-text-color-secondary);
}
.mn-tr__sub {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-size: 0.6875rem;
  color: var(--el-text-color-secondary);
}
.mn-tr__sub .m-mono {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mn-tr__label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}
.mn-tr__flag {
  flex: none;
  font-weight: 700;
  color: var(--m-warn);
}
.mn-legend.is-compact {
  margin-left: 0;
  gap: 6px;
}
.mn-legend.is-compact .mn-ty:not(:first-child) {
  margin-left: 0;
}

/* 手机：悬浮新增 */
.mn-fab {
  position: sticky;
  bottom: 0;
  z-index: 5;
  align-self: flex-end;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 60px;
  height: 60px;
  border: 0;
  border-radius: 20px;
  background: var(--m-p);
  color: var(--m-pOn);
  box-shadow: var(--app-shadow-brand);
  cursor: pointer;
}
.mn-fab :deep(svg) {
  width: 26px;
  height: 26px;
}
.mn-fab:active {
  transform: scale(0.97);
}

/* 调整排序 */
.mn-order-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.mn-order-form__row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.mn-order-form__row .app-input {
  flex: 1;
  min-width: 0;
  text-align: center;
  font-weight: 700;
}
.mn-order-form__hint {
  margin: 0;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.mn-skeleton {
  height: 420px;
  border-radius: var(--app-radius-lg);
  background: var(--el-fill-color-light);
}

/* 操作抽屉 */
.mn-sheet {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.mn-sheet__head {
  display: flex;
  align-items: center;
  gap: 12px;
}
.mn-sheet__title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.mn-sheet__name {
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.mn-sheet__sub {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mn-dl {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 12px;
  margin: 0;
  padding: 10px 12px;
  border-radius: 14px;
  background: var(--el-fill-color-light);
}
.mn-dl > div {
  min-width: 0;
}
.mn-dl__full {
  grid-column: 1 / -1;
}
.mn-dl dt {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.mn-dl dd {
  margin: 2px 0 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-primary);
  word-break: break-all;
}
.mn-sheet__acts {
  display: flex;
  flex-direction: column;
}
.mn-sheet__act {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 52px;
  padding: 0 8px;
  border: 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: transparent;
  font: inherit;
  font-size: 0.9375rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  text-align: left;
  cursor: pointer;
}
.mn-sheet__act:last-child {
  border-bottom: 0;
}
.mn-sheet__act :deep(svg) {
  width: 20px;
  height: 20px;
}
.mn-sheet__act.is-danger {
  color: var(--m-bad);
}
.mn-sheet__act:disabled {
  color: var(--el-text-color-placeholder);
  cursor: not-allowed;
}
.mn-sheet__cancel {
  min-height: 52px;
  border-radius: 999px;
  font-weight: 700;
}

@media (width < 768px) {
  .mn-page {
    /* 悬浮新增按钮贴在底部导航上方 100px 处；留出同样高度，滚到底时按钮不压住最后一行的操作 */
    padding-bottom: 100px;
  }
  .mn-seg {
    order: 4;
    width: 100%;
  }
  .mn-seg__btn {
    flex: 1;
    min-height: 44px;
  }
  .mn-tools {
    flex-wrap: nowrap;
    overflow-x: auto;
    scrollbar-width: none;
  }
  .mn-tools::-webkit-scrollbar {
    display: none;
  }
  .mn-tools :deep(button) {
    flex: none;
    min-height: 44px;
  }
  .mn-legend.is-compact {
    flex: none;
    flex-wrap: nowrap;
  }
}
</style>
