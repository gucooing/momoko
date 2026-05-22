<template>
  <BaseDialog
    v-model="open"
    :title="editingId ? '编辑连接' : '新增连接'"
    width="640"
    @close="close"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="formRules"
      label-width="100px"
      label-position="right"
    >
      <el-form-item label="名称" prop="name">
        <el-input v-model="form.name" placeholder="请输入连接名称" />
      </el-form-item>

      <el-row :gutter="10">
        <el-col :span="16">
          <el-form-item label="主机地址" prop="host" label-width="100px">
            <el-input v-model="form.host" placeholder="请输入主机地址" />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="端口" prop="port" label-width="60px">
            <el-input-number v-model="form.port" :min="1" :max="65535" style="width: 100%" />
          </el-form-item>
        </el-col>
      </el-row>

      <el-form-item label="用户名" prop="username">
        <el-input v-model="form.username" placeholder="请输入用户名" />
      </el-form-item>

      <el-form-item label="认证方式" prop="authType">
        <el-radio-group v-model="form.authType">
          <el-radio value="SSH_AUTH_TYPE_PASSWORD">密码登录</el-radio>
          <el-radio value="SSH_AUTH_TYPE_KEY">密钥登录</el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item v-if="form.authType === 'SSH_AUTH_TYPE_PASSWORD'" label="密码" prop="password">
        <el-input
          v-model="form.password"
          type="password"
          placeholder="请输入密码"
          show-password
        />
      </el-form-item>

      <template v-else>
        <el-form-item label="私钥" prop="privateKey">
          <el-input
            v-model="form.privateKey"
            type="textarea"
            :rows="4"
            placeholder="请输入 SSH 私钥内容"
          />
        </el-form-item>
        <el-form-item label="私钥口令" prop="passphrase">
          <el-input
            v-model="form.passphrase"
            type="password"
            placeholder="私钥口令（如有）"
            show-password
          />
        </el-form-item>
      </template>

      <el-form-item label="主机指纹" prop="fingerprint">
        <el-input v-model="form.fingerprint" placeholder="主机公钥指纹（可选，用于安全校验）" />
      </el-form-item>

      <el-form-item label="标签" prop="tags">
        <el-input v-model="form.tags" placeholder="多个标签用逗号分隔" />
      </el-form-item>

      <el-form-item label="分享用户" prop="sharedUserIds">
        <el-select
          v-model="form.sharedUserIds"
          multiple
          filterable
          remote
          reserve-keyword
          placeholder="搜索并选择要分享的用户"
          :remote-method="searchUsers"
          :loading="userSearchLoading"
          style="width: 100%"
        >
          <el-option
            v-for="user in userOptions"
            :key="user.userId"
            :label="user.name || user.userId"
            :value="user.userId"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="备注" prop="remark">
        <el-input
          v-model="form.remark"
          type="textarea"
          :rows="2"
          placeholder="请输入备注信息"
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button type="primary" :loading="loading" @click="confirm">确定</el-button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import { createSshHost, updateSshHost } from '@/api/openssh'
import { userPage } from '@/api/user'
import { SSHAuthType, type SSHHostInfo, type SSHHostSharedUser, type UpdateSSHHostRequest } from '@/types/v1/openssh'
import type { FormInstance, FormRules } from 'element-plus'

defineOptions({ name: 'SshConnectionCreate' })

const emits = defineEmits(['refresh'])
const formRef = useTemplateRef<FormInstance>('formRef')

const open = ref(false)
const loading = ref(false)
const editingId = ref('')
const userSearchLoading = ref(false)
const userOptions = ref<SSHHostSharedUser[]>([])

const defaultForm = () => ({
  name: '',
  host: '',
  port: 22,
  username: '',
  authType: 'SSH_AUTH_TYPE_PASSWORD' as string,
  password: '',
  privateKey: '',
  passphrase: '',
  fingerprint: '',
  tags: '',
  sharedUserIds: [] as string[],
  remark: '',
})

const form = ref(defaultForm())

const close = () => {
  open.value = false
  formRef.value?.resetFields()
  editingId.value = ''
  form.value = defaultForm()
  userOptions.value = []
}

const originalForm = ref(defaultForm())

const confirm = async () => {
  await formRef.value?.validate()
  loading.value = true
  try {
    if (editingId.value) {
      // Build patch payload: only changed fields
      const patch: Record<string, unknown> = { id: editingId.value }
      const tracked: (keyof typeof form.value)[] = [
        'name', 'host', 'port', 'username', 'authType',
        'fingerprint', 'tags', 'remark',
      ]
      for (const key of tracked) {
        if (form.value[key] !== originalForm.value[key]) {
          patch[key] = form.value[key]
        }
      }
      // credentials: only include if non-empty (user entered new ones)
      if (form.value.password) {
        patch.password = form.value.password
        // if switching to password auth or changing password, include authType
        if (!patch.authType) patch.authType = form.value.authType
      }
      if (form.value.privateKey) {
        patch.privateKey = form.value.privateKey
        if (form.value.passphrase) patch.passphrase = form.value.passphrase
        if (!patch.authType) patch.authType = form.value.authType
      }

      await updateSshHost(patch as unknown as UpdateSSHHostRequest)
    } else {
      await createSshHost({
        name: form.value.name,
        host: form.value.host,
        port: form.value.port,
        username: form.value.username,
        authType: form.value.authType as SSHAuthType,
        password: form.value.password,
        privateKey: form.value.privateKey,
        passphrase: form.value.passphrase,
        fingerprint: form.value.fingerprint,
        tags: form.value.tags,
        remark: form.value.remark,
        sharedUserIds: form.value.sharedUserIds,
      })
    }

    ElMessage.success(editingId.value ? '编辑成功' : '新增成功')
    emits('refresh', editingId.value ? 'update' : 'create')
    close()
  } finally {
    loading.value = false
  }
}

const formRules: FormRules = {
  name: [{ required: true, message: '请输入连接名称', trigger: 'blur' }],
  host: [{ required: true, message: '请输入主机地址', trigger: 'blur' }],
  port: [{ required: true, message: '请设置端口', trigger: 'blur' }],
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [
    {
      validator: (_, value: string, callback) => {
        if (form.value.authType === 'SSH_AUTH_TYPE_PASSWORD' && !editingId.value && !value?.trim()) {
          callback(new Error('请输入密码'))
          return
        }
        callback()
      },
      trigger: 'blur',
    },
  ],
  privateKey: [
    {
      validator: (_, value: string, callback) => {
        if (form.value.authType === 'SSH_AUTH_TYPE_KEY' && !editingId.value && !value?.trim()) {
          callback(new Error('请输入私钥内容'))
          return
        }
        callback()
      },
      trigger: 'blur',
    },
  ],
}

const searchUsers = async (query: string) => {
  if (!query) {
    userOptions.value = []
    return
  }
  userSearchLoading.value = true
  try {
    const { data } = await userPage({ page: 1, pageSize: 20, username: query })
    userOptions.value = (data?.users || []).map((u) => ({
      userId: u.userId,
      name: u.name || u.username,
    }))
  } finally {
    userSearchLoading.value = false
  }
}

const showDialog = (payload?: SSHHostInfo) => {
  open.value = true
  if (!payload?.id) {
    originalForm.value = defaultForm()
    return
  }

  editingId.value = payload.id
  form.value = {
    name: payload.name || '',
    host: payload.host || '',
    port: payload.port || 22,
    username: payload.username || '',
    authType: payload.authType || 'SSH_AUTH_TYPE_PASSWORD',
    password: '',
    privateKey: '',
    passphrase: '',
    fingerprint: payload.fingerprint || '',
    tags: payload.tags || '',
    sharedUserIds: payload.sharedUsers?.map((user) => user.userId) || [],
    remark: payload.remark || '',
  }
  originalForm.value = { ...form.value }

  // pre-load shared users into options so the select shows names, not IDs
  if (payload.sharedUsers?.length) {
    userOptions.value = [...payload.sharedUsers]
  }
}

defineExpose({ showDialog })
</script>
