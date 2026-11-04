<!-- 用户（第二期改版）：组标题「用户与权限」+ 计数说明 + 邀请链接 / 新建用户 + AccessSubNav；状态 chips（带数量）
     + 角色分段（手机为 chips 行）+ 搜索；PC 表格（用户贴纸/多个角色/状态/最近登录+方式+IP/创建于/行内图标操作 + ⋯）、
     手机卡片（<dl> 信息格、底部角色标签 + 图标操作、右下悬浮新建）；底部深色批量条（分配角色/启用/停用/删除）。
     用户可以有多个角色（权限取并集）。内置超级管理员不可选、不可停用、不可删除、不可改角色。 -->
<template>
  <div class="acc-page">
    <PageHeader :title="t('access.nav.access')">
      <template #description>
        <span class="tnum">{{ t('access.user.summary', { total: summary.total }) }}</span>
        <span class="acc-dot">·</span>
        <span class="acc-ok tnum">{{ t('accessAdmin.user.activeCount', { count: summary.active }) }}</span>
        <span class="acc-dot">·</span>
        <span class="tnum">{{ t('accessAdmin.user.inactiveCount', { count: summary.inactive }) }}</span>
        <template v-if="!menuStore.isMobile">
          <span class="acc-dot">·</span>
          <span class="tnum">{{ t('accessAdmin.user.todayLogin', { count: summary.today }) }}</span>
        </template>
      </template>
      <template #actions>
        <template v-if="!menuStore.isMobile">
          <UButton v-if="canAdd" color="neutral" variant="outline" icon="i-lucide-link" @click="openInvite">
            {{ t('accessV2.invite.button') }}
          </UButton>
          <UButton v-if="canAdd" color="primary" icon="i-lucide-plus" @click="openCreate">
            {{ t('system.user.addUser') }}
          </UButton>
        </template>
        <AppIconButton
          v-else-if="canAdd"
          icon="HOutline:LinkIcon"
          :box="44"
          :label="t('accessV2.invite.button')"
          @click="openInvite"
        />
      </template>
    </PageHeader>

    <AccessSubNav :user-count="summary.total" :role-count="roles.length || null" :menu-count="null" />

    <ListToolbar
      v-model:status="queryForm.status"
      v-model:keyword="queryForm.keyword"
      :chips="statusChips"
      :chips-label="t('system.common.status')"
      :placeholder="t('accessAdmin.user.searchPlaceholder')"
      @search="search"
    >
      <div
        v-if="roles.length"
        class="acc-seg"
        :class="{ 'is-chips': menuStore.isMobile }"
        role="group"
        :aria-label="t('accessAdmin.user.roleFilter')"
      >
        <button
          v-for="opt in roleFilters"
          :key="opt.value"
          type="button"
          class="acc-seg__btn"
          :class="{ 'is-on': queryForm.roleId === opt.value }"
          :aria-pressed="queryForm.roleId === opt.value"
          @click="pickRole(opt.value)"
        >
          {{ opt.label }}
        </button>
      </div>
    </ListToolbar>

    <!-- PC：表格 -->
    <DataTable
      v-if="!menuStore.isMobile"
      v-model="selectedIds"
      class="acc-table"
      :columns="columns"
      :rows="(userList as unknown as Record<string, unknown>[])"
      row-key="userId"
      :selectable="canBatch"
      :row-selectable="(row) => !row.isBuiltin"
      :loading="loading"
      :empty-text="emptyText"
    >
      <template #cell-username="{ row }">
        <div class="acc-user">
          <AppAvatar v-if="avatarOf(row)" :src="avatarOf(row)" :size="40" :name="String(row.name || row.username)" />
          <span
            v-else
            class="m-sticker acc-user__av"
            :class="sticker(row).cls"
            :style="{ transform: sticker(row).rot }"
          >{{ initialOf(String(row.name || row.username)) }}</span>
          <div class="acc-user__text">
            <div class="acc-user__line">
              <span class="acc-user__name">{{ row.name || row.username }}</span>
              <span v-if="row.isBuiltin" class="acc-tag">
                <component :is="menuStore.iconComponents['HOutline:LockClosedIcon']" />{{ t('system.common.builtIn') }}
              </span>
              <span v-if="row.userId === myId" class="acc-me">（{{ t('access.user.me') }}）</span>
            </div>
            <span class="acc-user__sub">
              <span class="m-mono">{{ row.username }}</span>
              <template v-if="row.email"> · {{ row.email }}</template>
            </span>
          </div>
        </div>
      </template>
      <template #cell-roles="{ row }">
        <div class="acc-roles">
          <span v-for="r in rolesOf(row)" :key="r.roleId" class="acc-role" :class="`is-${roleTone(r.roleId)}`">
            {{ r.name }}
          </span>
          <span v-if="!rolesOf(row).length" class="acc-role is-none">{{ t('access.user.noRole') }}</span>
        </div>
      </template>
      <template #cell-status="{ row }">
        <StatusPill :variant="statusVariant(row.status)" :label="statusLabel(row.status)" />
      </template>
      <template #cell-lastLoginTime="{ row }">
        <div class="acc-login">
          <div class="acc-login__when">
            <span class="tnum">{{ formatLastLogin(t, row.lastLoginTime) }}</span>
            <span v-if="row.lastLoginTime && row.lastLoginMethod" class="acc-via">
              {{ loginMethodLabel(t, String(row.lastLoginMethod)) }}
            </span>
          </div>
          <div class="acc-login__ip m-mono">{{ row.lastLoginIp || '—' }}</div>
        </div>
      </template>
      <template #cell-createTime="{ row }">
        <span class="acc-dim tnum">{{ fmtDate(row.createTime) }}</span>
      </template>
      <template #cell-operation="{ row }">
        <div class="acc-acts">
          <template v-if="canEdit">
            <AppIconButton
              icon="HOutline:PencilSquareIcon"
              :label="`${t('system.common.edit')} ${nameOf(asUser(row))}`"
              :box="36"
              :size="18"
              @click="openEdit(asUser(row))"
            />
            <AppIconButton
              icon="HOutline:KeyIcon"
              :label="t('accessAdmin.user.resetPassword')"
              :box="36"
              :size="18"
              @click="openReset(asUser(row))"
            />
            <AppIconButton
              :icon="isActive(row) ? 'HOutline:NoSymbolIcon' : 'HOutline:CheckCircleIcon'"
              :label="toggleLabel(asUser(row))"
              :box="36"
              :size="18"
              :class="row.isBuiltin ? 'is-locked' : isActive(row) ? 'is-stop' : 'is-start'"
              :disabled="!!row.isBuiltin"
              @click="toggleStatus(asUser(row))"
            />
          </template>
          <ActionMenu
            v-if="rowMenuOf(asUser(row)).length"
            :items="rowMenuOf(asUser(row))"
            @select="(key) => onMenu(key, asUser(row))"
          />
        </div>
      </template>
    </DataTable>

    <!-- 手机：卡片 -->
    <template v-else>
      <div v-if="loading && !userList.length" class="acc-cards">
        <div v-for="i in 3" :key="i" class="acc-skeleton" />
      </div>
      <EmptyState
        v-else-if="!userList.length"
        icon="HOutline:UserGroupIcon"
        :title="emptyText"
        :description="hasFilter ? t('accessAdmin.user.emptyFilteredDesc') : t('system.user.emptyDesc')"
      >
        <template v-if="hasFilter" #action>
          <UButton color="neutral" variant="soft" @click="clearFilters">{{ t('accessAdmin.user.clearFilters') }}</UButton>
        </template>
      </EmptyState>
      <div v-else class="acc-cards">
        <article
          v-for="row in userList"
          :key="row.userId"
          class="acc-card"
          :class="{ 'is-selected': selectedSet.has(row.userId) }"
        >
          <header class="acc-card__head">
            <label v-if="canBatch && !row.isBuiltin" class="acc-card__check">
              <input
                type="checkbox"
                :checked="selectedSet.has(row.userId)"
                :aria-label="row.name || row.username"
                @change="toggleSelect(row.userId)"
              />
            </label>
            <AppAvatar v-if="avatarOf(row)" :src="avatarOf(row)" :size="46" :name="row.name || row.username" />
            <span
              v-else
              class="m-sticker acc-card__av"
              :class="sticker(row).cls"
              :style="{ transform: sticker(row).rot }"
            >{{ initialOf(row.name || row.username) }}</span>
            <div class="acc-card__title">
              <div class="acc-user__line">
                <span class="acc-card__name">{{ row.name || row.username }}</span>
                <span v-if="row.isBuiltin" class="acc-tag">
                  <component :is="menuStore.iconComponents['HOutline:LockClosedIcon']" />{{ t('system.common.builtIn') }}
                </span>
                <span v-if="row.userId === myId" class="acc-me">（{{ t('access.user.me') }}）</span>
              </div>
              <span class="acc-card__sub">{{ row.email || row.username }}</span>
            </div>
            <StatusPill :variant="statusVariant(row.status)" :label="statusLabel(row.status)" />
          </header>
          <dl class="acc-dl">
            <div>
              <dt>{{ t('accessAdmin.user.colLastLogin') }}</dt>
              <dd class="tnum">{{ formatLastLogin(t, row.lastLoginTime) }}</dd>
            </div>
            <div>
              <dt>{{ t('accessAdmin.user.loginMethod') }}</dt>
              <dd>{{ (row.lastLoginTime && loginMethodLabel(t, row.lastLoginMethod)) || '—' }}</dd>
            </div>
            <div>
              <dt>{{ t('accessAdmin.user.loginIp') }}</dt>
              <dd class="m-mono acc-dl__mono">{{ row.lastLoginIp || '—' }}</dd>
            </div>
            <div>
              <dt>{{ t('accessAdmin.user.colCreated') }}</dt>
              <dd class="tnum">{{ fmtDate(row.createTime) }}</dd>
            </div>
          </dl>
          <footer class="acc-card__foot">
            <div class="acc-card__roles">
              <span v-for="r in row.roles" :key="r.roleId" class="acc-role" :class="`is-${roleTone(r.roleId)}`">{{ r.name }}</span>
              <span v-if="!row.roles?.length" class="acc-role is-none">{{ t('access.user.noRole') }}</span>
            </div>
            <template v-if="canEdit">
              <AppIconButton icon="HOutline:PencilSquareIcon" :box="44" :label="`${t('system.common.edit')} ${nameOf(row)}`" @click="openEdit(row)" />
              <AppIconButton icon="HOutline:KeyIcon" :box="44" :label="t('accessAdmin.user.resetPassword')" @click="openReset(row)" />
              <AppIconButton
                :icon="isActive(row) ? 'HOutline:NoSymbolIcon' : 'HOutline:CheckCircleIcon'"
                :box="44"
                :label="toggleLabel(row)"
                :class="row.isBuiltin ? 'is-locked' : isActive(row) ? 'is-stop' : 'is-start'"
                :disabled="row.isBuiltin"
                @click="toggleStatus(row)"
              />
            </template>
            <ActionMenu v-if="rowMenuOf(row).length" :items="rowMenuOf(row)" @select="(key) => onMenu(key, row)" />
          </footer>
        </article>
      </div>
    </template>

    <div class="acc-foot">
      <span class="acc-foot__tip">
        <component :is="menuStore.iconComponents['HOutline:LockClosedIcon']" />
        {{ menuStore.isMobile ? t('accessV2.user.builtinTipShort') : t('accessAdmin.user.builtinTip') }}
      </span>
      <span class="acc-foot__spacer" />
      <span v-if="pagination.total" class="acc-foot__shown tnum">
        {{ t('accessAdmin.user.shown', { count: userList.length, total: pagination.total }) }}
      </span>
      <Pagination
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        @change="getUserList"
      />
    </div>

    <!-- 手机：悬浮新建 -->
    <div v-if="menuStore.isMobile && canAdd && !selectedIds.length" class="acc-fab">
      <UButton color="primary" size="xl" icon="i-lucide-plus" class="acc-fab__btn" @click="openCreate">
        {{ t('system.user.addUser') }}
      </UButton>
    </div>

    <!-- 浮动批量条 -->
    <Transition name="acc-pop">
      <div v-if="selectedIds.length" class="acc-batch" role="toolbar">
        <span class="acc-batch__count">{{ t('accessAdmin.user.selectedUsers', { count: selectedIds.length }) }}</span>
        <template v-if="canEdit">
          <button type="button" class="acc-batch__btn" @click="openAssign()">
            <component :is="menuStore.iconComponents['HOutline:ShieldCheckIcon']" />{{ t('accessAdmin.user.assignRole') }}
          </button>
          <button type="button" class="acc-batch__btn" @click="batchStatus(UserStatus.Active)">
            <component :is="menuStore.iconComponents['HOutline:CheckCircleIcon']" />{{ t('accessAdmin.user.enable') }}
          </button>
          <button type="button" class="acc-batch__btn" @click="batchStatus(UserStatus.InActive)">
            <component :is="menuStore.iconComponents['HOutline:NoSymbolIcon']" />{{ t('accessAdmin.user.disable') }}
          </button>
        </template>
        <button v-if="canDelete" type="button" class="acc-batch__btn is-danger" @click="confirmDelete(selectedIds)">
          <component :is="menuStore.iconComponents['HOutline:TrashIcon']" />{{ t('system.common.delete') }}
        </button>
        <button
          type="button"
          class="acc-batch__btn is-icon"
          :aria-label="t('system.user.clearSelection')"
          @click="selectedIds = []"
        >
          <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
        </button>
      </div>
    </Transition>

    <!-- 分配角色（多选，替换为所选集合） -->
    <FormDialog
      v-model="assignOpen"
      :title="t('accessAdmin.user.assignTitle', { count: assignTargets.length })"
      :width="480"
      :loading="assignLoading"
      @confirm="submitAssign"
    >
      <div class="acc-assign">
        <button
          v-for="r in roles"
          :key="r.roleId"
          type="button"
          role="checkbox"
          class="acc-assign__opt"
          :class="{ 'is-on': assignRoleIds.includes(r.roleId) }"
          :aria-checked="assignRoleIds.includes(r.roleId)"
          @click="toggleAssign(r.roleId)"
        >
          <span class="acc-assign__check" aria-hidden="true">
            <component :is="menuStore.iconComponents['HOutline:CheckIcon']" />
          </span>
          <span class="acc-role" :class="`is-${roleTone(r.roleId)}`">{{ r.name }}</span>
          <span class="acc-assign__desc">{{ r.description }}</span>
          <span class="acc-assign__n tnum">{{ t('accessAdmin.role.memberCount', { count: Number(r.userCount || 0) }) }}</span>
        </button>
        <p class="acc-assign__hint">{{ t('accessV2.user.assignMultiHint') }}</p>
      </div>
    </FormDialog>

    <!-- 邀请链接 -->
    <FormDialog v-model="inviteOpen" :title="t('accessV2.invite.title')" :width="520" :show-footer="false">
      <div class="acc-invite">
        <div v-if="inviteLoading" class="acc-invite__loading">
          <span class="acc-spinner" />{{ t('accessV2.invite.loading') }}
        </div>
        <template v-else>
          <div class="acc-invite__state" :class="registerEnabled ? 'is-on' : 'is-off'">
            <component :is="menuStore.iconComponents[registerEnabled ? 'HOutline:CheckCircleIcon' : 'HOutline:ExclamationTriangleIcon']" />
            <span>{{ registerEnabled ? t('accessV2.invite.enabled') : t('accessV2.invite.disabled') }}</span>
          </div>
          <div class="acc-invite__link">
            <span class="acc-invite__url m-mono">{{ inviteUrl }}</span>
            <UButton
              color="primary"
              :icon="inviteCopied ? 'i-lucide-check' : 'i-lucide-copy'"
              :disabled="!registerEnabled"
              @click="copyInvite"
            >
              {{ inviteCopied ? t('accessV2.invite.copied') : t('accessV2.invite.copy') }}
            </UButton>
          </div>
          <ul class="acc-invite__notes">
            <li>{{ t('accessV2.invite.noteRegister') }}</li>
            <li>{{ t('accessV2.invite.noteRole') }}</li>
            <li v-if="registerEmailRequired">{{ t('accessV2.invite.noteEmail') }}</li>
          </ul>
          <div class="acc-invite__acts">
            <UButton
              v-if="canSettings"
              color="neutral"
              variant="soft"
              icon="i-lucide-settings"
              :to="pageTabTo('settings', 'general')"
              @click="inviteOpen = false"
            >
              {{ t('accessV2.invite.settings') }}
            </UButton>
            <UButton color="neutral" variant="soft" @click="inviteOpen = false">{{ t('system.common.close') }}</UButton>
          </div>
        </template>
      </div>
    </FormDialog>

    <UserCreate ref="userCreateRef" @refresh="refresh" />
    <UserResetPassword ref="resetRef" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { batchUpdateUser, deleteUser, userPage } from '@/api/user'
import { rolePage } from '@/api/role'
import { getLoginConfig } from '@/api/login'
import { pageTabTo } from '@/config/pageGroups'
import { PERM } from '@/config/permission'
import { Dialog } from '@/utils/dialog'
import { resolveAvatarUrl } from '@/utils/assets'
import { useUserStore } from '@/stores/user'
import { UserStatus, type UserInfo, type UserRole } from '@/types/v1/user'
import type { RoleInfo } from '@/types/v1/system'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'
import UserCreate from '@/views/system/user/create.vue'
import UserResetPassword from '@/views/system/user/resetPassword.vue'
import AccessSubNav from '@/views/system/components/AccessSubNav.vue'
import {
  NO_PERMISSION_ROLE_ID,
  formatLastLogin,
  initialOf,
  loginMethodLabel,
  roleTone,
  stickerOf,
} from '@/views/system/components/accessTone'

defineOptions({ name: 'UserView' })

const menuStore = useMenuStore()
const userStore = useUserStore()
const route = useRoute()
const { t } = useI18n()
const userCreateRef = useTemplateRef<InstanceType<typeof UserCreate> | null>('userCreateRef')
const resetRef = useTemplateRef<InstanceType<typeof UserResetPassword> | null>('resetRef')

const loading = ref(false)
const userList = ref<UserInfo[]>([])
const roles = ref<RoleInfo[]>([])
const selectedIds = ref<string[]>([])
const selectedSet = computed(() => new Set(selectedIds.value))
const myId = computed(() => userStore.userInfo?.userId || '')

const queryRoleId = () => (typeof route.query.roleId === 'string' ? route.query.roleId : '')
const queryForm = ref<{ keyword: string; status: string; roleId: string }>({
  keyword: '',
  status: '',
  roleId: queryRoleId(),
})
const pagination = ref({ page: 1, pageSize: 10, total: 0 })
// chips 计数（随关键字/角色筛选变化，不受状态筛选影响）
const counts = ref({ active: 0, inactive: 0, today: 0 })
// 页头概况（无关键字/角色筛选时的全量统计）
const summary = ref({ total: 0, active: 0, inactive: 0, today: 0 })

const canAdd = computed(() => menuStore.hasButtonPermission(PERM.USER_ADD))
const canEdit = computed(() => menuStore.hasButtonPermission(PERM.USER_EDIT))
const canDelete = computed(() => menuStore.hasButtonPermission(PERM.USER_DELETE))
const canSettings = computed(() => menuStore.hasButtonPermission(PERM.SYSTEM_CONFIG_VIEW))
const canBatch = computed(() => canEdit.value || canDelete.value)
const hasFilter = computed(
  () => !!(queryForm.value.keyword || queryForm.value.status || queryForm.value.roleId),
)
const emptyText = computed(() =>
  hasFilter.value ? t('accessAdmin.user.emptyFiltered') : t('system.common.noData'),
)

const statusChips = computed<ToolbarChip<string>[]>(() => [
  { label: t('system.common.all'), value: '', count: counts.value.active + counts.value.inactive },
  { label: t('accessAdmin.user.chipActive'), value: UserStatus.Active, count: counts.value.active },
  { label: t('accessAdmin.user.chipInactive'), value: UserStatus.InActive, count: counts.value.inactive },
])
const roleFilters = computed(() => [
  { label: t('accessAdmin.user.allRoles'), value: '' },
  ...roles.value.map((r) => ({ label: r.name, value: r.roleId })),
])

const columns = computed<DataTableColumn[]>(() => [
  { key: 'username', title: t('accessAdmin.user.colUser'), minWidth: 260 },
  { key: 'roles', title: t('system.common.role'), minWidth: 170, ellipsis: false },
  { key: 'status', title: t('system.common.status'), width: 110 },
  { key: 'lastLoginTime', title: t('accessAdmin.user.colLastLogin'), minWidth: 170 },
  { key: 'createTime', title: t('accessAdmin.user.colCreated'), width: 120 },
  { key: 'operation', title: t('system.common.operation'), width: 190, align: 'right' },
])

// 「⋯」：分配角色 / 删除（内置账号没有）
const rowMenuOf = (u: UserInfo): ActionMenuItem[] => {
  if (u.isBuiltin) return []
  const items: ActionMenuItem[] = []
  if (canEdit.value) items.push({ key: 'assign', label: t('accessAdmin.user.assignRole'), icon: 'HOutline:ShieldCheckIcon' })
  if (canDelete.value) items.push({ key: 'delete', label: t('system.common.delete'), icon: 'HOutline:TrashIcon', danger: true })
  return items
}

const asUser = (row: Record<string, unknown>) => row as unknown as UserInfo
const rolesOf = (row: Record<string, unknown>) => (row.roles || []) as UserRole[]
const avatarOf = (row: { avatar?: unknown }) => (row.avatar ? resolveAvatarUrl(String(row.avatar)) : '')
const indexMap = computed(() => new Map(userList.value.map((u, i) => [u.userId, i])))
const sticker = (row: { userId?: unknown }) =>
  stickerOf(String(row.userId || ''), indexMap.value.get(String(row.userId || '')))
const isActive = (row: { status?: unknown }) => row.status === UserStatus.Active
const statusVariant = (s: unknown) => (s === UserStatus.Active ? 'success' : 'neutral')
const statusLabel = (s: unknown) =>
  s === UserStatus.Active ? t('accessAdmin.user.chipActive') : t('accessAdmin.user.chipInactive')
const fmtDate = (v: unknown) => (v ? dayjs(v as string).format('YYYY-MM-DD') : '—')
const nameOf = (u: UserInfo) => u.name || u.username
const toggleLabel = (u: UserInfo) => {
  if (u.isBuiltin) return t('accessAdmin.user.builtinLocked')
  return `${isActive(u) ? t('accessAdmin.user.disable') : t('accessAdmin.user.enable')} ${nameOf(u)}`
}

const toggleSelect = (id: string) => {
  selectedIds.value = selectedSet.value.has(id)
    ? selectedIds.value.filter((x) => x !== id)
    : [...selectedIds.value, id]
}

const getUserList = async () => {
  loading.value = true
  try {
    const q = queryForm.value
    const { data: res } = await userPage({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      username: q.keyword.trim() || undefined,
      status: (q.status || undefined) as UserStatus | undefined,
      roleId: q.roleId || undefined,
    })
    userList.value = res?.users || []
    pagination.value.total = Number(res?.total || 0)
    pagination.value.page = Number(res?.page || pagination.value.page)
    pagination.value.pageSize = Number(res?.pageSize || pagination.value.pageSize)
    const active = Number(res?.activeCount || 0)
    const inactive = Number(res?.inactiveCount || 0)
    const today = Number(res?.todayLoginCount || 0)
    counts.value = { active, inactive, today }
    if (!q.keyword.trim() && !q.roleId) summary.value = { total: active + inactive, active, inactive, today }
    // 当前页上已不存在的选中项剔除
    const ids = new Set(userList.value.map((u) => u.userId))
    selectedIds.value = selectedIds.value.filter((id) => ids.has(id))
  } finally {
    loading.value = false
  }
}

const loadSummary = async () => {
  const { data: res } = await userPage({ page: 1, pageSize: 1 })
  const active = Number(res?.activeCount || 0)
  const inactive = Number(res?.inactiveCount || 0)
  summary.value = { total: active + inactive, active, inactive, today: Number(res?.todayLoginCount || 0) }
}

const getRoles = async () => {
  try {
    const { data: res } = await rolePage({ page: 1, pageSize: 500 })
    roles.value = res?.roles || []
  } catch {
    roles.value = []
  }
}

const search = () => {
  pagination.value.page = 1
  getUserList()
}
const pickRole = (roleId: string) => {
  queryForm.value.roleId = roleId
  search()
}
const clearFilters = () => {
  queryForm.value = { keyword: '', status: '', roleId: '' }
  search()
}

// 角色页「管理成员」跳过来时带 ?roleId=（同一页面内切换标签不会重新挂载）
watch(
  () => route.query.roleId,
  () => {
    const next = queryRoleId()
    if (next === queryForm.value.roleId) return
    queryForm.value.roleId = next
    search()
    if (next) loadSummary()
  },
)

const openCreate = () => userCreateRef.value?.showDialog()
const openEdit = (u: UserInfo) => userCreateRef.value?.showDialog({ userId: u.userId, isBuiltin: u.isBuiltin })
const openReset = (u: UserInfo) => resetRef.value?.showDialog(u)

const onMenu = (key: string, u: UserInfo) => {
  if (key === 'delete') confirmDelete([u.userId])
  else if (key === 'assign') openAssign([u.userId], (u.roles || []).map((r) => r.roleId))
}

const toggleStatus = (u: UserInfo) => {
  if (u.isBuiltin) return
  const name = nameOf(u)
  if (isActive(u)) {
    Dialog.info({
      showCancelButton: true,
      content: t('accessAdmin.user.confirmDisable', { name }),
      confirmText: t('accessAdmin.user.disable'),
      cancelText: t('system.common.cancel'),
      onConfirm: async () => {
        await batchUpdateUser({ userIds: [u.userId], status: UserStatus.InActive, roles: undefined })
        feedback.success(t('accessAdmin.user.disabledOne', { name }))
        afterChange()
      },
    })
    return
  }
  batchUpdateUser({ userIds: [u.userId], status: UserStatus.Active, roles: undefined }).then(() => {
    feedback.success(t('accessAdmin.user.enabledOne', { name }))
    afterChange()
  })
}

const reportBatch = (res?: { updated?: unknown; skipped?: unknown }) => {
  const updated = Number(res?.updated || 0)
  const skipped = Number(res?.skipped || 0)
  feedback.success(
    t('accessAdmin.user.updatedCount', { count: updated }) +
      (skipped ? t('accessAdmin.user.skippedBuiltin', { count: skipped }) : ''),
  )
}

const batchStatus = (status: UserStatus) => {
  const ids = [...selectedIds.value]
  if (!ids.length) return
  const run = async () => {
    const { data } = await batchUpdateUser({ userIds: ids, status, roles: undefined })
    reportBatch(data)
    selectedIds.value = []
    afterChange()
  }
  if (status === UserStatus.InActive) {
    Dialog.info({
      showCancelButton: true,
      content: t('accessAdmin.user.confirmBatchDisable', { count: ids.length }),
      confirmText: t('accessAdmin.user.disable'),
      cancelText: t('system.common.cancel'),
      onConfirm: run,
    })
  } else run()
}

// 分配角色：批量条（所选用户）或行内「⋯」（单个用户，预选其当前角色）；提交即替换为所选角色集合
const assignOpen = ref(false)
const assignLoading = ref(false)
const assignRoleIds = ref<string[]>([])
const assignTargets = ref<string[]>([])
const openAssign = (targets?: string[], preset?: string[]) => {
  if (!roles.value.length) getRoles()
  assignTargets.value = targets?.length ? targets : [...selectedIds.value]
  assignRoleIds.value = preset ? [...preset] : []
  assignOpen.value = true
}
// 「无权限」与其它角色互斥
const toggleAssign = (roleId: string) => {
  const cur = assignRoleIds.value
  if (cur.includes(roleId)) {
    assignRoleIds.value = cur.filter((id) => id !== roleId)
    return
  }
  assignRoleIds.value =
    roleId === NO_PERMISSION_ROLE_ID ? [roleId] : [...cur.filter((id) => id !== NO_PERMISSION_ROLE_ID), roleId]
}
const submitAssign = async () => {
  if (!assignRoleIds.value.length) {
    feedback.warning(t('accessAdmin.user.pickRole'))
    return
  }
  assignLoading.value = true
  try {
    const targets = [...assignTargets.value]
    const { data } = await batchUpdateUser({ userIds: targets, roles: { ids: [...assignRoleIds.value] } })
    reportBatch(data)
    assignOpen.value = false
    selectedIds.value = selectedIds.value.filter((id) => !targets.includes(id))
    afterChange()
    getRoles()
  } finally {
    assignLoading.value = false
  }
}

const confirmDelete = (ids: string[]) => {
  const deletable = ids.filter((id) => !userList.value.find((u) => u.userId === id)?.isBuiltin)
  if (!deletable.length) return
  Dialog.info({
    showCancelButton: true,
    content: t('system.user.confirmDeleteSelected'),
    confirmText: t('system.common.delete'),
    cancelText: t('system.common.cancel'),
    onConfirm: async () => {
      await deleteUser({ userIds: deletable })
      selectedIds.value = selectedIds.value.filter((id) => !deletable.includes(id))
      feedback.success(t('system.common.deleteSuccess'))
      afterChange()
      getRoles()
    },
  })
}

// 邀请链接：注册入口在登录页（「立即注册」），这里显示是否开放注册与新账号的默认角色
const inviteOpen = ref(false)
const inviteLoading = ref(false)
const inviteCopied = ref(false)
const registerEnabled = ref(false)
const registerEmailRequired = ref(false)
const inviteUrl = computed(() => `${window.location.origin}/login`)
const openInvite = async () => {
  inviteOpen.value = true
  inviteCopied.value = false
  inviteLoading.value = true
  try {
    const { data } = await getLoginConfig()
    registerEnabled.value = !!data?.config?.registerEnabled
    registerEmailRequired.value = !!data?.config?.registerEmailVerificationRequired
  } catch {
    registerEnabled.value = false
  } finally {
    inviteLoading.value = false
  }
}
const copyInvite = async () => {
  await navigator.clipboard.writeText(inviteUrl.value)
  inviteCopied.value = true
  feedback.success(t('accessV2.invite.copied'))
}

const afterChange = () => {
  getUserList()
  if (queryForm.value.keyword.trim() || queryForm.value.roleId) loadSummary()
}

const refresh = (type: 'create' | 'update') => {
  if (type === 'create') pagination.value.page = 1
  afterChange()
  getRoles()
}

onMounted(() => {
  getUserList()
  getRoles()
  if (queryForm.value.roleId) loadSummary()
})
</script>

<style scoped lang="scss">
.acc-page :deep(.page-header__actions .app-icon-btn) {
  padding: 0;
  border-radius: 14px;
}
.acc-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.acc-dot {
  margin: 0 6px;
  opacity: 0.6;
}
.acc-ok {
  color: var(--m-ok);
  font-weight: 700;
}
.acc-dim {
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
}

/* 角色分段（横向滚动、不显示滚动条） */
.acc-seg {
  display: inline-flex;
  gap: 2px;
  max-width: 100%;
  padding: 4px;
  border-radius: 999px;
  background: var(--m-s2);
  overflow-x: auto;
  scrollbar-width: none;
}
.acc-seg::-webkit-scrollbar {
  display: none;
}
.acc-seg__btn {
  flex: none;
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
.acc-seg__btn:hover {
  color: var(--el-text-color-primary);
}
.acc-seg__btn.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}

/* 手机：角色筛选改为 chips 行 */
.acc-seg.is-chips {
  gap: 8px;
  padding: 0;
  border-radius: 0;
  background: transparent;
}
.acc-seg.is-chips .acc-seg__btn {
  min-height: 44px;
  padding: 0 16px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  font-size: 0.875rem;
}
.acc-seg.is-chips .acc-seg__btn.is-on {
  background: var(--m-ink);
  border-color: var(--m-ink);
  color: var(--m-bg);
  box-shadow: none;
}

/* 用户单元格 */
.acc-user {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.acc-user__av {
  width: 40px;
  height: 40px;
  border-radius: 13px;
  font-size: 1rem;
}
.acc-user__text {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}
.acc-user__line {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.acc-user__name {
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.acc-user__sub {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.acc-user__sub .m-mono {
  font-size: 0.75rem;
}
.acc-tag {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  flex: none;
  height: 20px;
  padding: 0 7px;
  border-radius: 6px;
  background: var(--m-ink);
  color: var(--m-bg);
  font-size: 0.6875rem;
  font-weight: 700;
}
.acc-tag :deep(svg) {
  width: 11px;
  height: 11px;
}
.acc-me {
  flex: none;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--m-pInk);
}

/* 角色标签 */
.acc-roles {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  min-width: 0;
}
.acc-role {
  display: inline-flex;
  align-items: center;
  height: 24px;
  max-width: 100%;
  padding: 0 9px;
  border-radius: 8px;
  font-size: 0.75rem;
  font-weight: 700;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.acc-role.is-a { background: var(--m-aSoft); color: var(--m-aOn, var(--m-ink)); }
html.dark .acc-role.is-a { color: var(--m-a); }
.acc-role.is-p { background: var(--m-pSoft); color: var(--m-pInk); }
.acc-role.is-a2 { background: var(--m-a2Soft); color: var(--m-ink); }
.acc-role.is-ok { background: var(--m-okSoft); color: var(--m-ok); }
.acc-role.is-warn { background: var(--m-warnSoft); color: var(--m-warn); }
.acc-role.is-none { background: var(--m-s2); color: var(--m-ink2); }

/* 最近登录 */
.acc-login__when {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}
.acc-via {
  display: inline-flex;
  align-items: center;
  height: 18px;
  padding: 0 6px;
  border-radius: 6px;
  background: var(--m-s2);
  color: var(--m-ink2);
  font-size: 0.6875rem;
  font-weight: 700;
}
.acc-login__ip {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}

/* 行内操作 */
.acc-acts {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 2px;
}
.acc-acts :deep(.is-stop),
.acc-card__foot :deep(.is-stop) { color: var(--m-bad); }
.acc-acts :deep(.is-start),
.acc-card__foot :deep(.is-start) { color: var(--m-ok); }
.acc-acts :deep(.is-stop:hover),
.acc-card__foot :deep(.is-stop:hover) { background: var(--m-badSoft); }
.acc-acts :deep(.is-start:hover),
.acc-card__foot :deep(.is-start:hover) { background: var(--m-okSoft); }
.acc-acts :deep(.is-locked),
.acc-card__foot :deep(.is-locked) {
  color: var(--el-border-color);
  cursor: not-allowed;
  background: transparent;
}

/* 页脚 */
.acc-foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}
.acc-foot__tip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.acc-foot__tip :deep(svg) {
  width: 16px;
  height: 16px;
  flex: none;
}
.acc-foot__spacer {
  flex: 1;
}
.acc-foot__shown {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

/* 手机：悬浮新建 */
.acc-fab {
  position: sticky;
  bottom: 0;
  z-index: 5;
  display: flex;
  justify-content: flex-end;
  pointer-events: none;
}
.acc-fab__btn {
  pointer-events: auto;
  min-height: 56px;
  padding: 0 22px;
  border-radius: var(--app-radius-pill);
  box-shadow: var(--app-shadow-brand);
}

/* 手机卡片 */
.acc-cards {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.acc-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px 12px 6px 16px;
  border-radius: var(--app-radius-xl, var(--app-radius-lg));
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.acc-card.is-selected {
  border-color: var(--m-p);
  box-shadow: 0 0 0 1px var(--m-p);
}
.acc-card__head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-right: 4px;
}
.acc-card__check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 44px;
  margin: -8px -6px -8px -6px;
  flex: none;
}
.acc-card__check input {
  width: 20px;
  height: 20px;
  accent-color: var(--m-p);
}
.acc-card__av {
  width: 46px;
  height: 46px;
  border-radius: 15px;
  font-size: 1.125rem;
}
.acc-card__title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.acc-card__name {
  font-size: 1rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.acc-card__sub {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.acc-dl {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 12px;
  margin: 0 4px 0 0;
  padding: 10px 12px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.acc-dl > div {
  min-width: 0;
}
.acc-dl dt {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.acc-dl dd {
  margin: 2px 0 0;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.acc-dl__mono {
  font-size: 0.75rem !important;
  font-weight: 500 !important;
}
.acc-card__foot {
  display: flex;
  align-items: center;
  gap: 2px;
  padding-top: 6px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.acc-card__foot :deep(.app-icon-btn) {
  width: 44px !important;
  height: 44px !important;
}
.acc-card__roles {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}
.acc-skeleton {
  height: 210px;
  border-radius: var(--app-radius-lg);
  background: var(--el-fill-color-light);
  animation: acc-pulse 1.4s ease-in-out infinite;
}
@keyframes acc-pulse {
  50% {
    opacity: 0.55;
  }
}

/* 浮动批量条 */
.acc-batch {
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
.acc-batch::-webkit-scrollbar {
  display: none;
}
.acc-batch__count {
  margin-right: 10px;
  font-size: 0.875rem;
  font-weight: 800;
  white-space: nowrap;
}
.acc-batch__btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: none;
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
.acc-batch__btn:hover {
  background: color-mix(in srgb, var(--m-bg) 14%, transparent);
}
.acc-batch__btn.is-danger {
  color: var(--m-a);
}
.acc-batch__btn.is-icon {
  width: 40px;
  padding: 0;
  justify-content: center;
}
.acc-batch__btn :deep(svg) {
  width: 16px;
  height: 16px;
}
@media (width < 992px) {
  .acc-batch {
    bottom: calc(78px + env(safe-area-inset-bottom));
    padding-left: 14px;
  }
}
.acc-pop-enter-active,
.acc-pop-leave-active {
  transition: opacity 0.18s, transform 0.18s;
}
.acc-pop-enter-from,
.acc-pop-leave-to {
  opacity: 0;
  transform: translateY(12px);
}

/* 分配角色 */
.acc-assign {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.acc-assign__opt {
  display: grid;
  grid-template-columns: auto auto 1fr auto;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  padding: 8px 14px;
  border: 1.5px solid var(--el-border-color-light);
  border-radius: var(--app-radius);
  background: var(--el-bg-color);
  text-align: left;
  font: inherit;
  cursor: pointer;
}
.acc-assign__opt:hover {
  border-color: var(--m-p);
}
.acc-assign__opt.is-on {
  border-color: var(--m-p);
  background: var(--m-pSoft);
}
.acc-assign__check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 6px;
  border: 1.5px solid var(--el-border-color);
  color: transparent;
}
.acc-assign__check :deep(svg) {
  width: 13px;
  height: 13px;
  stroke-width: 3;
}
.acc-assign__opt.is-on .acc-assign__check {
  border-color: var(--m-p);
  background: var(--m-p);
  color: var(--m-pOn);
}
.acc-assign__desc {
  min-width: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.acc-assign__n {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.acc-assign__hint {
  margin: 4px 0 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}

/* 邀请链接 */
.acc-invite {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.acc-invite__loading {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 0;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.acc-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--el-border-color);
  border-top-color: var(--m-p);
  border-radius: 999px;
  animation: acc-spin 0.8s linear infinite;
}
@keyframes acc-spin {
  to {
    transform: rotate(360deg);
  }
}
.acc-invite__state {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border-radius: var(--app-radius);
  font-size: 0.875rem;
  font-weight: 700;
}
.acc-invite__state :deep(svg) {
  width: 18px;
  height: 18px;
  flex: none;
}
.acc-invite__state.is-on {
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.acc-invite__state.is-off {
  background: var(--m-warnSoft);
  color: var(--m-warn);
}
.acc-invite__link {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.acc-invite__url {
  flex: 1 1 260px;
  min-width: 0;
  min-height: 48px;
  display: flex;
  align-items: center;
  padding: 0 14px;
  border-radius: var(--app-radius);
  border: 1.5px dashed var(--el-border-color-light);
  background: var(--el-fill-color-light);
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow-x: auto;
  white-space: nowrap;
  scrollbar-width: none;
}
.acc-invite__url::-webkit-scrollbar {
  display: none;
}
.acc-invite__link :deep(button) {
  min-height: 48px;
  border-radius: 999px;
  padding: 0 20px;
  font-weight: 700;
}
.acc-invite__notes {
  margin: 0;
  padding-left: 18px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 0.8125rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.acc-invite__acts {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}
.acc-invite__acts :deep(button),
.acc-invite__acts :deep(a) {
  min-height: 44px;
  border-radius: 999px;
  font-weight: 700;
}

@media (width < 768px) {
  .acc-foot__tip {
    font-size: 0.75rem;
  }
  .acc-seg {
    order: 4;
    width: 100%;
  }
  .acc-foot__shown {
    display: none;
  }
}
</style>
