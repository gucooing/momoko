<!-- 实例新建/配置弹窗（重写）：FormDialog 外壳 + 令牌化字段 + AppSelect/AppSwitch + 内联校验。
     保留受控契约：props(visible/mode/loading/submitting/form/typeOptions) + emits(close/submit)。 -->
<template>
  <FormDialog
    :model-value="visible"
    :title="mode === 'create' ? t('instance.editorTitleCreate') : t('instance.editorTitleEdit')"
    :width="640"
    :loading="submitting"
    @close="handleClose"
    @confirm="handleSubmit"
  >
    <div v-if="loading" class="inst-form__loading">{{ t('common.loading') }}…</div>

    <div v-else class="inst-form">
      <div class="app-field">
        <label class="app-label app-label--required">{{ t('instance.instanceName') }}</label>
        <input
          v-model="localForm.name"
          class="app-input"
          :class="{ 'is-error': errors.name }"
          maxlength="100"
        />
        <span v-if="errors.name" class="app-field__error">{{ errors.name }}</span>
      </div>

      <div class="app-field">
        <label class="app-label app-label--required">{{ t('instance.instanceType') }}</label>
        <AppSelect
          v-model="localForm.type"
          :options="typeOptions"
          :placeholder="t('instance.selectInstanceType')"
          :error="!!errors.type"
        />
        <span v-if="errors.type" class="app-field__error">{{ errors.type }}</span>
      </div>

      <div class="app-field inst-form__full">
        <label class="app-label app-label--required">{{ t('instance.instancePath') }}</label>
        <input
          v-model="localForm.instancePath"
          class="app-input"
          :class="{ 'is-error': errors.instancePath }"
        />
        <span v-if="errors.instancePath" class="app-field__error">{{ errors.instancePath }}</span>
      </div>

      <div class="app-field">
        <label class="app-label app-label--required">{{ t('instance.startCommand') }}</label>
        <input
          v-model="localForm.startCommand"
          class="app-input"
          :class="{ 'is-error': errors.startCommand }"
        />
        <span v-if="errors.startCommand" class="app-field__error">{{ errors.startCommand }}</span>
      </div>

      <div class="app-field">
        <label class="app-label">{{ t('instance.stopCommand') }}</label>
        <input v-model="localForm.stopCommand" class="app-input" />
      </div>

      <div class="app-field">
        <label class="app-label">{{ t('instance.tags') }}</label>
        <input v-model="localForm.tags" class="app-input" :placeholder="t('instance.tagsPlaceholder')" />
      </div>

      <div class="app-field inst-form__switch">
        <label class="app-label">{{ t('instance.autoStart') }}</label>
        <AppSwitch v-model="localForm.autoStart" />
      </div>

      <div class="app-field inst-form__full">
        <label class="app-label">{{ t('common.remark') }}</label>
        <textarea v-model="localForm.remark" class="app-textarea" rows="2" maxlength="500" />
      </div>

      <div class="app-field inst-form__full">
        <label class="app-label">{{ t('instance.environmentVariables') }}</label>
        <textarea
          v-model="localForm.envText"
          class="app-textarea"
          rows="4"
          :placeholder="t('instance.envPlaceholder')"
        />
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
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

const errors = ref<Record<string, string>>({})

const validate = (): boolean => {
  const e: Record<string, string> = {}
  if (!localForm.name.trim()) e.name = t('instance.instanceNameRequired')
  if (!localForm.type.trim()) e.type = t('instance.instanceTypeRequired')
  if (!localForm.instancePath.trim()) e.instancePath = t('instance.instancePathRequired')
  if (!localForm.startCommand.trim()) e.startCommand = t('instance.startCommandRequired')
  errors.value = e
  return Object.keys(e).length === 0
}

const handleClose = () => {
  errors.value = {}
  emit('close')
}

const handleSubmit = () => {
  if (!validate()) return
  emit('submit', { ...localForm })
}

watch(
  () => props.visible,
  (visible) => {
    if (!visible) errors.value = {}
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
.inst-form {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}
.inst-form__full {
  grid-column: 1 / -1;
}
.inst-form__switch {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.inst-form__loading {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 160px;
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
}
@media (width <= 768px) {
  .inst-form {
    grid-template-columns: 1fr;
  }
}
</style>
