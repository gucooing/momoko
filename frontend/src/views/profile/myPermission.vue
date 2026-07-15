<!-- 我的权限：只读树表 / 移动缩进卡；统计用内联数字 -->
<template>
  <AppPanel :title="t('user.permissionTitle')" title-icon="HOutline:ShieldCheckIcon">
    <template #actions>
      <div class="perm-stats">
        <span class="perm-stats__item">
          <em>{{ menuCount }}</em>
          {{ t('user.menuPermission') }}
        </span>
        <span class="perm-stats__sep" aria-hidden="true" />
        <span class="perm-stats__item">
          <em>{{ menuStore.allButtonPermissions.length }}</em>
          {{ t('user.buttonPermission') }}
        </span>
      </div>
    </template>

    <p class="perm-desc">
      <StatusPill variant="primary" :dot="false" :label="roleLabel" />
      <span>{{ t('user.permissionDescription') }}</span>
    </p>

    <!-- 桌面：树表 -->
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

    <!-- 移动：缩进卡 -->
    <template v-else>
      <EmptyState
        v-if="!flatList.length"
        icon="HOutline:ShieldCheckIcon"
        :title="t('user.noPermission')"
      />
      <div v-else class="perm-cards">
        <EntityCard
          v-for="row in flatList"
          :key="row.id"
          class="perm-card"
          :style="{ marginInlineStart: `${row._depth * 14}px` }"
        >
          <template #title>
            <div class="perm-card__title">
              <component
                :is="menuStore.iconComponents[row.icon]"
                v-if="row.icon && menuStore.iconComponents[row.icon]"
                class="perm-icon"
              />
              <span>{{ translateKnownText(row.title) }}</span>
            </div>
          </template>
          <template #status>
            <StatusPill :variant="statusVariant(row.status)" :label="statusLabel(row.status)" />
          </template>
          <template #meta>
            <StatusPill :variant="typeVariant(row.type)" :dot="false" :label="typeLabel(row.type)" />
            <span v-if="row.path" class="perm-card__path">{{ row.path }}</span>
          </template>
        </EntityCard>
      </div>
    </template>
  </AppPanel>
</template>

<script setup lang="ts">
import { MenuStatus, MenuType } from '@/types/v1/system'
import type { MenuInfo } from '@/types/v1/system'
import { translateKnownText } from '@/locales'
import { useI18n } from 'vue-i18n'

const menuStore = useMenuStore()
const userStore = useUserStore()
const { t } = useI18n()

const roleLabel = computed(() => userStore.getUserRoleName() || t('user.noPermission'))

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
</script>

<style scoped lang="scss">
.perm-stats {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.perm-stats__item em {
  font-style: normal;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: var(--el-text-color-primary);
  margin-right: 0.2rem;
}
.perm-stats__sep {
  width: 1px;
  height: 12px;
  background: var(--el-border-color);
}
.perm-desc {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
  margin: 0 0 0.75rem;
  font-size: 0.8125rem;
  line-height: 1.45;
  color: var(--el-text-color-secondary);
}
.perm-icon {
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.text-dim {
  color: var(--el-text-color-placeholder);
}
.perm-cards {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.perm-card__title {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  min-width: 0;
}
.perm-card__path {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}
</style>
