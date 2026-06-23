<template>
  <BaseDialog
    :model-value="visible"
    :title="mode === 'create' ? t('instance.editorTitleCreate') : t('instance.editorTitleEdit')"
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
            <el-form-item :label="t('instance.instanceName')" prop="name">
              <el-input v-model="localForm.name" maxlength="100" clearable />
            </el-form-item>
          </el-col>

          <el-col :xs="24" :md="12">
            <el-form-item :label="t('instance.instanceType')" prop="type">
              <el-select v-model="localForm.type" clearable filterable :placeholder="t('instance.selectInstanceType')">
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
            <el-form-item :label="t('instance.instancePath')" prop="instancePath">
              <el-input v-model="localForm.instancePath" clearable />
            </el-form-item>
          </el-col>

          <el-col :xs="24" :md="12">
            <el-form-item :label="t('instance.startCommand')" prop="startCommand">
              <el-input v-model="localForm.startCommand" clearable />
            </el-form-item>
          </el-col>

          <el-col :xs="24" :md="12">
            <el-form-item :label="t('instance.stopCommand')" prop="stopCommand">
              <el-input v-model="localForm.stopCommand" clearable />
            </el-form-item>
          </el-col>

          <el-col :xs="24" :md="12">
            <el-form-item :label="t('instance.tags')" prop="tags">
              <el-input v-model="localForm.tags" clearable :placeholder="t('instance.tagsPlaceholder')" />
            </el-form-item>
          </el-col>

          <el-col :xs="24" :md="12">
            <el-form-item :label="t('instance.autoStart')" prop="autoStart">
              <el-switch v-model="localForm.autoStart" />
            </el-form-item>
          </el-col>

          <el-col :xs="24" :md="24">
            <el-form-item :label="t('common.remark')" prop="remark">
              <el-input v-model="localForm.remark" type="textarea" :rows="2" maxlength="500" show-word-limit />
            </el-form-item>
          </el-col>

          <el-col :xs="24" :md="24">
            <el-form-item :label="t('instance.environmentVariables')" prop="envText">
              <el-input
                v-model="localForm.envText"
                type="textarea"
                :rows="4"
                :placeholder="t('instance.envPlaceholder')"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </div>

    <template #footer>
      <el-button @click="handleDialogClose">{{ t('common.cancel') }}</el-button>
      <el-button
        type="primary"
        :loading="submitting"
        :disabled="loading"
        @click="handleSubmit"
      >
        {{ t('common.save') }}
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
import { useI18n } from 'vue-i18n'

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
const { t } = useI18n()
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

const formRules = computed<FormRules<InstanceEditorFormValue>>(() => ({
  name: [{ validator: requiredTrimmedValidator(t('instance.instanceNameRequired')), trigger: 'blur' }],
  type: [{ validator: requiredTrimmedValidator(t('instance.instanceTypeRequired')), trigger: 'change' }],
  instancePath: [{ validator: requiredTrimmedValidator(t('instance.instancePathRequired')), trigger: 'blur' }],
  startCommand: [{ validator: requiredTrimmedValidator(t('instance.startCommandRequired')), trigger: 'blur' }],
}))

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
