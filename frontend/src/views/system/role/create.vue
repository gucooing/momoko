<!-- 角色创建/编辑弹窗（重写 · P1）：FormDialog 外壳 + 令牌化字段 + 内联校验 + 自建 PermissionTree。
     保留 createRole/updateRole/roleInfo/adminPermissionsList 接口与 showDialog/refresh 契约（06d）。 -->
<template>
  <FormDialog
    v-model="open"
    :title="submitForm.roleId ? t('system.role.editRole') : t('system.role.addRole')"
    :width="600"
    :loading="submitLoading"
    @confirm="confirm"
    @close="close"
  >
    <div class="role-form">
      <div class="app-field">
        <label class="app-label app-label--required">{{ t('system.common.roleName') }}</label>
        <input
          v-model="submitForm.name"
          class="app-input"
          :class="{ 'is-error': errors.name }"
          :placeholder="t('system.role.namePlaceholder')"
        />
        <span v-if="errors.name" class="app-field__error">{{ errors.name }}</span>
      </div>

      <div class="app-field">
        <label class="app-label app-label--required">{{ t('system.common.roleDescription') }}</label>
        <textarea
          v-model="submitForm.description"
          class="app-textarea"
          :class="{ 'is-error': errors.description }"
          rows="3"
          :placeholder="t('system.role.descriptionPlaceholder')"
        />
        <span v-if="errors.description" class="app-field__error">{{ errors.description }}</span>
      </div>

      <div class="app-field">
        <label class="app-label">{{ t('system.common.status') }}</label>
        <div class="role-form__radios">
          <label class="role-form__radio">
            <input v-model="submitForm.status" type="radio" :value="RoleStatus.RoleStatus_Active" />
            <span>{{ t('system.common.enabled') }}</span>
          </label>
          <label class="role-form__radio">
            <input v-model="submitForm.status" type="radio" :value="RoleStatus.RoleStatus_InActive" />
            <span>{{ t('system.common.inactive') }}</span>
          </label>
        </div>
      </div>

      <div class="app-field">
        <label class="app-label">{{ t('system.role.menuPermission') }}</label>
        <PermissionTree v-model="submitForm.menuIds" :nodes="menuNodes" />
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { createRole, roleInfo, updateRole } from '@/api/role'
import { adminPermissionsList } from '@/api/menu'
import { translateKnownText } from '@/locales'
import { RoleStatus, type MenuInfo } from '@/types/v1/system'
import type { PermNode } from '@/components/ui/PermissionTree.vue'

defineOptions({ name: 'RoleCreate' })

const emits = defineEmits(['refresh'])
const { t } = useI18n()

const open = ref(false)
const submitLoading = ref(false)
const rawMenuList = ref<MenuInfo[]>([])
const errors = ref<Record<string, string>>({})

const emptyForm = () => ({
  roleId: undefined as string | undefined,
  name: '',
  description: '',
  status: RoleStatus.RoleStatus_Active as RoleStatus,
  menuIds: [] as string[],
})
const submitForm = ref(emptyForm())

// 菜单树 → PermissionTree 节点（翻译标题，保留层级）
const toNodes = (menus: MenuInfo[]): PermNode[] =>
  menus.map((m) => ({
    id: m.id,
    title: translateKnownText(m.title) || m.title,
    children: m.children?.length ? toNodes(m.children) : undefined,
  }))
const menuNodes = computed(() => toNodes(rawMenuList.value))

const close = () => {
  open.value = false
  submitLoading.value = false
  errors.value = {}
  rawMenuList.value = []
  submitForm.value = emptyForm()
}

const validate = (): boolean => {
  const e: Record<string, string> = {}
  if (!submitForm.value.name.trim()) e.name = t('system.role.nameRequired')
  if (!submitForm.value.description.trim()) e.description = t('system.role.descriptionRequired')
  errors.value = e
  return Object.keys(e).length === 0
}

const confirm = async () => {
  if (!validate()) return
  submitLoading.value = true
  try {
    const { name, description, status, menuIds, roleId } = submitForm.value
    if (roleId) {
      await updateRole({ roleId, name, description, status, menuIds })
    } else {
      await createRole({ name, description, status, menuIds })
    }
    feedback.success(roleId ? t('system.common.editSuccess') : t('system.common.addSuccess'))
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

const getRoleInfo = async (roleId: string) => {
  const { data: res } = await roleInfo({ roleId })
  if (!res?.role) return
  const { name, description, status, menuIds } = res.role
  submitForm.value = { roleId, name, description, status, menuIds: menuIds || [] }
}

const showDialog = async (roleId?: string) => {
  submitForm.value = emptyForm()
  submitForm.value.roleId = roleId
  errors.value = {}
  open.value = true
  await getMenuList()
  if (roleId) await getRoleInfo(roleId)
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.role-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.role-form__radios {
  display: flex;
  gap: 20px;
  min-height: 32px;
  align-items: center;
}
.role-form__radio {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8125rem;
  color: var(--el-text-color-regular);
  cursor: pointer;
}
.role-form__radio input {
  accent-color: var(--el-color-primary);
  width: 15px;
  height: 15px;
  cursor: pointer;
}
</style>
