<!-- 用户创建/编辑弹窗：FormDialog 外壳 + 令牌化字段 + 内联校验；PC 两列、手机单列；角色用卡片式多选（权限取并集，
     选「无权限」会清掉其它角色，选其它角色会去掉「无权限」）。
     内置超级管理员的角色与状态锁定。保留 rolePage/createUser/updateUser/userInfo 接口与 showDialog/refresh 契约。 -->
<template>
  <FormDialog
    v-model="open"
    :title="submitForm.userId ? t('system.user.editUser') : t('system.user.addUser')"
    :width="640"
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

      <div class="app-field user-form__full">
        <label class="app-label app-label--required">{{ t('system.common.userRole') }}</label>
        <div class="user-form__roles" role="group" :aria-label="t('system.common.userRole')">
          <button
            v-for="opt in roleOptions"
            :key="opt.value"
            type="button"
            role="checkbox"
            class="user-form__role"
            :class="{ 'is-on': submitForm.roleIds.includes(opt.value) }"
            :aria-checked="submitForm.roleIds.includes(opt.value)"
            :disabled="locked"
            @click="toggleRole(opt.value)"
          >
            <span class="user-form__role-name">
              <span class="user-form__role-check" aria-hidden="true">
                <component :is="menuStore.iconComponents['HOutline:CheckIcon']" />
              </span>
              {{ opt.label }}
            </span>
            <span class="user-form__role-desc">{{ opt.description || '—' }}</span>
          </button>
        </div>
        <span v-if="errors.roleIds" class="app-field__error">{{ errors.roleIds }}</span>
        <span v-else class="user-form__help">{{ t('accessV2.user.multiRoleHint') }}</span>
      </div>

      <div class="app-field user-form__full">
        <label class="app-label">{{ t('system.common.status') }}</label>
        <div class="user-form__status">
          <AppSwitch
            :model-value="submitForm.status === UserStatus.Active"
            :disabled="locked"
            @update:model-value="(v) => (submitForm.status = v ? UserStatus.Active : UserStatus.InActive)"
          />
          <span>{{ submitForm.status === UserStatus.Active ? t('accessAdmin.user.chipActive') : t('accessAdmin.user.chipInactive') }}</span>
        </div>
      </div>

      <p v-if="locked" class="user-form__lock user-form__full">
        <component :is="menuStore.iconComponents['HOutline:LockClosedIcon']" />
        {{ t('accessAdmin.user.roleLockedHint') }}
      </p>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { rolePage } from '@/api/role'
import { createUser, updateUser, userInfo } from '@/api/user'
import type { RoleInfo } from '@/types/v1/system'
import { UserStatus } from '@/types/v1/user'
import { NO_PERMISSION_ROLE_ID } from '@/views/system/components/accessTone'

defineOptions({ name: 'UserCreate' })

const emits = defineEmits(['refresh'])
const { t } = useI18n()
const menuStore = useMenuStore()
const locked = ref(false)

const open = ref(false)
const submitLoading = ref(false)
const roleList = ref<RoleInfo[]>([])
const errors = ref<Record<string, string>>({})

const roleOptions = computed<{ label: string; value: string; description: string }[]>(() =>
  roleList.value.map((r) => ({ label: r.name, value: r.roleId, description: r.description })),
)

const emptyForm = () => ({
  userId: undefined as string | undefined,
  username: '',
  password: '',
  name: '',
  email: '',
  roleIds: [] as string[],
  status: UserStatus.Active as UserStatus,
  avatar: '',
  bio: '',
  tags: '',
})
const submitForm = ref(emptyForm())

const onClose = () => {
  locked.value = false
  errors.value = {}
  roleList.value = []
  submitForm.value = emptyForm()
}

const getRoleList = async () => {
  const { data: res } = await rolePage({ page: 1, pageSize: 1000 })
  roleList.value = res?.roles || []
}

// 「无权限」与其它角色互斥：选它就只留它，选其它角色就去掉它
const toggleRole = (roleId: string) => {
  if (locked.value) return
  const cur = submitForm.value.roleIds
  if (cur.includes(roleId)) {
    submitForm.value.roleIds = cur.filter((id) => id !== roleId)
    return
  }
  submitForm.value.roleIds =
    roleId === NO_PERMISSION_ROLE_ID ? [roleId] : [...cur.filter((id) => id !== NO_PERMISSION_ROLE_ID), roleId]
}

const getUserDetail = async (userId: string) => {
  const { data: res } = await userInfo({ userId })
  if (!res?.user) return
  const user = res.user
  submitForm.value = {
    userId: user.userId,
    username: user.username,
    password: '',
    name: user.name || '',
    email: user.email || '',
    roleIds: (user.roles || []).map((r) => r.roleId),
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

  if (!f.roleIds.length) e.roleIds = t('system.user.roleRequired')

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
        roleIds: f.roleIds,
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
        roleIds: f.roleIds,
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

const showDialog = async (payload?: { userId: string; isBuiltin?: boolean }) => {
  submitForm.value = emptyForm()
  errors.value = {}
  locked.value = !!payload?.isBuiltin
  open.value = true
  await getRoleList()

  if (!payload?.userId) return
  submitForm.value.userId = payload.userId
  await getUserDetail(payload.userId)
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.user-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px 16px;
}
.user-form__full {
  grid-column: 1 / -1;
}
.user-form__roles {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 8px;
}
.user-form__role {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  min-height: 56px;
  padding: 8px 12px;
  border: 1.5px solid var(--el-border-color-light);
  border-radius: var(--app-radius);
  background: var(--el-bg-color);
  text-align: left;
  font: inherit;
  cursor: pointer;
  min-width: 0;
}
.user-form__role:hover:not(:disabled) {
  border-color: var(--m-p);
}
.user-form__role.is-on {
  border-color: var(--m-p);
  background: var(--m-pSoft);
}
.user-form__role:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}
.user-form__role-name {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 0.875rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.user-form__role-check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  flex: none;
  border-radius: 6px;
  border: 1.5px solid var(--el-border-color);
  color: transparent;
}
.user-form__role-check :deep(svg) {
  width: 12px;
  height: 12px;
  stroke-width: 3;
}
.user-form__role.is-on .user-form__role-check {
  border-color: var(--m-p);
  background: var(--m-p);
  color: var(--m-pOn);
}
.user-form__help {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.user-form__role-desc {
  max-width: 100%;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.user-form__status {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 40px;
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--el-text-color-regular);
}
.user-form__lock {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  padding: 10px 12px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.user-form__lock :deep(svg) {
  width: 16px;
  height: 16px;
  flex: none;
}
@media (width < 640px) {
  .user-form {
    grid-template-columns: 1fr;
  }
}
</style>
