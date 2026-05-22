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

    <el-table
      :data="menuStore.allMenuList"
      :border="TABLE_CONFIG.border"
      row-key="id"
      :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
      default-expand-all
      show-overflow-tooltip
    >
      <el-table-column
        prop="title"
        label="菜单/功能名称"
        min-width="200"
        :align="TABLE_CONFIG.align"
      />
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
          <el-icon v-if="row.icon">
            <component :is="menuStore.iconComponents[row.icon]" />
          </el-icon>
        </template>
      </el-table-column>

      <el-table-column label="状态" width="150" :align="TABLE_CONFIG.align">
        <template #default="{ row }">
          <BaseTag
            :type="row.status === MenuStatus.MenuStatus_Active ? 'success' : 'danger'"
            :text="row.status === MenuStatus.MenuStatus_Active ? '启用' : '禁用'"
          />
        </template>
      </el-table-column>
    </el-table>
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

<style scoped lang="scss"></style>
