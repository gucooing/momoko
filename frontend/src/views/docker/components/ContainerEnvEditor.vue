<!-- 环境变量编辑器：变量名 = 值 | 删除。PC 一行一条；手机在一张卡片里每条上下两格（变量名粗体）。 -->
<template>
  <div class="spec-ed">
    <template v-if="!menuStore.isMobile">
      <div v-if="model.length" class="spec-grid spec-head ee-cols">
        <span>{{ t('docker.wizard.envKey') }}</span><span /><span>{{ t('docker.wizard.envValue') }}</span><span />
      </div>
      <div v-for="(row, i) in model" :key="i" class="spec-grid ee-cols">
        <span class="spec-fld"><input v-model.trim="row.key" placeholder="KEY" :aria-label="t('docker.wizard.envKey')" /></span>
        <span class="spec-eq">=</span>
        <span class="spec-fld"><input v-model="row.value" placeholder="value" :aria-label="t('docker.wizard.envValueOf', { key: row.key || 'KEY' })" /></span>
        <AppIconButton class="spec-del" icon="HOutline:TrashIcon" :label="t('docker.wizard.removeEnv', { key: row.key })" :box="36" :size="17" @click="remove(i)" />
      </div>
      <div v-if="!model.length && emptyText" class="spec-empty">{{ emptyText }}</div>
    </template>

    <div v-else class="ee-mcard">
      <div v-for="(row, i) in model" :key="i" class="ee-mrow">
        <div class="ee-mfields">
          <span class="spec-fld"><input v-model.trim="row.key" class="ee-key" placeholder="KEY" :aria-label="t('docker.wizard.envKey')" /></span>
          <span class="spec-fld"><span class="spec-eq">=</span><input v-model="row.value" placeholder="value" :aria-label="t('docker.wizard.envValueOf', { key: row.key || 'KEY' })" /></span>
        </div>
        <AppIconButton class="spec-del" icon="HOutline:TrashIcon" :label="t('docker.wizard.removeEnv', { key: row.key })" :box="44" @click="remove(i)" />
      </div>
      <div class="ee-macts">
        <button type="button" class="ee-mbtn is-add" @click="add"><UIcon name="i-lucide-plus" />{{ t('docker.wizard.addEnv') }}</button>
        <button type="button" class="ee-mbtn" @click="emit('paste')"><UIcon name="i-lucide-clipboard-paste" />{{ t('docker.wizard.pasteEnv') }}</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { EnvRow } from './specTypes'

defineProps<{ emptyText?: string }>()
const emit = defineEmits<{ paste: [] }>()
const model = defineModel<EnvRow[]>({ required: true })
const { t } = useI18n()
const menuStore = useMenuStore()

const add = () => {
  model.value = [...model.value, { key: '', value: '' }]
}
const remove = (i: number) => {
  model.value = model.value.filter((_, idx) => idx !== i)
}
defineExpose({ add })
</script>

<style scoped lang="scss">
@use './spec-editor.scss';

.ee-cols {
  grid-template-columns: minmax(0, 0.85fr) 14px minmax(0, 1.15fr) 36px;
}
.ee-mcard {
  padding: 4px 4px 4px 14px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.ee-mrow {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 10px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.ee-mfields {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.ee-key {
  font-weight: 800 !important;
}
.ee-macts {
  display: flex;
  gap: 4px;
  padding: 6px 10px 6px 0;
}
.ee-mbtn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 44px;
  padding: 0 8px;
  border: none;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
  font-weight: 700;
  cursor: pointer;
}
.ee-mbtn.is-add {
  flex: 1;
  color: var(--m-pInk);
}
.ee-mbtn :deep(svg),
.ee-mbtn :deep(.iconify) {
  width: 18px;
  height: 18px;
}
</style>
