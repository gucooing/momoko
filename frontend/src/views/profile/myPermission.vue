<!-- 我的权限 -->
<template>
  <BaseCard>
    <template #header>
      <div class="flex flex-col md:flex-row md:items-end justify-between">
        <div class="space-y-2">
          <div class="flex items-center gap-3">
            <h1 class="text-2xl font-semibold">我的权限</h1>
            <BaseTag :text="userStore.getUserRoleName()" />
          </div>
          <p class="text-sm text-(--el-text-color-secondary)">
            查看您在系统中获准访问的菜单项与操作功能。如有权限变动，请联系系统管理员。
          </p>
        </div>

        <div class="flex items-center justify-center gap-10 mt-6 md:mt-0 pr-4">
          <el-statistic :value="menuCount" title="菜单权限" class="text-center" />

          <el-divider direction="vertical" />

          <el-statistic
            :value="menuStore.allButtonPermissions.length"
            title="按钮权限"
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
      <el-table-column prop="title" label="菜单/功能名称" min-width="200" :align="TABLE_CONFIG.align" />
      <el-table-column prop="type" label="类型" min-width="100" :align="TABLE_CONFIG.align">
        <template #default="{ row }">
          <BaseTag v-if="row.type === MenuType.MenuType_Directory" type="info" text="目录" />
          <BaseTag v-else-if="row.type === MenuType.MenuType_Menu" type="primary" text="菜单" />
          <BaseTag v-else-if="row.type === MenuType.MenuType_Button" type="warning" text="按钮" />
        </template>
      </el-table-column>
      <el-table-column prop="path" label="菜单路径" min-width="250" :align="TABLE_CONFIG.align" />
      <el-table-column prop="icon" label="图标" min-width="100" :align="TABLE_CONFIG.align">
        <template #default="{ row }">
          <el-icon v-if="row.icon"><component :is="menuStore.iconComponents[row.icon]" /></el-icon>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="150" :align="TABLE_CONFIG.align">
        <template #default="{ row }">
          <BaseTag :type="row.status === MenuStatus.MenuStatus_Active ? 'success' : 'danger'" :text="row.status === MenuStatus.MenuStatus_Active ? '启用' : '禁用'" />
        </template>
      </el-table-column>
    </el-table>

    <!-- mobile: cards -->
    <div v-else class="mobile-card-list">
      <template v-for="item in menuStore.allMenuList" :key="item.id">
        <div class="mobile-card">
          <div class="mobile-card-header">
            <span class="mobile-card-title">{{ item.title }}</span>
            <BaseTag v-if="item.type === MenuType.MenuType_Directory" type="info" text="目录" />
            <BaseTag v-else-if="item.type === MenuType.MenuType_Menu" type="primary" text="菜单" />
            <BaseTag v-else type="warning" text="按钮" />
          </div>
          <div class="mobile-card-meta">{{ item.path }}</div>
          <div class="mobile-card-meta">
            <el-icon v-if="item.icon" size="14"><component :is="menuStore.iconComponents[item.icon]" /></el-icon>
            <BaseTag :type="item.status === MenuStatus.MenuStatus_Active ? 'success' : 'danger'" :text="item.status === MenuStatus.MenuStatus_Active ? '启用' : '禁用'" />
          </div>
        </div>
        <template v-if="item.children?.length">
          <div v-for="child in item.children" :key="child.id" class="mobile-card" style="marginLeft: 1.2rem">
            <div class="mobile-card-header">
              <span class="mobile-card-title">{{ child.title }}</span>
              <BaseTag v-if="child.type === MenuType.MenuType_Directory" type="info" text="目录" />
              <BaseTag v-else-if="child.type === MenuType.MenuType_Menu" type="primary" text="菜单" />
              <BaseTag v-else type="warning" text="按钮" />
            </div>
            <div class="mobile-card-meta">{{ child.path }}</div>
            <div class="mobile-card-meta">
              <el-icon v-if="child.icon" size="14"><component :is="menuStore.iconComponents[child.icon]" /></el-icon>
              <BaseTag :type="child.status === MenuStatus.MenuStatus_Active ? 'success' : 'danger'" :text="child.status === MenuStatus.MenuStatus_Active ? '启用' : '禁用'" />
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

const menuStore = useMenuStore()
const userStore = useUserStore()

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
