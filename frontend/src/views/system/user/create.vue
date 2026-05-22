<template>
  <BaseDialog
    v-model="open"
    :title="submitForm.userId ? '编辑用户' : '新增用户'"
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
      <el-form-item label="用户名" prop="username">
        <el-input
          v-model="submitForm.username"
          placeholder="请输入用户名（不允许中文）"
          :disabled="!!submitForm.userId"
        />
      </el-form-item>

      <el-form-item v-if="!submitForm.userId" label="密码" prop="password" :required="true">
        <el-input
          v-model="submitForm.password"
          type="password"
          placeholder="请输入密码"
          show-password
        />
      </el-form-item>

      <el-form-item label="姓名" prop="name" :required="!submitForm.userId">
        <el-input v-model="submitForm.name" placeholder="请输入姓名" />
      </el-form-item>

      <el-form-item label="邮箱" prop="email" :required="!submitForm.userId">
        <el-input v-model="submitForm.email" placeholder="请输入邮箱" />
      </el-form-item>

      <el-form-item label="用户角色" prop="roleId">
        <el-select v-model="submitForm.roleId" placeholder="请选择用户角色" style="width: 100%">
          <el-option
            v-for="role in roleList"
            :key="role.roleId"
            :label="role.name"
            :value="role.roleId"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="状态" prop="status">
        <el-radio-group v-model="submitForm.status">
          <el-radio :label="UserStatus.Active">启用</el-radio>
          <el-radio :label="UserStatus.InActive">停用</el-radio>
        </el-radio-group>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button type="primary" :loading="submitLoading" @click="confirm">确定</el-button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import { rolePage } from '@/api/role'
import { createUser, updateUser, userInfo } from '@/api/user'
import type { RoleInfo } from '@/types/v1/system'
import { UserStatus, type UserInfo } from '@/types/v1/user'
import { type FormInstance, type FormRules } from 'element-plus'

defineOptions({ name: 'UserCreate' })

const emits = defineEmits(['refresh'])
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

    ElMessage.success(submitForm.value.userId ? '编辑成功' : '新增成功')
    emits('refresh', submitForm.value.userId ? 'update' : 'create')
    close()
  } finally {
    submitLoading.value = false
  }
}

const formRules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    {
      pattern: /^[^\u4e00-\u9fa5]+$/,
      message: '用户名不允许输入中文',
      trigger: 'blur',
    },
  ],
  password: [
    {
      validator: (_, value: string, callback) => {
        if (!submitForm.value.userId && !value?.trim()) {
          callback(new Error('请输入密码'))
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
          callback(new Error('请输入姓名'))
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
          callback(new Error('请输入邮箱'))
          return
        }
        if (value?.trim() && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
          callback(new Error('邮箱格式不正确'))
          return
        }
        callback()
      },
      trigger: 'blur',
    },
  ],
  roleId: [{ required: true, message: '请选择用户角色', trigger: 'change' }],
  status: [{ required: true, message: '请选择状态', trigger: 'change' }],
}

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
