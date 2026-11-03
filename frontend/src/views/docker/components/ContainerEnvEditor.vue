<!-- 环境变量编辑器（结构化键值行）：KEY = VALUE | 删除；底部虚线「添加」。v-model = 键值行数组。 -->
<template>
  <div class="spec-ed" :class="{ 'is-bare': bare }">
    <div v-if="!bare" class="spec-ed__title">{{ t('docker.container.env') }}</div>
    <div v-if="model.length" class="spec-ed__labels spec-ed__grid">
      <span>{{ t('docker.common.key') }}</span>
      <span />
      <span>{{ t('docker.common.value') }}</span>
      <span />
    </div>
    <div v-for="(row, i) in model" :key="i" class="spec-ed__row spec-ed__grid">
      <input v-model="row.key" class="app-input m-mono" placeholder="KEY" :aria-label="t('docker.common.key')" />
      <span class="spec-ed__arrow m-mono" aria-hidden="true">=</span>
      <input v-model="row.value" class="app-input m-mono" placeholder="value" :aria-label="t('docker.common.value')" />
      <AppIconButton icon="HOutline:TrashIcon" :label="t('docker.common.delete')" :size="17" @click="remove(i)" />
    </div>
    <div v-if="!model.length && emptyText" class="spec-ed__empty">{{ emptyText }}</div>
    <button type="button" class="spec-ed__add" @click="add">
      <component :is="menuStore.iconComponents['HOutline:PlusIcon']" />{{ addText || t('docker.common.add') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { EnvRow } from './specTypes'

withDefaults(
  defineProps<{
    bare?: boolean
    emptyText?: string
    addText?: string
  }>(),
  { bare: false, emptyText: '', addText: '' },
)
const model = defineModel<EnvRow[]>({ default: () => [] })
const { t } = useI18n()
const menuStore = useMenuStore()

const add = () => { model.value = [...model.value, { key: '', value: '' }] }
const remove = (i: number) => { model.value = model.value.filter((_, idx) => idx !== i) }
</script>

<style scoped lang="scss">
@use './spec-editor.scss';

.spec-ed__grid {
  grid-template-columns: minmax(0, 1fr) 14px minmax(0, 1.5fr) 40px;
}
@media (width < 640px) {
  .spec-ed__grid {
    grid-template-columns: minmax(0, 0.9fr) 10px minmax(0, 1.1fr) 40px;
    gap: 6px;
  }
}
</style>
