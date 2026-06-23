<template>
  <div>
    <el-card shadow="never" class="card-clear-mb">
      <el-form :model="queryForm" label-width="auto" ref="queryFormRef" @keyup.enter="getMenuList">
        <el-row :gutter="10">
          <el-col :xs="24" :sm="12" :md="12" :lg="6" :xl="5">
            <el-form-item :label="t('system.common.menuTitle')" prop="title">
              <el-input v-model="queryForm.title" :placeholder="t('system.menu.inputPlaceholder')" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="6" :xl="5">
            <el-form-item :label="t('system.common.menuPath')" prop="path">
              <el-input v-model="queryForm.path" :placeholder="t('system.menu.inputPlaceholder')" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="6" :xl="5">
            <el-form-item :label="t('system.common.type')" prop="type">
              <el-select v-model="queryForm.type" :placeholder="t('system.common.select')">
                <el-option :label="t('system.common.directory')" :value="MenuType.MenuType_Directory" />
                <el-option :label="t('system.common.menu')" :value="MenuType.MenuType_Menu" />
                <el-option :label="t('system.common.button')" :value="MenuType.MenuType_Button" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="6" :xl="5">
            <el-form-item :label="t('system.common.status')" prop="status">
              <el-select v-model="queryForm.status" :placeholder="t('system.common.select')">
                <el-option :label="t('system.common.enabled')" :value="MenuStatus.MenuStatus_Active" />
                <el-option :label="t('system.common.disabled')" :value="MenuStatus.MenuStatus_InActive" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="6" :xl="4">
            <el-form-item>
              <el-button type="primary" :icon="menuStore.iconComponents.Search" @click="getMenuList"
                >{{ t('system.common.search') }}</el-button
              >
              <el-button :icon="menuStore.iconComponents.Refresh" @click="reset">{{ t('system.common.reset') }}</el-button>
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
          >{{ t('system.menu.addMenu') }}</el-button
        >
      </div>
      <!-- desktop: tree table -->
      <el-table
        v-if="!menuStore.isMobile"
        v-loading="loading"
        :data="translatedFilteredMenuList"
        :border="TABLE_CONFIG.border"
        row-key="id"
        :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
        default-expand-all
        show-overflow-tooltip
      >
        <el-table-column prop="title" :label="t('system.common.menuTitle')" min-width="200" :align="TABLE_CONFIG.align" />
        <el-table-column prop="path" :label="t('system.common.menuPath')" min-width="250" :align="TABLE_CONFIG.align" />
        <el-table-column prop="type" :label="t('system.common.type')" min-width="100" :align="TABLE_CONFIG.align">
          <template #default="{ row }">
            <BaseTag v-if="row.type === MenuType.MenuType_Directory" type="info" :text="getMenuTypeText(row.type)" />
            <BaseTag v-else-if="row.type === MenuType.MenuType_Menu" type="primary" :text="getMenuTypeText(row.type)" />
            <BaseTag v-else-if="row.type === MenuType.MenuType_Button" type="warning" :text="getMenuTypeText(row.type)" />
          </template>
        </el-table-column>
        <el-table-column prop="icon" :label="t('system.common.icon')" min-width="100" :align="TABLE_CONFIG.align">
          <template #default="{ row }">
            <el-icon v-if="row.icon"><component :is="menuStore.iconComponents[row.icon]" /></el-icon>
          </template>
        </el-table-column>
        <el-table-column prop="order" :label="t('system.common.sort')" min-width="100" :align="TABLE_CONFIG.align" />
        <el-table-column prop="status" :label="t('system.common.status')" min-width="100" :align="TABLE_CONFIG.align">
          <template #default="{ row }">
            <BaseTag :type="row.status === MenuStatus.MenuStatus_Active ? 'success' : 'danger'" :text="getMenuStatusText(row.status)" />
          </template>
        </el-table-column>
        <el-table-column prop="createTime" :label="t('system.common.createTime')" min-width="180" :align="TABLE_CONFIG.align" />
        <el-table-column :label="t('system.common.operation')" width="150" fixed="right" :align="TABLE_CONFIG.align">
          <template #default="{ row }: { row: MenuInfo }">
            <el-button type="primary" link :icon="menuStore.iconComponents.Edit" @click="menuCreateRef?.showDialog(row.id)" v-permission="[PERM.MENU_EDIT]">{{ t('system.common.edit') }}</el-button>
            <AdaptiveConfirm v-if="!isSystemMenu(row)" :title="t('system.menu.confirmDeleteSelected')" :placement="POPCONFIRM_CONFIG.placement" :width="POPCONFIRM_CONFIG.width" @confirm="deleteMenuHandle(row.id)">
              <template #reference>
                <el-button type="danger" link :icon="menuStore.iconComponents.Delete" v-permission="[PERM.MENU_DELETE]">{{ t('system.common.delete') }}</el-button>
              </template>
            </AdaptiveConfirm>
          </template>
        </el-table-column>
      </el-table>

      <!-- mobile: cards with indent for tree -->
      <div v-else v-loading="loading" class="mobile-card-list">
        <div v-if="!translatedFilteredMenuList.length" class="mobile-empty"><el-empty :description="t('system.common.noData')" /></div>
        <template v-for="item in translatedFilteredMenuList" :key="item.id">
          <div class="mobile-card" :style="{ marginLeft: '0px' }">
            <div class="mobile-card-body">
              <div class="mobile-card-header">
                <span class="mobile-card-title">{{ item.title }}</span>
                <BaseTag v-if="item.type === MenuType.MenuType_Directory" type="info" :text="getMenuTypeText(item.type)" />
                <BaseTag v-else-if="item.type === MenuType.MenuType_Menu" type="primary" :text="getMenuTypeText(item.type)" />
                <BaseTag v-else type="warning" :text="getMenuTypeText(item.type)" />
              </div>
              <div class="mobile-card-meta">{{ item.path }}</div>
              <div class="mobile-card-meta">
                <el-icon v-if="item.icon" size="14"><component :is="menuStore.iconComponents[item.icon]" /></el-icon>
                <span>{{ t('system.menu.orderLabel', { order: item.order }) }}</span>
                <BaseTag :type="item.status === MenuStatus.MenuStatus_Active ? 'success' : 'danger'" :text="getMenuStatusText(item.status)" />
              </div>
            </div>
            <div class="mobile-card-actions">
              <el-button size="small" plain type="primary" @click.stop="menuCreateRef?.showDialog(item.id)" v-permission="[PERM.MENU_EDIT]">{{ t('system.common.edit') }}</el-button>
              <AdaptiveConfirm v-if="!isSystemMenu(item)" :title="t('system.menu.confirmDeleteSelected')" :placement="POPCONFIRM_CONFIG.placement" :width="POPCONFIRM_CONFIG.width" @confirm="deleteMenuHandle(item.id)">
                <template #reference>
                  <el-button size="small" plain type="danger" v-permission="[PERM.MENU_DELETE]">{{ t('system.common.delete') }}</el-button>
                </template>
              </AdaptiveConfirm>
            </div>
          </div>
          <template v-if="item.children?.length">
            <div v-for="child in item.children" :key="child.id" class="mobile-card" style="marginLeft: 1.2rem">
              <div class="mobile-card-body">
                <div class="mobile-card-header">
                  <span class="mobile-card-title">{{ child.title }}</span>
                  <BaseTag v-if="child.type === MenuType.MenuType_Directory" type="info" :text="getMenuTypeText(child.type)" />
                  <BaseTag v-else-if="child.type === MenuType.MenuType_Menu" type="primary" :text="getMenuTypeText(child.type)" />
                  <BaseTag v-else type="warning" :text="getMenuTypeText(child.type)" />
                </div>
                <div class="mobile-card-meta">{{ child.path }}</div>
                <div class="mobile-card-meta">
                  <span>{{ t('system.menu.orderLabel', { order: child.order }) }}</span>
                  <BaseTag :type="child.status === MenuStatus.MenuStatus_Active ? 'success' : 'danger'" :text="getMenuStatusText(child.status)" />
                </div>
              </div>
              <div class="mobile-card-actions">
                <el-button size="small" plain type="primary" @click.stop="menuCreateRef?.showDialog(child.id)" v-permission="[PERM.MENU_EDIT]">{{ t('system.common.edit') }}</el-button>
                <AdaptiveConfirm v-if="!isSystemMenu(child)" :title="t('system.menu.confirmDeleteSelected')" :placement="POPCONFIRM_CONFIG.placement" :width="POPCONFIRM_CONFIG.width" @confirm="deleteMenuHandle(child.id)">
                  <template #reference>
                    <el-button size="small" plain type="danger" v-permission="[PERM.MENU_DELETE]">{{ t('system.common.delete') }}</el-button>
                  </template>
                </AdaptiveConfirm>
              </div>
            </div>
          </template>
        </template>
      </div>
    </el-card>

    <MenuCreate ref="menuCreateRef" @refresh="refresh" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { adminPermissionsList, adminDeletePermissions } from '@/api/menu'
import { TABLE_CONFIG, POPCONFIRM_CONFIG } from '@/config/elementConfig'
import { PERM } from '@/config/permission'
import { translateKnownText } from '@/locales'
import MenuCreate from '@/views/system/menu/create.vue'
import { MenuType, MenuStatus } from '@/types/v1/system'
import type { MenuInfo } from '@/types/v1/system'
import type { FormInstance } from 'element-plus'

defineOptions({ name: 'MenuView' })

const menuStore = useMenuStore()
const { t } = useI18n()
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

const translateMenuTree = (menus: MenuInfo[]): MenuInfo[] =>
  menus.map((menu) => ({
    ...menu,
    title: translateKnownText(menu.title),
    children: menu.children?.length ? translateMenuTree(menu.children) : menu.children,
  }))

const translatedFilteredMenuList = computed(() => translateMenuTree(filteredMenuList.value))

const getMenuTypeText = (type: MenuType) => {
  if (type === MenuType.MenuType_Directory) return t('system.common.directory')
  if (type === MenuType.MenuType_Menu) return t('system.common.menu')
  return t('system.common.button')
}

const getMenuStatusText = (status: MenuStatus) => {
  if (status === MenuStatus.MenuStatus_Active) return t('system.common.enabled')
  return t('system.common.disabled')
}

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
  ElMessage.success(t('system.common.deleteSuccess'))
  getMenuList()
}

const refresh = () => {
  getMenuList()
}

onMounted(() => {
  getMenuList()
})
</script>

<style scoped>
.mobile-card-list { display: flex; flex-direction: column; gap: 0.55rem; }
.mobile-empty { padding: 1.5rem 0; }
.mobile-card { display: flex; align-items: flex-start; gap: 0.6rem; padding: 0.65rem 0.75rem; border: 1px solid var(--el-border-color-extra-light); border-radius: 0.6rem; background: var(--el-bg-color); }
.mobile-card-body { flex: 1; min-width: 0; }
.mobile-card-header { display: flex; align-items: center; gap: 0.4rem; }
.mobile-card-title { font-size: 0.85rem; font-weight: 700; color: var(--el-text-color-primary); }
.mobile-card-meta { display: flex; align-items: center; gap: 0.35rem; margin-top: 0.2rem; font-size: 0.72rem; color: var(--el-text-color-secondary); }
.mobile-card-actions { display: flex; flex-direction: column; gap: 0.3rem; flex-shrink: 0; }
</style>
