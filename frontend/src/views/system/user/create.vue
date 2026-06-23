<template>
  <BaseDialog
    v-model="open"
    :title="submitForm.userId ? t('system.user.editUser') : t('system.user.addUser')"
    width="600"
    @close="close"
  >
    <el-form
      ref="submitFormRef"
      :model="submitForm"
      :rules="formRules"
      label-width="100px"
      label-position="right"
    >
      <el-form-item :label="t('system.common.username')" prop="username">
        <el-input
          v-model="submitForm.username"
          :placeholder="t('system.user.usernameCreatePlaceholder')"
          :disabled="!!submitForm.userId"
        />
      </el-form-item>

      <el-form-item v-if="!submitForm.userId" :label="t('system.common.password')" prop="password" :required="true">
        <el-input
          v-model="submitForm.password"
          type="password"
          :placeholder="t('system.user.passwordPlaceholder')"
          show-password
        />
      </el-form-item>

      <el-form-item :label="t('system.common.name')" prop="name" :required="!submitForm.userId">
        <el-input v-model="submitForm.name" :placeholder="t('system.user.namePlaceholder')" />
      </el-form-item>

      <el-form-item :label="t('system.common.email')" prop="email" :required="!submitForm.userId">
        <el-input v-model="submitForm.email" :placeholder="t('system.user.emailPlaceholder')" />
      </el-form-item>

      <el-form-item :label="t('system.common.userRole')" prop="roleId">
        <el-select v-model="submitForm.roleId" :placeholder="t('system.user.rolePlaceholder')" style="width: 100%">
          <el-option
            v-for="role in roleList"
            :key="role.roleId"
            :label="role.name"
            :value="role.roleId"
          />
        </el-select>
      </el-form-item>

      <el-form-item :label="t('system.common.status')" prop="status">
        <el-radio-group v-model="submitForm.status">
          <el-radio :label="UserStatus.Active">{{ t('system.common.enabled') }}</el-radio>
          <el-radio :label="UserStatus.InActive">{{ t('system.common.inactive') }}</el-radio>
        </el-radio-group>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="close">{{ t('system.common.cancel') }}</el-button>
      <el-button type="primary" :loading="submitLoading" @click="confirm">{{ t('system.common.confirm') }}</el-button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import { rolePage } from '@/api/role'
import { createUser, updateUser, userInfo } from '@/api/user'
import type { RoleInfo } from '@/types/v1/system'
import { UserStatus, type UserInfo } from '@/types/v1/user'
import { type FormInstance, type FormRules } from 'element-plus'

defineOptions({ name: 'UserCreate' })

const emits = defineEmits(['refresh'])
const { t } = useI18n()
const submitFormRef = useTemplateRef<FormInstance>('submitFormRef')

const open = ref(false)
const submitLoading = ref(false)
const roleList = ref<RoleInfo[]>([])

const submitForm = ref({
  userId: undefined as string | undefined,
  username: '',
  password: '',
  name: '',
  email: '',
  roleId: undefined as string | undefined,
  status: UserStatus.Active as UserStatus,
  avatar: '',
  bio: '',
  tags: '',
})

const close = () => {
  open.value = false
  submitFormRef.value?.resetFields()
  roleList.value = []
  submitForm.value = {
    userId: undefined,
    username: '',
    password: '',
    name: '',
    email: '',
    roleId: undefined,
    status: UserStatus.Active,
    avatar: '',
    bio: '',
    tags: '',
  }
}

const getRoleList = async () => {
  const { data: res } = await rolePage({
    page: 1,
    pageSize: 1000,
  })
  roleList.value = res?.roles || []
}

const resolveRoleId = (user: UserInfo, fallbackRoleName?: string): string | undefined => {
  const roleIdFromPayload = (user as UserInfo & { roleId?: string }).roleId
  if (roleIdFromPayload) return roleIdFromPayload

  const roleName = user.roleName || fallbackRoleName
  if (!roleName) return undefined

  return roleList.value.find((role) => role.name === roleName)?.roleId
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

const confirm = async () => {
  await submitFormRef.value?.validate()

  submitLoading.value = true
  try {
    if (submitForm.value.userId) {
      await updateUser({
        userId: submitForm.value.userId,
        email: submitForm.value.email,
        name: submitForm.value.name,
        avatar: submitForm.value.avatar,
        bio: submitForm.value.bio,
        tags: submitForm.value.tags,
        status: submitForm.value.status,
        roleId: submitForm.value.roleId || '',
      })
    } else {
      await createUser({
        username: submitForm.value.username,
        password: submitForm.value.password,
        email: submitForm.value.email,
        name: submitForm.value.name,
        avatar: submitForm.value.avatar,
        bio: submitForm.value.bio,
        tags: submitForm.value.tags,
        status: submitForm.value.status,
        roleId: submitForm.value.roleId || '',
      })
    }

    ElMessage.success(
      submitForm.value.userId ? t('system.common.editSuccess') : t('system.common.addSuccess'),
    )
    emits('refresh', submitForm.value.userId ? 'update' : 'create')
    close()
  } finally {
    submitLoading.value = false
  }
}

const formRules = computed<FormRules>(() => ({
  username: [
    { required: true, message: t('system.user.usernameRequired'), trigger: 'blur' },
    {
      pattern: /^[^\u4e00-\u9fa5]+$/,
      message: t('system.user.usernameNoChinese'),
      trigger: 'blur',
    },
  ],
  password: [
    {
      validator: (_, value: string, callback) => {
        if (!submitForm.value.userId && !value?.trim()) {
          callback(new Error(t('system.user.passwordRequired')))
          return
        }
        callback()
      },
      trigger: 'blur',
    },
  ],
  name: [
    {
      validator: (_, value: string, callback) => {
        if (!submitForm.value.userId && !value?.trim()) {
          callback(new Error(t('system.user.nameRequired')))
          return
        }
        callback()
      },
      trigger: 'blur',
    },
  ],
  email: [
    {
      validator: (_, value: string, callback) => {
        if (!submitForm.value.userId && !value?.trim()) {
          callback(new Error(t('system.user.emailRequired')))
          return
        }
        if (value?.trim() && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
          callback(new Error(t('system.user.emailInvalid')))
          return
        }
        callback()
      },
      trigger: 'blur',
    },
  ],
  roleId: [{ required: true, message: t('system.user.roleRequired'), trigger: 'change' }],
  status: [{ required: true, message: t('system.user.statusRequired'), trigger: 'change' }],
}))

const showDialog = async (payload?: { userId: string; roleName?: string }) => {
  open.value = true
  await getRoleList()

  if (!payload?.userId) return
  submitForm.value.userId = payload.userId
  await getUserDetail(payload.userId, payload.roleName)
}

defineExpose({
  showDialog,
})
</script>

<style></style>
