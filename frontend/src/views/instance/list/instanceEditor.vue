<template>
  <BaseDialog
    :model-value="visible"
    :title="mode === 'create' ? '新建实例' : '实例配置'"
    width="820"
    @update:model-value="handleDialogVisibleChange"
    @close="handleDialogClose"
  >
    <div class="instance-editor-body" v-loading="loading">
      <el-form
        ref="formRef"
        :model="localForm"
        :rules="formRules"
        label-width="98px"
        class="instance-editor-form"
      >
        <el-row :gutter="12">
          <el-col :xs="24" :md="12">
            <el-form-item label="实例名称" prop="name">
              <el-input v-model="localForm.name" maxlength="100" clearable />
            </el-form-item>
          </el-col>

          <el-col :xs="24" :md="12">
            <el-form-item label="实例类型" prop="type">
              <el-select v-model="localForm.type" clearable filterable placeholder="请选择实例类型">
                <el-option
                  v-for="typeOption in typeOptions"
                  :key="typeOption.value"
                  :label="typeOption.label"
                  :value="typeOption.value"
                />
              </el-select>
            </el-form-item>
          </el-col>

          <el-col :xs="24" :md="24">
            <el-form-item label="实例路径" prop="instancePath">
              <el-input v-model="localForm.instancePath" clearable />
            </el-form-item>
          </el-col>

          <el-col :xs="24" :md="12">
            <el-form-item label="启动命令" prop="startCommand">
              <el-input v-model="localForm.startCommand" clearable />
            </el-form-item>
          </el-col>

          <el-col :xs="24" :md="12">
            <el-form-item label="停止命令" prop="stopCommand">
              <el-input v-model="localForm.stopCommand" clearable />
            </el-form-item>
          </el-col>

          <el-col :xs="24" :md="12">
            <el-form-item label="标签" prop="tags">
              <el-input v-model="localForm.tags" clearable placeholder="多个标签用逗号分隔" />
            </el-form-item>
          </el-col>

          <el-col :xs="24" :md="12">
            <el-form-item label="自动启动" prop="autoStart">
              <el-switch v-model="localForm.autoStart" />
            </el-form-item>
          </el-col>

          <el-col :xs="24" :md="24">
            <el-form-item label="备注" prop="remark">
              <el-input v-model="localForm.remark" type="textarea" :rows="2" maxlength="500" show-word-limit />
            </el-form-item>
          </el-col>

          <el-col :xs="24" :md="24">
            <el-form-item label="环境变量" prop="envText">
              <el-input
                v-model="localForm.envText"
                type="textarea"
                :rows="4"
                placeholder="一行一个变量，例如: KEY=value"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </div>

    <template #footer>
      <el-button @click="handleDialogClose">取消</el-button>
      <el-button
        type="primary"
        :loading="submitting"
        :disabled="loading"
        @click="handleSubmit"
      >
        保存
      </el-button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import { type FormInstance, type FormRules } from 'element-plus'
import {
  type InstanceEditorFormValue,
  type InstanceEditorMode,
  type InstanceTypeOption,
} from '@/stores/instance/types'

defineOptions({ name: 'InstanceEditor' })

const props = defineProps<{
  visible: boolean
  mode: InstanceEditorMode
  loading: boolean
  submitting: boolean
  form: InstanceEditorFormValue
  typeOptions: InstanceTypeOption[]
}>()

const emit = defineEmits<{
  close: []
  submit: [value: InstanceEditorFormValue]
}>()

const formRef = useTemplateRef<FormInstance>('formRef')
const localForm = reactive<InstanceEditorFormValue>({
  id: '',
  name: '',
  remark: '',
  tags: '',
  type: '',
  startCommand: '',
  stopCommand: '',
  instancePath: '',
  autoStart: false,
  envText: '',
})

const requiredTrimmedValidator = (message: string) => {
  return (_rule: unknown, value: unknown, callback: (error?: Error) => void) => {
    if (typeof value !== 'string' || !value.trim()) {
      callback(new Error(message))
      return
    }

    callback()
  }
}

const formRules: FormRules<InstanceEditorFormValue> = {
  name: [{ validator: requiredTrimmedValidator('请输入实例名称'), trigger: 'blur' }],
  type: [{ validator: requiredTrimmedValidator('请选择实例类型'), trigger: 'change' }],
  instancePath: [{ validator: requiredTrimmedValidator('请输入实例路径'), trigger: 'blur' }],
  startCommand: [{ validator: requiredTrimmedValidator('请输入启动命令'), trigger: 'blur' }],
}

const clearValidateState = () => {
  formRef.value?.clearValidate()
}

const handleDialogClose = () => {
  clearValidateState()
  emit('close')
}

const handleDialogVisibleChange = (visible: boolean) => {
  if (!visible) {
    handleDialogClose()
  }
}

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  emit('submit', { ...localForm })
}

watch(
  () => props.visible,
  (visible) => {
    if (!visible) {
      clearValidateState()
    }
  },
)

watch(
  () => props.form,
  (formValue) => {
    Object.assign(localForm, formValue)
  },
  { immediate: true, deep: true },
)
</script>

<style scoped lang="scss">
.instance-editor-body {
  min-height: 180px;
}

.instance-editor-form :deep(.el-form-item) {
  margin-bottom: 0.9rem;
}
</style>
