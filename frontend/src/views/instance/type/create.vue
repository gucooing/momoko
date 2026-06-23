<template>
  <BaseDialog
    v-model="open"
    :title="submitForm.id ? t('instance.editInstanceType') : t('instance.addInstanceType')"
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
      <el-form-item :label="t('instance.typeName')" prop="name">
        <el-input
          v-model="submitForm.name"
          :placeholder="t('instance.typeNamePlaceholder')"
          maxlength="50"
          clearable
        />
      </el-form-item>

      <el-form-item :label="t('instance.isEnabled')" prop="isEnable">
        <el-radio-group v-model="submitForm.isEnable">
          <el-radio :label="true">{{ t('common.enabled') }}</el-radio>
          <el-radio :label="false">{{ t('common.disabled') }}</el-radio>
        </el-radio-group>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="close">{{ t('common.cancel') }}</el-button>
      <el-button type="primary" :loading="submitLoading" @click="confirm">{{ t('common.confirm') }}</el-button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import { createInstanceType, updateInstanceType } from '@/api/instance'
import type { InstanceTypeInfo } from '@/types/v1/instance'
import { type FormInstance, type FormRules } from 'element-plus'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'InstanceTypeCreate' })

const emits = defineEmits(['refresh'])
const submitFormRef = useTemplateRef<FormInstance>('submitFormRef')
const { t } = useI18n()

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

    ElMessage.success(submitForm.value.id ? t('instance.editSuccess') : t('instance.addSuccess'))
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

const formRules = computed<FormRules>(() => ({
  name: [
    {
      validator: (_rule, value: string, callback) => {
        if (!value?.trim()) {
          callback(new Error(t('instance.instanceTypeNameRequired')))
          return
        }
        callback()
      },
      trigger: 'blur',
    },
  ],
  isEnable: [{ required: true, message: t('instance.isEnabledRequired'), trigger: 'change' }],
}))

defineExpose({
  showDialog,
})
</script>

<style></style>
