<template>
  <BaseDialog
    v-model="open"
    :title="isEdit ? t('system.menu.editMenu') : t('system.menu.addMenu')"
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
      <el-form-item :label="t('system.menu.menuType')" prop="type">
        <el-radio-group
          v-model="submitForm.type"
          :disabled="isEdit"
          @change="handleMenuTypeChange"
        >
          <el-radio :label="MenuType.MenuType_Directory">{{ t('system.common.directory') }}</el-radio>
          <el-radio :label="MenuType.MenuType_Menu">{{ t('system.common.menu') }}</el-radio>
          <el-radio :label="MenuType.MenuType_Button">{{ t('system.common.button') }}</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item :label="t('system.menu.parentMenu')" prop="parentId">
        <el-tree-select
          v-model="submitForm.parentId"
          :data="parentMenuList"
          :props="{ label: 'title', value: 'id', children: 'children' }"
          :placeholder="t('system.menu.parentPlaceholder')"
          clearable
          check-strictly
          :disabled="isEdit"
        />
      </el-form-item>
      <el-form-item :label="titleLabel" prop="title">
        <el-input v-model="submitForm.title" :placeholder="t('system.menu.titlePlaceholder', { label: titleLabel })" />
      </el-form-item>
      <el-form-item
        :label="t('system.common.menuPath')"
        prop="path"
        v-if="submitForm.type === MenuType.MenuType_Menu"
      >
        <el-input v-model="submitForm.path" :placeholder="t('system.menu.menuPathPlaceholder')" />
      </el-form-item>
      <el-form-item
        :label="t('system.menu.permissions')"
        prop="permissions"
        v-if="submitForm.type === MenuType.MenuType_Button"
      >
        <el-input
          v-model="submitForm.permissions"
          :placeholder="t('system.menu.permissionsPlaceholder')"
          clearable
        />
      </el-form-item>
      <el-form-item
        :label="t('system.common.icon')"
        prop="icon"
        v-if="submitForm.type !== MenuType.MenuType_Button"
      >
        <div class="icon-selector-wrapper">
          <el-input v-model="submitForm.icon" :placeholder="t('system.menu.iconPlaceholder')" clearable>
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
            <template #default v-if="!menuStore.isMobile">{{ t('system.menu.selectIcon') }}</template>
          </el-button>
        </div>
      </el-form-item>
      <el-form-item :label="t('system.common.sort')" prop="order">
        <el-input-number v-model="submitForm.order" :min="0" :max="999" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('system.common.status')" prop="status">
        <el-radio-group v-model="submitForm.status">
          <el-radio :label="MenuStatus.MenuStatus_Active">{{ t('system.common.enabled') }}</el-radio>
          <el-radio :label="MenuStatus.MenuStatus_InActive">{{ t('system.common.disabled') }}</el-radio>
        </el-radio-group>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="close">{{ t('system.common.cancel') }}</el-button>
      <el-button type="primary" :loading="submitLoading" @click="confirm">{{ t('system.common.confirm') }}</el-button>
    </template>
  </BaseDialog>

  <IconSelectorDialog ref="iconSelectorDialogRef" @selectIcon="getSelectIcon" />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import {
  adminPermissionsList,
  adminPermissionsInfo,
  adminAddPermissions,
  adminEditPermissions,
} from '@/api/menu'
import IconSelectorDialog from '@/components/dialog/IconSelectorDialog.vue'
import { translateKnownText } from '@/locales'
import { MenuType, MenuStatus } from '@/types/v1/system'
import type { MenuInfo, AdminAddPermissionsRequest, AdminEditPermissionsRequest } from '@/types/v1/system'
import type { FormInstance, FormRules } from 'element-plus'

defineOptions({ name: 'MenuCreate' })

const menuStore = useMenuStore()
const emits = defineEmits(['refresh'])
const { t } = useI18n()
const submitFormRef = useTemplateRef<FormInstance>('submitFormRef')
const iconSelectorDialogRef = useTemplateRef<InstanceType<typeof IconSelectorDialog> | null>(
  'iconSelectorDialogRef',
)

const open = ref(false)
const submitLoading = ref(false)
const editingMenuId = ref<string | undefined>(undefined)
const rawParentMenuList = ref<MenuInfo[]>([])
const parentMenuList = computed(() => translateMenuTree(rawParentMenuList.value))

const isEdit = computed(() => !!editingMenuId.value)

const titleLabel = computed(() => {
  if (submitForm.value.type === MenuType.MenuType_Directory) return t('system.menu.directoryTitle')
  if (submitForm.value.type === MenuType.MenuType_Menu) return t('system.menu.menuTitle')
  if (submitForm.value.type === MenuType.MenuType_Button) return t('system.menu.buttonTitle')
  return t('system.menu.menuTitle')
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
  rawParentMenuList.value = []
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

    ElMessage.success(isEdit.value ? t('system.common.editSuccess') : t('system.common.addSuccess'))
    emits('refresh')
    close()
  } finally {
    submitLoading.value = false
  }
}

const loadParentMenuList = async () => {
  const { data: res } = await adminPermissionsList({})
  rawParentMenuList.value = res?.menus || []
}

const translateMenuTree = (menus: MenuInfo[]): MenuInfo[] =>
  menus.map((menu) => ({
    ...menu,
    title: translateKnownText(menu.title),
    children: menu.children?.length ? translateMenuTree(menu.children) : menu.children,
  }))

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
    callback(new Error(t('system.menu.titleRequired', { label: titleLabel.value })))
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
    callback(new Error(t('system.menu.permissionsRequired')))
  } else {
    callback()
  }
}

const rules = computed<FormRules>(() => ({
  type: [{ required: true, message: t('system.menu.typeRequired'), trigger: 'blur' }],
  title: [{ required: true, validator: titleValidator, trigger: 'blur' }],
  path: [{ required: true, message: t('system.menu.pathRequired'), trigger: 'blur' }],
  status: [{ required: true, message: t('system.menu.statusRequired'), trigger: 'blur' }],
  permissions: [{ validator: permissionsValidator, trigger: 'blur' }],
}))

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
