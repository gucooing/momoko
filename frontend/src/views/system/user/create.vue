<!-- 用户创建/编辑弹窗（重写）：FormDialog 外壳 + 令牌化字段 + 内联校验。
     保留 rolePage/createUser/updateUser/userInfo 接口与 showDialog/refresh 契约（06d）。 -->
<template>
  <FormDialog
    v-model="open"
    :title="submitForm.userId ? t('system.user.editUser') : t('system.user.addUser')"
    :width="560"
    :loading="submitLoading"
    @confirm="confirm"
    @close="onClose"
  >
    <div class="user-form">
      <div class="app-field">
        <label class="app-label app-label--required">{{ t('system.common.username') }}</label>
        <input
          v-model="submitForm.username"
          class="app-input"
          :class="{ 'is-error': errors.username }"
          :placeholder="t('system.user.usernameCreatePlaceholder')"
          :disabled="!!submitForm.userId"
        />
        <span v-if="errors.username" class="app-field__error">{{ errors.username }}</span>
      </div>

      <div v-if="!submitForm.userId" class="app-field">
        <label class="app-label app-label--required">{{ t('system.common.password') }}</label>
        <input
          v-model="submitForm.password"
          type="password"
          class="app-input"
          :class="{ 'is-error': errors.password }"
          :placeholder="t('system.user.passwordPlaceholder')"
        />
        <span v-if="errors.password" class="app-field__error">{{ errors.password }}</span>
      </div>

      <div class="app-field">
        <label class="app-label" :class="{ 'app-label--required': !submitForm.userId }">
          {{ t('system.common.name') }}
        </label>
        <input
          v-model="submitForm.name"
          class="app-input"
          :class="{ 'is-error': errors.name }"
          :placeholder="t('system.user.namePlaceholder')"
        />
        <span v-if="errors.name" class="app-field__error">{{ errors.name }}</span>
      </div>

      <div class="app-field">
        <label class="app-label" :class="{ 'app-label--required': !submitForm.userId }">
          {{ t('system.common.email') }}
        </label>
        <input
          v-model="submitForm.email"
          class="app-input"
          :class="{ 'is-error': errors.email }"
          :placeholder="t('system.user.emailPlaceholder')"
        />
        <span v-if="errors.email" class="app-field__error">{{ errors.email }}</span>
      </div>

      <div class="app-field">
        <label class="app-label app-label--required">{{ t('system.common.userRole') }}</label>
        <AppSelect
          v-model="submitForm.roleId"
          :options="roleOptions"
          :placeholder="t('system.user.rolePlaceholder')"
          :error="!!errors.roleId"
        />
        <span v-if="errors.roleId" class="app-field__error">{{ errors.roleId }}</span>
      </div>

      <div class="app-field">
        <label class="app-label">{{ t('system.common.status') }}</label>
        <div class="user-form__radios">
          <label class="user-form__radio">
            <input v-model="submitForm.status" type="radio" :value="UserStatus.Active" />
            <span>{{ t('system.common.enabled') }}</span>
          </label>
          <label class="user-form__radio">
            <input v-model="submitForm.status" type="radio" :value="UserStatus.InActive" />
            <span>{{ t('system.common.inactive') }}</span>
          </label>
        </div>
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { rolePage } from '@/api/role'
import { createUser, updateUser, userInfo } from '@/api/user'
import type { RoleInfo } from '@/types/v1/system'
import { UserStatus, type UserInfo } from '@/types/v1/user'

defineOptions({ name: 'UserCreate' })

const emits = defineEmits(['refresh'])
const { t } = useI18n()

const open = ref(false)
const submitLoading = ref(false)
const roleList = ref<RoleInfo[]>([])
const errors = ref<Record<string, string>>({})

const roleOptions = computed<{ label: string; value: string }[]>(() =>
  roleList.value.map((r) => ({ label: r.name, value: r.roleId })),
)

const emptyForm = () => ({
  userId: undefined as string | undefined,
  username: '',
  password: '',
  name: '',
  email: '',
  roleId: '' as string,
  status: UserStatus.Active as UserStatus,
  avatar: '',
  bio: '',
  tags: '',
})
const submitForm = ref(emptyForm())

const onClose = () => {
  errors.value = {}
  roleList.value = []
  submitForm.value = emptyForm()
}

const getRoleList = async () => {
  const { data: res } = await rolePage({ page: 1, pageSize: 1000 })
  roleList.value = res?.roles || []
}

const resolveRoleId = (user: UserInfo, fallbackRoleName?: string): string => {
  if (user.roleId) return user.roleId
  const roleName = user.roleName || fallbackRoleName
  if (!roleName) return ''
  return roleList.value.find((role) => role.name === roleName)?.roleId || ''
}

const getUserDetail = async (userId: string, fallbackRoleName?: string) => {
  const { data: res } = await userInfo({ userId })
  if (!res?.user) return
  const user = res.user
  submitForm.value = {
    userId: user.userId,
    username: user.username,
    password: '',
    name: user.name || '',
    email: user.email || '',
    roleId: resolveRoleId(user, fallbackRoleName),
    status: user.status || UserStatus.Active,
    avatar: user.avatar || '',
    bio: user.bio || '',
    tags: user.tags || '',
  }
}

const validate = (): boolean => {
  const e: Record<string, string> = {}
  const f = submitForm.value
  const creating = !f.userId

  if (!f.username.trim()) e.username = t('system.user.usernameRequired')
  else if (/[一-龥]/.test(f.username)) e.username = t('system.user.usernameNoChinese')

  if (creating && !f.password.trim()) e.password = t('system.user.passwordRequired')
  if (creating && !f.name.trim()) e.name = t('system.user.nameRequired')

  if (creating && !f.email.trim()) e.email = t('system.user.emailRequired')
  else if (f.email.trim() && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(f.email))
    e.email = t('system.user.emailInvalid')

  if (!f.roleId) e.roleId = t('system.user.roleRequired')

  errors.value = e
  return Object.keys(e).length === 0
}

const confirm = async () => {
  if (!validate()) return

  submitLoading.value = true
  try {
    const f = submitForm.value
    if (f.userId) {
      await updateUser({
        userId: f.userId,
        email: f.email,
        name: f.name,
        avatar: f.avatar,
        bio: f.bio,
        tags: f.tags,
        status: f.status,
        roleId: f.roleId,
      })
    } else {
      await createUser({
        username: f.username,
        password: f.password,
        email: f.email,
        name: f.name,
        avatar: f.avatar,
        bio: f.bio,
        tags: f.tags,
        status: f.status,
        roleId: f.roleId,
      })
    }

    feedback.success(f.userId ? t('system.common.editSuccess') : t('system.common.addSuccess'))
    emits('refresh', f.userId ? 'update' : 'create')
    open.value = false
    onClose()
  } finally {
    submitLoading.value = false
  }
}

const showDialog = async (payload?: { userId: string; roleName?: string }) => {
  submitForm.value = emptyForm()
  errors.value = {}
  open.value = true
  await getRoleList()

  if (!payload?.userId) return
  submitForm.value.userId = payload.userId
  await getUserDetail(payload.userId, payload.roleName)
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.user-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.user-form__radios {
  display: flex;
  gap: 20px;
  min-height: 32px;
  align-items: center;
}
.user-form__radio {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8125rem;
  color: var(--el-text-color-regular);
  cursor: pointer;
}
.user-form__radio input {
  accent-color: var(--el-color-primary);
  width: 15px;
  height: 15px;
  cursor: pointer;
}
</style>
