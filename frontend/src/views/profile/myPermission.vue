<!-- 我的权限：角色摘要 + 可访问页面标签；「查看全部」弹窗里是完整的菜单 / 按钮权限树（桌面树表 / 手机缩进卡）。 -->
<template>
  <AppPanel class="perm">
    <template #header>
      <h2 class="pf-side-title">{{ t('user.permissionTitle') }}</h2>
    </template>
    <template #actions>
      <RouterLink v-if="canViewRoles" :to="pageTabTo('access', 'role')" class="pf-link">{{ t('profile.perm.viewRoles') }} →</RouterLink>
    </template>

    <button type="button" class="perm-sum" @click="detailOpen = true">
      <span class="m-sticker m-sticker--solid-a2 perm-sum__sk" aria-hidden="true">
        <UIcon name="i-lucide-shield-check" />
      </span>
      <span class="perm-sum__text">
        <strong>{{ roleLabel }}</strong>
        <span class="m-num">{{ summaryText }}</span>
      </span>
      <UIcon name="i-lucide-chevron-right" class="perm-sum__chev" aria-hidden="true" />
    </button>

    <div v-if="pageTags.length" class="perm-tags">
      <span v-for="tag in visibleTags" :key="tag" class="perm-tag">
        <UIcon name="i-lucide-check" class="perm-tag__ico" aria-hidden="true" />{{ tag }}
      </span>
      <button v-if="hiddenCount > 0" type="button" class="perm-tag perm-tag--more" @click="detailOpen = true">
        +{{ hiddenCount }}
      </button>
    </div>
    <p v-else class="perm-empty">{{ t('user.noPermission') }}</p>

    <FormDialog v-model="detailOpen" :title="t('user.permissionTitle')" :width="860" :show-footer="false">
      <p class="perm-desc">
        <StatusPill variant="primary" :dot="false" :label="roleLabel" />
        <span>{{ t('user.permissionDescription') }}</span>
        <span class="perm-desc__stats m-num">
          {{ menuCount }} {{ t('user.menuPermission') }} · {{ menuStore.allButtonPermissions.length }}
          {{ t('user.buttonPermission') }}
        </span>
      </p>

      <DataTable
        v-if="!menuStore.isMobile"
        :columns="columns"
        :rows="(menuStore.allMenuList as unknown as Record<string, unknown>[])"
        row-key="id"
        tree
        :empty-text="t('user.noPermission')"
      >
        <template #cell-title="{ row }">
          {{ translateKnownText(String(row.title || '')) }}
        </template>
        <template #cell-type="{ row }">
          <StatusPill :variant="typeVariant(row.type)" :dot="false" :label="typeLabel(row.type)" />
        </template>
        <template #cell-icon="{ row }">
          <component
            :is="menuStore.iconComponents[row.icon as string]"
            v-if="row.icon && menuStore.iconComponents[row.icon as string]"
            class="perm-icon"
          />
          <span v-else class="text-dim">—</span>
        </template>
        <template #cell-status="{ row }">
          <StatusPill :variant="statusVariant(row.status)" :label="statusLabel(row.status)" />
        </template>
      </DataTable>

      <template v-else>
        <EmptyState v-if="!flatList.length" icon="HOutline:ShieldCheckIcon" :title="t('user.noPermission')" />
        <div v-else class="perm-cards">
          <div
            v-for="row in flatList"
            :key="row.id"
            class="perm-line"
            :style="{ marginInlineStart: `${row._depth * 14}px` }"
          >
            <component
              :is="menuStore.iconComponents[row.icon]"
              v-if="row.icon && menuStore.iconComponents[row.icon]"
              class="perm-icon"
            />
            <span class="perm-line__title">{{ translateKnownText(row.title) }}</span>
            <StatusPill :variant="typeVariant(row.type)" :dot="false" :label="typeLabel(row.type)" />
            <StatusPill
              v-if="row.status !== MenuStatus.MenuStatus_Active"
              :variant="statusVariant(row.status)"
              :label="statusLabel(row.status)"
            />
          </div>
        </div>
      </template>
    </FormDialog>
  </AppPanel>
</template>

<script setup lang="ts">
import { pageTabTo } from '@/config/pageGroups'
import { MenuStatus, MenuType } from '@/types/v1/system'
import type { MenuInfo } from '@/types/v1/system'
import { translateKnownText } from '@/locales'
import { useAccessibleRoutes } from '@/composables/useAccessibleRoutes'
import { useI18n } from 'vue-i18n'

const menuStore = useMenuStore()
const userStore = useUserStore()
const nodeStore = useNodeStore()
const accessible = useAccessibleRoutes()
const { t } = useI18n()

const detailOpen = ref(false)
const TAG_LIMIT = 9

const roleLabel = computed(() => userStore.getUserRoleName() || t('user.noPermission'))
const canViewRoles = computed(() => accessible.value.has('/system/access') && menuStore.hasButtonPermission('role:view'))

const columns = computed(() => [
  { key: 'title', title: t('user.menuFunctionName'), minWidth: 200 },
  { key: 'type', title: t('common.type'), width: 100 },
  { key: 'path', title: t('user.menuPath'), minWidth: 180 },
  { key: 'icon', title: t('user.icon'), width: 72, align: 'center' as const },
  { key: 'status', title: t('common.status'), width: 100 },
])

const typeVariant = (type: unknown) => {
  if (type === MenuType.MenuType_Directory) return 'info' as const
  if (type === MenuType.MenuType_Menu) return 'primary' as const
  return 'warning' as const
}
const typeLabel = (type: unknown) => {
  if (type === MenuType.MenuType_Directory) return t('user.directory')
  if (type === MenuType.MenuType_Menu) return t('user.menu')
  return t('user.button')
}
const statusVariant = (status: unknown) =>
  status === MenuStatus.MenuStatus_Active ? ('success' as const) : ('error' as const)
const statusLabel = (status: unknown) =>
  status === MenuStatus.MenuStatus_Active ? t('common.enabled') : t('common.disabled')

const menuCount = computed(() => {
  const count = (list: MenuInfo[]): number =>
    list.reduce((sum, item) => sum + 1 + (item.children?.length ? count(item.children) : 0), 0)
  return count(menuStore.allMenuList)
})

type FlatRow = MenuInfo & { _depth: number }
const flatList = computed(() => {
  const out: FlatRow[] = []
  const walk = (list: MenuInfo[], depth: number) => {
    list.forEach((item) => {
      out.push({ ...item, _depth: depth })
      if (item.children?.length) walk(item.children, depth + 1)
    })
  }
  walk(menuStore.allMenuList, 0)
  return out
})

/** 可进入的页面（菜单类型、已启用）做成标签 */
const pageTags = computed(() => {
  const tags: string[] = []
  const seen = new Set<string>()
  for (const row of flatList.value) {
    if (row.type !== MenuType.MenuType_Menu || row.status !== MenuStatus.MenuStatus_Active) continue
    const title = translateKnownText(row.title)
    if (!title || seen.has(title)) continue
    seen.add(title)
    tags.push(title)
  }
  return tags
})
const visibleTags = computed(() => pageTags.value.slice(0, TAG_LIMIT))
const hiddenCount = computed(() => Math.max(0, pageTags.value.length - TAG_LIMIT))

const summaryText = computed(() => {
  const base = t('profile.perm.summary', {
    pages: pageTags.value.length,
    actions: menuStore.allButtonPermissions.length,
  })
  const nodes = nodeStore.nodes.length
  return nodes ? `${base} · ${t('profile.perm.nodes', { n: nodes })}` : base
})
</script>

<style scoped lang="scss">
.perm :deep(.app-panel__body) {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding-top: 12px;
}
.pf-side-title {
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.pf-link {
  display: inline-flex;
  align-items: center;
  min-height: 36px;
  color: var(--m-pInk);
  font-size: 0.8125rem;
  font-weight: 700;
  text-decoration: none;
}
.pf-link:hover {
  color: var(--el-text-color-primary);
}
.perm-sum {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 12px 14px;
  border: 0;
  border-radius: 16px;
  background: var(--m-s2);
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.perm-sum:hover {
  background: var(--el-fill-color-light);
}
.perm-sum__sk {
  width: 40px;
  height: 40px;
  border-radius: 13px;
  font-size: 20px;
  transform: rotate(4deg);
}
.perm-sum__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  strong {
    font-size: 0.875rem;
    font-weight: 800;
  }
  span {
    font-size: 0.75rem;
    color: var(--el-text-color-secondary);
  }
}
.perm-sum__chev {
  width: 18px;
  height: 18px;
  color: var(--el-text-color-secondary);
}
.perm-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.perm-tag {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 30px;
  padding: 0 11px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-regular);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 600;
}
.perm-tag--more {
  cursor: pointer;
  color: var(--m-pInk);
  font-weight: 800;
}
.perm-tag__ico {
  width: 14px;
  height: 14px;
  color: var(--m-ok);
}
.perm-empty {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.perm-desc {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
  margin: 0 0 0.85rem;
  font-size: 0.8125rem;
  line-height: 1.45;
  color: var(--el-text-color-secondary);
}
.perm-desc__stats {
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.perm-icon {
  width: 16px;
  height: 16px;
  flex: none;
  color: var(--el-text-color-secondary);
}
.text-dim {
  color: var(--el-text-color-placeholder);
}
.perm-cards {
  display: flex;
  flex-direction: column;
}
.perm-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-height: 44px;
  padding: 6px 0;
  border-top: 1px solid var(--el-border-color-light);
}
.perm-line:first-child {
  border-top: 0;
}
.perm-line__title {
  flex: 1;
  min-width: 0;
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
</style>
