<template>
  <BaseDialog
    v-model="open"
    :title="submitForm.id ? '编辑实例类型' : '新增实例类型'"
    width="520"
    @close="close"
  >
    <el-form
      ref="submitFormRef"
      :model="submitForm"
      :rules="formRules"
      label-width="100px"
      label-position="right"
    >
      <el-form-item label="类型名称" prop="name">
        <el-input
          v-model="submitForm.name"
          placeholder="请输入实例类型名称"
          maxlength="50"
          clearable
        />
      </el-form-item>

      <el-form-item label="是否启用" prop="isEnable">
        <el-radio-group v-model="submitForm.isEnable">
          <el-radio :label="true">启用</el-radio>
          <el-radio :label="false">禁用</el-radio>
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
import { createInstanceType, updateInstanceType } from '@/api/instance'
import type { InstanceTypeInfo } from '@/types/v1/instance'
import { type FormInstance, type FormRules } from 'element-plus'

defineOptions({ name: 'InstanceTypeCreate' })

const emits = defineEmits(['refresh'])
const submitFormRef = useTemplateRef<FormInstance>('submitFormRef')

const open = ref(false)
const submitLoading = ref(false)

const getDefaultForm = () => ({
  id: undefined as string | undefined,
  name: '',
  isEnable: true,
})

const submitForm = ref(getDefaultForm())

const close = () => {
  open.value = false
  submitLoading.value = false
  submitFormRef.value?.resetFields()
  submitForm.value = getDefaultForm()
}

const confirm = async () => {
  await submitFormRef.value?.validate()

  const payload = {
    name: submitForm.value.name.trim(),
    isEnable: submitForm.value.isEnable,
  }

  submitLoading.value = true
  try {
    if (submitForm.value.id) {
      await updateInstanceType({ id: submitForm.value.id, ...payload })
    } else {
      await createInstanceType(payload)
    }

    ElMessage.success(submitForm.value.id ? '编辑成功' : '新增成功')
    emits('refresh')
    close()
  } finally {
    submitLoading.value = false
  }
}

const showDialog = (instanceType?: InstanceTypeInfo) => {
  open.value = true

  if (!instanceType) return

  submitForm.value = {
    id: instanceType.id,
    name: instanceType.name,
    isEnable: instanceType.isEnable,
  }
}

const formRules: FormRules = {
  name: [
    {
      validator: (_rule, value: string, callback) => {
        if (!value?.trim()) {
          callback(new Error('请输入实例类型名称'))
          return
        }
        callback()
      },
      trigger: 'blur',
    },
  ],
  isEnable: [{ required: true, message: '请选择是否启用', trigger: 'change' }],
}

defineExpose({
  showDialog,
})
</script>

<style></style>
