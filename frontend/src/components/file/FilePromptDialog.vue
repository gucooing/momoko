<template>
  <FileDialog v-model="visible" :title="title" :width="440" @close="onClose">
    <div class="fp-form">
      <p v-if="description" class="fp-desc">{{ description }}</p>
      <label v-if="label" class="fp-label">{{ label }}</label>
      <input
        ref="inputRef"
        v-model="text"
        class="fm-input"
        :placeholder="placeholder"
        @keyup.enter="confirm"
      />
    </div>

    <template #footer>
      <button type="button" class="fm-btn" @click="visible = false">
        {{ t('system.common.cancel') }}
      </button>
      <button
        type="button"
        class="fm-btn fm-btn--primary"
        :disabled="!text.trim() || confirming"
        @click="confirm"
      >
        {{ confirmText || t('system.common.confirm') }}
      </button>
    </template>
  </FileDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import FileDialog from './FileDialog.vue'

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
.fp-form {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.fp-desc {
  margin: 0 0 0.25rem;
  font-size: 13px;
  line-height: 1.5;
  color: var(--fm-text-2);
  word-break: break-all;
}
.fp-label {
  font-size: 13px;
  color: var(--fm-text-2);
}
</style>
