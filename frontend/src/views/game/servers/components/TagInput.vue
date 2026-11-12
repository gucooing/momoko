<!-- 自由输入的标签：回车或逗号添加，退格删除最后一个。 -->
<template>
  <div class="tag-input" @click="inputRef?.focus()">
    <span v-for="(tag, i) in modelValue" :key="tag" class="tag-input__tag">
      {{ tag }}
      <button type="button" class="tag-input__x" :aria-label="`${t('gameServers.common.close')} ${tag}`" @click.stop="remove(i)">
        <UIcon name="i-lucide-x" />
      </button>
    </span>
    <input
      ref="inputRef"
      v-model="draft"
      class="tag-input__input"
      :placeholder="modelValue.length ? '' : placeholder"
      :aria-label="label"
      @keydown.enter.prevent="add"
      @keydown="onKey"
      @blur="add"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ modelValue: string[]; placeholder?: string; label?: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string[]] }>()
const { t } = useI18n()
const draft = ref('')
const inputRef = ref<HTMLInputElement>()

const add = () => {
  const parts = draft.value
    .split(/[,，]/)
    .map((s) => s.trim())
    .filter(Boolean)
  draft.value = ''
  if (!parts.length) return
  const next = [...props.modelValue]
  for (const p of parts) if (!next.includes(p) && p.length <= 32 && next.length < 20) next.push(p)
  emit('update:modelValue', next)
}
const remove = (i: number) => emit('update:modelValue', props.modelValue.filter((_, idx) => idx !== i))
const onKey = (e: KeyboardEvent) => {
  if (e.key === ',' || e.key === '，') {
    e.preventDefault()
    add()
  } else if (e.key === 'Backspace' && !draft.value && props.modelValue.length) {
    remove(props.modelValue.length - 1)
  }
}
</script>

<style scoped lang="scss">
.tag-input {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 5px 10px;
  border-radius: var(--app-radius-sm);
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  cursor: text;
}
.tag-input:focus-within {
  border-color: var(--m-p);
}
.tag-input__tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 26px;
  padding: 0 4px 0 10px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  font-size: 0.8125rem;
  font-weight: 700;
}
.tag-input__x {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
}
.tag-input__x:hover {
  background: var(--el-fill-color);
}
.tag-input__input {
  flex: 1;
  min-width: 100px;
  border: none;
  outline: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.875rem;
}
</style>
