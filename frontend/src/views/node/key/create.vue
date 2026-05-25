<template>
  <BaseDialog
    v-model="open"
    :title="submitForm.id ? '编辑 API Key' : '新增 API Key'"
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
      <el-form-item label="名称" prop="name">
        <el-input v-model="submitForm.name" placeholder="请输入 API Key 名称" />
      </el-form-item>

      <el-form-item label="过期时间" prop="expiresAt">
        <el-date-picker
          v-model="submitForm.expiresAt"
          type="datetime"
          placeholder="请选择过期时间（不选表示永久有效）"
          format="YYYY-MM-DD HH:mm:ss"
          value-format="YYYY-MM-DD HH:mm:ss"
          style="width: 100%"
          :disabled="submitForm.neverExpires"
        />
      </el-form-item>

      <el-form-item v-if="submitForm.id" label="永久有效" prop="neverExpires">
        <el-checkbox v-model="submitForm.neverExpires" />
        <span class="text-muted">勾选后将清除过期时间，API Key 永久有效</span>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button type="primary" :loading="submitLoading" @click="confirm">确定</el-button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { createAPIKey, updateAPIKey } from '@/api/node'
import type { APIKeyInfo } from '@/types/v1/node'
import type { FormInstance, FormRules } from 'element-plus'

defineOptions({ name: 'ApiKeyCreate' })

const emits = defineEmits(['refresh'])
const submitFormRef = useTemplateRef<FormInstance>('submitFormRef')

const open = ref(false)
const submitLoading = ref(false)

const submitForm = ref({
  id: undefined as string | undefined,
  name: '',
  expiresAt: undefined as string | undefined,
  neverExpires: false,
})

const close = () => {
  open.value = false
  submitFormRef.value?.resetFields()
  submitForm.value = {
    id: undefined,
    name: '',
    expiresAt: undefined,
    neverExpires: false,
  }
}

const confirm = async () => {
  await submitFormRef.value?.validate()

  submitLoading.value = true
  try {
    if (submitForm.value.id) {
      await updateAPIKey({
        id: submitForm.value.id,
        name: submitForm.value.name || undefined,
        expiresAt: submitForm.value.neverExpires
          ? undefined
          : submitForm.value.expiresAt
            ? new Date(submitForm.value.expiresAt)
            : undefined,
        neverExpires: submitForm.value.neverExpires || undefined,
      })
    } else {
      await createAPIKey({
        name: submitForm.value.name,
        expiresAt: submitForm.value.expiresAt
          ? new Date(submitForm.value.expiresAt)
          : undefined,
      })
    }

    ElMessage.success(submitForm.value.id ? '编辑成功' : '新增成功')
    emits('refresh', submitForm.value.id ? 'update' : 'create')
    close()
  } finally {
    submitLoading.value = false
  }
}

const formRules: FormRules = {
  name: [{ required: true, message: '请输入 API Key 名称', trigger: 'blur' }],
}

const showDialog = (record?: APIKeyInfo) => {
  open.value = true

  if (record) {
    submitForm.value.id = record.id
    submitForm.value.name = record.name
    if (record.expiresAt) {
      // Convert ISO date to local datetime string
      const d = new Date(record.expiresAt)
      const pad = (n: number) => String(n).padStart(2, '0')
      submitForm.value.expiresAt = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
    } else {
      submitForm.value.expiresAt = undefined
      submitForm.value.neverExpires = true
    }
  }
}

defineExpose({ showDialog })
</script>
