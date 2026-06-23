<!-- 我的权限 -->
<template>
  <BaseCard>
    <template #header>
      <div class="flex flex-col md:flex-row md:items-end justify-between">
        <div class="space-y-2">
          <div class="flex items-center gap-3">
            <h1 class="text-2xl font-semibold">{{ t('user.permissionTitle') }}</h1>
            <BaseTag :text="userStore.getUserRoleName()" />
          </div>
          <p class="text-sm text-(--el-text-color-secondary)">
            {{ t('user.permissionDescription') }}
          </p>
        </div>

        <div class="flex items-center justify-center gap-10 mt-6 md:mt-0 pr-4">
          <el-statistic :value="menuCount" :title="t('user.menuPermission')" class="text-center" />

          <el-divider direction="vertical" />

          <el-statistic
            :value="menuStore.allButtonPermissions.length"
            :title="t('user.buttonPermission')"
            class="text-center"
          />
        </div>
      </div>
    </template>

    <!-- desktop: table -->
    <el-table
      v-if="!menuStore.isMobile"
      :data="menuStore.allMenuList"
      :border="TABLE_CONFIG.border"
      row-key="id"
      :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
      default-expand-all
      show-overflow-tooltip
    >
      <el-table-column prop="title" :label="t('user.menuFunctionName')" min-width="200" :align="TABLE_CONFIG.align">
        <template #default="{ row }">{{ translateKnownText(row.title) }}</template>
      </el-table-column>
      <el-table-column prop="type" :label="t('common.type')" min-width="100" :align="TABLE_CONFIG.align">
        <template #default="{ row }">
          <BaseTag :type="getMenuTypeTagType(row.type)" :text="getMenuTypeText(row.type)" />
        </template>
      </el-table-column>
      <el-table-column prop="path" :label="t('user.menuPath')" min-width="250" :align="TABLE_CONFIG.align" />
      <el-table-column prop="icon" :label="t('user.icon')" min-width="100" :align="TABLE_CONFIG.align">
        <template #default="{ row }">
          <el-icon v-if="row.icon"><component :is="menuStore.iconComponents[row.icon]" /></el-icon>
        </template>
      </el-table-column>
      <el-table-column :label="t('common.status')" width="150" :align="TABLE_CONFIG.align">
        <template #default="{ row }">
          <BaseTag :type="row.status === MenuStatus.MenuStatus_Active ? 'success' : 'danger'" :text="getMenuStatusText(row.status)" />
        </template>
      </el-table-column>
    </el-table>

    <!-- mobile: cards -->
    <div v-else class="mobile-card-list">
      <template v-for="item in menuStore.allMenuList" :key="item.id">
        <div class="mobile-card">
          <div class="mobile-card-header">
            <span class="mobile-card-title">{{ translateKnownText(item.title) }}</span>
            <BaseTag :type="getMenuTypeTagType(item.type)" :text="getMenuTypeText(item.type)" />
          </div>
          <div class="mobile-card-meta">{{ item.path }}</div>
          <div class="mobile-card-meta">
            <el-icon v-if="item.icon" size="14"><component :is="menuStore.iconComponents[item.icon]" /></el-icon>
              <BaseTag :type="item.status === MenuStatus.MenuStatus_Active ? 'success' : 'danger'" :text="getMenuStatusText(item.status)" />
          </div>
        </div>
        <template v-if="item.children?.length">
          <div v-for="child in item.children" :key="child.id" class="mobile-card" style="marginLeft: 1.2rem">
            <div class="mobile-card-header">
              <span class="mobile-card-title">{{ translateKnownText(child.title) }}</span>
              <BaseTag :type="getMenuTypeTagType(child.type)" :text="getMenuTypeText(child.type)" />
            </div>
            <div class="mobile-card-meta">{{ child.path }}</div>
            <div class="mobile-card-meta">
              <el-icon v-if="child.icon" size="14"><component :is="menuStore.iconComponents[child.icon]" /></el-icon>
              <BaseTag :type="child.status === MenuStatus.MenuStatus_Active ? 'success' : 'danger'" :text="getMenuStatusText(child.status)" />
            </div>
          </div>
        </template>
      </template>
    </div>
  </BaseCard>
</template>

<script setup lang="ts">
import { TABLE_CONFIG } from '@/config/elementConfig'
import { MenuStatus, MenuType } from '@/types/v1/system'
import type { MenuInfo } from '@/types/v1/system'
import { translateKnownText } from '@/locales'
import { useI18n } from 'vue-i18n'

const menuStore = useMenuStore()
const userStore = useUserStore()
const { t } = useI18n()

const getMenuTypeTagType = (type: MenuType) => {
  if (type === MenuType.MenuType_Directory) return 'info'
  if (type === MenuType.MenuType_Menu) return 'primary'
  return 'warning'
}

const getMenuTypeText = (type: MenuType) => {
  if (type === MenuType.MenuType_Directory) return t('user.directory')
  if (type === MenuType.MenuType_Menu) return t('user.menu')
  return t('user.button')
}

const getMenuStatusText = (status: MenuStatus) => {
  return status === MenuStatus.MenuStatus_Active ? t('common.enabled') : t('common.disabled')
}

const menuCount = computed(() => {
  const count = (list: MenuInfo[]): number => {
    return list.reduce((sum, item) => {
      return sum + 1 + (item.children?.length ? count(item.children) : 0)
    }, 0)
  }
  return count(menuStore.allMenuList)
})
</script>

<style scoped lang="scss">
.mobile-card-list { display: flex; flex-direction: column; gap: 0.5rem; }
.mobile-card { padding: 0.65rem 0.75rem; border: 1px solid var(--el-border-color-extra-light); border-radius: 0.6rem; background: var(--el-bg-color); }
.mobile-card-header { display: flex; align-items: center; gap: 0.4rem; }
.mobile-card-title { font-size: 0.85rem; font-weight: 700; color: var(--el-text-color-primary); }
.mobile-card-meta { display: flex; align-items: center; gap: 0.35rem; margin-top: 0.2rem; font-size: 0.72rem; color: var(--el-text-color-secondary); }
</style>
