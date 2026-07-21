<!-- API Key 新建/编辑弹窗（重写 · P1）：FormDialog 外壳 + 令牌字段 + 原生 datetime-local + AppSwitch(永久有效)。
     保留 ref 契约 showDialog(row?) + @refresh('create'|'update')；直连 create/updateAPIKey。 -->
<template>
  <FormDialog
    v-model="open"
    :title="submitForm.id ? t('node.key.edit') : t('node.key.add')"
    :width="480"
    :loading="submitLoading"
    @close="close"
    @confirm="confirm"
  >
    <div class="key-form">
      <div class="app-field">
        <label class="app-label app-label--required">{{ t('node.key.name') }}</label>
        <input
          v-model="submitForm.name"
          class="app-input"
          :class="{ 'is-error': errors.name }"
          :placeholder="t('node.key.namePlaceholder')"
          maxlength="50"
          @keyup.enter="confirm"
        />
        <span v-if="errors.name" class="app-field__error">{{ errors.name }}</span>
      </div>

      <div class="app-field">
        <label class="app-label">{{ t('node.key.expiresAt') }}</label>
        <input
          v-model="submitForm.expiresAt"
          type="datetime-local"
          class="app-input"
          :disabled="submitForm.neverExpires"
        />
        <span class="key-form__hint">{{ t('node.key.expiresPlaceholder') }}</span>
      </div>

      <div v-if="submitForm.id" class="app-field key-form__switch">
        <div class="key-form__switch-text">
          <label class="app-label">{{ t('node.key.neverExpires') }}</label>
          <span class="key-form__hint">{{ t('node.key.neverExpiresTip') }}</span>
        </div>
        <AppSwitch v-model="submitForm.neverExpires" />
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { createAPIKey, updateAPIKey } from '@/api/node'
import type { APIKeyInfo } from '@/types/v1/node'

defineOptions({ name: 'ApiKeyCreate' })

const emits = defineEmits<{ refresh: [type: 'create' | 'update'] }>()
const { t } = useI18n()

const open = ref(false)
const submitLoading = ref(false)
const errors = ref<Record<string, string>>({})

const getDefaultForm = () => ({
  id: undefined as string | undefined,
  name: '',
  expiresAt: '' as string,
  neverExpires: false,
})
const submitForm = ref(getDefaultForm())

// Date → datetime-local 字符串（YYYY-MM-DDTHH:mm，本地时区）
const toLocalInput = (value: Date | string): string => {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const close = () => {
  open.value = false
  submitLoading.value = false
  errors.value = {}
  submitForm.value = getDefaultForm()
}

const validate = () => {
  const e: Record<string, string> = {}
  if (!submitForm.value.name.trim()) e.name = t('node.key.namePlaceholder')
  errors.value = e
  return Object.keys(e).length === 0
}

const confirm = async () => {
  if (!validate()) return

  const expiresDate = submitForm.value.expiresAt ? new Date(submitForm.value.expiresAt) : undefined

  submitLoading.value = true
  try {
    if (submitForm.value.id) {
      await updateAPIKey({
        id: submitForm.value.id,
        name: submitForm.value.name || undefined,
        expiresAt: submitForm.value.neverExpires ? undefined : expiresDate,
        neverExpires: submitForm.value.neverExpires || undefined,
      })
    } else {
      await createAPIKey({
        name: submitForm.value.name,
        expiresAt: expiresDate,
      })
    }
    feedback.success(submitForm.value.id ? t('node.key.editSuccess') : t('node.key.addSuccess'))
    emits('refresh', submitForm.value.id ? 'update' : 'create')
    close()
  } finally {
    submitLoading.value = false
  }
}

const showDialog = (record?: APIKeyInfo) => {
  errors.value = {}
  const form = getDefaultForm()
  if (record) {
    form.id = record.id
    form.name = record.name
    if (record.expiresAt) {
      form.expiresAt = toLocalInput(record.expiresAt)
      form.neverExpires = false
    } else {
      form.neverExpires = true
    }
  }
  submitForm.value = form
  open.value = true
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.key-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.key-form__hint {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
.key-form__switch {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.key-form__switch-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
</style>
