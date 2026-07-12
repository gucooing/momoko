<!-- 实例类型 新建/编辑弹窗（重写 · P1）：FormDialog 外壳 + 令牌字段 + AppSwitch + 内联校验。
     保留 ref 契约 showDialog(row?) + @refresh；直连 create/updateInstanceType。 -->
<template>
  <FormDialog
    v-model="open"
    :title="submitForm.id ? t('instance.editInstanceType') : t('instance.addInstanceType')"
    :width="460"
    :loading="submitLoading"
    @close="close"
    @confirm="confirm"
  >
    <div class="type-form">
      <div class="app-field">
        <label class="app-label app-label--required">{{ t('instance.typeName') }}</label>
        <input
          v-model="submitForm.name"
          class="app-input"
          :class="{ 'is-error': errors.name }"
          :placeholder="t('instance.typeNamePlaceholder')"
          maxlength="50"
          @keyup.enter="confirm"
        />
        <span v-if="errors.name" class="app-field__error">{{ errors.name }}</span>
      </div>

      <div class="app-field type-form__switch">
        <label class="app-label">{{ t('instance.isEnabled') }}</label>
        <AppSwitch v-model="submitForm.isEnable" />
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { createInstanceType, updateInstanceType } from '@/api/instance'
import type { InstanceTypeInfo } from '@/types/v1/instance'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'InstanceTypeCreate' })

const emits = defineEmits<{ refresh: [] }>()
const { t } = useI18n()

const open = ref(false)
const submitLoading = ref(false)
const errors = ref<Record<string, string>>({})

const getDefaultForm = () => ({
  id: undefined as string | undefined,
  name: '',
  isEnable: true,
})
const submitForm = ref(getDefaultForm())

const close = () => {
  open.value = false
  submitLoading.value = false
  errors.value = {}
  submitForm.value = getDefaultForm()
}

const validate = () => {
  const e: Record<string, string> = {}
  if (!submitForm.value.name.trim()) e.name = t('instance.instanceTypeNameRequired')
  errors.value = e
  return Object.keys(e).length === 0
}

const confirm = async () => {
  if (!validate()) return

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
  errors.value = {}
  if (instanceType) {
    submitForm.value = {
      id: instanceType.id,
      name: instanceType.name,
      isEnable: instanceType.isEnable,
    }
  } else {
    submitForm.value = getDefaultForm()
  }
  open.value = true
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.type-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.type-form__switch {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
</style>
