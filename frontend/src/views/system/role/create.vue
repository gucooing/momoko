<template>
  <BaseDialog
    v-model="open"
    :title="submitForm.roleId ? t('system.role.editRole') : t('system.role.addRole')"
    width="600"
    @close="close"
    style="height: 60vh"
  >
    <el-scrollbar>
      <el-form
        ref="submitFormRef"
        :model="submitForm"
        :rules="formRules"
        label-width="100px"
        label-position="right"
      >
        <el-form-item :label="t('system.common.roleName')" prop="name">
          <el-input v-model="submitForm.name" :placeholder="t('system.role.namePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('system.common.roleDescription')" prop="description">
          <el-input
            v-model="submitForm.description"
            type="textarea"
            :rows="3"
            :placeholder="t('system.role.descriptionPlaceholder')"
          />
        </el-form-item>
        <el-form-item :label="t('system.common.status')" prop="status">
          <el-radio-group v-model="submitForm.status">
            <el-radio :label="RoleStatus.RoleStatus_Active">{{ t('system.common.enabled') }}</el-radio>
            <el-radio :label="RoleStatus.RoleStatus_InActive">{{ t('system.common.inactive') }}</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="t('system.role.menuPermission')" prop="menuIds">
          <el-tree
            ref="menuTreeRef"
            :data="menuList"
            :props="{ label: 'title', children: 'children' }"
            show-checkbox
            default-expand-all
            node-key="id"
            @check="handleMenuCheck as unknown"
            style="width: 100%"
          />
        </el-form-item>
      </el-form>
    </el-scrollbar>

    <template #footer>
      <el-button @click="close">{{ t('system.common.cancel') }}</el-button>
      <el-button type="primary" :loading="submitLoading" @click="confirm">{{ t('system.common.confirm') }}</el-button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import { createRole, roleInfo, updateRole } from '@/api/role'
import { adminPermissionsList } from '@/api/menu'
import { translateKnownText } from '@/locales'
import { RoleStatus, type MenuInfo } from '@/types/v1/system'
import { type ElTree, type FormInstance, type FormRules } from 'element-plus'

defineOptions({ name: 'RoleCreate' })

const emits = defineEmits(['refresh'])
const { t } = useI18n()

const submitFormRef = useTemplateRef<FormInstance>('submitFormRef')
const menuTreeRef = useTemplateRef<InstanceType<typeof ElTree> | null>('menuTreeRef')

const open = ref(false)
const submitLoading = ref(false)
const rawMenuList = ref<MenuInfo[]>([])
const menuList = computed(() => translateMenuTree(rawMenuList.value))

const getDefaultForm = () => ({
  roleId: undefined as string | undefined,
  name: '',
  description: '',
  status: RoleStatus.RoleStatus_Active as RoleStatus,
  menuIds: [] as string[],
})

const submitForm = ref(getDefaultForm())

const normalizeMenuIds = (menuIds: Array<string | number>) => {
  return [...new Set(menuIds.map((id) => String(id)))]
}

const getTreeSelectedMenuIds = () => {
  if (!menuTreeRef.value) return submitForm.value.menuIds

  return normalizeMenuIds([
    ...(menuTreeRef.value.getCheckedKeys(false) as Array<string | number>),
    ...(menuTreeRef.value.getHalfCheckedKeys() as Array<string | number>),
  ])
}

const getDeepestCheckedKeys = (menus: MenuInfo[], selectedMenuIds: string[]) => {
  const selectedIdSet = new Set(selectedMenuIds)
  const checkedKeys: string[] = []

  const visit = (menu: MenuInfo): boolean => {
    let hasSelectedDescendant = false
    for (const child of (menu.children || [])) {
      if (visit(child)) hasSelectedDescendant = true
    }
    const isSelected = selectedIdSet.has(menu.id)

    // Restore the tree from the deepest selected nodes so parent nodes can stay half-checked.
    if (isSelected && !hasSelectedDescendant) {
      checkedKeys.push(menu.id)
    }

    return isSelected || hasSelectedDescendant
  }

  menus.forEach(visit)

  return checkedKeys
}

const close = () => {
  open.value = false
  submitLoading.value = false
  menuTreeRef.value?.setCheckedKeys([])
  submitFormRef.value?.resetFields()
  rawMenuList.value = []
  submitForm.value = getDefaultForm()
}

const confirm = async () => {
  submitForm.value.menuIds = getTreeSelectedMenuIds()
  await submitFormRef.value?.validate()
  submitLoading.value = true

  try {
    const { name, description, status, menuIds, roleId } = submitForm.value

    if (roleId) {
      await updateRole({ roleId, name, description, status, menuIds })
    } else {
      await createRole({ name, description, status, menuIds })
    }

    ElMessage.success(roleId ? t('system.common.editSuccess') : t('system.common.addSuccess'))
    emits('refresh', roleId ? 'update' : 'create')
    close()
  } finally {
    submitLoading.value = false
  }
}

const getMenuList = async () => {
  const { data: res } = await adminPermissionsList({})
  rawMenuList.value = res?.menus || []
}

const getRoleInfo = async () => {
  if (!submitForm.value.roleId) return
  const { data: res } = await roleInfo({ roleId: submitForm.value.roleId })
  if (!res?.role) return

  const { roleId, name, description, status, menuIds } = res.role
  submitForm.value = {
    roleId,
    name,
    description,
    status,
    menuIds: menuIds || [],
  }

  await nextTick()
  if (!menuTreeRef.value || menuList.value.length === 0) return

  menuTreeRef.value.setCheckedKeys(
    getDeepestCheckedKeys(menuList.value, menuIds || []),
  )
}

const handleMenuCheck = (
  _data: MenuInfo,
  checked: { checkedKeys: string[]; halfCheckedKeys: string[] },
) => {
  submitForm.value.menuIds = normalizeMenuIds([
    ...checked.checkedKeys,
    ...checked.halfCheckedKeys,
  ])
}

const translateMenuTree = (menus: MenuInfo[]): MenuInfo[] =>
  menus.map((menu) => ({
    ...menu,
    title: translateKnownText(menu.title),
    children: menu.children?.length ? translateMenuTree(menu.children) : menu.children,
  }))

const formRules = computed<FormRules>(() => ({
  name: [{ required: true, message: t('system.role.nameRequired'), trigger: 'blur' }],
  status: [{ required: true, message: t('system.role.statusRequired'), trigger: 'change' }],
}))

const showDialog = async (roleId: string | undefined) => {
  submitForm.value.roleId = roleId
  submitForm.value.menuIds = []
  open.value = true
  await getMenuList()
  if (roleId) await getRoleInfo()
}

defineExpose({
  showDialog,
})
</script>

<style></style>
