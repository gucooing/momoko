<!-- 智商检测 · 提示词 创建/编辑弹窗：FormDialog + 标题/内容/启用/排序 + 内联校验。保存后 emit refresh。 -->
<template>
  <FormDialog
    v-model="open"
    :title="form.id ? t('iqcheck.prompt.editTitle') : t('iqcheck.prompt.createTitle')"
    :width="620"
    :loading="saving"
    @confirm="confirm"
    @close="close"
  >
    <div class="iq-prompt-form">
      <div class="app-field">
        <label class="app-label app-label--required">{{ t('iqcheck.prompt.colTitle') }}</label>
        <input
          v-model="form.title"
          class="app-input"
          :class="{ 'is-error': errors.title }"
          :placeholder="t('iqcheck.prompt.titlePlaceholder')"
        />
        <span v-if="errors.title" class="app-field__error">{{ errors.title }}</span>
      </div>

      <div class="app-field">
        <label class="app-label app-label--required">{{ t('iqcheck.prompt.colContent') }}</label>
        <textarea
          v-model="form.content"
          class="app-textarea"
          :class="{ 'is-error': errors.content }"
          rows="8"
          :placeholder="t('iqcheck.prompt.contentPlaceholder')"
        />
        <span v-if="errors.content" class="app-field__error">{{ errors.content }}</span>
      </div>

      <div class="iq-prompt-form__row">
        <div class="app-field">
          <label class="app-label">{{ t('iqcheck.prompt.colOrder') }}</label>
          <input v-model.number="form.order" type="number" class="app-input" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('iqcheck.prompt.colEnabled') }}</label>
          <div class="iq-prompt-form__switch">
            <AppSwitch v-model="form.enabled" />
            <span>{{ form.enabled ? t('system.common.enabled') : t('system.common.inactive') }}</span>
          </div>
        </div>
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { createIQCheckPrompt, updateIQCheckPrompt } from '@/api/iqcheck'
import type { IQCheckPromptInfo } from '@/types/v1/iqcheck'

defineOptions({ name: 'IQCheckPromptForm' })

const emit = defineEmits<{ refresh: [] }>()
const { t } = useI18n()

const open = ref(false)
const saving = ref(false)
const errors = ref<Record<string, string>>({})

const emptyForm = () => ({ id: '', title: '', content: '', enabled: true, order: 0 })
const form = ref(emptyForm())

const close = () => {
  open.value = false
  saving.value = false
  errors.value = {}
  form.value = emptyForm()
}

const validate = (): boolean => {
  const e: Record<string, string> = {}
  if (!form.value.title.trim()) e.title = t('iqcheck.prompt.titleRequired')
  if (!form.value.content.trim()) e.content = t('iqcheck.prompt.contentRequired')
  errors.value = e
  return Object.keys(e).length === 0
}

const confirm = async () => {
  if (!validate()) return
  saving.value = true
  try {
    const payload = {
      title: form.value.title.trim(),
      content: form.value.content.trim(),
      enabled: form.value.enabled,
      order: Number(form.value.order) || 0,
    }
    if (form.value.id) await updateIQCheckPrompt({ id: form.value.id, ...payload })
    else await createIQCheckPrompt(payload)
    feedback.success(form.value.id ? t('iqcheck.prompt.editSuccess') : t('iqcheck.prompt.createSuccess'))
    emit('refresh')
    close()
  } finally {
    saving.value = false
  }
}

const showDialog = (row?: IQCheckPromptInfo) => {
  errors.value = {}
  form.value = {
    id: row?.id || '',
    title: row?.title || '',
    content: row?.content || '',
    enabled: row?.enabled ?? true,
    order: row?.order ?? 0,
  }
  open.value = true
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.iq-prompt-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.iq-prompt-form__row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}
.iq-prompt-form__switch {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 36px;
  font-size: 0.8125rem;
  color: var(--el-text-color-regular);
}
</style>
