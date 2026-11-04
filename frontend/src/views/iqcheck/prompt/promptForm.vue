<!-- 智商检测 · 提示词 创建/编辑弹窗：FormDialog + 标题/内容/分类/评分方式/排序/启用 + 内联校验。保存后 emit refresh。 -->
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
          <label class="app-label">{{ t('iqcheck.ui.prompt.form.category') }}</label>
          <AppSelect v-model="form.category" :options="categoryOptions" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('iqcheck.ui.prompt.form.grading') }}</label>
          <input
            v-model="form.grading"
            class="app-input"
            :class="{ 'is-error': errors.grading }"
            maxlength="64"
            :placeholder="t('iqcheck.ui.prompt.form.gradingPlaceholder')"
          />
          <span v-if="errors.grading" class="app-field__error">{{ errors.grading }}</span>
          <span v-else class="iq-prompt-form__hint">{{ t('iqcheck.ui.prompt.form.gradingHint') }}</span>
        </div>
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
import { IQCheckPromptCategory, type IQCheckPromptInfo } from '@/types/v1/iqcheck'
import { PROMPT_CATEGORIES } from '@/views/iqcheck/prompt/category'

defineOptions({ name: 'IQCheckPromptForm' })

const emit = defineEmits<{ refresh: [] }>()
const { t } = useI18n()

const open = ref(false)
const saving = ref(false)
const errors = ref<Record<string, string>>({})

const emptyForm = () => ({
  id: '',
  title: '',
  content: '',
  enabled: true,
  order: 0,
  category: IQCheckPromptCategory.IQ_CHECK_PROMPT_CATEGORY_OTHER as IQCheckPromptCategory,
  grading: '',
})
const form = ref(emptyForm())

const categoryOptions = computed(() =>
  PROMPT_CATEGORIES.map((c) => ({ label: t(`iqcheck.ui.category.${c.key}`), value: c.value })),
)

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
  if ([...form.value.grading.trim()].length > 64) e.grading = t('iqcheck.ui.prompt.form.gradingTooLong')
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
      category: form.value.category,
      grading: form.value.grading.trim(),
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
    category: row?.category || IQCheckPromptCategory.IQ_CHECK_PROMPT_CATEGORY_OTHER,
    grading: row?.grading || '',
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
@media (width <= 768px) {
  .iq-prompt-form__row {
    grid-template-columns: 1fr;
  }
}
.iq-prompt-form__hint {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
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
