<template>
  <BaseDialog
    v-model="open"
    :title="isEdit ? '编辑菜单' : '新增菜单'"
    width="600"
    @close="close"
  >
    <el-form
      ref="submitFormRef"
      :model="submitForm"
      :rules="rules"
      label-width="100px"
      label-position="right"
    >
      <el-form-item label="菜单类型" prop="type">
        <el-radio-group
          v-model="submitForm.type"
          :disabled="isEdit"
          @change="handleMenuTypeChange"
        >
          <el-radio :label="MenuType.MenuType_Directory">目录</el-radio>
          <el-radio :label="MenuType.MenuType_Menu">菜单</el-radio>
          <el-radio :label="MenuType.MenuType_Button">按钮</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="父级菜单" prop="parentId">
        <el-tree-select
          v-model="submitForm.parentId"
          :data="parentMenuList"
          :props="{ label: 'title', value: 'id', children: 'children' }"
          placeholder="请选择父菜单（不选则为顶级菜单）"
          clearable
          check-strictly
          :disabled="isEdit"
        />
      </el-form-item>
      <el-form-item :label="titleLabel" prop="title">
        <el-input v-model="submitForm.title" :placeholder="`请输入${titleLabel}`" />
      </el-form-item>
      <el-form-item
        label="菜单路径"
        prop="path"
        v-if="submitForm.type === MenuType.MenuType_Menu"
      >
        <el-input v-model="submitForm.path" placeholder="请输入菜单路径" />
      </el-form-item>
      <el-form-item
        label="权限标识"
        prop="permissions"
        v-if="submitForm.type === MenuType.MenuType_Button"
      >
        <el-input
          v-model="submitForm.permissions"
          placeholder="请输入权限标识，例如 user:add"
          clearable
        />
      </el-form-item>
      <el-form-item
        label="图标"
        prop="icon"
        v-if="submitForm.type !== MenuType.MenuType_Button"
      >
        <div class="icon-selector-wrapper">
          <el-input v-model="submitForm.icon" placeholder="请选择图标或输入图标名称" clearable>
            <template #prefix>
              <el-icon v-if="submitForm.icon && menuStore.iconComponents[submitForm.icon]">
                <component :is="menuStore.iconComponents[submitForm.icon]" />
              </el-icon>
            </template>
          </el-input>
          <el-button
            :icon="menuStore.iconComponents['Element:Search']"
            @click="iconSelectorDialogRef?.showDialog(submitForm.icon)"
          >
            <template #default v-if="!menuStore.isMobile">选择图标</template>
          </el-button>
        </div>
      </el-form-item>
      <el-form-item label="排序" prop="order">
        <el-input-number v-model="submitForm.order" :min="0" :max="999" style="width: 100%" />
      </el-form-item>
      <el-form-item label="状态" prop="status">
        <el-radio-group v-model="submitForm.status">
          <el-radio :label="MenuStatus.MenuStatus_Active">启用</el-radio>
          <el-radio :label="MenuStatus.MenuStatus_InActive">禁用</el-radio>
        </el-radio-group>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button type="primary" :loading="submitLoading" @click="confirm">确定</el-button>
    </template>
  </BaseDialog>

  <IconSelectorDialog ref="iconSelectorDialogRef" @selectIcon="getSelectIcon" />
</template>

<script setup lang="ts">
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import {
  adminPermissionsList,
  adminPermissionsInfo,
  adminAddPermissions,
  adminEditPermissions,
} from '@/api/menu'
import IconSelectorDialog from '@/components/dialog/IconSelectorDialog.vue'
import { MenuType, MenuStatus } from '@/types/v1/system'
import type { MenuInfo, AdminAddPermissionsRequest, AdminEditPermissionsRequest } from '@/types/v1/system'
import type { FormInstance, FormRules } from 'element-plus'

defineOptions({ name: 'MenuCreate' })

const menuStore = useMenuStore()
const emits = defineEmits(['refresh'])
const submitFormRef = useTemplateRef<FormInstance>('submitFormRef')
const iconSelectorDialogRef = useTemplateRef<InstanceType<typeof IconSelectorDialog> | null>(
  'iconSelectorDialogRef',
)

const open = ref(false)
const submitLoading = ref(false)
const editingMenuId = ref<string | undefined>(undefined)
const parentMenuList = ref<MenuInfo[]>([])

const isEdit = computed(() => !!editingMenuId.value)

const titleLabel = computed(() => {
  if (submitForm.value.type === MenuType.MenuType_Directory) return '目录标题'
  if (submitForm.value.type === MenuType.MenuType_Menu) return '菜单标题'
  if (submitForm.value.type === MenuType.MenuType_Button) return '按钮标题'
  return '菜单标题'
})

const getDefaultForm = () => ({
  type: MenuType.MenuType_Directory as MenuType,
  title: '',
  path: '',
  icon: '',
  permissions: '',
  parentId: '' as string,
  order: 0,
  status: MenuStatus.MenuStatus_Active as MenuStatus,
})

const submitForm = ref(getDefaultForm())

const close = () => {
  open.value = false
  submitFormRef.value?.resetFields()
  submitLoading.value = false
  editingMenuId.value = undefined
  parentMenuList.value = []
  submitForm.value = getDefaultForm()
}

const isButtonType = computed(() => submitForm.value.type === MenuType.MenuType_Button)

const handleMenuTypeChange = () => {
  submitFormRef.value?.clearValidate()
  if (!isButtonType.value) {
    submitForm.value.permissions = ''
  }
}

const confirm = async () => {
  await submitFormRef.value?.validate()
  submitLoading.value = true

  try {
    if (isEdit.value) {
      const payload: AdminEditPermissionsRequest = {
        menuId: editingMenuId.value!,
        title: submitForm.value.title,
        icon: submitForm.value.icon,
        order: submitForm.value.order,
        path: submitForm.value.path,
        status: submitForm.value.status,
        permissions: isButtonType.value ? submitForm.value.permissions.trim() : '',
      }
      await adminEditPermissions(payload)
    } else {
      const payload: AdminAddPermissionsRequest = {
        type: submitForm.value.type,
        parentId: submitForm.value.parentId,
        title: submitForm.value.title,
        icon: submitForm.value.icon,
        order: submitForm.value.order,
        path: submitForm.value.path,
        status: submitForm.value.status,
        permissions: isButtonType.value ? submitForm.value.permissions.trim() : '',
      }
      await adminAddPermissions(payload)
    }

    ElMessage.success(isEdit.value ? '编辑成功' : '新增成功')
    emits('refresh')
    close()
  } finally {
    submitLoading.value = false
  }
}

const loadParentMenuList = async () => {
  const { data: res } = await adminPermissionsList({})
  parentMenuList.value = res?.menus || []
}

const getSelectIcon = (iconName: string) => {
  submitForm.value.icon = iconName
}

const loadMenuInfo = async (menuId: string) => {
  const { data: res } = await adminPermissionsInfo({ menuId })
  if (!res?.menu) return
  const { type, title, path, icon, parentId, order, status, permissions } = res.menu
  submitForm.value = { type, title, path, icon, parentId, order, status, permissions: permissions || '' }
}

const showDialog = async (menuId: string | undefined) => {
  editingMenuId.value = menuId
  open.value = true
  await loadParentMenuList()
  if (menuId) await loadMenuInfo(menuId)
}

const titleValidator = (
  _rule: unknown,
  value: string,
  callback: (error?: string | Error | undefined) => void,
) => {
  if (value === '') {
    callback(new Error(`请输入${titleLabel.value}`))
  } else {
    callback()
  }
}

const permissionsValidator = (
  _rule: unknown,
  value: string,
  callback: (error?: string | Error | undefined) => void,
) => {
  if (isButtonType.value && !String(value || '').trim()) {
    callback(new Error('请输入权限标识'))
  } else {
    callback()
  }
}

const rules: FormRules = {
  type: [{ required: true, message: '请选择菜单类型', trigger: 'blur' }],
  title: [{ required: true, validator: titleValidator, trigger: 'blur' }],
  path: [{ required: true, message: '请输入菜单路径', trigger: 'blur' }],
  status: [{ required: true, message: '请选择状态', trigger: 'blur' }],
  permissions: [{ validator: permissionsValidator, trigger: 'blur' }],
}

defineExpose({
  showDialog,
})
</script>

<style scoped lang="scss">
.icon-selector-wrapper {
  display: flex;
  gap: 8px;
  align-items: center;
  width: 100%;

  .el-input {
    flex: 1;
  }
}
</style>
