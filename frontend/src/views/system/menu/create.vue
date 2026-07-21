<!-- 菜单创建/编辑弹窗（重写 · P1）：FormDialog 外壳 + 令牌化字段 + 内联校验 + 复用 IconSelectorDialog。
     保留 adminAddPermissions/adminEditPermissions/adminPermissionsInfo 接口与 showDialog/refresh 契约（06d）。
     父级选择改用扁平化原生 <select>（缩进层级）以规避嵌套 EP popper 的层级问题；类型/父级在编辑时禁用（与后端 Edit 契约一致：不改 type/parent）。
     图标选择器用 spacious 模式（内部无子 popper），确保其浮层置于 FormDialog 之上时不会遮挡子级提示。 -->
<template>
  <FormDialog
    v-model="open"
    :title="isEdit ? t('system.menu.editMenu') : t('system.menu.addMenu')"
    :width="600"
    :loading="submitLoading"
    @confirm="confirm"
    @close="close"
  >
    <div class="menu-form">
      <!-- 类型（编辑时禁用） -->
      <div class="app-field">
        <label class="app-label app-label--required">{{ t('system.menu.menuType') }}</label>
        <div class="menu-form__radios">
          <label
            v-for="opt in typeOptions"
            :key="opt.value"
            class="menu-form__radio"
            :class="{ 'is-disabled': isEdit }"
          >
            <input
              v-model="submitForm.type"
              type="radio"
              :value="opt.value"
              :disabled="isEdit"
              @change="onTypeChange"
            />
            <span>{{ opt.label }}</span>
          </label>
        </div>
      </div>

      <!-- 父级菜单（编辑时禁用；扁平缩进） -->
      <div class="app-field">
        <label class="app-label">{{ t('system.menu.parentMenu') }}</label>
        <AppSelect v-model="submitForm.parentId" :options="parentSelectOptions" :disabled="isEdit" />
      </div>

      <!-- 标题（标签随类型变化） -->
      <div class="app-field">
        <label class="app-label app-label--required">{{ titleLabel }}</label>
        <input
          v-model="submitForm.title"
          class="app-input"
          :class="{ 'is-error': errors.title }"
          :placeholder="t('system.menu.titlePlaceholder', { label: titleLabel })"
        />
        <span v-if="errors.title" class="app-field__error">{{ errors.title }}</span>
      </div>

      <!-- 路径（仅菜单类型） -->
      <div v-if="submitForm.type === MenuType.MenuType_Menu" class="app-field">
        <label class="app-label app-label--required">{{ t('system.common.menuPath') }}</label>
        <input
          v-model="submitForm.path"
          class="app-input"
          :class="{ 'is-error': errors.path }"
          :placeholder="t('system.menu.menuPathPlaceholder')"
        />
        <span v-if="errors.path" class="app-field__error">{{ errors.path }}</span>
      </div>

      <!-- 权限标识（仅按钮类型） -->
      <div v-if="submitForm.type === MenuType.MenuType_Button" class="app-field">
        <label class="app-label app-label--required">{{ t('system.menu.permissions') }}</label>
        <input
          v-model="submitForm.permissions"
          class="app-input"
          :class="{ 'is-error': errors.permissions }"
          :placeholder="t('system.menu.permissionsPlaceholder')"
        />
        <span v-if="errors.permissions" class="app-field__error">{{ errors.permissions }}</span>
      </div>

      <!-- 图标（非按钮类型） -->
      <div v-if="submitForm.type !== MenuType.MenuType_Button" class="app-field">
        <label class="app-label">{{ t('system.common.icon') }}</label>
        <div class="menu-form__icon">
          <span class="menu-form__icon-preview">
            <component
              :is="menuStore.iconComponents[submitForm.icon]"
              v-if="submitForm.icon && menuStore.iconComponents[submitForm.icon]"
            />
            <span v-else class="menu-form__icon-dash">—</span>
          </span>
          <input
            v-model="submitForm.icon"
            class="app-input"
            :placeholder="t('system.menu.iconPlaceholder')"
          />
          <UButton color="neutral" variant="soft" icon="i-lucide-search" @click="openIconSelector">
            {{ t('system.menu.selectIcon') }}
          </UButton>
        </div>
      </div>

      <!-- 排序 -->
      <div class="app-field">
        <label class="app-label">{{ t('system.common.sort') }}</label>
        <input v-model.number="submitForm.order" type="number" min="0" max="999" class="app-input" />
      </div>

      <!-- 状态 -->
      <div class="app-field">
        <label class="app-label">{{ t('system.common.status') }}</label>
        <div class="menu-form__radios">
          <label class="menu-form__radio">
            <input v-model="submitForm.status" type="radio" :value="MenuStatus.MenuStatus_Active" />
            <span>{{ t('system.common.enabled') }}</span>
          </label>
          <label class="menu-form__radio">
            <input v-model="submitForm.status" type="radio" :value="MenuStatus.MenuStatus_InActive" />
            <span>{{ t('system.common.disabled') }}</span>
          </label>
        </div>
      </div>
    </div>
  </FormDialog>

  <IconSelectorDialog ref="iconSelectorDialogRef" density="spacious" @selectIcon="onSelectIcon" />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  adminPermissionsList,
  adminPermissionsInfo,
  adminAddPermissions,
  adminEditPermissions,
} from '@/api/menu'
import IconSelectorDialog from '@/components/dialog/IconSelectorDialog.vue'
import { translateKnownText } from '@/locales'
import { MenuType, MenuStatus, type MenuInfo } from '@/types/v1/system'
import type { AdminAddPermissionsRequest, AdminEditPermissionsRequest } from '@/types/v1/system'

defineOptions({ name: 'MenuCreate' })

const menuStore = useMenuStore()
const emits = defineEmits(['refresh'])
const { t } = useI18n()
const iconSelectorDialogRef = useTemplateRef<InstanceType<typeof IconSelectorDialog> | null>(
  'iconSelectorDialogRef',
)

const open = ref(false)
const submitLoading = ref(false)
const editingMenuId = ref<string | undefined>(undefined)
const rawParentMenuList = ref<MenuInfo[]>([])
const errors = ref<Record<string, string>>({})

const isEdit = computed(() => !!editingMenuId.value)
const isButtonType = computed(() => submitForm.value.type === MenuType.MenuType_Button)

const typeOptions = computed(() => [
  { value: MenuType.MenuType_Directory, label: t('system.common.directory') },
  { value: MenuType.MenuType_Menu, label: t('system.common.menu') },
  { value: MenuType.MenuType_Button, label: t('system.common.button') },
])

const titleLabel = computed(() => {
  if (submitForm.value.type === MenuType.MenuType_Directory) return t('system.menu.directoryTitle')
  if (submitForm.value.type === MenuType.MenuType_Button) return t('system.menu.buttonTitle')
  return t('system.menu.menuTitle')
})

const emptyForm = () => ({
  type: MenuType.MenuType_Directory as MenuType,
  parentId: '' as string,
  title: '',
  path: '',
  icon: '',
  permissions: '',
  order: 0,
  status: MenuStatus.MenuStatus_Active as MenuStatus,
})
const submitForm = ref(emptyForm())

// 扁平化父级候选（排除按钮，缩进表示层级）
interface ParentOption {
  id: string
  label: string
}
const flattenParents = (menus: MenuInfo[], depth = 0, acc: ParentOption[] = []): ParentOption[] => {
  for (const m of menus) {
    const isButton = m.type === MenuType.MenuType_Button
    if (!isButton) {
      acc.push({
        id: m.id,
        label: `${'　'.repeat(depth)}${translateKnownText(m.title) || m.title}`,
      })
    }
    if (m.children?.length) flattenParents(m.children, isButton ? depth : depth + 1, acc)
  }
  return acc
}
const parentOptions = computed(() => flattenParents(rawParentMenuList.value))
const parentSelectOptions = computed<{ label: string; value: string }[]>(() => [
  { label: t('system.menu.topLevel'), value: '' },
  ...parentOptions.value.map((o) => ({ label: o.label, value: o.id })),
])

const onTypeChange = () => {
  errors.value = {}
  if (!isButtonType.value) submitForm.value.permissions = ''
  if (isButtonType.value) submitForm.value.icon = ''
}

const openIconSelector = () => iconSelectorDialogRef.value?.showDialog(submitForm.value.icon)
const onSelectIcon = (iconName: string) => {
  submitForm.value.icon = iconName
}

const validate = (): boolean => {
  const e: Record<string, string> = {}
  if (!submitForm.value.title.trim())
    e.title = t('system.menu.titleRequired', { label: titleLabel.value })
  if (submitForm.value.type === MenuType.MenuType_Menu && !submitForm.value.path.trim())
    e.path = t('system.menu.pathRequired')
  if (isButtonType.value && !submitForm.value.permissions.trim())
    e.permissions = t('system.menu.permissionsRequired')
  errors.value = e
  return Object.keys(e).length === 0
}

const close = () => {
  open.value = false
  submitLoading.value = false
  editingMenuId.value = undefined
  rawParentMenuList.value = []
  errors.value = {}
  submitForm.value = emptyForm()
}

const confirm = async () => {
  if (!validate()) return
  submitLoading.value = true
  try {
    const f = submitForm.value
    const order = Number(f.order) || 0
    const path = f.type === MenuType.MenuType_Menu ? f.path.trim() : ''
    const icon = isButtonType.value ? '' : f.icon
    const permissions = isButtonType.value ? f.permissions.trim() : ''
    if (isEdit.value) {
      const payload: AdminEditPermissionsRequest = {
        menuId: editingMenuId.value!,
        title: f.title.trim(),
        icon,
        order,
        path,
        status: f.status,
        permissions,
      }
      await adminEditPermissions(payload)
    } else {
      const payload: AdminAddPermissionsRequest = {
        type: f.type,
        parentId: f.parentId,
        title: f.title.trim(),
        icon,
        order,
        path,
        status: f.status,
        permissions,
      }
      await adminAddPermissions(payload)
    }
    feedback.success(isEdit.value ? t('system.common.editSuccess') : t('system.common.addSuccess'))
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

const loadMenuInfo = async (menuId: string) => {
  const { data: res } = await adminPermissionsInfo({ menuId })
  if (!res?.menu) return
  const { type, title, path, icon, parentId, order, status, permissions } = res.menu
  submitForm.value = {
    type,
    parentId: parentId || '',
    title,
    path: path || '',
    icon: icon || '',
    permissions: permissions || '',
    order: order ?? 0,
    status,
  }
}

const showDialog = async (menuId?: string) => {
  submitForm.value = emptyForm()
  errors.value = {}
  editingMenuId.value = menuId
  open.value = true
  await loadParentMenuList()
  if (menuId) await loadMenuInfo(menuId)
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.menu-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.menu-form__radios {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  min-height: 32px;
  align-items: center;
}
.menu-form__radio {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8125rem;
  color: var(--el-text-color-regular);
  cursor: pointer;
}
.menu-form__radio input {
  accent-color: var(--el-color-primary);
  width: 15px;
  height: 15px;
  cursor: pointer;
}
.menu-form__radio.is-disabled {
  cursor: not-allowed;
  color: var(--el-text-color-placeholder);
}
.menu-form__radio.is-disabled input {
  cursor: not-allowed;
}
.menu-form__icon {
  display: flex;
  align-items: center;
  gap: 8px;
}
.menu-form__icon .app-input {
  flex: 1;
  min-width: 0;
}
.menu-form__icon-preview {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  border: 1px solid var(--el-border-color);
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-lighter);
  color: var(--el-text-color-regular);
}
.menu-form__icon-preview :deep(svg) {
  width: 18px;
  height: 18px;
}
.menu-form__icon-dash {
  color: var(--el-text-color-placeholder);
}
</style>
