<!-- 角色（第二期改版）：组标题 + 计数说明 + AccessSubNav；PC 左侧角色列表（成员数）+ 右侧角色详情
     （权限进度、成员头像（RoleInfo.members 预览）、从其它角色复制/编辑信息/删除）+ 模块化权限编辑 + 底部深色保存条；
     手机角色卡片（权限进度 + 「N 项 · M 人」+ 右箭头），点开底部抽屉编辑权限（底部保存状态 + 放弃/保存）。
     用户可以有多个角色，成员数按角色各自统计。内置角色只读。 -->
<template>
  <div class="role-page">
    <PageHeader :title="t('access.nav.access')">
      <template #description>
        <template v-if="!menuStore.isMobile">
          <span class="tnum">{{ t('access.role.summary', { total: roles.length, builtin: builtinCount }) }}</span>
          <span class="role-dot">·</span>{{ t('accessAdmin.role.summaryTail') }}
        </template>
        <span v-else class="tnum">{{ t('accessV2.role.mobileSummary', { total: roles.length }) }}</span>
      </template>
      <template #actions>
        <UButton v-if="canAdd && !menuStore.isMobile" color="primary" icon="i-lucide-plus" @click="openCreate">
          {{ t('system.role.addRole') }}
        </UButton>
        <AppIconButton
          v-else-if="canAdd"
          icon="HOutline:PlusIcon"
          :box="44"
          :label="t('system.role.addRole')"
          @click="openCreate"
        />
      </template>
    </PageHeader>

    <AccessSubNav :role-count="roles.length || null" :menu-count="null" />

    <div v-if="loading && !roles.length" class="role-layout">
      <div class="role-skeleton role-skeleton--list" />
      <div class="role-skeleton role-skeleton--detail" />
    </div>

    <EmptyState
      v-else-if="!roles.length"
      icon="HOutline:ShieldCheckIcon"
      :title="t('system.common.noData')"
      :description="t('system.role.emptyDesc')"
    />

    <div v-else class="role-layout">
      <!-- 角色列表 -->
      <section class="role-list" :aria-label="t('access.role.list')">
        <div class="role-list__head">
          <h2>{{ t('access.role.list') }}</h2>
          <span class="role-list__n tnum">{{ t('accessAdmin.role.count', { count: filteredRoles.length }) }}</span>
          <ActionMenu v-if="canDelete && deletableRoles.length" :items="listMenu" @select="onListMenu" />
        </div>
        <label class="role-search">
          <component :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']" />
          <input v-model="keyword" type="search" :placeholder="t('access.role.searchPlaceholder')" :aria-label="t('access.role.searchPlaceholder')" />
        </label>
        <div class="role-filter" role="tablist" :aria-label="t('system.common.status')">
          <button
            v-for="opt in statusOptions"
            :key="opt.value"
            type="button"
            role="tab"
            class="role-filter__btn"
            :class="{ 'is-on': statusFilter === opt.value }"
            :aria-selected="statusFilter === opt.value"
            @click="statusFilter = opt.value"
          >
            {{ opt.label }}
          </button>
        </div>

        <button
          v-for="(r, ri) in filteredRoles"
          :key="r.roleId"
          type="button"
          class="role-item"
          :class="{ 'is-on': !menuStore.isMobile && r.roleId === currentId }"
          :aria-pressed="r.roleId === currentId"
          @click="pickRole(r.roleId)"
        >
          <span class="m-sticker role-item__av" :class="stickerOf(r.roleId, ri).cls" :style="{ transform: stickerOf(r.roleId, ri).rot }">
            {{ initialOf(r.name) }}
          </span>
          <span class="role-item__text">
            <span class="role-item__line">
              <span class="role-item__name">{{ r.name }}</span>
              <span v-if="r.isBuiltin" class="role-tag">{{ t('system.common.builtIn') }}</span>
              <span v-if="r.status !== RoleStatus.RoleStatus_Active" class="role-tag role-tag--off">{{ t('accessAdmin.role.inactive') }}</span>
              <template v-if="r.roleId === currentId && dirtyCount">
                <span v-if="menuStore.isMobile" class="role-item__unsaved">● {{ t('accessV2.role.unsaved') }}</span>
                <span v-else class="role-item__dirty" :aria-label="t('accessV2.role.unsaved')" />
              </template>
            </span>
            <span class="role-item__desc">{{ r.description || '—' }}</span>
            <span v-if="menuStore.isMobile" class="role-item__bar">
              <span class="role-bar"><span :style="{ width: pct(grantedOf(r.menuIds)) }" /></span>
              <span class="tnum">{{ t('accessV2.role.itemsMembers', { items: grantedOf(r.menuIds), members: Number(r.userCount || 0) }) }}</span>
            </span>
          </span>
          <component
            :is="menuStore.iconComponents['HOutline:ChevronRightIcon']"
            v-if="menuStore.isMobile"
            class="role-item__chev"
            aria-hidden="true"
          />
          <span v-else class="role-item__members">
            <span class="role-item__count tnum">{{ Number(r.userCount || 0) }}</span>
            <span class="role-item__label">{{ t('accessAdmin.role.members') }}</span>
          </span>
        </button>
        <p v-if="!filteredRoles.length" class="role-list__empty">{{ t('system.common.noData') }}</p>

        <UButton
          v-if="!menuStore.isMobile && canAdd"
          class="role-list__add"
          color="neutral"
          variant="soft"
          icon="i-lucide-plus"
          block
          @click="openCreate"
        >
          {{ t('system.role.addRole') }}
        </UButton>
      </section>

      <!-- PC：详情 + 权限 -->
      <div v-if="!menuStore.isMobile" class="role-detail">
        <template v-if="current">
          <section class="role-hero">
            <div class="role-hero__top">
              <span class="m-sticker role-hero__av" :class="currentSticker">{{ initialOf(current.name) }}</span>
              <div class="role-hero__text">
                <div class="role-hero__line">
                  <h2>{{ current.name }}</h2>
                  <span v-if="current.isBuiltin" class="role-tag">
                    <component :is="menuStore.iconComponents['HOutline:LockClosedIcon']" />{{ t('system.common.builtIn') }}
                  </span>
                  <StatusPill v-if="current.status !== RoleStatus.RoleStatus_Active" variant="neutral" :label="t('accessAdmin.role.inactive')" />
                </div>
                <p>{{ current.description || '—' }}</p>
              </div>
              <div v-if="!locked && canEdit" class="role-hero__acts">
                <UButton color="neutral" variant="outline" size="sm" icon="i-lucide-copy" @click="openCopy">
                  {{ t('accessAdmin.role.copyFrom') }}
                </UButton>
                <UButton color="neutral" variant="outline" size="sm" icon="i-lucide-pencil" @click="openEditInfo">
                  {{ t('accessAdmin.role.editInfo') }}
                </UButton>
                <ActionMenu v-if="canDelete" :items="roleMenu" @select="onRoleMenu" />
              </div>
            </div>

            <div class="role-hero__stats">
              <div class="role-hero__progress">
                <div class="role-hero__granted">
                  <span>{{ t('access.role.granted') }}</span>
                  <strong class="m-num">{{ grantedOf(draft) }}</strong>
                  <em class="tnum">/ {{ leafTotal }}</em>
                </div>
                <span class="role-bar role-bar--lg"><span :style="{ width: pct(grantedOf(draft)) }" /></span>
              </div>
              <div class="role-members">
                <div v-if="members.length" class="role-members__avs">
                  <template v-for="(u, ui) in members" :key="u.userId">
                    <AppAvatar
                      v-if="u.avatar"
                      class="role-members__av is-img"
                      :src="resolveAvatarUrl(u.avatar)"
                      :size="32"
                      :name="u.name || u.username"
                    />
                    <span
                      v-else
                      class="m-sticker role-members__av"
                      :class="stickerOf(u.userId, ui + 1).cls"
                      :title="u.name || u.username"
                    >{{ initialOf(u.name || u.username) }}</span>
                  </template>
                </div>
                <span class="role-members__text">{{ memberText }}</span>
                <RouterLink
                  v-if="canViewUsers"
                  :to="pageTabTo('access', 'user', { roleId: current.roleId || '' })"
                  class="role-members__link"
                >
                  {{ t('accessAdmin.role.manageMembers') }}
                </RouterLink>
              </div>
            </div>

            <div v-if="locked" class="role-lock">
              <component :is="menuStore.iconComponents['HOutline:LockClosedIcon']" />
              <span>{{ lockText }}</span>
            </div>
          </section>

          <EmptyState
            v-if="!draft.length && locked"
            icon="HOutline:ShieldCheckIcon"
            :title="t('access.role.noPermission')"
          />
          <PermissionEditor v-else v-model="draft" :menus="menus" :locked="locked || !canEdit" />

          <div class="role-save" role="status">
            <span class="role-save__dot" :class="saveTone" />
            <span class="role-save__text">{{ saveText }}</span>
            <UButton
              v-if="!locked && canEdit"
              color="neutral"
              variant="ghost"
              class="role-save__ghost"
              :disabled="!dirtyCount"
              @click="discard"
            >
              {{ t('accessAdmin.role.discard') }}
            </UButton>
            <UButton
              v-if="!locked && canEdit"
              color="primary"
              icon="i-lucide-check"
              :loading="saving"
              :disabled="!dirtyCount"
              @click="save"
            >
              {{ t('accessAdmin.role.save') }}
            </UButton>
          </div>
        </template>
        <EmptyState v-else icon="HOutline:ShieldCheckIcon" :title="t('access.role.pickHint')" />
      </div>
    </div>



    <!-- 手机：权限抽屉 -->
    <FormDialog
      v-if="menuStore.isMobile"
      v-model="sheetOpen"
      :title="current?.name || ''"
      :width="720"
      @close="onSheetClose"
    >
      <div v-if="current" class="role-sheet">
        <div class="role-sheet__head">
          <span class="m-sticker role-sheet__av" :class="currentSticker">{{ initialOf(current.name) }}</span>
          <span class="role-sheet__title">
            <span class="role-sheet__name">
              {{ current.name }}
              <span v-if="current.isBuiltin" class="role-tag">{{ t('system.common.builtIn') }}</span>
            </span>
            <span class="role-sheet__stats tnum">
              {{ t('accessV2.role.sheetStats', { on: grantedOf(draft), total: leafTotal, members: memberCount }) }}
            </span>
          </span>
        </div>
        <p class="role-sheet__desc">{{ current.description || '—' }}</p>
        <div v-if="members.length" class="role-members">
          <div class="role-members__avs">
            <span
              v-for="(u, ui) in members"
              :key="u.userId"
              class="m-sticker role-members__av"
              :class="stickerOf(u.userId, ui + 1).cls"
              :title="u.name || u.username"
            >{{ initialOf(u.name || u.username) }}</span>
          </div>
          <span class="role-members__text">{{ memberText }}</span>
        </div>
        <div v-if="!locked && canEdit" class="role-sheet__acts">
          <UButton color="neutral" variant="soft" size="sm" icon="i-lucide-copy" @click="openCopy">
            {{ t('accessAdmin.role.copyFrom') }}
          </UButton>
          <UButton color="neutral" variant="soft" size="sm" icon="i-lucide-pencil" @click="openEditInfo">
            {{ t('accessAdmin.role.editInfo') }}
          </UButton>
          <ActionMenu v-if="canDelete" :items="roleMenu" @select="onRoleMenu" />
        </div>
        <div v-if="locked" class="role-lock">
          <component :is="menuStore.iconComponents['HOutline:LockClosedIcon']" />
          <span>{{ lockText }}</span>
        </div>
        <EmptyState v-if="!draft.length && locked" icon="HOutline:ShieldCheckIcon" :title="t('access.role.noPermission')" />
        <PermissionEditor v-else v-model="draft" :menus="menus" :locked="locked || !canEdit" />
      </div>
      <template #footer="{ close }">
        <span class="role-sheet__save">
          <span class="role-save__dot" :class="saveTone" />
          <span>{{ saveTextShort }}</span>
        </span>
        <template v-if="!locked && canEdit">
          <UButton color="neutral" variant="soft" :disabled="!dirtyCount" @click="discard">
            {{ t('accessAdmin.role.discardShort') }}
          </UButton>
          <UButton color="primary" :loading="saving" :disabled="!dirtyCount" @click="save">
            {{ t('accessAdmin.role.saveShort') }}
          </UButton>
        </template>
        <UButton v-else color="neutral" variant="soft" @click="close">{{ t('system.common.close') }}</UButton>
      </template>
    </FormDialog>

    <!-- 从其它角色复制 -->
    <FormDialog
      v-model="copyOpen"
      :title="t('accessAdmin.role.copyTitle')"
      :width="460"
      @confirm="applyCopy"
    >
      <p class="role-copy__hint">{{ t('accessAdmin.role.copyHint', { name: current?.name || '' }) }}</p>
      <div class="role-copy">
        <button
          v-for="r in copySources"
          :key="r.roleId"
          type="button"
          class="role-copy__opt"
          :class="{ 'is-on': copyFromId === r.roleId }"
          :aria-pressed="copyFromId === r.roleId"
          @click="copyFromId = r.roleId"
        >
          <strong>{{ r.name }}</strong>
          <span class="tnum">{{ t('access.role.grantedCount', { count: grantedOf(r.menuIds) }) }}</span>
        </button>
      </div>
    </FormDialog>

    <!-- 批量删除 -->
    <FormDialog
      v-model="batchOpen"
      :title="t('system.role.batchDelete')"
      :width="460"
      :confirm-text="t('system.common.delete')"
      @confirm="confirmDelete(batchIds)"
    >
      <div class="role-copy">
        <label v-for="r in deletableRoles" :key="r.roleId" class="role-copy__opt role-copy__check">
          <input v-model="batchIds" type="checkbox" :value="r.roleId" />
          <strong>{{ r.name }}</strong>
          <span class="tnum">{{ t('accessAdmin.role.memberCount', { count: Number(r.userCount || 0) }) }}</span>
        </label>
      </div>
    </FormDialog>

    <RoleCreate ref="roleCreateRef" @refresh="refresh" />
  </div>
</template>

<script setup lang="ts">
import { pageTabTo } from '@/config/pageGroups'
import { useI18n } from 'vue-i18n'
import { deleteRole, rolePage, updateRole } from '@/api/role'
import { adminPermissionsList } from '@/api/menu'
import { PERM } from '@/config/permission'
import { Dialog } from '@/utils/dialog'
import { resolveAvatarUrl } from '@/utils/assets'
import { MenuType, RoleStatus, type MenuInfo, type RoleInfo } from '@/types/v1/system'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import RoleCreate from '@/views/system/role/create.vue'
import PermissionEditor from '@/views/system/role/permissionEditor.vue'
import AccessSubNav from '@/views/system/components/AccessSubNav.vue'
import {
  NO_PERMISSION_ROLE_ID,
  SUPER_ADMIN_ROLE_ID,
  initialOf,
  stickerOf,
} from '@/views/system/components/accessTone'

defineOptions({ name: 'RoleView' })

const menuStore = useMenuStore()
const { t } = useI18n()
const roleCreateRef = useTemplateRef<InstanceType<typeof RoleCreate> | null>('roleCreateRef')

const loading = ref(false)
const saving = ref(false)
const roles = ref<RoleInfo[]>([])
const menus = ref<MenuInfo[]>([])

const currentId = ref('')
const draft = ref<string[]>([])
const keyword = ref('')
const statusFilter = ref<'' | RoleStatus>('')
const sheetOpen = ref(false)

const canAdd = computed(() => menuStore.hasButtonPermission(PERM.ROLE_ADD))
const canEdit = computed(() => menuStore.hasButtonPermission(PERM.ROLE_EDIT))
const canViewUsers = computed(() => menuStore.hasButtonPermission('user:view'))
const canDelete = computed(() => menuStore.hasButtonPermission(PERM.ROLE_DELETE))

const statusOptions = computed<{ label: string; value: '' | RoleStatus }[]>(() => [
  { label: t('system.common.all'), value: '' },
  { label: t('system.common.enabled'), value: RoleStatus.RoleStatus_Active },
  { label: t('system.common.inactive'), value: RoleStatus.RoleStatus_InActive },
])

const builtinCount = computed(() => roles.value.filter((r) => r.isBuiltin).length)
const filteredRoles = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  return roles.value.filter(
    (r) =>
      (!kw || r.name.toLowerCase().includes(kw) || (r.description || '').toLowerCase().includes(kw)) &&
      (!statusFilter.value || r.status === statusFilter.value),
  )
})
const deletableRoles = computed(() => roles.value.filter((r) => !r.isBuiltin))
const current = computed(() => roles.value.find((r) => r.roleId === currentId.value))
const locked = computed(() => !!current.value?.isBuiltin)
const currentSticker = computed(() => {
  const i = roles.value.findIndex((r) => r.roleId === currentId.value)
  return stickerOf(currentId.value, Math.max(0, i)).cls
})
const lockText = computed(() =>
  current.value?.roleId === NO_PERMISSION_ROLE_ID
    ? t('accessAdmin.role.lockNone')
    : current.value?.roleId === SUPER_ADMIN_ROLE_ID
      ? t('accessAdmin.role.lockAdmin')
      : t('access.role.locked'),
)

// 统计：全部「页面 + 操作」（不含目录）
const leafSet = computed(() => {
  const s = new Set<string>()
  const walk = (list: MenuInfo[]) =>
    list.forEach((m) => {
      if (m.type !== MenuType.MenuType_Directory) s.add(m.id)
      if (m.children?.length) walk(m.children)
    })
  walk(menus.value)
  return s
})
const leafTotal = computed(() => leafSet.value.size)
const grantedOf = (ids?: string[]) => (ids || []).filter((id) => leafSet.value.has(id)).length
const pct = (n: number) => (leafTotal.value ? `${Math.round((n / leafTotal.value) * 100)}%` : '0%')

const dirtyCount = computed(() => {
  if (!current.value) return 0
  const saved = new Set(current.value.menuIds || [])
  const now = new Set(draft.value)
  let leaf = 0
  let any = 0
  for (const id of new Set([...saved, ...now])) {
    if (saved.has(id) !== now.has(id)) {
      any++
      if (leafSet.value.has(id)) leaf++
    }
  }
  return leaf || (any ? 1 : 0)
})

const memberCount = computed(() => Number(current.value?.userCount || 0))
// 成员预览（后端随角色返回前 5 个）
const members = computed(() => current.value?.members || [])
const memberText = computed(() => {
  if (!memberCount.value) return t('accessAdmin.role.noMembers')
  const names = members.value
    .slice(0, 3)
    .map((u) => u.name || u.username)
    .join('、')
  if (!names) return t('accessAdmin.role.memberCount', { count: memberCount.value })
  return memberCount.value > 3
    ? t('accessAdmin.role.memberMore', { names, count: memberCount.value })
    : t('accessAdmin.role.memberAll', { names, count: memberCount.value })
})

const saveTone = computed(() => (locked.value ? 'is-locked' : dirtyCount.value ? 'is-dirty' : 'is-ok'))
const saveText = computed(() => {
  if (locked.value) return t('accessAdmin.role.lockedBar')
  if (dirtyCount.value)
    return t('accessAdmin.role.dirty', { count: dirtyCount.value, members: memberCount.value })
  return t('accessAdmin.role.saved')
})
// 手机抽屉底部：短文案
const saveTextShort = computed(() => {
  if (locked.value) return t('accessAdmin.role.lockedBar')
  if (dirtyCount.value) return t('accessV2.role.dirtyShort', { count: dirtyCount.value })
  return t('accessAdmin.role.saved')
})

const roleMenu = computed<ActionMenuItem[]>(() => [
  { key: 'delete', label: t('system.common.delete'), icon: 'HOutline:TrashIcon', danger: true },
])
const listMenu = computed<ActionMenuItem[]>(() => [
  { key: 'batchDelete', label: t('system.role.batchDelete'), icon: 'HOutline:TrashIcon', danger: true },
])



const selectRole = (roleId: string) => {
  currentId.value = roleId
  draft.value = [...(roles.value.find((r) => r.roleId === roleId)?.menuIds || [])]
}

const confirmLeave = (next: () => void) => {
  if (!dirtyCount.value || !current.value) return next()
  Dialog.info({
    showCancelButton: true,
    content: t('accessAdmin.role.leaveConfirm', { name: current.value.name }),
    confirmText: t('accessAdmin.role.discardShort'),
    cancelText: t('system.common.cancel'),
    onConfirm: next,
  })
}

const pickRole = (roleId: string) => {
  if (menuStore.isMobile) {
    selectRole(roleId)
    sheetOpen.value = true
    return
  }
  if (roleId === currentId.value) return
  confirmLeave(() => selectRole(roleId))
}

const onSheetClose = () => {
  // 抽屉关闭即丢弃未保存修改
  if (current.value) draft.value = [...(current.value.menuIds || [])]
}

const getRoles = async (prefer?: string) => {
  loading.value = true
  try {
    const { data: res } = await rolePage({ page: 1, pageSize: 500 })
    roles.value = res?.roles || []
    const keep = prefer || currentId.value
    const target =
      roles.value.find((r) => r.roleId === keep) ||
      roles.value.find((r) => !r.isBuiltin) ||
      roles.value[0]
    if (target && (target.roleId !== currentId.value || prefer)) selectRole(target.roleId)
  } finally {
    loading.value = false
  }
}

const getMenus = async () => {
  const { data: res } = await adminPermissionsList({})
  menus.value = res?.menus || []
}

const discard = () => {
  if (current.value) draft.value = [...(current.value.menuIds || [])]
}

const save = async () => {
  const r = current.value
  if (!r || locked.value) return
  saving.value = true
  try {
    const { data } = await updateRole({
      roleId: r.roleId,
      name: r.name,
      description: r.description,
      status: r.status,
      menuIds: draft.value,
    })
    const updated = data?.role
    const idx = roles.value.findIndex((x) => x.roleId === r.roleId)
    if (idx >= 0)
      roles.value[idx] = {
        ...roles.value[idx]!,
        ...(updated || {}),
        menuIds: [...draft.value],
        userCount: r.userCount,
        members: r.members,
      }
    feedback.success(t('accessAdmin.role.saveSuccess'))
    if (menuStore.isMobile) sheetOpen.value = false
  } finally {
    saving.value = false
  }
}

// 复制
const copyOpen = ref(false)
const copyFromId = ref('')
const copySources = computed(() => roles.value.filter((r) => r.roleId !== currentId.value))
const openCopy = () => {
  copyFromId.value = ''
  copyOpen.value = true
}
const applyCopy = () => {
  const src = roles.value.find((r) => r.roleId === copyFromId.value)
  if (!src) return
  draft.value = [...(src.menuIds || [])]
  copyOpen.value = false
  feedback.success(t('accessAdmin.role.copied', { name: src.name }))
}

// 批量删除
const batchOpen = ref(false)
const batchIds = ref<string[]>([])
const onListMenu = (key: string) => {
  if (key !== 'batchDelete') return
  batchIds.value = []
  batchOpen.value = true
}

const openCreate = () => roleCreateRef.value?.showDialog()
const openEditInfo = () => {
  if (!current.value || locked.value) return
  roleCreateRef.value?.showDialog(current.value.roleId)
}
const onRoleMenu = (key: string) => {
  if (key === 'delete' && current.value) confirmDelete([current.value.roleId])
}

const confirmDelete = (ids: string[]) => {
  const deletable = ids.filter((id) => !roles.value.find((r) => r.roleId === id)?.isBuiltin)
  if (!deletable.length) return
  Dialog.info({
    showCancelButton: true,
    content: t('system.role.confirmDeleteSelected'),
    confirmText: t('system.common.delete'),
    cancelText: t('system.common.cancel'),
    onConfirm: async () => {
      await deleteRole({ roleIds: deletable })
      feedback.success(t('system.common.deleteSuccess'))
      batchOpen.value = false
      sheetOpen.value = false
      if (deletable.includes(currentId.value)) currentId.value = ''
      getRoles()
    },
  })
}

const refresh = async (type: 'create' | 'update') => {
  if (type === 'create') {
    const before = new Set(roles.value.map((r) => r.roleId))
    await getRoles()
    const added = roles.value.find((r) => !before.has(r.roleId))
    if (added) selectRole(added.roleId)
    return
  }
  getRoles(currentId.value)
}

onMounted(() => {
  getMenus()
  getRoles()
})
</script>

<style scoped lang="scss">
.role-page :deep(.page-header__actions .app-icon-btn) {
  padding: 0;
  border-radius: 14px;
}
.role-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.role-dot {
  margin: 0 6px;
  opacity: 0.6;
}
.role-layout {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 24px;
}

/* 左侧角色列表 */
.role-list {
  position: sticky;
  top: 16px;
  flex: 1 1 280px;
  max-width: 320px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 14px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-xl, var(--app-radius-lg));
  background: var(--el-bg-color);
}
.role-list__head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 2px 4px 4px 8px;
}
.role-list__head h2 {
  flex: 1;
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.role-list__n {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.role-list__empty {
  margin: 8px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.role-list__add {
  margin-top: 6px;
  min-height: 44px;
  border-radius: 999px;
  font-weight: 700;
}
.role-search {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  color: var(--el-text-color-secondary);
}
.role-search:focus-within {
  border-color: var(--m-p);
}
.role-search :deep(svg) {
  width: 16px;
  height: 16px;
  flex: none;
}
.role-search input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: none;
  background: transparent;
  font: inherit;
  font-size: 0.875rem;
  color: var(--el-text-color-primary);
}
.role-filter {
  display: flex;
  gap: 2px;
  padding: 3px;
  margin-bottom: 4px;
  border-radius: 999px;
  background: var(--m-s2);
}
.role-filter__btn {
  flex: 1;
  min-height: 34px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  cursor: pointer;
}
.role-filter__btn.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}
.role-item {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-height: 64px;
  padding: 10px 12px;
  border: 1.5px solid transparent;
  border-radius: 18px;
  background: transparent;
  text-align: left;
  font: inherit;
  color: var(--el-text-color-primary);
  cursor: pointer;
}
.role-item:hover {
  background: var(--m-s2);
}
.role-item.is-on {
  border-color: var(--m-p);
  background: var(--m-pSoft);
}
.role-item__av {
  width: 42px;
  height: 42px;
  border-radius: 14px;
  font-size: 1rem;
}
.role-item__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.role-item__line {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.role-item__name {
  font-size: 0.9375rem;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.role-item__dirty {
  width: 8px;
  height: 8px;
  flex: none;
  border-radius: 50%;
  background: var(--m-warn);
}
.role-item__desc {
  font-size: 0.75rem;
  line-height: 1.4;
  color: var(--el-text-color-secondary);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.role-item__bar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.role-item__bar .role-bar {
  flex: 1;
}
.role-item__unsaved {
  flex: none;
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--m-warn);
}
.role-item__chev {
  width: 20px;
  height: 20px;
  flex: none;
  color: var(--el-text-color-secondary);
}
.role-item__members {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  flex: none;
}
.role-item__count {
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
}
.role-item__label {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.role-tag {
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
.role-tag :deep(svg) {
  width: 11px;
  height: 11px;
}
.role-tag--off {
  background: var(--m-s2);
  color: var(--m-ink2);
}

/* 进度条 */
.role-bar {
  display: block;
  height: 8px;
  border-radius: 999px;
  background: var(--m-s2);
  overflow: hidden;
}
.role-bar > span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--m-p);
}
.role-bar--lg {
  height: 10px;
}

/* 右侧详情 */
.role-detail {
  flex: 999 1 560px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.role-hero {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 22px 24px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-xl, var(--app-radius-lg));
  background: var(--el-bg-color);
}
.role-hero__top {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 14px;
}
.role-hero__av {
  width: 56px;
  height: 56px;
  border-radius: 18px;
  font-size: 1.375rem;
  transform: rotate(-5deg);
}
.role-hero__text {
  flex: 1 1 260px;
  min-width: 0;
}
.role-hero__line {
  display: flex;
  align-items: center;
  gap: 8px;
}
.role-hero__line h2 {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.5rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.role-hero__text p {
  margin: 4px 0 0;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.role-hero__acts {
  display: flex;
  align-items: center;
  gap: 8px;
}
.role-hero__acts :deep(button:not(.action-menu__trigger)) {
  min-height: 40px;
  border-radius: 999px;
  font-weight: 700;
}
.role-hero__stats {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 20px;
}
.role-hero__progress {
  flex: 1 1 280px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.role-hero__granted {
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.role-hero__granted span {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.role-hero__granted strong {
  font-size: 1.375rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.role-hero__granted em {
  font-style: normal;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.role-members {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.role-members__avs {
  display: flex;
  padding-left: 6px;
}
.role-members__av {
  width: 32px;
  height: 32px;
  margin-left: -6px;
  border-radius: 50%;
  border: 2px solid var(--el-bg-color);
  font-size: 0.8125rem;
}
.role-members__av.is-img {
  flex: none;
  overflow: hidden;
}
.role-members__text {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.role-members__link {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--m-pInk);
  white-space: nowrap;
}
.role-lock {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border-radius: 14px;
  background: var(--m-s2);
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.role-lock :deep(svg) {
  width: 18px;
  height: 18px;
  flex: none;
}

/* 保存条 */
.role-save {
  position: sticky;
  bottom: 18px;
  z-index: 10;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  padding: 10px 10px 10px 22px;
  border-radius: 999px;
  background: var(--m-ink);
  color: var(--m-bg);
  box-shadow: var(--app-shadow-lg);
}
.role-save__dot {
  width: 10px;
  height: 10px;
  flex: none;
  border-radius: 50%;
}
.role-save__dot.is-ok { background: var(--m-ok); }
.role-save__dot.is-dirty { background: var(--m-a); }
.role-save__dot.is-locked { background: var(--m-ink2); }
.role-save__text {
  flex: 1;
  min-width: 0;
  font-size: 0.875rem;
  font-weight: 700;
}
.role-save :deep(button) {
  min-height: 40px;
  border-radius: 999px;
  font-weight: 700;
}
.role-save__ghost {
  color: var(--m-bg) !important;
}
.role-save__ghost:hover {
  background: color-mix(in srgb, var(--m-bg) 14%, transparent) !important;
}

/* 复制 / 批量删除 */
.role-copy__hint {
  margin: 0 0 12px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.role-copy {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.role-copy__opt {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 48px;
  padding: 0 14px;
  border: 1.5px solid var(--el-border-color-light);
  border-radius: var(--app-radius);
  background: var(--el-bg-color);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.role-copy__opt strong {
  flex: 1;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.role-copy__opt span {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.role-copy__opt.is-on {
  border-color: var(--m-p);
  background: var(--m-pSoft);
}
.role-copy__check input {
  width: 18px;
  height: 18px;
  accent-color: var(--m-p);
}

/* 手机 */
.role-sheet__head {
  display: flex;
  align-items: center;
  gap: 12px;
}
.role-sheet__av {
  width: 44px;
  height: 44px;
  flex: none;
  border-radius: 15px;
  font-size: 1.125rem;
  transform: rotate(-5deg);
}
.role-sheet__title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.role-sheet__name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-family: var(--app-font-display);
  font-size: 1.1875rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.role-sheet__save {
  flex: 1;
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.role-sheet {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.role-sheet__desc {
  margin: 0;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.role-sheet__stats {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.role-sheet__acts {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.role-skeleton {
  border-radius: var(--app-radius-xl, var(--app-radius-lg));
  background: var(--el-fill-color-light);
  animation: role-pulse 1.4s ease-in-out infinite;
}
.role-skeleton--list {
  flex: 1 1 280px;
  max-width: 320px;
  height: 360px;
}
.role-skeleton--detail {
  flex: 999 1 560px;
  height: 480px;
}
@keyframes role-pulse {
  50% {
    opacity: 0.55;
  }
}

@media (width < 768px) {
  .role-layout {
    display: block;
  }
  .role-filter__btn {
    min-height: 44px;
  }
  .role-list {
    position: static;
    max-width: none;
    padding: 0;
    border: 0;
    background: transparent;
    gap: 12px;
  }
  .role-item {
    min-height: 88px;
    padding: calc(var(--cmp-card-unit) * 16);
    border: 1px solid var(--el-border-color-light);
    border-radius: var(--app-radius-lg);
    background: var(--el-bg-color);
  }
  .role-item__av {
    width: 46px;
    height: 46px;
  }
  .role-skeleton--list {
    max-width: none;
  }
  .role-skeleton--detail {
    display: none;
  }
}
</style>
