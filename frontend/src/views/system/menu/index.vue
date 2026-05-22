<template>
  <div>
    <el-card shadow="never" class="card-clear-mb">
      <el-form :model="queryForm" label-width="auto" ref="queryFormRef" @keyup.enter="getMenuList">
        <el-row :gutter="10">
          <el-col :xs="24" :sm="12" :md="12" :lg="6" :xl="5">
            <el-form-item label="菜单标题" prop="title">
              <el-input v-model="queryForm.title" placeholder="请输入" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="6" :xl="5">
            <el-form-item label="菜单路径" prop="path">
              <el-input v-model="queryForm.path" placeholder="请输入" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="6" :xl="5">
            <el-form-item label="类型" prop="type">
              <el-select v-model="queryForm.type" placeholder="请选择">
                <el-option label="目录" :value="MenuType.MenuType_Directory" />
                <el-option label="菜单" :value="MenuType.MenuType_Menu" />
                <el-option label="按钮" :value="MenuType.MenuType_Button" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="6" :xl="5">
            <el-form-item label="状态" prop="status">
              <el-select v-model="queryForm.status" placeholder="请选择">
                <el-option label="启用" :value="MenuStatus.MenuStatus_Active" />
                <el-option label="禁用" :value="MenuStatus.MenuStatus_InActive" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="6" :xl="4">
            <el-form-item>
              <el-button type="primary" :icon="menuStore.iconComponents.Search" @click="getMenuList"
                >搜索</el-button
              >
              <el-button :icon="menuStore.iconComponents.Refresh" @click="reset">重置</el-button>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </el-card>

    <el-card shadow="never" class="card-mt-16">
      <div class="operation-container">
        <el-button
          type="primary"
          :icon="menuStore.iconComponents.Plus"
          @click="menuCreateRef?.showDialog(undefined)"
          v-permission="[PERM.MENU_ADD]"
          >新增菜单</el-button
        >
      </div>
      <el-table
        v-loading="loading"
        :data="filteredMenuList"
        :border="TABLE_CONFIG.border"
        row-key="id"
        :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
        default-expand-all
        show-overflow-tooltip
      >
        <el-table-column
          prop="title"
          label="菜单标题"
          min-width="200"
          :align="TABLE_CONFIG.align"
        />
        <el-table-column prop="path" label="菜单路径" min-width="250" :align="TABLE_CONFIG.align" />
        <el-table-column prop="type" label="类型" min-width="100" :align="TABLE_CONFIG.align">
          <template #default="{ row }">
            <BaseTag v-if="row.type === MenuType.MenuType_Directory" type="info" text="目录" />
            <BaseTag
              v-else-if="row.type === MenuType.MenuType_Menu"
              type="primary"
              text="菜单"
            />
            <BaseTag
              v-else-if="row.type === MenuType.MenuType_Button"
              type="warning"
              text="按钮"
            />
          </template>
        </el-table-column>
        <el-table-column prop="icon" label="图标" min-width="100" :align="TABLE_CONFIG.align">
          <template #default="{ row }">
            <el-icon v-if="row.icon">
              <component :is="menuStore.iconComponents[row.icon]" />
            </el-icon>
          </template>
        </el-table-column>
        <el-table-column prop="order" label="排序" min-width="100" :align="TABLE_CONFIG.align" />
        <el-table-column prop="status" label="状态" min-width="100" :align="TABLE_CONFIG.align">
          <template #default="{ row }">
            <BaseTag
              :type="row.status === MenuStatus.MenuStatus_Active ? 'success' : 'danger'"
              :text="row.status === MenuStatus.MenuStatus_Active ? '启用' : '禁用'"
            />
          </template>
        </el-table-column>
        <el-table-column
          prop="createTime"
          label="创建时间"
          min-width="180"
          :align="TABLE_CONFIG.align"
        />
        <el-table-column label="操作" width="150" fixed="right" :align="TABLE_CONFIG.align">
          <template #default="{ row }: { row: MenuInfo }">
            <el-button
              type="primary"
              link
              :icon="menuStore.iconComponents.Edit"
              @click="menuCreateRef?.showDialog(row.id)"
              v-permission="[PERM.MENU_EDIT]"
              >编辑</el-button
            >
            <AdaptiveConfirm
              v-if="!isSystemMenu(row)"
              title="确定要删除选中的菜单吗？"
              :placement="POPCONFIRM_CONFIG.placement"
              :width="POPCONFIRM_CONFIG.width"
              @confirm="deleteMenuHandle(row.id)"
            >
              <template #reference>
                <el-button
                  type="danger"
                  link
                  :icon="menuStore.iconComponents.Delete"
                  v-permission="[PERM.MENU_DELETE]"
                >
                  删除
                </el-button>
              </template>
            </AdaptiveConfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <MenuCreate ref="menuCreateRef" @refresh="refresh" />
  </div>
</template>

<script setup lang="ts">
import { adminPermissionsList, adminDeletePermissions } from '@/api/menu'
import { TABLE_CONFIG, POPCONFIRM_CONFIG } from '@/config/elementConfig'
import { PERM } from '@/config/permission'
import MenuCreate from '@/views/system/menu/create.vue'
import { MenuType, MenuStatus } from '@/types/v1/system'
import type { MenuInfo } from '@/types/v1/system'
import type { FormInstance } from 'element-plus'

defineOptions({ name: 'MenuView' })

const menuStore = useMenuStore()
const queryFormRef = useTemplateRef<FormInstance>('queryFormRef')
const menuCreateRef = useTemplateRef<InstanceType<typeof MenuCreate> | null>('menuCreateRef')

const loading = ref(false)

const queryForm = ref({
  title: '',
  path: '',
  type: '' as string,
  status: '' as string,
})

const menuList = ref<MenuInfo[]>([])

const filterMenuTree = (
  menus: MenuInfo[],
  predicate: (menu: MenuInfo) => boolean,
): MenuInfo[] => {
  return menus
    .map((menu) => {
      const children = menu.children?.length
        ? filterMenuTree(menu.children, predicate)
        : []
      if (predicate(menu) || children.length > 0) {
        return { ...menu, children: children.length > 0 ? children : menu.children } as MenuInfo
      }
      return null
    })
    .filter((m): m is MenuInfo => m !== null)
}

const filteredMenuList = computed(() => {
  let filtered = menuList.value

  if (queryForm.value.title) {
    filtered = filterMenuTree(filtered, (m) => m.title.includes(queryForm.value.title))
  }
  if (queryForm.value.path) {
    filtered = filterMenuTree(filtered, (m) => m.path.includes(queryForm.value.path))
  }
  if (queryForm.value.type) {
    filtered = filterMenuTree(filtered, (m) => m.type === queryForm.value.type)
  }
  if (queryForm.value.status) {
    filtered = filterMenuTree(filtered, (m) => m.status === queryForm.value.status)
  }

  return filtered
})

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
  queryFormRef.value?.resetFields()
}

const isSystemMenu = (menu: MenuInfo) => {
  const builtInFallback = (menu as MenuInfo & { isBuiltIn?: boolean }).isBuiltIn
  return Boolean(menu.isSystem || builtInFallback)
}

const deleteMenuHandle = async (id: string) => {
  await adminDeletePermissions({ menuId: id })
  ElMessage.success('删除成功')
  getMenuList()
}

const refresh = () => {
  getMenuList()
}

onMounted(() => {
  getMenuList()
})
</script>

<style scoped></style>
