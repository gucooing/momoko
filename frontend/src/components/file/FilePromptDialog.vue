<!-- 文件名输入弹窗（新建文件夹/新建文件/重命名/压缩/解压 共用）：令牌驱动 FormDialog + .app-* 表单控件。 -->
<template>
  <FormDialog
    v-model="visible"
    :title="title"
    :width="440"
    :loading="confirming"
    @close="onClose"
  >
    <div class="fpd">
      <p v-if="description" class="fpd__desc">{{ description }}</p>
      <div class="app-field">
        <label v-if="label" class="app-label">{{ label }}</label>
        <input
          ref="inputRef"
          v-model="text"
          class="app-input"
          :placeholder="placeholder"
          @keyup.enter="confirm"
        />
      </div>
    </div>

    <template #footer="{ close }">
      <UButton color="neutral" variant="soft" @click="close">
        {{ t('system.common.cancel') }}
      </UButton>
      <UButton color="primary" :loading="confirming" :disabled="!text.trim()" @click="confirm">
        {{ confirmText || t('system.common.confirm') }}
      </UButton>
    </template>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  modelValue: boolean
  title: string
  label?: string
  description?: string
  placeholder?: string
  initialValue?: string
  confirmText?: string
  confirming?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  confirm: [value: string]
}>()

const { t } = useI18n()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

const text = ref('')
const inputRef = ref<HTMLInputElement | null>(null)

watch(
  () => props.modelValue,
  async (open) => {
    if (!open) return
    text.value = props.initialValue ?? ''
    await nextTick()
    inputRef.value?.focus()
    // 文件名预选中“名称”部分，方便改扩展名外的主名
    inputRef.value?.select()
  },
)

const confirm = () => {
  const value = text.value.trim()
  if (!value || props.confirming) return
  emit('confirm', value)
}

const onClose = () => {
  text.value = ''
}
</script>

<style scoped>
.fpd {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.fpd__desc {
  margin: 0;
  font-size: 0.8125rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}
</style>
