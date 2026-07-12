<!-- 环境变量编辑器（结构化键值行，替代 “KEY=VALUE” 文本域）：键 | 值 | 删除。v-model = 键值行数组。 -->
<template>
  <div class="spec-ed">
    <div class="spec-ed__head">
      <span>{{ t('docker.container.env') }}</span>
      <button type="button" class="spec-ed__add" @click="add">
        <component :is="menuStore.iconComponents['HOutline:PlusIcon']" />{{ t('docker.common.add') }}
      </button>
    </div>
    <div v-if="model.length" class="spec-ed__labels">
      <span>{{ t('docker.common.key') }}</span>
      <span>{{ t('docker.common.value') }}</span>
      <span />
    </div>
    <div v-for="(row, i) in model" :key="i" class="spec-ed__row">
      <input v-model="row.key" class="app-input" placeholder="KEY" />
      <input v-model="row.value" class="app-input" placeholder="VALUE" />
      <AppIconButton icon="HOutline:TrashIcon" :label="t('docker.common.delete')" :box="30" @click="remove(i)" />
    </div>
    <div v-if="!model.length" class="spec-ed__empty">{{ t('docker.container.emptyEnv') }}</div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { EnvRow } from './specTypes'

const model = defineModel<EnvRow[]>({ default: () => [] })
const { t } = useI18n()
const menuStore = useMenuStore()

const add = () => { model.value = [...model.value, { key: '', value: '' }] }
const remove = (i: number) => { model.value = model.value.filter((_, idx) => idx !== i) }
</script>

<style scoped lang="scss">
.spec-ed { border: 1px solid var(--el-border-color-lighter); border-radius: var(--app-radius); padding: 10px; }
.spec-ed__head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; font-size: 0.8125rem; font-weight: 600; color: var(--el-text-color-primary); }
.spec-ed__add { display: inline-flex; align-items: center; gap: 3px; border: none; background: transparent; color: var(--el-color-primary); font-size: 0.8125rem; cursor: pointer; }
.spec-ed__add :deep(svg) { width: 14px; height: 14px; }
.spec-ed__labels, .spec-ed__row { display: grid; grid-template-columns: 1fr 1.4fr 30px; gap: 6px; align-items: center; }
.spec-ed__labels { margin-bottom: 4px; }
.spec-ed__labels span { font-size: 0.7rem; color: var(--el-text-color-placeholder); }
.spec-ed__row { margin-bottom: 6px; }
.spec-ed__empty { padding: 8px; text-align: center; color: var(--el-text-color-placeholder); font-size: 0.78rem; }
</style>
